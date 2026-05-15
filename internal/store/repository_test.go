package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/agentcall"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/event"
	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*Repository, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	drv := entsql.OpenDB(entdialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()
	err = client.Schema.Create(ctx)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	logger := zerolog.New(nil)
	repo := NewRepository(client, db, &logger)

	cleanup := func() {
		client.Close()
	}

	return repo, cleanup
}

func TestCreateTask(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	card := &TaskCard{
		ID:          "task_001",
		DispatchRef: "dispatch_001",
		State:       "queued",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task_001","dispatch_ref":"dispatch_001","state":"queued","transport":"cli","wave":1}`,
	}

	taskID, err := repo.CreateTask(ctx, card)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if taskID != "task_001" {
		t.Errorf("expected taskID to be 'task_001', got %q", taskID)
	}

	task, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}

	if task.ID != "task_001" {
		t.Errorf("expected task.ID to be 'task_001', got %q", task.ID)
	}
}

func TestRepositoryTaskCardJSONSourceOfTruthOnCreate(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:                 "outer-id",
		DispatchRef:        "outer-dispatch",
		State:              "outer-state",
		RetryCount:         99,
		LoopIterationCount: 88,
		Transport:          "outer-transport",
		Wave:               7,
		TopoRank:           6,
		WorkspacePath:      "outer-workspace",
		ArtifactPath:       "outer-artifact",
		LastErrorReason:    "outer-error",
		CardJSON:           `{"id":"task-from-json","dispatch_ref":"dispatch-from-json","state":"verify_failed","retry_count":1,"loop_iteration_count":2,"transport":"claude-code","wave":3,"topo_rank":4,"workspace_path":"json-workspace","artifact_path":"json-artifact","last_error_reason":"json-error"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-from-json")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}

	if task.DispatchRef != "dispatch-from-json" || task.State != "verify_failed" || task.Transport != "claude-code" {
		t.Fatalf("expected card_json values to drive structured fields, got dispatch=%q state=%q transport=%q", task.DispatchRef, task.State, task.Transport)
	}
	if task.RetryCount != 1 || task.LoopIterationCount != 2 || task.Wave != 3 || task.TopoRank != 4 {
		t.Fatalf("expected numeric fields from card_json, got retry=%d loop=%d wave=%d topo=%d", task.RetryCount, task.LoopIterationCount, task.Wave, task.TopoRank)
	}
	if task.WorkspacePath != "json-workspace" || task.ArtifactPath != "json-artifact" || task.LastErrorReason != "json-error" {
		t.Fatalf("expected path/error fields from card_json, got workspace=%q artifact=%q error=%q", task.WorkspacePath, task.ArtifactPath, task.LastErrorReason)
	}
}

func TestRepositoryTaskCardJSONSourceOfTruthOnUpdate(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-update-json",
		DispatchRef: "dispatch-1",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-update-json","dispatch_ref":"dispatch-1","state":"queued","retry_count":0,"loop_iteration_count":0,"transport":"cli","wave":1,"topo_rank":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	err = repo.UpdateTask(ctx, "task-update-json", &TaskCard{
		DispatchRef:        "outer-dispatch-ignored",
		State:              "outer-state-ignored",
		RetryCount:         55,
		LoopIterationCount: 66,
		Transport:          "outer-ignored",
		Wave:               77,
		TopoRank:           88,
		WorkspacePath:      "outer-workspace-ignored",
		ArtifactPath:       "outer-artifact-ignored",
		LastErrorReason:    "outer-error-ignored",
		CardJSON:           `{"id":"task-update-json","dispatch_ref":"dispatch-2","state":"running","retry_count":3,"loop_iteration_count":4,"transport":"worker","wave":5,"topo_rank":6,"workspace_path":"workspace-2","artifact_path":"artifact-2","last_error_reason":"error-2"}`,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-update-json")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}

	if task.DispatchRef != "dispatch-2" || task.State != "running" || task.Transport != "worker" {
		t.Fatalf("expected updated structured fields from card_json, got dispatch=%q state=%q transport=%q", task.DispatchRef, task.State, task.Transport)
	}
	if task.RetryCount != 3 || task.LoopIterationCount != 4 || task.Wave != 5 || task.TopoRank != 6 {
		t.Fatalf("expected updated numeric fields from card_json, got retry=%d loop=%d wave=%d topo=%d", task.RetryCount, task.LoopIterationCount, task.Wave, task.TopoRank)
	}
}

func TestRepositoryTaskCardJSONRejectsInvalidCardJSON(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-invalid-create",
		DispatchRef: "dispatch-invalid",
		Transport:   "cli",
		CardJSON:    `{"id":"oops"`,
	}); err == nil || !strings.Contains(err.Error(), "invalid card_json") {
		t.Fatalf("expected invalid card_json error on create, got %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-invalid-create")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task != nil {
		t.Fatal("expected no task to be written for invalid create")
	}

	_, err = repo.CreateTask(ctx, &TaskCard{
		ID:          "task-invalid-update",
		DispatchRef: "dispatch-invalid",
		Transport:   "cli",
		CardJSON:    `{"id":"task-invalid-update","dispatch_ref":"dispatch-invalid","state":"queued","transport":"cli"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := repo.UpdateTask(ctx, "task-invalid-update", &TaskCard{CardJSON: `not-json`}); err == nil || !strings.Contains(err.Error(), "invalid card_json") {
		t.Fatalf("expected invalid card_json error on update, got %v", err)
	}

	task, err = repo.GetTaskByID(ctx, "task-invalid-update")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != "queued" || task.Transport != "cli" {
		t.Fatalf("expected task to remain unchanged after rejected update, got state=%q transport=%q", task.State, task.Transport)
	}
}

func TestUpdateTaskStateStillUsesStructuredColumns(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-state-columns",
		DispatchRef: "dispatch-columns",
		Transport:   "cli",
		CardJSON:    `{"id":"task-state-columns","dispatch_ref":"dispatch-columns","state":"verify_failed","retry_count":2,"loop_iteration_count":9,"transport":"cli"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := repo.UpdateTaskState(ctx, "task-state-columns", "verify_failed", "retry_waiting", "manual_retry", &EventData{EventID: "evt-1", EventType: "state_transition", Attempt: 3, Transport: "cli"}); err != nil {
		t.Fatalf("UpdateTaskState failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-state-columns")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != "retry_waiting" {
		t.Fatalf("expected structured state column to update, got %q", task.State)
	}
	if task.RetryCount != 2 {
		t.Fatalf("expected retry_count to remain unchanged for non-consuming reason, got %d", task.RetryCount)
	}
	if !task.TerminalAt.IsZero() {
		t.Fatalf("expected non-terminal transition to keep terminal_at empty, got %v", task.TerminalAt)
	}

	evt, err := repo.client.Event.Query().
		Where(event.TaskID("task-state-columns")).
		Only(ctx)
	if err != nil {
		t.Fatalf("failed to query event: %v", err)
	}
	if evt.Attempt != 2 {
		t.Fatalf("expected event attempt to match current retry_count, got %d", evt.Attempt)
	}
}

func TestUpdateTaskStateIncrementsRetryCountForConsumingReason(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-retry-consuming",
		DispatchRef: "dispatch-consuming",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-retry-consuming","dispatch_ref":"dispatch-consuming","state":"running","retry_count":1,"transport":"cli","wave":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := repo.UpdateTaskState(ctx, "task-retry-consuming", "running", "retry_waiting", "execution_failure", &EventData{EventID: "evt-consuming", EventType: "state_transition", Attempt: 99, Transport: "cli"}); err != nil {
		t.Fatalf("UpdateTaskState failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-retry-consuming")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.RetryCount != 2 {
		t.Fatalf("expected retry_count to increment for consuming reason, got %d", task.RetryCount)
	}
	if !task.TerminalAt.IsZero() {
		t.Fatalf("expected retry_waiting transition to keep terminal_at empty, got %v", task.TerminalAt)
	}

	evt, err := repo.client.Event.Query().
		Where(event.EventID("evt-consuming")).
		Only(ctx)
	if err != nil {
		t.Fatalf("failed to query event: %v", err)
	}
	if evt.Attempt != 2 {
		t.Fatalf("expected event attempt to use persisted retry_count, got %d", evt.Attempt)
	}
}

func TestUpdateTaskStateSetsTerminalAtForTerminalState(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-terminal",
		DispatchRef: "dispatch-terminal",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-terminal","dispatch_ref":"dispatch-terminal","state":"merged","retry_count":1,"transport":"cli","wave":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := repo.UpdateTaskState(ctx, "task-terminal", "merged", "done", "", &EventData{EventID: "evt-terminal", EventType: "state_transition", Transport: "cli"}); err != nil {
		t.Fatalf("UpdateTaskState failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-terminal")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.TerminalAt.IsZero() {
		t.Fatal("expected terminal_at to be set for terminal transition")
	}

	evt, err := repo.client.Event.Query().
		Where(event.EventID("evt-terminal")).
		Only(ctx)
	if err != nil {
		t.Fatalf("failed to query event: %v", err)
	}
	if evt.Attempt != 1 {
		t.Fatalf("expected terminal transition attempt to match current retry_count, got %d", evt.Attempt)
	}
}

func TestCleanupTaskResourcesRemovesPathsAndClearsColumns(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tempRoot := t.TempDir()
	workspacePath := filepath.Join(tempRoot, "workspace")
	artifactPath := filepath.Join(tempRoot, "artifacts")

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace path: %v", err)
	}
	if err := os.MkdirAll(artifactPath, 0755); err != nil {
		t.Fatalf("failed to create artifact path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "result.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to seed workspace file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "report.md"), []byte("done"), 0644); err != nil {
		t.Fatalf("failed to seed artifact file: %v", err)
	}

	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-cleanup",
		DispatchRef: "dispatch-cleanup",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-cleanup","project_id":"default","dispatch_ref":"dispatch-cleanup","state":"done","transport":"cli","wave":1,"workspace_path":"` + filepath.ToSlash(workspacePath) + `","artifact_path":"` + filepath.ToSlash(artifactPath) + `"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	cleaned, err := repo.CleanupTaskResources(ctx, "task-cleanup")
	if err != nil {
		t.Fatalf("CleanupTaskResources failed: %v", err)
	}
	if !cleaned {
		t.Fatal("expected cleanup to report work done")
	}

	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace path to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact path to be removed, stat err=%v", err)
	}

	task, err := repo.GetTaskByID(ctx, "task-cleanup")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.WorkspacePath != "" || task.ArtifactPath != "" {
		t.Fatalf("expected structured paths to be cleared, got workspace=%q artifact=%q", task.WorkspacePath, task.ArtifactPath)
	}
	if strings.Contains(task.CardJSON, filepath.ToSlash(workspacePath)) || strings.Contains(task.CardJSON, filepath.ToSlash(artifactPath)) {
		t.Fatalf("expected card_json paths to be cleared, got %s", task.CardJSON)
	}
}

func TestDeleteTaskRemovesTaskAndRelatedRows(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tempRoot := t.TempDir()
	workspacePath := filepath.Join(tempRoot, "delete-workspace")
	artifactPath := filepath.Join(tempRoot, "delete-artifacts")

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace path: %v", err)
	}
	if err := os.MkdirAll(artifactPath, 0755); err != nil {
		t.Fatalf("failed to create artifact path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "state.txt"), []byte("running"), 0644); err != nil {
		t.Fatalf("failed to seed workspace file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "result.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatalf("failed to seed artifact file: %v", err)
	}

	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-delete-001",
		DispatchRef: "dispatch-delete",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-delete-001","dispatch_ref":"dispatch-delete","state":"running","transport":"cli","wave":1,"workspace_path":"` + filepath.ToSlash(workspacePath) + `","artifact_path":"` + filepath.ToSlash(artifactPath) + `"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := repo.CreateEvent(ctx, &EventData{
		EventID:   "evt-delete-1",
		TaskID:    "task-delete-001",
		EventType: "state_transition",
		FromState: "queued",
		ToState:   "running",
		Timestamp: time.Now().UTC(),
		Attempt:   0,
		Transport: "cli",
	}); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if err := repo.RecordAgentCall(ctx, &AgentCallRecord{
		ID:         "call-delete-1",
		TaskID:     "task-delete-001",
		AgentID:    "agent-cli",
		RunnerType: "cli",
		TaskType:   "execution",
		Status:     "success",
		DurationMs: 120,
		StartedAt:  time.Now().UTC().Add(-2 * time.Second),
		FinishedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RecordAgentCall failed: %v", err)
	}

	deleted, err := repo.DeleteTask(ctx, "task-delete-001")
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if !deleted {
		t.Fatal("expected DeleteTask to return deleted=true")
	}

	task, err := repo.GetTaskByID(ctx, "task-delete-001")
	if err != nil {
		t.Fatalf("GetTaskByID after delete failed: %v", err)
	}
	if task != nil {
		t.Fatalf("expected task row to be deleted, got %+v", task)
	}

	eventCount, err := repo.client.Event.Query().Where(event.TaskID("task-delete-001")).Count(ctx)
	if err != nil {
		t.Fatalf("failed to count events after delete: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected event rows to be deleted, got %d", eventCount)
	}

	callCount, err := repo.client.AgentCall.Query().Where(agentcall.TaskID("task-delete-001")).Count(ctx)
	if err != nil {
		t.Fatalf("failed to count agent_calls after delete: %v", err)
	}
	if callCount != 0 {
		t.Fatalf("expected agent call rows to be deleted, got %d", callCount)
	}

	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace path to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact path to be removed, stat err=%v", err)
	}
}

func TestDeleteTaskReturnsFalseWhenMissing(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	deleted, err := repo.DeleteTask(ctx, "missing-task")
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if deleted {
		t.Fatal("expected DeleteTask to return deleted=false for missing task")
	}
}

func TestListEventsByTaskIDReturnsOrderedEvents(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	baseTs := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)

	_, err := repo.CreateTask(ctx, &TaskCard{
		ID:          "task-events-ordered",
		DispatchRef: "dispatch-events",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-events-ordered","dispatch_ref":"dispatch-events","state":"queued","transport":"cli","wave":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	_, err = repo.CreateTask(ctx, &TaskCard{
		ID:          "task-events-other",
		DispatchRef: "dispatch-events",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    `{"id":"task-events-other","dispatch_ref":"dispatch-events","state":"queued","transport":"cli","wave":1}`,
	})
	if err != nil {
		t.Fatalf("CreateTask other failed: %v", err)
	}

	if err := repo.CreateEvent(ctx, &EventData{EventID: "evt-b", TaskID: "task-events-ordered", EventType: "state_transition", FromState: "queued", ToState: "running", Timestamp: baseTs, Transport: "cli"}); err != nil {
		t.Fatalf("CreateEvent evt-b failed: %v", err)
	}
	if err := repo.CreateEvent(ctx, &EventData{EventID: "evt-a", TaskID: "task-events-ordered", EventType: "state_transition", FromState: "running", ToState: "done", Timestamp: baseTs, Transport: "cli"}); err != nil {
		t.Fatalf("CreateEvent evt-a failed: %v", err)
	}
	if err := repo.CreateEvent(ctx, &EventData{EventID: "evt-c", TaskID: "task-events-ordered", EventType: "state_transition", FromState: "done", ToState: "done", Timestamp: baseTs.Add(time.Second), Transport: "cli"}); err != nil {
		t.Fatalf("CreateEvent evt-c failed: %v", err)
	}
	if err := repo.CreateEvent(ctx, &EventData{EventID: "evt-other", TaskID: "task-events-other", EventType: "state_transition", FromState: "queued", ToState: "running", Timestamp: baseTs.Add(-time.Second), Transport: "cli"}); err != nil {
		t.Fatalf("CreateEvent evt-other failed: %v", err)
	}

	events, err := repo.ListEventsByTaskID(ctx, "task-events-ordered")
	if err != nil {
		t.Fatalf("ListEventsByTaskID failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events for target task, got %d", len(events))
	}

	gotIDs := []string{events[0].EventID, events[1].EventID, events[2].EventID}
	expectedIDs := []string{"evt-a", "evt-b", "evt-c"}
	for i := range expectedIDs {
		if gotIDs[i] != expectedIDs[i] {
			t.Fatalf("expected event order %v, got %v", expectedIDs, gotIDs)
		}
	}

	emptyEvents, err := repo.ListEventsByTaskID(ctx, "missing-task")
	if err != nil {
		t.Fatalf("ListEventsByTaskID for missing task failed: %v", err)
	}
	if emptyEvents == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(emptyEvents) != 0 {
		t.Fatalf("expected empty slice for missing task, got %d events", len(emptyEvents))
	}
}

func TestGetTaskByID_NotFound(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	task, err := repo.GetTaskByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}

	if task != nil {
		t.Errorf("expected task to be nil for nonexistent ID, got %v", task)
	}
}
