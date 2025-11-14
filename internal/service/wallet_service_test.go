package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tx, walletID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletRepository) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error {
	args := m.Called(ctx, tx, walletID, newBalance)
	return args.Error(0)
}

func (m *MockWalletRepository) OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error) {
	args := m.Called(ctx, tx, requestID)
	return args.Bool(0), args.Error(1)
}

func (m *MockWalletRepository) CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error {
	args := m.Called(ctx, tx, walletID, amount, requestID)
	return args.Error(0)
}

func (m *MockWalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

// Mock TxManager
type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, fn)

	// Выполняем переданную функцию с nil tx (мы мокируем repository)
	if args.Error(0) == nil {
		return fn(nil)
	}
	return args.Error(0)
}

// Тесты
func TestUpdateBalance_Success_Deposit(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-request-1"

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        100,
		RequestID:     requestID,
	}

	// Mock expectations
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("OperationExistsTx", mock.Anything, mock.Anything, requestID).Return(false, nil)
	mockRepo.On("GetWalletBalanceForUpdateTx", mock.Anything, mock.Anything, walletID).Return(int64(1000), nil)
	mockRepo.On("UpdateBalanceTx", mock.Anything, mock.Anything, walletID, int64(1100)).Return(nil)
	mockRepo.On("CreateOperationTx", mock.Anything, mock.Anything, walletID, int64(100), requestID).Return(nil)

	// Act
	err := service.UpdateBalance(ctx, request)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
}

func TestUpdateBalance_Success_Withdraw(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-request-2"

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationWithdraw,
		Amount:        100,
		RequestID:     requestID,
	}

	// Mock expectations
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("OperationExistsTx", mock.Anything, mock.Anything, requestID).Return(false, nil)
	mockRepo.On("GetWalletBalanceForUpdateTx", mock.Anything, mock.Anything, walletID).Return(int64(1000), nil)
	mockRepo.On("UpdateBalanceTx", mock.Anything, mock.Anything, walletID, int64(900)).Return(nil)
	mockRepo.On("CreateOperationTx", mock.Anything, mock.Anything, walletID, int64(-100), requestID).Return(nil)

	// Act
	err := service.UpdateBalance(ctx, request)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateBalance_InsufficientFunds(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-request-3"

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationWithdraw,
		Amount:        2000, // Больше чем баланс
		RequestID:     requestID,
	}

	// Mock expectations
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(custom_err.ErrInsufficientFunds)
	mockRepo.On("OperationExistsTx", mock.Anything, mock.Anything, requestID).Return(false, nil)
	mockRepo.On("GetWalletBalanceForUpdateTx", mock.Anything, mock.Anything, walletID).Return(int64(1000), nil)

	// Act
	err := service.UpdateBalance(ctx, request)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, custom_err.ErrInsufficientFunds))
}

func TestUpdateBalance_WalletNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "test-request-4"

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        100,
		RequestID:     requestID,
	}

	// Mock expectations
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(custom_err.ErrNotFound)
	mockRepo.On("OperationExistsTx", mock.Anything, mock.Anything, requestID).Return(false, nil)
	mockRepo.On("GetWalletBalanceForUpdateTx", mock.Anything, mock.Anything, walletID).Return(int64(0), pgx.ErrNoRows)

	// Act
	err := service.UpdateBalance(ctx, request)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, custom_err.ErrNotFound))
}

func TestUpdateBalance_DuplicateRequest(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()
	requestID := "duplicate-request"

	request := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        100,
		RequestID:     requestID,
	}

	// Mock expectations - операция уже существует
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("OperationExistsTx", mock.Anything, mock.Anything, requestID).Return(true, nil)

	// Act
	err := service.UpdateBalance(ctx, request)

	// Assert
	assert.NoError(t, err) // Идемпотентность - не ошибка!
	mockRepo.AssertExpectations(t)
}

func TestGetWalletByID_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()

	expectedWallet := &models.Wallet{
		ID:      walletID,
		Balance: 1000,
	}

	mockRepo.On("GetByID", ctx, walletID).Return(expectedWallet, nil)

	// Act
	wallet, err := service.GetWalletByID(ctx, walletID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWallet, wallet)
	mockRepo.AssertExpectations(t)
}

func TestGetWalletByID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockWalletRepository)
	mockTxManager := new(MockTxManager)
	service := NewWalletService(mockRepo, mockTxManager)

	ctx := context.Background()
	walletID := uuid.New()

	mockRepo.On("GetByID", ctx, walletID).Return(nil, custom_err.ErrNotFound)

	// Act
	wallet, err := service.GetWalletByID(ctx, walletID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, wallet)
	assert.True(t, errors.Is(err, custom_err.ErrNotFound))
}
