package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
)

type Store interface {
	db.Querier
	ExecTx(ctx context.Context, fn func(q db.Querier) error) error
}

type SQLStore struct {
	pool *pgxpool.Pool
	*db.Queries
}

func NewStore(pool *pgxpool.Pool) Store {
	return &SQLStore{
		pool:    pool,
		Queries: db.New(pool),
	}
}

func (s *SQLStore) ExecTx(ctx context.Context, fn func(q db.Querier) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := db.New(tx)
	if err := fn(q); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}
