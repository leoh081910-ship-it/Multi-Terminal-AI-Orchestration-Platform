package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
)

var errCompatTaskNotFound = errors.New("task not found")

func (s *Server) dispatchCompatTask(ctx context.Context, taskID string, isRetry bool, requestedProjectID string) (*ent.Task, error) {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errCompatTaskNotFound
	}
	if requestedProjectID != "" && !s.compatTaskBelongsToProject(task, requestedProjectID) {
		return nil, errCompatTaskNotFound
	}

	if isRetry && !engine.CanRetry(task.State) && task.State != engine.StateRetryWaiting {
		return nil, fmt.Errorf("task state does not allow retry")
	}

	if isRetry && engine.CanRetry(task.State) {
		if err := s.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, "manual_retry", ""); err != nil {
			return nil, err
		}
		task, err = s.repo.GetTaskByID(ctx, taskID)
		if err != nil {
			return nil, err
		}
	}

	payload := mergeCompatPayload(decodeCompatPayload(task.CardJSON), map[string]interface{}{})
	if err := validateCompatDispatchEligibility(task, payload, isRetry); err != nil {
		return nil, err
	}
	view := s.mapTaskView(task)
	ownerAgent := normalizeCompatAgent(firstCompatNonEmpty(readString(payload, "owner_agent"), view.Transport))
	projectID := normalizeCompatProjectID(firstCompatNonEmpty(readString(payload, "project_id"), requestedProjectID, s.defaultCompatProjectID()))
	executionManager := s.compatExecutionForProject(projectID)

	runtimeSpec := compatRuntimeSpec{Name: inferRuntimeFromAgent(ownerAgent, "")}
	if executionManager != nil {
		runtimeSpec = executionManager.runtimeForAgent(ownerAgent)
	}

	command := firstCompatNonEmpty(readString(payload, "command"), runtimeSpec.Command)
	shell := firstCompatNonEmpty(readString(payload, "shell"), runtimeSpec.Shell)
	transportType := firstCompatNonEmpty(readString(payload, "transport"), view.Transport, string(transport.TransportCLI))
	workspacePath := firstCompatNonEmpty(readString(payload, "workspace_path"), view.WorkspacePath)
	artifactPath := firstCompatNonEmpty(readString(payload, "artifact_path"), view.ArtifactPath)
	if executionManager != nil {
		if workspacePath == "" {
			workspacePath = executionManager.workspacePath(taskID, transportType)
		}
		if artifactPath == "" {
			artifactPath = executionManager.artifactPath(taskID)
		}
	}

	filesToModify := compatStringSliceField(payload, "files_to_modify")
	if len(filesToModify) == 0 {
		filesToModify = compatStringSliceField(payload, "output_artifacts")
	}
	payload["files_to_modify"] = filesToModify

	if err := s.prepareCompatTaskForExecution(ctx, task); err != nil {
		return nil, err
	}

	task, err = s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	view = s.mapTaskView(task)

	attempts := readIntDefault(payload, view.RetryCount, "dispatch_attempts") + 1
	now := time.Now().UTC()
	sessionID := "SE-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:10]

	payload["owner_agent"] = ownerAgent
	payload["dispatch_attempts"] = attempts
	payload["dispatch_status"] = "running"
	payload["execution_runtime"] = firstCompatNonEmpty(runtimeSpec.Name, inferRuntimeFromAgent(ownerAgent, ""))
	payload["execution_session_id"] = sessionID
	payload["last_dispatch_at"] = now.Format(time.RFC3339)

	// PR-OPS-003: heartbeat and timeout tracking
	payload["started_at"] = now.Format(time.RFC3339)
	payload["last_heartbeat_at"] = now.Format(time.RFC3339)
	payload["timeout_at"] = now.Add(30 * time.Minute).Format(time.RFC3339) // default 30min timeout
	payload["stalled"] = false

	payload["last_dispatch_error"] = nil
	payload["last_error_reason"] = nil
	payload["workspace_path"] = workspacePath
	payload["artifact_path"] = artifactPath
	payload["transport"] = transportType
	payload["project_id"] = projectID
	payload["status"] = "in_progress"
	payload["command"] = command
	if shell != "" {
		payload["shell"] = shell
	}

	if err := s.persistCompatPayload(ctx, taskID, payload); err != nil {
		return nil, err
	}

	updatedTask, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if executionManager == nil || strings.TrimSpace(command) == "" {
		reason := payload["execution_runtime"].(string) + " runtime command is not configured"
		s.finishCompatExecutionFailure(context.Background(), taskID, reason)
		failedTask, reloadErr := s.repo.GetTaskByID(context.Background(), taskID)
		if reloadErr != nil {
			return nil, reloadErr
		}
		return failedTask, nil
	}

	go s.runCompatExecution(executionManager, taskID, mergeCompatPayload(payload, map[string]interface{}{}), transportType, command, shell)
	return updatedTask, nil
}

