package runner

import (
	"context"
	"testing"
)

func TestRunnerInterface(t *testing.T) {
	// Verify CLIRunner implements Runner interface at compile time.
	// This test passes if the code compiles; it catches interface breakages early.
	var _ Runner = (*CLIRunner)(nil)
	_ = t // suppress unused warning
}

func TestNoopRunner(t *testing.T) {
	n := &NoopRunner{}
	ctx := context.Background()

	// Execute
	result, err := n.Execute(ctx, RunnerTask{ID: "test-task"})
	if err != nil {
		t.Fatalf("NoopRunner.Execute failed: %v", err)
	}
	if !result.Success {
		t.Errorf("NoopRunner.Execute: expected Success=true, got false")
	}

	// HealthCheck
	if err := n.HealthCheck(ctx); err != nil {
		t.Errorf("NoopRunner.HealthCheck failed: %v", err)
	}

	// Cancel
	if err := n.Cancel(ctx, "test-task"); err != nil {
		t.Errorf("NoopRunner.Cancel failed: %v", err)
	}

	// GetCapabilities
	m, err := n.GetCapabilities(ctx)
	if err != nil {
		t.Fatalf("NoopRunner.GetCapabilities failed: %v", err)
	}
	if m.Name != "noop" {
		t.Errorf("NoopRunner.GetCapabilities: expected Name=noop, got %q", m.Name)
	}

	// Type
	if n.Type() != RunnerTypeCLI {
		t.Errorf("NoopRunner.Type: expected cli, got %s", n.Type())
	}

	// String
	if n.String() != "noop" {
		t.Errorf("NoopRunner.String: expected noop, got %s", n.String())
	}
}

func TestCapabilityManifest_SupportsTaskType(t *testing.T) {
	tests := []struct {
		name      string
		manifest  *CapabilityManifest
		taskType  string
		want      bool
	}{
		{
			name:     "empty task types means all",
			manifest: &CapabilityManifest{},
			taskType: "feature",
			want:     true,
		},
		{
			name:     "exact match",
			manifest: &CapabilityManifest{TaskTypes: []string{"feature", "bugfix"}},
			taskType: "feature",
			want:     true,
		},
		{
			name:     "no match",
			manifest: &CapabilityManifest{TaskTypes: []string{"bugfix", "refactor"}},
			taskType: "feature",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.manifest.SupportsTaskType(tt.taskType)
			if got != tt.want {
				t.Errorf("SupportsTaskType(%q) = %v, want %v", tt.taskType, got, tt.want)
			}
		})
	}
}

func TestDefaultCLIManifest(t *testing.T) {
	m := DefaultCLIManifest("test-agent")
	if m.Name != "test-agent" {
		t.Errorf("expected name test-agent, got %q", m.Name)
	}
	if m.ModelFamily != "claude" {
		t.Errorf("expected modelFamily claude, got %q", m.ModelFamily)
	}
	if m.Runtime != "cli" {
		t.Errorf("expected runtime cli, got %q", m.Runtime)
	}
	if !m.SupportsThinking {
		t.Errorf("expected SupportsThinking=true")
	}
	if len(m.TaskTypes) == 0 {
		t.Errorf("expected some task types")
	}
}

func TestRunnerTypeConstants(t *testing.T) {
	if RunnerTypeCLI != "cli" {
		t.Errorf("RunnerTypeCLI: expected cli, got %s", RunnerTypeCLI)
	}
	if RunnerTypeHTTP != "http" {
		t.Errorf("RunnerTypeHTTP: expected http, got %s", RunnerTypeHTTP)
	}
	if RunnerTypeMCP != "mcp" {
		t.Errorf("RunnerTypeMCP: expected mcp, got %s", RunnerTypeMCP)
	}
}

func TestRunnerResultStats(t *testing.T) {
	r := &RunnerResult{
		Success:   true,
		ExitCode:  0,
		Duration:  0,
		Stats: RunnerStats{
			TokensUsed:  1234,
			RoundTrips:  2,
			ModelUsed:   "claude-sonnet-4",
			Attempt:     1,
		},
	}
	if r.Stats.TokensUsed != 1234 {
		t.Errorf("TokensUsed: expected 1234, got %d", r.Stats.TokensUsed)
	}
}

func TestCLIRunnerConfig(t *testing.T) {
	cfg := CLIRunnerConfig{
		ID:       "test-id",
		Name:     "test-name",
		BasePath: "/tmp/worktrees",
		MainRepo: "/tmp/repo",
	}
	runner := NewCLIRunner(cfg)
	if runner.Type() != RunnerTypeCLI {
		t.Errorf("expected type cli, got %s", runner.Type())
	}
	if runner.ID() != "test-id" {
		t.Errorf("expected id test-id, got %s", runner.ID())
	}
}

func TestNewRunnerTaskFromCompat(t *testing.T) {
	task := NewRunnerTaskFromCompat(
		"/tmp/repo",
		"task-123",
		"/tmp/worktrees/task-123",
		"/tmp/artifacts/task-123",
		[]string{"*.go", "**/*.tsx"},
		"claude-code --dangerously-skip-permissions",
		"powershell",
		map[string]string{"TASK_ID": "task-123"},
	)
	if task.ID != "task-123" {
		t.Errorf("expected id task-123, got %s", task.ID)
	}
	if task.Workspace.Path != "/tmp/worktrees/task-123" {
		t.Errorf("unexpected workspace path: %s", task.Workspace.Path)
	}
	if !task.Workspace.Isolate {
		t.Errorf("expected Isolate=true")
	}
	if len(task.FilesToModify) != 2 {
		t.Errorf("expected 2 files to modify, got %d", len(task.FilesToModify))
	}
}