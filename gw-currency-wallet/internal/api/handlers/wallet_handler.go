package handlers

import (
	"encoding/json"
	"errors"
	"gw-currency-wallet/internal/api/middlew"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/pkg/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WalletHandler struct {
	service service.Wallet
}

func NewWalletHandler(service service.Wallet) *WalletHandler {
	return &WalletHandler{
		service: service,
	}
}

// ========== СТАРЫЕ HANDLERS (для совместимости) ==========

// GetWalletByID получает кошелек по ID
// GET /api/v1/wallets/{walletID}
func (h *WalletHandler) GetWalletByID(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetWalletByID"
	log := middlew.GetLogger(r.Context())

	idStr := chi.URLParam(r, "walletID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("invalid UUID", slog.String("op", op), slog.String("uuid", idStr))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_request", "Invalid wallet ID format")
		return
	}

	wallet, err := h.service.GetWalletByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("wallet not found", slog.String("op", op), slog.String("id", id.String()))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		default:
			log.Error("failed to get wallet", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Failed to retrieve wallet")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, wallet)
}

// UpdateBalance обновляет баланс кошелька (старый метод)
// POST /api/v1/wallet
func (h *WalletHandler) UpdateBalance(w http.ResponseWriter, r *http.Request) {
	const op = "handler.UpdateBalance"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	var req models.WalletOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Валидация
	if req.WalletID == uuid.Nil {
		log.Warn("walletID is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "walletID is required and must be valid UUID")
		return
	}
	if req.RequestID == "" {
		log.Warn("requestID is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "requestID is required")
		return
	}
	if !req.OperationType.IsValid() {
		log.Warn("invalid operation type", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "Invalid operationType")
		return
	}
	if req.Amount <= 0 {
		log.Warn("amount must be positive", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "Amount must be positive")
		return
	}

	err := h.service.UpdateBalance(r.Context(), req)

	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("wallet not found", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrInsufficientFunds):
			log.Warn("insufficient funds", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "insufficient_funds", "Insufficient funds in the wallet")
		case errors.Is(err, custom_err.ErrDuplicateRequest):
			log.Info("operation already processed", slog.String("op", op))
			response.WriteJSONSuccess(w, log, http.StatusOK, map[string]string{
				"status":        "already_processed",
				"walletId":      req.WalletID.String(),
				"operationType": string(req.OperationType),
			})
		default:
			log.Error("failed to execute operation", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, map[string]string{
		"status":        "success",
		"walletId":      req.WalletID.String(),
		"operationType": string(req.OperationType),
	})
}

// ========== НОВЫЕ HANDLERS (мультивалютность) ==========

// GetBalance godoc
// @Summary      Получить баланс
// @Tags         wallet
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} models.BalanceResponse
// @Failure      401 {object} response.ErrorResponse
// @Router       /balance [get]

// GetBalance получает балансы пользователя по всем валютам
// GET /api/v1/balance
func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetBalance"
	log := middlew.GetLogger(r.Context())

	// Извлекаем userID из контекста (добавлен JWT middleware)
	userID, err := middlew.GetUserID(r.Context())
	if err != nil {
		log.Error("failed to get user ID from context", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	log.Info("getting user balance", slog.String("op", op), slog.String("user_id", userID.String()))

	balances, err := h.service.GetUserBalance(r.Context(), userID)
	if err != nil {
		log.Error("failed to get balance", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Failed to retrieve balance")
		return
	}

	// Формируем ответ согласно спецификации
	responseData := map[string]interface{}{
		"balance": balances,
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, responseData)
}

// Deposit godoc
// @Summary      Пополнить кошелек
// @Tags         wallet
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body models.DepositRequest true "Данные пополнения"
// @Success      200 {object} models.DepositResponse
// @Failure      400 {object} response.ErrorResponse
// @Router       /wallet/deposit [post]

// Deposit пополняет кошелек пользователя
// POST /api/v1/wallet/deposit
func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	const op = "handler.Deposit"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	// Извлекаем userID из контекста
	userID, err := middlew.GetUserID(r.Context())
	if err != nil {
		log.Error("failed to get user ID from context", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	log.Info("deposit request",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.Float64("amount", req.Amount),
		slog.String("currency", string(req.Currency)))

	// Валидация
	if req.Amount <= 0 {
		log.Warn("amount must be positive", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Amount must be positive")
		return
	}
	if !req.Currency.IsValid() {
		log.Warn("invalid currency", slog.String("op", op), slog.String("currency", string(req.Currency)))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid currency")
		return
	}
	if req.RequestID == "" {
		log.Warn("requestID is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "requestID is required")
		return
	}

	result, err := h.service.Deposit(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("wallet not found", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrDuplicateRequest):
			log.Info("operation already processed", slog.String("op", op))
			// Получаем текущий баланс для идемпотентного ответа
			balances, _ := h.service.GetUserBalance(r.Context(), userID)
			response.WriteJSONSuccess(w, log, http.StatusOK, map[string]interface{}{
				"message":     "Account topped up successfully",
				"new_balance": balances,
			})
		case errors.Is(err, custom_err.ErrInvalidCurrency):
			log.Warn("invalid currency", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid currency")
		case errors.Is(err, custom_err.ErrInvalidAmount):
			log.Warn("invalid amount", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Invalid amount")
		default:
			log.Error("failed to deposit", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, result)
}

// Withdraw godoc
// @Summary      Вывести средства
// @Tags         wallet
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body models.WithdrawRequest true "Данные вывода"
// @Success      200 {object} models.WithdrawResponse
// @Failure      400 {object} response.ErrorResponse
// @Router       /wallet/withdraw [post]

// Withdraw списывает средства с кошелька пользователя
// POST /api/v1/wallet/withdraw
func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	const op = "handler.Withdraw"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	// Извлекаем userID из контекста
	userID, err := middlew.GetUserID(r.Context())
	if err != nil {
		log.Error("failed to get user ID from context", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	log.Info("withdraw request",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.Float64("amount", req.Amount),
		slog.String("currency", string(req.Currency)))

	// Валидация
	if req.Amount <= 0 {
		log.Warn("amount must be positive", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Amount must be positive")
		return
	}
	if !req.Currency.IsValid() {
		log.Warn("invalid currency", slog.String("op", op), slog.String("currency", string(req.Currency)))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid currency")
		return
	}
	if req.RequestID == "" {
		log.Warn("requestID is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "requestID is required")
		return
	}

	result, err := h.service.Withdraw(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("wallet not found", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrInsufficientFunds):
			log.Warn("insufficient funds", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "insufficient_funds", "Insufficient funds in the wallet")
		case errors.Is(err, custom_err.ErrDuplicateRequest):
			log.Info("operation already processed", slog.String("op", op))
			// Получаем текущий баланс для идемпотентного ответа
			balances, _ := h.service.GetUserBalance(r.Context(), userID)
			response.WriteJSONSuccess(w, log, http.StatusOK, map[string]interface{}{
				"message":     "Withdrawal successful",
				"new_balance": balances,
			})
		case errors.Is(err, custom_err.ErrInvalidCurrency):
			log.Warn("invalid currency", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid currency")
		case errors.Is(err, custom_err.ErrInvalidAmount):
			log.Warn("invalid amount", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Invalid amount")
		default:
			log.Error("failed to withdraw", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, result)
}
