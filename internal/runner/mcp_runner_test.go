package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Compile-time interface check.
var _ Runner = (*MCPRunner)(nil)

func TestMCPRunnerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     MCPRunnerConfig
		wantErr string
	}{
		{
			name:    "empty endpoint",
			cfg:     MCPRunnerConfig{},
			wantErr: "endpoint is required",
		},
		{
			name:    "whitespace endpoint",
			cfg:     MCPRunnerConfig{Endpoint: "  "},
			wantErr: "endpoint is required",
		},
		{
			name:    "relative endpoint",
			cfg:     MCPRunnerConfig{Endpoint: "localhost:3000/mcp"},
			wantErr: "endpoint must use http or https",
		},
		{
			name:    "hostless endpoint",
			cfg:     MCPRunnerConfig{Endpoint: "http://"},
			wantErr: "endpoint must include a host",
		},
		{
			name:    "non http endpoint",
			cfg:     MCPRunnerConfig{Endpoint: "ftp://example.com/mcp"},
			wantErr: "endpoint must use http or https",
		},
		{
			name:    "invalid transport",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", Transport: "websocket"},
			wantErr: "transport must be http",
		},
		{
			name:    "sse transport",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", Transport: "sse"},
			wantErr: "transport sse is not supported in Phase 6; use http",
		},
		{
			name:    "whitespace tool name",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", ToolName: "   "},
			wantErr: "tool_name is required",
		},
		{
			name:    "negative timeout",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", TimeoutMs: -5},
			wantErr: "timeout_ms must be non-negative",
		},
		{
			name: "valid http transport and default tool name",
			cfg:  MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp"},
		},
		{
			name: "valid custom tool name",
			cfg:  MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", Transport: "http", ToolName: "run_agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMCPRunner_Execute_UsesToolName(t *testing.T) {
	tests := []struct {
		name     string
		cfg      MCPRunnerConfig
		wantName string
	}{
		{
			name:     "default tool name",
			cfg:      MCPRunnerConfig{Endpoint: "http://example.invalid/mcp"},
			wantName: "execute_task",
		},
		{
			name:     "custom tool name",
			cfg:      MCPRunnerConfig{Endpoint: "http://example.invalid/mcp", ToolName: "run_agent"},
			wantName: "run_agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req mcpRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("failed to decode request: %v", err)
				}
				if req.Method != "tools/call" {
					t.Fatalf("expected method tools/call, got %s", req.Method)
				}

				var params struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				}
				if err := json.Unmarshal(req.Params, &params); err != nil {
					t.Fatalf("failed to decode params: %v", err)
				}
				if params.Name != tt.wantName {
					t.Fatalf("expected tool name %q, got %q", tt.wantName, params.Name)
				}

				resp := mcpResponse{
					JSONRPC: "2.0",
					Result: json.RawMessage(`{"content":[{"type":"text","text":"task completed successfully"}]}`),
					ID:      req.ID,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL, ToolName: tt.cfg.ToolName})
			result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1", Type: "feature"})
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			if !result.Success {
				t.Fatalf("expected Success=true, got false")
			}
			if result.Output != "task completed successfully" {
				t.Fatalf("expected output %q, got %q", "task completed successfully", result.Output)
			}
		})
	}
}

