// Package reverse provides reverse engineering task execution with quantifiable verification loops.
package reverse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Executor handles reverse engineering task execution.
type Executor struct {
	// IDAMCPClient is the IDA Pro MCP client.
	IDAMCPClient IDAMCPClient
	// FridaClient is the Frida client for dynamic analysis.
	FridaClient FridaClient
	// Logger logs execution events.
	Logger Logger
}

// IDAMCPClient provides IDA Pro MCP integration.
type IDAMCPClient interface {
	// GetStaticAnalysis retrieves static analysis information from IDA Pro.
	GetStaticAnalysis(ctx context.Context, targetPath string) (*StaticAnalysis, error)
	// GetFunctionInfo retrieves function information by address.
	GetFunctionInfo(ctx context.Context, address uint64) (*FunctionInfo, error)
	// GetStructInfo retrieves structure information.
	GetStructInfo(ctx context.Context, name string) (*StructInfo, error)
}

// FridaClient provides Frida integration for dynamic analysis.
type FridaClient interface {
	// RunHook executes a Frida hook and captures output.
	RunHook(ctx context.Context, hookSpec map[string]interface{}, inputSpec map[string]interface{}) (*HookResult, error)
	// IsDeviceAvailable checks if the target device is available.
	IsDeviceAvailable(ctx context.Context) bool
}

// Logger provides logging for reverse engineering tasks.
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// StaticAnalysis contains static analysis results from IDA Pro.
type StaticAnalysis struct {
	Functions []FunctionInfo `json:"functions"`
	Structs   []StructInfo   `json:"structs"`
	Exports   []ExportInfo   `json:"exports"`
	Imports   []ImportInfo   `json:"imports"`
	Segments  []SegmentInfo  `json:"segments"`
}

// FunctionInfo represents a function in the binary.
type FunctionInfo struct {
	Name      string `json:"name"`
	Address   uint64 `json:"address"`
	Size      uint64 `json:"size"`
	Signature string `json:"signature,omitempty"`
}

// StructInfo represents a structure definition.
type StructInfo struct {
	Name   string      `json:"name"`
	Size   uint64      `json:"size"`
	Fields []FieldInfo `json:"fields"`
}

// FieldInfo represents a struct field.
type FieldInfo struct {
	Name   string `json:"name"`
	Offset uint64 `json:"offset"`
	Type   string `json:"type"`
	Size   uint64 `json:"size"`
}

// ExportInfo represents an exported symbol.
type ExportInfo struct {
	Name    string `json:"name"`
	Address uint64 `json:"address"`
}

// ImportInfo represents an imported symbol.
type ImportInfo struct {
	Name   string `json:"name"`
	Module string `json:"module"`
}

// SegmentInfo represents a memory segment.
type SegmentInfo struct {
	Name  string `json:"name"`
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
	Perms string `json:"permissions"`
}

