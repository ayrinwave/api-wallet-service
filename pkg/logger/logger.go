package logger

import (
	"context"
	"io"
	"log"
	"os"

	"log/slog"
)

type LevelBasedMuxHandler struct {
	stdoutHandler slog.Handler
	fileHandler   slog.Handler
}

func NewLevelBasedMuxHandler(stdout, file io.Writer) *LevelBasedMuxHandler {
	return &LevelBasedMuxHandler{
		stdoutHandler: slog.NewTextHandler(stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
		fileHandler: slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		}),
	}
}

func (h *LevelBasedMuxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdoutHandler.Enabled(ctx, level) || h.fileHandler.Enabled(ctx, level)
}

func (h *LevelBasedMuxHandler) Handle(ctx context.Context, r slog.Record) error {
	// ИСПРАВЛЕНО: Условие изменено на WARN
	if r.Level >= slog.LevelWarn {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return h.stdoutHandler.Handle(ctx, r)
}

func (h *LevelBasedMuxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LevelBasedMuxHandler{
		stdoutHandler: h.stdoutHandler.WithAttrs(attrs),
		fileHandler:   h.fileHandler.WithAttrs(attrs),
	}
}

func (h *LevelBasedMuxHandler) WithGroup(name string) slog.Handler {
	return &LevelBasedMuxHandler{
		stdoutHandler: h.stdoutHandler.WithGroup(name),
		fileHandler:   h.fileHandler.WithGroup(name),
	}
}

func NewLogger() *slog.Logger {
	logFile, err := os.OpenFile("errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("не удалось открыть файл логов: %v", err)
	}

	handler := NewLevelBasedMuxHandler(os.Stdout, logFile)
	return slog.New(handler)
}
