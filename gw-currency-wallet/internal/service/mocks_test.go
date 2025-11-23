package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"

	"gw-currency-wallet/internal/grpc_client"
	"gw-currency-wallet/internal/models"
)

// ========== MockUserRepository ==========
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) CreateTx(ctx context.Context, tx pgx.Tx, user *models.User) (*models.User, error) {
	args := m.Called(ctx, tx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// ========== MockWalletRepo ==========
type MockWalletRepo struct {
	mock.Mock
}

func (m *MockWalletRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) CreateWalletTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency models.Currency) (*models.Wallet, error) {
	args := m.Called(ctx, tx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency) (*models.Wallet, error) {
	args := m.Called(ctx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) GetUserWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Wallet), args.Error(1)
}

func (m *MockWalletRepo) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error {
	args := m.Called(ctx, tx, walletID, newBalance)
	return args.Error(0)
}

func (m *MockWalletRepo) GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tx, walletID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletRepo) CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error {
	args := m.Called(ctx, tx, walletID, amount, requestID)
	return args.Error(0)
}

func (m *MockWalletRepo) OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error) {
	args := m.Called(ctx, tx, requestID)
	return args.Bool(0), args.Error(1)
}

// ✅ ИСПРАВЛЕНО: возвращает []*models.Wallet
func (m *MockWalletRepo) GetAllUserWallets(ctx context.Context, userID uuid.UUID) ([]*models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Wallet), args.Error(1)
}

// ========== MockTxManager ==========
type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	args := m.Called(ctx, fn)
	if args.Error(0) != nil {
		return args.Error(0)
	}
	return fn(nil)
}

// ========== MockExchangerClient ==========
type MockExchangerClient struct {
	mock.Mock
}

func (m *MockExchangerClient) GetExchangeRates(ctx context.Context) (*grpc_client.ExchangeRatesResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpc_client.ExchangeRatesResponse), args.Error(1)
}

func (m *MockExchangerClient) GetExchangeRateForCurrency(ctx context.Context, from, to string) (*grpc_client.ExchangeRateResponse, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpc_client.ExchangeRateResponse), args.Error(1)
}

func (m *MockExchangerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// ========== MockKafkaProducer ==========
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}
