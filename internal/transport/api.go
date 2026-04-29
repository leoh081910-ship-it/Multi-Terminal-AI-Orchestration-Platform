// Package transport provides API transport implementation using isolated workspaces.
package transport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// APITransport implements task execution using an isolated workspace directory.
type APITransport struct {
	// BasePath is the base directory for isolated workspaces.
	BasePath string
	// PathValidator validates paths for Windows compatibility.
	PathValidator *PathValidator
}

// NewAPITransport creates a new API transport.
func NewAPITransport(basePath string) *APITransport {
	return &APITransport{
		BasePath:      basePath,
		PathValidator: NewPathValidator(),
	}
}

// Type returns the transport type.
func (t *APITransport) Type() TransportType {
	return TransportAPI
}

// Validate validates the task configuration.
func (t *APITransport) Validate(config *TaskConfig) error {
	if config.WorkspacePath == "" {
		return fmt.Errorf("workspace_path is required for API transport")
	}

	if config.ArtifactPath == "" {
		return fmt.Errorf("artifact_path is required for API transport")
	}

	if err := t.PathValidator.ValidatePath(config.WorkspacePath); err != nil {
		return fmt.Errorf("workspace_path validation failed: %w", err)
	}

	if err := t.PathValidator.ValidatePath(config.ArtifactPath); err != nil {
		return fmt.Errorf("artifact_path validation failed: %w", err)
	}

	return nil
}

// Execute runs the task in an isolated workspace directory.
func (t *APITransport) Execute(ctx context.Context, config *TaskConfig) (*ExecutionResult, error) {
	if err := t.Validate(config); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    err.Error(),
		}, err
	}

	workspacePath := config.WorkspacePath
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to create workspace: %v", err),
		}, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Execute command and capture output
	var commandOutput string
	if config.Command != "" {
		output, exitCode, err := executeCommand(ctx, workspacePath, config.Command, config.Shell, config.Env, config.OutputPath)
		commandOutput = output
		if err != nil {
			return &ExecutionResult{
				Success:  false,
				ExitCode: exitCode,
				Output:   output,
				Error:    fmt.Sprintf("command execution failed: %v", err),
			}, fmt.Errorf("command execution failed: %w", err)
		}
	}

	artifacts := []Artifact{}
	if len(config.FilesToModify) > 0 {
		for _, relPath := range config.FilesToModify {
			fullPath := filepath.Join(workspacePath, relPath)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			artifact := Artifact{
				Path:  relPath,
				Size:  info.Size(),
				IsDir: info.IsDir(),
			}
			if !info.IsDir() {
				content, err := os.ReadFile(fullPath)
				if err != nil {
					return &ExecutionResult{
						Success:  false,
						ExitCode: -1,
						Error:    fmt.Sprintf("failed to read artifact: %v", err),
						Output:   commandOutput,
					}, fmt.Errorf("failed to read artifact: %w", err)
				}
				artifact.Content = content
			}
			artifacts = append(artifacts, artifact)
		}
	}

	if len(config.FilesToModify) > 0 && len(artifacts) == 0 {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    "empty_artifact_match: no files matched the specified patterns",
			Output:   commandOutput,
		}, fmt.Errorf("empty_artifact_match: no files matched the specified patterns")
	}

	if err := writeArtifactsToPath(artifacts, config.ArtifactPath); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to write artifacts: %v", err),
			Output:   commandOutput,
		}, fmt.Errorf("failed to write artifacts: %w", err)
	}

	return &ExecutionResult{
		Success:   true,
		ExitCode:  0,
		Artifacts: artifacts,
		Output:    commandOutput,
	}, nil
}
