package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Error   string `json:"error" example:"invalid_input"`
	Message string `json:"message,omitempty"`
}

func WriteJSONError(w http.ResponseWriter, log *slog.Logger, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: errCode, Message: message}); err != nil {
		log.Error("ошибка при кодировании JSON-ошибки", slog.String("error", err.Error()))
	}
}

func WriteJSONSuccess(w http.ResponseWriter, log *slog.Logger, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Error("ошибка при кодировании JSON-ответа", slog.String("error", err.Error()))
		}
	}
}
