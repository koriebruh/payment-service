// internal/middleware/request_logger.go
// Layer: Middleware (wraps adapter/handler)
//
// Logs every HTTP request/response with:
//   - request_id, method, path, status, latency_ms
//
// Only logs: completion line + error details when status >= 500.
// Does NOT dump headers or body — that would be excessive and a security risk.

package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/koriebruh/payment-service/pkg/logger"
)

// RequestLogger produces one structured log line per request.
// INFO for 2xx/3xx/4xx, ERROR for 5xx.
func RequestLogger(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Generate request_id if not provided by client
		requestID := c.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		// Extract trace_id from OTEL context
		span := trace.SpanFromContext(c.UserContext())
		traceID := span.SpanContext().TraceID().String()

		// Store in locals so handlers can access it
		c.Locals("request_id", requestID)
		c.Locals("trace_id", traceID)

		// Create request-scoped logger and store in context
		reqLogger := log.With("request_id", requestID, "trace_id", traceID)
		ctx := logger.WithContext(c.UserContext(), reqLogger)
		c.SetUserContext(ctx)

		// Execute handler chain
		err := c.Next()

		// Log completion
		latency := time.Since(start)
		status := c.Response().StatusCode()

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
		}

		switch {
		case status >= 500:
			if err != nil {
				attrs = append(attrs, "error", err.Error())
			}
			reqLogger.Error("request completed", attrs...)
		case status >= 400:
			reqLogger.Warn("request completed", attrs...)
		default:
			reqLogger.Info("request completed", attrs...)
		}

		return err
	}
}
