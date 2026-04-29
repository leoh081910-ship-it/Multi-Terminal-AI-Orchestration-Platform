package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
)

// ReviewWorker implements PRD-DA-001 §10.2: code review gate.
// It monitors tasks in review_pending state, creates review tasks,
// and processes review results to either approve or reject.
type ReviewWorker struct {
	server   *Server
	repo     *store.Repository
	logger   zerolog.Logger
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

// NewReviewWorker creates a new ReviewWorker.
func NewReviewWorker(server *Server, repo *store.Repository, logger zerolog.Logger, interval time.Duration) *ReviewWorker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &ReviewWorker{
		server:   server,
		repo:     repo,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the review worker loop.
func (w *ReviewWorker) Start(ctx context.Context) error {
	if w.running {
		return fmt.Errorf("review worker already running")
	}
	w.running = true
	w.logger.Info().Dur("interval", w.interval).Msg("review worker started")
	go w.loop(ctx)
	return nil
}

// Stop halts the review worker.
func (w *ReviewWorker) Stop() error {
	if !w.running {
		return nil
	}
	close(w.stopCh)
	w.running = false
	w.logger.Info().Msg("review worker stopped")
	return nil
}

func (w *ReviewWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.processReviewPendingTasks(ctx); err != nil {
				w.logger.Error().Err(err).Msg("review worker scan failed")
			}
			if err := w.processReviewResults(ctx); err != nil {
				w.logger.Error().Err(err).Msg("review result processing failed")
			}
		}
	}
}

// processReviewPendingTasks finds tasks in review_pending that need review tasks created.
func (w *ReviewWorker) processReviewPendingTasks(ctx context.Context) error {
	tasks, err := w.repo.ListAllTasks(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.State != engine.StateReviewPending {
			continue
		}

		payload := decodeCompatPayload(task.CardJSON)
		// Check if a review task already exists
		reviewTaskID := readString(payload, "last_review_task_id")
		if reviewTaskID != "" {
			active := w.hasActiveReviewTask(ctx, reviewTaskID, tasks)

			if !active {
				// Review task is done or doesn't exist, but parent is still in review_pending.
				// This means processReviewResults hasn't caught up yet, or the review task
				// completed without updating the parent. Auto-approve to unblock.
				w.logger.Warn().
					Str("task_id", task.ID).
					Str("review_task_id", reviewTaskID).
					Msg("parent still in review_pending after review task finished, auto-approving")
				w.approveTask(ctx, task, payload, reviewTaskID)
				continue
			}

			// Review task is still running — check if it's been running too long (stuck)
			reviewTask := w.findTaskByID(reviewTaskID, tasks)
			if reviewTask != nil && w.isReviewTaskStuck(reviewTask) {
				w.logger.Warn().
					Str("task_id", task.ID).
					Str("review_task_id", reviewTaskID).
					Msg("review task stuck for too long, auto-approving parent")
				w.approveTask(ctx, task, payload, reviewTaskID)
			}
			continue
		}

		if err := w.createReviewTask(ctx, task, payload, tasks); err != nil {
			w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to create review task")
		}
	}

	return nil
}

// hasActiveReviewTask checks if a review task is still active.
func (w *ReviewWorker) hasActiveReviewTask(ctx context.Context, reviewTaskID string, allTasks []*ent.Task) bool {
	for _, t := range allTasks {
		if t.ID != reviewTaskID {
			continue
		}
		return t.State != engine.StateDone && t.State != engine.StateVerified && t.State != engine.StateFailed
	}
	return false
}

// findTaskByID finds a task by ID in the given list.
func (w *ReviewWorker) findTaskByID(taskID string, allTasks []*ent.Task) *ent.Task {
	for _, t := range allTasks {
		if t.ID == taskID {
			return t
		}
	}
	return nil
}

// isReviewTaskStuck checks if a review task has been running for too long.
// PR-2: Review tasks running longer than 15 minutes are considered stuck.
func (w *ReviewWorker) isReviewTaskStuck(task *ent.Task) bool {
	if task.State != engine.StateRunning {
		return false
	}
	return time.Since(task.UpdatedAt) > 15*time.Minute
}

