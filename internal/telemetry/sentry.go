package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryzerolog "github.com/getsentry/sentry-go/zerolog"
	"github.com/rs/zerolog"
)

// SentryConfig holds Sentry initialization parameters.
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
}

// InitSentry initializes the Sentry SDK. If DSN is empty, it becomes a no-op.
func InitSentry(cfg SentryConfig, logger zerolog.Logger) error {
	if cfg.DSN == "" {
		logger.Info().Msg("sentry DSN not configured, error reporting disabled")
		return nil
	}

	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.TracesSampleRate <= 0 {
		cfg.TracesSampleRate = 0.1
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
		EnableTracing:    true,
	})
	if err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}

	// Hook zerolog Error+ into Sentry via multi-writer
	sw, err := sentryzerolog.New(sentryzerolog.Config{
		Options: sentryzerolog.Options{
			Levels: []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel},
		},
	})
	if err != nil {
		return fmt.Errorf("sentry zerolog writer: %w", err)
	}
	_ = sw // Writer captures events sent through it; route via SentryMiddleware instead

	logger.Info().
		Str("environment", cfg.Environment).
		Str("release", cfg.Release).
		Msg("sentry initialized")
	return nil
}

// FlushSentry blocks until all buffered events are sent (call on shutdown).
func FlushSentry() {
	sentry.Flush(2 * time.Second)
}

// SentryMiddleware is an HTTP middleware that creates a Sentry hub per request,
// captures panics, and reports 5xx responses.
func SentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		scope := hub.Scope()
		scope.SetRequest(r)
		scope.SetTag("http.method", r.Method)
		scope.SetTag("http.path", r.URL.Path)

		ctx := sentry.SetHubOnContext(r.Context(), hub)
		r = r.WithContext(ctx)

		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			if err := recover(); err != nil {
				hub.Recover(err)
				sentry.Flush(2 * time.Second)
				http.Error(ww, "internal server error", http.StatusInternalServerError)
			}
			if ww.statusCode >= 500 {
				hub.Scope().SetTag("http.status_code", fmt.Sprintf("%d", ww.statusCode))
				hub.CaptureEvent(&sentry.Event{
					Level:   sentry.LevelError,
					Message: fmt.Sprintf("HTTP %d on %s %s", ww.statusCode, r.Method, r.URL.Path),
				})
			}
		}()

		next.ServeHTTP(ww, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
