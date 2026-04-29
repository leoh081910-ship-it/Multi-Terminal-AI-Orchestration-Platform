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

// StartupRecovery scans for tasks that were left in active states when the
// previous server instance was stopped. It runs once during boot.
//
// PR-2 of PLAN-OPS-001: running tasks without an active execution session are
// zombies and are corrected to triage or retry_waiting. Other worker-active
// states (triage, retry_waiting, review_pending, verified) are simply logged —
// the existing background workers will re-take them on their next tick.
type StartupRecovery struct {
	server *Server
	repo   *store.Repository
	logger zerolog.Logger
}

// NewStartupRecovery creates a new StartupRecovery instance.
func NewStartupRecovery(server *Server, repo *store.Repository, logger zerolog.Logger) *StartupRecovery {
	return &StartupRecovery{
		server: server,
		repo:   repo,
		logger: logger,
	}
}

// recoverableStates are the states that the recovery scan examines.
var recoverableStates = []string{
	engine.StateRunning,
	engine.StateTriage,
	engine.StateRetryWaiting,
	engine.StateReviewPending,
	engine.StateVerified,
}

// Run performs a single startup recovery scan.
func (r *StartupRecovery) Run(ctx context.Context) error {
	r.logger.Info().Msg("startup recovery: scanning for tasks left in active states")

	recovered := 0
	for _, state := range recoverableStates {
		tasks, err := r.repo.ListTasksByState(ctx, state)
		if err != nil {
			r.logger.Error().Err(err).Str("state", state).Msg("startup recovery: failed to list tasks")
			continue
		}

		if len(tasks) == 0 {
			continue
		}

		r.logger.Info().Str("state", state).Int("count", len(tasks)).Msg("startup recovery: found active tasks")

		for _, t := range tasks {
			if err := r.recoverTask(ctx, t); err != nil {
				r.logger.Error().Err(err).Str("task_id", t.ID).Str("state", t.State).Msg("startup recovery: failed to recover task")
			} else {
				recovered++
			}
		}
	}

	r.logger.Info().Int("recovered", recovered).Msg("startup recovery: scan complete")
	return nil
}

// recoverTask handles a single task found in an active state.
// Running tasks are reclaimed as zombies. Review_pending tasks that have been
// stuck for too long are auto-approved. Other states are simply logged
// because the existing workers (FailureOrchestrator, RetryWorker, ReviewWorker)
// will pick them up on their next tick.
func (r *StartupRecovery) recoverTask(ctx context.Context, t *ent.Task) error {
	switch t.State {
	case engine.StateRunning:
		return r.recoverRunningTask(ctx, t)
	case engine.StateReviewPending:
		return r.recoverReviewPendingTask(ctx, t)
	case engine.StateTriage, engine.StateRetryWaiting, engine.StateVerified:
		r.logger.Info().
			Str("task_id", t.ID).
			Str("state", t.State).
			Msg("startup recovery: task in worker-active state, workers will re-take")
		return nil
	default:
		return nil
	}
}

// recoverRunningTask checks whether a running task still has an active execution
// session.  If the session is gone (server restarted → process died), the task
// is a zombie and is moved to triage with a recovery event.
// PR-OPS-003: Uses heartbeat timing to determine if task was actively executing.
func (r *StartupRecovery) recoverRunningTask(ctx context.Context, t *ent.Task) error {
	payload := decodeCompatPayload(t.CardJSON)
	sessionID := readString(payload, "execution_session_id")
	lastHeartbeat := readString(payload, "last_heartbeat_at")

	// If there is no session ID, the task was never actually dispatched
	// or the session was already cleaned up — treat as zombie.
	if sessionID == "" {
		r.logger.Warn().
			Str("task_id", t.ID).
			Msg("startup recovery: running task with no session, reclaiming as zombie")
		return r.reclaimZombieRunning(ctx, t, payload, "no_active_session")
	}

	// PR-OPS-003: Check heartbeat timing
	// Since server restarted, all executions are dead regardless of heartbeat.
	// But we record heartbeat info for observability.
	heartbeatAge := "unknown"
	if lastHeartbeat != "" {
		heartbeatTime, err := time.Parse(time.RFC3339, lastHeartbeat)
		if err == nil {
			heartbeatAge = time.Since(heartbeatTime).Round(time.Second).String()
		}
	}

	r.logger.Warn().
		Str("task_id", t.ID).
		Str("session_id", sessionID).
		Str("last_heartbeat_age", heartbeatAge).
		Msg("startup recovery: running task with session (process died on restart), reclaiming as zombie")
	return r.reclaimZombieRunning(ctx, t, payload, "server_restart")
}

