package reverse

import (
	"testing"
)

func TestReverseTaskConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    ReverseTaskConfig
		wantError bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: ReverseTaskConfig{
				TaskID:              "task_001",
				TaskType:            TaskTypeStaticCRebuild,
				TargetSOPath:        "/path/to/lib.so",
				IDAMCPEndpoint:      "localhost:8080",
				FridaHookSpec:       map[string]interface{}{"function": "main"},
				OracleInputSpec:     map[string]interface{}{"input": "test"},
				OracleOutputRef:     "expected_output",
				AnalysisStateMDPath: "/tmp/analysis.md",
				FinalArtifactPath:   "src/final.c",
				ArtifactBasePath:    "/tmp/artifacts",
			},
			wantError: false,
		},
		{
			name: "missing task_id",
			config: ReverseTaskConfig{
				TaskType:     TaskTypeStaticCRebuild,
				TargetSOPath: "/path/to/lib.so",
			},
			wantError: true,
			errMsg:    "task_id",
		},
		{
			name: "invalid task type",
			config: ReverseTaskConfig{
				TaskID:       "task_001",
				TaskType:     "invalid_type",
				TargetSOPath: "/path/to/lib.so",
			},
			wantError: true,
			errMsg:    "unsupported task type",
		},
		{
			name: "missing target_so_path",
			config: ReverseTaskConfig{
				TaskID:   "task_001",
				TaskType: TaskTypeStaticCRebuild,
			},
			wantError: true,
			errMsg:    "target_so_path",
		},
		{
			name: "missing ida_mcp_endpoint",
			config: ReverseTaskConfig{
				TaskID:       "task_001",
				TaskType:     TaskTypeStaticCRebuild,
				TargetSOPath: "/path/to/lib.so",
			},
			wantError: true,
			errMsg:    "ida_mcp_endpoint",
		},
		{
			name: "missing frida_hook_spec",
			config: ReverseTaskConfig{
				TaskID:         "task_001",
				TaskType:       TaskTypeStaticCRebuild,
				TargetSOPath:   "/path/to/lib.so",
				IDAMCPEndpoint: "localhost:8080",
			},
			wantError: true,
			errMsg:    "frida_hook_spec",
		},
		{
			name: "default max_loop_iterations",
			config: ReverseTaskConfig{
				TaskID:              "task_001",
				TaskType:            TaskTypeStaticCRebuild,
				TargetSOPath:        "/path/to/lib.so",
				IDAMCPEndpoint:      "localhost:8080",
				FridaHookSpec:       map[string]interface{}{},
				OracleInputSpec:     map[string]interface{}{},
				OracleOutputRef:     "ref",
				AnalysisStateMDPath: "/tmp/analysis.md",
				FinalArtifactPath:   "src/final.c",
				ArtifactBasePath:    "/tmp/artifacts",
				MaxLoopIterations:   0, // Should default to 50
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, expected to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}

			// Check default value for MaxLoopIterations
			if !tt.wantError && tt.config.MaxLoopIterations == 0 {
				if tt.config.MaxLoopIterations != 50 {
					t.Errorf("MaxLoopIterations should default to 50, got %d", tt.config.MaxLoopIterations)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
