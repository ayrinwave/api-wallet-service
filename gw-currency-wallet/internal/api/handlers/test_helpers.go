package handlers

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

// contextWithLogger создает контекст с логгером для тестов
func contextWithLogger() context.Context {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Устанавливаем уровень ERROR чтобы не засорять вывод тестов
	}))
	return context.WithValue(context.Background(), "logger", logger)
}

// contextWithUserID добавляет userID в контекст (имитация JWT middleware)
// Это имитирует работу middlew.GetUserID из internal/api/middlew
func contextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// getUserIDFromContext извлекает userID из контекста (для проверки)
func getUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	return userID, ok
}
