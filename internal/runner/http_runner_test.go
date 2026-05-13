package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Compile-time interface check.
var _ Runner = (*HTTPRunner)(nil)

func TestHTTPRunnerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     HTTPRunnerConfig
		wantErr string
	}{
		{
			name:    "empty endpoint",
			cfg:     HTTPRunnerConfig{},
			wantErr: "endpoint is required",
		},
		{
			name:    "whitespace-only endpoint",
			cfg:     HTTPRunnerConfig{Endpoint: "   "},
			wantErr: "endpoint is required",
		},
		{
			name:    "relative endpoint",
			cfg:     HTTPRunnerConfig{Endpoint: "/run"},
			wantErr: "endpoint scheme must be http or https",
		},
		{
			name:    "hostless endpoint",
			cfg:     HTTPRunnerConfig{Endpoint: "http:///run"},
			wantErr: "endpoint host is required",
		},
		{
			name:    "non-http scheme",
			cfg:     HTTPRunnerConfig{Endpoint: "ftp://example.test/run"},
			wantErr: "endpoint scheme must be http or https",
		},
		{
			name:    "invalid method",
			cfg:     HTTPRunnerConfig{Endpoint: "http://localhost:8080", Method: "TRACE"},
			wantErr: "method must be one of GET, POST, PUT, PATCH",
		},
		{
			name: "negative timeout",
			cfg:  HTTPRunnerConfig{Endpoint: "http://localhost:8080", TimeoutMs: -1},
			wantErr: "timeout_ms must be non-negative",
		},
		{
			name:    "invalid template",
			cfg:     HTTPRunnerConfig{Endpoint: "http://localhost:8080/v1/chat", BodyTemplate: `{{.Context.prompt`},
			wantErr: "invalid body template",
		},
		{
			name: "valid config",
			cfg:  HTTPRunnerConfig{Endpoint: "http://localhost:8080/v1/chat"},
		},
		{
			name: "valid config with all fields",
			cfg: HTTPRunnerConfig{
				Endpoint:  "https://api.openai.com/v1/chat/completions",
				Method:    "POST",
				AuthToken: "Bearer sk-test",
				Model:     "gpt-4o",
				TimeoutMs: 30000,
			},
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

func TestHTTPRunner_Execute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Content-Type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		// Verify Authorization
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Authorization Bearer test-key, got %s", auth)
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "hello world"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := NewHTTPRunner("test", "test-agent", HTTPRunnerConfig{
		Endpoint:   srv.URL,
		AuthToken:  "Bearer test-key",
		OutputPath: "choices[0].message.content",
	})

	result, err := runner.Execute(context.Background(), RunnerTask{
		ID: "task-1",
		Context: map[string]interface{}{
			"prompt": "say hello",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}
	if result.Output != "hello world" {
		t.Errorf("expected output 'hello world', got %q", result.Output)
	}
}

func TestHTTPRunner_TemplateToJSON(t *testing.T) {
	cfg := HTTPRunnerConfig{
		Endpoint:     "http://localhost:8080",
		BodyTemplate: `{"prompt": {{.Context.prompt | toJSON}}}`,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestHTTPRunner_Execute_TemplateBody(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "ok"})
	}))
	defer srv.Close()

	runner := NewHTTPRunner("test", "test-agent", HTTPRunnerConfig{
		Endpoint: srv.URL,
		BodyTemplate: `{
			"id":"{{.ID}}",
			"type":"{{.Type}}",
			"command":"{{.Command}}",
			"shell":"{{.Shell}}",
			"files": {{.FilesToModify | toJSON}},
			"workspace":"{{.Workspace.Path}}",
			"path":"{{index .Env "PATH"}}",
			"issue":"{{.Context.issue}}",
			"payload": {{.Context.payload | toJSON}},
			"timeout":"{{.Timeout}}"
		}`,
		OutputPath: "result",
	})

	result, err := runner.Execute(context.Background(), RunnerTask{
		ID:            "task-1",
		Type:          "feature",
		Command:       "echo hi",
		Shell:         "bash",
		FilesToModify: []string{"a.go", "b.go"},
		Workspace: Workspace{Path: "/tmp/workspace"},
		Env: map[string]string{
			"PATH": "/usr/bin",
		},
		Context: map[string]interface{}{
			"issue": "ISSUE-1",
			"payload": map[string]interface{}{
				"nested": []interface{}{"x", 2, true},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}
	if receivedBody["id"] != "task-1" {
		t.Errorf("expected id task-1, got %v", receivedBody["id"])
	}
	if receivedBody["type"] != "feature" {
		t.Errorf("expected type feature, got %v", receivedBody["type"])
	}
	if receivedBody["command"] != "echo hi" {
		t.Errorf("expected command echo hi, got %v", receivedBody["command"])
	}
	if receivedBody["shell"] != "bash" {
		t.Errorf("expected shell bash, got %v", receivedBody["shell"])
	}
	if receivedBody["workspace"] != "/tmp/workspace" {
		t.Errorf("expected workspace /tmp/workspace, got %v", receivedBody["workspace"])
	}
	if receivedBody["path"] != "/usr/bin" {
		t.Errorf("expected path /usr/bin, got %v", receivedBody["path"])
	}
	if receivedBody["issue"] != "ISSUE-1" {
		t.Errorf("expected issue ISSUE-1, got %v", receivedBody["issue"])
	}
	files, ok := receivedBody["files"].([]interface{})
		if !ok || len(files) != 2 {
		t.Fatalf("expected files array len 2, got %T %#v", receivedBody["files"], receivedBody["files"])
	}
	payload, ok := receivedBody["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload object, got %T", receivedBody["payload"])
	}
	nested, ok := payload["nested"].([]interface{})
	if !ok || len(nested) != 3 {
		t.Fatalf("expected nested array len 3, got %#v", payload["nested"])
	}
}

func TestHTTPRunner_Execute_DefaultBody(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "ok"})
	}))
	defer srv.Close()

	runner := NewHTTPRunner("test", "test-agent", HTTPRunnerConfig{
		Endpoint:   srv.URL,
		OutputPath: "result",
	})

	result, err := runner.Execute(context.Background(), RunnerTask{
		ID:   "task-1",
		Type: "feature",
		Context: map[string]interface{}{
			"issue": "ISSUE-2",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success")
	}
	if receivedBody["task_id"] != "task-1" {
		t.Errorf("expected task_id task-1, got %v", receivedBody["task_id"])
	}
	if receivedBody["task_type"] != "feature" {
		t.Errorf("expected task_type feature, got %v", receivedBody["task_type"])
	}
	if receivedBody["issue"] != "ISSUE-2" {
		t.Errorf("expected issue ISSUE-2, got %v", receivedBody["issue"])
	}
}

func TestHTTPRunner_Execute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	runner := NewHTTPRunner("test", "test-agent", HTTPRunnerConfig{
		Endpoint: srv.URL,
	})

	result, err := runner.Execute(context.Background(), RunnerTask{ID: "task-1"})
	if err != nil {
		t.Fatalf("Execute should not return error for HTTP errors: %v", err)
	}
	if result.Success {
		t.Errorf("expected Success=false for HTTP 500")
	}
	if result.ExitCode != 500 {
		t.Errorf("expected ExitCode=500, got %d", result.ExitCode)
	}
}

