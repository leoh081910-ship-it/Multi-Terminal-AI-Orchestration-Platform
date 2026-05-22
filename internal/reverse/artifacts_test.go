package reverse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFinalCRejectsPlaceholderContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "final.c")
	content := "#include <stdio.h>\nint main(void) {\n    // TODO: Implement main logic\n    return 0;\n}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write final.c: %v", err)
	}

	err := validateFinalC(path)
	if err == nil {
		t.Fatal("expected placeholder final.c to be rejected")
	}
	if !contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder error, got %v", err)
	}
}

func TestValidateFinalCAcceptsMinimalCompleteProgram(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "final.c")
	content := "#include <stdio.h>\nint main(void) {\n    printf(\"ok\\n\");\n    return 0;\n}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write final.c: %v", err)
	}

	if err := validateFinalC(path); err != nil {
		t.Fatalf("expected final.c to be valid, got %v", err)
	}
}

func TestReverseArtifactPaths(t *testing.T) {
	paths := reverseArtifactPaths("/tmp/artifacts", "task-123")

	wantDir := filepath.Join("/tmp/artifacts", "task-123", "reverse")
	if paths.Dir != wantDir {
		t.Fatalf("Dir = %q, want %q", paths.Dir, wantDir)
	}
	if paths.FinalC != filepath.Join(wantDir, "final.c") {
		t.Fatalf("FinalC = %q", paths.FinalC)
	}
	if paths.State != filepath.Join(wantDir, "RE-STATE.md") {
		t.Fatalf("State = %q", paths.State)
	}
	if paths.StaticOutput != filepath.Join(wantDir, "static_output.json") {
		t.Fatalf("StaticOutput = %q", paths.StaticOutput)
	}
	if paths.FridaOutput != filepath.Join(wantDir, "frida_oracle_output.json") {
		t.Fatalf("FridaOutput = %q", paths.FridaOutput)
	}
	if paths.DiffReport != filepath.Join(wantDir, "diff_report.json") {
		t.Fatalf("DiffReport = %q", paths.DiffReport)
	}
}
