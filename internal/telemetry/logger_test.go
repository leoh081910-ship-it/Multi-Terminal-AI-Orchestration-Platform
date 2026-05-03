package telemetry

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestID(t *testing.T) {
	id := NewRequestID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 16, "request ID should be 16 chars (truncated UUID)")
}

func TestNewRequestIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewRequestID()
		assert.False(t, ids[id], "duplicate request ID generated")
		ids[id] = true
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	rid := "test-rid-123"
	ctx = WithRequestID(ctx, rid)
	assert.Equal(t, rid, RequestIDFromContext(ctx))
}

func TestRequestIDFromEmptyContext(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, RequestIDFromContext(ctx))
}

func TestLoggerWithContext(t *testing.T) {
	buf := &testWriter{}
	base := zerolog.New(buf)
	ctx := context.Background()
	ctx = WithRequestID(ctx, "abc-123")

	logger := LoggerWithContext(base, ctx)
	logger.Info().Msg("test")

	output := buf.String()
	assert.Contains(t, output, "abc-123")
	assert.Contains(t, output, "test")
}

func TestLoggerWithoutRequestID(t *testing.T) {
	buf := &testWriter{}
	base := zerolog.New(buf)
	ctx := context.Background()

	logger := LoggerWithContext(base, ctx)
	logger.Info().Msg("no-rid")

	output := buf.String()
	assert.Contains(t, output, "no-rid")
	assert.NotContains(t, output, "request_id")
}

func TestContextWithLogger(t *testing.T) {
	buf := &testWriter{}
	logger := zerolog.New(buf)
	ctx := ContextWithLogger(context.Background(), logger)

	retrieved := FromContext(ctx)
	require.NotEqual(t, zerolog.Logger{}, retrieved)
	retrieved.Info().Msg("from-context")

	assert.Contains(t, buf.String(), "from-context")
}

func TestFromContextFallback(t *testing.T) {
	retrieved := FromContext(context.Background())
	assert.Equal(t, zerolog.Logger{}, retrieved, "empty context should return zero logger")
}

// testWriter captures log output as a string
type testWriter struct {
	data []byte
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *testWriter) String() string {
	return string(w.data)
}
