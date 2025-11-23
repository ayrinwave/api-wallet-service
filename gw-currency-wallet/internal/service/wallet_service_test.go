package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
)

// Вспомогательные функции
func setupWalletService(t *testing.T) (*WalletService, *MockWalletRepo, *MockTxManager) {
	repo := new(MockWalletRepo)
	txManager := new(MockTxManager)

	service := &WalletService{
		repo:      repo,
		txManager: txManager,
	}

	return service, repo, txManager
}

// ========== Тесты для GetUserBalance ==========

func TestWalletService_GetUserBalance_Success(t *testing.T) {
	service, repo, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	wallets := []*models.Wallet{
		{
			ID:       uuid.New(),
			UserID:   userID,
			Currency: string(models.CurrencyUSD),
			Balance:  100050, // 1000.50 USD
		},
		{
			ID:       uuid.New(),
			UserID:   userID,
			Currency: string(models.CurrencyRUB),
			Balance:  5000000, // 50000.00 RUB
		},
		{
			ID:       uuid.New(),
			UserID:   userID,
			Currency: string(models.CurrencyEUR),
			Balance:  85075, // 850.75 EUR
		},
	}

	repo.On("GetAllUserWallets", ctx, userID).Return(wallets, nil)

	// Выполняем тест
	resp, err := service.GetUserBalance(ctx, userID)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1000.50, resp.USD)
	assert.Equal(t, 50000.00, resp.RUB)
	assert.Equal(t, 850.75, resp.EUR)

	repo.AssertExpectations(t)
}

func TestWalletService_GetUserBalance_EmptyWallets(t *testing.T) {
	service, repo, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	// ✅ ИСПРАВЛЕНО: []*models.Wallet вместо []models.Wallet
	repo.On("GetAllUserWallets", ctx, userID).Return([]*models.Wallet{}, nil)

	resp, err := service.GetUserBalance(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 0.0, resp.USD)
	assert.Equal(t, 0.0, resp.RUB)
	assert.Equal(t, 0.0, resp.EUR)

	repo.AssertExpectations(t)
}

// ========== Тесты для Deposit ==========

func TestWalletService_Deposit_Success(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()
	walletID := uuid.New()

	req := models.DepositRequest{
		Amount:    500.00,
		Currency:  models.CurrencyUSD,
		RequestID: "deposit-001",
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Currency: string(models.CurrencyUSD),
		Balance:  100000,
	}

	repo.On("GetByUserAndCurrency", ctx, userID, req.Currency).Return(wallet, nil)

	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	repo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, walletID).Return(int64(100000), nil)
	repo.On("UpdateBalanceTx", ctx, mock.Anything, walletID, int64(150000)).Return(nil)
	repo.On("CreateOperationTx", ctx, mock.Anything, walletID, int64(50000), req.RequestID).Return(nil)

	// ✅ ИСПРАВЛЕНО: []*models.Wallet вместо []models.Wallet
	repo.On("GetAllUserWallets", ctx, userID).Return([]*models.Wallet{
		{ID: walletID, UserID: userID, Currency: string(models.CurrencyUSD), Balance: 150000},
	}, nil)

	resp, err := service.Deposit(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Account topped up successfully", resp.Message)
	assert.Equal(t, 1500.00, resp.NewBalance.USD)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestWalletService_Deposit_InvalidCurrency(t *testing.T) {
	service, _, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	req := models.DepositRequest{
		Amount:    500.00,
		Currency:  "INVALID",
		RequestID: "deposit-001",
	}

	// Выполняем тест
	resp, err := service.Deposit(ctx, userID, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrInvalidCurrency, err)
}

func TestWalletService_Deposit_InvalidAmount(t *testing.T) {
	service, _, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name   string
		amount float64
	}{
		{"zero amount", 0.0},
		{"negative amount", -100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := models.DepositRequest{
				Amount:    tt.amount,
				Currency:  models.CurrencyUSD,
				RequestID: "deposit-001",
			}

			resp, err := service.Deposit(ctx, userID, req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, custom_err.ErrInvalidAmount, err)
		})
	}
}

func TestWalletService_Deposit_EmptyRequestID(t *testing.T) {
	service, _, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	req := models.DepositRequest{
		Amount:    500.00,
		Currency:  models.CurrencyUSD,
		RequestID: "",
	}

	// Выполняем тест
	resp, err := service.Deposit(ctx, userID, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrInvalidInput, err)
}

