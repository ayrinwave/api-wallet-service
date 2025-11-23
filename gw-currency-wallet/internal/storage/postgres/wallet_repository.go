package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WalletRepository интерфейс для работы с кошельками
type WalletRepository interface {
	// ========== Методы С транзакциями (Tx) ==========
	// Используются в wallet_service для атомарных операций
	GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error)
	UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error
	OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error)
	CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error
	CreateWalletTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, currency models.Currency) (*models.Wallet, error)

	// ========== Методы БЕЗ транзакций ==========
	// Используются для простых read операций
	GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency) (*models.Wallet, error)
	GetAllUserWallets(ctx context.Context, userID uuid.UUID) ([]*models.Wallet, error)
}
type PgWalletRepository struct {
	db *pgxpool.Pool
}

// NewWalletRepository создает новый экземпляр репозитория
func NewWalletRepository(db *pgxpool.Pool) WalletRepository {
	return &PgWalletRepository{db: db}
}

// GetByID получает кошелек по его ID
// Используется редко, в основном для админских операций или дебага
func (r *PgWalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "storage.GetByID"
	var wallet models.Wallet
	err := r.db.QueryRow(ctx, storage.GetWalletByIDQuery, id).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Version,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}

// GetByUserAndCurrency получает конкретный кошелек пользователя по валюте
// Используется для операций deposit/withdraw - нужно знать userID и currency
// Пример: пользователь хочет пополнить USD кошелек
func (r *PgWalletRepository) GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency) (*models.Wallet, error) {
	const op = "storage.GetByUserAndCurrency"
	var wallet models.Wallet
	err := r.db.QueryRow(ctx, storage.GetWalletByUserAndCurrencyQuery, userID, currency).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Version,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}

// GetAllUserWallets получает ВСЕ кошельки пользователя (USD, RUB, EUR)
// Используется для GET /api/v1/balance - показываем балансы всех валют
func (r *PgWalletRepository) GetAllUserWallets(ctx context.Context, userID uuid.UUID) ([]*models.Wallet, error) {
	const op = "storage.GetAllUserWallets"

	rows, err := r.db.Query(ctx, storage.GetAllUserWalletsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		var wallet models.Wallet
		err := rows.Scan(
			&wallet.ID,
			&wallet.UserID,
			&wallet.Currency,
			&wallet.Balance,
			&wallet.Version,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		wallets = append(wallets, &wallet)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return wallets, nil
}
