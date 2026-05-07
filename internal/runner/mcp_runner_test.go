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
			name:    "invalid transport",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000", Transport: "websocket"},
			wantErr: "transport must be",
		},
		{
			name:    "negative timeout",
			cfg:     MCPRunnerConfig{Endpoint: "http://localhost:3000", TimeoutMs: -5},
			wantErr: "timeout_ms must be non-negative",
		},
		{
			name: "valid http transport",
			cfg:  MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", Transport: "http"},
		},
		{
			name: "valid sse transport",
			cfg:  MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp", Transport: "sse"},
		},
		{
			name: "valid default (empty transport)",
			cfg:  MCPRunnerConfig{Endpoint: "http://localhost:3000/mcp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMCPRunner_Execute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify JSON-RPC request
		var req mcpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			w.WriteHeader(400)
			return
		}
		if req.JSONRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
		}
		if req.Method != "tools/call" {
			t.Errorf("expected method tools/call, got %s", req.Method)
		}

		// Return a valid MCP response with content
		resp := mcpResponse{
			JSONRPC: "2.0",
			Result: json.RawMessage(`{
				"content": [{"type": "text", "text": "task completed successfully"}]
			}`),
			ID: req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint: srv.URL,
	})

	result, err := runner.Execute(context.Background(), RunnerTask{
		ID:   "task-1",
		Type: "feature",
		Context: map[string]interface{}{
			"prompt": "implement feature X",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}
	if result.Output != "task completed successfully" {
		t.Errorf("expected output 'task completed successfully', got %q", result.Output)
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
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint: srv.URL,
	})

	result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1"})
	if err != nil {
		t.Fatalf("Execute should not return error for RPC errors: %v", err)
	}
	if result.Success {
		t.Errorf("expected Success=false for RPC error")
	}
	if !strings.Contains(result.Error, "Method not found") {
		t.Errorf("expected error message about method not found, got %q", result.Error)
	}
}

func TestMCPRunner_Execute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint: srv.URL,
	})

	result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1"})
	if err != nil {
		t.Fatalf("Execute should not return error for HTTP errors: %v", err)
	}
	if result.Success {
		t.Errorf("expected Success=false for HTTP 503")
	}
	if result.ExitCode != 503 {
		t.Errorf("expected ExitCode=503, got %d", result.ExitCode)
	}
}

func TestMCPRunner_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mcpRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "initialize" {
			t.Errorf("expected method 'initialize', got %s", req.Method)
		}

		resp := mcpResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"protocolVersion": "2024-11-05"}`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint: srv.URL,
	})

	if err := runner.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestMCPRunner_HealthCheck_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	runner := NewMCPRunner("test-mcp", "test-agent", MCPRunnerConfig{
		Endpoint: srv.URL,
	})

	if err := runner.HealthCheck(context.Background()); err == nil {
		t.Errorf("expected HealthCheck to fail for HTTP 500")
	}
}

func TestMCPRunner_Interface(t *testing.T) {
	r := NewMCPRunner("id", "name", MCPRunnerConfig{Endpoint: "http://localhost"})
	if r.Type() != RunnerTypeMCP {
		t.Errorf("Type: got %s, want %s", r.Type(), RunnerTypeMCP)
	}
	if r.String() != "id" {
		t.Errorf("String: got %s, want id", r.String())
	}
	if r.ID() != "id" {
		t.Errorf("ID: got %s, want id", r.ID())
	}

	m, err := r.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if m.Runtime != "mcp" {
		t.Errorf("Runtime: got %s, want mcp", m.Runtime)
	}
}
