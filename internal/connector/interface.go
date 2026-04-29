// Package connector defines the interface for agent connectors.
// Connectors bridge external systems (like GSD) with the orchestration platform.
package connector

import (
	"context"
	"fmt"
	"time"
)

// Connector is the interface that all agent connectors must implement.
// It defines the contract for discovering tasks, hydrating context,
// acknowledging results, and writing back artifacts.
type Connector interface {
	// DiscoverTasks discovers new tasks from the external system.
	// REQUIRES (CONN-01): Connector discovers tasks from external system.
	// Called periodically by the platform to find new work.
	DiscoverTasks(ctx context.Context) ([]TaskCard, error)

	// HydrateContext hydrates task context with additional information.
	// REQUIRES (CONN-02): Connector provides additional context for task execution.
	// Called before task execution to enrich the task with connector-specific data.
	HydrateContext(ctx context.Context, taskID string, baseContext map[string]interface{}) (map[string]interface{}, error)

	// AckResult acknowledges task completion to the external system.
	// REQUIRES (CONN-03): Connector acknowledges task results.
	// Called after task completion to notify the external system.
	AckResult(ctx context.Context, result TaskResult) error

	// WriteBackArtifacts writes generated artifacts back to the external system.
	// REQUIRES (CONN-03): Connector handles artifact write-back.
	// Called after merge to update external systems with final artifacts.
	WriteBackArtifacts(ctx context.Context, taskID string, artifacts []Artifact) error

	// GetConnectorType returns the type identifier for this connector.
	GetConnectorType() string

	// GetConnectorVersion returns the version of this connector.
	GetConnectorVersion() string

	// HealthCheck checks if the connector is healthy and able to communicate with the external system.
	HealthCheck(ctx context.Context) error
}

// TaskCard represents a task discovered by a connector.
// This is the minimal set of fields required for task creation.
type TaskCard struct {
	// ID is the unique task identifier.
	ID string `json:"id"`

	// DispatchRef is the platform-generated dispatch reference.
	DispatchRef string `json:"dispatch_ref"`

	// Source identifies the originating system (e.g., "gsd", "github", "jira").
	Source string `json:"source"`

	// SourceRef is the original identifier from the source system.
	SourceRef string `json:"source_ref"`

	// Type is the task type (e.g., "feature", "bugfix", "refactor", "reverse_static_c_rebuild").
	Type string `json:"type"`

	// Objective describes what needs to be accomplished.
	Objective string `json:"objective"`

	// Context provides additional context for execution.
	Context map[string]interface{} `json:"context"`

	// FilesToRead lists files that should be examined (glob patterns supported).
	FilesToRead []string `json:"files_to_read"`

	// FilesToModify lists files that will be modified (glob patterns supported).
	FilesToModify []string `json:"files_to_modify"`

	// AcceptanceCriteria defines when the task is considered complete.
	AcceptanceCriteria []string `json:"acceptance_criteria"`

	// Relations defines dependencies and conflicts with other tasks.
	Relations []Relation `json:"relations"`

	// Wave is the wave number for batching (optional, auto-assigned if not set).
	Wave int `json:"wave"`

	// Priority indicates task priority (higher = more urgent).
	Priority int `json:"priority"`

	// Transport specifies the transport type ("cli" or "api").
	Transport string `json:"transport"`
}

// Relation represents a relationship between tasks.
type Relation struct {
	// TaskID is the related task ID.
	TaskID string `json:"task_id"`

	// Type is the relation type: "depends_on" or "conflicts_with".
	Type string `json:"type"`

	// Reason explains why this relation exists.
	Reason string `json:"reason,omitempty"`
}

// TaskResult represents the result of task execution.
type TaskResult struct {
	// TaskID is the task identifier.
	TaskID string `json:"task_id"`

	// Success indicates if the task completed successfully.
	Success bool `json:"success"`

	// State is the final task state.
	State string `json:"state"`

	// Artifacts lists the produced artifacts.
	Artifacts []Artifact `json:"artifacts,omitempty"`

	// Output contains execution output.
	Output string `json:"output,omitempty"`

	// Error contains error information if failed.
	Error string `json:"error,omitempty"`

	// CompletedAt is when the task completed.
	CompletedAt time.Time `json:"completed_at"`
}

// Artifact represents a file artifact.
type Artifact struct {
	// Path is the relative path.
	Path string `json:"path"`

	// Content is the file content (may be empty for large files).
	Content []byte `json:"content,omitempty"`

	// Size is the file size in bytes.
	Size int64 `json:"size"`

	// Checksum is the file checksum.
	CheckSum string `json:"checksum,omitempty"`
}

// ConnectorRegistry manages available connectors.
type ConnectorRegistry struct {
	connectors map[string]Connector
}

// NewConnectorRegistry creates a new connector registry.
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		connectors: make(map[string]Connector),
	}
}

// Register registers a connector.
func (r *ConnectorRegistry) Register(connector Connector) error {
	connectorType := connector.GetConnectorType()
	if _, exists := r.connectors[connectorType]; exists {
		return fmt.Errorf("connector type '%s' already registered", connectorType)
	}
	r.connectors[connectorType] = connector
	return nil
}

// Get retrieves a connector by type.
func (r *ConnectorRegistry) Get(connectorType string) (Connector, error) {
	connector, exists := r.connectors[connectorType]
	if !exists {
		return nil, fmt.Errorf("connector type '%s' not found", connectorType)
	}
	return connector, nil
}

// List returns all registered connector types.
func (r *ConnectorRegistry) List() []string {
	types := make([]string, 0, len(r.connectors))
	for t := range r.connectors {
		types = append(types, t)
	}
	return types
}

// HealthCheck checks all registered connectors.
func (r *ConnectorRegistry) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for t, c := range r.connectors {
		results[t] = c.HealthCheck(ctx)
	}
	return results
}
