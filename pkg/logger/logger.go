package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/koriebruh/payment-service/config"
)

type loggerKey struct{}

// New creates a structured JSON logger.
// Writes to stdout always; also writes to file if cfg.LogFilePath is set.
// Every log entry includes: service, env — enabling centralized log aggregation.
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
		AddSource: cfg.Env != "production",
	}

	// Build writer: stdout + optional file
	var w io.Writer = os.Stdout
	if cfg.LogFilePath != "" {
		dir := filepath.Dir(cfg.LogFilePath)
		if err := os.MkdirAll(dir, 0o750); err == nil {
			f, err := os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err == nil {
				w = io.MultiWriter(os.Stdout, f)
			}
		}
	}

	handler := slog.NewJSONHandler(w, opts)

	l := slog.New(handler).With(
		"service", cfg.Name,
		"env", cfg.Env,
	)

	// Set as default so slog.Info(...) in adapter layer also goes through our handler
	slog.SetDefault(l)

	return l
}

// WithContext stores a logger (with request-scoped fields like request_id) in context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves the logger from context.
// Falls back to slog.Default() if no logger is stored — never returns nil.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
