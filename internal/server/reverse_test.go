package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/reverse"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
)

func TestIsReverseCompatTask(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected bool
	}{
		{"reverse task", map[string]interface{}{"type": "reverse_static_c_rebuild"}, true},
		{"regular task", map[string]interface{}{"type": "integration"}, false},
		{"empty type", map[string]interface{}{"type": ""}, false},
		{"nil payload", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReverseCompatTask(tt.payload); got != tt.expected {
				t.Errorf("isReverseCompatTask() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildReverseTaskConfig_MissingFields(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	// Minimal project config so execution manager exists
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:     t.TempDir(),
		ArtifactBasePath: t.TempDir(),
	})

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          "task-reverse-missing",
		DispatchRef: "dispatch-rm",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-reverse-missing","dispatch_ref":"dispatch-rm","state":"queued","transport":"cli","wave":1,"topo_rank":1,"type":"reverse_static_c_rebuild"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	projectID := srv.defaultCompatProjectID()
	mgr := srv.compatExecutionForProject(projectID)

	payload := map[string]interface{}{
		"type": "reverse_static_c_rebuild",
	}
	_, err = srv.buildReverseTaskConfig("task-reverse-missing", payload, mgr)
	if err == nil {
		t.Fatal("expected error for missing reverse fields")
	}
	// Should fail validation on target_so_path
	if !contains(err.Error(), "target_so_path") {
		t.Errorf("expected target_so_path error, got: %v", err)
	}
}

func TestBuildReverseTaskConfig_Valid(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	tmpDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:     tmpDir,
		ArtifactBasePath: filepath.Join(tmpDir, "artifacts"),
	})

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          "task-reverse-valid",
		DispatchRef: "dispatch-rv",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-reverse-valid","dispatch_ref":"dispatch-rv","state":"queued","transport":"cli","wave":1,"topo_rank":1,"type":"reverse_static_c_rebuild"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	projectID := srv.defaultCompatProjectID()
	mgr := srv.compatExecutionForProject(projectID)

	payload := map[string]interface{}{
		"type":             "reverse_static_c_rebuild",
		"target_so_path":   filepath.Join(tmpDir, "libtest.so"),
		"ida_mcp_endpoint": "localhost:8080",
		"frida_hook_spec":  map[string]interface{}{"function": "main"},
		"oracle_input_spec": map[string]interface{}{"input": "test"},
		"oracle_output_ref": "expected_output",
		"max_loop_iterations": 10,
	}
	config, err := srv.buildReverseTaskConfig("task-reverse-valid", payload, mgr)
	if err != nil {
		t.Fatalf("buildReverseTaskConfig failed: %v", err)
	}
	if config.TaskID != "task-reverse-valid" {
		t.Errorf("expected TaskID task-reverse-valid, got %q", config.TaskID)
	}
	if config.MaxLoopIterations != 10 {
		t.Errorf("expected MaxLoopIterations 10, got %d", config.MaxLoopIterations)
	}
}

func TestValidateReverseArtifacts_MissingFinal(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	config := &reverse.ReverseTaskConfig{
		TaskID:           "task-no-final",
		ArtifactBasePath: t.TempDir(),
	}
	err := srv.validateReverseArtifacts(config)
	if err == nil {
		t.Fatal("expected error for missing final artifact")
	}
}

