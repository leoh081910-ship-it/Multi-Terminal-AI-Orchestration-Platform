// Package executor provides the task execution coordinator that integrates transport and reverse engineering.
package executor

import (
	"context"
	"fmt"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/reverse"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
)

// Coordinator manages task execution across different transport types and special handlers.
type Coordinator struct {
	cliTransport    transport.Executor
	apiTransport    transport.Executor
	reverseExecutor *reverse.Executor
	orchestrator    *engine.Orchestrator
}

// NewCoordinator creates a new execution coordinator.
func NewCoordinator(
	cliTransport transport.Executor,
	apiTransport transport.Executor,
	reverseExecutor *reverse.Executor,
	orchestrator *engine.Orchestrator,
) *Coordinator {
	return &Coordinator{
		cliTransport:    cliTransport,
		apiTransport:    apiTransport,
		reverseExecutor: reverseExecutor,
		orchestrator:    orchestrator,
	}
}

// ExecuteTask executes a task based on its transport type and configuration.
func (c *Coordinator) ExecuteTask(ctx context.Context, taskID string, config *transport.TaskConfig) (*transport.ExecutionResult, error) {
	// Determine transport type
	var executor transport.Executor
	switch config.Transport {
	case transport.TransportCLI:
		executor = c.cliTransport
	case transport.TransportAPI:
		executor = c.apiTransport
	default:
		return nil, fmt.Errorf("unknown transport type: %s", config.Transport)
	}

	// Execute the task
	result, err := executor.Execute(ctx, config)
	if err != nil {
		return result, err
	}

	return result, nil
}

// ExecuteReverseTask executes a reverse engineering task with the verification loop.
func (c *Coordinator) ExecuteReverseTask(ctx context.Context, config *reverse.ReverseTaskConfig) (*reverse.FinalArtifact, error) {
	return c.reverseExecutor.Execute(ctx, config)
}

// ExecutionHandler provides a unified interface for task execution.
type ExecutionHandler struct {
	coordinator *Coordinator
}

// NewExecutionHandler creates a new execution handler.
func NewExecutionHandler(coordinator *Coordinator) *ExecutionHandler {
	return &ExecutionHandler{coordinator: coordinator}
}

// HandleTaskExecution handles the complete execution flow for a task.
func (h *ExecutionHandler) HandleTaskExecution(ctx context.Context, taskID string, transportType transport.TransportType, config map[string]interface{}) error {
	// Convert config to TaskConfig
	taskConfig := &transport.TaskConfig{
		TaskID:    taskID,
		Transport: transportType,
	}

	// Extract common fields
	if v, ok := config["workspace_path"].(string); ok {
		taskConfig.WorkspacePath = v
	}
	if v, ok := config["artifact_path"].(string); ok {
		taskConfig.ArtifactPath = v
	}
	filesToModify, err := parseStringSlice(config["files_to_modify"])
	if err != nil {
		return fmt.Errorf("invalid files_to_modify: %w", err)
	}
	if filesToModify != nil {
		taskConfig.FilesToModify = filesToModify
	}
	if v, ok := config["command"].(string); ok {
		taskConfig.Command = v
	}
	if v, ok := config["shell"].(string); ok {
		taskConfig.Shell = v
	}
	if v, ok := config["context"].(map[string]interface{}); ok {
		taskConfig.Context = v
	}

	// Execute the task
	result, err := h.coordinator.ExecuteTask(ctx, taskID, taskConfig)
	if err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("task execution unsuccessful: %s", result.Error)
	}

	return nil
}

// HandleReverseTaskExecution handles reverse engineering task execution.
func (h *ExecutionHandler) HandleReverseTaskExecution(ctx context.Context, config *reverse.ReverseTaskConfig) (*reverse.FinalArtifact, error) {
	return h.coordinator.ExecuteReverseTask(ctx, config)
}

func parseStringSlice(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...), nil
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string item, got %T", item)
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected []string or []interface{}, got %T", value)
	}
}
