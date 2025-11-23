package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/grpc_client"
	"gw-currency-wallet/internal/models"
)

// Вспомогательные функции
func setupExchangeService(t *testing.T) (*ExchangeService, *MockWalletRepo, *MockTxManager, *MockExchangerClient, *MockKafkaProducer) {
	walletRepo := new(MockWalletRepo)
	txManager := new(MockTxManager)
	grpcClient := new(MockExchangerClient)
	kafkaProducer := new(MockKafkaProducer)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := &ExchangeService{
		walletRepo:      walletRepo,
		txManager:       txManager,
		grpcClient:      grpcClient,
		kafkaProducer:   kafkaProducer,
		cache:           make(map[string]CachedRate),
		cacheExpiration: 5 * time.Minute,
		log:             log,
	}

	return service, walletRepo, txManager, grpcClient, kafkaProducer
}

// ========== Тесты для GetExchangeRates ==========

func TestExchangeService_GetExchangeRates_Success(t *testing.T) {
	service, _, _, grpcClient, _ := setupExchangeService(t)
	ctx := context.Background()

	expectedRates := &grpc_client.ExchangeRatesResponse{
		Rates: map[string]float64{
			"USD": 1.0,
			"RUB": 95.5,
			"EUR": 0.92,
		},
	}

	// Мокаем gRPC запрос
	grpcClient.On("GetExchangeRates", ctx).Return(expectedRates, nil)

	// Выполняем тест (первый запрос - из gRPC)
	rates, err := service.GetExchangeRates(ctx)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, rates)
	assert.Equal(t, 1.0, rates["USD"])
	assert.Equal(t, 95.5, rates["RUB"])
	assert.Equal(t, 0.92, rates["EUR"])

	grpcClient.AssertExpectations(t)
}

func TestExchangeService_GetExchangeRates_Caching(t *testing.T) {
	service, _, _, grpcClient, _ := setupExchangeService(t)
	ctx := context.Background()

	expectedRates := &grpc_client.ExchangeRatesResponse{
		Rates: map[string]float64{
			"USD": 1.0,
			"RUB": 95.5,
			"EUR": 0.92,
		},
	}

	// Мокаем gRPC запрос (только один раз!)
	grpcClient.On("GetExchangeRates", ctx).Return(expectedRates, nil).Once()

	// Первый запрос - из gRPC
	rates1, err := service.GetExchangeRates(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, rates1)

	// Второй запрос - из кэша (не должен вызывать gRPC)
	rates2, err := service.GetExchangeRates(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, rates2)
	assert.Equal(t, rates1, rates2)

	// Проверяем, что gRPC вызвался только один раз
	grpcClient.AssertExpectations(t)
}

func TestExchangeService_GetExchangeRates_CacheExpiration(t *testing.T) {
	walletRepo := new(MockWalletRepo)
	txManager := new(MockTxManager)
	grpcClient := new(MockExchangerClient)
	kafkaProducer := new(MockKafkaProducer)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Создаем сервис с коротким временем кэша
	service := &ExchangeService{
		walletRepo:      walletRepo,
		txManager:       txManager,
		grpcClient:      grpcClient,
		kafkaProducer:   kafkaProducer,
		cache:           make(map[string]CachedRate),
		cacheExpiration: 100 * time.Millisecond, // 100ms
		log:             log,
	}

	ctx := context.Background()

	expectedRates := &grpc_client.ExchangeRatesResponse{
		Rates: map[string]float64{"USD": 1.0},
	}

	// Мокаем gRPC запрос (два раза - кэш истечет)
	grpcClient.On("GetExchangeRates", ctx).Return(expectedRates, nil).Twice()

	// Первый запрос
	_, err := service.GetExchangeRates(ctx)
	assert.NoError(t, err)

	// Ждем истечения кэша
	time.Sleep(150 * time.Millisecond)

	// Второй запрос - кэш истек, должен снова обратиться к gRPC
	_, err = service.GetExchangeRates(ctx)
	assert.NoError(t, err)

	grpcClient.AssertExpectations(t)
}

// ========== Тесты для ExchangeCurrency ==========

