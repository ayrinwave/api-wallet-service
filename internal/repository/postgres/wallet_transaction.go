package postgres

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ✅ Теперь возвращает только balance (без version)
func (r *PgWalletRepository) GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error) {
	var balance int64
	err := tx.QueryRow(ctx,
		repository.GetWalletStateQuery,
		walletID,
	).Scan(&balance)
	return balance, err
}

// ✅ НОВЫЙ метод - упрощённый UPDATE без version
func (r *PgWalletRepository) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error {
	res, err := tx.Exec(ctx,
		repository.UpdateWalletBalanceQuery,
		newBalance,
		walletID,
	)
	if err != nil {
		// Проверка на constraint violation
		if pgerr, ok := err.(*pgconn.PgError); ok {
			if pgerr.Code == "23514" { // check_violation (balance < 0)
				return custom_err.ErrInsufficientFunds
			}
		}
		return err
	}

	// Проверяем что строка была обновлена
	if res.RowsAffected() == 0 {
		return custom_err.ErrNotFound
	}

	return nil
}

func (r *PgWalletRepository) OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, repository.CheckOperationExistsQuery, requestID).Scan(&exists)
	return exists, err
}

func (r *PgWalletRepository) CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error {
	_, err := tx.Exec(ctx,
		repository.CreateOperationQuery,
		walletID, amount, requestID,
	)
	if pgerr, ok := err.(*pgconn.PgError); ok && pgerr.Code == "23505" {
		return custom_err.ErrDuplicateRequest
	}
	return err
}

// CreateWalletTx создает новый кошелек внутри транзакции
// Используется при регистрации пользователя - создаем 3 кошелька атомарно
func (r *PgWalletRepository) CreateWalletTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency models.Currency) (*models.Wallet, error) {
	const op = "repository.CreateWalletTx"

	walletID := uuid.New()
	var wallet models.Wallet

	err := tx.QueryRow(
		ctx,
		repository.CreateWalletQuery,
		walletID,
		userID,
		currency,
		0, // начальный баланс = 0
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Version,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &wallet, nil
}
