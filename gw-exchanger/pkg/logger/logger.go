package logger

import (
	"context"
	"io"
	"log"
	"os"

	"log/slog"
)

// LevelBasedMuxHandler - мультиплексор для разных выходов логов
type LevelBasedMuxHandler struct {
	stdoutHandler slog.Handler
	fileHandler   slog.Handler
}

type LoggerWithFile struct {
	Logger  *slog.Logger
	LogFile *os.File
}

// NewLevelBasedMuxHandler создает handler с разными уровнями для stdout и файла
func NewLevelBasedMuxHandler(stdout, file io.Writer) *LevelBasedMuxHandler {
	return &LevelBasedMuxHandler{
		// Консоль: JSON формат, INFO+ (удобнее для production)
		stdoutHandler: slog.NewJSONHandler(stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false, // Можно включить для отладки
		}),
		// Файл: JSON формат, INFO+ (сохраняем всю важную информацию)
		fileHandler: slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true, // Добавляем файл и строку для отладки
		}),
	}
}

func (h *LevelBasedMuxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdoutHandler.Enabled(ctx, level) || h.fileHandler.Enabled(ctx, level)
}

func (h *LevelBasedMuxHandler) Handle(ctx context.Context, r slog.Record) error {
	// Пишем в файл все логи >= INFO
	if r.Level >= slog.LevelInfo {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	// Пишем в консоль все логи >= INFO
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

//// NewLogger создает логгер с записью в файл и stdout
//// Файл НЕ удаляется при перезапуске (append mode)
//func NewLogger() *LoggerWithFile {
//	// Имя файла лога
//	logFileName := "service.log"
//
//	// Открываем файл в режиме APPEND (не удаляем старые логи!)
//	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		log.Fatalf("не удалось открыть файл логов: %v", err)
//	}
//
//	handler := NewLevelBasedMuxHandler(os.Stdout, logFile)
//	return &LoggerWithFile{
//		Logger:  slog.New(handler),
//		LogFile: logFile,
//	}
//}

// NewLoggerWithFile - вариант с кастомным именем файла
func NewLoggerWithFile(fileName string) *LoggerWithFile {
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("не удалось открыть файл логов: %v", err)
	}

	handler := NewLevelBasedMuxHandler(os.Stdout, logFile)
	return &LoggerWithFile{
		Logger:  slog.New(handler),
		LogFile: logFile,
	}
}
