package telemetry

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

// WithRequestID stores a request ID in context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext retrieves the request ID.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// NewRequestID generates a new random request ID.
func NewRequestID() string {
	return uuid.New().String()[:16]
}

// LoggerWithContext returns a zerolog.Logger with request_id attached.
func LoggerWithContext(base zerolog.Logger, ctx context.Context) zerolog.Logger {
	if rid := RequestIDFromContext(ctx); rid != "" {
		return base.With().Str("request_id", rid).Logger()
	}
	return base
}

// ContextWithLogger stores a logger in context for downstream use.
func ContextWithLogger(ctx context.Context, log zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// FromContext retrieves the stored logger, falling back to the global logger.
func FromContext(ctx context.Context) zerolog.Logger {
	if l, ok := ctx.Value(loggerKey).(zerolog.Logger); ok {
		return l
	}
	return zerolog.Logger{}
}
