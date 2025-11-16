package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgxTxManager_WithTx_Success(t *testing.T) {
	// Arrange
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	// Ожидаем начало транзакции
	mock.ExpectBegin()
	// Ожидаем commit
	mock.ExpectCommit()

	// Act
	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// Успешное выполнение
		return nil
	})

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_FunctionError_Rollback(t *testing.T) {
	// Arrange
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("business logic error")

	// Ожидаем начало транзакции
	mock.ExpectBegin()
	// Ожидаем rollback при ошибке
	mock.ExpectRollback()

	// Act
	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		return expectedErr
	})

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_BeginError(t *testing.T) {
	// Arrange
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("cannot begin transaction")

	// Ожидаем ошибку при начале транзакции
	mock.ExpectBegin().WillReturnError(expectedErr)

	// Act
	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		t.Fatal("function should not be called")
		return nil
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_CommitError(t *testing.T) {
	// Arrange
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("cannot commit transaction")

	// Ожидаем начало транзакции
	mock.ExpectBegin()
	// Ожидаем ошибку при commit
	mock.ExpectCommit().WillReturnError(expectedErr)

	// Act
	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		return nil
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_ContextCanceled(t *testing.T) {
	// Arrange
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx, cancel := context.WithCancel(context.Background())

	// Отменяем контекст сразу
	cancel()

	// Ожидаем попытку начать транзакцию с отменённым контекстом
	mock.ExpectBegin().WillReturnError(context.Canceled)

	// Act
	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		t.Fatal("function should not be called")
		return nil
	})

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
