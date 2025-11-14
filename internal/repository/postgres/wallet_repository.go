package postgres

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository interface {
	// ✅ НОВЫЕ методы - упрощённые
	GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error)
	UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error

	// Существующие методы
	OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error)
	CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
}

func NewWalletRepository(db *pgxpool.Pool) WalletRepository {
	return &PgWalletRepository{db: db}
}

func (r *PgWalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "repository.GetByID"
	var wallet models.Wallet
	err := r.db.QueryRow(ctx, repository.GetWalletByIDQuery, id).Scan(
		&wallet.ID, &wallet.Balance, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}
