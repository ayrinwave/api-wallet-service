package handlers

import (
	"api_wallet/internal/api/middlew"
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/service"
	"api_wallet/pkg/response"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WalletHandler struct {
	service *service.WalletService
}

func NewWalletHandler(service *service.WalletService) *WalletHandler {
	return &WalletHandler{
		service: service,
	}
}

func (h *WalletHandler) GetWalletByID(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetWalletByID"
	log := middlew.GetLogger(r.Context())

	idStr := chi.URLParam(r, "walletID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("невалидный UUID", slog.String("op", op), slog.String("uuid", idStr))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_request", "Invalid wallet ID format")
		return
	}

	wallet, err := h.service.GetWalletByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("кошелек не найден", slog.String("op", op), slog.String("id", id.String()))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		default:
			log.Error("ошибка получения кошелька", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Failed to retrieve wallet")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, wallet)
}

func (h *WalletHandler) UpdateBalance(w http.ResponseWriter, r *http.Request) {
	const op = "handler.UpdateBalance"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	var req models.WalletOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("ошибка декодирования JSON", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	log.Info("Received request",
		slog.String("op", op),
		slog.String("walletID", req.WalletID.String()), // Покажет 00000000-0000... если nil
		slog.String("requestID", req.RequestID),
		slog.Any("operationType", req.OperationType),
		slog.Int64("amount", req.Amount))

	if req.WalletID == uuid.Nil {
		log.Warn("walletID обязателен", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "walletID is required and must be valid UUID")
		return
	}

	// Валидация requestID
	if req.RequestID == "" {
		log.Warn("requestID обязателен", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "requestID is required")
		return
	}

	if !req.OperationType.IsValid() {
		log.Warn("невалидный тип операции", slog.String("op", op), slog.Any("req", req))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "Invalid operationType")
		return
	}
	if req.Amount <= 0 {
		log.Warn("сумма операции должна быть положительной", slog.String("op", op), slog.Any("req", req))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "Amount must be positive")
		return
	}

	log.Info("WalletID после парсинга", slog.String("op", op), slog.String("wallet_id", req.WalletID.String()))

	err := h.service.UpdateBalance(r.Context(), req)

	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("кошелек не найден", slog.String("op", op), slog.Any("req", req))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrInsufficientFunds):
			log.Warn("недостаточно средств", slog.String("op", op), slog.Any("req", req))
			response.WriteJSONError(w, log, http.StatusBadRequest, "insufficient_funds", "Insufficient funds in the wallet")
		case errors.Is(err, custom_err.ErrDuplicateRequest):
			log.Info("операция уже выполнена", slog.String("op", op))
			// ✅ Возвращаем 200 OK для идемпотентности
			response.WriteJSONSuccess(w, log, http.StatusOK, map[string]string{
				"status":        "already_processed",
				"walletId":      req.WalletID.String(),
				"operationType": string(req.OperationType),
			})
		default:
			log.Error("не удалось выполнить операцию", slog.String("op", op), slog.String("error", err.Error()))
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