func validateCompatDispatchEligibility(task *ent.Task, payload map[string]interface{}, isRetry bool) error {
	dispatchStatus := firstCompatNonEmpty(readString(payload, "dispatch_status"), mapCompatDispatchStatus(task.State))
	sessionID := readString(payload, "execution_session_id")

	if !isRetry {
		switch task.State {
		case engine.StateWorkspacePrepared, engine.StateRunning:
			return fmt.Errorf("task is already executing")
		case engine.StatePatchReady, engine.StateVerified, engine.StateMerged, engine.StateDone:
			return fmt.Errorf("task is already completed")
		}
	}

	if sessionID != "" {
		return fmt.Errorf("task already has an active execution session")
	}

	if !isRetry && compatDispatchInFlight(dispatchStatus) {
		return fmt.Errorf("task is already dispatching")
	}

	return nil
}

func (s *Server) prepareCompatTaskForExecution(ctx context.Context, task *ent.Task) error {
	current, err := s.repo.GetTaskByID(ctx, task.ID)
	if err != nil {
		return err
	}
	if current == nil {
		return errCompatTaskNotFound
	}

	for {
		switch current.State {
		case engine.StateQueued, engine.StateRetryWaiting:
			if err := s.transitionCompatTaskState(ctx, current, engine.StateRouted, "", ""); err != nil {
				return err
			}
		case engine.StateRouted:
			if err := s.transitionCompatTaskState(ctx, current, engine.StateWorkspacePrepared, "", ""); err != nil {
				return err
			}
		case engine.StateWorkspacePrepared:
			if err := s.transitionCompatTaskState(ctx, current, engine.StateRunning, "", ""); err != nil {
				return err
			}
		case engine.StateRunning:
			return nil
		default:
			return fmt.Errorf("task state does not allow dispatch")
		}

		current, err = s.repo.GetTaskByID(ctx, task.ID)
		if err != nil {
			return err
		}
		if current == nil {
			return errCompatTaskNotFound
		}
	}
}

func (s *Server) transitionCompatTaskState(ctx context.Context, task *ent.Task, toState, reason, details string) error {
	return s.repo.UpdateTaskState(ctx, task.ID, task.State, toState, reason, &store.EventData{
		EventID:   uuid.NewString(),
		TaskID:    task.ID,
		EventType: "state_transition",
		FromState: task.State,
		ToState:   toState,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		Attempt:   task.RetryCount,
		Transport: task.Transport,
		Details:   details,
	})
}

func (s *Server) persistCompatPayload(ctx context.Context, taskID string, payload map[string]interface{}) error {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return errCompatTaskNotFound
	}

	card, err := buildCompatTaskCard(payload, s.mapTaskView(task), taskID, s.defaultCompatProjectID())
	if err != nil {
		return err
	}
	return s.repo.UpdateTask(ctx, taskID, card)
}

