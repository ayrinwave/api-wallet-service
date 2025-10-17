package postgres

import (
	"context"
	"testing"

	"api_wallet/internal/custom_err"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	pool, cleanup := setupRepoTest(t)
	defer cleanup()

	repo := NewWalletRepository(pool)
	ctx := context.Background()

	walletID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO wallets (id, balance, version) VALUES ($1, 123, 1)", walletID)
	require.NoError(t, err)

	wallet, err := repo.GetByID(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, int64(123), wallet.Balance)

	_, err = repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, custom_err.ErrNotFound)
}