func TestWalletService_Deposit_DuplicateRequest(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()
	walletID := uuid.New()

	req := models.DepositRequest{
		Amount:    500.00,
		Currency:  models.CurrencyUSD,
		RequestID: "deposit-001",
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Currency: string(models.CurrencyUSD),
		Balance:  100000,
	}

	repo.On("GetByUserAndCurrency", ctx, userID, req.Currency).Return(wallet, nil)

	// ✅ ИСПРАВЛЕНО: Добавляем .Run() чтобы выполнить функцию внутри транзакции
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(pgx.Tx) error)
			fn(nil) // Выполняем функцию транзакции
		}).
		Return(custom_err.ErrDuplicateRequest)

	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(true, nil)

	resp, err := service.Deposit(ctx, userID, req)

	// ✅ Проверяем ошибку
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, custom_err.ErrDuplicateRequest)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

// ========== Тесты для Withdraw ==========

func TestWalletService_Withdraw_Success(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()
	walletID := uuid.New()

	req := models.WithdrawRequest{
		Amount:    300.00,
		Currency:  models.CurrencyUSD,
		RequestID: "withdraw-001",
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Currency: string(models.CurrencyUSD),
		Balance:  100000,
	}

	repo.On("GetByUserAndCurrency", ctx, userID, req.Currency).Return(wallet, nil)

	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	repo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, walletID).Return(int64(100000), nil)
	repo.On("UpdateBalanceTx", ctx, mock.Anything, walletID, int64(70000)).Return(nil)
	repo.On("CreateOperationTx", ctx, mock.Anything, walletID, int64(30000), req.RequestID).Return(nil)

	// ✅ ИСПРАВЛЕНО: []*models.Wallet вместо []models.Wallet
	repo.On("GetAllUserWallets", ctx, userID).Return([]*models.Wallet{
		{ID: walletID, UserID: userID, Currency: string(models.CurrencyUSD), Balance: 70000},
	}, nil)

	resp, err := service.Withdraw(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Withdrawal successful", resp.Message)
	assert.Equal(t, 700.00, resp.NewBalance.USD)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestWalletService_Withdraw_InsufficientFunds(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()
	walletID := uuid.New()

	req := models.WithdrawRequest{
		Amount:    1500.00, // Больше чем на балансе
		Currency:  models.CurrencyUSD,
		RequestID: "withdraw-001",
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Currency: string(models.CurrencyUSD),
		Balance:  100000, // 1000.00 USD
	}

	repo.On("GetByUserAndCurrency", ctx, userID, req.Currency).Return(wallet, nil)

	// ✅ ИСПРАВЛЕНО: Добавляем .Run()
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(pgx.Tx) error)
			fn(nil) // Выполняем функцию транзакции
		}).
		Return(custom_err.ErrInsufficientFunds)

	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	repo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, walletID).Return(int64(100000), nil)
	// Код вернёт ошибку после проверки баланса - дальнейшие моки не нужны
	// Выполняем тест
	resp, err := service.Withdraw(ctx, userID, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, custom_err.ErrInsufficientFunds)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestWalletService_Withdraw_WalletNotFound(t *testing.T) {
	service, repo, _ := setupWalletService(t)
	ctx := context.Background()
	userID := uuid.New()

	req := models.WithdrawRequest{
		Amount:    100.00,
		Currency:  models.CurrencyUSD,
		RequestID: "withdraw-001",
	}

	// Мокаем GetByUserAndCurrency - кошелек не найден
	repo.On("GetByUserAndCurrency", ctx, userID, req.Currency).Return(nil, custom_err.ErrNotFound)

	// Выполняем тест
	resp, err := service.Withdraw(ctx, userID, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrNotFound, err)

	repo.AssertExpectations(t)
}

// ========== Тесты для UpdateBalance (старый метод) ==========

func TestWalletService_UpdateBalance_Deposit(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	walletID := uuid.New()

	req := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationDeposit,
		Amount:        50000, // 500.00
		RequestID:     "op-001",
	}

	// Мокаем транзакцию
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	repo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, walletID).Return(int64(100000), nil)
	repo.On("UpdateBalanceTx", ctx, mock.Anything, walletID, int64(150000)).Return(nil)
	repo.On("CreateOperationTx", ctx, mock.Anything, walletID, int64(50000), req.RequestID).Return(nil)

	// Выполняем тест
	err := service.UpdateBalance(ctx, req)

	// Проверки
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestWalletService_UpdateBalance_Withdraw(t *testing.T) {
	service, repo, txManager := setupWalletService(t)
	ctx := context.Background()
	walletID := uuid.New()

	req := models.WalletOperationRequest{
		WalletID:      walletID,
		OperationType: models.OperationWithdraw,
		Amount:        30000, // 300.00
		RequestID:     "op-002",
	}

	// Мокаем транзакцию
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	repo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	repo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, walletID).Return(int64(100000), nil)
	repo.On("UpdateBalanceTx", ctx, mock.Anything, walletID, int64(70000)).Return(nil)
	repo.On("CreateOperationTx", ctx, mock.Anything, walletID, int64(30000), req.RequestID).Return(nil)

	// Выполняем тест
	err := service.UpdateBalance(ctx, req)

	// Проверки
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}
