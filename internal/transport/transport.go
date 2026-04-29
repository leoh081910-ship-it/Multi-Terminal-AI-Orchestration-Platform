// Package transport provides execution transport abstractions for the AI orchestration platform.
// It supports CLI transport (git worktree based) and API transport (isolated workspace based).
package transport

import (
	"context"
	"fmt"
	"path/filepath"
)

// TransportType represents the type of transport.
type TransportType string

const (
	// TransportCLI uses git worktree for isolated execution.
	TransportCLI TransportType = "cli"
	// TransportAPI uses isolated directory for execution.
	TransportAPI TransportType = "api"
)

// Artifact represents a file artifact produced by task execution.
type Artifact struct {
	// Path is the relative path within the artifact directory.
	Path string `json:"path"`
	// Content is the file content (for small files).
	Content []byte `json:"content,omitempty"`
	// Size is the file size in bytes.
	Size int64 `json:"size"`
	// IsDir indicates if this is a directory.
	IsDir bool `json:"is_dir"`
}

// ExecutionResult contains the result of task execution.
type ExecutionResult struct {
	// Success indicates if execution was successful.
	Success bool `json:"success"`
	// Artifacts contains the produced artifacts.
	Artifacts []Artifact `json:"artifacts,omitempty"`
	// Error is the error message if execution failed.
	Error string `json:"error,omitempty"`
	// Output contains execution output (stdout/stderr).
	Output string `json:"output,omitempty"`
	// ExitCode is the process exit code.
	ExitCode int `json:"exit_code"`
}

// TaskConfig contains configuration for task execution.
type TaskConfig struct {
	// TaskID is the unique task identifier.
	TaskID string `json:"task_id"`
	// Transport is the transport type (cli or api).
	Transport TransportType `json:"transport"`
	// WorkspacePath is the base workspace path.
	WorkspacePath string `json:"workspace_path"`
	// ArtifactPath is the artifact output path.
	ArtifactPath string `json:"artifact_path"`
	// FilesToModify are glob patterns for files to extract (CLI transport).
	FilesToModify []string `json:"files_to_modify,omitempty"`
	// Command is the execution command (optional).
	Command string `json:"command,omitempty"`
	// Shell overrides the command shell. Empty means OS default.
	Shell string `json:"shell,omitempty"`
	// Env contains additional environment variables for command execution.
	Env map[string]string `json:"env,omitempty"`
	// OutputPath is the optional execution log file path.
	OutputPath string `json:"-"`
	// Context contains additional task context.
	Context map[string]interface{} `json:"context,omitempty"`
}

// Executor is the interface for task execution transports.
type Executor interface {
	// Execute runs the task and returns the execution result.
	Execute(ctx context.Context, config *TaskConfig) (*ExecutionResult, error)
	// Type returns the transport type.
	Type() TransportType
	// Validate validates the task configuration.
	Validate(config *TaskConfig) error
}

// PathValidator provides path validation utilities.
type PathValidator struct {
	// MaxPathLength is the maximum allowed path length (Windows: 260).
	MaxPathLength int
}

// NewPathValidator creates a new path validator.
func NewPathValidator() *PathValidator {
	return &PathValidator{
		MaxPathLength: 260,
	}
}

// ValidatePath validates a path for Windows compatibility.
// REQUIRES (TRAN-06): Windows paths validated for length while allowing spaces and Chinese characters.
func (pv *PathValidator) ValidatePath(path string) error {
	// Check path length
	if len(path) > pv.MaxPathLength {
		return fmt.Errorf("path exceeds maximum length of %d characters: %d", pv.MaxPathLength, len(path))
	}

	return nil
}

// NormalizePath normalizes a path for the current OS.
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// ArtifactManager manages artifact storage and retrieval.
type ArtifactManager struct {
	// BasePath is the base directory for artifacts.
	BasePath string
}

// NewArtifactManager creates a new artifact manager.
func NewArtifactManager(basePath string) *ArtifactManager {
	return &ArtifactManager{
		BasePath: basePath,
	}
}

// GetArtifactPath returns the artifact path for a task.
func (am *ArtifactManager) GetArtifactPath(taskID string) string {
	return filepath.Join(am.BasePath, taskID)
}

// GetReverseArtifactPath returns the reverse engineering artifact path.
func (am *ArtifactManager) GetReverseArtifactPath(taskID string) string {
	return filepath.Join(am.BasePath, taskID, "reverse")
}
