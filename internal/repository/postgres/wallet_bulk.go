package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *WalletRepository) BulkUpdateBalances(ctx context.Context, wallets map[uuid.UUID]int64) error {
	if len(wallets) == 0 {
		return nil
	}

	stats := r.db.Stat()
	log.Printf("[BulkUpdate] Pool stats: Acquired=%d Idle=%d Total=%d Max=%d",
		stats.AcquiredConns(), stats.IdleConns(), stats.TotalConns(), stats.MaxConns())

	startTime := time.Now()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
        CREATE TEMP TABLE wallet_updates_tmp (
            id UUID PRIMARY KEY,
            balance BIGINT NOT NULL
        ) ON COMMIT DROP
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания TEMP таблицы: %w", err)
	}

	rows := make([][]any, 0, len(wallets))
	for id, balance := range wallets {
		rows = append(rows, []any{id, balance})
	}

	_, err = tx.CopyFrom(ctx,
		pgx.Identifier{"wallet_updates_tmp"},
		[]string{"id", "balance"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("ошибка COPY в TEMP таблицу: %w", err)
	}

	cmdTag, err := tx.Exec(ctx, `
        UPDATE wallets w
        SET balance = u.balance,
            updated_at = NOW()
        FROM wallet_updates_tmp u
        WHERE w.id = u.id
    `)
	if err != nil {
		return fmt.Errorf("ошибка UPDATE из TEMP таблицы: %w", err)
	}
	updatedCount := cmdTag.RowsAffected()

	cmdTag, err = tx.Exec(ctx, `
        INSERT INTO wallets (id, balance, created_at, updated_at)
        SELECT u.id, u.balance, NOW(), NOW()
        FROM wallet_updates_tmp u
        WHERE NOT EXISTS (
            SELECT 1 FROM wallets w WHERE w.id = u.id
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка INSERT из TEMP таблицы: %w", err)
	}
	insertedCount := cmdTag.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("[BulkUpdate] Success: %d wallets (updated=%d, inserted=%d) in %v",
		len(wallets), updatedCount, insertedCount, duration)

	return nil
}
