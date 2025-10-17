package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletState_Add(t *testing.T) {
	state := &WalletState{}
	state.balance.Store(100)
	state.dirty.Store(false)

	state.Add(50)

	assert.Equal(t, int64(150), state.balance.Load(), "balance should be increased")
	assert.True(t, state.dirty.Load(), "wallet should be marked as dirty after Add")
}

func TestWalletState_Withdraw(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		state := &WalletState{}
		state.balance.Store(100)
		state.dirty.Store(false)

		err := state.Withdraw(30)

		assert.NoError(t, err)
		assert.Equal(t, int64(70), state.balance.Load(), "balance should be decreased")
		assert.True(t, state.dirty.Load(), "wallet should be marked as dirty after Withdraw")
	})

	t.Run("Error - Insufficient Funds", func(t *testing.T) {
		state := &WalletState{}
		state.balance.Store(100)
		state.dirty.Store(false)

		err := state.Withdraw(150)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, custom_err.ErrInsufficientFunds))
		assert.Equal(t, int64(100), state.balance.Load(), "balance should not change on failed withdraw")
		assert.False(t, state.dirty.Load(), "wallet should not be marked as dirty on failed withdraw")
	})

	t.Run("Concurrency - Multiple Withdraws", func(t *testing.T) {
		state := &WalletState{}
		state.balance.Store(200)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			err := state.Withdraw(70)
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()
			err := state.Withdraw(50)
			assert.NoError(t, err)
		}()

		wg.Wait()

		assert.Equal(t, int64(80), state.balance.Load())
		assert.True(t, state.dirty.Load())
	})
}

func TestShard_loadStateIntoCacheIfExists(t *testing.T) {
	walletID := uuid.New()
	initialBalance := int64(777)
	ctx := context.Background()

	t.Run("Cache Miss - Load from DB Success", func(t *testing.T) {
		callCount := 0
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				callCount++
				return &models.Wallet{ID: id, Balance: initialBalance}, nil
			},
		}

		shard := &Shard{wallets: make(map[uuid.UUID]*WalletState)}

		state, err := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)

		require.NoError(t, err)
		assert.Equal(t, initialBalance, state.balance.Load())
		assert.Equal(t, 1, callCount, "GetByID should be called once")

		shard.mu.RLock()
		_, ok := shard.wallets[walletID]
		shard.mu.RUnlock()
		assert.True(t, ok, "Wallet should now be in cache")

		stateHit, errHit := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)
		require.NoError(t, errHit)
		assert.Equal(t, initialBalance, stateHit.balance.Load())
		assert.Equal(t, 1, callCount, "GetByID should NOT be called again")
		assert.False(t, stateHit.dirty.Load(), "State loaded from DB must not be dirty")
	})

	t.Run("Cache Miss - Not Found in DB", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return nil, custom_err.ErrNotFound
			},
		}
		shard := &Shard{wallets: make(map[uuid.UUID]*WalletState)}

		state, err := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)

		require.Error(t, err)
		assert.True(t, errors.Is(err, custom_err.ErrNotFound), "Should return ErrNotFound")
		assert.Nil(t, state)

		shard.mu.RLock()
		_, ok := shard.wallets[walletID]
		shard.mu.RUnlock()
		assert.False(t, ok, "Wallet should not be cached if not found in DB")
	})

	t.Run("Cache Miss - Internal DB Error", func(t *testing.T) {
		dbError := errors.New("connection failed")
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return nil, dbError
			},
		}
		shard := &Shard{wallets: make(map[uuid.UUID]*WalletState)}

		_, err := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)

		require.Error(t, err)
		assert.True(t, errors.Is(err, dbError), "Should return the original DB error")
	})

	t.Run("Concurrency - Double Check Locking Prevention", func(t *testing.T) {
		mockRepo := &mockRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				time.Sleep(10 * time.Millisecond)
				return &models.Wallet{ID: id, Balance: 100}, nil
			},
		}
		shard := &Shard{wallets: make(map[uuid.UUID]*WalletState)}

		var wg sync.WaitGroup
		var firstState *WalletState

		wg.Add(2)

		go func() {
			defer wg.Done()
			state, _ := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)
			firstState = state
		}()

		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			state, _ := shard.loadStateIntoCacheIfExists(ctx, walletID, mockRepo)
			assert.Same(t, firstState, state, "Concurrent access should return the same object due to double-check locking")
		}()

		wg.Wait()

		shard.mu.RLock()
		assert.Len(t, shard.wallets, 1)
		shard.mu.RUnlock()
	})
}

func TestWalletState_RaceCondition(t *testing.T) {
	t.Run("Concurrent Add and Withdraw", func(t *testing.T) {
		const initialBalance = 10000
		const numOperations = 1000
		const amount = 10

		state := &WalletState{}
		state.balance.Store(initialBalance)

		var wg sync.WaitGroup
		wg.Add(numOperations * 2)

		for i := 0; i < numOperations; i++ {
			go func() {
				defer wg.Done()
				state.Add(amount)
			}()
		}

		for i := 0; i < numOperations; i++ {
			go func() {
				defer wg.Done()
				_ = state.Withdraw(amount)
			}()
		}
		wg.Wait()
		assert.Equal(t, int64(initialBalance), state.balance.Load(), "Final balance should be equal to initial balance after concurrent operations")
	})
}
