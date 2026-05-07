// Package registry provides the runtime Agent registry for v3 multi-agent support.
package registry

import (
	"context"
	"time"
)

// HeartbeatLoop runs a background goroutine that periodically checks agent liveness
// via Runner.HealthCheck and updates agent status accordingly.
//
// It is started via Registry.StartHeartbeat and stopped via Registry.StopHeartbeat.
// The loop is non-blocking: slow HealthCheck calls do not block the ticker.
func (r *Registry) StartHeartbeat(ctx context.Context) {
	r.mu.Lock()
	interval := r.heartbeatInterval
	r.mu.Unlock()

	go r.heartbeatLoop(ctx, interval)
}

// heartbeatLoop periodically calls HealthCheck on all registered runners.
func (r *Registry) heartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info().Msg("heartbeat loop stopped")
			return
		case <-ticker.C:
			r.runHealthChecks(ctx)
		}
	}
}

// runHealthChecks checks all registered agents and updates their status.
func (r *Registry) runHealthChecks(ctx context.Context) {
	entries := r.ListEntries()
	for _, entry := range entries {
		agentID := entry.Runner.String()
		newStatus := StatusIdle
		if err := entry.Runner.HealthCheck(ctx); err != nil {
			newStatus = StatusError
			r.logger.Warn().Err(err).Str("agent_id", agentID).Msg("agent health check failed")
		}
		if entry.Status != newStatus {
			r.SetStatus(agentID, newStatus)
		}
	}
}

// SetRunning marks an agent as actively executing a task.
// Call with StatusRunning during execution start, and StatusIdle after completion.
func (r *Registry) SetRunning(agentID string, running bool) {
	if running {
		r.SetStatus(agentID, StatusRunning)
	} else {
		r.SetStatus(agentID, StatusIdle)
	}
}

// RunnerRegistry is an alias for Registry used in dispatch code that
// prefers the more descriptive name. Both names refer to the same type.
type RunnerRegistry = Registry