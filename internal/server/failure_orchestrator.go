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

// FailureOrchestrator implements PRD-DA-001 §10: automatic failure triage and remediation.
// It runs as a background worker that processes tasks in the `triage` state.
type FailureOrchestrator struct {
	server   *Server
	repo     *store.Repository
	policy   *FailurePolicy
	logger   zerolog.Logger
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

// NewFailureOrchestrator creates a new FailureOrchestrator.
func NewFailureOrchestrator(server *Server, repo *store.Repository, logger zerolog.Logger, interval time.Duration) *FailureOrchestrator {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &FailureOrchestrator{
		server:   server,
		repo:     repo,
		policy:   NewFailurePolicy(repo),
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the orchestrator loop.
func (o *FailureOrchestrator) Start(ctx context.Context) error {
	if o.running {
		return fmt.Errorf("failure orchestrator already running")
	}
	o.running = true
	o.logger.Info().Dur("interval", o.interval).Msg("failure orchestrator started")
	go o.loop(ctx)
	return nil
}

// Stop halts the orchestrator.
func (o *FailureOrchestrator) Stop() error {
	if !o.running {
		return nil
	}
	close(o.stopCh)
	o.running = false
	o.logger.Info().Msg("failure orchestrator stopped")
	return nil
}

func (o *FailureOrchestrator) loop(ctx context.Context) {
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			if err := o.processTriageTasks(ctx); err != nil {
				o.logger.Error().Err(err).Msg("failure orchestrator scan failed")
			}
		}
	}
}

// processTriageTasks finds all tasks in triage state and creates remediation tasks.
func (o *FailureOrchestrator) processTriageTasks(ctx context.Context) error {
	tasks, err := o.repo.ListAllTasks(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.State != engine.StateTriage {
			continue
		}

		if err := o.processTriageTask(ctx, task); err != nil {
			o.logger.Error().Err(err).Str("task_id", task.ID).Msg("failed to process triage task")
		}
	}

	return nil
}

// processTriageTask handles a single task in triage state.
// PRD §5: triage → classify → create remediation → retry_waiting
func (o *FailureOrchestrator) processTriageTask(ctx context.Context, task *ent.Task) error {
	payload := decodeCompatPayload(task.CardJSON)
	rootTaskID := readString(payload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = task.ID
	}
	failureCode := FailureCode(readString(payload, "failure_code"))
	failureSignature := readString(payload, "failure_signature")
	reason := firstCompatNonEmpty(readString(payload, "last_dispatch_error"), readString(payload, "last_error_reason"))

	o.logger.Info().
		Str("task_id", task.ID).
		Str("failure_code", string(failureCode)).
		Str("root_task_id", rootTaskID).
		Msg("processing triage task")

	// Stop-loss check
	shouldStop, stopReason := o.policy.ShouldStopLoss(ctx, task)
	if shouldStop {
		o.logger.Warn().
			Str("task_id", task.ID).
			Str("stop_reason", stopReason).
			Msg("stop-loss triggered, moving to blocked")

		if err := o.transitionToBlocked(ctx, task, stopReason); err != nil {
			return err
		}
		return nil
	}

	// Signature dedup check
	isDup, err := o.policy.CheckSignatureDuplication(ctx, rootTaskID, failureSignature, task.ID)
	if err != nil {
		return fmt.Errorf("signature dedup check failed: %w", err)
	}
	if isDup {
		o.logger.Warn().
			Str("task_id", task.ID).
			Str("signature", failureSignature).
			Msg("duplicate failure signature, skipping remediation")

		// Still transition to retry_waiting so the retry worker can pick it up
		return o.transitionToRetryWaiting(ctx, task, payload)
	}

	// Classify and create remediation task
	failure := ClassifyError(reason)
	remediationTaskID, err := o.createRemediationTask(ctx, task, failure, payload)
	if err != nil {
		return fmt.Errorf("failed to create remediation task: %w", err)
	}

	// Update original task with remediation info
	payload["coordination_stage"] = "remediation_created"
	payload["auto_repair_count"] = readIntDefault(payload, 0, "auto_repair_count") + 1
	payload["remediation_task_id"] = remediationTaskID

	// Transition original task to retry_waiting
	if err := o.transitionToRetryWaiting(ctx, task, payload); err != nil {
		return err
	}

	o.logger.Info().
		Str("task_id", task.ID).
		Str("remediation_task_id", remediationTaskID).
		Str("remediation_type", string(failure.RemediationType)).
		Msg("remediation task created, original moved to retry_waiting")

	return nil
}

// createRemediationTask creates a new system task to handle the failure.
func (o *FailureOrchestrator) createRemediationTask(ctx context.Context, originalTask *ent.Task, failure StructuredFailure, originalPayload map[string]interface{}) (string, error) {
	rootTaskID := readString(originalPayload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = originalTask.ID
	}

	taskID := "TS-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	projectID := firstCompatNonEmpty(readString(originalPayload, "project_id"), originalTask.ProjectID)

	reason := firstCompatNonEmpty(
		readString(originalPayload, "last_dispatch_error"),
		readString(originalPayload, "last_error_reason"),
	)

	title := fmt.Sprintf("[%s] %s", failure.RemediationType, truncateForTitle(reason, 60))

	outputArtifacts, filesToModify := remediationTaskOutputs(taskID, failure.RemediationType, originalPayload)
	inputArtifacts := readStringSlice(originalPayload, "output_artifacts")
	if originalArtifactPath := readString(originalPayload, "artifact_path"); originalArtifactPath != "" {
		inputArtifacts = append(inputArtifacts, originalArtifactPath)
	}

	remediationPayload := map[string]interface{}{
		"task_id":               taskID,
		"id":                    taskID,
		"title":                 title,
		"project_id":            projectID,
		"type":                  string(failure.RemediationType),
		"owner_agent":           failure.RemediationOwner,
		"status":                "backlog",
		"priority":              2, // higher priority than normal tasks
		"description":           fmt.Sprintf("Auto-generated remediation for task %s: %s", originalTask.ID, failure.SuggestedAction),
		"dispatch_mode":         "auto",
		"auto_dispatch_enabled": true,
		"parent_task_id":        originalTask.ID,
		"root_task_id":          rootTaskID,
		"derived_from_failure":  string(failure.FailureCode),
		"coordination_stage":    "remediation",
		"failure_code":          string(failure.FailureCode),
		"failure_signature":     readString(originalPayload, "failure_signature"),
		"input_artifacts":       inputArtifacts,
		"output_artifacts":      outputArtifacts,
		"files_to_modify":       filesToModify,
		"context": map[string]interface{}{
			"original_task_id":        originalTask.ID,
			"root_task_id":            rootTaskID,
			"failure_code":            string(failure.FailureCode),
			"error_message":           reason,
			"suggested_action":        failure.SuggestedAction,
			"original_artifact_path":  readString(originalPayload, "artifact_path"),
			"original_workspace_path": readString(originalPayload, "workspace_path"),
		},
	}

	card, err := buildCompatTaskCard(remediationPayload, nil, taskID, o.server.defaultCompatProjectID())
	if err != nil {
		return "", fmt.Errorf("failed to build remediation task card: %w", err)
	}

	if _, err := o.repo.CreateTask(ctx, card); err != nil {
		return "", fmt.Errorf("failed to create remediation task: %w", err)
	}

	return taskID, nil
}

// transitionToRetryWaiting moves a task from triage to retry_waiting.
func (o *FailureOrchestrator) transitionToRetryWaiting(ctx context.Context, task *ent.Task, payload map[string]interface{}) error {
	if err := o.server.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "triage_complete", ""); err != nil {
		return fmt.Errorf("failed to transition task to retry_waiting: %w", err)
	}

	payload["dispatch_status"] = "failed"
	payload["status"] = "ready"
	payload["execution_session_id"] = nil
	return o.server.persistCompatPayload(ctx, task.ID, payload)
}

