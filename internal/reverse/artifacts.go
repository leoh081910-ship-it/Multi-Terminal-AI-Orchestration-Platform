package reverse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type artifactPaths struct {
	Dir          string
	FinalC       string
	State        string
	StaticOutput string
	FridaOutput  string
	DiffReport   string
}

func reverseArtifactPaths(basePath, taskID string) artifactPaths {
	dir := filepath.Join(basePath, taskID, "reverse")
	return artifactPaths{
		Dir:          dir,
		FinalC:       filepath.Join(dir, "final.c"),
		State:        filepath.Join(dir, "RE-STATE.md"),
		StaticOutput: filepath.Join(dir, "static_output.json"),
		FridaOutput:  filepath.Join(dir, "frida_oracle_output.json"),
		DiffReport:   filepath.Join(dir, "diff_report.json"),
	}
}

func validateFinalC(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read final.c: %w", err)
	}
	content := string(data)
	lower := strings.ToLower(content)
	if strings.Contains(lower, "todo") || strings.Contains(lower, "placeholder") || strings.Contains(lower, "unresolved offset") {
		return fmt.Errorf("final.c contains placeholder content")
	}
	if !strings.Contains(content, "int main") {
		return fmt.Errorf("final.c must contain an int main entrypoint")
	}
	return nil
}
