package service

import (
	"context"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/grpc_client"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/repository/postgres"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CachedRate структура для хранения закэшированного курса
type CachedRate struct {
	Rate      float64
	Timestamp time.Time
}

// Exchange интерфейс для работы с обменом валют
type Exchange interface {
	GetExchangeRates(ctx context.Context) (map[string]float64, error)
	ExchangeCurrency(ctx context.Context, userID uuid.UUID, req models.ExchangeRequest) (*models.ExchangeResponse, error)
}

type ExchangeService struct {
	walletRepo      postgres.WalletRepository
	txManager       TxManager
	grpcClient      grpc_client.ExchangerClient
	cache           map[string]CachedRate
	cacheMutex      sync.RWMutex
	cacheExpiration time.Duration
	log             *slog.Logger
}

func NewExchangeService(
	walletRepo postgres.WalletRepository,
	txManager TxManager,
	grpcClient grpc_client.ExchangerClient,
	cacheExpiration time.Duration,
	log *slog.Logger,
) Exchange {
	return &ExchangeService{
		walletRepo:      walletRepo,
		txManager:       txManager,
		grpcClient:      grpcClient,
		cache:           make(map[string]CachedRate),
		cacheExpiration: cacheExpiration,
		log:             log,
	}
}

// GetExchangeRates получает курсы валют (с кэшированием)
func (s *ExchangeService) GetExchangeRates(ctx context.Context) (map[string]float64, error) {
	const op = "service.GetExchangeRates"

	// Проверяем кэш
	s.cacheMutex.RLock()
	cacheKey := "all_rates"
	if cached, ok := s.cache[cacheKey]; ok {
		if time.Since(cached.Timestamp) < s.cacheExpiration {
			s.log.Debug("курсы взяты из кэша")
			s.cacheMutex.RUnlock()

			// Возвращаем все курсы из кэша
			rates := make(map[string]float64)
			for k, v := range s.cache {
				if k != "all_rates" {
					rates[k] = v.Rate
				}
			}
			return rates, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Запрашиваем у gRPC сервиса
	s.log.Debug("запрос курсов у exchanger сервиса")
	resp, err := s.grpcClient.GetExchangeRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Обновляем кэш
	s.cacheMutex.Lock()
	now := time.Now()
	for currency, rate := range resp.Rates {
		s.cache[currency] = CachedRate{
			Rate:      rate,
			Timestamp: now,
		}
	}
	s.cache[cacheKey] = CachedRate{Timestamp: now}
	s.cacheMutex.Unlock()

	s.log.Debug("курсы обновлены в кэше")

	return resp.Rates, nil
}

// getExchangeRate получает курс для конкретной пары валют (с кэшированием)
func (s *ExchangeService) getExchangeRate(ctx context.Context, from, to string) (float64, error) {
	const op = "service.getExchangeRate"

	cacheKey := fmt.Sprintf("%s_%s", from, to)

	// Проверяем кэш
	s.cacheMutex.RLock()
	if cached, ok := s.cache[cacheKey]; ok {
		if time.Since(cached.Timestamp) < s.cacheExpiration {
			s.log.Debug("курс взят из кэша",
				slog.String("from", from),
				slog.String("to", to),
				slog.Float64("rate", cached.Rate))
			s.cacheMutex.RUnlock()
			return cached.Rate, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Запрашиваем у gRPC сервиса
	s.log.Debug("запрос курса у exchanger сервиса",
		slog.String("from", from),
		slog.String("to", to))

	resp, err := s.grpcClient.GetExchangeRateForCurrency(ctx, from, to)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Обновляем кэш
	s.cacheMutex.Lock()
	s.cache[cacheKey] = CachedRate{
		Rate:      resp.Rate,
		Timestamp: time.Now(),
	}
	s.cacheMutex.Unlock()

	s.log.Debug("курс обновлен в кэше",
		slog.String("from", from),
		slog.String("to", to),
		slog.Float64("rate", resp.Rate))

	return resp.Rate, nil
}

// ExchangeCurrency выполняет обмен валют
func (s *ExchangeService) ExchangeCurrency(ctx context.Context, userID uuid.UUID, req models.ExchangeRequest) (*models.ExchangeResponse, error) {
	const op = "service.ExchangeCurrency"

	// Валидация
	if !req.FromCurrency.IsValid() || !req.ToCurrency.IsValid() {
		return nil, custom_err.ErrInvalidCurrency
	}
	if req.Amount <= 0 {
		return nil, custom_err.ErrInvalidAmount
	}
	if req.FromCurrency == req.ToCurrency {
		return nil, fmt.Errorf("%s: cannot exchange same currency", op)
	}
	if req.RequestID == "" {
		return nil, custom_err.ErrInvalidInput
	}

	// Получаем курс обмена
	rate, err := s.getExchangeRate(ctx, string(req.FromCurrency), string(req.ToCurrency))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get exchange rate: %w", op, err)
	}

	// Вычисляем сумму после обмена
	exchangedAmount := req.Amount * rate

	s.log.Info("обмен валют",
		slog.String("user_id", userID.String()),
		slog.String("from", string(req.FromCurrency)),
		slog.String("to", string(req.ToCurrency)),
		slog.Float64("amount", req.Amount),
		slog.Float64("rate", rate),
		slog.Float64("exchanged_amount", exchangedAmount))

	// Выполняем операцию в транзакции
	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// Проверяем идемпотентность (используем тот же requestID)
		exists, err := s.walletRepo.OperationExistsTx(ctx, tx, req.RequestID)
		if err != nil {
			return fmt.Errorf("failed to check operation: %w", err)
		}
		if exists {
			return custom_err.ErrDuplicateRequest
		}

		// Получаем кошелек источника (FROM)
		fromWallet, err := s.walletRepo.GetByUserAndCurrency(ctx, userID, req.FromCurrency)
		if err != nil {
			return fmt.Errorf("failed to get source wallet: %w", err)
		}

		// Получаем кошелек назначения (TO)
		toWallet, err := s.walletRepo.GetByUserAndCurrency(ctx, userID, req.ToCurrency)
		if err != nil {
			return fmt.Errorf("failed to get destination wallet: %w", err)
		}

		// Конвертируем суммы в минимальные единицы
		amountInMinorUnits := models.AmountToMinorUnits(req.Amount)
		exchangedAmountInMinorUnits := models.AmountToMinorUnits(exchangedAmount)

		// Списываем с исходного кошелька
		fromBalance, err := s.walletRepo.GetWalletBalanceForUpdateTx(ctx, tx, fromWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to get source balance: %w", err)
		}

		newFromBalance := fromBalance - amountInMinorUnits
		if newFromBalance < 0 {
			return custom_err.ErrInsufficientFunds
		}

		if err := s.walletRepo.UpdateBalanceTx(ctx, tx, fromWallet.ID, newFromBalance); err != nil {
			return fmt.Errorf("failed to update source balance: %w", err)
		}

		// Пополняем целевой кошелек
		toBalance, err := s.walletRepo.GetWalletBalanceForUpdateTx(ctx, tx, toWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to get destination balance: %w", err)
		}

		newToBalance := toBalance + exchangedAmountInMinorUnits

		if err := s.walletRepo.UpdateBalanceTx(ctx, tx, toWallet.ID, newToBalance); err != nil {
			return fmt.Errorf("failed to update destination balance: %w", err)
		}

		// Создаем записи об операциях (для обоих кошельков)
		if err := s.walletRepo.CreateOperationTx(ctx, tx, fromWallet.ID, -amountInMinorUnits, req.RequestID); err != nil {
			return fmt.Errorf("failed to create source operation: %w", err)
		}

		if err := s.walletRepo.CreateOperationTx(ctx, tx, toWallet.ID, exchangedAmountInMinorUnits, req.RequestID+"_to"); err != nil {
			return fmt.Errorf("failed to create destination operation: %w", err)
		}

		return nil
	})

	if err != nil {
		// Обрабатываем идемпотентность на уровне сервиса
		if err == custom_err.ErrDuplicateRequest {
			s.log.Info("операция обмена уже была выполнена", slog.String("request_id", req.RequestID))
			// Возвращаем текущие балансы
			// (можно было бы сохранить результат обмена в БД и вернуть его)
			return &models.ExchangeResponse{
				Message:         "Exchange successful",
				ExchangedAmount: exchangedAmount,
				Rate:            rate,
			}, nil
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.ExchangeResponse{
		Message:         "Exchange successful",
		ExchangedAmount: exchangedAmount,
		Rate:            rate,
	}, nil
}
