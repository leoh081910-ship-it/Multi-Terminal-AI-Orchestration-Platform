// Package transport provides CLI transport implementation using git worktree.
package transport

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLITransport implements task execution using git worktree for isolation.
type CLITransport struct {
	// BasePath is the base directory for worktrees.
	BasePath string
	// MainRepo is the path to the main git repository.
	MainRepo string
	// PathValidator validates paths for Windows compatibility.
	PathValidator *PathValidator
}

// NewCLITransport creates a new CLI transport.
func NewCLITransport(basePath, mainRepo string) *CLITransport {
	return &CLITransport{
		BasePath:      basePath,
		MainRepo:      mainRepo,
		PathValidator: NewPathValidator(),
	}
}

// Type returns the transport type.
func (t *CLITransport) Type() TransportType {
	return TransportCLI
}

// Validate validates the task configuration.
func (t *CLITransport) Validate(config *TaskConfig) error {
	if config.WorkspacePath == "" {
		return fmt.Errorf("workspace_path is required for CLI transport")
	}

	if config.ArtifactPath == "" {
		return fmt.Errorf("artifact_path is required for CLI transport")
	}

	if len(config.FilesToModify) == 0 {
		return fmt.Errorf("files_to_modify is required for CLI transport")
	}

	// Validate paths for Windows
	if err := t.PathValidator.ValidatePath(config.WorkspacePath); err != nil {
		return fmt.Errorf("workspace_path validation failed: %w", err)
	}

	if err := t.PathValidator.ValidatePath(config.ArtifactPath); err != nil {
		return fmt.Errorf("artifact_path validation failed: %w", err)
	}

	return nil
}

// Execute runs the task using git worktree for isolation.
// REQUIRES (TRAN-01~03): CLI transport creates git worktree, extracts artifacts by glob, writes to artifacts/{task_id}/.
func (t *CLITransport) Execute(ctx context.Context, config *TaskConfig) (*ExecutionResult, error) {
	// Validate configuration
	if err := t.Validate(config); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    err.Error(),
		}, err
	}

	// Create worktree path
	worktreePath := filepath.Join(t.BasePath, config.TaskID)

	// Validate worktree path for Windows
	if err := t.PathValidator.ValidatePath(worktreePath); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("worktree path validation failed: %v", err),
		}, fmt.Errorf("worktree path validation failed: %w", err)
	}

	// Create worktree using git worktree add
	if err := t.createWorktree(ctx, worktreePath, config); err != nil {
		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to create worktree: %v", err),
		}, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Cleanup worktree after execution
	defer t.removeWorktree(ctx, worktreePath)

	// Execute the task command in the worktree if provided
	var commandOutput string
	if config.Command != "" {
		output, exitCode, err := executeCommand(ctx, worktreePath, config.Command, config.Shell, config.Env, config.OutputPath)
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

	// Extract artifacts matching files_to_modify patterns
	artifacts, err := t.extractArtifacts(ctx, worktreePath, config)
	if err != nil {
		// Check for empty artifact match
		if strings.Contains(err.Error(), "no files matched") {
			return &ExecutionResult{
				Success:  false,
				ExitCode: -1,
				Error:    "empty_artifact_match: no files matched the specified patterns",
				Output:   commandOutput,
			}, fmt.Errorf("empty_artifact_match: %w", err)
		}

		return &ExecutionResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("artifact extraction failed: %v", err),
			Output:   commandOutput,
		}, fmt.Errorf("artifact extraction failed: %w", err)
	}

	// Write artifacts to artifact directory
	artifactPath := config.ArtifactPath
	if err := t.writeArtifacts(artifacts, artifactPath); err != nil {
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

// createWorktree creates a git worktree for task execution.
func (t *CLITransport) createWorktree(ctx context.Context, worktreePath string, config *TaskConfig) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return fmt.Errorf("failed to create worktree parent directory: %w", err)
	}

	// Create worktree using git worktree add
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", worktreePath)
	cmd.Dir = t.MainRepo

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Worktree might already exist, try to remove and recreate
		if strings.Contains(string(output), "already exists") {
			if rmErr := t.removeWorktree(ctx, worktreePath); rmErr != nil {
				return fmt.Errorf("failed to remove existing worktree: %w (original error: %v)", rmErr, err)
			}
			// Retry creating worktree
			cmd = exec.CommandContext(ctx, "git", "worktree", "add", worktreePath)
			cmd.Dir = t.MainRepo
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to create worktree after cleanup: %w (output: %s)", err, output)
			}
		} else {
			return fmt.Errorf("failed to create worktree: %w (output: %s)", err, output)
		}
	}

	return nil
}

// removeWorktree removes a git worktree.
func (t *CLITransport) removeWorktree(ctx context.Context, worktreePath string) error {
	// Remove worktree using git worktree remove
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = t.MainRepo

	if output, err := cmd.CombinedOutput(); err != nil {
		// Try manual cleanup if git worktree remove fails
		if rmErr := os.RemoveAll(worktreePath); rmErr != nil {
			return fmt.Errorf("failed to remove worktree (git: %w, manual: %v, output: %s)", err, rmErr, output)
		}
	}

	return nil
}

// extractArtifacts extracts artifacts from the worktree matching the specified patterns.
func (t *CLITransport) extractArtifacts(ctx context.Context, worktreePath string, config *TaskConfig) ([]Artifact, error) {
	artifacts := []Artifact{}

	for _, pattern := range config.FilesToModify {
		patternArtifacts, err := t.collectArtifactsForPattern(worktreePath, worktreePath, pattern)
		if err != nil {
			return nil, err
		}

		if len(patternArtifacts) == 0 && strings.TrimSpace(t.MainRepo) != "" {
			patternArtifacts, err = t.collectArtifactsForPattern(t.MainRepo, t.MainRepo, pattern)
			if err != nil {
				return nil, err
			}
		}

		if len(patternArtifacts) == 0 {
			return nil, fmt.Errorf("no files matched pattern: %s", pattern)
		}

		artifacts = append(artifacts, patternArtifacts...)
	}

	return artifacts, nil
}

func (t *CLITransport) collectArtifactsForPattern(basePath, relRoot, pattern string) ([]Artifact, error) {
	fullPattern := filepath.Join(basePath, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
	}

	artifacts := make([]Artifact, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", match, err)
		}

		relPath, err := filepath.Rel(relRoot, match)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", match, err)
		}

		artifact := Artifact{
			Path:  relPath,
			Size:  info.Size(),
			IsDir: info.IsDir(),
		}

		if !info.IsDir() {
			content, err := os.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", match, err)
			}
			artifact.Content = content
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// writeArtifacts writes artifacts to the specified directory.
func (t *CLITransport) writeArtifacts(artifacts []Artifact, artifactPath string) error {
	return writeArtifactsToPath(artifacts, artifactPath)
}

// ArchiveArtifacts creates a tarball of artifacts for transfer.
func ArchiveArtifacts(artifacts []Artifact) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, artifact := range artifacts {
		// Create tar header
		header := &tar.Header{
			Name: artifact.Path,
			Size: int64(len(artifact.Content)),
			Mode: 0644,
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header for %s: %w", artifact.Path, err)
		}

		if _, err := tw.Write(artifact.Content); err != nil {
			return nil, fmt.Errorf("failed to write tar content for %s: %w", artifact.Path, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