// createReviewTask creates a code-review task for a task in review_pending state.
func (w *ReviewWorker) createReviewTask(ctx context.Context, originalTask *ent.Task, originalPayload map[string]interface{}, allTasks []*ent.Task) error {
	rootTaskID := readString(originalPayload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = originalTask.ID
	}

	taskID := "TS-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	projectID := firstCompatNonEmpty(readString(originalPayload, "project_id"), originalTask.ProjectID)

	// Determine review type based on task type
	reviewType := RemediationTypeCodeReview
	taskType := readString(originalPayload, "type")
	title := fmt.Sprintf("[code-review] Review task %s", originalTask.ID)

	// For doc/analysis tasks, use lighter review
	if isDocOrAnalysisTask(taskType) {
		title = fmt.Sprintf("[review] Verify task %s output", originalTask.ID)
	}

	reportPath := compatSystemReportPath(taskID, string(reviewType))
	outputArtifacts := []string{reportPath}
	filesToModify := []string{reportPath}
	inputArtifacts := readStringSlice(originalPayload, "output_artifacts")
	if originalArtifactPath := readString(originalPayload, "artifact_path"); originalArtifactPath != "" {
		inputArtifacts = append(inputArtifacts, originalArtifactPath)
	}

	reviewPayload := map[string]interface{}{
		"task_id":               taskID,
		"id":                    taskID,
		"title":                 title,
		"project_id":            projectID,
		"type":                  string(reviewType),
		"owner_agent":           compatAgentReviewer,
		"status":                "backlog",
		"priority":              1, // highest priority
		"description":           fmt.Sprintf("Review task %s (%s) output for correctness", originalTask.ID, taskType),
		"dispatch_mode":         "auto",
		"auto_dispatch_enabled": true,
		"parent_task_id":        originalTask.ID,
		"root_task_id":          rootTaskID,
		"coordination_stage":    "review",
		"input_artifacts":       inputArtifacts,
		"output_artifacts":      outputArtifacts,
		"files_to_modify":       filesToModify,
		"context": map[string]interface{}{
			"original_task_id":        originalTask.ID,
			"original_type":           taskType,
			"review_type":             string(reviewType),
			"original_artifact_path":  readString(originalPayload, "artifact_path"),
			"original_workspace_path": readString(originalPayload, "workspace_path"),
		},
	}

	card, err := buildCompatTaskCard(reviewPayload, nil, taskID, w.server.defaultCompatProjectID())
	if err != nil {
		return fmt.Errorf("failed to build review task card: %w", err)
	}

	if _, err := w.repo.CreateTask(ctx, card); err != nil {
		return fmt.Errorf("failed to create review task: %w", err)
	}

	// Update original task with review task reference
	originalPayload["last_review_task_id"] = taskID
	originalPayload["coordination_stage"] = "under_review"

	return w.server.persistCompatPayload(ctx, originalTask.ID, originalPayload)
}

// processReviewResults checks completed review tasks and updates originals.
func (w *ReviewWorker) processReviewResults(ctx context.Context) error {
	tasks, err := w.repo.ListAllTasks(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		payload := decodeCompatPayload(task.CardJSON)
		taskType := readString(payload, "type")

		// Only process review tasks (code-review or review type)
		if taskType != string(RemediationTypeCodeReview) && taskType != "review" {
			continue
		}
		// Process completed or failed review tasks
		if task.State != engine.StateDone && task.State != engine.StateVerified && task.State != engine.StateFailed {
			continue
		}

		parentTaskID := readString(payload, "parent_task_id")
		if parentTaskID == "" {
			continue
		}

		// Find parent task
		parentTask, err := w.repo.GetTaskByID(ctx, parentTaskID)
		if err != nil || parentTask == nil {
			continue
		}
		if parentTask.State != engine.StateReviewPending {
			continue
		}

		// Review task completed — determine result
		parentPayload := decodeCompatPayload(parentTask.CardJSON)
		lastReviewTaskID := readString(parentPayload, "last_review_task_id")
		if lastReviewTaskID != task.ID {
			continue // not the latest review
		}

		resultSummary := readString(payload, "result_summary")

		// Determine decision from review task state and payload
		var reviewDecision string
		if task.State == engine.StateFailed {
			// PR-2: Review task failed (reviewer agent errored).
			// Auto-approve parent rather than blocking - execution was fine.
			reviewDecision = "approved"
		} else {
			// Check review_decision field first
			reviewDecision = readString(payload, "review_decision")
			if reviewDecision == "" {
				// Fall back to result_summary heuristic
				// Look for rejection keywords in the review result
				if containsRejectionKeywords(resultSummary) {
					reviewDecision = "rejected"
				} else {
					reviewDecision = "approved"
				}
			}
		}

		if reviewDecision == "approved" {
			w.approveTask(ctx, parentTask, parentPayload, task.ID)
		} else if reviewDecision == "rejected" {
			w.rejectTask(ctx, parentTask, parentPayload, task.ID, resultSummary, tasks)
		}
	}

	return nil
}

