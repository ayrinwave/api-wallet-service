//go:build integration
// +build integration

package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"api_wallet/internal/custom_err"
	"api_wallet/internal/testhelpers"
)

func setupRepository(t *testing.T) (*PgWalletRepository, *testhelpers.TestDB) {
	t.Helper()

	testDB := testhelpers.SetupTestDB(t)
	testDB.RunMigrations(t)
	testDB.CleanupDB(t)

	repo := &PgWalletRepository{db: testDB.Pool}

	return repo, testDB
}

func TestGetByID_Success(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	// Act
	wallet, err := repo.GetByID(context.Background(), walletID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, int64(1000), wallet.Balance)
	assert.NotZero(t, wallet.CreatedAt)
	assert.NotZero(t, wallet.UpdatedAt)
}

func TestGetByID_NotFound(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	nonExistentID := uuid.New()

	// Act
	wallet, err := repo.GetByID(context.Background(), nonExistentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, wallet)
	assert.ErrorIs(t, err, custom_err.ErrNotFound)
}

func TestGetWalletBalanceForUpdateTx_Success(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 5000)

	// Начинаем транзакцию
	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	balance, err := repo.GetWalletBalanceForUpdateTx(ctx, tx, walletID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(5000), balance)
}

func TestGetWalletBalanceForUpdateTx_NotFound(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	nonExistentID := uuid.New()

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	balance, err := repo.GetWalletBalanceForUpdateTx(ctx, tx, nonExistentID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, int64(0), balance)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestUpdateBalanceTx_Success(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	err = repo.UpdateBalanceTx(ctx, tx, walletID, 1500)

	// Assert
	require.NoError(t, err)

	// Проверяем что баланс обновился
	var newBalance int64
	err = tx.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1", walletID).Scan(&newBalance)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), newBalance)
}

func TestUpdateBalanceTx_NegativeBalance_ConstraintViolation(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act - пытаемся установить отрицательный баланс
	err = repo.UpdateBalanceTx(ctx, tx, walletID, -100)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, custom_err.ErrInsufficientFunds)
}

func TestUpdateBalanceTx_WalletNotFound(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	nonExistentID := uuid.New()

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	err = repo.UpdateBalanceTx(ctx, tx, nonExistentID, 1000)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, custom_err.ErrNotFound)
}

func TestOperationExistsTx_Exists(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-request-exists"

	testDB.SeedWallet(t, walletID.String(), 1000)

	// Создаём операцию
	_, err := testDB.Pool.Exec(ctx,
		"INSERT INTO operations (wallet_id, amount, request_id) VALUES ($1, $2, $3)",
		walletID, 100, requestID,
	)
	require.NoError(t, err)

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	exists, err := repo.OperationExistsTx(ctx, tx, requestID)

	// Assert
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestOperationExistsTx_NotExists(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	exists, err := repo.OperationExistsTx(ctx, tx, "non-existent-request")

	// Assert
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateOperationTx_Success(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-operation-1"

	testDB.SeedWallet(t, walletID.String(), 1000)

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act
	err = repo.CreateOperationTx(ctx, tx, walletID, 100, requestID)

	// Assert
	require.NoError(t, err)

	// Проверяем что операция создалась
	var count int
	err = tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM operations WHERE wallet_id = $1 AND request_id = $2",
		walletID, requestID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCreateOperationTx_DuplicateRequest(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "duplicate-request"

	testDB.SeedWallet(t, walletID.String(), 1000)

	// Создаём первую операцию
	_, err := testDB.Pool.Exec(ctx,
		"INSERT INTO operations (wallet_id, amount, request_id) VALUES ($1, $2, $3)",
		walletID, 100, requestID,
	)
	require.NoError(t, err)

	tx, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Act - пытаемся создать дубликат
	err = repo.CreateOperationTx(ctx, tx, walletID, 100, requestID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, custom_err.ErrDuplicateRequest)
}

func TestConcurrentOperations_ForUpdateLocks(t *testing.T) {
	// Arrange
	repo, testDB := setupRepository(t)
	defer testDB.TeardownTestDB()

	ctx := context.Background()
	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	// Транзакция 1 - получает блокировку
	tx1, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	balance1, err := repo.GetWalletBalanceForUpdateTx(ctx, tx1, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance1)

	// Транзакция 2 - должна ждать освобождения блокировки
	tx2, err := testDB.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	// Создаём канал для результата
	done := make(chan bool)

	go func() {
		// Эта операция должна заблокироваться
		_, err := repo.GetWalletBalanceForUpdateTx(ctx, tx2, walletID)
		if err != nil {
			t.Logf("tx2 error: %v", err)
		}
		done <- true
	}()

	// Обновляем в tx1
	err = repo.UpdateBalanceTx(ctx, tx1, walletID, 1100)
	require.NoError(t, err)

	// Commit tx1 - освобождаем блокировку
	err = tx1.Commit(ctx)
	require.NoError(t, err)

	// Теперь tx2 должна получить доступ
	<-done

	// Проверяем финальный баланс
	var finalBalance int64
	err = testDB.Pool.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1", walletID).Scan(&finalBalance)
	require.NoError(t, err)
	assert.Equal(t, int64(1100), finalBalance)
}
