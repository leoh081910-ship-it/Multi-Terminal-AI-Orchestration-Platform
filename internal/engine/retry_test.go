package engine

import (
	"testing"
	"time"
)

func TestRetryManager(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:        2,
		BackoffIntervals:  []time.Duration{30 * time.Second, 60 * time.Second},
		MaxLoopIterations: 50,
	}

	rm := NewRetryManager(config)

	t.Run("ShouldRetry returns true for valid retry", func(t *testing.T) {
		state := &RetryState{RetryCount: 0}
		reason := "execution_failure"

		if !rm.ShouldRetry(state, reason) {
			t.Errorf("ShouldRetry should return true for %s with retry_count=0", reason)
		}
	})

	t.Run("ShouldRetry returns false when max retries exceeded", func(t *testing.T) {
		state := &RetryState{RetryCount: 2} // MaxRetries is 2
		reason := "execution_failure"

		if rm.ShouldRetry(state, reason) {
			t.Errorf("ShouldRetry should return false when max retries exceeded")
		}
	})

	t.Run("ShouldRetry always returns true for non-consuming reasons", func(t *testing.T) {
		state := &RetryState{RetryCount: 10} // Exceeds max
		reason := "dependency_failed"        // Does not consume retry

		if !rm.ShouldRetry(state, reason) {
			t.Errorf("ShouldRetry should return true for non-consuming reason even with high retry count")
		}
	})
}

func TestCalculateBackoffDuration(t *testing.T) {
	config := &RetryConfig{
		BackoffIntervals: []time.Duration{30 * time.Second, 60 * time.Second},
	}

	rm := NewRetryManager(config)

	tests := []struct {
		retryCount int
		expected   time.Duration
	}{
		{0, 30 * time.Second},
		{1, 60 * time.Second},
		{2, 60 * time.Second}, // Uses last interval when exceeded
		{10, 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.expected.String(), func(t *testing.T) {
			duration := rm.CalculateBackoffDuration(tt.retryCount)
			if duration != tt.expected {
				t.Errorf("CalculateBackoffDuration(%d) = %v, want %v",
					tt.retryCount, duration, tt.expected)
			}
		})
	}
}

func TestIsBackoffElapsed(t *testing.T) {
	config := &RetryConfig{
		BackoffIntervals: []time.Duration{100 * time.Millisecond},
	}

	rm := NewRetryManager(config)

	t.Run("returns true when backoff elapsed", func(t *testing.T) {
		since := time.Now().Add(-200 * time.Millisecond)
		if !rm.IsBackoffElapsed(0, since) {
			t.Error("IsBackoffElapsed should return true when enough time passed")
		}
	})

	t.Run("returns false when backoff not elapsed", func(t *testing.T) {
		since := time.Now()
		if rm.IsBackoffElapsed(0, since) {
			t.Error("IsBackoffElapsed should return false when not enough time passed")
		}
	})
}

func TestTTLManager(t *testing.T) {
	config := &TTLConfig{
		DefaultTTL: 72 * time.Hour,
	}

	tm := NewTTLManager(config)

	t.Run("IsExpired returns true for old terminal timestamp", func(t *testing.T) {
		terminalAt := time.Now().Add(-73 * time.Hour)
		if !tm.IsExpired(terminalAt) {
			t.Error("IsExpired should return true for old terminal timestamp")
		}
	})

	t.Run("IsExpired returns false for recent terminal timestamp", func(t *testing.T) {
		terminalAt := time.Now().Add(-1 * time.Hour)
		if tm.IsExpired(terminalAt) {
			t.Error("IsExpired should return false for recent terminal timestamp")
		}
	})

	t.Run("IsExpired returns false for zero time", func(t *testing.T) {
		if tm.IsExpired(time.Time{}) {
			t.Error("IsExpired should return false for zero time")
		}
	})

	t.Run("GetRemainingTTL returns correct duration", func(t *testing.T) {
		terminalAt := time.Now().Add(-1 * time.Hour)
		remaining := tm.GetRemainingTTL(terminalAt)

		expected := 71 * time.Hour
		if remaining < expected-1*time.Hour || remaining > expected+1*time.Hour {
			t.Errorf("GetRemainingTTL returned unexpected value: %v", remaining)
		}
	})
}
