// Package runner provides the universal AI Agent execution interface.
// HTTPRunner implementation for HTTP API-based agents.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

const defaultHTTPRunnerTimeout = 5 * time.Minute

// HTTPRunnerConfig holds the configuration for an HTTP-based Agent Runner.
// This struct is serialized as JSON and stored in agent.runner_config.
type HTTPRunnerConfig struct {
	// Endpoint is the full URL of the Agent's HTTP API.
	Endpoint string `json:"endpoint"`

	// Method is the HTTP method to use. Defaults to "POST".
	Method string `json:"method,omitempty"`

	// Headers are extra HTTP headers to include in every request.
	// Common values: Content-Type, Authorization.
	Headers map[string]string `json:"headers,omitempty"`

	// AuthToken is an optional bearer token or API key.
	// Supports ${ENV_VAR} syntax for environment variable expansion.
	AuthToken string `json:"auth_token,omitempty"`

	// BodyTemplate is a Go text/template string used to render the request body
	// from a RunnerTask. The template receives a RunnerTask as its data object.
	// Example: `{"model":"{{.Context.model}}","messages":[{"role":"user","content":{{.Context.prompt | toJSON}}}]}`
	BodyTemplate string `json:"body_template,omitempty"`

	// OutputPath is a dot-notation path to extract the result from the JSON response.
	// Example: "choices[0].message.content" extracts {"choices": [{"message": {"content": "..."}}]}
	OutputPath string `json:"output_path,omitempty"`

	// TimeoutMs is the request timeout in milliseconds. Defaults to 300000 (5min).
	TimeoutMs int64 `json:"timeout_ms,omitempty"`

	// Model is the model name, used to populate CapabilityManifest.
	Model string `json:"model,omitempty"`
}

// expandEnv expands ${VAR} patterns in a string using os.Getenv.
func expandEnv(s string) string {
	return os.Expand(s, func(k string) string {
		return os.Getenv(k)
	})
}

// HTTPRunner executes tasks by calling an HTTP API endpoint.
// It supports templated request bodies, dot-notation output extraction,
// and environment variable expansion for secrets.
type HTTPRunner struct {
	id      string
	name    string
	config  HTTPRunnerConfig
	client  *http.Client
	manifest *CapabilityManifest
}

// NewHTTPRunner creates a new HTTPRunner with the given configuration.
func NewHTTPRunner(id, name string, config HTTPRunnerConfig) *HTTPRunner {
	timeout := time.Duration(config.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = defaultHTTPRunnerTimeout
	}

	cli := &http.Client{Timeout: timeout}

	manifest := DefaultHTTPManifest(name, inferModelFamily(config.Model))
	if config.Model != "" {
		manifest.ModelFamily = inferModelFamily(config.Model)
	}
	// Default to all task types; caller can override
	manifest.TaskTypes = DefaultTaskTypes

	return &HTTPRunner{
		id:       id,
		name:     name,
		config:   config,
		client:   cli,
		manifest: manifest,
	}
}

// Type implements Runner.
func (r *HTTPRunner) Type() RunnerType { return RunnerTypeHTTP }

// String implements Runner.
func (r *HTTPRunner) String() string { return r.id }

// ID returns the Runner's unique identifier.
func (r *HTTPRunner) ID() string { return r.id }