func (s *Server) runCompatExecution(executionManager *compatExecutionManager, taskID string, payload map[string]interface{}, transportType, command, shell string) {
	// Create heartbeat context - will be cancelled when execution completes
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	defer heartbeatCancel()

	// PR-OPS-003: Start heartbeat updater goroutine (lightweight, only updates last_heartbeat_at)
	go s.runExecutionHeartbeat(heartbeatCtx, taskID)

	ctx := context.Background()

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil || task == nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to load task before async execution")
		return
	}

	if executionManager == nil {
		s.finishCompatExecutionFailure(ctx, taskID, "project execution is not configured")
		return
	}

	executor, err := executionManager.executor(transportType)
	if err != nil {
		s.finishCompatExecutionFailure(ctx, taskID, err.Error())
		return
	}

	compatTask := s.mapCompatTask(task)
	execConfig := &transport.TaskConfig{
		TaskID:        taskID,
		Transport:     transport.TransportType(firstCompatNonEmpty(transportType, string(transport.TransportCLI))),
		WorkspacePath: firstCompatNonEmpty(readString(payload, "workspace_path"), s.mapTaskView(task).WorkspacePath),
		ArtifactPath:  firstCompatNonEmpty(readString(payload, "artifact_path"), s.mapTaskView(task).ArtifactPath),
		OutputPath:    executionManager.logPath(taskID),
		FilesToModify: compatFilesToModify(payload),
		Command:       command,
		Shell:         shell,
		Env:           buildCompatExecutionEnv(compatTask),
		Context:       compatContextPayload(payload),
	}

	result, execErr := executor.Execute(ctx, execConfig)
	if execErr != nil || result == nil || !result.Success {
		// PR-1: System tasks (review, remediation) that fail only because the agent
		// didn't write the expected report file should auto-create the report from
		// execution output and proceed as success.
		taskType := readString(payload, "type")
		errMsg := compatExecutionError(execErr, result)
		if IsSystemTaskType(taskType) && result != nil && strings.Contains(errMsg, "empty_artifact_match") {
			if s.autoCreateSystemReport(executionManager, taskID, taskType, result.Output, payload) {
				result = &transport.ExecutionResult{
					Success:  true,
					ExitCode: 0,
					Output:   result.Output,
				}
				if err := s.finishCompatExecutionSuccess(ctx, taskID, payload, result); err != nil {
					s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to mark async execution success after report auto-creation")
				}
				return
			}
		}
		s.finishCompatExecutionFailure(ctx, taskID, errMsg)
		return
	}

	// Pass execution result to success handler for review decision extraction
	if err := s.finishCompatExecutionSuccess(ctx, taskID, payload, result); err != nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to mark async execution success")
	}
}

// runExecutionHeartbeat periodically updates the last_heartbeat_at field during execution.
// PR-OPS-003: This provides true execution liveness tracking, not just a one-time write.
// Uses UpdateHeartbeatOnly to avoid clobbering other concurrent edits.
func (s *Server) runExecutionHeartbeat(ctx context.Context, taskID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update heartbeat using lightweight method
			now := time.Now().UTC()
			if err := s.repo.UpdateHeartbeatOnly(ctx, taskID, now); err != nil {
				s.logger.Warn().Err(err).Str("task_id", taskID).Msg("heartbeat update failed")
			} else {
				s.logger.Debug().Str("task_id", taskID).Msg("heartbeat updated")
			}
		}
	}
}