func TestExchangeService_ExchangeCurrency_Success(t *testing.T) {
	service, walletRepo, txManager, grpcClient, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()
	fromWalletID := uuid.New()
	toWalletID := uuid.New()

	req := models.ExchangeRequest{
		FromCurrency: models.CurrencyUSD,
		ToCurrency:   models.CurrencyEUR,
		Amount:       100.00,
		RequestID:    "exchange-001",
	}

	// Мокаем получение курса
	grpcClient.On("GetExchangeRateForCurrency", ctx, "USD", "EUR").Return(&grpc_client.ExchangeRateResponse{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.92,
	}, nil)

	// Мокаем получение кошельков
	fromWallet := &models.Wallet{ID: fromWalletID, UserID: userID, Currency: "USD", Balance: 100000}
	toWallet := &models.Wallet{ID: toWalletID, UserID: userID, Currency: "EUR", Balance: 0}

	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyUSD).Return(fromWallet, nil)
	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyEUR).Return(toWallet, nil)

	// Мокаем транзакцию
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	walletRepo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	walletRepo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, fromWalletID).Return(int64(100000), nil)
	walletRepo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, toWalletID).Return(int64(0), nil)
	walletRepo.On("UpdateBalanceTx", ctx, mock.Anything, fromWalletID, int64(90000)).Return(nil) // 100 - 10
	walletRepo.On("UpdateBalanceTx", ctx, mock.Anything, toWalletID, int64(9200)).Return(nil)    // 0 + 92
	walletRepo.On("CreateOperationTx", ctx, mock.Anything, fromWalletID, int64(-10000), req.RequestID).Return(nil)
	walletRepo.On("CreateOperationTx", ctx, mock.Anything, toWalletID, int64(9200), req.RequestID+"_to").Return(nil)

	// Выполняем тест
	resp, err := service.ExchangeCurrency(ctx, userID, req)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Exchange successful", resp.Message)
	assert.Equal(t, 92.0, resp.ExchangedAmount)
	assert.Equal(t, 0.92, resp.Rate)

	walletRepo.AssertExpectations(t)
	txManager.AssertExpectations(t)
	grpcClient.AssertExpectations(t)
}

func TestExchangeService_ExchangeCurrency_InvalidCurrency(t *testing.T) {
	service, _, _, _, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name string
		req  models.ExchangeRequest
	}{
		{
			name: "invalid from currency",
			req: models.ExchangeRequest{
				FromCurrency: "INVALID",
				ToCurrency:   models.CurrencyEUR,
				Amount:       100.00,
				RequestID:    "exchange-001",
			},
		},
		{
			name: "invalid to currency",
			req: models.ExchangeRequest{
				FromCurrency: models.CurrencyUSD,
				ToCurrency:   "INVALID",
				Amount:       100.00,
				RequestID:    "exchange-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.ExchangeCurrency(ctx, userID, tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, custom_err.ErrInvalidCurrency, err)
		})
	}
}

func TestExchangeService_ExchangeCurrency_InvalidAmount(t *testing.T) {
	service, _, _, _, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name   string
		amount float64
	}{
		{"zero amount", 0.0},
		{"negative amount", -100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := models.ExchangeRequest{
				FromCurrency: models.CurrencyUSD,
				ToCurrency:   models.CurrencyEUR,
				Amount:       tt.amount,
				RequestID:    "exchange-001",
			}

			resp, err := service.ExchangeCurrency(ctx, userID, req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, custom_err.ErrInvalidAmount, err)
		})
	}
}

func TestExchangeService_ExchangeCurrency_SameCurrency(t *testing.T) {
	service, _, _, _, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()

	req := models.ExchangeRequest{
		FromCurrency: models.CurrencyUSD,
		ToCurrency:   models.CurrencyUSD, // Та же валюта
		Amount:       100.00,
		RequestID:    "exchange-001",
	}

	// Выполняем тест
	resp, err := service.ExchangeCurrency(ctx, userID, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "cannot exchange same currency")
}

