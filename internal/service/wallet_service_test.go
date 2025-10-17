package service

import (
	"context"
	"errors"
	"testing"

	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ repository.Wallet = (*mockRepository)(nil)

type mockRepository struct {
	GetByIDFunc             func(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	BulkUpdateBalancesFunc  func(ctx context.Context, wallets map[uuid.UUID]int64) error
	UpsertWalletBalanceFunc func(ctx context.Context, id uuid.UUID, balance int64) error
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("GetByIDFunc not implemented")
}

func (m *mockRepository) BulkUpdateBalances(ctx context.Context, wallets map[uuid.UUID]int64) error {
	if m.BulkUpdateBalancesFunc != nil {
		return m.BulkUpdateBalancesFunc(ctx, wallets)
	}
	return nil
}

func (m *mockRepository) UpsertWalletBalance(ctx context.Context, id uuid.UUID, balance int64) error {
	if m.UpsertWalletBalanceFunc != nil {
		return m.UpsertWalletBalanceFunc(ctx, id, balance)
	}
	return nil
}

func TestWalletService_GetWalletByID(t *testing.T) {
	walletID := uuid.New()

	t.Run("Success - Found in DB (Cache Miss)", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return &models.Wallet{ID: id, Balance: 500}, nil
			},
		}
		service := NewWalletService(mockRepo, nil) // txManager не нужен для этого теста

		wallet, err := service.GetWalletByID(context.Background(), walletID)

		require.NoError(t, err)
		assert.Equal(t, int64(500), wallet.Balance)

		shard := service.getShard(walletID)
		shard.mu.RLock()
		_, ok := shard.wallets[walletID]
		shard.mu.RUnlock()
		assert.True(t, ok, "wallet should be in cache after a cache miss")
	})

	t.Run("Success - Found in Cache", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				t.Fatal("repository GetByID should not be called when wallet is in cache")
				return nil, nil
			},
		}
		service := NewWalletService(mockRepo, nil)

		shard := service.getShard(walletID)
		state := &WalletState{}
		state.balance.Store(123)
		shard.wallets[walletID] = state

		wallet, err := service.GetWalletByID(context.Background(), walletID)

		require.NoError(t, err)
		assert.Equal(t, int64(123), wallet.Balance)
	})

	t.Run("Error - Not Found Anywhere", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return nil, custom_err.ErrNotFound
			},
		}
		service := NewWalletService(mockRepo, nil)

		_, err := service.GetWalletByID(context.Background(), walletID)

		require.Error(t, err)
		assert.True(t, errors.Is(err, custom_err.ErrNotFound))
	})
}

func TestWalletService_UpdateBalance(t *testing.T) {
	walletID := uuid.New()

	t.Run("Success - Deposit", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return &models.Wallet{ID: id, Balance: 100}, nil
			},
		}
		service := NewWalletService(mockRepo, nil)

		req := models.WalletOperationRequest{WalletID: walletID, OperationType: models.DepositOperation, Amount: 50}
		err := service.UpdateBalance(context.Background(), req)

		require.NoError(t, err)

		shard := service.getShard(walletID)
		state := shard.wallets[walletID]
		assert.Equal(t, int64(150), state.balance.Load())
		assert.True(t, state.dirty.Load())
	})

	t.Run("Error - Insufficient Funds", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return &models.Wallet{ID: id, Balance: 100}, nil
			},
		}
		service := NewWalletService(mockRepo, nil)

		req := models.WalletOperationRequest{WalletID: walletID, OperationType: models.WithdrawOperation, Amount: 200}
		err := service.UpdateBalance(context.Background(), req)

		require.Error(t, err)
		assert.True(t, errors.Is(err, custom_err.ErrInsufficientFunds))

		shard := service.getShard(walletID)
		state := shard.wallets[walletID]
		assert.Equal(t, int64(100), state.balance.Load())
		assert.False(t, state.dirty.Load())
	})

	t.Run("Error - Wallet Not Found on Update", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return nil, custom_err.ErrNotFound
			},
		}
		service := NewWalletService(mockRepo, nil)

		req := models.WalletOperationRequest{WalletID: walletID, OperationType: models.DepositOperation, Amount: 100}
		err := service.UpdateBalance(context.Background(), req)

		require.Error(t, err)
		assert.True(t, errors.Is(err, custom_err.ErrNotFound))
	})
}

func TestWalletService_getShard(t *testing.T) {
	service := NewWalletService(&mockRepository{}, nil)

	t.Run("Deterministic", func(t *testing.T) {
		walletID := uuid.New()
		shard1 := service.getShard(walletID)
		shard2 := service.getShard(walletID)
		assert.Same(t, shard1, shard2, "getShard should be deterministic for the same ID")
	})

	t.Run("Distribution", func(t *testing.T) {
		walletID1 := uuid.New()
		walletID2 := uuid.New()
		shard1 := service.getShard(walletID1)
		shard2 := service.getShard(walletID2)

		if walletID1 != walletID2 {
			assert.NotNil(t, shard1)
			assert.NotNil(t, shard2)
		}
	})
}