func (s *Server) finishCompatExecutionSuccess(ctx context.Context, taskID string, payload map[string]interface{}, result *transport.ExecutionResult) error {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return errCompatTaskNotFound
	}

	taskType := readString(payload, "type")
	// System tasks (remediation, review, rework) skip review, go directly to verified.
	// Non-system tasks enter review_pending per PRD-DA-001 §7.3.
	shouldReview := !IsSystemTaskType(taskType)

	// For review tasks, extract structured decision from output
	if taskType == "code-review" || taskType == "review" {
		reviewDecision, resultSummary := extractReviewDecision(result.Output)
		if reviewDecision != "" {
			payload["review_decision"] = reviewDecision
			payload["result_summary"] = resultSummary
			s.logger.Info().
				Str("task_id", taskID).
				Str("review_decision", reviewDecision).
				Msg("review decision extracted from execution output")
		}
	}

	if task.State == engine.StateRunning {
		if shouldReview {
			// PRD-DA-001: code tasks go to review_pending instead of verified
			if err := s.transitionCompatTaskState(ctx, task, engine.StateReviewPending, "", ""); err != nil {
				return err
			}
		} else {
			// System tasks (remediation, review) skip review, go directly to verified
			if err := s.transitionCompatTaskState(ctx, task, engine.StatePatchReady, "", ""); err != nil {
				return err
			}
		}
	}

	task, err = s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return errCompatTaskNotFound
	}

	if shouldReview {
		payload["dispatch_status"] = "completed"
		payload["status"] = "review_pending"
		payload["coordination_stage"] = "review_pending"
		payload["last_dispatch_error"] = nil
		payload["last_error_reason"] = nil
		payload["execution_session_id"] = nil
		return s.persistCompatPayload(ctx, taskID, payload)
	}

	// Legacy path for system tasks: patch_ready → verified → done
	if task.State == engine.StatePatchReady {
		if err := s.transitionCompatTaskState(ctx, task, engine.StateVerified, "", ""); err != nil {
			return err
		}
	}

	payload["dispatch_status"] = "completed"
	payload["status"] = "verified"
	payload["last_dispatch_error"] = nil
	payload["last_error_reason"] = nil
	payload["execution_session_id"] = nil
	return s.persistCompatPayload(ctx, taskID, payload)
}

// extractReviewDecision parses review decision from execution output.
// Looks for JSON block: {"review_decision": "approved|rejected", "result_summary": "..."}
func extractReviewDecision(output string) (decision, summary string) {
	if output == "" {
		return "", ""
	}

	// Try to find JSON block in output
	startIdx := strings.Index(output, "{")
	endIdx := strings.LastIndex(output, "}")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return "", ""
	}

	jsonStr := output[startIdx : endIdx+1]
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", ""
	}

	if d, ok := data["review_decision"].(string); ok {
		decision = strings.ToLower(strings.TrimSpace(d))
	}
	if s, ok := data["result_summary"].(string); ok {
		summary = s
	}

	return decision, summary
}

func (s *Server) finishCompatExecutionFailure(ctx context.Context, taskID, reason string) {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil || task == nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to load task for execution failure")
		return
	}

	// Classify the failure into structured info per PRD-DA-001 §8
	failure := ClassifyError(reason)
	payload := mergeCompatPayload(decodeCompatPayload(task.CardJSON), map[string]interface{}{})

	rootTaskID := readString(payload, "root_task_id")
	if rootTaskID == "" {
		rootTaskID = taskID
	}
	failureSignature := GenerateFailureSignature(rootTaskID, failure.FailureCode, reason)
	autoRepairCount := readIntDefault(payload, 0, "auto_repair_count")

	payload["failure_code"] = string(failure.FailureCode)
	payload["failure_signature"] = failureSignature
	payload["error_stage"] = failure.FailureStage
	payload["retryable"] = failure.Retryable
	payload["suggested_action"] = failure.SuggestedAction
	payload["auto_repair_count"] = autoRepairCount
	payload["coordination_stage"] = "triage"

	if task.State == engine.StateRunning {
		if failure.Retryable {
			// Retryable failures enter triage for automatic remediation
			if err := s.transitionCompatTaskState(ctx, task, engine.StateTriage, string(failure.FailureCode), reason); err != nil {
				s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to transition task to triage")
				return
			}
			syncCompatPayloadState(payload, engine.StateTriage)
		} else {
			// Non-retryable failures go directly to retry_waiting (legacy path)
			failureReason := compatFailureReason(reason)
			if err := s.transitionCompatTaskState(ctx, task, engine.StateRetryWaiting, failureReason, reason); err != nil {
				s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to transition task after execution failure")
				return
			}
			syncCompatPayloadState(payload, engine.StateRetryWaiting)
		}
	}

	payload["last_dispatch_error"] = reason
	payload["last_error_reason"] = reason
	payload["execution_session_id"] = nil
	if err := s.persistCompatPayload(ctx, taskID, payload); err != nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to persist execution failure payload")
	}
}

