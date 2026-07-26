package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
)

var (
	ErrInvalidOrderType    = errors.New("limit orders require a price")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type OrderService struct {
	store  *db.Store
	engine *LimitOrderEngine // cache for open limit orders
}

func (s *OrderService) SetEngine(engine *LimitOrderEngine) {
	s.engine = engine
}

// NewOrderService now expects the engine instance.
func NewOrderService(store *db.Store, engine *LimitOrderEngine) *OrderService {
	return &OrderService{
		store:  store,
		engine: engine,
	}
}

// splitSymbol splits "BTCUSDT" into ("BTC", "USDT").
func splitSymbol(symbol string) (base, quote string, err error) {
	knownQuotes := []string{"USDT", "USDC", "BUSD"}
	for _, q := range knownQuotes {
		if strings.HasSuffix(symbol, q) {
			return strings.TrimSuffix(symbol, q), q, nil
		}
	}
	return "", "", fmt.Errorf("unrecognized quote asset in symbol %s", symbol)
}

func (s *OrderService) PlaceOrder(ctx context.Context, userID uuid.UUID, req models.PlaceOrderRequest) (gen.Order, error) {
	if req.OrderType == "limit" && req.Price == nil {
		return gen.Order{}, ErrInvalidOrderType
	}

	base, quote, err := splitSymbol(req.Symbol)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid symbol: %w", err)
	}

	// 1. Parse Quantity
	qtyDec, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid quantity format: %w", err)
	}
	quantityNumeric, err := StringToNumeric(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid pg quantity: %w", err)
	}

	// 2. Determine Price
	var priceDec decimal.Decimal
	var priceParam pgtype.Numeric

	if req.OrderType == "market" {
		execPriceFloat, err := GetCurrentPrice(req.Symbol)
		if err != nil {
			return gen.Order{}, fmt.Errorf("failed to get current price: %w", err)
		}
		priceDec = decimal.NewFromFloat(execPriceFloat)
		priceParam, err = StringToNumeric(priceDec.String())
		if err != nil {
			return gen.Order{}, fmt.Errorf("invalid price format: %w", err)
		}
	} else {
		priceDec, err = decimal.NewFromString(*req.Price)
		if err != nil {
			return gen.Order{}, fmt.Errorf("invalid price format: %w", err)
		}
		priceParam, err = StringToNumeric(*req.Price)
		if err != nil {
			return gen.Order{}, fmt.Errorf("invalid pg price: %w", err)
		}
	}

	if req.Leverage < 1 {
		req.Leverage = 1
	}
	var order gen.Order

	// 3. Exact multiplication for trade values and fees
	totalValueDec := priceDec.Mul(qtyDec)
	feeDec := totalValueDec.Mul(decimal.NewFromFloat(0.003))

	fee, err := StringToNumeric(feeDec.String())
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid fee: %w", err)
	}

	totalCostDec := totalValueDec.Add(feeDec)
	netProceedsDec := totalValueDec.Sub(feeDec)

	totalCostNumeric, _ := StringToNumeric(totalCostDec.String())
	netProceedsNumeric, _ := StringToNumeric(netProceedsDec.String())

	err = s.store.ExecTx(ctx, func(q *gen.Queries) error {
		// ---------- Spot Market Order ----------
		if req.OrderType == "market" {
			if req.Side == "buy" {
				if _, err := q.DecreaseAvailableBalance(ctx, gen.DecreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     quote,
					Available: totalCostNumeric,
				}); err != nil {
					return ErrInsufficientBalance
				}

				if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
					UserID: UUIDToPGType(userID), Asset: base,
				}); err != nil {
					return fmt.Errorf("failed to upsert %s balance: %w", base, err)
				}
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     base,
					Available: quantityNumeric,
				}); err != nil {
					return fmt.Errorf("failed to credit %s: %w", base, err)
				}
			} else { // sell
				if _, err := q.DecreaseAvailableBalance(ctx, gen.DecreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     base,
					Available: quantityNumeric,
				}); err != nil {
					return ErrInsufficientBalance
				}

				if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
					UserID: UUIDToPGType(userID), Asset: quote,
				}); err != nil {
					return fmt.Errorf("failed to upsert %s balance: %w", quote, err)
				}
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     quote,
					Available: netProceedsNumeric,
				}); err != nil {
					return fmt.Errorf("failed to credit %s: %w", quote, err)
				}
			}

			created, err := q.CreateOrder(ctx, gen.CreateOrderParams{
				UserID:    UUIDToPGType(userID),
				Symbol:    req.Symbol,
				Side:      req.Side,
				OrderType: req.OrderType,
				Leverage:  req.Leverage,
				Price:     priceParam,
				Quantity:  quantityNumeric,
				Margin:    MustStringToNumeric("0"),
				Fee:       fee,
			})
			if err != nil {
				return err
			}

			filled, err := q.MarkOrderFilled(ctx, created.ID)
			if err != nil {
				return fmt.Errorf("failed to mark market order filled: %w", err)
			}
			order = filled

			if _, err := q.CreateTrade(ctx, gen.CreateTradeParams{
				OrderID:  created.ID,
				UserID:   UUIDToPGType(userID),
				Symbol:   req.Symbol,
				Price:    priceParam,
				Quantity: quantityNumeric,
				Fee:      fee,
			}); err != nil {
				return fmt.Errorf("failed to record trade: %w", err)
			}

			return nil
		}

		// ---------- Spot Limit Order ----------
		var lockAsset string
		var lockAmount pgtype.Numeric

		if req.Side == "buy" {
			lockAmount = totalCostNumeric
			lockAsset = quote
		} else {
			lockAmount = quantityNumeric
			lockAsset = base
		}

		rows, err := q.LockBalance(ctx, gen.LockBalanceParams{
			UserID: UUIDToPGType(userID),
			Asset:  lockAsset,
			Amount: lockAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to lock balance: %w", err)
		}
		if rows == 0 {
			return ErrInsufficientBalance
		}

		created, err := q.CreateOrder(ctx, gen.CreateOrderParams{
			UserID:    UUIDToPGType(userID),
			Symbol:    req.Symbol,
			Side:      req.Side,
			OrderType: req.OrderType,
			Leverage:  req.Leverage,
			Price:     priceParam,
			Quantity:  quantityNumeric,
			Margin:    lockAmount,
			Fee:       fee,
		})
		if err != nil {
			return err
		}

		order = created
		return nil
	})

	if err != nil {
		return gen.Order{}, err
	}

	// After the transaction commits, add the limit order to the in‑memory cache
	if req.OrderType == "limit" {
		if s.engine != nil {
			s.engine.AddOrder(order)
		} else {
			log.Printf("⚠️ limit order %s created but engine is nil — won't be monitored until next reload", order.ID)
		}
	}

	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error {
	var order gen.Order

	err := s.store.ExecTx(ctx, func(q *gen.Queries) error {
		o, err := q.GetOrderByID(ctx, UUIDToPGType(orderID))
		if err != nil {
			return err
		}
		if PGTypeToUUID(o.UserID) != userID {
			return errors.New("not authorized to cancel this order")
		}
		if o.Status != "open" {
			return errors.New("order is not open")
		}

		base, quote, err := splitSymbol(o.Symbol)
		if err != nil {
			return fmt.Errorf("invalid symbol: %w", err)
		}

		var lockAsset string
		if o.Side == "buy" {
			lockAsset = quote
		} else {
			lockAsset = base
		}

		if _, err := q.UnlockBalance(ctx, gen.UnlockBalanceParams{
			UserID: UUIDToPGType(userID),
			Asset:  lockAsset,
			Locked: o.Margin,
		}); err != nil {
			return fmt.Errorf("failed to unlock balance: %w", err)
		}

		// Mark order cancelled
		if _, err := q.CancelOrder(ctx, gen.CancelOrderParams{
			ID:     UUIDToPGType(orderID),
			UserID: UUIDToPGType(userID),
		}); err != nil {
			return err
		}

		order = o // store for later use
		return nil
	})

	if err != nil {
		return err
	}

	// Remove from cache if it's a limit order and the engine is present
	if order.OrderType == "limit" && s.engine != nil {
		s.engine.RemoveOrder(order.Symbol, orderID.String())
	}

	return nil
}

