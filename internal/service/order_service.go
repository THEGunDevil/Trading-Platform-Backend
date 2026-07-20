package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidOrderType    = errors.New("limit orders require a price")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type OrderService struct {
	store *db.Store
}

func NewOrderService(store *db.Store) *OrderService {
	return &OrderService{store: store}
}

// splitSymbol splits "BTCUSDT" into ("BTC", "USDT").
// Extend knownQuotes if you support more quote assets.
func splitSymbol(symbol string) (base, quote string, err error) {
	knownQuotes := []string{"USDT", "USDC", "BUSD"}
	for _, q := range knownQuotes {
		if strings.HasSuffix(symbol, q) {
			return strings.TrimSuffix(symbol, q), q, nil
		}
	}
	return "", "", fmt.Errorf("unrecognized quote asset in symbol %s", symbol)
}

// PlaceOrder creates and optionally executes a spot order.
// Market orders are executed immediately.
// Limit orders lock the asset being sold (quote for buy, base for sell) and stay open.
func (s *OrderService) PlaceOrder(ctx context.Context, userID uuid.UUID, req models.PlaceOrderRequest) (gen.Order, error) {
	if req.OrderType == "limit" && req.Price == nil {
		return gen.Order{}, ErrInvalidOrderType
	}

	base, quote, err := splitSymbol(req.Symbol)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid symbol: %w", err)
	}

	// 1. Parse Quantity safely as a decimal
	qtyDec, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid quantity format: %w", err)
	}
	quantityNumeric, err := StringToNumeric(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid pg quantity: %w", err)
	}

	// 2. Determine Execution Price as a decimal
	var priceDec decimal.Decimal
	var priceParam pgtype.Numeric

	if req.OrderType == "market" {
		execPriceFloat, err := GetCurrentPrice(req.Symbol)
		if err != nil {
			return gen.Order{}, fmt.Errorf("failed to get current price: %w", err)
		}
		// Convert the float price to a precise decimal immediately
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
	fee := MustStringToNumeric("0")
	var order gen.Order

	// 3. Exact multiplication
	totalValueDec := priceDec.Mul(qtyDec)
	totalValueNumeric, err := StringToNumeric(totalValueDec.String())
	if err != nil {
		return gen.Order{}, fmt.Errorf("failed to convert total value to numeric: %w", err)
	}

	err = s.store.ExecTx(ctx, func(q *gen.Queries) error {
		// ---------- Spot Market Order (immediate execution) ----------
		if req.OrderType == "market" {
			if req.Side == "buy" {
				// Deduct quote (USDT) using the exact total value
				if _, err := q.DecreaseAvailableBalance(ctx, gen.DecreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     quote,
					Available: totalValueNumeric,
				}); err != nil {
					return ErrInsufficientBalance
				}
				
				// Credit base (BTC)
				if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
					UserID: UUIDToPGType(userID),
					Asset:  base,
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
				// Deduct base (BTC)
				if _, err := q.DecreaseAvailableBalance(ctx, gen.DecreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     base,
					Available: quantityNumeric,
				}); err != nil {
					return ErrInsufficientBalance
				}
				
				// Credit quote (USDT) with exact sale proceeds
				if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
					UserID: UUIDToPGType(userID),
					Asset:  quote,
				}); err != nil {
					return fmt.Errorf("failed to upsert %s balance: %w", quote, err)
				}
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID:    UUIDToPGType(userID),
					Asset:     quote,
					Available: totalValueNumeric,
				}); err != nil {
					return fmt.Errorf("failed to credit %s: %w", quote, err)
				}
			}

			// Create, fill, and record the trade
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
			if err != nil { return err }

			filled, err := q.FillOrder(ctx, gen.FillOrderParams{
				ID:     created.ID,
				UserID: UUIDToPGType(userID),
				Price:  priceParam,
			})
			if err != nil { return fmt.Errorf("failed to fill order: %w", err) }
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
			// Buying base (BTC) using quote (USDT) → lock total exact cost in USDT
			lockAmount = totalValueNumeric
			lockAsset = quote
		} else {
			// Selling base (BTC) → lock the exact BTC amount
			lockAmount = quantityNumeric
			lockAsset = base
		}

		if _, err := q.LockBalance(ctx, gen.LockBalanceParams{
			UserID: UUIDToPGType(userID),
			Asset:  lockAsset,
			Amount: lockAmount,
		}); err != nil {
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
		if err != nil { return err }
		
		order = created
		return nil
	})

	return order, err
}

// CancelOrder unlocks the reserved asset and marks the order cancelled.
// For a buy limit, it unlocks quote (USDT); for sell limit, it unlocks base (BTC).
func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *gen.Queries) error {
		order, err := q.GetOrderByID(ctx, UUIDToPGType(orderID))
		if err != nil {
			return err
		}
		if PGTypeToUUID(order.UserID) != userID {
			return errors.New("not authorized to cancel this order")
		}
		if order.Status != "open" {
			return errors.New("order is not open")
		}

		// Determine which asset was locked
		base, quote, err := splitSymbol(order.Symbol)
		if err != nil {
			return fmt.Errorf("invalid symbol: %w", err)
		}

		var lockAsset string
		if order.Side == "buy" {
			lockAsset = quote // USDT was locked for buy
		} else {
			lockAsset = base // BTC was locked for sell
		}

		// Unlock the funds (moving from locked back to available)
		if _, err := q.UnlockBalance(ctx, gen.UnlockBalanceParams{
			UserID: UUIDToPGType(userID),
			Asset:  lockAsset,
			Locked: order.Margin, // margin field holds the locked amount
		}); err != nil {
			return fmt.Errorf("failed to unlock balance: %w", err)
		}

		// Mark order as cancelled
		if _, err := q.CancelOrder(ctx, gen.CancelOrderParams{
			ID:     UUIDToPGType(orderID),
			UserID: UUIDToPGType(userID),
		}); err != nil {
			return err
		}

		return nil
	})
}