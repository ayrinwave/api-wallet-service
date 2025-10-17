package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/repository"
	"api_wallet/internal/repository/postgres"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

type WalletState struct {
	balance atomic.Int64
	dirty   atomic.Bool
}
type Shard struct {
	mu      sync.RWMutex
	wallets map[uuid.UUID]*WalletState
}

func (w *WalletState) Add(amount int64) {
	w.balance.Add(amount)
	w.dirty.Store(true)
}

func (w *WalletState) Withdraw(amount int64) error {
	for {
		current := w.balance.Load()
		if current < amount {
			return custom_err.ErrInsufficientFunds
		}
		if w.balance.CompareAndSwap(current, current-amount) {
			w.dirty.Store(true)
			return nil
		}
	}
}

func (s *Shard) getState(ctx context.Context, id uuid.UUID, repo *postgres.WalletRepository) (*WalletState, error) {

	s.mu.RLock()
	state, ok := s.wallets[id]
	s.mu.RUnlock()

	if ok {
		return state, nil
	}

	wallet, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("ошибка загрузки кошелька из БД: %w", err)
	}
	newState := &WalletState{}
	newState.balance.Store(wallet.Balance)

	s.mu.Lock()
	if existing, exists := s.wallets[id]; exists {
		s.mu.Unlock()
		return existing, nil
	}
	s.wallets[id] = newState
	s.mu.Unlock()

	return newState, nil
}

func (s *Shard) loadStateIntoCacheIfExists(ctx context.Context, id uuid.UUID, repo repository.Wallet) (*WalletState, error) {
	s.mu.RLock()
	state, ok := s.wallets[id]
	s.mu.RUnlock()

	if ok {
		return state, nil
	}

	wallet, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("ошибка загрузки кошелька из БД: %w", err)
	}

	newState := &WalletState{}
	newState.balance.Store(wallet.Balance)

	s.mu.Lock()
	if existing, exists := s.wallets[id]; exists {
		s.mu.Unlock()
		return existing, nil
	}
	s.wallets[id] = newState
	s.mu.Unlock()

	return newState, nil
}
