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

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
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

func (r *WalletRepository) UpsertWalletBalance(ctx context.Context, id uuid.UUID, balance int64) error {
	const op = "repository.UpsertWalletBalance"
	_, err := r.db.Exec(ctx, repository.UpsertWalletBalanceQuery, id, balance)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
