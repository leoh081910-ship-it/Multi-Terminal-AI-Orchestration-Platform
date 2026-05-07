// Package runner provides the universal AI Agent execution interface.
package runner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/transport"
)

// CLIRunner adapts the existing transport.CLITransport to the Runner interface.
// This is the v3 zero-breaking-change bridge: the CLITransport stays untouched,
// and we add a thin adapter that maps RunnerTask → transport.TaskConfig → result.
//
// Key design decisions:
// - CLITransport.Execute is not modified — it remains the implementation.
// - CLIRunner adds no new execution logic; it only translates between interfaces.
// - Cancel is not currently implemented by CLITransport, so Cancel returns nil.
// - HealthCheck runs a lightweight git status in the main repo.
type CLIRunner struct {
	// id is the Runner's unique identifier (matches the agent ID).
	id string

	// name is the human-readable name.
	name string

	// transport is the underlying CLITransport instance.
	transport *transport.CLITransport

	// mainRepo is the path to the main git repository (used for health checks).
	mainRepo string

	// basePath is the parent directory for worktrees.
	basePath string
}

// CLIRunnerConfig holds construction-time configuration for CLIRunner.
type CLIRunnerConfig struct {
	// ID is the Runner's unique identifier.
	ID string
	// Name is the display name (defaults to ID if empty).
	Name string
	// BasePath is the parent directory for worktrees.
	BasePath string
	// MainRepo is the path to the main git repository.
	MainRepo string
}

// NewCLIRunner creates a new CLIRunner wrapping the existing CLITransport.
func NewCLIRunner(cfg CLIRunnerConfig) *CLIRunner {
	name := cfg.Name
	if name == "" {
		name = cfg.ID
	}
	t := transport.NewCLITransport(cfg.BasePath, cfg.MainRepo)
	return &CLIRunner{
		id:        cfg.ID,
		name:      name,
		transport: t,
		mainRepo:  cfg.MainRepo,
		basePath:  cfg.BasePath,
	}
}

// Type implements Runner.
func (r *CLIRunner) Type() RunnerType { return RunnerTypeCLI }

// String implements Runner.
func (r *CLIRunner) String() string { return r.id }

// ID returns the Runner's unique identifier.
func (r *CLIRunner) ID() string { return r.id }

// Transport returns the underlying CLITransport for direct use (e.g., by compat dispatch).
// Deprecated: prefer using the Runner interface. Exposed only for backward-compatible
// compat dispatch path that calls transport methods directly.
func (r *CLIRunner) Transport() *transport.CLITransport {
	return r.transport
}

// Execute implements Runner.
// It translates RunnerTask into transport.TaskConfig and delegates to CLITransport.Execute.
func (r *CLIRunner) Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error) {
	start := time.Now()

	execConfig := &transport.TaskConfig{
		TaskID:        task.ID,
		Transport:     transport.TransportCLI,
		WorkspacePath: task.Workspace.Path,
		ArtifactPath:  task.Workspace.ArtifactPath,
		FilesToModify: task.FilesToModify,
		Command:       task.Command,
		Shell:         task.Shell,
		Env:           task.Env,
		Context:       task.Context,
	}

	// Respect task-level timeout if set
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	result, err := r.transport.Execute(ctx, execConfig)
	if err != nil {
		return &RunnerResult{
			Success:   false,
			Error:     err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
		}, err
	}

	// Translate transport artifacts to runner artifacts
	runnerArtifacts := make([]Artifact, len(result.Artifacts))
	for i, a := range result.Artifacts {
		runnerArtifacts[i] = Artifact{
			Path:    a.Path,
			Content: a.Content,
			Size:    a.Size,
			IsDir:   a.IsDir,
		}
	}

	return &RunnerResult{
		Success:   result.Success,
		Artifacts: runnerArtifacts,
		Output:    result.Output,
		Error:     result.Error,
		ExitCode:  result.ExitCode,
		Duration:  time.Since(start),
	}, nil
}