// containsRejectionKeywords checks if a review result contains rejection indicators.
// PR-2: Tightened to avoid false positives — "fail", "error", "wrong" in isolation
// don't mean rejection. Only explicit rejection phrases count.
func containsRejectionKeywords(text string) bool {
	text = strings.ToLower(text)
	rejectionPhrases := []string{
		"reject", "rejected", "rejection",
		"deny", "denied", "denial",
		"not approved", "disapprove", "disapproved",
		"needs revision", "needs rework", "requires changes",
		"must be fixed", "must fix", "must be corrected",
		"do not approve", "cannot approve",
		"review decision: rejected",
	}
	for _, phrase := range rejectionPhrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

// approveTask moves a task from review_pending to verified (review passed).
// The merge queue will handle the actual merge process (verified -> merged -> done).
func (w *ReviewWorker) approveTask(ctx context.Context, task *ent.Task, payload map[string]interface{}, reviewTaskID string) {
	if err := w.server.transitionCompatTaskState(ctx, task, engine.StateVerified, "review_approved", ""); err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to approve task after review")
		return
	}

	payload["review_decision"] = "approved"
	payload["coordination_stage"] = "review_approved"
	payload["dispatch_status"] = "completed"
	payload["status"] = "verified"
	payload["last_review_task_id"] = reviewTaskID

	if err := w.server.persistCompatPayload(ctx, task.ID, payload); err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to persist review approval")
	}

	w.logger.Info().Str("task_id", task.ID).Msg("review approved, task moved to verified (merge queue will handle merge)")
}

// rejectTask creates a rework task and moves original back to retry_waiting.
func (w *ReviewWorker) rejectTask(ctx context.Context, task *ent.Task, payload map[string]interface{}, reviewTaskID string, rejectionReason string, allTasks []*ent.Task) {
	// Create rework task
	rootTaskID := readString(payload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = task.ID
	}

	reworkTaskID := "TS-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	projectID := firstCompatNonEmpty(readString(payload, "project_id"), task.ProjectID)

	// Inherit output targets from original task so CLI executor can validate
	outputArtifacts := readStringSlice(payload, "output_artifacts")
	filesToModify := readStringSlice(payload, "files_to_modify")

	reworkPayload := map[string]interface{}{
		"task_id":               reworkTaskID,
		"id":                    reworkTaskID,
		"title":                 fmt.Sprintf("[rework] Fix issues from review of %s", task.ID),
		"project_id":            projectID,
		"type":                  string(RemediationTypeRework),
		"owner_agent":           readString(payload, "owner_agent"),
		"status":                "backlog",
		"priority":              2,
		"description":           fmt.Sprintf("Fix issues found in review: %s", firstCompatNonEmpty(rejectionReason, "review rejected")),
		"dispatch_mode":         "auto",
		"auto_dispatch_enabled": true,
		"parent_task_id":        task.ID,
		"root_task_id":          rootTaskID,
		"coordination_stage":    "rework",
		"output_artifacts":      outputArtifacts,
		"files_to_modify":       filesToModify,
		"context": map[string]interface{}{
			"original_task_id": task.ID,
			"review_task_id":   reviewTaskID,
			"rejection_reason": rejectionReason,
		},
	}

	card, err := buildCompatTaskCard(reworkPayload, nil, reworkTaskID, w.server.defaultCompatProjectID())
	if err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to build rework task card")
		return
	}

	if _, err := w.repo.CreateTask(ctx, card); err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to create rework task")
		return
	}

	// Move original back to retry_waiting
	if err := w.server.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "review_rejected", rejectionReason); err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to move task back to retry_waiting after review rejection")
		return
	}

	payload["review_decision"] = "rejected"
	payload["coordination_stage"] = "rework_created"
	payload["dispatch_status"] = "failed"
	payload["status"] = "ready"
	payload["last_review_task_id"] = reviewTaskID
	payload["auto_repair_count"] = readIntDefault(payload, 0, "auto_repair_count") + 1

	if err := w.server.persistCompatPayload(ctx, task.ID, payload); err != nil {
		w.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to persist review rejection")
	}

	w.logger.Info().
		Str("task_id", task.ID).
		Str("rework_task_id", reworkTaskID).
		Msg("review rejected, rework task created")
}

// isDocOrAnalysisTask determines if a task type should get a lighter review.
func isDocOrAnalysisTask(taskType string) bool {
	switch taskType {
	case "analyze", "collect", "list", "select":
		return true
	default:
		return false
	}
}