// Execute runs the task by sending an HTTP request to the configured endpoint.
func (r *HTTPRunner) Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error) {
	start := time.Now()

	// Build the request body from the template
	bodyBytes, err := r.renderBody(task)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("body template render failed: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	// Determine HTTP method
	method := r.config.Method
	if method == "" {
		method = "POST"
	}

	// Build the request
	req, err := http.NewRequestWithContext(ctx, method, r.config.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to create request: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range r.config.Headers {
		req.Header.Set(k, expandEnv(v))
	}

	// Set Authorization header if AuthToken is configured
	if r.config.AuthToken != "" {
		req.Header.Set("Authorization", expandEnv(r.config.AuthToken))
	}

	// Execute the request
	resp, err := r.client.Do(req)
	if err != nil {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("request failed: %v", err),
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
			Error:    fmt.Sprintf("failed to read response body: %v", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}, err
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return &RunnerResult{
			Success:  false,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			ExitCode: resp.StatusCode,
			Output:   string(respBody),
			Duration: time.Since(start),
		}, nil
	}

	// Extract output from response using OutputPath
	output, err := r.extractOutput(respBody)
	if err != nil {
		// If output extraction fails, use the raw body as output
		output = string(respBody)
	}

	// Build runner stats from response headers if available
	stats := RunnerStats{}
	if model := resp.Header.Get("X-Model-Used"); model != "" {
		stats.ModelUsed = model
	}

	return &RunnerResult{
		Success:  true,
		Output:   output,
		ExitCode: 0,
		Duration: time.Since(start),
		Stats:    stats,
	}, nil
}

// renderBody renders the body template with the RunnerTask data.
func (r *HTTPRunner) renderBody(task RunnerTask) ([]byte, error) {
	if r.config.BodyTemplate == "" {
		// No template — use a default payload based on RunnerTask.Context
		payload := map[string]interface{}{
			"task_id":   task.ID,
			"task_type": task.Type,
		}
		if task.Context != nil {
			for k, v := range task.Context {
				payload[k] = v
			}
		}
		return json.Marshal(payload)
	}

	// Render the template
	tmpl, err := template.New("body").Parse(r.config.BodyTemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid body template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, task); err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	return buf.Bytes(), nil
}

// extractOutput extracts a value from JSON using dot-notation path.
// For example: "choices[0].message.content" navigates {"choices": [{"message": {"content": "..."}}]}
func (r *HTTPRunner) extractOutput(data []byte) (string, error) {
	if r.config.OutputPath == "" {
		return string(data), nil
	}

	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return string(data), nil
	}

	val, err := NavigateJSON(root, r.config.OutputPath)
	if err != nil {
		return string(data), nil
	}

	// Convert the extracted value to string
	switch v := val.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

// HealthCheck implements Runner.
// It sends a GET request to the endpoint to verify the Agent is reachable.
func (r *HTTPRunner) HealthCheck(ctx context.Context) error {
	// Try a simple GET request first
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.config.Endpoint, nil)
	if err != nil {
		return fmt.Errorf("health check request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.config.AuthToken != "" {
		req.Header.Set("Authorization", expandEnv(r.config.AuthToken))
	}
	for k, v := range r.config.Headers {
		req.Header.Set(k, expandEnv(v))
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check returned HTTP %d", resp.StatusCode)
	}

	return nil
}

// Cancel implements Runner.
// HTTP cancellation is best-effort — sends a DELETE or POST to a cancel endpoint
// if the runner config specifies one. Otherwise returns nil.
func (r *HTTPRunner) Cancel(ctx context.Context, taskID string) error {
	// Build cancel URL — try appending /cancel/{taskID}
	cancelURL := r.config.Endpoint + "/cancel/" + taskID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, cancelURL, nil)
	if err != nil {
		return nil // best-effort, don't fail on bad URL
	}
	if r.config.AuthToken != "" {
		req.Header.Set("Authorization", expandEnv(r.config.AuthToken))
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil // best-effort
	}
	defer resp.Body.Close()
	return nil
}

// GetCapabilities implements Runner.
// Returns the manifest built at construction time.
func (r *HTTPRunner) GetCapabilities(ctx context.Context) (*CapabilityManifest, error) {
	return r.manifest, nil
}

// SetManifest updates the Runner's CapabilityManifest.
// Call this after discovering capabilities dynamically.
func (r *HTTPRunner) SetManifest(m *CapabilityManifest) {
	r.manifest = m
}

// inferModelFamily guesses the model family from the model name string.
func inferModelFamily(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.Contains(model, "gpt"):
		return "gpt"
	case strings.Contains(model, "claude"):
		return "claude"
	case strings.Contains(model, "gemini"):
		return "gemini"
	case strings.Contains(model, "llama"):
		return "llama"
	case strings.Contains(model, "qwen"):
		return "qwen"
	case strings.Contains(model, "deepseek"):
		return "deepseek"
	case strings.Contains(model, "mistral"):
		return "mistral"
	default:
		return "unknown"
	}
}

// NavigateJSON navigates into a parsed JSON value using dot-notation.
// Supports map["key"] and array[index] access.
// Examples:
//   NavigateJSON(data, "choices[0].message.content")
//   NavigateJSON(data, "data.result")
func NavigateJSON(root interface{}, path string) (interface{}, error) {
	segments := strings.Split(path, ".")
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Handle array index: "choices[0]"
		idx := strings.IndexByte(seg, '[')
		if idx >= 0 {
			key := seg[:idx]
			bracketClose := strings.IndexByte(seg, ']')
			if bracketClose < idx {
				return nil, fmt.Errorf("invalid array syntax: %s", seg)
			}
			indexStr := seg[idx+1 : bracketClose]
			rest := ""
			if bracketClose+1 < len(seg) {
				rest = seg[bracketClose+1:]
			}

			var val interface{}
			if key != "" {
				m, ok := root.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("cannot index non-object at %s", key)
				}
				val = m[key]
			} else {
				val = root
			}

			// Parse array index
			var idx2 int
			if _, err := fmt.Sscanf(indexStr, "%d", &idx2); err != nil {
				return nil, fmt.Errorf("invalid array index: %s", indexStr)
			}

			arr, ok := val.([]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot index non-array at %s", seg)
			}
			if idx2 < 0 || idx2 >= len(arr) {
				return nil, fmt.Errorf("array index out of bounds: %d", idx2)
			}
			root = arr[idx2]

			if rest != "" {
				// Continue with remaining path
				seg = rest
			} else {
				continue
			}
		}

		// Handle map access
		m, ok := root.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot access field on non-object at %s", seg)
		}
		val, ok := m[seg]
		if !ok {
			return nil, fmt.Errorf("key not found: %s", seg)
		}
		root = val
	}
	return root, nil
}