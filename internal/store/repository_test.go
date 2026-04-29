package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/event"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/migrate"
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
	err = client.Schema.Create(ctx, migrate.WithGlobalUniqueID(true))
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	logger := zerolog.New(nil)
	repo := NewRepository(client, &logger)

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
