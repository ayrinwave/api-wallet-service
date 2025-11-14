//go:build integration
// +build integration

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository/postgres"
	"api_wallet/internal/testhelpers"
)

func setupService(t *testing.T) (*WalletService, *testhelpers.TestDB) {
	t.Helper()

	testDB := testhelpers.SetupTestDB(t)
	testDB.RunMigrations(t)
	testDB.CleanupDB(t)

	repo := postgres.NewWalletRepository(testDB.Pool)
	txManager := NewPgxTxManager(testDB.Pool)
	service := NewWalletService(repo, txManager)

	return service, testDB
}

func TestUpdateBalance_Integration_Deposit_Success(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        500,
		RequestID:     "deposit-test-1",
	}

	// Act
	err := service.UpdateBalance(context.Background(), request)

	// Assert
	require.NoError(t, err)

	// Проверяем что баланс обновился
	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), wallet.Balance)
}

func TestUpdateBalance_Integration_Withdraw_Success(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationWithdraw,
		Amount:        300,
		RequestID:     "withdraw-test-1",
	}

	// Act
	err := service.UpdateBalance(context.Background(), request)

	// Assert
	require.NoError(t, err)

	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(700), wallet.Balance)
}

func TestUpdateBalance_Integration_InsufficientFunds(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 100)

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationWithdraw,
		Amount:        500,
		RequestID:     "insufficient-test-1",
	}

	// Act
	err := service.UpdateBalance(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, custom_err.ErrInsufficientFunds)

	// Баланс не должен измениться
	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), wallet.Balance)
}

func TestUpdateBalance_Integration_WalletNotFound(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	nonExistentID := uuid.New()

	request := models.WalletOperationRequest{
		WalletID:      nonExistentID,
		OperationType: models.OperationDeposit,
		Amount:        100,
		RequestID:     "notfound-test-1",
	}

	// Act
	err := service.UpdateBalance(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, custom_err.ErrNotFound)
}

func TestUpdateBalance_Integration_Idempotency(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        100,
		RequestID:     "idempotent-request",
	}

	// Act - первый запрос
	err := service.UpdateBalance(context.Background(), request)
	require.NoError(t, err)

	// Act - повторный запрос с тем же requestID
	err = service.UpdateBalance(context.Background(), request)

	// Assert
	assert.NoError(t, err) // Не должно быть ошибки (идемпотентность)

	// Баланс должен увеличиться только один раз
	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(1100), wallet.Balance)

	// Должна быть только одна операция
	var count int
	err = testDB.Pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM operations WHERE request_id = $1",
		"idempotent-request",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpdateBalance_Integration_ConcurrentDeposits(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	initialBalance := int64(1000)
	testDB.SeedWallet(t, walletID.String(), initialBalance)

	numGoroutines := 100
	depositAmount := int64(10)

	// Act - 100 параллельных депозитов
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			request := models.WalletOperationRequest{
				WalletID:      walletID,
				OperationType: models.OperationDeposit,
				Amount:        depositAmount,
				RequestID:     uuid.New().String(),
			}

			errors[idx] = service.UpdateBalance(context.Background(), request)
		}(i)
	}

	wg.Wait()

	// Assert
	// Все операции должны быть успешными
	for i, err := range errors {
		assert.NoError(t, err, "Operation %d failed", i)
	}

	// Финальный баланс должен быть правильным
	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)

	expectedBalance := initialBalance + (depositAmount * int64(numGoroutines))
	assert.Equal(t, expectedBalance, wallet.Balance)

	// Должно быть ровно 100 операций
	var operationCount int
	err = testDB.Pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM operations WHERE wallet_id = $1",
		walletID,
	).Scan(&operationCount)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines, operationCount)
}

func TestUpdateBalance_Integration_MixedOperations(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	initialBalance := int64(10000)
	testDB.SeedWallet(t, walletID.String(), initialBalance)

	numOperations := 50

	// Act - 50 депозитов и 50 снятий параллельно
	var wg sync.WaitGroup
	errors := make([]error, numOperations*2)

	for i := 0; i < numOperations; i++ {
		// Deposit
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			request := models.WalletOperationRequest{
				WalletID:      walletID,
				OperationType: models.OperationDeposit,
				Amount:        100,
				RequestID:     uuid.New().String(),
			}

			errors[idx] = service.UpdateBalance(context.Background(), request)
		}(i)

		// Withdraw
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			request := models.WalletOperationRequest{
				WalletID:      walletID,
				OperationType: models.OperationWithdraw,
				Amount:        50,
				RequestID:     uuid.New().String(),
			}

			errors[numOperations+idx] = service.UpdateBalance(context.Background(), request)
		}(i)
	}

	wg.Wait()

	// Assert
	for i, err := range errors {
		assert.NoError(t, err, "Operation %d failed", i)
	}

	// Финальный баланс: 10000 + (100*50) - (50*50) = 10000 + 5000 - 2500 = 12500
	wallet, err := service.GetWalletByID(context.Background(), walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(12500), wallet.Balance)
}

func TestGetWalletByID_Integration(t *testing.T) {
	// Arrange
	service, testDB := setupService(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 5000)

	// Act
	wallet, err := service.GetWalletByID(context.Background(), walletID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, int64(5000), wallet.Balance)
	assert.NotZero(t, wallet.CreatedAt)
	assert.NotZero(t, wallet.UpdatedAt)
}
