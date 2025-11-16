//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"api_wallet/internal/models"
	"api_wallet/internal/repository/postgres"
	"api_wallet/internal/service"
	"api_wallet/internal/testhelpers"
	"api_wallet/pkg/logger"
)

func setupHandler(t *testing.T) (*WalletHandler, *testhelpers.TestDB, *chi.Mux) {
	t.Helper()

	testDB := testhelpers.SetupTestDB(t)
	testDB.RunMigrations(t)
	testDB.CleanupDB(t)

	repo := postgres.NewWalletRepository(testDB.Pool)
	txManager := service.NewPgxTxManager(testDB.Pool)
	svc := service.NewWalletService(repo, txManager)
	handler := NewWalletHandler(svc)

	// Setup router
	r := chi.NewRouter()
	r.Post("/api/v1/wallet", handler.UpdateBalance)
	r.Get("/api/v1/wallets/{walletID}", handler.GetWalletByID)

	return handler, testDB, r
}

func TestWalletHandler_UpdateBalance_Deposit_Success(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "DEPOSIT",
		"amount":        500,
		"requestID":     "http-deposit-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Add logger to context (для middleware)
	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	// Проверяем баланс в БД
	var balance int64
	err = testDB.Pool.QueryRow(context.Background(),
		"SELECT balance FROM wallets WHERE id = $1", walletID,
	).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), balance)
}

func TestWalletHandler_UpdateBalance_Withdraw_Success(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "WITHDRAW",
		"amount":        300,
		"requestID":     "http-withdraw-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])

	// Проверяем баланс в БД
	var balance int64
	err = testDB.Pool.QueryRow(context.Background(),
		"SELECT balance FROM wallets WHERE id = $1", walletID,
	).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(700), balance)
}

func TestWalletHandler_UpdateBalance_InsufficientFunds(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 100)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "WITHDRAW",
		"amount":        500,
		"requestID":     "http-insufficient-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "insufficient_funds", response["error"])
}

func TestWalletHandler_UpdateBalance_WalletNotFound(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	nonExistentID := uuid.New()

	requestBody := map[string]interface{}{
		"walletId":      nonExistentID.String(),
		"operationType": "DEPOSIT",
		"amount":        100,
		"requestID":     "http-notfound-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_found", response["error"])
}

func TestWalletHandler_UpdateBalance_InvalidJSON(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	invalidJSON := []byte(`{"walletId": "invalid-json"`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_json", response["error"])
}

func TestWalletHandler_UpdateBalance_InvalidWalletID(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	requestBody := map[string]interface{}{
		"walletId":      "00000000-0000-0000-0000-000000000000", // nil UUID
		"operationType": "DEPOSIT",
		"amount":        100,
		"requestID":     "http-invalid-id-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_field", response["error"])
}

func TestWalletHandler_UpdateBalance_InvalidOperationType(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "TRANSFER", // неверный тип
		"amount":        100,
		"requestID":     "http-invalid-op-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_field", response["error"])
}

func TestWalletHandler_UpdateBalance_NegativeAmount(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "DEPOSIT",
		"amount":        -100, // отрицательная сумма
		"requestID":     "http-negative-1",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_field", response["error"])
}

func TestWalletHandler_UpdateBalance_EmptyRequestID(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "DEPOSIT",
		"amount":        100,
		"requestID":     "", // пустой requestID
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_field", response["error"])
}

func TestWalletHandler_UpdateBalance_Idempotency(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 1000)

	requestBody := map[string]interface{}{
		"walletId":      walletID.String(),
		"operationType": "DEPOSIT",
		"amount":        100,
		"requestID":     "idempotent-http-request",
	}
	body, _ := json.Marshal(requestBody)

	// Первый запрос
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	ctx1 := context.WithValue(req1.Context(), "logger", logger.NewLogger())
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()

	// Второй запрос (дубликат)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	ctx2 := context.WithValue(req2.Context(), "logger", logger.NewLogger())
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w1, req1)
	router.ServeHTTP(w2, req2)

	// Assert
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Баланс должен увеличиться только один раз
	var balance int64
	err := testDB.Pool.QueryRow(context.Background(),
		"SELECT balance FROM wallets WHERE id = $1", walletID,
	).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1100), balance)
}

func TestWalletHandler_GetWalletByID_Success(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	walletID := uuid.New()
	testDB.SeedWallet(t, walletID.String(), 5000)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wallets/%s", walletID.String()), nil)

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var wallet models.Wallet
	err := json.Unmarshal(w.Body.Bytes(), &wallet)
	require.NoError(t, err)
	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, int64(5000), wallet.Balance)
}

func TestWalletHandler_GetWalletByID_NotFound(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	nonExistentID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wallets/%s", nonExistentID.String()), nil)

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "not_found", response["error"])
}

func TestWalletHandler_GetWalletByID_InvalidUUID(t *testing.T) {
	// Arrange
	_, testDB, router := setupHandler(t)
	defer testDB.TeardownTestDB()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/invalid-uuid", nil)

	ctx := context.WithValue(req.Context(), "logger", logger.NewLogger())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", response["error"])
}
