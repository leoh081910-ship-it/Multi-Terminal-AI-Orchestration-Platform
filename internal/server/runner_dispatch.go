// Package server provides the HTTP API server for the AI orchestration platform.
// This file contains multi-agent runner dispatch integration helpers.
//
// Key concepts:
// - getRunnerForAgent: resolves an agent name to a Runner from the registry
// - buildRunnerTask: converts compat payload into a RunnerTask for execution
// - runnerResultToExecutionResult: translates RunnerResult back to transport.ExecutionResult
//   so the existing finishCompatExecutionSuccess/Failure handlers work unchanged
package server

import (
	"context"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/router"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
)

// getRunnerForAgent resolves an agent identifier to a Runner from the registry.
// Returns nil if the registry is not configured or no runner is found for the agent.
func (s *Server) getRunnerForAgent(agentID string) runner.Runner {
	if s.runnerRegistry == nil || agentID == "" {
		return nil
	}
	return s.runnerRegistry.Get(agentID)
}

// buildRunnerTask constructs a RunnerTask from compat execution parameters.
// This is the translation layer from the legacy compat path to the Runner interface.
func (s *Server) buildRunnerTask(taskID string, payload map[string]interface{}, executionManager *compatExecutionManager) runner.RunnerTask {
	// Fetch task for workspace info
	task, err := s.repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		s.logger.Warn().Err(err).Str("task_id", taskID).Msg("failed to fetch task for runner, using defaults")
	}
	var view taskView
	if task != nil {
		view = s.mapTaskView(task)
	}

	workspacePath := firstCompatNonEmpty(readString(payload, "workspace_path"), view.WorkspacePath)
	if workspacePath == "" && executionManager != nil {
		transportType := firstCompatNonEmpty(readString(payload, "transport"), string(transport.TransportCLI))
		workspacePath = executionManager.workspacePath(taskID, transportType)
	}

	artifactPath := firstCompatNonEmpty(readString(payload, "artifact_path"), view.ArtifactPath)
	if artifactPath == "" && executionManager != nil {
		artifactPath = executionManager.artifactPath(taskID)
	}

	shell := readString(payload, "shell")
	command := readString(payload, "command")

	var mainRepo string
	if executionManager != nil {
		mainRepo = executionManager.mainRepoPath
	}

	// Build context map from payload
	taskCtx := make(map[string]interface{})
	for k, v := range payload {
		if k != "" && v != nil {
			taskCtx[k] = v
		}
	}

	var compatTask compatSchedulerTask
	if task != nil {
		compatTask = s.mapCompatTask(task)
	}

	return runner.RunnerTask{
		ID:      taskID,
		Type:    readString(payload, "type"),
		Command: command,
		Shell:   shell,
		Workspace: runner.Workspace{
			Path:         workspacePath,
			ArtifactPath: artifactPath,
			MainRepo:     mainRepo,
			Isolate:      true,
		},
		FilesToModify: compatFilesToModify(payload),
		Env:          buildCompatExecutionEnv(compatTask),
		Context:      taskCtx,
		Timeout:      defaultExecutionTimeout,
	}
}

// taskView is the local alias used by mapTaskView return type.
// We need to reference it since mapTaskView is defined elsewhere in this package.
type taskView = *store.TaskView

// runnerResultToExecutionResult converts a RunnerResult to a transport.ExecutionResult.
// This allows the existing finishCompatExecutionSuccess/Failure handlers to work
// without modification, since they accept transport.ExecutionResult.
func runnerResultToExecutionResult(r *runner.RunnerResult, err error) (*transport.ExecutionResult, error) {
	if err != nil {
		return &transport.ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    err.Error(),
		}, err
	}
	if r == nil {
		return nil, nil
	}

	artifacts := make([]transport.Artifact, len(r.Artifacts))
	for i, a := range r.Artifacts {
		artifacts[i] = transport.Artifact{
			Path:    a.Path,
			Content: a.Content,
			Size:    a.Size,
			IsDir:   a.IsDir,
		}
	}

	return &transport.ExecutionResult{
		Success:   r.Success,
		Artifacts: artifacts,
		Output:    r.Output,
		Error:     r.Error,
		ExitCode:  r.ExitCode,
	}, nil
}

// RunCompatExecutionViaRunner executes a task fully via the Runner interface.
// When all tasks use this, the compat path in
// runCompatExecution can be removed.
//
// Usage: replaces the call to runCompatExecution when the RunnerRegistry is available.
func (s *Server) RunCompatExecutionViaRunner(
	ctx context.Context,
	taskID string,
	payload map[string]interface{},
	executionManager *compatExecutionManager,
) (*transport.ExecutionResult, error) {

	agentID := readString(payload, "owner_agent")

	run := s.getRunnerForAgent(agentID)
	if run == nil {
		// Try router-based fallback
		if s.taskRouter != nil {
			taskType := readString(payload, "type")
			orgID := readString(payload, "org_id")
			if orgID == "" {
				orgID = "default-org"
			}
			routeInput := router.RouteInput{
				TaskID:   taskID,
				TaskType: taskType,
				OrgID:    orgID,
				Capabilities: compatStringSliceField(payload, "capabilities"),
			}
			if selected, err := s.taskRouter.SelectBestAgent(ctx, routeInput); err == nil {
				run = s.runnerRegistry.Get(selected)
			}
		}
	}

	// Ultimate fallback: first available runner
	if run == nil && s.runnerRegistry != nil {
		runners := s.runnerRegistry.List()
		if len(runners) > 0 {
			run = runners[0]
		}
	}

	if run == nil {
		return nil, &runnerError{msg: "no Runner available for execution"}
	}

	// Mark agent running
	if agentID != "" {
		s.runnerRegistry.SetRunning(agentID, true)
		defer s.runnerRegistry.SetRunning(agentID, false)
	}

	// Heartbeat
	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.runExecutionHeartbeat(heartbeatCtx, taskID)

	// Build task
	runnerTask := s.buildRunnerTask(taskID, payload, executionManager)

	// Execute
	result, execErr := run.Execute(ctx, runnerTask)

	// Translate to compat result
	compatResult, _ := runnerResultToExecutionResult(result, execErr)
	return compatResult, execErr
}

// runnerError is a simple error type for runner dispatch errors.
type runnerError struct {
	msg string
}

func (e *runnerError) Error() string { return e.msg }