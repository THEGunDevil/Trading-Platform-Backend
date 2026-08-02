package db

import (
	"context"
	"fmt"

	gen "github.com/internal/db/gen"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*gen.Queries
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: gen.New(pool),
		pool:    pool,
	}
}

// ExecTx runs fn within a database transaction, committing on success and
// rolling back on any error (including a panic recovered by pgx itself).
func (s *Store) ExecTx(ctx context.Context, fn func(*gen.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if already committed

	q := s.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func (s *Store) GetOpenLimitOrders(ctx context.Context) ([]gen.Order, error) {
	return s.Queries.GetOpenLimitOrders(ctx)
}
