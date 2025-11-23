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
)

type ExchangeHandler struct {
	service service.Exchange
}

func NewExchangeHandler(service service.Exchange) *ExchangeHandler {
	return &ExchangeHandler{
		service: service,
	}
}

// GetExchangeRates godoc
// @Summary      Получить курсы валют
// @Description  Возвращает текущие курсы обмена для всех поддерживаемых валют относительно USD
// @Tags         exchange
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.ExchangeRatesResponse "Курсы валют"
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /exchange/rates [get]

// GetExchangeRates получает курсы всех валют
// GET /api/v1/exchange/rates
func (h *ExchangeHandler) GetExchangeRates(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetExchangeRates"
	log := middlew.GetLogger(r.Context())

	// Извлекаем userID (JWT middleware уже проверил)
	userID, err := middlew.GetUserID(r.Context())
	if err != nil {
		log.Error("failed to get user ID from context", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	log.Info("запрос курсов валют", slog.String("op", op), slog.String("user_id", userID.String()))

	rates, err := h.service.GetExchangeRates(r.Context())
	if err != nil {
		log.Error("failed to get exchange rates", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Failed to retrieve exchange rates")
		return
	}

	// Формируем ответ согласно спецификации
	responseData := models.ExchangeRatesResponse{
		Rates: rates,
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, responseData)
}

// ExchangeCurrency godoc
// @Summary      Обменять валюту
// @Description  Выполняет обмен одной валюты на другую по текущему курсу. При обмене суммы >= 30000 USD отправляется уведомление в Kafka
// @Tags         exchange
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.ExchangeRequest true "Данные для обмена валют"
// @Success      200 {object} models.ExchangeResponse "Обмен выполнен успешно"
// @Failure      400 {object} response.ErrorResponse "Невалидные данные, недостаточно средств или одинаковые валюты"
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      404 {object} response.ErrorResponse "Кошелек не найден"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /exchange [post]

// ExchangeCurrency выполняет обмен валют
// POST /api/v1/exchange
func (h *ExchangeHandler) ExchangeCurrency(w http.ResponseWriter, r *http.Request) {
	const op = "handler.ExchangeCurrency"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	// Извлекаем userID из контекста
	userID, err := middlew.GetUserID(r.Context())
	if err != nil {
		log.Error("failed to get user ID from context", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var req models.ExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	log.Info("запрос на обмен валют",
		slog.String("op", op),
		slog.String("user_id", userID.String()),
		slog.String("from", string(req.FromCurrency)),
		slog.String("to", string(req.ToCurrency)),
		slog.Float64("amount", req.Amount))

	// Валидация
	if !req.FromCurrency.IsValid() {
		log.Warn("invalid from_currency", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid from_currency")
		return
	}
	if !req.ToCurrency.IsValid() {
		log.Warn("invalid to_currency", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid to_currency")
		return
	}
	if req.FromCurrency == req.ToCurrency {
		log.Warn("same currencies", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_request", "Cannot exchange same currency")
		return
	}
	if req.Amount <= 0 {
		log.Warn("amount must be positive", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Amount must be positive")
		return
	}
	if req.RequestID == "" {
		log.Warn("requestID is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "requestID is required")
		return
	}

	result, err := h.service.ExchangeCurrency(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrNotFound):
			log.Info("wallet not found", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusNotFound, "not_found", "Wallet not found")
		case errors.Is(err, custom_err.ErrInsufficientFunds):
			log.Warn("insufficient funds", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "insufficient_funds", "Insufficient funds for exchange")
		case errors.Is(err, custom_err.ErrInvalidCurrency):
			log.Warn("invalid currency", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_currency", "Invalid currencies")
		case errors.Is(err, custom_err.ErrInvalidAmount):
			log.Warn("invalid amount", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_amount", "Invalid amount")
		default:
			log.Error("failed to exchange currency", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, result)
}
