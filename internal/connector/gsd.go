// Package connector implements the GSD (Get Shit Done) connector.
// It bridges GSD-style PLAN files with the orchestration platform.
package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GSDConnector implements the GSD connector for PLAN file integration.
type GSDConnector struct {
	// config holds the connector configuration
	config Config
	// planDir is the directory containing PLAN files
	planDir string
	// workspaceDir is the working directory for task execution
	workspaceDir string
}

// Config holds GSD connector configuration.
type Config struct {
	// PlanDir is the directory containing PLAN files
	PlanDir string `json:"plan_dir" yaml:"plan_dir"`
	// WorkspaceDir is the working directory
	WorkspaceDir string `json:"workspace_dir" yaml:"workspace_dir"`
	// DefaultPriority is the default task priority
	DefaultPriority int `json:"default_priority" yaml:"default_priority"`
	// DefaultTransport is the default transport type
	DefaultTransport string `json:"default_transport" yaml:"default_transport"`
}

// PlanFile represents a GSD PLAN file structure.
type PlanFile struct {
	// Objective describes the plan's goal
	Objective string `yaml:"objective"`
	// Phase is the current phase (e.g., "01-foundation")
	Phase string `yaml:"phase"`
	// Wave is the wave number for batching
	Wave int `yaml:"wave"`
	// Tasks is the list of tasks in this plan
	Tasks []PlanTask `yaml:"tasks"`
	// FilesModified lists expected file changes
	FilesModified []string `yaml:"files_modified"`
	// Dependencies lists external dependencies
	Dependencies []string `yaml:"dependencies"`
}

// PlanTask represents a task within a PLAN file.
type PlanTask struct {
	// ID is the task identifier
	ID string `yaml:"id"`
	// Type is the task type (e.g., "implementation", "refactor", "reverse")
	Type string `yaml:"type"`
	// Description describes what needs to be done
	Description string `yaml:"description"`
	// FilesToModify lists files this task will modify
	FilesToModify []string `yaml:"files_to_modify"`
	// DependsOn lists task IDs this task depends on
	DependsOn []string `yaml:"depends_on"`
	// AcceptanceCriteria defines completion criteria
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
	// Context provides additional execution context
	Context map[string]interface{} `yaml:"context"`
}

// NewGSDConnector creates a new GSD connector.
func NewGSDConnector(config Config) (*GSDConnector, error) {
	// Validate configuration
	if config.PlanDir == "" {
		return nil, fmt.Errorf("plan_dir is required")
	}
	if config.WorkspaceDir == "" {
		return nil, fmt.Errorf("workspace_dir is required")
	}
	if config.DefaultTransport == "" {
		config.DefaultTransport = "cli"
	}

	return &GSDConnector{
		config:       config,
		planDir:      config.PlanDir,
		workspaceDir: config.WorkspaceDir,
	}, nil
}

// DiscoverTasks discovers tasks from PLAN files.
// REQUIRES (CONN-01): Connector discovers tasks from external system (PLAN files).
func (c *GSDConnector) DiscoverTasks(ctx context.Context) ([]TaskCard, error) {
	// Find all PLAN files in the plan directory
	planFiles, err := c.findPlanFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find plan files: %w", err)
	}

	var tasks []TaskCard

	for _, planPath := range planFiles {
		plan, err := c.parsePlanFile(planPath)
		if err != nil {
			continue // Skip invalid plan files
		}

		// Generate tasks from plan
		planTasks := c.convertPlanToTasks(plan, planPath)
		tasks = append(tasks, planTasks...)
	}

	return tasks, nil
}

// HydrateContext hydrates task context with GSD-specific information.
// REQUIRES (CONN-02): Connector provides additional context for task execution.
func (c *GSDConnector) HydrateContext(ctx context.Context, taskID string, baseContext map[string]interface{}) (map[string]interface{}, error) {
	hydrated := make(map[string]interface{})

	// Copy base context
	for k, v := range baseContext {
		hydrated[k] = v
	}

	// Add GSD-specific context
	hydrated["gsd_plan_dir"] = c.planDir
	hydrated["gsd_workspace_dir"] = c.workspaceDir
	hydrated["gsd_timestamp"] = time.Now().UTC()

	// Add connector type info
	hydrated["connector_type"] = c.GetConnectorType()
	hydrated["connector_version"] = c.GetConnectorVersion()

	return hydrated, nil
}

