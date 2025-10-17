package postgres

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *WalletRepository) GetWalletStateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (balance int64, version int64, err error) {
	err = tx.QueryRow(ctx, repository.GetWalletStateQuery, walletID).Scan(&balance, &version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, custom_err.ErrNotFound
		}
		return 0, 0, fmt.Errorf("ошибка чтения состояния кошелька: %w", err)
	}
	return balance, version, nil
}
func (r *WalletRepository) UpdateBalanceWithOptimisticLockTx(
	ctx context.Context,
	tx pgx.Tx,
	walletID uuid.UUID,
	newBalance int64,
	expectedVersion int64,
) error {
	cmdTag, err := tx.Exec(ctx, repository.UpdateWalletBalanceWithLockQuery, newBalance, expectedVersion, walletID)
	if err != nil {
		return fmt.Errorf("ошибка выполнения обновления с блокировкой: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return custom_err.ErrConflict
	}

	return nil
}
func (r *WalletRepository) CheckOperationExistsTx(ctx context.Context, tx pgx.Tx, requestID uuid.UUID) (bool, error) {
	var exists bool

	if err := tx.QueryRow(ctx, repository.CheckOperationExistsQuery, requestID).Scan(&exists); err != nil {
		return false, fmt.Errorf("ошибка проверки идемпотентности: %w", err)
	}
	return exists, nil
}

func (r *WalletRepository) CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID uuid.UUID) error {
	_, err := tx.Exec(ctx, repository.CreateOperationQuery, walletID, amount, requestID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return custom_err.ErrDuplicateRequest
		}
		return fmt.Errorf("ошибка сохранения операции в историю: %w", err)
	}
	return nil
}
