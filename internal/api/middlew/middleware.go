package middlew

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const loggerKey = contextKey("logger")

func WithLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := middleware.GetReqID(r.Context())
			loggerWithTrace := log.With(slog.String("trace_id", traceID))

			ctx := context.WithValue(r.Context(), loggerKey, loggerWithTrace)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