func TestNavigateJSON(t *testing.T) {
	data := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": "hello",
				},
			},
		},
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	tests := []struct {
		name    string
		path    string
		want    interface{}
		wantErr bool
	}{
		{"simple key", "nested.key", "value", false},
		{"array index", "choices[0].message.content", "hello", false},
		{"missing key", "nonexistent", nil, true},
		{"bad array index", "choices[99]", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := NavigateJSON(data, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val != tt.want {
				t.Errorf("got %v, want %v", val, tt.want)
			}
		})
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_RUNNER_VAR", "expanded-value")
	defer os.Unsetenv("TEST_RUNNER_VAR")

	got := expandEnv("prefix-${TEST_RUNNER_VAR}-suffix")
	want := "prefix-expanded-value-suffix"
	if got != want {
		t.Errorf("expandEnv: got %q, want %q", got, want)
	}

	// Unset variable returns empty
	got = expandEnv("${NONEXISTENT_VAR_12345}")
	if got != "" {
		t.Errorf("expandEnv for unset var: got %q, want empty", got)
	}
}

func TestInferModelFamily(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"gpt-4o", "gpt"},
		{"claude-sonnet-4-20250514", "claude"},
		{"gemini-2.0-flash", "gemini"},
		{"llama3", "llama"},
		{"qwen-2.5", "qwen"},
		{"deepseek-coder", "deepseek"},
		{"mistral-large", "mistral"},
		{"some-random-model", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := inferModelFamily(tt.model)
			if got != tt.want {
				t.Errorf("inferModelFamily(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestHTTPRunner_Interface(t *testing.T) {
	r := NewHTTPRunner("id", "name", HTTPRunnerConfig{Endpoint: "http://localhost"})
	if r.Type() != RunnerTypeHTTP {
		t.Errorf("Type: got %s, want %s", r.Type(), RunnerTypeHTTP)
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
	if m.Runtime != "http" {
		t.Errorf("Runtime: got %s, want http", m.Runtime)
	}
}
