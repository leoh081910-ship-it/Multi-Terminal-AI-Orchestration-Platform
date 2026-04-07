package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/migrate"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) (*Server, *store.Repository, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:server?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	drv := entsql.OpenDB(entdialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()
	if err := client.Schema.Create(ctx, migrate.WithGlobalUniqueID(true)); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	logger := zerolog.New(nil)
	repo := store.NewRepository(client, &logger)
	srv := New(repo, logger)

	cleanup := func() {
		client.Close()
	}

	return srv, repo, cleanup
}

func createTestTask(t *testing.T, repo *store.Repository, state string) string {
	t.Helper()

	id := "task_" + state
	ctx := context.Background()
	cardJSON := `{"id":"` + id + `","dispatch_ref":"dispatch_001","state":"` + state + `","transport":"cli","wave":1,"topo_rank":1}`

	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          id,
		DispatchRef: "dispatch_001",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    cardJSON,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	return id
}


func TestServerTaskStatsRecentUsesMappedTaskView(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	firstID := createTestTask(t, repo, engine.StateQueued)
	secondID := createTestTask(t, repo, engine.StateVerifyFailed)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/stats", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Total   int            `json:"total"`
			ByState map[string]int `json:"byState"`
			Recent  []struct {
				ID string `json:"id"`
			} `json:"recent"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success response")
	}
	if payload.Data.Total != 2 {
		t.Fatalf("expected total %d, got %d", 2, payload.Data.Total)
	}
	if payload.Data.ByState[engine.StateQueued] != 1 {
		t.Fatalf("expected queued count %d, got %d", 1, payload.Data.ByState[engine.StateQueued])
	}
	if payload.Data.ByState[engine.StateVerifyFailed] != 1 {
		t.Fatalf("expected verify_failed count %d, got %d", 1, payload.Data.ByState[engine.StateVerifyFailed])
	}
	if len(payload.Data.Recent) != 2 {
		t.Fatalf("expected 2 recent tasks, got %d", len(payload.Data.Recent))
	}
	if payload.Data.Recent[0].ID != secondID || payload.Data.Recent[1].ID != firstID {
		t.Fatalf("expected recent order [%s %s], got [%s %s]", secondID, firstID, payload.Data.Recent[0].ID, payload.Data.Recent[1].ID)
	}
}

func TestHandleRetryTask_Success(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateVerifyFailed)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	task, err := repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateRetryWaiting {
		t.Fatalf("expected state %q, got %q", engine.StateRetryWaiting, task.State)
	}

	var payload APIResponse
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success response")
	}
}


func TestServerTaskResponsesUseCardJSONBusinessFields(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          "outer-id",
		DispatchRef: "outer-dispatch",
		Transport:   "outer-transport",
		Wave:        9,
		CardJSON:    `{"id":"task-from-json","dispatch_ref":"dispatch-from-json","state":"verify_failed","retry_count":1,"loop_iteration_count":2,"transport":"claude-code","wave":3,"topo_rank":4,"workspace_path":"json-workspace","artifact_path":"json-artifact","last_error_reason":"json-error"}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task-from-json", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload struct {
		Success bool           `json:"success"`
		Data    store.TaskView `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success {
		t.Fatal("expected success response")
	}
	if payload.Data.ID != "task-from-json" || payload.Data.DispatchRef != "dispatch-from-json" || payload.Data.Transport != "claude-code" {
		t.Fatalf("expected task business fields from card_json, got id=%q dispatch=%q transport=%q", payload.Data.ID, payload.Data.DispatchRef, payload.Data.Transport)
	}
	if payload.Data.State != "verify_failed" || payload.Data.Wave != 3 || payload.Data.TopoRank != 4 {
		t.Fatalf("expected mapped task view from card_json, got state=%q wave=%d topo=%d", payload.Data.State, payload.Data.Wave, payload.Data.TopoRank)
	}
}

func TestServerTaskListsUseCardJSONBusinessFields(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          "outer-list-id",
		DispatchRef: "outer-dispatch",
		Transport:   "outer-transport",
		Wave:        1,
		CardJSON:    `{"id":"task-list-json","dispatch_ref":"dispatch-list-json","state":"queued","transport":"worker","wave":5,"topo_rank":6}`,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload struct {
		Success bool             `json:"success"`
		Data    []store.TaskView `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success || len(payload.Data) != 1 {
		t.Fatalf("expected one successful task response, got success=%v len=%d", payload.Success, len(payload.Data))
	}
	if payload.Data[0].ID != "task-list-json" || payload.Data[0].DispatchRef != "dispatch-list-json" || payload.Data[0].Transport != "worker" {
		t.Fatalf("expected list task view from card_json, got id=%q dispatch=%q transport=%q", payload.Data[0].ID, payload.Data[0].DispatchRef, payload.Data[0].Transport)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/dispatches/dispatch-list-json/tasks", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected dispatch list status %d, got %d", http.StatusOK, res.Code)
	}

	var dispatchPayload struct {
		Success bool             `json:"success"`
		Data    []store.TaskView `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&dispatchPayload); err != nil {
		t.Fatalf("failed to decode dispatch list response: %v", err)
	}
	if len(dispatchPayload.Data) != 1 || dispatchPayload.Data[0].ID != "task-list-json" {
		t.Fatalf("expected dispatch-scoped mapped task, got %+v", dispatchPayload.Data)
	}
}

func TestServerRetryFlowPreservesPhase1TaskBehavior(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateVerifyFailed)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	task, err := repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateRetryWaiting {
		t.Fatalf("expected state %q, got %q", engine.StateRetryWaiting, task.State)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var payload struct {
		Success bool           `json:"success"`
		Data    store.TaskView `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if payload.Data.State != engine.StateRetryWaiting {
		t.Fatalf("expected API state %q after retry, got %q", engine.StateRetryWaiting, payload.Data.State)
	}
	if payload.Data.DispatchRef != "dispatch_001" || payload.Data.Transport != "cli" {
		t.Fatalf("expected business fields preserved after retry, got dispatch=%q transport=%q", payload.Data.DispatchRef, payload.Data.Transport)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tasks/stats", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var stats struct {
		Success bool `json:"success"`
		Data    struct {
			ByState map[string]int `json:"byState"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}
	if stats.Data.ByState[engine.StateRetryWaiting] != 1 {
		t.Fatalf("expected retry_waiting count %d, got %d", 1, stats.Data.ByState[engine.StateRetryWaiting])
	}
}

func TestHandleRetryTask_Conflict(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateFailed)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, res.Code)
	}
}

func TestHandleCancelTask_NotImplemented(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateRunning)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/cancel", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, res.Code)
	}
}
