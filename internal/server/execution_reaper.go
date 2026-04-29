package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

// ExecutionReaper periodically scans for zombie running tasks — tasks that are
// marked as running but have no active execution process. This handles the case
// where a task was left in running state due to an ungraceful shutdown that the
// startup recovery did not catch (e.g. the server ran for a while after boot
// but the execution process died mid-run).
//
// PR-2 of PLAN-OPS-001.
type ExecutionReaper struct {
	server   *Server
	repo     *store.Repository
	logger   zerolog.Logger
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

// ExecutionReaperConfig holds configuration for the reaper.
type ExecutionReaperConfig struct {
	Interval time.Duration
}

// NewExecutionReaper creates a new ExecutionReaper.
func NewExecutionReaper(server *Server, repo *store.Repository, logger zerolog.Logger, cfg ExecutionReaperConfig) *ExecutionReaper {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &ExecutionReaper{
		server:   server,
		repo:     repo,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the reaper loop.
func (r *ExecutionReaper) Start(ctx context.Context) error {
	if r.running {
		return fmt.Errorf("execution reaper already running")
	}
	r.running = true
	r.logger.Info().Dur("interval", r.interval).Msg("execution reaper started")
	go r.loop(ctx)
	return nil
}

// Stop halts the reaper.
func (r *ExecutionReaper) Stop() error {
	if !r.running {
		return nil
	}
	close(r.stopCh)
	r.running = false
	r.logger.Info().Msg("execution reaper stopped")
	return nil
}

func (r *ExecutionReaper) loop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			if err := r.scanAndReap(ctx); err != nil {
				r.logger.Error().Err(err).Msg("execution reaper scan failed")
			}
		}
	}
}

// scanAndReap finds running tasks with stale sessions and reclaims them.
func (r *ExecutionReaper) scanAndReap(ctx context.Context) error {
	tasks, err := r.repo.ListTasksByState(ctx, engine.StateRunning)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return nil
	}

	for _, t := range tasks {
		if r.isZombie(t) {
			if err := r.reclaim(ctx, t); err != nil {
				r.logger.Error().Err(err).Str("task_id", t.ID).Msg("execution reaper: failed to reclaim zombie task")
			}
		}
	}

	return nil
}

// isZombie determines if a running task is a zombie (no active execution).
// PR-OPS-003: Uses heartbeat timing and timeout_at for liveness detection.
// A task is considered zombie if:
// - It has no execution_session_id in its payload
// - It has exceeded its timeout_at deadline (hard timeout)
// - It has no heartbeat recorded (legacy data) and is stale
// - Its heartbeat has expired (no update for more than 5 minutes)
func (r *ExecutionReaper) isZombie(t *ent.Task) bool {
	payload := decodeCompatPayload(t.CardJSON)
	sessionID := readString(payload, "execution_session_id")
	lastHeartbeat := readString(payload, "last_heartbeat_at")
	timeoutAt := readString(payload, "timeout_at")

	// No session means the task was never properly dispatched or session was
	// already cleared — definitely a zombie.
	if sessionID == "" {
		return true
	}

	// PR-OPS-003: Hard timeout enforcement
	// If timeout_at is set and has passed, task is zombie regardless of heartbeat
	if timeoutAt != "" {
		timeoutTime, err := time.Parse(time.RFC3339, timeoutAt)
		if err == nil && time.Now().UTC().After(timeoutTime) {
			r.logger.Info().
				Str("task_id", t.ID).
				Str("timeout_at", timeoutAt).
				Msg("execution reaper: task exceeded timeout_at")
			return true
		}
	}

	// If no heartbeat recorded, fall back to updated_at staleness check
	// (legacy data before heartbeat implementation)
	if lastHeartbeat == "" {
		staleThreshold := 10 * time.Minute
		return time.Since(t.UpdatedAt) > staleThreshold
	}

	// Parse heartbeat timestamp
	heartbeatTime, err := time.Parse(time.RFC3339, lastHeartbeat)
	if err != nil {
		// Invalid heartbeat format, treat as stale
		return true
	}

	// Heartbeat expiration threshold: 5 minutes without update
	// This is shorter than the stale threshold because heartbeat
	// should be updating every 30 seconds during active execution.
	heartbeatThreshold := 5 * time.Minute
	return time.Since(heartbeatTime) > heartbeatThreshold
}

// reclaim transitions a zombie running task to triage.
func (r *ExecutionReaper) reclaim(ctx context.Context, t *ent.Task) error {
	payload := decodeCompatPayload(t.CardJSON)
	sessionID := readString(payload, "execution_session_id")
	timeoutAt := readString(payload, "timeout_at")
	lastHeartbeat := readString(payload, "last_heartbeat_at")

	// Determine reason based on what triggered the zombie detection
	reason := "execution_stalled"
	details := "execution reaper detected zombie running task"
	if timeoutAt != "" {
		timeoutTime, err := time.Parse(time.RFC3339, timeoutAt)
		if err == nil && time.Now().UTC().After(timeoutTime) {
			reason = "execution_timeout"
			details = fmt.Sprintf("execution exceeded timeout_at (%s)", timeoutAt)
		}
	}

	r.logger.Warn().
		Str("task_id", t.ID).
		Str("session_id", sessionID).
		Str("timeout_at", timeoutAt).
		Str("last_heartbeat_at", lastHeartbeat).
		Time("last_updated", t.UpdatedAt).
		Str("reason", reason).
		Msg("execution reaper: reclaiming zombie running task")

	// Transition: running → triage
	if err := r.server.transitionCompatTaskState(ctx, t, engine.StateTriage, reason, details); err != nil {
		// Fallback: try retry_waiting
		if err2 := r.server.transitionCompatTaskState(ctx, t, engine.StateRetryWaiting, reason, details+" (fallback)"); err2 != nil {
			return fmt.Errorf("failed to reclaim zombie: triage err=%v, retry_waiting err=%v", err, err2)
		}
	}

	// Update payload
	syncCompatPayloadState(payload, engine.StateTriage)
	payload["execution_session_id"] = nil
	payload["coordination_stage"] = "reaper_reclaim"
	payload["last_error_reason"] = reason
	if err := r.server.persistCompatPayload(ctx, t.ID, payload); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("execution reaper: failed to update payload after reclaim")
	}

	// Write event with appropriate type
	eventType := "execution_stalled"
	if reason == "execution_timeout" {
		eventType = "execution_timeout"
	}
	if err := r.repo.CreateEvent(ctx, &store.EventData{
		EventID:   uuid.NewString(),
		ProjectID: t.ProjectID,
		TaskID:    t.ID,
		EventType: eventType,
		FromState: engine.StateRunning,
		ToState:   engine.StateTriage,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		Attempt:   t.RetryCount,
		Transport: t.Transport,
		Details:   details,
	}); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("execution reaper: failed to write event")
	}

	r.logger.Info().
		Str("task_id", t.ID).
		Str("reason", reason).
		Msg("execution reaper: zombie task reclaimed to triage")

	return nil
}
