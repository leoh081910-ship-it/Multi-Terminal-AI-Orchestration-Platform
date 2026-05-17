package connector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverTasks(t *testing.T) {
	planDir := t.TempDir()
	workspaceDir := t.TempDir()
	planPath := filepath.Join(planDir, "sample-plan.yaml")
	planContent := `objective: ship integration
phase: 04-integration
wave: 2
tasks:
  - id: task-a
    type: implementation
    description: implement feature
    files_to_modify:
      - internal/a.go
    depends_on:
      - task-root
    acceptance_criteria:
      - works
    context:
      prompt: do it
`
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	c, err := NewGSDConnector(Config{PlanDir: planDir, WorkspaceDir: workspaceDir, DefaultPriority: 7, DefaultTransport: "api"})
	if err != nil {
		t.Fatalf("NewGSDConnector: %v", err)
	}

	tasks, err := c.DiscoverTasks(context.Background())
	if err != nil {
		t.Fatalf("DiscoverTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "task-a" || tasks[0].Wave != 2 || tasks[0].Transport != "api" {
		t.Fatalf("unexpected task card: %+v", tasks[0])
	}
	if len(tasks[0].Relations) != 1 || tasks[0].Relations[0].TaskID != "task-root" || tasks[0].Relations[0].Type != "depends_on" {
		t.Fatalf("unexpected relations: %+v", tasks[0].Relations)
	}
}

func TestHydrateContext(t *testing.T) {
	c, err := NewGSDConnector(Config{PlanDir: t.TempDir(), WorkspaceDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewGSDConnector: %v", err)
	}

	base := map[string]interface{}{"key": "value"}
	hydrated, err := c.HydrateContext(context.Background(), "task-1", base)
	if err != nil {
		t.Fatalf("HydrateContext: %v", err)
	}
	if hydrated["key"] != "value" {
		t.Fatalf("expected base key to survive, got %#v", hydrated["key"])
	}
	if hydrated["connector_type"] != "gsd" {
		t.Fatalf("expected connector_type gsd, got %#v", hydrated["connector_type"])
	}
	if _, ok := hydrated["gsd_timestamp"]; !ok {
		t.Fatal("expected gsd_timestamp to be added")
	}
	if len(base) != 1 {
		t.Fatalf("expected base context to remain unchanged, got %+v", base)
	}
}

func TestAckResult(t *testing.T) {
	workspaceDir := t.TempDir()
	c, err := NewGSDConnector(Config{PlanDir: t.TempDir(), WorkspaceDir: workspaceDir})
	if err != nil {
		t.Fatalf("NewGSDConnector: %v", err)
	}

	result := TaskResult{TaskID: "task-ack", Success: true, State: "done"}
	if err := c.AckResult(context.Background(), result); err != nil {
		t.Fatalf("AckResult: %v", err)
	}

	ackPath := filepath.Join(workspaceDir, "acks", "task-ack.json")
	data, err := os.ReadFile(ackPath)
	if err != nil {
		t.Fatalf("read ack file: %v", err)
	}
	var decoded TaskResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode ack json: %v", err)
	}
	if decoded.TaskID != "task-ack" || decoded.State != "done" {
		t.Fatalf("unexpected ack payload: %+v", decoded)
	}
}

func TestWriteBackArtifacts(t *testing.T) {
	planDir := t.TempDir()
	workspaceDir := t.TempDir()
	statePath := filepath.Join(planDir, "STATE.md")
	if err := os.WriteFile(statePath, []byte("# State\n"), 0644); err != nil {
		t.Fatalf("write state doc: %v", err)
	}

	c, err := NewGSDConnector(Config{PlanDir: planDir, WorkspaceDir: workspaceDir})
	if err != nil {
		t.Fatalf("NewGSDConnector: %v", err)
	}

	artifacts := []Artifact{
		{Path: "nested/result.txt", Content: []byte("ok")},
	}
	if err := c.WriteBackArtifacts(context.Background(), "task-writeback", artifacts); err != nil {
		t.Fatalf("WriteBackArtifacts: %v", err)
	}

	artifactPath := filepath.Join(workspaceDir, "artifacts", "task-writeback", "nested", "result.txt")
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(content) != "ok" {
		t.Fatalf("unexpected artifact content: %q", string(content))
	}

	planningDoc, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read planning doc: %v", err)
	}
	text := string(planningDoc)
	if !strings.Contains(text, "<!-- gsd-writeback:task-writeback -->") {
		t.Fatalf("expected write-back marker in planning doc, got %q", text)
	}
	if !strings.Contains(text, "nested/result.txt") {
		t.Fatalf("expected artifact path in planning doc, got %q", text)
	}

	if err := c.WriteBackArtifacts(context.Background(), "task-writeback", artifacts); err != nil {
		t.Fatalf("WriteBackArtifacts second call: %v", err)
	}
	planningDoc, err = os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read planning doc after second call: %v", err)
	}
	if strings.Count(string(planningDoc), "<!-- gsd-writeback:task-writeback -->") != 1 {
		t.Fatalf("expected idempotent write-back note, got %q", string(planningDoc))
	}
}