// transitionToBlocked moves a task to failed (stop-loss triggered).
func (o *FailureOrchestrator) transitionToBlocked(ctx context.Context, task *ent.Task, reason string) error {
	payload := decodeCompatPayload(task.CardJSON)

	// Transition triage → failed (via retry_waiting → failed is not in valid transitions, so go directly)
	if err := o.server.transitionCompatTaskState(ctx, task, engine.StateFailed, "stop_loss", reason); err != nil {
		// If direct transition fails, try via valid path
		if err2 := o.server.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "stop_loss", reason); err2 != nil {
			return fmt.Errorf("failed to transition task to failed: %w (retry path: %v)", err, err2)
		}
	}

	payload["coordination_stage"] = "stopped"
	payload["dispatch_status"] = "failed"
	payload["status"] = "blocked"
	payload["execution_session_id"] = nil
	return o.server.persistCompatPayload(ctx, task.ID, payload)
}

func truncateForTitle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func remediationTaskOutputs(taskID string, taskType RemediationTaskType, originalPayload map[string]interface{}) ([]string, []string) {
	switch taskType {
	case RemediationTypeCodeReview, RemediationTypeNoopReview, RemediationTypeTriage:
		reportPath := compatSystemReportPath(taskID, string(taskType))
		return []string{reportPath}, []string{reportPath}
	default:
		outputArtifacts := readStringSlice(originalPayload, "output_artifacts")
		filesToModify := readStringSlice(originalPayload, "files_to_modify")
		if len(filesToModify) == 0 {
			filesToModify = outputArtifacts
		}
		return outputArtifacts, filesToModify
	}
}
