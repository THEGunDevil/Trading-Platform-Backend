package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
)

// BalanceService encapsulates balance-related operations and holds a reference to the db.Store.
type BalanceService struct {
	store *db.Store
}

// NewBalanceService creates a new BalanceService instance.
func NewBalanceService(store *db.Store) *BalanceService {
	return &BalanceService{store: store}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID uuid.UUID, asset string) (gen.Balance, error) {
	return s.store.GetBalance(ctx, gen.GetBalanceParams{
		UserID: UUIDToPGType(userID),
		Asset:  asset,
	})
}

func (s *BalanceService) ListBalances(ctx context.Context, userID uuid.UUID) ([]gen.Balance, error) {
	return s.store.ListBalances(ctx, UUIDToPGType(userID))
}

// CreditDeposit increases available balance and marks a deposit confirmed,
// atomically — a deposit should never be marked confirmed without funds
// actually landing, or vice versa.
func (s *BalanceService) CreditDeposit(ctx context.Context, depositID uuid.UUID, userID uuid.UUID, asset string, amount pgtype.Numeric) error {
	userIDPg := UUIDToPGType(userID)

	// Using the ExecTx method from the Store instance
	return s.store.ExecTx(ctx, func(q *gen.Queries) error {
		if _, err := q.UpsertBalance(ctx, gen.UpsertBalanceParams{
			UserID: userIDPg,
			Asset:  asset,
		}); err != nil {
			return err
		}
		if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
			UserID:    userIDPg,
			Asset:     asset,
			Available: amount,
		}); err != nil {
			return err
		}
		if _, err := q.MarkDepositConfirmed(ctx, UUIDToPGType(depositID)); err != nil {
			return err
		}
		return nil
	})
}

// LockForOrder moves funds available -> locked.
func LockForOrder(ctx context.Context, q *gen.Queries, userID uuid.UUID, asset string, amount pgtype.Numeric) error {
    rows, err := q.LockBalance(ctx, gen.LockBalanceParams{
        UserID:  UUIDToPGType(userID),
        Asset:   asset,
        Amount:  amount,
    })
    if err != nil {
        return err
    }
    if rows == 0 {
        return ErrInsufficientBalance
    }
    return nil
}

// UnlockBalance unlocks the reserved margin.
func UnlockBalance(ctx context.Context, q *gen.Queries, userID uuid.UUID, asset string, amount pgtype.Numeric) error {
	_, err := q.UnlockBalance(ctx, gen.UnlockBalanceParams{
		UserID: UUIDToPGType(userID),
		Asset:  asset,
		Locked: amount,
	})
	return err
}

// ToBalanceModel converts a sqlc-generated gen.Balance (pgtype fields) into
// the clean, JSON-friendly models.Balance the API actually returns.
func ToBalanceModel(b gen.Balance) (models.Balance, error) {
	available, err := NumericToString(b.Available)
	if err != nil {
		return models.Balance{}, err
	}
	locked, err := NumericToString(b.Locked)
	if err != nil {
		return models.Balance{}, err
	}

	return models.Balance{
		UserID:    PGTypeToUUID(b.UserID),
		Asset:     b.Asset,
		Available: available,
		Locked:    locked,
		UpdatedAt: b.UpdatedAt.Time,
	}, nil
}

// ToBalanceModels converts a slice, for list endpoints.
func ToBalanceModels(balances []gen.Balance) ([]models.Balance, error) {
	result := make([]models.Balance, 0, len(balances))
	for _, b := range balances {
		m, err := ToBalanceModel(b)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}