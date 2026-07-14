package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
)

var ErrWithdrawalBelowMinimum = errors.New("amount is below the minimum withdrawal")

// WithdrawalService handles withdrawal requests and status updates with transaction safety.
type WithdrawalService struct {
	store *db.Store
}

// NewWithdrawalService initializes a new WithdrawalService with the DB store.
func NewWithdrawalService(store *db.Store) *WithdrawalService {
	return &WithdrawalService{store: store}
}

func (s *WithdrawalService) RequestWithdrawal(
	ctx context.Context,
	userID uuid.UUID,
	req models.CreateWithdrawalRequest,
	amount, fee, totalDebit pgtype.Numeric,
) (gen.Withdrawal, error) {
	var withdrawal gen.Withdrawal
	userIDPg := UUIDToPGType(userID)

	// Using s.store.ExecTx instead of db.WithTx
	err := s.store.ExecTx(ctx, func(q *gen.Queries) error {
		if _, err := q.DecreaseAvailableBalance(ctx, gen.DecreaseAvailableBalanceParams{
			UserID:    userIDPg,
			Asset:     req.Asset,
			Available: totalDebit,
		}); err != nil {
			return ErrInsufficientBalance
		}

		created, err := q.CreateWithdrawal(ctx, gen.CreateWithdrawalParams{
			UserID:             userIDPg,
			Asset:              req.Asset,
			Network:            req.Network,
			DestinationAddress: req.DestinationAddress,
			Amount:             amount,
			Fee:                fee,
		})
		if err != nil {
			return err
		}

		withdrawal = created
		return nil
	})

	return withdrawal, err
}

func (s *WithdrawalService) RejectWithdrawal(ctx context.Context, withdrawalID uuid.UUID) error {
	withdrawalIDPg := UUIDToPGType(withdrawalID)

	// Using s.store.ExecTx instead of db.WithTx
	return s.store.ExecTx(ctx, func(q *gen.Queries) error {
		w, err := q.GetWithdrawalByID(ctx, withdrawalIDPg)
		if err != nil {
			return err
		}
		if w.Status != "pending" {
			return errors.New("withdrawal is not pending")
		}

		if _, err := q.MarkWithdrawalRejected(ctx, withdrawalIDPg); err != nil {
			return err
		}

		// Refund the balance.
		// Note: If you debited 'totalDebit' (amount + fee) during creation, 
		// make sure you refund the full total (amount + fee) here instead of just w.Amount!
		if _, err := q.IncreaseAvailableBalance(ctx, gen.IncreaseAvailableBalanceParams{
			UserID:    w.UserID,
			Asset:     w.Asset,
			Available: w.Amount, // adjust if you need to refund the fee as well
		}); err != nil {
			return err
		}

		return nil
	})
}