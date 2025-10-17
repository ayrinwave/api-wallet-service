package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"api_wallet/internal/custom_err"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRepoTest(t *testing.T) (*pgxpool.Pool, func()) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"127.0.0.1", "5432", "user", "password", "wallet")

	if envDsn := os.Getenv("TEST_DATABASE_URL"); envDsn != "" {
		dsn = envDsn
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "Failed to connect to database")

	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE wallets, operations RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

func TestWalletRepository_OptimisticLockingTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	pool, cleanup := setupRepoTest(t)
	defer cleanup()

	repo := NewWalletRepository(pool)
	ctx := context.Background()

	walletID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO wallets (id, balance, version) VALUES ($1, $2, 1)", walletID, 100)
	require.NoError(t, err)

	t.Run("GetWalletStateTx", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		balance, version, err := repo.GetWalletStateTx(ctx, tx, walletID)
		require.NoError(t, err)
		assert.Equal(t, int64(100), balance)
		assert.Equal(t, int64(1), version)

		_, _, err = repo.GetWalletStateTx(ctx, tx, uuid.New())
		assert.ErrorIs(t, err, custom_err.ErrNotFound)
	})

	t.Run("UpdateBalanceWithOptimisticLockTx", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		err = repo.UpdateBalanceWithOptimisticLockTx(ctx, tx, walletID, 200, 1)
		require.NoError(t, err)

		err = repo.UpdateBalanceWithOptimisticLockTx(ctx, tx, walletID, 300, 1)
		assert.ErrorIs(t, err, custom_err.ErrConflict)
	})

	t.Run("Idempotency (Check and Create Operation)", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		requestID := uuid.New()

		exists, err := repo.CheckOperationExistsTx(ctx, tx, requestID)
		require.NoError(t, err)
		assert.False(t, exists)

		err = repo.CreateOperationTx(ctx, tx, walletID, 50, requestID)
		require.NoError(t, err)

		exists, err = repo.CheckOperationExistsTx(ctx, tx, requestID)
		require.NoError(t, err)
		assert.True(t, exists)

		err = repo.CreateOperationTx(ctx, tx, walletID, 50, requestID)
		assert.ErrorIs(t, err, custom_err.ErrDuplicateRequest)
	})
}
