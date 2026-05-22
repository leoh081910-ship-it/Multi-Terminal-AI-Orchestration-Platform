package reverse

import (
	"context"
	"testing"
)

// --- Mocks ---

type mockLogger struct{}

func (m *mockLogger) Info(string, map[string]interface{})  {}
func (m *mockLogger) Error(string, error, map[string]interface{}) {}
func (m *mockLogger) Debug(string, map[string]interface{}) {}

type mockIDAMCPClient struct {
	result *StaticAnalysis
	err    error
}

func (m *mockIDAMCPClient) GetStaticAnalysis(ctx context.Context, targetPath string) (*StaticAnalysis, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &StaticAnalysis{
		Functions: []FunctionInfo{{Name: "target_function", Address: 0x1000, Size: 32, Signature: "int target_function(void)"}},
		Structs:   []StructInfo{{Name: "Input", Size: 4, Fields: []FieldInfo{{Name: "value", Offset: 0, Type: "int", Size: 4}}}},
	}, nil
}
func (m *mockIDAMCPClient) GetFunctionInfo(ctx context.Context, address uint64) (*FunctionInfo, error) {
	return nil, nil
}
func (m *mockIDAMCPClient) GetStructInfo(ctx context.Context, name string) (*StructInfo, error) {
	return nil, nil
}

type mockFridaClient struct {
	avail  bool
	err    error
	result *HookResult
}

func (m *mockFridaClient) RunHook(ctx context.Context, hookSpec map[string]interface{}, inputSpec map[string]interface{}) (*HookResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &HookResult{Output: "ok", ExitCode: 0}, nil
}
func (m *mockFridaClient) IsDeviceAvailable(ctx context.Context) bool { return m.avail }

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

// --- Task 6: Generated C must not contain placeholder TODO ---

func TestGenerateCCodeDoesNotEmitPlaceholderTODO(t *testing.T) {
	executor := NewExecutor(nil, nil, &mockLogger{})
	state := defaultAnalysisState()
	state.LoopIterationCount = 1
	analysis := &StaticAnalysis{
		Functions: []FunctionInfo{{Name: "target_function", Address: 0x1000, Size: 32, Signature: "int target_function(void)"}},
		Structs:   []StructInfo{{Name: "Input", Size: 4, Fields: []FieldInfo{{Name: "value", Offset: 0, Type: "int", Size: 4}}}},
	}

	code := executor.generateCCode(analysis, state)
	if contains(code, "TODO") {
		t.Fatalf("generated C contains TODO: %s", code)
	}
	if !contains(code, "int main") {
		t.Fatalf("generated C should contain main: %s", code)
	}
}

// --- Task 8: IDA unavailable classification ---

func TestExecutorClassifiesIDAUnavailable(t *testing.T) {
	artifactBase := t.TempDir()
	config := &ReverseTaskConfig{
		TaskID:              "task-ida-unavailable",
		TaskType:            TaskTypeStaticCRebuild,
		TargetSOPath:        "/path/to/lib.so",
		IDAMCPEndpoint:      "localhost:8080",
		FridaHookSpec:       map[string]interface{}{"function": "main"},
		OracleInputSpec:     map[string]interface{}{"input": "test"},
		OracleOutputRef:     "expected_output",
		AnalysisStateMDPath: artifactBase + "/analysis.md",
		FinalArtifactPath:   "src/final.c",
		ArtifactBasePath:    artifactBase,
		MaxLoopIterations:   1,
	}

	ida := &mockIDAMCPClient{err: context.DeadlineExceeded}
	frida := &mockFridaClient{avail: true}
	executor := NewExecutor(ida, frida, &mockLogger{})

	_, err := executor.Execute(context.Background(), config)
	if err == nil {
		t.Fatal("expected IDA environment unavailable error")
	}
	unavailable, ok := err.(*EnvironmentUnavailableError)
	if !ok {
		t.Fatalf("expected EnvironmentUnavailableError, got %T: %v", err, err)
	}
	if unavailable.Reason != "ida_mcp_unavailable" {
		t.Fatalf("Reason = %q, want ida_mcp_unavailable", unavailable.Reason)
	}
}
