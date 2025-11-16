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

type AuthHandler struct {
	service service.Auth // ← Используем интерфейс вместо конкретного типа
}

func NewAuthHandler(service service.Auth) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// Register обрабатывает регистрацию нового пользователя
// POST /api/v1/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	const op = "handler.Register"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON body", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Базовая валидация
	if req.Username == "" {
		log.Warn("username is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "username is required")
		return
	}
	if req.Password == "" {
		log.Warn("password is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "password is required")
		return
	}
	if req.Email == "" {
		log.Warn("email is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "email is required")
		return
	}

	// Дополнительная валидация
	if len(req.Username) < 3 || len(req.Username) > 50 {
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "username must be between 3 and 50 characters")
		return
	}
	if len(req.Password) < 6 {
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "password must be at least 6 characters")
		return
	}

	log.Info("registering new user",
		slog.String("op", op),
		slog.String("username", req.Username),
		slog.String("email", req.Email))

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrUsernameExists):
			log.Info("username already exists", slog.String("op", op), slog.String("username", req.Username))
			response.WriteJSONError(w, log, http.StatusBadRequest, "username_exists", "Username already exists")
		case errors.Is(err, custom_err.ErrEmailExists):
			log.Info("email already exists", slog.String("op", op), slog.String("email", req.Email))
			response.WriteJSONError(w, log, http.StatusBadRequest, "email_exists", "Email already exists")
		case errors.Is(err, custom_err.ErrInvalidInput):
			log.Warn("invalid input", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_input", "Invalid input data")
		default:
			log.Error("failed to register user", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusCreated, resp)
}

// Login обрабатывает авторизацию пользователя
// POST /api/v1/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	const op = "handler.Login"
	log := middlew.GetLogger(r.Context())

	defer r.Body.Close()

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON body", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	// Валидация
	if req.Username == "" {
		log.Warn("username is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "username is required")
		return
	}
	if req.Password == "" {
		log.Warn("password is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "password is required")
		return
	}

	log.Info("user login attempt", slog.String("op", op), slog.String("username", req.Username))

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrInvalidCredentials):
			log.Info("invalid credentials", slog.String("op", op), slog.String("username", req.Username))
			response.WriteJSONError(w, log, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
		case errors.Is(err, custom_err.ErrInvalidInput):
			log.Warn("invalid input", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_input", "Invalid input data")
		default:
			log.Error("failed to login user", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, resp)
}
