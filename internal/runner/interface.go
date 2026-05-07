// Package runner provides the universal AI Agent execution interface.
// Replaces hardcoded transport logic with a pluggable Runner abstraction.
// Each Runner implementation (CLI/HTTP/MCP) conforms to this interface.
package runner

import (
	"context"
	"time"
)

// RunnerType defines the kind of Runner implementation.
type RunnerType string

const (
	RunnerTypeCLI RunnerType = "cli"
	RunnerTypeHTTP RunnerType = "http"
	RunnerTypeMCP  RunnerType = "mcp"
)

const DefaultClaudeContextWindow = 200000

var DefaultTaskTypes = []string{"feature", "bugfix", "refactor", "documentation", "code-review", "analysis"}

// Runner is the universal execution contract for AI Agents.
// All Agent types (CLI, HTTP API, MCP) implement this interface.
type Runner interface {
	// Execute runs a task and returns the result.
	// It is responsible for workspace setup, execution, artifact collection,
	// and cleanup — the same lifecycle that CLITransport.Execute handles today.
	Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error)

	// HealthCheck verifies the Runner is alive and ready to accept tasks.
	// Returns nil if healthy, non-nil error otherwise.
	HealthCheck(ctx context.Context) error

	// Cancel signals the Runner to abort an in-flight execution.
	// Returns error if cancellation fails (e.g., task already completed).
	Cancel(ctx context.Context, taskID string) error

	// GetCapabilities returns the Agent's self-described capability manifest.
	// Called by the Registry to populate routing data.
	GetCapabilities(ctx context.Context) (*CapabilityManifest, error)

	// Type returns the Runner type (cli/http/mcp).
	Type() RunnerType

	// String returns a human-readable name for this Runner (e.g., agent ID).
	String() string
}

// RunnerTask is the unified task input for all Runner implementations.
// It wraps transport.TaskConfig with additional metadata needed for routing.
type RunnerTask struct {
	// ID is the task identifier (maps to task.ID in the store).
	ID string

	// Type is the task type (e.g., "feature", "bugfix", "code-review", "reverse").
	Type string

	// Workspace describes where and how the Runner should operate.
	Workspace Workspace

	// Command is the raw command string to execute.
	// For CLIRunner this becomes the Claude Code CLI invocation.
	Command string

	// Shell overrides the command shell. Empty means OS default.
	Shell string

	// FilesToModify are glob patterns for files the task will modify.
	// Used by CLIRunner for artifact extraction and by HTTP runners
	// to know which outputs to expect.
	FilesToModify []string

	// Env is a map of environment variables injected into the execution context.
	Env map[string]string

	// Context holds additional task-specific data passed through the pipeline.
	// Keys mirror the Task Card fields: type, description, owner_agent, etc.
	Context map[string]interface{}

	// Timeout is the maximum duration for this execution.
	// Zero means no timeout (use the platform default).
	Timeout time.Duration

	// CancelFunc is an optional cancel function the Runner should invoke
	// when it detects the task should be aborted. Set by the caller.
	CancelFunc context.CancelFunc
}

// Workspace describes the isolation and artifact layout for a task execution.
type Workspace struct {
	// Path is the root directory for the isolated workspace.
	// For CLIRunner this is the worktree root.
	Path string

	// ArtifactPath is where the Runner writes output artifacts.
	// For CLIRunner this is artifacts/{task_id}/.
	ArtifactPath string

	// MainRepo is the path to the main git repository.
	// Used only by CLIRunner for git worktree operations.
	MainRepo string

	// Isolate indicates whether the workspace should be isolated from the main repo.
	// true = git worktree (full isolation), false = shared directory.
	Isolate bool
}

// RunnerResult is the unified result output from any Runner implementation.
type RunnerResult struct {
	// Success is true if the task completed without fatal errors.
	Success bool

	// Artifacts contains the file artifacts produced during execution.
	Artifacts []Artifact

	// Output is the raw execution output (stdout/stderr).
	Output string

	// Error is the error message if Success is false.
	Error string

	// ExitCode is the process exit code (0 for success).
	ExitCode int

	// Duration is how long the execution took.
	Duration time.Duration

	// Stats holds Runner-specific execution statistics.
	Stats RunnerStats
}

// Artifact is a file produced during task execution.
type Artifact struct {
	// Path is the relative path within the artifact directory.
	Path string

	// Content is the file content (for small files).
	Content []byte

	// Size is the file size in bytes.
	Size int64

	// IsDir indicates if this is a directory.
	IsDir bool
}

// RunnerStats holds telemetry data from a Runner execution.
type RunnerStats struct {
	// TokensUsed is the number of tokens consumed (HTTP/MCP Runners).
	TokensUsed int64

	// RoundTrips is the number of API calls made.
	RoundTrips int

	// ModelUsed identifies which model was used (for HTTP/MCP).
	ModelUsed string

	// Attempt is the execution attempt number (1-indexed).
	Attempt int
}

// NoopRunner is a Runner that does nothing — used for testing or disabled agents.
type NoopRunner struct{}

func (n *NoopRunner) Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error) {
	return &RunnerResult{Success: true, ExitCode: 0}, nil
}
func (n *NoopRunner) HealthCheck(ctx context.Context) error { return nil }
func (n *NoopRunner) Cancel(ctx context.Context, taskID string) error {
	return nil
}
func (n *NoopRunner) GetCapabilities(ctx context.Context) (*CapabilityManifest, error) {
	return &CapabilityManifest{Name: "noop", TaskTypes: []string{}}, nil
}
func (n *NoopRunner) Type() RunnerType { return RunnerTypeCLI }
func (n *NoopRunner) String() string    { return "noop" }