func buildCompatExecutionEnv(task compatSchedulerTask) map[string]string {
	return map[string]string{
		"TASK_ID":               task.TaskID,
		"TASK_TITLE":            task.Title,
		"TASK_TYPE":             task.Type,
		"TASK_OWNER_AGENT":      task.OwnerAgent,
		"TASK_STATUS":           task.Status,
		"TASK_DESCRIPTION":      task.Description,
		"TASK_CURRENT_FOCUS":    task.CurrentFocus,
		"TASK_INPUT_ARTIFACTS":  strings.Join(task.InputArtifacts, "\n"),
		"TASK_OUTPUT_ARTIFACTS": strings.Join(task.OutputArtifacts, "\n"),
		"TASK_ACCEPTANCE":       strings.Join(task.AcceptanceCriteria, "\n"),
		"TASK_PROMPT":           buildCompatTaskPrompt(task),
	}
}

func buildCompatTaskPrompt(task compatSchedulerTask) string {
	lines := []string{
		"You are executing a scheduler task inside the current workspace.",
		"Make the required file changes directly in this workspace.",
		"Do not only describe the change. Write the files before exiting.",
		"Task ID: " + task.TaskID,
		"Title: " + task.Title,
		"Type: " + task.Type,
		"Owner: " + task.OwnerAgent,
	}
	if task.Description != "" {
		lines = append(lines, "Description: "+task.Description)
	}
	if task.CurrentFocus != "" {
		lines = append(lines, "Current focus: "+task.CurrentFocus)
	}
	if len(task.InputArtifacts) > 0 {
		lines = append(lines, "Input artifacts:\n- "+strings.Join(task.InputArtifacts, "\n- "))
	}
	if len(task.OutputArtifacts) > 0 {
		lines = append(lines, "Expected outputs:\n- "+strings.Join(task.OutputArtifacts, "\n- "))
	}
	if len(task.AcceptanceCriteria) > 0 {
		lines = append(lines, "Acceptance criteria:\n- "+strings.Join(task.AcceptanceCriteria, "\n- "))
	}

	if task.Type == string(RemediationTypeCodeReview) || task.Type == string(RemediationTypeNoopReview) || task.Type == string(RemediationTypeTriage) {
		lines = append(lines, "")
		lines = append(lines, "This task is analysis/review oriented.")
		lines = append(lines, "Write your findings into the expected output file paths inside the workspace before exiting.")
		lines = append(lines, "If no source change is required, still create the review/report file and explain the outcome there.")
	}

	// Special handling for review tasks: require structured decision output
	if task.Type == "code-review" || task.Type == "review" {
		lines = append(lines, "")
		lines = append(lines, "IMPORTANT: This is a review task. You must output a structured decision.")
		lines = append(lines, "At the end of your response, include a JSON block with:")
		lines = append(lines, "  {\"review_decision\": \"approved\" or \"rejected\", \"result_summary\": \"your detailed review findings\"}")
		lines = append(lines, "If the code/output is correct and meets requirements, set review_decision to \"approved\".")
		lines = append(lines, "If there are issues, errors, or improvements needed, set review_decision to \"rejected\" and explain the issues in result_summary.")
	}

	// Special handling for rework tasks: reference the rejection reason
	if task.Type == "rework" {
		lines = append(lines, "")
		lines = append(lines, "This is a rework task. Address the issues from the previous review and ensure all feedback is incorporated.")
	}

	lines = append(lines, "Before exiting, ensure every expected output path exists if the task requires it.")
	return strings.Join(lines, "\n")
}

