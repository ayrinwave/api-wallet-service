package repository

import (
	"context"

	"api_wallet/internal/models"

	"github.com/google/uuid"
)

type Wallet interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	BulkUpdateBalances(ctx context.Context, wallets map[uuid.UUID]int64) error
	UpsertWalletBalance(ctx context.Context, id uuid.UUID, balance int64) error
}