// AckResult acknowledges task completion to the GSD system.
// REQUIRES (CONN-03): Connector acknowledges task results.
func (c *GSDConnector) AckResult(ctx context.Context, result TaskResult) error {
	// Write result acknowledgment to a tracking file
	ackFile := filepath.Join(c.workspaceDir, "acks", result.TaskID+".json")

	if err := os.MkdirAll(filepath.Dir(ackFile), 0755); err != nil {
		return fmt.Errorf("failed to create ack directory: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(ackFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write ack file: %w", err)
	}

	return nil
}

// WriteBackArtifacts writes generated artifacts back to the GSD system.
// REQUIRES (CONN-03): Connector handles artifact write-back.
func (c *GSDConnector) WriteBackArtifacts(ctx context.Context, taskID string, artifacts []Artifact) error {
	// Write artifacts to the workspace artifacts directory
	artifactDir := filepath.Join(c.workspaceDir, "artifacts", taskID)

	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	for _, artifact := range artifacts {
		artifactPath := filepath.Join(artifactDir, artifact.Path)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
			return fmt.Errorf("failed to create artifact parent directory: %w", err)
		}

		// Write artifact content
		if err := os.WriteFile(artifactPath, artifact.Content, 0644); err != nil {
			return fmt.Errorf("failed to write artifact file: %w", err)
		}
	}

	// Update GSD planning documents (SUMMARY, STATE, ROADMAP, VERIFICATION)
	if err := c.updatePlanningDocuments(taskID, artifacts); err != nil {
		return fmt.Errorf("failed to update planning documents: %w", err)
	}

	return nil
}

// GetConnectorType returns the GSD connector type.
func (c *GSDConnector) GetConnectorType() string {
	return "gsd"
}

// GetConnectorVersion returns the GSD connector version.
func (c *GSDConnector) GetConnectorVersion() string {
	return "1.0.0"
}

// HealthCheck checks if the GSD connector is healthy.
func (c *GSDConnector) HealthCheck(ctx context.Context) error {
	// Check if plan directory exists and is accessible
	if _, err := os.Stat(c.planDir); err != nil {
		return fmt.Errorf("plan directory not accessible: %w", err)
	}

	// Check if workspace directory exists and is accessible
	if _, err := os.Stat(c.workspaceDir); err != nil {
		return fmt.Errorf("workspace directory not accessible: %w", err)
	}

	return nil
}

// Helper methods

// findPlanFiles finds all PLAN files in the plan directory.
func (c *GSDConnector) findPlanFiles() ([]string, error) {
	var planFiles []string

	err := filepath.Walk(c.planDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if file is a PLAN file (matches PLAN.yaml, PLAN.yml, or *-PLAN.yaml)
		if !info.IsDir() {
			name := strings.ToLower(info.Name())
			if name == "plan.yaml" || name == "plan.yml" ||
				strings.HasSuffix(name, "-plan.yaml") || strings.HasSuffix(name, "-plan.yml") {
				planFiles = append(planFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return planFiles, nil
}

// parsePlanFile parses a PLAN YAML file.
func (c *GSDConnector) parsePlanFile(path string) (*PlanFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan PlanFile
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan file: %w", err)
	}

	return &plan, nil
}

// convertPlanToTasks converts a PLAN file to task cards.
func (c *GSDConnector) convertPlanToTasks(plan *PlanFile, planPath string) []TaskCard {
	var tasks []TaskCard

	dispatchRef := generateDispatchRef()

	for _, task := range plan.Tasks {
		taskCard := TaskCard{
			ID:                 task.ID,
			DispatchRef:        dispatchRef,
			Source:             "gsd",
			SourceRef:          planPath,
			Type:               task.Type,
			Objective:          task.Description,
			Context:            task.Context,
			FilesToRead:        task.FilesToModify,
			FilesToModify:      task.FilesToModify,
			AcceptanceCriteria: task.AcceptanceCriteria,
			Wave:               plan.Wave,
			Priority:           c.config.DefaultPriority,
			Transport:          c.config.DefaultTransport,
		}

		// Add relations
		for _, depID := range task.DependsOn {
			taskCard.Relations = append(taskCard.Relations, Relation{
				TaskID: depID,
				Type:   "depends_on",
				Reason: "plan dependency",
			})
		}

		tasks = append(tasks, taskCard)
	}

	return tasks
}

// updatePlanningDocuments updates GSD planning documents after task completion.
func (c *GSDConnector) updatePlanningDocuments(taskID string, artifacts []Artifact) error {
	// This would update SUMMARY, STATE, ROADMAP, VERIFICATION files
	// Implementation depends on the specific GSD format
	return nil
}

// generateDispatchRef generates a unique dispatch reference.
func generateDispatchRef() string {
	// Simple implementation - in production, use proper UUID
	return fmt.Sprintf("gsd-%d", time.Now().UnixNano())
}
