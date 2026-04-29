package server

import (
	"context"
	"fmt"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

// RetryWorker implements PRD-DA-001 §10.3: automatic retry after remediation.
// It monitors tasks in retry_waiting and auto-dispatches when:
// 1. Backoff period has elapsed
// 2. All child remediation tasks are completed
// 3. Retry limit is not exceeded
type RetryWorker struct {
	server   *Server
	repo     *store.Repository
	policy   *FailurePolicy
	logger   zerolog.Logger
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

// NewRetryWorker creates a new RetryWorker.
func NewRetryWorker(server *Server, repo *store.Repository, logger zerolog.Logger, interval time.Duration) *RetryWorker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &RetryWorker{
		server:   server,
		repo:     repo,
		policy:   NewFailurePolicy(repo),
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the retry worker loop.
func (w *RetryWorker) Start(ctx context.Context) error {
	if w.running {
		return fmt.Errorf("retry worker already running")
	}
	w.running = true
	w.logger.Info().Dur("interval", w.interval).Msg("retry worker started")
	go w.loop(ctx)
	return nil
}

// Stop halts the retry worker.
func (w *RetryWorker) Stop() error {
	if !w.running {
		return nil
	}
	close(w.stopCh)
	w.running = false
	w.logger.Info().Msg("retry worker stopped")
	return nil
}

func (w *RetryWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.processRetryWaitingTasks(ctx); err != nil {
				w.logger.Error().Err(err).Msg("retry worker scan failed")
			}
		}
	}
}

// processRetryWaitingTasks finds tasks in retry_waiting that are ready for retry.
func (w *RetryWorker) processRetryWaitingTasks(ctx context.Context) error {
	tasks, err := w.repo.ListAllTasks(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.State != engine.StateRetryWaiting {
			continue
		}

		if err := w.processRetryTask(ctx, task, tasks); err != nil {
			w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to process retry task")
		}
	}

	return nil
}

// processRetryTask handles a single task in retry_waiting state.
func (w *RetryWorker) processRetryTask(ctx context.Context, task *ent.Task, allTasks []*ent.Task) error {
	payload := decodeCompatPayload(task.CardJSON)

	// Skip system-generated tasks (remediation, review) — they don't auto-retry
	taskType := readString(payload, "type")
	if IsSystemTaskType(taskType) {
		return nil
	}

	// Check if task has parent (it's a derived task, not an original)
	parentTaskID := readString(payload, "parent_task_id")
	if parentTaskID != "" {
		return nil // derived tasks don't auto-retry through this path
	}

	// PRD §10.3: retry limit check
	if !w.policy.CheckRetryLimit(task.RetryCount) {
		w.logger.Warn().
			Str("task_id", task.ID).
			Int("retry_count", task.RetryCount).
			Msg("retry limit exceeded, moving to blocked")

		return w.moveToBlocked(ctx, task, payload, fmt.Sprintf("auto-retry limit reached (%d)", task.RetryCount))
	}

	// Check if all child remediation tasks are done
	remediationComplete, err := w.checkRemediationCompletion(ctx, task.ID, allTasks)
	if err != nil {
		return err
	}
	if !remediationComplete {
		return nil // still waiting for remediation
	}

	// Check backoff — PRD §10.3: backoff starts from retry_waiting event timestamp
	backoffElapsed := w.isBackoffElapsed(task)
	if !backoffElapsed {
		return nil // backoff not yet elapsed
	}

	// All conditions met — auto-dispatch the original task
	projectID := firstCompatNonEmpty(readString(payload, "project_id"), task.ProjectID)
	w.logger.Info().
		Str("task_id", task.ID).
		Int("retry_count", task.RetryCount).
		Msg("auto-retrying task")

	_, err = w.server.dispatchCompatTask(ctx, task.ID, true, projectID)
	if err != nil {
		return fmt.Errorf("failed to auto-dispatch retry task: %w", err)
	}

	return nil
}

// checkRemediationCompletion checks if all child remediation tasks for a root task are done.
// Only done/verified children count as successfully completed.
// A failed child blocks retry — the fix did not succeed.
func (w *RetryWorker) checkRemediationCompletion(ctx context.Context, taskID string, allTasks []*ent.Task) (bool, error) {
	hasChildren := false
	for _, t := range allTasks {
		payload := decodeCompatPayload(t.CardJSON)
		parentID := readString(payload, "parent_task_id")
		if parentID != taskID {
			continue
		}
		hasChildren = true
		// If any child is still active, not ready for retry
		if t.State != engine.StateDone && t.State != engine.StateVerified && t.State != engine.StateFailed &&
			t.State != engine.StateVerifyFailed && t.State != engine.StateApplyFailed {
			return false, nil
		}
		// If a child failed (any failure state), the remediation did not succeed — block retry
		if t.State == engine.StateFailed || t.State == engine.StateVerifyFailed || t.State == engine.StateApplyFailed {
			w.logger.Warn().
				Str("parent_task_id", taskID).
				Str("child_task_id", t.ID).
				Str("child_state", t.State).
				Msg("remediation child failed, blocking parent retry")
			return false, nil
		}
	}

	// If no children, the task might have been put in retry_waiting directly (legacy path)
	// or the remediation might not have been created yet.
	// In either case, we allow retry if backoff has elapsed.
	if !hasChildren {
		return true, nil
	}

	return true, nil
}

// isBackoffElapsed checks if the backoff period has elapsed.
// Uses the engine's backoff intervals: 30s, 60s.
func (w *RetryWorker) isBackoffElapsed(task *ent.Task) bool {
	backoffDurations := []time.Duration{30 * time.Second, 60 * time.Second}
	retryCount := task.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}

	var backoff time.Duration
	if retryCount < len(backoffDurations) {
		backoff = backoffDurations[retryCount]
	} else {
		backoff = backoffDurations[len(backoffDurations)-1]
	}

	// Use updated_at as the start of the backoff period
	elapsed := time.Since(task.UpdatedAt)
	return elapsed >= backoff
}

// moveToBlocked transitions a task to failed/blocked state (stop-loss).
func (w *RetryWorker) moveToBlocked(ctx context.Context, task *ent.Task, payload map[string]interface{}, reason string) error {
	// Try direct transition to failed
	if err := w.server.transitionCompatTaskState(ctx, task, engine.StateFailed, "stop_loss", reason); err != nil {
		return fmt.Errorf("failed to transition task to failed: %w", err)
	}

	payload["coordination_stage"] = "stopped"
	payload["dispatch_status"] = "failed"
	payload["status"] = "blocked"
	payload["last_error_reason"] = reason
	payload["execution_session_id"] = nil
	return w.server.persistCompatPayload(ctx, task.ID, payload)
}
