package transport

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCLITransportType(t *testing.T) {
	transport := NewCLITransport("/tmp/base", "/tmp/repo")

	if transport.Type() != TransportCLI {
		t.Errorf("Expected type %s, got %s", TransportCLI, transport.Type())
	}
}

func TestCLITransportValidate(t *testing.T) {
	transport := NewCLITransport("/tmp/base", "/tmp/repo")

	tests := []struct {
		name      string
		config    *TaskConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: &TaskConfig{
				TaskID:        "task1",
				WorkspacePath: "/tmp/workspace",
				ArtifactPath:  "/tmp/artifacts",
				FilesToModify: []string{"*.go"},
			},
			wantError: false,
		},
		{
			name: "missing workspace_path",
			config: &TaskConfig{
				TaskID:        "task1",
				ArtifactPath:  "/tmp/artifacts",
				FilesToModify: []string{"*.go"},
			},
			wantError: true,
		},
		{
			name: "missing artifact_path",
			config: &TaskConfig{
				TaskID:        "task1",
				WorkspacePath: "/tmp/workspace",
				FilesToModify: []string{"*.go"},
			},
			wantError: true,
		},
		{
			name: "missing files_to_modify",
			config: &TaskConfig{
				TaskID:        "task1",
				WorkspacePath: "/tmp/workspace",
				ArtifactPath:  "/tmp/artifacts",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transport.Validate(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantError)
			}
		})
	}
}

func TestAPITransport(t *testing.T) {
	transport := NewAPITransport("/tmp/base")

	if transport.Type() != TransportAPI {
		t.Errorf("Expected type %s, got %s", TransportAPI, transport.Type())
	}

	// Test validation
	validConfig := &TaskConfig{
		TaskID:        "task1",
		WorkspacePath: "/tmp/workspace",
		ArtifactPath:  "/tmp/artifacts",
	}

	if err := transport.Validate(validConfig); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}

	// Test missing required fields
	invalidConfig := &TaskConfig{
		TaskID: "task1",
	}

	if err := transport.Validate(invalidConfig); err == nil {
		t.Error("Expected invalid config to fail validation")
	}
}

func TestTransportType(t *testing.T) {
	tests := []struct {
		transportType TransportType
		expected      string
	}{
		{TransportCLI, "cli"},
		{TransportAPI, "api"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.transportType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.transportType)
			}
		})
	}
}

func TestPathValidatorAllowsSpacesAndChinese(t *testing.T) {
	validator := NewPathValidator()

	err := validator.ValidatePath(`E:\04-Claude\Projects\多终端 AI 编排平台\workspace with spaces`)
	if err != nil {
		t.Fatalf("expected Windows-compatible path to pass, got %v", err)
	}
}

func TestBuildShellCommandUsesOSDefaultShell(t *testing.T) {
	cmd, err := BuildShellCommand(context.Background(), "echo hello", "")
	if err != nil {
		t.Fatalf("BuildShellCommand failed: %v", err)
	}

	if runtime.GOOS == "windows" {
		if filepath.Base(cmd.Path) != "cmd.exe" || len(cmd.Args) < 3 || cmd.Args[1] != "/C" {
			t.Fatalf("expected Windows default shell cmd /C, got path=%q args=%v", cmd.Path, cmd.Args)
		}
		return
	}

	if cmd.Path != "sh" || len(cmd.Args) < 3 || cmd.Args[1] != "-c" {
		t.Fatalf("expected POSIX default shell sh -c, got path=%q args=%v", cmd.Path, cmd.Args)
	}
}

func TestBuildShellCommandHonorsExplicitShell(t *testing.T) {
	cmd, err := BuildShellCommand(context.Background(), "Write-Host test", "powershell")
	if err != nil {
		t.Fatalf("BuildShellCommand failed: %v", err)
	}

	if filepath.Base(cmd.Path) != "powershell.exe" && filepath.Base(cmd.Path) != "powershell" {
		t.Fatalf("expected powershell executable, got path=%q args=%v", cmd.Path, cmd.Args)
	}
	if len(cmd.Args) < 4 || cmd.Args[1] != "-NoProfile" || cmd.Args[2] != "-Command" {
		t.Fatalf("expected powershell command shape, got path=%q args=%v", cmd.Path, cmd.Args)
	}
}

func TestCLITransportExtractArtifactsFallsBackToMainRepo(t *testing.T) {
	basePath := t.TempDir()
	mainRepo := t.TempDir()
	worktreePath := filepath.Join(basePath, "task-1")

	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(mainRepo, "frontend", "src", "pages"), 0755); err != nil {
		t.Fatalf("failed to create main repo dir: %v", err)
	}

	expectedContent := []byte("export const AccountWorkspace = () => null;\n")
	mainRepoFile := filepath.Join(mainRepo, "frontend", "src", "pages", "AccountWorkspace.tsx")
	if err := os.WriteFile(mainRepoFile, expectedContent, 0644); err != nil {
		t.Fatalf("failed to write main repo file: %v", err)
	}

	transport := NewCLITransport(basePath, mainRepo)
	artifacts, err := transport.extractArtifacts(context.Background(), worktreePath, &TaskConfig{
		FilesToModify: []string{"frontend/src/pages/AccountWorkspace.tsx"},
	})
	if err != nil {
		t.Fatalf("extractArtifacts failed: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Path != filepath.Join("frontend", "src", "pages", "AccountWorkspace.tsx") {
		t.Fatalf("unexpected artifact path %q", artifacts[0].Path)
	}
	if string(artifacts[0].Content) != string(expectedContent) {
		t.Fatalf("unexpected artifact content %q", string(artifacts[0].Content))
	}
}
