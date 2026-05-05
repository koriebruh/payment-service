package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/koriebruh/payment-service/config"
)

type loggerKey struct{}

func New(cfg *config.AppConfig) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Env != "production", // source only in non-prod
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	return slog.New(handler).With(
		"service", cfg.Name,
		"version", cfg.Version,
		"env", cfg.Env,
	)
}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
