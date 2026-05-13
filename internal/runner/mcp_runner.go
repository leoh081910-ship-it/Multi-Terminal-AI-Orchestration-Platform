// Package runner provides the universal AI Agent execution interface.
// MCPRunner implementation for MCP (Model Context Protocol) agents.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMCPRunnerTimeout = 5 * time.Minute

// MCPRunnerConfig holds the configuration for an MCP-based Agent Runner.
// MCP uses JSON-RPC over HTTP. This config describes how to reach the MCP server.
type MCPRunnerConfig struct {
	// Endpoint is the full URL of the MCP server's HTTP endpoint.
	Endpoint string `json:"endpoint"`

	// Transport is the MCP transport type. Phase 6 supports HTTP only.
	Transport string `json:"transport,omitempty"`

	// Headers are extra HTTP headers to include in every request.
	Headers map[string]string `json:"headers,omitempty"`

	// AuthToken is an optional bearer token for MCP server authentication.
	// Supports ${ENV_VAR} syntax for environment variable expansion.
	AuthToken string `json:"auth_token,omitempty"`

	// TimeoutMs is the request timeout in milliseconds. Defaults to 300000 (5min).
	TimeoutMs int64 `json:"timeout_ms,omitempty"`

	// ToolName is the MCP tool invoked via tools/call. Defaults to execute_task.
	ToolName string `json:"tool_name,omitempty"`

	// ToolsEnabled indicates whether this runner uses MCP tools.
	ToolsEnabled bool `json:"tools_enabled,omitempty"`
}

// Validate checks that the MCPRunnerConfig has all required fields and sane values.
func (c MCPRunnerConfig) Validate() error {
	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint is not a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("endpoint must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("endpoint must include a host")
	}

	transport := strings.TrimSpace(c.Transport)
	switch transport {
	case "", "http":
	case "sse":
		return fmt.Errorf("transport sse is not supported in Phase 6; use http")
	default:
		return fmt.Errorf("transport must be http")
	}

	if c.ToolName != "" && strings.TrimSpace(c.ToolName) == "" {
		return fmt.Errorf("tool_name is required")
	}

	if c.TimeoutMs < 0 {
		return fmt.Errorf("timeout_ms must be non-negative, got %d", c.TimeoutMs)
	}
	return nil
}

// mcpRequest is a JSON-RPC 2.0 request.
type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID     any             `json:"id"`
}

// mcpResponse is a JSON-RPC 2.0 response.
type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *mcpError      `json:"error,omitempty"`
	ID      any            `json:"id"`
}

// mcpError is a JSON-RPC 2.0 error object.
type mcpError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// mcpInitializeParams is the params for the initialize method.
type mcpInitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// MCPRunner executes tasks via the Model Context Protocol.
// MCP is a JSON-RPC 2.0 protocol over HTTP. This runner supports both
// request-response and SSE-based MCP servers.
type MCPRunner struct {
	id       string
	name     string
	config   MCPRunnerConfig
	client   *http.Client
	manifest *CapabilityManifest
}

// NewMCPRunner creates a new MCPRunner with the given configuration.
func NewMCPRunner(id, name string, config MCPRunnerConfig) *MCPRunner {
	timeout := time.Duration(config.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = defaultMCPRunnerTimeout
	}

	cli := &http.Client{Timeout: timeout}

	manifest := DefaultMCPManifest(name, "mcp")
	manifest.TaskTypes = DefaultTaskTypes

	return &MCPRunner{
		id:       id,
		name:     name,
		config:   config,
		client:   cli,
		manifest: manifest,
	}
}

// Type implements Runner.
func (r *MCPRunner) Type() RunnerType { return RunnerTypeMCP }

// String implements Runner.
func (r *MCPRunner) String() string { return r.id }

// ID returns the Runner's unique identifier.
func (r *MCPRunner) ID() string { return r.id }

