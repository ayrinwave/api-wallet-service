package service

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}
type PgxPoolIface interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type PgxTxManager struct {
	pool PgxPoolIface
}

func NewPgxTxManager(pool PgxPoolIface) *PgxTxManager {
	return &PgxTxManager{pool: pool}
}

func (m *PgxTxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