// HealthCheck implements Runner.
// It runs "git status" in the main repo as a lightweight liveness check.
func (r *CLIRunner) HealthCheck(ctx context.Context) error {
	if r.mainRepo == "" {
		// No main repo configured — check worktree base path exists
		if r.basePath != "" {
			if _, err := os.Stat(r.basePath); err != nil {
				return err
			}
		}
		return nil
	}
	// Verify the main repo is a valid git directory
	cmd := exec.CommandContext(ctx, "git", "-C", r.mainRepo, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// Cancel implements Runner.
// CLITransport does not currently support cancellation, so this returns nil.
// Cancellation is best-effort: the goroutine executing the task checks ctx.Done().
func (r *CLIRunner) Cancel(ctx context.Context, taskID string) error {
	// Cancellation is cooperative via the context passed to Execute.
	// CLITransport does not maintain a per-task cancel registry.
	// The CancelFunc in RunnerTask is the proper mechanism.
	return nil
}

// GetCapabilities implements Runner.
// Returns a default CLI manifest. Auto-detects Claude Code version if available.
func (r *CLIRunner) GetCapabilities(ctx context.Context) (*CapabilityManifest, error) {
	manifest := DefaultCLIManifest(r.name)

	// Try to detect Claude Code version and model
	if detected := detectCLICapabilities(ctx, r.id); detected != nil {
		if detected.ModelFamily != "" {
			manifest.ModelFamily = detected.ModelFamily
		}
		if detected.Version != "" {
			manifest.Version = detected.Version
		}
	}

	return manifest, nil
}

// detectedCapabilities holds auto-detected CLI runner capabilities.
type detectedCapabilities struct {
	ModelFamily   string
	ModelName     string
	Version       string
	ContextWindow int
}

// detectCLICapabilities runs the agent CLI with --version to extract model info.
// Returns nil if detection fails (non-fatal — defaults are still used).
func detectCLICapabilities(ctx context.Context, agentID string) *detectedCapabilities {
	// Try "claude --version" first
	out, err := runCapture(ctx, "claude", "--version")
	if err == nil && len(out) > 0 {
		return parseVersionOutput(string(out), "claude")
	}

	// Try "claude code --version"
	out, err = runCapture(ctx, "claude", "code", "--version")
	if err == nil && len(out) > 0 {
		return parseVersionOutput(string(out), "claude")
	}

	return nil
}

// runCapture runs a command and captures stdout.
func runCapture(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

// parseVersionOutput extracts model and version info from CLI version output.
func parseVersionOutput(output, cliType string) *detectedCapabilities {
	dc := &detectedCapabilities{}

	// Claude Code format: "Claude Code CLI 1.2.3 (model: opus-4, build: ...)"
	// or "claude/1.2.3 linux-x64"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to extract version
		if dc.Version == "" {
			// Match version patterns: 1.2.3, v1.2.3
			if matches := regexpVersion.FindStringSubmatch(line); len(matches) > 1 {
				dc.Version = matches[1]
			}
		}

		// Try to extract model name
		if cliType == "claude" {
			lower := strings.ToLower(line)
			// Look for model in parentheses or after "model:"
			if strings.Contains(lower, "model:") || strings.Contains(lower, "model=") {
				if idx := strings.Index(lower, "model"); idx >= 0 {
					rest := line[idx:]
					parts := strings.Split(rest, ",")
					if len(parts) > 0 {
						modelPart := strings.TrimPrefix(parts[0], "model")
						modelPart = strings.TrimPrefix(modelPart, ":")
						modelPart = strings.TrimPrefix(modelPart, " ")
						modelPart = strings.Trim(modelPart, " =")
						if modelPart != "" {
							dc.ModelName = modelPart
							dc.ModelFamily = "claude"
						}
					}
				}
			}

			// Also detect "opus", "sonnet", "haiku", "claude-3" in output
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "opus") {
				dc.ModelFamily = "claude"
				if !strings.Contains(dc.ModelName, "opus") {
					dc.ModelName = "claude-opus"
				}
				dc.ContextWindow = DefaultClaudeContextWindow
			} else if strings.Contains(lowerLine, "sonnet") {
				dc.ModelFamily = "claude"
				if !strings.Contains(dc.ModelName, "sonnet") {
					dc.ModelName = "claude-sonnet"
				}
				dc.ContextWindow = DefaultClaudeContextWindow
			} else if strings.Contains(lowerLine, "haiku") {
				dc.ModelFamily = "claude"
				if !strings.Contains(dc.ModelName, "haiku") {
					dc.ModelName = "claude-haiku"
				}
				dc.ContextWindow = DefaultClaudeContextWindow
			}
		}
	}

	return dc
}

// regexpVersion matches version strings like "1.2.3" or "v1.2.3".
var regexpVersion = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)

// WorkspacePathForTask returns the worktree path for a given task.
// Convenience method used by compat dispatch to pre-compute workspace paths.
func (r *CLIRunner) WorkspacePathForTask(taskID string) string {
	return filepath.Join(r.basePath, taskID)
}

// NewRunnerTaskFromCompat converts a transport compat execConfig back to RunnerTask.
// mainRepo must be provided explicitly since this is a package-level function.
// Used when refactoring compat dispatch to use Runner while keeping existing code paths.
func NewRunnerTaskFromCompat(mainRepo, taskID, workspacePath, artifactPath string,
	filesToModify []string, command, shell string, env map[string]string) RunnerTask {

	return RunnerTask{
		ID: taskID,
		Workspace: Workspace{
			Path:         workspacePath,
			ArtifactPath: artifactPath,
			MainRepo:     mainRepo,
			Isolate:      true,
		},
		FilesToModify: filesToModify,
		Command:       command,
		Shell:         shell,
		Env:           env,
	}
}