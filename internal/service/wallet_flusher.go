package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

func (s *WalletService) flusher(workerID int) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	shardsPerWorker := numShards / numFlushWorkers
	startShard := workerID * shardsPerWorker
	endShard := startShard + shardsPerWorker

	for range ticker.C {
		totalFlushed := 0

		for i := startShard; i < endShard; i++ {
			shard := s.shards[i]

			shard.mu.RLock()
			dirtyCount := 0
			for _, state := range shard.wallets {
				if state.dirty.Load() {
					dirtyCount++
				}
			}

			if dirtyCount == 0 {
				shard.mu.RUnlock()
				continue
			}

			batchSize := min(dirtyCount, maxBatchSize)
			dirtyWallets := make(map[uuid.UUID]int64, batchSize)
			dirtyStates := make(map[uuid.UUID]*WalletState, batchSize)

			collected := 0
			for id, state := range shard.wallets {
				if collected >= batchSize {
					break
				}
				if state.dirty.Load() {
					dirtyWallets[id] = state.balance.Load()
					dirtyStates[id] = state
					collected++
				}
			}
			shard.mu.RUnlock()

			if len(dirtyWallets) == 0 {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := s.repo.BulkUpdateBalances(ctx, dirtyWallets)
			cancel()

			s.metrics.flushesTotal.Add(1)

			if err != nil {
				s.metrics.flushesFailed.Add(1)
				log.Printf("[Worker %d] Flush failed: %v, queueing %d wallets for retry",
					workerID, err, len(dirtyWallets))

				for id, balance := range dirtyWallets {
					select {
					case s.retryQueue <- retryItem{walletID: id, balance: balance}:
						s.metrics.retriesTotal.Add(1)
					default:
						log.Printf("[Worker %d] Retry queue full!", workerID)
					}
				}
			} else {
				for id, state := range dirtyStates {
					if state.balance.Load() == dirtyWallets[id] {
						state.dirty.Store(false)
					}
				}
				totalFlushed += len(dirtyWallets)
			}
		}

		if totalFlushed > 0 {
			log.Printf("[Worker %d] Flushed %d wallets", workerID, totalFlushed)
		}
	}
}

func (s *WalletService) retryWorker(workerID int) {
	for item := range s.retryQueue {
		if item.attempts >= 3 {
			log.Printf("[Retry %d] Max attempts for wallet %s", workerID, item.walletID)
			continue
		}

		backoff := time.Duration(1<<item.attempts) * time.Second
		time.Sleep(backoff)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := s.repo.UpsertWalletBalance(ctx, item.walletID, item.balance)
		cancel()

		if err != nil {
			item.attempts++
			select {
			case s.retryQueue <- item:
			default:
				log.Printf("[Retry %d] Queue full, dropping wallet %s", workerID, item.walletID)
			}
		} else {
			shard := s.getShard(item.walletID)
			shard.mu.Lock()
			if state, ok := shard.wallets[item.walletID]; ok {
				if state.balance.Load() == item.balance {
					state.dirty.Store(false)
				}
			}
			shard.mu.Unlock()
		}
	}
}

func (s *WalletService) metricsLogger() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var totalWallets int64
		var dirtyWallets int64

		for i := 0; i < numShards; i++ {
			s.shards[i].mu.RLock()
			totalWallets += int64(len(s.shards[i].wallets))
			for _, state := range s.shards[i].wallets {
				if state.dirty.Load() {
					dirtyWallets++
				}
			}
			s.shards[i].mu.RUnlock()
		}

		log.Printf("[METRICS] Wallets=%d Dirty=%d Flushes=%d Failed=%d Retries=%d QueueLen=%d",
			totalWallets,
			dirtyWallets,
			s.metrics.flushesTotal.Load(),
			s.metrics.flushesFailed.Load(),
			s.metrics.retriesTotal.Load(),
			len(s.retryQueue),
		)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