func TestValidateReverseArtifacts_Sub100MatchRate(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, "task-sub100", "reverse")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create final.c
	if err := os.WriteFile(filepath.Join(taskDir, "final.c"), []byte("int main(){}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create diff_report.json with 85% match rate
	report := reverse.DiffReport{MatchRate: 85.0, TotalComparisons: 100, MatchedComparisons: 85}
	data, _ := json.Marshal(report)
	if err := os.WriteFile(filepath.Join(taskDir, "diff_report.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	config := &reverse.ReverseTaskConfig{
		TaskID:           "task-sub100",
		ArtifactBasePath: tmpDir,
		FinalArtifactPath: filepath.Join(taskDir, "final.c"),
	}
	err := srv.validateReverseArtifacts(config)
	if err == nil {
		t.Fatal("expected error for < 100% match rate")
	}
}

func TestValidateReverseArtifacts_100Match(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, "task-100match", "reverse")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(taskDir, "final.c"), []byte("int main(){}"), 0644); err != nil {
		t.Fatal(err)
	}

	report := reverse.DiffReport{MatchRate: 100.0, TotalComparisons: 100, MatchedComparisons: 100}
	data, _ := json.Marshal(report)
	if err := os.WriteFile(filepath.Join(taskDir, "diff_report.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	config := &reverse.ReverseTaskConfig{
		TaskID:           "task-100match",
		ArtifactBasePath: tmpDir,
		FinalArtifactPath: filepath.Join(taskDir, "final.c"),
	}
	if err := srv.validateReverseArtifacts(config); err != nil {
		t.Fatalf("expected no error for 100%% match, got: %v", err)
	}
}

func TestReportLoopIterationPersistsCountAndEvent(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	taskID := "task-loop-report"
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          taskID,
		DispatchRef: "dispatch-lr",
		State:       "running",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-loop-report","dispatch_ref":"dispatch-lr","state":"running","transport":"cli","wave":1,"topo_rank":1,"loop_iteration_count":0,"owner_agent":"claude"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Advance to running
	_ = repo.UpdateTaskState(ctx, taskID, "queued", "routed", "", &store.EventData{
		EventID: "evt-route", TaskID: taskID, EventType: "state_transition",
		FromState: "queued", ToState: "routed", Transport: "cli",
	})
	_ = repo.UpdateTaskState(ctx, taskID, "routed", "workspace_prepared", "", &store.EventData{
		EventID: "evt-wp", TaskID: taskID, EventType: "state_transition",
		FromState: "routed", ToState: "workspace_prepared", Transport: "cli",
	})
	_ = repo.UpdateTaskState(ctx, taskID, "workspace_prepared", "running", "", &store.EventData{
		EventID: "evt-run", TaskID: taskID, EventType: "state_transition",
		FromState: "workspace_prepared", ToState: "running", Transport: "cli",
	})

	event := reverse.LoopIterationEvent{
		TaskID:       taskID,
		Iteration:    5,
		CurrentPhase: "diff",
		MatchRate:    87.5,
		Error:        "",
	}
	if err := srv.ReportLoopIteration(ctx, event); err != nil {
		t.Fatalf("ReportLoopIteration failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.LoopIterationCount != 5 {
		t.Errorf("expected LoopIterationCount 5, got %d", task.LoopIterationCount)
	}

	// Verify loop_iteration event was created
	events, err := repo.ListEventsByTaskID(ctx, taskID)
	if err != nil {
		t.Fatalf("ListTaskEvents failed: %v", err)
	}
	found := false
	for _, evt := range events {
		if evt.EventType == "loop_iteration" {
			found = true
			if evt.FromState != engine.StateRunning || evt.ToState != engine.StateRunning {
				t.Errorf("expected running->running, got %s->%s", evt.FromState, evt.ToState)
			}
			if evt.Reason != "diff" {
				t.Errorf("expected reason diff, got %q", evt.Reason)
			}
			var details map[string]interface{}
			if err := json.Unmarshal([]byte(evt.Details), &details); err != nil {
				t.Fatalf("failed to parse event details: %v", err)
			}
			if details["iteration"].(float64) != 5 {
				t.Errorf("expected iteration 5, got %v", details["iteration"])
			}
			if details["current_phase"] != "diff" {
				t.Errorf("expected phase diff, got %v", details["current_phase"])
			}
			break
		}
	}
	if !found {
		t.Error("expected loop_iteration event to be created")
	}
}

func TestFinishReverseExecutionFailure_EntersRetryWaiting(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	taskID := "task-reverse-fail"
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          taskID,
		DispatchRef: "dispatch-rf",
		State:       "running",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-reverse-fail","dispatch_ref":"dispatch-rf","state":"running","transport":"cli","wave":1,"topo_rank":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	_ = repo.UpdateTaskState(ctx, taskID, "queued", "routed", "", &store.EventData{
		EventID: "evt-r", TaskID: taskID, EventType: "state_transition",
		FromState: "queued", ToState: "routed", Transport: "cli",
	})
	_ = repo.UpdateTaskState(ctx, taskID, "routed", "workspace_prepared", "", &store.EventData{
		EventID: "evt-w", TaskID: taskID, EventType: "state_transition",
		FromState: "routed", ToState: "workspace_prepared", Transport: "cli",
	})
	_ = repo.UpdateTaskState(ctx, taskID, "workspace_prepared", "running", "", &store.EventData{
		EventID: "evt-go", TaskID: taskID, EventType: "state_transition",
		FromState: "workspace_prepared", ToState: "running", Transport: "cli",
	})

	srv.finishReverseExecutionFailure(ctx, taskID, "environment_error: gcc not found")

	task, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateRetryWaiting {
		t.Errorf("expected state retry_waiting, got %q", task.State)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