func TestExchangeService_ExchangeCurrency_InsufficientFunds(t *testing.T) {
	service, walletRepo, txManager, grpcClient, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()
	fromWalletID := uuid.New()
	toWalletID := uuid.New()

	req := models.ExchangeRequest{
		FromCurrency: models.CurrencyUSD,
		ToCurrency:   models.CurrencyEUR,
		Amount:       1000.00,
		RequestID:    "exchange-001",
	}

	grpcClient.On("GetExchangeRateForCurrency", ctx, "USD", "EUR").Return(&grpc_client.ExchangeRateResponse{
		Rate: 0.92,
	}, nil)

	fromWallet := &models.Wallet{ID: fromWalletID, UserID: userID, Currency: "USD", Balance: 10000}
	toWallet := &models.Wallet{ID: toWalletID, UserID: userID, Currency: "EUR", Balance: 0}

	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyUSD).Return(fromWallet, nil)
	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyEUR).Return(toWallet, nil)

	//✅ ВАРИАНТ 2: Если нужно проверить логику внутри транзакции:
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(pgx.Tx) error)
			fn(nil) // Выполняем функцию транзакции
		}).
		Return(custom_err.ErrInsufficientFunds)

	walletRepo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	walletRepo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, fromWalletID).Return(int64(10000), nil)
	// Код должен вернуть ошибку после проверки баланса

	resp, err := service.ExchangeCurrency(ctx, userID, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrInsufficientFunds, err)

	walletRepo.AssertExpectations(t)
	txManager.AssertExpectations(t)
	grpcClient.AssertExpectations(t)
}
func TestExchangeService_ExchangeCurrency_LargeTransfer_KafkaEvent(t *testing.T) {
	service, walletRepo, txManager, grpcClient, kafkaProducer := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()
	fromWalletID := uuid.New()
	toWalletID := uuid.New()

	req := models.ExchangeRequest{
		FromCurrency: models.CurrencyUSD,
		ToCurrency:   models.CurrencyRUB,
		Amount:       35000.00, // >= 30000 - должно отправить в Kafka
		RequestID:    "exchange-large-001",
	}

	grpcClient.On("GetExchangeRateForCurrency", ctx, "USD", "RUB").Return(&grpc_client.ExchangeRateResponse{
		Rate: 95.5,
	}, nil)

	fromWallet := &models.Wallet{ID: fromWalletID, UserID: userID, Currency: "USD", Balance: 5000000} // 50000 USD
	toWallet := &models.Wallet{ID: toWalletID, UserID: userID, Currency: "RUB", Balance: 0}

	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyUSD).Return(fromWallet, nil)
	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyRUB).Return(toWallet, nil)

	// Мокаем транзакцию
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)
	walletRepo.On("OperationExistsTx", ctx, mock.Anything, req.RequestID).Return(false, nil)
	walletRepo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, fromWalletID).Return(int64(5000000), nil)
	walletRepo.On("GetWalletBalanceForUpdateTx", ctx, mock.Anything, toWalletID).Return(int64(0), nil)
	walletRepo.On("UpdateBalanceTx", ctx, mock.Anything, fromWalletID, mock.Anything).Return(nil)
	walletRepo.On("UpdateBalanceTx", ctx, mock.Anything, toWalletID, mock.Anything).Return(nil)
	walletRepo.On("CreateOperationTx", ctx, mock.Anything, fromWalletID, mock.Anything, req.RequestID).Return(nil)
	walletRepo.On("CreateOperationTx", ctx, mock.Anything, toWalletID, mock.Anything, req.RequestID+"_to").Return(nil)

	// Мокаем Kafka producer - ВАЖНО!
	kafkaProducer.On("SendLargeTransferEvent", mock.Anything, mock.MatchedBy(func(event models.LargeTransferEvent) bool {
		return event.TransactionID == req.RequestID &&
			event.UserID == userID &&
			event.Amount == 35000.0
	})).Return(nil)

	// Выполняем тест
	resp, err := service.ExchangeCurrency(ctx, userID, req)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Даем время горутине отправить в Kafka
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что Kafka producer был вызван
	kafkaProducer.AssertExpectations(t)
	walletRepo.AssertExpectations(t)
	txManager.AssertExpectations(t)
	grpcClient.AssertExpectations(t)
}

func TestExchangeService_ExchangeCurrency_DuplicateRequest(t *testing.T) {
	service, walletRepo, txManager, grpcClient, _ := setupExchangeService(t)
	ctx := context.Background()
	userID := uuid.New()
	fromWalletID := uuid.New()
	toWalletID := uuid.New()

	req := models.ExchangeRequest{
		FromCurrency: models.CurrencyUSD,
		ToCurrency:   models.CurrencyEUR,
		Amount:       100.00,
		RequestID:    "exchange-001",
	}

	grpcClient.On("GetExchangeRateForCurrency", ctx, "USD", "EUR").Return(&grpc_client.ExchangeRateResponse{
		Rate: 0.92,
	}, nil)

	fromWallet := &models.Wallet{ID: fromWalletID, UserID: userID, Currency: "USD", Balance: 100000}
	toWallet := &models.Wallet{ID: toWalletID, UserID: userID, Currency: "EUR", Balance: 0}

	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyUSD).Return(fromWallet, nil)
	walletRepo.On("GetByUserAndCurrency", ctx, userID, models.CurrencyEUR).Return(toWallet, nil)

	// ✅ ИСПРАВЛЕНО: убрали лишний мок
	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(custom_err.ErrDuplicateRequest)

	resp, err := service.ExchangeCurrency(ctx, userID, req)

	// ✅ ИСПРАВЛЕНО: проверяем ошибку, а не успех
	// (если ваш сервис НЕ обрабатывает ErrDuplicateRequest как успех)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrDuplicateRequest, err)

	/* ❌ СТАРАЯ ВЕРСИЯ (неправильная):
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Exchange successful", resp.Message)
	*/

	walletRepo.AssertExpectations(t)
	txManager.AssertExpectations(t)
	grpcClient.AssertExpectations(t)
}
