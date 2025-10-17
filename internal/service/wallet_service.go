package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WalletServicer описывает, что должен уметь сервис кошелька.
type WalletServicer interface {
	UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
}

var _ WalletServicer = (*WalletService)(nil)

type TxManager interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

const (
	numShards       = 256
	numFlushWorkers = 2
	flushInterval   = 1 * time.Second
	maxBatchSize    = 500
)

type WalletService struct {
	repo       repository.Wallet // Используем абстрактный интерфейс
	txManager  TxManager
	shards     [numShards]*Shard
	retryQueue chan retryItem
	metrics    *Metrics
}

type Metrics struct {
	flushesTotal   atomic.Int64
	flushesFailed  atomic.Int64
	walletsInCache atomic.Int64
	retriesTotal   atomic.Int64
}

type retryItem struct {
	walletID uuid.UUID
	balance  int64
	attempts int
}

// NewWalletService теперь принимает интерфейс repository.Wallet
func NewWalletService(repo repository.Wallet, txManager TxManager) *WalletService {
	s := &WalletService{
		repo:       repo,
		txManager:  txManager,
		retryQueue: make(chan retryItem, 50000),
		metrics:    &Metrics{},
	}

	for i := 0; i < numShards; i++ {
		s.shards[i] = &Shard{
			wallets: make(map[uuid.UUID]*WalletState),
		}
	}

	// Запускаем фоновые воркеры
	for i := 0; i < numFlushWorkers; i++ {
		go s.flusher(i)
	}
	for i := 0; i < 2; i++ {
		go s.retryWorker(i)
	}
	go s.metricsLogger()

	return s
}

func (s *WalletService) getShard(id uuid.UUID) *Shard {
	hasher := fnv.New64a()
	hasher.Write(id[:])
	return s.shards[hasher.Sum64()&(numShards-1)]
}

func (s *WalletService) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "service.GetWalletByID"
	shard := s.getShard(id)

	state, err := shard.loadStateIntoCacheIfExists(ctx, id, s.repo)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, err // Пробрасываем ErrNotFound как есть
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.Wallet{ID: id, Balance: state.balance.Load()}, nil
}

func (s *WalletService) UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error {
	const op = "service.UpdateBalance"
	shard := s.getShard(req.WalletID)

	state, err := shard.loadStateIntoCacheIfExists(ctx, req.WalletID, s.repo)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return err // Пробрасываем ErrNotFound как есть
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if req.OperationType == models.WithdrawOperation {
		return state.Withdraw(req.Amount)
	}

	state.Add(req.Amount)
	return nil
}