func (s *OrderService) GetOpenLimitOrders(ctx context.Context) ([]gen.Order, error) {
	return s.store.GetOpenLimitOrders(ctx)
}

// ExecuteLimitOrder fills a limit order.
func (s *OrderService) ExecuteLimitOrder(ctx context.Context, orderID uuid.UUID, execPrice float64) error {
	log.Printf("⚡ ExecuteLimitOrder: orderID=%s execPrice=%.4f", orderID, execPrice)
	// First, fetch the order outside the transaction to check status quickly
	order, err := s.store.Queries.GetOrderByID(ctx, UUIDToPGType(orderID))
	if err != nil {
		return err
	}
	if order.Status != "open" {
		// Already processed – remove from cache and exit
		if s.engine != nil {
			s.engine.RemoveOrder(order.Symbol, orderID.String())
		}
		return nil
	}

	execPriceDec := decimal.NewFromFloat(execPrice)
	execPriceNum, err := StringToNumeric(execPriceDec.String())
	if err != nil {
		return fmt.Errorf("invalid exec price: %w", err)
	}

	var filledOrder gen.Order

	err = s.store.ExecTx(ctx, func(q *gen.Queries) error {
		base, quote, err := splitSymbol(order.Symbol)
		if err != nil {
			return err
		}

		qtyStr, err := NumericToString(order.Quantity)
		if err != nil {
			return fmt.Errorf("invalid quantity: %w", err)
		}
		qtyDec, err := decimal.NewFromString(qtyStr)
		if err != nil {
			return fmt.Errorf("invalid quantity decimal: %w", err)
		}

		execValueDec := execPriceDec.Mul(qtyDec)
		execFeeDec := execValueDec.Mul(decimal.NewFromFloat(0.003))
		execFeeNum, err := StringToNumeric(execFeeDec.String())
		if err != nil {
			return fmt.Errorf("invalid fee: %w", err)
		}

		if order.Side == "buy" {
			if _, err := q.ConsumeLockedBalance(ctx, gen.ConsumeLockedBalanceParams{
				UserID: order.UserID,
				Asset:  quote,
				Locked: order.Margin,
			}); err != nil {
				return fmt.Errorf("failed to consume locked %s: %w", quote, err)
			}

			// Refund excess if filled at better price than locked for
			marginStr, err := NumericToString(order.Margin)
			if err != nil {
				return err
			}
			origLockedDec, err := decimal.NewFromString(marginStr)
			if err != nil {
				return err
			}
			totalCostDec := execValueDec.Add(execFeeDec)

			if origLockedDec.GreaterThan(totalCostDec) {
				refundDec := origLockedDec.Sub(totalCostDec)
				refundNum, _ := StringToNumeric(refundDec.String())
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID: order.UserID, Asset: quote, Available: refundNum,
				}); err != nil {
					return fmt.Errorf("failed to refund excess: %w", err)
				}
				log.Printf("💰 Refunded %.4f %s (locked %.4f, cost %.4f)", refundDec.InexactFloat64(), quote, origLockedDec.InexactFloat64(), totalCostDec.InexactFloat64())
			}
			// Never charge extra — margin discrepancy is absorbed

			// Credit base asset
			if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
				UserID: order.UserID, Asset: base,
			}); err != nil {
				return err
			}
			if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
				UserID: order.UserID, Asset: base, Available: order.Quantity,
			}); err != nil {
				return fmt.Errorf("failed to credit %s: %w", base, err)
			}

		} else { // sell
			// Consume locked base
			if _, err := q.ConsumeLockedBalance(ctx, gen.ConsumeLockedBalanceParams{
				UserID: order.UserID,
				Asset:  base,
				Locked: order.Margin,
			}); err != nil {
				return fmt.Errorf("failed to consume locked %s: %w", base, err)
			}

			// Credit net proceeds
			netProceedsDec := execValueDec.Sub(execFeeDec)
			netProceedsNum, _ := StringToNumeric(netProceedsDec.String())
			if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
				UserID: order.UserID, Asset: quote,
			}); err != nil {
				return err
			}
			if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
				UserID: order.UserID, Asset: quote, Available: netProceedsNum,
			}); err != nil {
				return fmt.Errorf("failed to credit %s: %w", quote, err)
			}
		}

		// Fill order
		filled, err := q.FillOrder(ctx, gen.FillOrderParams{
			ID:     order.ID,
			UserID: order.UserID,
			Price:  execPriceNum,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
				return fmt.Errorf("order already processed or not found")
			}
			return fmt.Errorf("failed to fill order: %w", err)
		}
		filledOrder = filled

		// Record trade
		if _, err := q.CreateTrade(ctx, gen.CreateTradeParams{
			OrderID:  filled.ID,
			UserID:   filled.UserID,
			Symbol:   filled.Symbol,
			Price:    execPriceNum,
			Quantity: filled.Quantity,
			Fee:      execFeeNum,
		}); err != nil {
			return fmt.Errorf("failed to record trade: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Remove from cache after successful fill
	if filledOrder.OrderType == "limit" && s.engine != nil {
		s.engine.RemoveOrder(filledOrder.Symbol, orderID.String())
	}

	return nil
}