// Execute sends a task to the MCP server as a JSON-RPC request.
// It calls the MCP server's tools/call method with MCP-standard name/arguments params.
func (r *MCPRunner) Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error) {
	start := time.Now()

	arguments := map[string]interface{}{
		"task_id":   task.ID,
		"task_type": task.Type,
	}
	if task.Context != nil {
		for k, v := range task.Context {
			arguments[k] = v
		}
	}
	if task.Workspace.Path != "" {
		arguments["workspace_path"] = task.Workspace.Path
	}
	if len(task.FilesToModify) > 0 {
		arguments["files_to_modify"] = task.FilesToModify
	}

	params := map[string]interface{}{
		"name":      r.toolName(),
		"arguments": arguments,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to marshal params: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
		}

	// Build the JSON-RPC request
	rpcReq := mcpRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      1,
	}

	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to marshal RPC request: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to create request: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	r.applyHeaders(req)

	// Execute the request
	resp, err := r.client.Do(req)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("MCP request failed: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to read MCP response: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("MCP HTTP %d: %s", resp.StatusCode, string(respBody)),
			ExitCode: resp.StatusCode,
			Output:   string(respBody),
			Duration: time.Since(start),
		}, nil
	}

	// Parse JSON-RPC response
	var rpcResp mcpResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to parse MCP response: %v", err),
			ExitCode: -1,
			Output:   string(respBody),
			Duration: time.Since(start),
		}, err
	}

	// Check JSON-RPC error
	if rpcResp.Error != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("MCP RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message),
			ExitCode: -1,
			Duration: time.Since(start),
		}, nil
	}

	// Extract result as output string
	var output string
	if rpcResp.Result != nil {
		// Try to extract text content from result
		// MCP tools/call returns { "content": [{ "type": "text", "text": "..." }] }
		var result map[string]interface{}
		if err := json.Unmarshal(rpcResp.Result, &result); err == nil {
			if content, ok := result["content"].([]interface{}); ok {
				var sb strings.Builder
				for _, item := range content {
					if m, ok := item.(map[string]interface{}); ok {
						if text, ok := m["text"].(string); ok {
							sb.WriteString(text)
						}
					}
				}
				output = sb.String()
			}
		}
		if output == "" {
			// Fall back to raw JSON
			output = string(rpcResp.Result)
		}
	}

	return &RunnerResult{
		Success:  true,
		Output:   output,
		ExitCode: 0,
		Duration: time.Since(start),
		Stats:    RunnerStats{RoundTrips: 1},
	}, nil
}

// HealthCheck implements Runner.
// It sends an MCP "initialize" request to verify the server is reachable.
func (r *MCPRunner) HealthCheck(ctx context.Context) error {
	initParams := mcpInitializeParams{
		ProtocolVersion: "2024-11-05",
	}
	initParams.Capabilities = struct{}{}
	initParams.ClientInfo.Name = "ai-orchestration-platform"
	initParams.ClientInfo.Version = "3.0"

	paramsJSON, _ := json.Marshal(initParams)
	rpcReq := mcpRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  paramsJSON,
		ID:      0,
	}

	body, _ := json.Marshal(rpcReq)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("health check request creation failed: %w", err)
	}
	r.applyHeaders(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("MCP health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("MCP health check returned HTTP %d", resp.StatusCode)
	}

	return nil
}

// Cancel implements Runner.
// MCP cancellation is best-effort — sends a "cancel" notification.
func (r *MCPRunner) Cancel(ctx context.Context, taskID string) error {
	cancelReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "cancel",
		"params": map[string]interface{}{
			"task_id": taskID,
		},
		"id": nil, // notifications don't have an ID
	}
	body, _ := json.Marshal(cancelReq)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	r.applyHeaders(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	return nil
}

// GetCapabilities implements Runner.
// Returns the manifest built at construction time.
// On first call, it attempts to discover tools from the MCP server.
func (r *MCPRunner) GetCapabilities(ctx context.Context) (*CapabilityManifest, error) {
	// Try to discover tools from the MCP server
	tools, err := r.discoverTools(ctx)
	if err == nil && len(tools) > 0 && r.manifest != nil {
		// Update manifest with discovered task types
		taskTypes := make([]string, 0, len(tools))
		for _, tool := range tools {
			taskTypes = append(taskTypes, strings.ToLower(tool))
		}
		r.manifest.TaskTypes = taskTypes
	}

	return r.manifest, nil
}

func (r *MCPRunner) toolName() string {
	if strings.TrimSpace(r.config.ToolName) == "" {
		return "execute_task"
	}
	return strings.TrimSpace(r.config.ToolName)
}

func (r *MCPRunner) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if r.config.AuthToken != "" {
		req.Header.Set("Authorization", expandEnv(r.config.AuthToken))
	}
	for k, v := range r.config.Headers {
		req.Header.Set(k, expandEnv(v))
	}
}

// discoverTools queries the MCP server for available tools.
// It sends a tools/list request and returns the tool names.
func (r *MCPRunner) discoverTools(ctx context.Context) ([]string, error) {
	rpcReq := mcpRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		Params:  []byte("{}"),
		ID:      99,
	}
	body, _ := json.Marshal(rpcReq)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.applyHeaders(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tools/list returned HTTP %d", resp.StatusCode)
	}

	var rpcResp mcpResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return nil, nil
	}

	// Parse result to extract tool names
	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, err
	}

	names := make([]string, len(result.Tools))
	for i, t := range result.Tools {
		names[i] = t.Name
	}
	return names, nil
}

// SetManifest updates the Runner's CapabilityManifest.
func (r *MCPRunner) SetManifest(m *CapabilityManifest) {
	r.manifest = m
}