package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
)

var (
	ErrInvalidOrderType = errors.New("limit orders require a price")
)

// OrderService handles order lifecycle operations with safe database transactions.
type OrderService struct {
	store *db.Store
}

// NewOrderService initializes a new OrderService with the DB store.
func NewOrderService(store *db.Store) *OrderService {
	return &OrderService{store: store}
}

// PlaceOrder locks the required margin and creates the order atomically.
// If locking fails (insufficient balance) or order creation fails, nothing
// is committed.
func (s *OrderService) PlaceOrder(ctx context.Context, userID uuid.UUID, req models.PlaceOrderRequest) (gen.Order, error) {
	if req.OrderType == "limit" && req.Price == nil {
		return gen.Order{}, ErrInvalidOrderType
	}

	// TODO: replace with real margin/fee calculation logic — placeholder
	// values below just wire the plumbing together for now.
	marginAsset := "USDT"
	marginAmount, err := StringToNumeric(req.Quantity)
	if err != nil {
		return gen.Order{}, fmt.Errorf("invalid quantity: %w", err)
	}
	fee := MustStringToNumeric("0")

	var order gen.Order

	// Using s.store.ExecTx instead of db.WithTx
	err = s.store.ExecTx(ctx, func(q *gen.Queries) error {
		if err := LockForOrder(ctx, q, userID, marginAsset, marginAmount); err != nil {
			return err
		}

		var priceParam pgtype.Numeric
		if req.Price != nil {
			p, err := StringToNumeric(*req.Price)
			if err != nil {
				return fmt.Errorf("invalid price: %w", err)
			}
			priceParam = p
		}

		quantityParam, err := StringToNumeric(req.Quantity)
		if err != nil {
			return fmt.Errorf("invalid quantity: %w", err)
		}

		created, err := q.CreateOrder(ctx, gen.CreateOrderParams{
			UserID:    UUIDToPGType(userID),
			Symbol:    req.Symbol,
			Side:      req.Side,
			OrderType: req.OrderType,
			Leverage:  req.Leverage,
			Price:     priceParam,
			Quantity:  quantityParam,
			Margin:    marginAmount,
			Fee:       fee,
		})
		if err != nil {
			return err
		}

		order = created
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