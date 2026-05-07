package registry

import (
	"encoding/json"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
)

func TestBuildRunner_HTTP(t *testing.T) {
	cfg := runner.HTTPRunnerConfig{
		Endpoint: "https://api.openai.com/v1/chat/completions",
		Model:    "gpt-4o",
	}
	cfgJSON, _ := json.Marshal(cfg)

	r, err := BuildRunner(BuilderInput{
		AgentID:    "openai-agent",
		AgentName:  "OpenAI GPT-4o",
		RunnerType: runner.RunnerTypeHTTP,
		Config:     cfgJSON,
	})
	if err != nil {
		t.Fatalf("BuildRunner HTTP failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if r.Type() != runner.RunnerTypeHTTP {
		t.Errorf("expected type http, got %s", r.Type())
	}
	if r.String() != "openai-agent" {
		t.Errorf("expected id openai-agent, got %s", r.String())
	}
}

func TestBuildRunner_MCP(t *testing.T) {
	cfg := runner.MCPRunnerConfig{
		Endpoint:      "http://localhost:3000/mcp",
		Transport:     "http",
		ToolsEnabled:  true,
	}
	cfgJSON, _ := json.Marshal(cfg)

	r, err := BuildRunner(BuilderInput{
		AgentID:    "mcp-agent",
		AgentName:  "MCP Server",
		RunnerType: runner.RunnerTypeMCP,
		Config:     cfgJSON,
	})
	if err != nil {
		t.Fatalf("BuildRunner MCP failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if r.Type() != runner.RunnerTypeMCP {
		t.Errorf("expected type mcp, got %s", r.Type())
	}
}

func TestBuildRunner_HTTP_InvalidConfig(t *testing.T) {
	// Empty endpoint should fail validation
	cfg := runner.HTTPRunnerConfig{
		Model: "gpt-4o",
	}
	cfgJSON, _ := json.Marshal(cfg)

	_, err := BuildRunner(BuilderInput{
		AgentID:    "bad-agent",
		RunnerType: runner.RunnerTypeHTTP,
		Config:     cfgJSON,
	})
	if err == nil {
		t.Fatal("expected error for empty endpoint, got nil")
	}
	if !contains(err.Error(), "endpoint is required") {
		t.Errorf("expected 'endpoint is required' error, got: %v", err)
	}
}

func TestBuildRunner_MCP_InvalidConfig(t *testing.T) {
	cfg := runner.MCPRunnerConfig{
		Endpoint:  "http://localhost:3000/mcp",
		Transport: "grpc",
	}
	cfgJSON, _ := json.Marshal(cfg)

	_, err := BuildRunner(BuilderInput{
		AgentID:    "bad-mcp",
		RunnerType: runner.RunnerTypeMCP,
		Config:     cfgJSON,
	})
	if err == nil {
		t.Fatal("expected error for invalid transport, got nil")
	}
	if !contains(err.Error(), "transport must be") {
		t.Errorf("expected transport validation error, got: %v", err)
	}
}

func TestBuildRunner_UnsupportedType(t *testing.T) {
	_, err := BuildRunner(BuilderInput{
		AgentID:    "unknown",
		RunnerType: runner.RunnerType("grpc"),
		Config:     nil,
	})
	if err == nil {
		t.Fatal("expected error for unsupported runner type")
	}
	if !contains(err.Error(), "unsupported runner type") {
		t.Errorf("expected 'unsupported runner type' error, got: %v", err)
	}
}

func TestBuildRunner_CLI(t *testing.T) {
	r, err := BuildRunner(BuilderInput{
		AgentID:    "cli-agent",
		AgentName:  "Claude Code",
		RunnerType: runner.RunnerTypeCLI,
		RegistryConfig: RegistryConfig{
			BasePath: "/tmp/worktrees",
			MainRepo: "/tmp/repo",
		},
	})
	if err != nil {
		t.Fatalf("BuildRunner CLI failed: %v", err)
	}
	if r.Type() != runner.RunnerTypeCLI {
		t.Errorf("expected type cli, got %s", r.Type())
	}
}

func TestBuildRunner_HTTP_NilConfig(t *testing.T) {
	// nil config should fail validation (empty endpoint)
	_, err := BuildRunner(BuilderInput{
		AgentID:    "nil-cfg-agent",
		RunnerType: runner.RunnerTypeHTTP,
		Config:     nil,
	})
	if err == nil {
		t.Fatal("expected error for nil config with HTTP runner")
	}
}

func TestRunnerTypeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  runner.RunnerType
	}{
		{"cli", runner.RunnerTypeCLI},
		{"http", runner.RunnerTypeHTTP},
		{"mcp", runner.RunnerTypeMCP},
		{"", runner.RunnerTypeCLI},          // empty defaults to CLI
		{"unknown", runner.RunnerTypeCLI},    // unknown defaults to CLI
		{"grpc", runner.RunnerTypeCLI},       // unsupported defaults to CLI
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := RunnerTypeFromString(tt.input)
			if got != tt.want {
				t.Errorf("RunnerTypeFromString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
