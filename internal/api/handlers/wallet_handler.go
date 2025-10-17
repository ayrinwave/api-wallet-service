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
	service service.WalletServicer
}

func NewWalletHandler(service service.WalletServicer) *WalletHandler {
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

	err := h.service.UpdateBalance(r.Context(), req)

	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("кошелек не найден", slog.String("op", op), slog.Any("req", req))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrInsufficientFunds):
			log.Warn("недостаточно средств", slog.String("op", op), slog.Any("req", req))
			response.WriteJSONError(w, log, http.StatusBadRequest, "insufficient_funds", "Insufficient funds in the wallet")
		default:
			log.Error("не удалось выполнить операцию", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
