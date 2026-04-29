package reverse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TaskType represents the reverse engineering task type.
type TaskType string

const (
	// TaskTypeStaticCRebuild rebuilds static C code with verification loops.
	TaskTypeStaticCRebuild TaskType = "reverse_static_c_rebuild"

	defaultMaxLoopIterations = 50
)

// ReverseTaskConfig holds configuration for a reverse engineering task.
type ReverseTaskConfig struct {
	TaskID              string                 `json:"task_id"`
	TaskType            TaskType               `json:"task_type"`
	TargetSOPath        string                 `json:"target_so_path"`
	IDAMCPEndpoint      string                 `json:"ida_mcp_endpoint"`
	FridaHookSpec       map[string]interface{} `json:"frida_hook_spec"`
	OracleInputSpec     map[string]interface{} `json:"oracle_input_spec"`
	OracleOutputRef     string                 `json:"oracle_output_ref"`
	AnalysisStateMDPath string                 `json:"analysis_state_md_path"`
	FinalArtifactPath   string                 `json:"final_artifact_path"`
	ArtifactBasePath    string                 `json:"artifact_base_path"`
	MaxLoopIterations   int                    `json:"max_loop_iterations"`
}

// Validate validates the reverse task configuration.
func (c *ReverseTaskConfig) Validate() error {
	if c.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if c.TaskType != TaskTypeStaticCRebuild {
		return fmt.Errorf("unsupported task type: %s", c.TaskType)
	}
	if c.TargetSOPath == "" {
		return fmt.Errorf("target_so_path is required")
	}
	if c.IDAMCPEndpoint == "" {
		return fmt.Errorf("ida_mcp_endpoint is required")
	}
	if c.FridaHookSpec == nil {
		return fmt.Errorf("frida_hook_spec is required")
	}
	if c.OracleInputSpec == nil {
		return fmt.Errorf("oracle_input_spec is required")
	}
	if c.OracleOutputRef == "" {
		return fmt.Errorf("oracle_output_ref is required")
	}
	if c.AnalysisStateMDPath == "" {
		return fmt.Errorf("analysis_state_md_path is required")
	}
	if c.FinalArtifactPath == "" {
		return fmt.Errorf("final_artifact_path is required")
	}
	if c.ArtifactBasePath == "" {
		return fmt.Errorf("artifact_base_path is required")
	}
	if c.MaxLoopIterations <= 0 {
		c.MaxLoopIterations = defaultMaxLoopIterations
	}
	return nil
}

// AnalysisState stores persisted reverse analysis progress.
type AnalysisState struct {
	LoopIterationCount int                    `json:"loop_iteration_count"`
	CurrentPhase       string                 `json:"current_phase"`
	LastMatchRate      float64                `json:"last_match_rate"`
	StaticOutput       string                 `json:"static_output,omitempty"`
	FridaOracleOutput  string                 `json:"frida_oracle_output,omitempty"`
	KnownStructures    map[string]StructInfo  `json:"known_structures,omitempty"`
	DiscoveredOffsets  map[string]interface{} `json:"discovered_offsets,omitempty"`
	LastError          string                 `json:"last_error,omitempty"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// LoadAnalysisState loads analysis state from disk, creating a default state if missing.
func LoadAnalysisState(path string) (*AnalysisState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			state := defaultAnalysisState()
			if err := SaveAnalysisState(path, state); err != nil {
				return nil, err
			}
			return state, nil
		}
		return nil, err
	}

	state := defaultAnalysisState()
	if len(data) > 0 {
		if err := json.Unmarshal(data, state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analysis state: %w", err)
		}
	}

	if state.KnownStructures == nil {
		state.KnownStructures = make(map[string]StructInfo)
	}
	if state.DiscoveredOffsets == nil {
		state.DiscoveredOffsets = make(map[string]interface{})
	}
	return state, nil
}

// SaveAnalysisState saves analysis state to disk.
func SaveAnalysisState(path string, state *AnalysisState) error {
	if state == nil {
		return fmt.Errorf("analysis state is nil")
	}
	if state.KnownStructures == nil {
		state.KnownStructures = make(map[string]StructInfo)
	}
	if state.DiscoveredOffsets == nil {
		state.DiscoveredOffsets = make(map[string]interface{})
	}
	state.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create analysis state directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write analysis state: %w", err)
	}
	return nil
}

func defaultAnalysisState() *AnalysisState {
	return &AnalysisState{
		KnownStructures:   make(map[string]StructInfo),
		DiscoveredOffsets: make(map[string]interface{}),
		UpdatedAt:         time.Now().UTC(),
	}
}
