package middlew

import (
	"context"
	"errors"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/pkg/response"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const userIDKey = contextKey("user_id")

func RequireAuth(authService service.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := GetLogger(r.Context())

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("missing authorization header")
				response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Authorization header is required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.EqualFold(parts[0], "Bearer") {
				log.Warn("invalid authorization header format")
				response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				if errors.Is(err, custom_err.ErrInvalidToken) {
					log.Warn("invalid or expired token")
					response.WriteJSONError(w, log, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
				} else {
					log.Error("failed to validate token", slog.String("error", err.Error()))
					response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
				}
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)

			loggerWithUser := log.With(slog.String("user_id", claims.UserID.String()))
			ctx = context.WithValue(ctx, loggerKey, loggerWithUser)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, custom_err.ErrUnauthorized
	}
	return userID, nil
}
