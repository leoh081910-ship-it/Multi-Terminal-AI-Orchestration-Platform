// Package runner provides the universal AI Agent execution interface.
package runner

// CapabilityManifest is the self-described capability statement for an Agent.
// Agents publish their capabilities at registration time and the Registry
// refreshes them periodically via Runner.GetCapabilities.
//
// The Manifest replaces the old flat "specialties" string array with a
// structured, machine-readable description that enables intelligent routing.
type CapabilityManifest struct {
	// Name is the Agent's display name.
	Name string `json:"name"`

	// TaskTypes declares which task types this Agent handles natively.
	// Entries should be lowercase task-type strings: "feature", "bugfix",
	// "code-review", "reverse", "refactor", "documentation", etc.
	// An empty list means the Agent handles all types.
	TaskTypes []string `json:"task_types,omitempty"`

	// MaxInputSize is the maximum task prompt size in bytes this Agent accepts.
	// Zero means no limit.
	MaxInputSize int64 `json:"max_input_size,omitempty"`

	// OutputGlobs are the file patterns the Agent is known to produce.
	// Used by routing to sanity-check task expectations.
	OutputGlobs []string `json:"output_globs,omitempty"`

	// ModelFamily identifies the underlying model: "claude", "gpt", "gemini",
	// "llama", "qwen", "local", or a specific model like "claude-sonnet-4".
	ModelFamily string `json:"model_family,omitempty"`

	// ContextWindow is the maximum context window in tokens.
	// Used to filter Agents that can't handle very long prompts.
	ContextWindow int `json:"context_window,omitempty"`

	// SupportsThinking indicates whether the Agent supports a
	// chain-of-thought or thinking mode (e.g., Claude extended thinking).
	SupportsThinking bool `json:"supports_thinking,omitempty"`

	// SupportsMultimodal indicates the Agent can process images/audio/video.
	SupportsMultimodal bool `json:"supports_multimodal,omitempty"`

	// SupportsStreaming indicates the Agent can stream responses incrementally.
	SupportsStreaming bool `json:"supports_streaming,omitempty"`

	// LatencyP50 is the typical round-trip latency in milliseconds.
	LatencyP50 int `json:"latency_p50_ms,omitempty"`

	// CostPer1KInput is the cost per 1000 input tokens (in cents).
	CostPer1KInput float64 `json:"cost_per_1k_input_cents,omitempty"`

	// CostPer1KOutput is the cost per 1000 output tokens (in cents).
	CostPer1KOutput float64 `json:"cost_per_1k_output_cents,omitempty"`

	// Runtime is the execution runtime type: "cli", "http", "mcp".
	Runtime string `json:"runtime,omitempty"`

	// Version is the Agent/Runner version string.
	Version string `json:"version,omitempty"`

	// Tags are arbitrary freeform labels for filtering.
	Tags []string `json:"tags,omitempty"`
}

// DefaultCLIManifest returns a baseline CapabilityManifest for CLI-based Agents
// (e.g., Claude Code CLI). Callers can modify fields before registering.
func DefaultCLIManifest(name string) *CapabilityManifest {
	return &CapabilityManifest{
		Name:              name,
		TaskTypes:          []string{"feature", "bugfix", "refactor", "documentation", "code-review", "reverse"},
		ModelFamily:        "claude",
		SupportsThinking:   true,
		SupportsMultimodal: true,
		ContextWindow:      DefaultClaudeContextWindow,
		Runtime:            string(RunnerTypeCLI),
	}
}

// DefaultHTTPManifest returns a baseline CapabilityManifest for HTTP API Agents.
func DefaultHTTPManifest(name, modelFamily string) *CapabilityManifest {
	return &CapabilityManifest{
		Name:       name,
		ModelFamily: modelFamily,
		Runtime:    string(RunnerTypeHTTP),
	}
}

// DefaultMCPManifest returns a baseline CapabilityManifest for MCP Protocol Agents.
func DefaultMCPManifest(name, modelFamily string) *CapabilityManifest {
	return &CapabilityManifest{
		Name:       name,
		ModelFamily: modelFamily,
		Runtime:    string(RunnerTypeMCP),
	}
}

// SupportsTaskType returns true if the Manifest covers the given task type.
func (m *CapabilityManifest) SupportsTaskType(taskType string) bool {
	if len(m.TaskTypes) == 0 {
		return true // no restriction means all types
	}
	for _, t := range m.TaskTypes {
		if t == taskType {
			return true
		}
	}
	return false
}