package service

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Wallet интерфейс для работы с кошельками
type Wallet interface {
	// Старые методы (для совместимости)
	UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)

	// Новые методы для мультивалютности
	GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalanceResponse, error)
	Deposit(ctx context.Context, userID uuid.UUID, req models.DepositRequest) (*models.BalanceOperationResponse, error)
	Withdraw(ctx context.Context, userID uuid.UUID, req models.WithdrawRequest) (*models.BalanceOperationResponse, error)
}

type WalletService struct {
	repo      postgres.WalletRepository
	txManager TxManager
}

func NewWalletService(repo postgres.WalletRepository, txManager TxManager) Wallet {
	return &WalletService{
		repo:      repo,
		txManager: txManager,
	}
}

// ========== СТАРЫЕ МЕТОДЫ (для совместимости) ==========

// UpdateBalance обновляет баланс кошелька (старый метод)
func (s *WalletService) UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error {
	const op = "service.UpdateBalance"

	return s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// Проверяем идемпотентность
		exists, err := s.repo.OperationExistsTx(ctx, tx, req.RequestID)
		if err != nil {
			return fmt.Errorf("%s: failed to check operation: %w", op, err)
		}
		if exists {
			return custom_err.ErrDuplicateRequest
		}

		// Получаем текущий баланс с блокировкой
		currentBalance, err := s.repo.GetWalletBalanceForUpdateTx(ctx, tx, req.WalletID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return custom_err.ErrNotFound
			}
			return fmt.Errorf("%s: failed to get balance: %w", op, err)
		}

		// Вычисляем новый баланс
		var newBalance int64
		switch req.OperationType {
		case models.OperationDeposit:
			newBalance = currentBalance + req.Amount
		case models.OperationWithdraw:
			newBalance = currentBalance - req.Amount
			if newBalance < 0 {
				return custom_err.ErrInsufficientFunds
			}
		default:
			return fmt.Errorf("%s: invalid operation type", op)
		}

		// Обновляем баланс
		if err := s.repo.UpdateBalanceTx(ctx, tx, req.WalletID, newBalance); err != nil {
			return fmt.Errorf("%s: failed to update balance: %w", op, err)
		}

		// Создаем запись об операции
		if err := s.repo.CreateOperationTx(ctx, tx, req.WalletID, req.Amount, req.RequestID); err != nil {
			return fmt.Errorf("%s: failed to create operation: %w", op, err)
		}

		return nil
	})
}

// GetWalletByID получает кошелек по ID (старый метод)
func (s *WalletService) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "service.GetWalletByID"

	wallet, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return wallet, nil
}

// ========== НОВЫЕ МЕТОДЫ (мультивалютность) ==========

// GetUserBalance получает балансы пользователя по всем валютам
func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalanceResponse, error) {
	const op = "service.GetUserBalance"

	// Получаем все кошельки пользователя
	wallets, err := s.repo.GetAllUserWallets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Инициализируем ответ с нулевыми балансами
	response := &models.UserBalanceResponse{
		USD: 0.0,
		RUB: 0.0,
		EUR: 0.0,
	}

	// Заполняем балансы из полученных кошельков
	for _, wallet := range wallets {
		balance := models.AmountFromMinorUnits(wallet.Balance)
		switch models.Currency(wallet.Currency) {
		case models.CurrencyUSD:
			response.USD = balance
		case models.CurrencyRUB:
			response.RUB = balance
		case models.CurrencyEUR:
			response.EUR = balance
		}
	}

	return response, nil
}

// Deposit пополняет кошелек пользователя в указанной валюте
func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, req models.DepositRequest) (*models.BalanceOperationResponse, error) {
	const op = "service.Deposit"

	// Валидация
	if !req.Currency.IsValid() {
		return nil, custom_err.ErrInvalidCurrency
	}
	if req.Amount <= 0 {
		return nil, custom_err.ErrInvalidAmount
	}
	if req.RequestID == "" {
		return nil, custom_err.ErrInvalidInput
	}

	// Получаем кошелек пользователя по валюте
	wallet, err := s.repo.GetByUserAndCurrency(ctx, userID, req.Currency)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	// Конвертируем сумму в минимальные единицы
	amountInMinorUnits := models.AmountToMinorUnits(req.Amount)

	// Используем существующий UpdateBalance для выполнения операции
	updateReq := models.WalletOperationRequest{
		WalletID:      wallet.ID,
		OperationType: models.OperationDeposit,
		Amount:        amountInMinorUnits,
		RequestID:     req.RequestID,
	}

	if err := s.UpdateBalance(ctx, updateReq); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Получаем обновленные балансы
	balances, err := s.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get updated balance: %w", op, err)
	}

	return &models.BalanceOperationResponse{
		Message:    "Account topped up successfully",
		NewBalance: *balances,
	}, nil
}

// Withdraw списывает средства с кошелька пользователя в указанной валюте
func (s *WalletService) Withdraw(ctx context.Context, userID uuid.UUID, req models.WithdrawRequest) (*models.BalanceOperationResponse, error) {
	const op = "service.Withdraw"

	// Валидация
	if !req.Currency.IsValid() {
		return nil, custom_err.ErrInvalidCurrency
	}
	if req.Amount <= 0 {
		return nil, custom_err.ErrInvalidAmount
	}
	if req.RequestID == "" {
		return nil, custom_err.ErrInvalidInput
	}

	// Получаем кошелек пользователя по валюте
	wallet, err := s.repo.GetByUserAndCurrency(ctx, userID, req.Currency)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	// Конвертируем сумму в минимальные единицы
	amountInMinorUnits := models.AmountToMinorUnits(req.Amount)

	// Используем существующий UpdateBalance для выполнения операции
	updateReq := models.WalletOperationRequest{
		WalletID:      wallet.ID,
		OperationType: models.OperationWithdraw,
		Amount:        amountInMinorUnits,
		RequestID:     req.RequestID,
	}

	if err := s.UpdateBalance(ctx, updateReq); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Получаем обновленные балансы
	balances, err := s.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get updated balance: %w", op, err)
	}

	return &models.BalanceOperationResponse{
		Message:    "Withdrawal successful",
		NewBalance: *balances,
	}, nil
}