func TestMCPRunner_Execute_ArgumentsShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mcpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Method != "tools/call" {
			t.Fatalf("expected method tools/call, got %s", req.Method)
		}

		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.Fatalf("failed to decode params: %v", err)
		}
		if params.Name != "execute_task" {
			t.Fatalf("expected default tool name execute_task, got %q", params.Name)
		}

		if got := params.Arguments["task_id"]; got != "task-1" {
			t.Fatalf("task_id = %#v, want %q", got, "task-1")
		}
		if got := params.Arguments["task_type"]; got != "feature" {
			t.Fatalf("task_type = %#v, want %q", got, "feature")
		}
		if got := params.Arguments["prompt"]; got != "build the thing" {
			t.Fatalf("prompt = %#v, want %q", got, "build the thing")
		}
		if got := params.Arguments["attempt"]; got != float64(2) {
			t.Fatalf("attempt = %#v, want %v", got, 2)
		}
		if got := params.Arguments["workspace_path"]; got != "C:/workspace/task-1" {
			t.Fatalf("workspace_path = %#v, want %q", got, "C:/workspace/task-1")
		}
		files, ok := params.Arguments["files_to_modify"].([]interface{})
		if !ok {
			t.Fatalf("files_to_modify has unexpected type %T", params.Arguments["files_to_modify"])
		}
		if len(files) != 2 || files[0] != "src/**/*.go" || files[1] != "README.md" {
			t.Fatalf("files_to_modify = %#v, want [src/**/*.go README.md]", files)
		}

		resp := mcpResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"content":[{"type":"text","text":"ok"}]}`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL})
	result, err := runner.Execute(context.Background(), RunnerTask{
		ID:   "task-1",
		Type: "feature",
		Workspace: Workspace{
			Path: "C:/workspace/task-1",
		},
		FilesToModify: []string{"src/**/*.go", "README.md"},
		Context: map[string]interface{}{
			"prompt":  "build the thing",
			"attempt": 2,
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got false")
	}
}

func TestMCPRunner_Execute_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mcpResponse{
			JSONRPC: "2.0",
			Error: &mcpError{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL})

	result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1"})
	if err != nil {
		t.Fatalf("Execute should not return error for RPC errors: %v", err)
	}
	if result.Success {
		t.Fatalf("expected Success=false for RPC error")
	}
	if !strings.Contains(result.Error, "Method not found") {
		t.Fatalf("expected error message about method not found, got %q", result.Error)
	}
}

func TestMCPRunner_Execute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL})

	result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1"})
	if err != nil {
		t.Fatalf("Execute should not return error for HTTP errors: %v", err)
	}
	if result.Success {
		t.Fatalf("expected Success=false for HTTP 503")
	}
	if result.ExitCode != 503 {
		t.Fatalf("expected ExitCode=503, got %d", result.ExitCode)
	}
}

func TestMCPRunner_GetCapabilities_UsesDiscoveryHeaders(t *testing.T) {
	t.Setenv("MCP_TEST_TOKEN", "super-secret")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer super-secret" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer super-secret")
		}
		if got := r.Header.Get("X-MCP-Test"); got != "header-value" {
			t.Fatalf("X-MCP-Test header = %q, want %q", got, "header-value")
		}

		var req mcpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode discovery request: %v", err)
		}
		if req.Method != "tools/list" {
			t.Fatalf("expected method tools/list, got %s", req.Method)
		}
		if string(req.Params) != "{}" {
			t.Fatalf("expected empty object params, got %s", string(req.Params))
		}

		resp := mcpResponse{
			JSONRPC: "2.0",
			Result: json.RawMessage(`{"tools":[{"name":"execute_task"},{"name":"run_agent"}]}`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint:  srv.URL,
		Headers:   map[string]string{"X-MCP-Test": "header-value"},
		AuthToken: "Bearer ${MCP_TEST_TOKEN}",
	})

	manifest, err := runner.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetCapabilities failed: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected manifest, got nil")
	}
	if len(manifest.TaskTypes) != 2 || manifest.TaskTypes[0] != "execute_task" || manifest.TaskTypes[1] != "run_agent" {
		t.Fatalf("unexpected task types: %#v", manifest.TaskTypes)
	}
}

func TestMCPRunner_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mcpRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "initialize" {
			t.Fatalf("expected method initialize, got %s", req.Method)
		}

		resp := mcpResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL})

	if err := runner.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestMCPRunner_HealthCheck_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{Endpoint: srv.URL})

	if err := runner.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected HealthCheck to fail for HTTP 500")
	}
}

func TestMCPRunner_Interface(t *testing.T) {
	r := NewMCPRunner("id", "name", MCPRunnerConfig{Endpoint: "http://localhost"})
	if r.Type() != RunnerTypeMCP {
		t.Fatalf("Type: got %s, want %s", r.Type(), RunnerTypeMCP)
	}
	if r.String() != "id" {
		t.Fatalf("String: got %s, want id", r.String())
	}
	if r.ID() != "id" {
		t.Fatalf("ID: got %s, want id", r.ID())
	}

	m, err := r.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if m.Runtime != "mcp" {
		t.Fatalf("Runtime: got %s, want mcp", m.Runtime)
	}
}
