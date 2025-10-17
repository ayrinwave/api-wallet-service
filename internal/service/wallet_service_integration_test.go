package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"api_wallet/internal/models"
	"api_wallet/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIntegrationTest(t *testing.T) (*pgxpool.Pool, func()) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"127.0.0.1",
		"5432",
		"user",
		"password",
		"wallet",
	)

	if envDsn := os.Getenv("TEST_DATABASE_URL"); envDsn != "" {
		dsn = envDsn
	}

	var pool *pgxpool.Pool
	var err error

	for i := 0; i < 5; i++ {
		pool, err = pgxpool.New(context.Background(), dsn)
		if err == nil {
			if err = pool.Ping(context.Background()); err == nil {
				break
			}
		}
		t.Logf("Attempt %d failed to connect to database: %v. Retrying...", i+1, err)
		time.Sleep(1 * time.Second)
	}
	require.NoError(t, err, "Failed to connect to database after multiple retries")

	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE wallets RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

func TestWalletService_Integration_Flush(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupIntegrationTest(t)
	defer cleanup()

	repo := postgres.NewWalletRepository(pool)
	service := NewWalletService(repo, pool)

	walletID := uuid.New()
	initialBalance := int64(1000)

	_, err := pool.Exec(context.Background(), "INSERT INTO wallets (id, balance) VALUES ($1, $2)", walletID, initialBalance)
	require.NoError(t, err)

	amountToAdd := int64(500)
	req := models.WalletOperationRequest{WalletID: walletID, OperationType: models.DepositOperation, Amount: amountToAdd}
	err = service.UpdateBalance(context.Background(), req)
	require.NoError(t, err)

	getBalanceFromDB := func() int64 {
		var balance int64
		err := pool.QueryRow(context.Background(), "SELECT balance FROM wallets WHERE id = $1", walletID).Scan(&balance)
		require.NoError(t, err)
		return balance
	}

	currentDBBalance := getBalanceFromDB()
	assert.Equal(t, initialBalance, currentDBBalance, "Balance in DB should NOT be updated immediately")

	t.Log("Waiting for flusher to run...")
	time.Sleep(1500 * time.Millisecond)

	finalDBBalance := getBalanceFromDB()
	expectedBalance := initialBalance + amountToAdd
	assert.Equal(t, expectedBalance, finalDBBalance, "Balance in DB SHOULD be updated by flusher")

	t.Log("Flusher successfully updated the balance in the database!")
}
