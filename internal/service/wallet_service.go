package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository/postgres"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Wallet interface {
	UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
}
type WalletService struct {
	repo      postgres.WalletRepository
	txManager TxManager
}

func NewWalletService(repo postgres.WalletRepository, txManager TxManager) *WalletService {
	return &WalletService{
		repo:      repo,
		txManager: txManager,
	}
}
func (s *WalletService) UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error {
	// Таймаут для всей операции
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	amount := req.Amount
	if req.OperationType == models.OperationWithdraw {
		amount = -amount
	}

	// Один вызов транзакции, без retry
	return s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// 1. Проверка идемпотентности
		exists, err := s.repo.OperationExistsTx(ctx, tx, req.RequestID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		// 2. Получить баланс с блокировкой (FOR UPDATE)
		// Блокировка гарантирует, что только одна транзакция
		// может читать/изменять этот кошелёк одновременно
		balance, err := s.repo.GetWalletBalanceForUpdateTx(ctx, tx, req.WalletID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return custom_err.ErrNotFound
			}
			return err
		}

		// 3. Вычислить новый баланс
		newBalance := balance + amount
		if newBalance < 0 {
			return custom_err.ErrInsufficientFunds
		}

		// 4. Обновить баланс
		err = s.repo.UpdateBalanceTx(ctx, tx, req.WalletID, newBalance)
		if err != nil {
			return err
		}

		// 5. Записать операцию для аудита
		err = s.repo.CreateOperationTx(ctx, tx, req.WalletID, amount, req.RequestID)
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *WalletService) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	return s.repo.GetByID(ctx, id)
}
