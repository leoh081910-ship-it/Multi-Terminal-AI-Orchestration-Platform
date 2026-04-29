// Package engine provides retry logic, backoff, and TTL management.
package engine

import (
	"fmt"
	"time"
)

// RetryConfig holds retry configuration parameters.
type RetryConfig struct {
	MaxRetries        int             `json:"max_retries"`
	BackoffIntervals  []time.Duration `json:"backoff_intervals"`
	MaxLoopIterations int             `json:"max_loop_iterations"`
}

// DefaultRetryConfig returns the default retry configuration per PRD.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        2,
		BackoffIntervals:  []time.Duration{30 * time.Second, 60 * time.Second},
		MaxLoopIterations: 50, // For reverse tasks
	}
}

// RetryState tracks the current retry state for a task.
type RetryState struct {
	RetryCount         int       `json:"retry_count"`
	LoopIterationCount int       `json:"loop_iteration_count"`
	LastRetryAt        time.Time `json:"last_retry_at"`
	LastRetryReason    string    `json:"last_retry_reason"`
}

// RetryManager handles retry logic and backoff calculations.
type RetryManager struct {
	config *RetryConfig
}

// NewRetryManager creates a new retry manager with the given config.
func NewRetryManager(config *RetryConfig) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryManager{config: config}
}

// ShouldRetry determines if a task should be retried based on its state and failure reason.
// REQUIRES (RETR-01): retry_count tracks failures across all active failures.
// CONSUMES retry_count for: execution_failure, workspace_write_failed, empty_artifact_match,
// deterministic_check_failed, test_command_failed, reverse_loop_exhausted, reverse_env_unavailable.
// DOES NOT CONSUME for: process_resume, dependency_failed.
func (rm *RetryManager) ShouldRetry(state *RetryState, reason string) bool {
	// Check if this reason consumes a retry
	if !rm.consumesRetry(reason) {
		return true // Always retry for non-consuming reasons
	}

	// Check if we've exceeded max retries
	if state.RetryCount >= rm.config.MaxRetries {
		return false
	}

	return true
}

// consumesRetry returns true if the given reason should increment retry_count.
func (rm *RetryManager) consumesRetry(reason string) bool {
	consumingReasons := map[string]bool{
		"execution_failure":          true,
		"workspace_write_failed":     true,
		"empty_artifact_match":       true,
		"deterministic_check_failed": true,
		"test_command_failed":        true,
		"reverse_loop_exhausted":     true,
		"reverse_env_unavailable":    true,
	}

	return consumingReasons[reason]
}

// CalculateBackoffDuration calculates the backoff duration for the next retry.
// REQUIRES (RETR-02): Default backoff is 30s, 60s.
// Backoff starts from retry_waiting event timestamp.
func (rm *RetryManager) CalculateBackoffDuration(retryCount int) time.Duration {
	if retryCount < len(rm.config.BackoffIntervals) {
		return rm.config.BackoffIntervals[retryCount]
	}
	// Use last interval if exceeded configured intervals
	return rm.config.BackoffIntervals[len(rm.config.BackoffIntervals)-1]
}

// IsBackoffElapsed checks if the backoff period has elapsed since the given timestamp.
func (rm *RetryManager) IsBackoffElapsed(retryCount int, since time.Time) bool {
	backoff := rm.CalculateBackoffDuration(retryCount)
	return time.Since(since) >= backoff
}

// CheckLoopIterations checks if the loop iteration count has exceeded the maximum.
// For reverse tasks (REVR), returns error if max_loop_iterations exceeded.
func (rm *RetryManager) CheckLoopIterations(loopIterationCount int) error {
	if loopIterationCount >= rm.config.MaxLoopIterations {
		return &MaxLoopIterationsError{
			MaxIterations: rm.config.MaxLoopIterations,
			Current:       loopIterationCount,
		}
	}
	return nil
}

// MaxLoopIterationsError is returned when the maximum loop iterations is exceeded.
type MaxLoopIterationsError struct {
	MaxIterations int
	Current       int
}

func (e *MaxLoopIterationsError) Error() string {
	return fmt.Sprintf("maximum loop iterations exceeded: %d of %d", e.Current, e.MaxIterations)
}

// TTLConfig holds TTL (time-to-live) configuration.
type TTLConfig struct {
	DefaultTTL time.Duration `json:"default_ttl"`
}

// DefaultTTLConfig returns the default TTL configuration per PRD.
func DefaultTTLConfig() *TTLConfig {
	return &TTLConfig{
		DefaultTTL: 72 * time.Hour, // Default 72 hours
	}
}

// TTLManager handles TTL (time-to-live) management for tasks.
type TTLManager struct {
	config *TTLConfig
}

// NewTTLManager creates a new TTL manager with the given config.
func NewTTLManager(config *TTLConfig) *TTLManager {
	if config == nil {
		config = DefaultTTLConfig()
	}
	return &TTLManager{config: config}
}

// IsExpired checks if a task has expired based on its terminal_at timestamp.
// REQUIRES (RETR-07): TTL starts from terminal_at, not updated_at.
func (tm *TTLManager) IsExpired(terminalAt time.Time) bool {
	if terminalAt.IsZero() {
		return false
	}
	return time.Since(terminalAt) >= tm.config.DefaultTTL
}

// GetRemainingTTL returns the remaining TTL for a task.
func (tm *TTLManager) GetRemainingTTL(terminalAt time.Time) time.Duration {
	if terminalAt.IsZero() {
		return tm.config.DefaultTTL
	}

	elapsed := time.Since(terminalAt)
	if elapsed >= tm.config.DefaultTTL {
		return 0
	}

	return tm.config.DefaultTTL - elapsed
}

// NonRetryableReasons that should not trigger a retry.
var NonRetryableReasons = map[string]bool{
	"dependency_failed":   true,
	"wave_already_sealed": true,
	"invalid_dependency":  true,
	"process_resume":      true,
}