// HookResult contains the result of a Frida hook execution.
type HookResult struct {
	Output   string                 `json:"output"`
	ExitCode int                    `json:"exit_code"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoopResult contains the result of a single verification loop iteration.
type LoopResult struct {
	Iteration    int           `json:"iteration"`
	MatchRate    float64       `json:"match_rate"`
	StaticOutput string        `json:"static_output,omitempty"`
	FridaOutput  string        `json:"frida_output,omitempty"`
	DiffReport   *DiffReport   `json:"diff_report,omitempty"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// DiffReport contains the diff analysis results.
type DiffReport struct {
	MatchRate          float64           `json:"match_rate"`
	MismatchCases      []MismatchCase    `json:"mismatch_cases,omitempty"`
	NormalizationRules map[string]string `json:"normalization_rules,omitempty"`
	TotalComparisons   int               `json:"total_comparisons"`
	MatchedComparisons int               `json:"matched_comparisons"`
}

// MismatchCase represents a single mismatch in the diff.
type MismatchCase struct {
	Index    int    `json:"index"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Field    string `json:"field,omitempty"`
	Context  string `json:"context,omitempty"`
}

// FinalArtifact contains the final reverse engineering output.
type FinalArtifact struct {
	// SourceCode is the reconstructed C source code.
	SourceCode string `json:"source_code"`
	// StructDefinitions contains all discovered structure definitions.
	StructDefinitions []StructInfo `json:"struct_definitions,omitempty"`
	// FunctionDefinitions contains all discovered function definitions.
	FunctionDefinitions []FunctionInfo `json:"function_definitions,omitempty"`
	// Metadata contains additional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewExecutor creates a new reverse engineering executor.
func NewExecutor(idaClient IDAMCPClient, fridaClient FridaClient, logger Logger) *Executor {
	return &Executor{
		IDAMCPClient: idaClient,
		FridaClient:  fridaClient,
		Logger:       logger,
	}
}

// Execute runs the reverse engineering task with the quantifiable verification loop.
// REQUIRES (REVR-04~14): Full verification loop with 100% match rate requirement.
func (e *Executor) Execute(ctx context.Context, config *ReverseTaskConfig) (*FinalArtifact, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid reverse task configuration: %w", err)
	}

	e.Logger.Info("Starting reverse engineering task execution", map[string]interface{}{
		"task_id":   config.TaskID,
		"task_type": config.TaskType,
	})

	// Load or initialize analysis state
	state, err := LoadAnalysisState(config.AnalysisStateMDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis state: %w", err)
	}

	// Main verification loop
	for {
		// Check loop iteration limit
		if state.LoopIterationCount >= config.MaxLoopIterations {
			e.Logger.Error("Maximum loop iterations exceeded", nil, map[string]interface{}{
				"task_id":              config.TaskID,
				"max_loop_iterations":  config.MaxLoopIterations,
				"loop_iteration_count": state.LoopIterationCount,
			})
			return nil, &MaxLoopIterationsError{
				MaxIterations: config.MaxLoopIterations,
				Current:       state.LoopIterationCount,
			}
		}

		// Increment iteration counter
		state.LoopIterationCount++
		state.CurrentPhase = "ida_analysis"

		e.Logger.Info("Starting loop iteration", map[string]interface{}{
			"task_id":       config.TaskID,
			"iteration":     state.LoopIterationCount,
			"max_iteration": config.MaxLoopIterations,
		})

		// Step 1: Get static analysis from IDA Pro
		staticAnalysis, err := e.IDAMCPClient.GetStaticAnalysis(ctx, config.TargetSOPath)
		if err != nil {
			e.Logger.Error("IDA analysis failed", err, map[string]interface{}{
				"task_id":     config.TaskID,
				"target_path": config.TargetSOPath,
			})
			state.CurrentPhase = "ida_failed"
			SaveAnalysisState(config.AnalysisStateMDPath, state)
			return nil, fmt.Errorf("IDA analysis failed: %w", err)
		}

		// Update state with discovered structures and functions
		for _, s := range staticAnalysis.Structs {
			state.KnownStructures[s.Name] = s
		}

		state.CurrentPhase = "static_rebuild"

		// Step 2: Generate/Update static C rebuild
		sourceCode := e.generateCCode(staticAnalysis, state)

		// Write source code to artifact directory
		artifactDir := filepath.Join(config.ArtifactBasePath, config.TaskID, "reverse")
		if err := os.MkdirAll(artifactDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create artifact directory: %w", err)
		}

		sourcePath := filepath.Join(artifactDir, "final.c")
		if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
			return nil, fmt.Errorf("failed to write source file: %w", err)
		}

		state.CurrentPhase = "compile"

		// Step 3: Compile the C code
		binaryPath := filepath.Join(artifactDir, "static_rebuild")
		cmd := exec.CommandContext(ctx, "gcc", "-o", binaryPath, sourcePath, "-Wall", "-O2")
		compileOutput, err := cmd.CombinedOutput()
		if err != nil {
			e.Logger.Error("Compilation failed", err, map[string]interface{}{
				"task_id":        config.TaskID,
				"source_path":    sourcePath,
				"compile_output": string(compileOutput),
			})
			state.CurrentPhase = "compile_failed"
			state.LastError = fmt.Sprintf("Compilation failed: %v\nOutput: %s", err, string(compileOutput))
			SaveAnalysisState(config.AnalysisStateMDPath, state)
			continue // Retry loop
		}

		state.CurrentPhase = "static_run"

		// Step 4: Run static rebuild with oracle input
		cmd = exec.CommandContext(ctx, binaryPath)
		cmd.Stdin = strings.NewReader(formatOracleInput(config.OracleInputSpec))
		staticOutput, err := cmd.CombinedOutput()
		if err != nil {
			e.Logger.Error("Static run failed", err, map[string]interface{}{
				"task_id": config.TaskID,
				"output":  string(staticOutput),
			})
			state.CurrentPhase = "static_run_failed"
			state.LastError = fmt.Sprintf("Static run failed: %v", err)
			SaveAnalysisState(config.AnalysisStateMDPath, state)
			continue // Retry loop
		}

		state.StaticOutput = string(staticOutput)
		state.CurrentPhase = "frida_run"

		// Step 5: Run Frida hook on target device
		hookResult, err := e.FridaClient.RunHook(ctx, config.FridaHookSpec, config.OracleInputSpec)
		if err != nil {
			e.Logger.Error("Frida hook failed", err, map[string]interface{}{
				"task_id": config.TaskID,
			})
			// Check if it's an environment issue
			if !e.FridaClient.IsDeviceAvailable(ctx) {
				return nil, &EnvironmentUnavailableError{Reason: "frida_device_unavailable"}
			}
			state.CurrentPhase = "frida_oracle_failed"
			state.LastError = fmt.Sprintf("Frida hook failed: %v", err)
			SaveAnalysisState(config.AnalysisStateMDPath, state)
			continue // Retry loop
		}

		state.FridaOracleOutput = hookResult.Output
		state.CurrentPhase = "diff"

		// Step 6: Calculate diff
		diffReport := e.calculateDiff(state.StaticOutput, state.FridaOracleOutput)

		// Write diff report
		diffReportPath := filepath.Join(artifactDir, "diff_report.json")
		diffReportData, _ := json.MarshalIndent(diffReport, "", "  ")
		os.WriteFile(diffReportPath, diffReportData, 0644)

		// Write static output
		staticOutputPath := filepath.Join(artifactDir, "static_output.json")
		os.WriteFile(staticOutputPath, []byte(state.StaticOutput), 0644)

		// Write Frida output
		fridaOutputPath := filepath.Join(artifactDir, "frida_oracle_output.json")
		os.WriteFile(fridaOutputPath, []byte(state.FridaOracleOutput), 0644)

		state.LastMatchRate = diffReport.MatchRate
		e.Logger.Info("Loop iteration completed", map[string]interface{}{
			"task_id":    config.TaskID,
			"iteration":  state.LoopIterationCount,
			"match_rate": diffReport.MatchRate,
		})

		// Step 7: Check match rate
		if diffReport.MatchRate >= 100.0 {
			// Success! Copy final.c to the final artifact path
			finalArtifactPath := filepath.Join(config.ArtifactBasePath, config.TaskID, "reverse", "final.c")
			if err := copyFile(sourcePath, finalArtifactPath); err != nil {
				return nil, fmt.Errorf("failed to copy final artifact: %w", err)
			}

			state.CurrentPhase = "complete"
			SaveAnalysisState(config.AnalysisStateMDPath, state)

			return &FinalArtifact{
				SourceCode:          sourceCode,
				StructDefinitions:   staticAnalysis.Structs,
				FunctionDefinitions: staticAnalysis.Functions,
				Metadata: map[string]interface{}{
					"loop_iterations":  state.LoopIterationCount,
					"final_match_rate": diffReport.MatchRate,
					"target_so":        config.TargetSOPath,
				},
			}, nil
		}

		// Not yet 100%, continue loop
		state.CurrentPhase = "refining"
		SaveAnalysisState(config.AnalysisStateMDPath, state)
	}
}

// generateCCode generates C code from static analysis and current state.
func (e *Executor) generateCCode(analysis *StaticAnalysis, state *AnalysisState) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString("/*\n")
	buf.WriteString(" * Auto-generated C code from reverse engineering analysis\n")
	buf.WriteString(fmt.Sprintf(" * Loop iteration: %d\n", state.LoopIterationCount))
	buf.WriteString(" */\n\n")

	// Includes
	buf.WriteString("#include <stdio.h>\n")
	buf.WriteString("#include <stdlib.h>\n")
	buf.WriteString("#include <string.h>\n")
	buf.WriteString("#include <stdint.h>\n\n")

	// Struct definitions
	for _, s := range analysis.Structs {
		buf.WriteString(fmt.Sprintf("// Structure: %s (size: %d bytes)\n", s.Name, s.Size))
		buf.WriteString(fmt.Sprintf("typedef struct %s {\n", s.Name))
		for _, f := range s.Fields {
			buf.WriteString(fmt.Sprintf("    %s %s; // offset: %d, size: %d\n", f.Type, f.Name, f.Offset, f.Size))
		}
		buf.WriteString(fmt.Sprintf("} %s;\n\n", s.Name))
	}

	// Function declarations
	for _, f := range analysis.Functions {
		if f.Signature != "" {
			buf.WriteString(fmt.Sprintf("// Function at 0x%x (size: %d bytes)\n", f.Address, f.Size))
			buf.WriteString(fmt.Sprintf("%s;\n\n", f.Signature))
		}
	}

	// Main function for testing
	buf.WriteString("// Main function for oracle input processing\n")
	buf.WriteString("int main(int argc, char* argv[]) {\n")
	buf.WriteString("    // TODO: Implement main logic based on oracle_input_spec\n")
	buf.WriteString("    printf(\"Reverse engineering target initialized\\n\");\n")
	buf.WriteString("    return 0;\n")
	buf.WriteString("}\n")

	return buf.String()
}

// calculateDiff calculates the diff between static and Frida outputs.
func (e *Executor) calculateDiff(staticOutput, fridaOutput string) *DiffReport {
	// Normalize outputs
	staticNormalized := normalizeOutput(staticOutput)
	fridaNormalized := normalizeOutput(fridaOutput)

	// Split into lines/records for comparison
	staticLines := strings.Split(staticNormalized, "\n")
	fridaLines := strings.Split(fridaNormalized, "\n")

	// Calculate matches and mismatches
	mismatches := []MismatchCase{}
	matched := 0
	total := 0

	maxLen := len(staticLines)
	if len(fridaLines) > maxLen {
		maxLen = len(fridaLines)
	}

	for i := 0; i < maxLen; i++ {
		total++
		var staticLine, fridaLine string
		if i < len(staticLines) {
			staticLine = staticLines[i]
		}
		if i < len(fridaLines) {
			fridaLine = fridaLines[i]
		}

		if staticLine == fridaLine {
			matched++
		} else {
			mismatches = append(mismatches, MismatchCase{
				Index:    i,
				Expected: fridaLine, // Frida output is the ground truth
				Actual:   staticLine,
				Context:  fmt.Sprintf("line %d", i),
			})
		}
	}

	// Calculate match rate
	matchRate := 0.0
	if total > 0 {
		matchRate = float64(matched) / float64(total) * 100.0
	}

	return &DiffReport{
		MatchRate:          matchRate,
		MismatchCases:      mismatches,
		TotalComparisons:   total,
		MatchedComparisons: matched,
		NormalizationRules: map[string]string{
			"whitespace": "trim_trailing",
			"case":       "sensitive",
		},
	}
}

// normalizeOutput normalizes output for comparison.
func normalizeOutput(output string) string {
	// Trim whitespace
	output = strings.TrimSpace(output)

	// Normalize line endings
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	// Remove trailing whitespace from each line
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// formatOracleInput formats oracle input from spec.
func formatOracleInput(spec map[string]interface{}) string {
	// Convert spec to JSON for processing
	data, _ := json.Marshal(spec)
	return string(data)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}

// Errors for reverse engineering.
type (
	// MaxLoopIterationsError is returned when maximum loop iterations is exceeded.
	MaxLoopIterationsError struct {
		MaxIterations int
		Current       int
	}

	// EnvironmentUnavailableError is returned when the environment is unavailable.
	EnvironmentUnavailableError struct {
		Reason string
	}
)

func (e *MaxLoopIterationsError) Error() string {
	return fmt.Sprintf("maximum loop iterations exceeded: %d of %d", e.Current, e.MaxIterations)
}

func (e *EnvironmentUnavailableError) Error() string {
	return fmt.Sprintf("environment unavailable: %s", e.Reason)
}
