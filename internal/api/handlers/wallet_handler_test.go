package handlers

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/service"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Эта строка проверит во время компиляции, что наш мок подходит под интерфейс.
var _ service.WalletServicer = (*mockWalletService)(nil)

// 1. Создаем "подделку" (мок) нашего сервиса
type mockWalletService struct {
	UpdateBalanceFunc func(ctx context.Context, req models.WalletOperationRequest) error
	GetWalletByIDFunc func(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
}

// Реализуем методы интерфейса, которые просто вызывают наши функции-заглушки
func (m *mockWalletService) UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error {
	if m.UpdateBalanceFunc != nil {
		return m.UpdateBalanceFunc(ctx, req)
	}
	return nil
}

func (m *mockWalletService) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	if m.GetWalletByIDFunc != nil {
		return m.GetWalletByIDFunc(ctx, id)
	}
	return nil, nil
}

// 2. Основной тест для хендлера UpdateBalance
func TestWalletHandler_UpdateBalance(t *testing.T) {
	// Создаем экземпляры мока и хендлера
	mockService := &mockWalletService{}
	handler := NewWalletHandler(mockService)

	// Определяем тестовые сценарии
	testCases := []struct {
		name           string
		inputBody      string
		mockError      error // Ошибка, которую вернет наш мок-сервис
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success - Deposit",
			inputBody:      `{"walletId": "a7c9a494-386b-436d-8a58-29b7a3f754a3", "operationType": "DEPOSIT", "amount": 100}`,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Error - Wallet Not Found",
			inputBody:      `{"walletId": "a7c9a494-386b-436d-8a58-29b7a3f754a3", "operationType": "DEPOSIT", "amount": 100}`,
			mockError:      custom_err.ErrNotFound, // Мок вернет ошибку "не найдено"
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"not_found","message":"Wallet not found"}`,
		},
		{
			name:           "Error - Insufficient Funds",
			inputBody:      `{"walletId": "a7c9a494-386b-436d-8a58-29b7a3f754a3", "operationType": "WITHDRAW", "amount": 500}`,
			mockError:      custom_err.ErrInsufficientFunds, // Мок вернет ошибку "недостаточно средств"
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"insufficient_funds","message":"Insufficient funds in the wallet"}`,
		},
		{
			name:           "Error - Invalid JSON",
			inputBody:      `{`,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid_json","message":"Invalid JSON body"}`,
		},
		{
			name:           "Error - Negative Amount",
			inputBody:      `{"walletId": "a7c9a494-386b-436d-8a58-29b7a3f754a3", "operationType": "DEPOSIT", "amount": -100}`,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid_field","message":"Amount must be positive"}`,
		},
		{
			name:           "Error - Internal Server Error",
			inputBody:      `{"walletId": "a7c9a494-386b-436d-8a58-29b7a3f754a3", "operationType": "DEPOSIT", "amount": 100}`,
			mockError:      errors.New("some unexpected database error"), // Мок вернет неизвестную ошибку
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal_error","message":"An internal error occurred"}`,
		},
	}

	// 3. Запускаем тесты в цикле
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Настраиваем мок-сервис для текущего теста
			mockService.UpdateBalanceFunc = func(ctx context.Context, req models.WalletOperationRequest) error {
				return tc.mockError
			}

			// Создаем фейковый HTTP-запрос
			req, err := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBufferString(tc.inputBody))
			require.NoError(t, err)

			// Создаем фейковый ResponseWriter для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем наш хендлер
			handler.UpdateBalance(rr, req)

			// 4. Проверяем результат
			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String())
			}
		})
	}
}

// 3. Тесты для GetWalletByID
func TestWalletHandler_GetWalletByID(t *testing.T) {
	mockService := &mockWalletService{}
	handler := NewWalletHandler(mockService)

	walletID := uuid.New()

	testCases := []struct {
		name           string
		walletIDParam  string
		mockWallet     *models.Wallet
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			walletIDParam:  walletID.String(),
			mockWallet:     &models.Wallet{ID: walletID, Balance: 123},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"id":"%s","balance":123,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, walletID.String()),
		},
		{
			name:           "Error - Not Found",
			walletIDParam:  walletID.String(),
			mockWallet:     nil,
			mockError:      custom_err.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"not_found","message":"Wallet not found"}`,
		},
		{
			name:           "Error - Invalid UUID",
			walletIDParam:  "not-a-valid-uuid",
			mockWallet:     nil,
			mockError:      nil, // Сервис не будет вызван
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid_request","message":"Invalid wallet ID format"}`,
		},
		{
			name:           "Error - Internal Server Error",
			walletIDParam:  walletID.String(),
			mockWallet:     nil,
			mockError:      errors.New("unexpected db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal_error","message":"Failed to retrieve wallet"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService.GetWalletByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
				return tc.mockWallet, tc.mockError
			}

			url := fmt.Sprintf("/api/v1/wallets/%s", tc.walletIDParam)
			req := httptest.NewRequest(http.MethodGet, url, nil)

			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("walletID", tc.walletIDParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

			rr := httptest.NewRecorder()
			handler.GetWalletByID(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String())
			}
		})
	}
}
