package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletRepository_BulkUpdateBalances(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	pool, cleanup := setupRepoTest(t)
	defer cleanup()

	repo := NewWalletRepository(pool)
	ctx := context.Background()

	existingWalletID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO wallets (id, balance) VALUES ($1, $2)", existingWalletID, 1000)
	require.NoError(t, err)

	newWalletID := uuid.New()

	walletsToUpdate := map[uuid.UUID]int64{
		existingWalletID: 250,
		newWalletID:      500,
	}

	err = repo.BulkUpdateBalances(ctx, walletsToUpdate)
	require.NoError(t, err)

	var existingBalance int64
	err = pool.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1", existingWalletID).Scan(&existingBalance)
	require.NoError(t, err)
	assert.Equal(t, int64(250), existingBalance, "Balance of existing wallet should be updated")

	var newBalance int64
	err = pool.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1", newWalletID).Scan(&newBalance)
	require.NoError(t, err)
	assert.Equal(t, int64(500), newBalance, "New wallet should be created with correct balance")
}