func compatFilesToModify(payload map[string]interface{}) []string {
	files := compatStringSliceField(payload, "files_to_modify")
	if len(files) > 0 {
		return files
	}
	return compatStringSliceField(payload, "output_artifacts")
}

func compatContextPayload(payload map[string]interface{}) map[string]interface{} {
	value, ok := payload["context"]
	if !ok || value == nil {
		return nil
	}
	contextMap, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	return contextMap
}

func compatExecutionError(err error, result *transport.ExecutionResult) string {
	if err != nil {
		return err.Error()
	}
	if result == nil {
		return "task execution failed"
	}
	if strings.TrimSpace(result.Error) != "" {
		return result.Error
	}
	if strings.TrimSpace(result.Output) != "" {
		return result.Output
	}
	return "task execution failed"
}

func compatFailureReason(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "empty_artifact_match"),
		strings.Contains(lower, "no files matched"):
		return "empty_artifact_match"
	default:
		return "execution_failure"
	}
}

func compatDispatchErrorDetail(err error) string {
	if err == nil {
		return "Failed to dispatch scheduler task"
	}
	if err == errCompatTaskNotFound {
		return "Task not found"
	}
	return err.Error()
}

// autoCreateSystemReport writes a synthetic report file for system tasks when the
// agent execution succeeded but didn't produce the expected report file.
// PR-1: This prevents empty_artifact_match failures for review/remediation tasks.
func (s *Server) autoCreateSystemReport(mgr *compatExecutionManager, taskID, taskType, output string, payload map[string]interface{}) bool {
	artifactBase := mgr.artifactPath(taskID)
	reportRelPath := compatSystemReportPath(taskID, taskType)
	fullReportPath := filepath.Join(artifactBase, reportRelPath)

	if err := os.MkdirAll(filepath.Dir(fullReportPath), 0755); err != nil {
		s.logger.Error().Err(err).Str("path", fullReportPath).Msg("failed to create report directory for system task")
		return false
	}

	reviewDecision, resultSummary := extractReviewDecision(output)
	content := generateSystemReportContent(taskID, taskType, output, reviewDecision, resultSummary)

	if err := os.WriteFile(fullReportPath, []byte(content), 0644); err != nil {
		s.logger.Error().Err(err).Str("path", fullReportPath).Msg("failed to write auto-generated system report")
		return false
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("task_type", taskType).
		Str("report_path", reportRelPath).
		Msg("auto-created system task report file")

	return true
}

// generateSystemReportContent produces a markdown report from execution output.
func generateSystemReportContent(taskID, taskType, output, reviewDecision, resultSummary string) string {
	var b strings.Builder
	b.WriteString("# System Task Report\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Task ID | %s |\n", taskID))
	b.WriteString(fmt.Sprintf("| Type | %s |\n", taskType))
	b.WriteString(fmt.Sprintf("| Generated | %s |\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("| Source | auto-generated from execution output |\n"))
	if reviewDecision != "" {
		b.WriteString(fmt.Sprintf("| Review Decision | %s |\n", reviewDecision))
	}
	if resultSummary != "" {
		b.WriteString(fmt.Sprintf("| Result Summary | %s |\n", resultSummary))
	}
	b.WriteString("\n## Execution Output\n\n")
	if output != "" {
		// Truncate very long outputs
		truncated := output
		if len(truncated) > 8000 {
			truncated = truncated[:8000] + "\n... (truncated)"
		}
		b.WriteString(truncated)
	} else {
		b.WriteString("(no output captured)")
	}
	b.WriteString("\n")
	return b.String()
}
