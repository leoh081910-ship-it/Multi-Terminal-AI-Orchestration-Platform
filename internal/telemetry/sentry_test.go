package telemetry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInitSentryEmptyDSN(t *testing.T) {
	err := InitSentry(SentryConfig{DSN: ""}, zerolog.Nop())
	assert.NoError(t, err, "empty DSN should be a no-op")
}

func TestSentryMiddlewarePassesThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	SentryMiddleware(next).ServeHTTP(rec, req)

	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSentryMiddlewareCaptures5xx(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest("GET", "/fail", nil)
	rec := httptest.NewRecorder()

	// Should not panic even without real Sentry
	SentryMiddleware(next).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSentryMiddlewareCapturesPanic(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()

	// SentryMiddleware should recover the panic
	SentryMiddleware(next).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