// reclaimZombieRunning transitions a zombie running task to triage and writes a
// recovery event.
func (r *StartupRecovery) reclaimZombieRunning(ctx context.Context, t *ent.Task, payload map[string]interface{}, trigger string) error {
	reason := "startup_recovery_" + trigger

	// Transition: running → triage
	if err := r.server.transitionCompatTaskState(ctx, t, engine.StateTriage, reason, "zombie running task reclaimed at startup"); err != nil {
		// Fallback: if triage transition fails, try retry_waiting
		if err2 := r.server.transitionCompatTaskState(ctx, t, engine.StateRetryWaiting, reason, "zombie running task reclaimed at startup (fallback)"); err2 != nil {
			return fmt.Errorf("failed to reclaim zombie running task: %w (triage err: %v, retry_waiting err: %v)", err2, err, err2)
		}
	}

	// Update payload
	syncCompatPayloadState(payload, engine.StateTriage)
	payload["execution_session_id"] = nil
	payload["coordination_stage"] = "recovery"
	payload["last_error_reason"] = reason
	if err := r.server.persistCompatPayload(ctx, t.ID, payload); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("startup recovery: failed to update payload after zombie reclaim")
	}

	// Write a recovery event (separate from the state_transition already written
	// by transitionCompatTaskState).
	if err := r.repo.CreateEvent(ctx, &store.EventData{
		EventID:   uuid.NewString(),
		ProjectID: t.ProjectID,
		TaskID:    t.ID,
		EventType: "recovered_running_task",
		FromState: engine.StateRunning,
		ToState:   engine.StateTriage,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		Attempt:   t.RetryCount,
		Transport: t.Transport,
		Details:   "zombie running task reclaimed at server startup",
	}); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("startup recovery: failed to write recovery event")
	}

	r.logger.Info().
		Str("task_id", t.ID).
		Str("trigger", trigger).
		Msg("startup recovery: zombie running task reclaimed to triage")

	return nil
}

// recoverReviewPendingTask handles tasks stuck in review_pending at startup.
// PR-3: If a task has been in review_pending for more than 30 minutes,
// the review is likely stuck (review task hung or missing). Auto-approve.
func (r *StartupRecovery) recoverReviewPendingTask(ctx context.Context, t *ent.Task) error {
	// Only auto-approve if stuck for more than 30 minutes
	if time.Since(t.UpdatedAt) < 30*time.Minute {
		r.logger.Info().
			Str("task_id", t.ID).
			Str("state", t.State).
			Msg("startup recovery: review_pending task recently updated, workers will re-take")
		return nil
	}

	r.logger.Warn().
		Str("task_id", t.ID).
		Str("state", t.State).
		Dur("stuck_duration", time.Since(t.UpdatedAt)).
		Msg("startup recovery: review_pending task stuck too long, auto-approving")

	// Transition review_pending -> verified
	if err := r.server.transitionCompatTaskState(ctx, t, engine.StateVerified, "startup_recovery_review_stuck", "auto-approved after 30min stuck in review_pending"); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("startup recovery: failed to auto-approve stuck review task")
		return err
	}

	payload := decodeCompatPayload(t.CardJSON)
	syncCompatPayloadState(payload, engine.StateVerified)
	payload["review_decision"] = "approved"
	payload["coordination_stage"] = "recovery_approved"
	payload["dispatch_status"] = "completed"
	payload["status"] = "verified"
	payload["execution_session_id"] = nil
	if err := r.server.persistCompatPayload(ctx, t.ID, payload); err != nil {
		r.logger.Error().Err(err).Str("task_id", t.ID).Msg("startup recovery: failed to persist auto-approved review task")
	}

	return nil
}
