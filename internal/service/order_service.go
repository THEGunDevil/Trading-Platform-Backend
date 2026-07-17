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
)

var (
	ErrInvalidOrderType = errors.New("limit orders require a price")
	// ErrInsufficientBalance = errors.New("insufficient balance")
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

// PlaceOrder locks the required margin and creates the order atomically.
// If locking fails (insufficient balance) or order creation fails, nothing
// is committed.
func (s *OrderService) PlaceOrder(ctx context.Context, userID uuid.UUID, req models.PlaceOrderRequest) (gen.Order, error) {
	if req.OrderType == "limit" && req.Price == nil {
		return gen.Order{}, ErrInvalidOrderType
	}

	base, quote, err := splitSymbol(req.Symbol)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid symbol: %w", err)
	}

	quantityNumeric, err := StringToNumeric(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid quantity: %w", err)
	}
	quantityFloat := NumericToFloat64(quantityNumeric) // no error, matches prediction.go usage

	var execPrice float64
	var priceParam pgtype.Numeric

	if req.OrderType == "market" {
		execPrice, err = GetCurrentPrice(req.Symbol)
		if err != nil {
			return gen.Order{}, fmt.Errorf("failed to get current price: %w", err)
		}
		priceParam, err = Float64ToNumeric(execPrice)
		if err != nil {
			return gen.Order{}, fmt.Errorf("invalid price: %w", err)
		}
	} else {
		priceParam, err = StringToNumeric(*req.Price)
		if err != nil {
			return gen.Order{}, fmt.Errorf("invalid price: %w", err)
		}
		execPrice = NumericToFloat64(priceParam)
	}

	if req.Leverage < 1 {
		req.Leverage = 1
	}
	marginFloat := (execPrice * quantityFloat) / float64(req.Leverage)
	marginAmount, err := Float64ToNumeric(marginFloat)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid margin: %w", err)
	}

	fee := MustStringToNumeric("0")

	var order gen.Order

	err = s.store.ExecTx(ctx, func(q *gen.Queries) error {
		if err := LockForOrder(ctx, q, userID, quote, marginAmount); err != nil {
			return err // already ErrInsufficientBalance — no need to wrap
		}

		created, err := q.CreateOrder(ctx, gen.CreateOrderParams{
			UserID:    UUIDToPGType(userID),
			Symbol:    req.Symbol,
			Side:      req.Side,
			OrderType: req.OrderType,
			Leverage:  req.Leverage,
			Price:     priceParam,
			Quantity:  quantityNumeric,
			Margin:    marginAmount,
			Fee:       fee,
		})
		if err != nil {
			return err
		}
		order = created

		if req.OrderType == "market" {
			filled, err := db.Q.FillOrder(ctx, gen.FillOrderParams{
				ID: created.ID, UserID: UUIDToPGType(userID), Price: priceParam,
			})
			if err != nil {
				return fmt.Errorf("failed to fill order: %w", err)
			}
			order = filled

			if _, err := q.CreateTrade(ctx, gen.CreateTradeParams{
				OrderID: created.ID, UserID: UUIDToPGType(userID),
				Symbol: req.Symbol, Price: priceParam, Quantity: quantityNumeric, Fee: fee,
			}); err != nil {
				return fmt.Errorf("failed to record trade: %w", err)
			}

			// Release locked margin, then credit whichever asset was received
			if err := UnlockBalance(ctx, q, userID, quote, marginAmount); err != nil {
				return fmt.Errorf("failed to release margin: %w", err)
			}

			if req.Side == "buy" {
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID: UUIDToPGType(userID), Asset: base, Available: quantityNumeric,
				}); err != nil {
					return fmt.Errorf("failed to credit %s: %w", base, err)
				}
			} else {
				if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
					UserID: UUIDToPGType(userID), Asset: quote, Available: marginAmount,
				}); err != nil {
					return fmt.Errorf("failed to credit %s: %w", quote, err)
				}
			}
		}

		return nil
	})

	return order, err
}

// CancelOrder unlocks the reserved margin and marks the order cancelled,
// atomically. Ownership is enforced both here and inside the SQL query
// itself (WHERE user_id = $2).
func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error {
	// Using s.store.ExecTx instead of db.WithTx
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

		if _, err := q.CancelOrder(ctx, gen.CancelOrderParams{
			ID:     UUIDToPGType(orderID),
			UserID: UUIDToPGType(userID),
		}); err != nil {
			return err
		}

		marginAsset := quoteAssetFromSymbol(order.Symbol)

		if err := UnlockBalance(ctx, q, userID, marginAsset, order.Margin); err != nil {
			return err
		}

		return nil
	})
}

// quoteAssetFromSymbol is a placeholder — replace with real symbol parsing
// matching how you store trading pairs (e.g. "BTCUSDT" -> "USDT").
func quoteAssetFromSymbol(symbol string) string {
	return "USDT"
}
