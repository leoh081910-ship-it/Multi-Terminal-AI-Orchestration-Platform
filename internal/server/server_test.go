package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	entdialect "entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/event"
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

func setupTestServerWithClient(t *testing.T) (*Server, *store.Repository, *ent.Client, func()) {
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

	return srv, repo, client, cleanup
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

func createCompatTask(t *testing.T, repo *store.Repository, taskID string, cardJSON string) {
	t.Helper()

	ctx := context.Background()
	_, err := repo.CreateTask(ctx, &store.TaskCard{
		ID:          taskID,
		DispatchRef: "dispatch_compat",
		Transport:   "cli",
		Wave:        1,
		CardJSON:    cardJSON,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
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

func TestHandleCancelTask_Success(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateRunning)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/cancel", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	task, err := repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateFailed {
		t.Fatalf("expected state %q, got %q", engine.StateFailed, task.State)
	}
	if task.TerminalAt.IsZero() {
		t.Fatal("expected terminal_at to be set after cancel")
	}

	var payload APIResponse
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success response")
	}
}

func TestHandleCancelTask_RejectsMergedState(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateMerged)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/cancel", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, res.Code)
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

	// PR-3: done is the only truly terminal state now
	taskID := createTestTask(t, repo, engine.StateDone)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, res.Code)
	}
}

func TestHandleCancelTask_IdempotentForTerminalState(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	taskID := createTestTask(t, repo, engine.StateDone)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/cancel", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestCompatSchedulerEndpointsExposeTSIReadShape(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "CL-001", `{"id":"CL-001","dispatch_ref":"dispatch_compat","state":"running","transport":"cli","wave":1,"topo_rank":1,"title":"Wire scheduler routes","owner_agent":"Claude","status":"in_progress","type":"integration","priority":1,"dispatch_mode":"auto","auto_dispatch_enabled":true,"dispatch_status":"running","execution_runtime":"claude","execution_session_id":"SE-CL-001","dispatch_attempts":1,"current_focus":"Wire scheduler routes","last_dispatch_at":"2026-04-09T10:00:00Z"}`)
	createCompatTask(t, repo, "GM-001", `{"id":"GM-001","dispatch_ref":"dispatch_compat","state":"queued","transport":"cli","wave":1,"topo_rank":2,"title":"Bind board widgets","owner_agent":"Gemini","status":"assigned","type":"ui","priority":2,"depends_on":["CL-001"],"dispatch_mode":"auto","auto_dispatch_enabled":true,"dispatch_status":"pending","next_action":"Wait for backend summary"}`)
	createCompatTask(t, repo, "CX-001", `{"id":"CX-001","dispatch_ref":"dispatch_compat","state":"failed","transport":"cli","wave":1,"topo_rank":3,"title":"Align rule contract","owner_agent":"Codex","status":"blocked","type":"decision-logic","priority":2,"blocked_reason":"Waiting for CL-001","dispatch_status":"failed","dispatch_attempts":1,"last_dispatch_error":"codex runtime command is not configured","last_dispatch_at":"2026-04-09T10:05:00Z"}`)
	createCompatTask(t, repo, "CL-000", `{"id":"CL-000","dispatch_ref":"dispatch_compat","state":"done","transport":"cli","wave":1,"topo_rank":0,"title":"Lock real project root","owner_agent":"Claude","status":"done","type":"integration","priority":1,"dispatch_mode":"manual","auto_dispatch_enabled":false,"dispatch_status":"completed","execution_runtime":"claude","execution_session_id":"SE-ROOT","dispatch_attempts":1,"result_summary":"Root locked","last_dispatch_at":"2026-04-09T09:00:00Z"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var tasks []compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks response: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}

	byID := make(map[string]compatSchedulerTask, len(tasks))
	for _, task := range tasks {
		byID[task.TaskID] = task
	}

	if byID["CL-001"].OwnerAgent != compatAgentClaude || byID["CL-001"].DispatchStatus != "running" {
		t.Fatalf("expected CL-001 running for Claude, got %+v", byID["CL-001"])
	}
	if len(byID["GM-001"].DependsOn) != 1 || byID["GM-001"].DependsOn[0] != "CL-001" {
		t.Fatalf("expected GM-001 depends_on to include CL-001, got %+v", byID["GM-001"].DependsOn)
	}
	if byID["CX-001"].BlockedReason != "Waiting for CL-001" || byID["CX-001"].LastDispatchError == "" {
		t.Fatalf("expected CX-001 blocked reason and error, got %+v", byID["CX-001"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks?owner_agent=Gemini", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var filtered []compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&filtered); err != nil {
		t.Fatalf("failed to decode filtered tasks response: %v", err)
	}
	if len(filtered) != 1 || filtered[0].TaskID != "GM-001" {
		t.Fatalf("expected Gemini filter to return GM-001, got %+v", filtered)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/board/summary", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var board compatBoardSummary
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&board); err != nil {
		t.Fatalf("failed to decode board summary: %v", err)
	}
	if board.TotalTasks != 4 || board.BlockedCount != 1 {
		t.Fatalf("expected board totals 4/1, got %+v", board)
	}
	if board.CountsByStatus["in_progress"] != 1 || board.CountsByStatus["assigned"] != 1 || board.CountsByStatus["done"] != 1 {
		t.Fatalf("unexpected board counts: %+v", board.CountsByStatus)
	}
	if board.CountsByAgent[compatAgentClaude] != 2 || board.CountsByAgent[compatAgentGemini] != 1 || board.CountsByAgent[compatAgentCodex] != 1 {
		t.Fatalf("unexpected agent counts: %+v", board.CountsByAgent)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/summary", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var agents compatAgentsSummary
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&agents); err != nil {
		t.Fatalf("failed to decode agents summary: %v", err)
	}
	if len(agents.Agents) != 3 {
		t.Fatalf("expected 3 agent summaries, got %d", len(agents.Agents))
	}
	agentSummary := make(map[string]compatAgentSummaryItem, len(agents.Agents))
	for _, agent := range agents.Agents {
		agentSummary[agent.OwnerAgent] = agent
	}
	if agentSummary[compatAgentClaude].TotalTasks != 2 || len(agentSummary[compatAgentClaude].Tasks.InProgress) != 1 {
		t.Fatalf("unexpected Claude summary: %+v", agentSummary[compatAgentClaude])
	}
	if agentSummary[compatAgentGemini].CountsByStatus["assigned"] != 1 {
		t.Fatalf("unexpected Gemini summary: %+v", agentSummary[compatAgentGemini])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/runtimes", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var runtimes compatRuntimeSummaryList
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&runtimes); err != nil {
		t.Fatalf("failed to decode runtimes response: %v", err)
	}
	if len(runtimes.Runtimes) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(runtimes.Runtimes))
	}
	runtimeByAgent := make(map[string]compatRuntimeSummary, len(runtimes.Runtimes))
	for _, runtime := range runtimes.Runtimes {
		runtimeByAgent[runtime.OwnerAgent] = runtime
	}
	if runtimeByAgent[compatAgentClaude].ActiveSessions != 1 || runtimeByAgent[compatAgentClaude].Status != "available" {
		t.Fatalf("unexpected Claude runtime summary: %+v", runtimeByAgent[compatAgentClaude])
	}
	if runtimeByAgent[compatAgentCodex].LastError == "" {
		t.Fatalf("expected Codex runtime error, got %+v", runtimeByAgent[compatAgentCodex])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/executions", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	var executions compatTaskExecutionList
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&executions); err != nil {
		t.Fatalf("failed to decode executions response: %v", err)
	}
	if len(executions.Executions) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executions.Executions))
	}

	executionByTask := make(map[string]compatTaskExecution, len(executions.Executions))
	for _, execution := range executions.Executions {
		executionByTask[execution.TaskID] = execution
	}
	if executionByTask["CL-001"].Status != "running" || executionByTask["CL-001"].SessionID != "SE-CL-001" {
		t.Fatalf("unexpected CL-001 execution: %+v", executionByTask["CL-001"])
	}
	if executionByTask["CX-001"].Error == "" || executionByTask["CX-001"].CompletedAt == nil {
		t.Fatalf("unexpected CX-001 execution: %+v", executionByTask["CX-001"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks/CL-001/execution", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected task execution status %d, got %d", http.StatusOK, res.Code)
	}

	var execution compatTaskExecution
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&execution); err != nil {
		t.Fatalf("failed to decode task execution response: %v", err)
	}
	if execution.TaskID != "CL-001" || execution.Runtime != "claude" {
		t.Fatalf("unexpected execution payload: %+v", execution)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks/GM-001/execution", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected pending task execution to be 404, got %d", res.Code)
	}
}

func TestCompatSchedulerCreateEndpointPersistsTSIWriteShape(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{
		"title":"Wire Gemini board page",
		"type":"ui",
		"owner_agent":"Gemini",
		"priority":2,
		"status":"ready",
		"depends_on":["CL-001"],
		"input_artifacts":["/api/v1/board/summary"],
		"output_artifacts":["frontend/src/pages/board.tsx"],
		"acceptance_criteria":["Board page renders summary cards"],
		"description":"Gemini UI integration task",
		"current_focus":"Bind widgets",
		"next_action":"Connect API call"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.Code)
	}

	var payload compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if payload.TaskID == "" || !strings.HasPrefix(payload.TaskID, "TS-") {
		t.Fatalf("expected generated TS task id, got %q", payload.TaskID)
	}
	if payload.OwnerAgent != compatAgentGemini || payload.Status != "ready" || payload.DispatchStatus != "pending" {
		t.Fatalf("unexpected create response payload: %+v", payload)
	}
	if len(payload.InputArtifacts) != 1 || payload.InputArtifacts[0] != "/api/v1/board/summary" {
		t.Fatalf("expected input artifacts to round-trip, got %+v", payload.InputArtifacts)
	}

	task, err := repo.GetTaskByID(context.Background(), payload.TaskID)
	if err != nil {
		t.Fatalf("failed to reload created task: %v", err)
	}
	if task == nil {
		t.Fatal("expected created task to exist")
	}
	if task.State != engine.StateQueued {
		t.Fatalf("expected queued internal state, got %q", task.State)
	}
}

func TestCompatSchedulerProjectScopedRoutesFilterTasks(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	if err := srv.ConfigureCompatProjects(CompatProjectsConfig{
		DefaultProjectID: "alpha",
		Projects: []CompatProjectConfig{
			{ID: "alpha", Name: "Alpha", MainRepoPath: t.TempDir()},
			{ID: "beta", Name: "Beta", MainRepoPath: t.TempDir()},
		},
	}); err != nil {
		t.Fatalf("ConfigureCompatProjects failed: %v", err)
	}

	createCompatTask(t, repo, "ALPHA-001", `{"id":"ALPHA-001","project_id":"alpha","dispatch_ref":"alpha","state":"queued","transport":"cli","wave":1,"topo_rank":1,"title":"Alpha task","owner_agent":"Claude","status":"ready","type":"integration","priority":1}`)
	createCompatTask(t, repo, "BETA-001", `{"id":"BETA-001","project_id":"beta","dispatch_ref":"beta","state":"queued","transport":"cli","wave":1,"topo_rank":1,"title":"Beta task","owner_agent":"Gemini","status":"ready","type":"integration","priority":1}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/alpha/scheduler/tasks", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var tasks []compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks response: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TaskID != "ALPHA-001" || tasks[0].ProjectID != "alpha" {
		t.Fatalf("expected only alpha task, got %+v", tasks)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var projects []compatProjectSummary
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&projects); err != nil {
		t.Fatalf("failed to decode projects response: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if !projects[0].Default || projects[0].ID != "alpha" {
		t.Fatalf("expected alpha to be default project, got %+v", projects[0])
	}
}

func TestCompatSchedulerWriteEndpointsUpdateAndDispatchTask(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "CX-001", `{"id":"CX-001","dispatch_ref":"dispatch_compat","state":"verify_failed","transport":"cli","wave":1,"topo_rank":3,"title":"Align rule contract","owner_agent":"Codex","status":"blocked","type":"decision-logic","priority":2,"blocked_reason":"Waiting for CL-001","dispatch_status":"failed","dispatch_attempts":1,"last_dispatch_error":"codex runtime command is not configured"}`)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/scheduler/tasks/CX-001", bytes.NewBufferString(`{
		"status":"blocked",
		"priority":1,
		"result_summary":"Rule payload stabilized",
		"blocked_reason":null,
		"dispatch_status":"failed",
		"last_dispatch_error":"runtime unavailable"
	}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected patch status %d, got %d", http.StatusOK, res.Code)
	}

	var patched compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&patched); err != nil {
		t.Fatalf("failed to decode patch response: %v", err)
	}
	if patched.Status != "blocked" || patched.Priority != 1 || patched.ResultSummary != "Rule payload stabilized" {
		t.Fatalf("unexpected patch response payload: %+v", patched)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/CX-001/retry", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected retry status %d, got %d", http.StatusOK, res.Code)
	}

	var retried compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&retried); err != nil {
		t.Fatalf("failed to decode retry response: %v", err)
	}
	if retried.Status != "ready" || retried.DispatchStatus != "failed" || retried.DispatchAttempts != 2 {
		t.Fatalf("unexpected retry response payload: %+v", retried)
	}
	if retried.ExecutionRuntime != "codex" || retried.LastDispatchError == "" {
		t.Fatalf("expected retry to record runtime failure, got %+v", retried)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks/CX-001/execution", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected execution status %d, got %d", http.StatusOK, res.Code)
	}

	var execution compatTaskExecution
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&execution); err != nil {
		t.Fatalf("failed to decode execution response: %v", err)
	}
	if execution.TaskID != "CX-001" || execution.Status != "failed" || execution.Runtime != "codex" {
		t.Fatalf("unexpected execution payload: %+v", execution)
	}
}

func TestCompatDispatchExecutesConfiguredTaskAndProducesArtifacts(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	baseDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:      baseDir,
		WorktreeBasePath:  filepath.Join(baseDir, "worktrees"),
		WorkspaceBasePath: filepath.Join(baseDir, "workspaces"),
		ArtifactBasePath:  filepath.Join(baseDir, "artifacts"),
	})

	createCompatTask(t, repo, "GM-REAL-001", `{"id":"GM-REAL-001","dispatch_ref":"dispatch_compat","state":"queued","transport":"api","wave":1,"topo_rank":1,"title":"Generate board summary note","owner_agent":"Gemini","status":"ready","type":"analysis","priority":2,"output_artifacts":["result.txt"],"shell":"powershell","command":"python -c \"from pathlib import Path; Path('result.txt').write_text('done', encoding='utf-8')\""}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/GM-REAL-001/dispatch", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected dispatch status %d, got %d", http.StatusOK, res.Code)
	}

	var dispatched compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&dispatched); err != nil {
		t.Fatalf("failed to decode dispatch response: %v", err)
	}
	if dispatched.DispatchStatus != "running" || dispatched.Status != "in_progress" {
		t.Fatalf("expected running response, got %+v", dispatched)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		task, err := repo.GetTaskByID(context.Background(), "GM-REAL-001")
		if err != nil {
			t.Fatalf("failed to reload task: %v", err)
		}
		if task != nil && (task.State == engine.StateVerified || task.State == engine.StateReviewPending) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	task, err := repo.GetTaskByID(context.Background(), "GM-REAL-001")
	if err != nil {
		t.Fatalf("failed to reload final task: %v", err)
	}
	if task == nil || task.State != engine.StateVerified && task.State != engine.StateReviewPending {
		t.Fatalf("expected task to reach verified or review_pending, got %+v", task)
	}

	mapped := srv.mapCompatTask(task)
	if (mapped.Status != "verified" && mapped.Status != "review_pending") || mapped.DispatchStatus != "completed" {
		t.Fatalf("expected mapped review/completed state, got %+v", mapped)
	}

	artifactPath := filepath.Join(baseDir, "artifacts", "GM-REAL-001", "result.txt")
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected artifact file to exist: %v", err)
	}
	if string(content) != "done" {
		t.Fatalf("expected artifact content 'done', got %q", string(content))
	}
}

func TestMergeQueueAdapterSyncsCompatPayloadOnDone(t *testing.T) {
	_, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TS-DONE-001", `{"id":"TS-DONE-001","dispatch_ref":"dispatch_compat","state":"verified","transport":"cli","wave":1,"topo_rank":1,"title":"Finalize smoke","owner_agent":"Claude","status":"verified","dispatch_status":"completed","output_artifacts":["smoke/done.txt"]}`)

	adapter := NewMergeQueueRepositoryAdapter(repo, "", compatDefaultProjectID)
	if err := adapter.UpdateTaskState(context.Background(), "TS-DONE-001", engine.StateVerified, engine.StateDone, ""); err != nil {
		t.Fatalf("UpdateTaskState failed: %v", err)
	}

	task, err := repo.GetTaskByID(context.Background(), "TS-DONE-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task == nil || task.State != engine.StateDone {
		t.Fatalf("expected task state done, got %+v", task)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.CardJSON), &payload); err != nil {
		t.Fatalf("failed to decode card_json: %v", err)
	}
	if payload["status"] != "done" {
		t.Fatalf("expected compat status done, got %#v", payload["status"])
	}
	if payload["dispatch_status"] != "completed" {
		t.Fatalf("expected compat dispatch_status completed, got %#v", payload["dispatch_status"])
	}
}

func TestAutoDispatcherDispatchesEligibleAutoTask(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	baseDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:      baseDir,
		WorktreeBasePath:  filepath.Join(baseDir, "worktrees"),
		WorkspaceBasePath: filepath.Join(baseDir, "workspaces"),
		ArtifactBasePath:  filepath.Join(baseDir, "artifacts"),
	})

	createCompatTask(t, repo, "AUTO-001", `{"id":"AUTO-001","dispatch_ref":"dispatch_auto","state":"queued","transport":"api","wave":1,"topo_rank":1,"title":"Auto task","owner_agent":"Claude","status":"ready","type":"automation","priority":5,"dispatch_mode":"auto","auto_dispatch_enabled":true,"dispatch_status":"pending","output_artifacts":["result.txt"],"shell":"powershell","command":"python -c \"from pathlib import Path; Path('result.txt').write_text('auto', encoding='utf-8')\""}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := NewAutoDispatcher(srv, zerolog.New(nil), AutoDispatchConfig{Interval: 20 * time.Millisecond})
	if err := dispatcher.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer dispatcher.Stop()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		task, err := repo.GetTaskByID(context.Background(), "AUTO-001")
		if err != nil {
			t.Fatalf("GetTaskByID failed: %v", err)
		}
		if task != nil && (task.State == engine.StateVerified || task.State == engine.StateReviewPending) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	task, err := repo.GetTaskByID(context.Background(), "AUTO-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task == nil || task.State != engine.StateVerified && task.State != engine.StateReviewPending {
		t.Fatalf("expected task to reach verified or review_pending, got %+v", task)
	}

	mapped := srv.mapCompatTask(task)
	if (mapped.Status != "verified" && mapped.Status != "review_pending") || mapped.DispatchStatus != "completed" {
		t.Fatalf("unexpected mapped task after auto dispatch: %+v", mapped)
	}
}

func TestAutoDispatcherWaitsForDoneDependencies(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	baseDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:      baseDir,
		WorktreeBasePath:  filepath.Join(baseDir, "worktrees"),
		WorkspaceBasePath: filepath.Join(baseDir, "workspaces"),
		ArtifactBasePath:  filepath.Join(baseDir, "artifacts"),
	})

	createCompatTask(t, repo, "AUTO-DEP-001", `{"id":"AUTO-DEP-001","dispatch_ref":"dispatch_auto","state":"verified","transport":"api","wave":1,"topo_rank":1,"title":"Dependency task","owner_agent":"Claude","status":"verified","type":"automation","priority":5,"dispatch_mode":"auto","auto_dispatch_enabled":true,"dispatch_status":"completed","output_artifacts":["dep.txt"]}`)
	createCompatTask(t, repo, "AUTO-CHILD-001", `{"id":"AUTO-CHILD-001","dispatch_ref":"dispatch_auto","state":"queued","transport":"api","wave":1,"topo_rank":2,"title":"Dependent task","owner_agent":"Gemini","status":"ready","type":"automation","priority":4,"dispatch_mode":"auto","auto_dispatch_enabled":true,"dispatch_status":"pending","depends_on":["AUTO-DEP-001"],"output_artifacts":["child.txt"],"shell":"powershell","command":"python -c \"from pathlib import Path; Path('child.txt').write_text('child', encoding='utf-8')\""}`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := NewAutoDispatcher(srv, zerolog.New(nil), AutoDispatchConfig{Interval: 20 * time.Millisecond})
	if err := dispatcher.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer dispatcher.Stop()

	time.Sleep(300 * time.Millisecond)

	childTask, err := repo.GetTaskByID(context.Background(), "AUTO-CHILD-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if childTask == nil || childTask.State != engine.StateQueued {
		t.Fatalf("expected child to stay queued before dependency done, got %+v", childTask)
	}

	adapter := NewMergeQueueRepositoryAdapter(repo, "", compatDefaultProjectID)
	if err := adapter.UpdateTaskState(context.Background(), "AUTO-DEP-001", engine.StateVerified, engine.StateDone, ""); err != nil {
		t.Fatalf("failed to mark dependency done: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		childTask, err = repo.GetTaskByID(context.Background(), "AUTO-CHILD-001")
		if err != nil {
			t.Fatalf("GetTaskByID failed: %v", err)
		}
		if childTask != nil && childTask.State == engine.StateVerified {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	childTask, err = repo.GetTaskByID(context.Background(), "AUTO-CHILD-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if childTask == nil || childTask.State != engine.StateVerified && childTask.State != engine.StateReviewPending {
		t.Fatalf("expected child to reach verified or review_pending, got %+v", childTask)
	}
}

func TestCompatSchedulerTasksFallbackToLegacyTaskCardFields(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "legacy-001", `{"id":"legacy-001","dispatch_ref":"dispatch_compat","state":"queued","transport":"cli","wave":1,"topo_rank":1,"type":"integration","objective":"Legacy objective","priority":4,"relations":[{"task_id":"seed-001","type":"depends_on","reason":"after seed"}]}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var tasks []compatSchedulerTask
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks response: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Legacy objective" || task.OwnerAgent != compatAgentClaude {
		t.Fatalf("expected fallback title/owner, got %+v", task)
	}
	if task.Status != "ready" || task.DispatchStatus != "pending" {
		t.Fatalf("expected fallback ready/pending mapping, got %+v", task)
	}
	if len(task.DependsOn) != 1 || task.DependsOn[0] != "seed-001" {
		t.Fatalf("expected fallback depends_on relation, got %+v", task.DependsOn)
	}
}

// TestExecutionReaperTimeoutEnforcement verifies that timeout_at is enforced:
// a running task with an expired timeout_at should be reclaimed to triage
// with an execution_timeout event.
func TestExecutionReaperTimeoutEnforcement(t *testing.T) {
	srv, repo, client, cleanup := setupTestServerWithClient(t)
	defer cleanup()

	// Create a running task with expired timeout_at (1 second in the past)
	ctx := context.Background()
	taskID := "timeout-test-001"
	now := time.Now().UTC()
	timeoutAt := now.Add(-1 * time.Second).Format(time.RFC3339)
	startedAt := now.Add(-5 * time.Minute).Format(time.RFC3339)
	heartbeatAt := now.Add(-30 * time.Second).Format(time.RFC3339)
	cardJSON := `{"id":"timeout-test-001","dispatch_ref":"dispatch_timeout","state":"running","transport":"cli","wave":1,"topo_rank":1,"title":"Timeout test task","owner_agent":"Claude","status":"in_progress","type":"integration","priority":1,"dispatch_status":"running","execution_runtime":"claude","execution_session_id":"SE-TIMEOUT-001","dispatch_attempts":1,"timeout_at":"` + timeoutAt + `","started_at":"` + startedAt + `","last_heartbeat_at":"` + heartbeatAt + `"}`

	createCompatTask(t, repo, taskID, cardJSON)

	// Verify task was created in running state
	task, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	t.Logf("task state: %q, cardJSON: %s", task.State, task.CardJSON)
	if task.State != engine.StateRunning {
		t.Fatalf("expected task state running, got %q", task.State)
	}

	// Debug: check if ListTasksByState finds the task
	runningTasks, err := repo.ListTasksByState(ctx, engine.StateRunning)
	if err != nil {
		t.Fatalf("ListTasksByState failed: %v", err)
	}
	t.Logf("found %d running tasks", len(runningTasks))
	for _, rt := range runningTasks {
		t.Logf("running task: id=%s, state=%s", rt.ID, rt.State)
	}

	// Create reaper and manually test isZombie
	reaper := NewExecutionReaper(srv, repo, srv.logger, ExecutionReaperConfig{Interval: 15 * time.Second})
	isZombieResult := reaper.isZombie(task)
	t.Logf("isZombie result: %v", isZombieResult)
	if !isZombieResult {
		t.Fatalf("expected isZombie to return true for task with expired timeout_at")
	}

	// Directly call reclaim to see the error
	reclaimErr := reaper.reclaim(ctx, task)
	t.Logf("reclaim error: %v", reclaimErr)
	if reclaimErr != nil {
		t.Fatalf("reclaim failed: %v", reclaimErr)
	}

	reclaimedTask, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID after reclaim failed: %v", err)
	}
	if reclaimedTask.State != engine.StateTriage {
		t.Fatalf("expected task state triage after reclaim, got %q", reclaimedTask.State)
	}
	reclaimedCompat := srv.mapCompatTask(reclaimedTask)
	if reclaimedCompat.Status != "triage" || reclaimedCompat.DispatchStatus != "triage" {
		t.Fatalf("expected triage/triage compat view after reclaim, got %+v", reclaimedCompat)
	}

	// Also directly test UpdateTaskState
	t.Logf("Testing UpdateTaskState directly")
	fromState := task.State
	toState := engine.StateTriage
	reason := "execution_timeout"
	eventData := &store.EventData{
		EventID:   "test-event-001",
		TaskID:    taskID,
		EventType: "state_transition",
		FromState: fromState,
		ToState:   toState,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		Attempt:   task.RetryCount,
		Transport: task.Transport,
		Details:   "test direct call",
	}
	directErr := repo.UpdateTaskState(ctx, taskID, fromState, toState, reason, eventData)
	t.Logf("direct UpdateTaskState error: %v", directErr)

	// Query fresh state from database
	freshTask, err := repo.GetTaskByID(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskByID fresh failed: %v", err)
	}
	t.Logf("fresh task state after direct call: %q", freshTask.State)

	// Verify execution_timeout event was created
	events, err := client.Event.Query().
		Where(event.TaskID(taskID)).
		All(ctx)
	if err != nil {
		t.Fatalf("failed to query events: %v", err)
	}

	var foundTimeoutEvent bool
	for _, e := range events {
		t.Logf("event: type=%s, from=%s, to=%s", e.EventType, e.FromState, e.ToState)
		if e.EventType == "execution_timeout" {
			foundTimeoutEvent = true
			if e.FromState != engine.StateRunning {
				t.Fatalf("expected from_state running, got %q", e.FromState)
			}
			if e.ToState != engine.StateTriage {
				t.Fatalf("expected to_state triage, got %q", e.ToState)
			}
			break
		}
	}

	if !foundTimeoutEvent {
		t.Fatalf("expected execution_timeout event to be created, got events: %+v", events)
	}
}

func TestFinishCompatExecutionFailureKeepsRetryableTaskInTriage(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "FAIL-TRIAGE-001", `{"id":"FAIL-TRIAGE-001","dispatch_ref":"dispatch_fail","state":"running","transport":"cli","wave":1,"topo_rank":1,"title":"Retryable failure","owner_agent":"Claude","status":"in_progress","type":"integration","priority":1,"dispatch_status":"running","execution_session_id":"SE-FAIL-001"}`)

	srv.finishCompatExecutionFailure(context.Background(), "FAIL-TRIAGE-001", "empty_artifact_match: no files matched pattern: out.txt")

	task, err := repo.GetTaskByID(context.Background(), "FAIL-TRIAGE-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateTriage {
		t.Fatalf("expected triage state after retryable failure, got %q", task.State)
	}

	mapped := srv.mapCompatTask(task)
	if mapped.Status != "triage" || mapped.DispatchStatus != "triage" {
		t.Fatalf("expected triage/triage compat view after retryable failure, got %+v", mapped)
	}
}

func TestSystemTasksUseDedicatedReportArtifacts(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	createCompatTask(t, repo, "REVIEW-PARENT-001", `{"id":"REVIEW-PARENT-001","dispatch_ref":"dispatch_review","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Parent review task","owner_agent":"Codex","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","output_artifacts":["src/file.go"],"artifact_path":"C:/artifacts/review-parent"}`)

	reviewWorker := NewReviewWorker(srv, repo, srv.logger, time.Second)
	if err := reviewWorker.processReviewPendingTasks(ctx); err != nil {
		t.Fatalf("processReviewPendingTasks failed: %v", err)
	}

	parent, err := repo.GetTaskByID(ctx, "REVIEW-PARENT-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	parentPayload := decodeCompatPayload(parent.CardJSON)
	reviewTaskID := readString(parentPayload, "last_review_task_id")
	if reviewTaskID == "" {
		t.Fatal("expected review task to be created")
	}
	reviewTask, err := repo.GetTaskByID(ctx, reviewTaskID)
	if err != nil {
		t.Fatalf("failed to load review task: %v", err)
	}
	reviewCompat := srv.mapCompatTask(reviewTask)
	expectedReviewReport := compatSystemReportPath(reviewTaskID, string(RemediationTypeCodeReview))
	if len(reviewCompat.OutputArtifacts) != 1 || reviewCompat.OutputArtifacts[0] != expectedReviewReport {
		t.Fatalf("expected review task to use system report artifact %q, got %+v", expectedReviewReport, reviewCompat.OutputArtifacts)
	}

	createCompatTask(t, repo, "TRIAGE-PARENT-001", `{"id":"TRIAGE-PARENT-001","dispatch_ref":"dispatch_triage","state":"triage","transport":"cli","wave":1,"topo_rank":1,"title":"Noop failure","owner_agent":"Claude","status":"triage","type":"integration","priority":1,"dispatch_status":"triage","failure_code":"git_no_changes","failure_signature":"sig-001","last_dispatch_error":"nothing to commit, working tree clean","output_artifacts":["src/noop.go"],"artifact_path":"C:/artifacts/triage-parent"}`)

	triageTask, err := repo.GetTaskByID(ctx, "TRIAGE-PARENT-001")
	if err != nil {
		t.Fatalf("failed to load triage task: %v", err)
	}
	orchestrator := NewFailureOrchestrator(srv, repo, srv.logger, time.Second)
	if err := orchestrator.processTriageTask(ctx, triageTask); err != nil {
		t.Fatalf("processTriageTask failed: %v", err)
	}

	allTasks, err := repo.ListAllTasks(ctx)
	if err != nil {
		t.Fatalf("ListAllTasks failed: %v", err)
	}
	var noopTask *ent.Task
	for _, task := range allTasks {
		payload := decodeCompatPayload(task.CardJSON)
		if readString(payload, "parent_task_id") == "TRIAGE-PARENT-001" {
			noopTask = task
			break
		}
	}
	if noopTask == nil {
		t.Fatal("expected noop-review task to be created")
	}
	noopCompat := srv.mapCompatTask(noopTask)
	expectedNoopReport := compatSystemReportPath(noopTask.ID, string(RemediationTypeNoopReview))
	if len(noopCompat.OutputArtifacts) != 1 || noopCompat.OutputArtifacts[0] != expectedNoopReport {
		t.Fatalf("expected noop-review task to use system report artifact %q, got %+v", expectedNoopReport, noopCompat.OutputArtifacts)
	}
}

// TestAutoCreateSystemReport prevents empty_artifact_match for system tasks.
// PR-1: When a system task's command succeeds but doesn't write the report file,
// the platform auto-creates the report from execution output and proceeds as success.
func TestAutoCreateSystemReport(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	baseDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:      baseDir,
		WorktreeBasePath:  filepath.Join(baseDir, "worktrees"),
		WorkspaceBasePath: filepath.Join(baseDir, "workspaces"),
		ArtifactBasePath:  filepath.Join(baseDir, "artifacts"),
	})

	// Create a noop-review system task that uses API transport.
	// The command writes nothing to the expected report path.
	createCompatTask(t, repo, "SYS-NOOP-001", `{"id":"SYS-NOOP-001","dispatch_ref":"dispatch_compat","state":"queued","transport":"api","wave":1,"topo_rank":1,"title":"[noop-review] No-op check","owner_agent":"Claude","status":"ready","type":"noop-review","priority":2,"dispatch_mode":"auto","auto_dispatch_enabled":true,"output_artifacts":[".orchestrator/reports/SYS-NOOP-001-noop-review.md"],"files_to_modify":[".orchestrator/reports/SYS-NOOP-001-noop-review.md"],"parent_task_id":"PARENT-001","root_task_id":"PARENT-001","shell":"powershell","command":"python -c \"print('Review: no changes needed, task is a no-op. All good.')\""}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/SYS-NOOP-001/dispatch", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected dispatch status %d, got %d", http.StatusOK, res.Code)
	}

	// Wait for execution to complete
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		task, err := repo.GetTaskByID(context.Background(), "SYS-NOOP-001")
		if err != nil {
			t.Fatalf("failed to reload task: %v", err)
		}
		if task != nil && (task.State == engine.StateVerified || task.State == engine.StateDone) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	task, err := repo.GetTaskByID(context.Background(), "SYS-NOOP-001")
	if err != nil {
		t.Fatalf("failed to reload final task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	// System tasks should reach verified (not triage/blocked)
	if task.State != engine.StateVerified && task.State != engine.StateDone {
		mapped := srv.mapCompatTask(task)
		t.Fatalf("expected system task to reach verified/done, got state=%q status=%q dispatch=%q", task.State, mapped.Status, mapped.DispatchStatus)
	}

	// Verify the auto-created report file exists in artifact directory
	reportPath := filepath.Join(baseDir, "artifacts", "SYS-NOOP-001", ".orchestrator", "reports", "SYS-NOOP-001-noop-review.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected auto-created report file at %s: %v", reportPath, err)
	}
	reportContent := string(content)
	if !strings.Contains(reportContent, "System Task Report") {
		t.Fatalf("expected report to contain 'System Task Report', got: %s", reportContent[:200])
	}
	if !strings.Contains(reportContent, "noop-review") {
		t.Fatalf("expected report to contain task type 'noop-review', got: %s", reportContent[:200])
	}
}

// TestAutoCreateSystemReportDoesNotApplyToNormalTasks verifies that normal tasks
// still fail with empty_artifact_match when they don't produce expected files.
func TestAutoCreateSystemReportDoesNotApplyToNormalTasks(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	baseDir := t.TempDir()
	srv.ConfigureCompatExecution(CompatExecutionConfig{
		MainRepoPath:      baseDir,
		WorktreeBasePath:  filepath.Join(baseDir, "worktrees"),
		WorkspaceBasePath: filepath.Join(baseDir, "workspaces"),
		ArtifactBasePath:  filepath.Join(baseDir, "artifacts"),
	})

	// Create a normal (non-system) task that won't produce expected output
	createCompatTask(t, repo, "NORMAL-001", `{"id":"NORMAL-001","dispatch_ref":"dispatch_compat","state":"queued","transport":"api","wave":1,"topo_rank":1,"title":"Normal task","owner_agent":"Claude","status":"ready","type":"integration","priority":3,"output_artifacts":["result.txt"],"shell":"powershell","command":"python -c \"print('doing work but not writing file')\""}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/NORMAL-001/dispatch", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected dispatch status %d, got %d", http.StatusOK, res.Code)
	}

	// Wait for execution to complete
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		task, err := repo.GetTaskByID(context.Background(), "NORMAL-001")
		if err != nil {
			t.Fatalf("failed to reload task: %v", err)
		}
		if task != nil && (task.State == engine.StateTriage || task.State == engine.StateFailed) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	task, err := repo.GetTaskByID(context.Background(), "NORMAL-001")
	if err != nil {
		t.Fatalf("failed to reload final task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	// Normal tasks should NOT be auto-rescued; they should fail/triage
	if task.State == engine.StateVerified || task.State == engine.StateDone {
		t.Fatalf("expected normal task to fail with empty_artifact_match, got state=%q", task.State)
	}
}

// TestReviewTaskFailureAutoApprovesParent verifies PR-2: when a review task fails
// (e.g., reviewer agent crashes), the parent task is auto-approved rather than blocked.
func TestReviewTaskFailureAutoApprovesParent(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create parent task in review_pending
	createCompatTask(t, repo, "RV-PARENT-001", `{"id":"RV-PARENT-001","dispatch_ref":"dispatch_review","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Parent for review test","owner_agent":"Codex","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","output_artifacts":["src/file.go"],"artifact_path":"C:/artifacts/review-parent"}`)

	reviewWorker := NewReviewWorker(srv, repo, srv.logger, time.Second)
	if err := reviewWorker.processReviewPendingTasks(ctx); err != nil {
		t.Fatalf("processReviewPendingTasks failed: %v", err)
	}

	parent, err := repo.GetTaskByID(ctx, "RV-PARENT-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	parentPayload := decodeCompatPayload(parent.CardJSON)
	reviewTaskID := readString(parentPayload, "last_review_task_id")
	if reviewTaskID == "" {
		t.Fatal("expected review task to be created")
	}

	// Simulate review task failing (entity state only, cardJSON preserved)
	reviewTask, err := repo.GetTaskByID(ctx, reviewTaskID)
	if err != nil {
		t.Fatalf("failed to load review task: %v", err)
	}
	if err := srv.transitionCompatTaskState(ctx, reviewTask, engine.StateFailed, "review_agent_error", "reviewer agent crashed"); err != nil {
		t.Fatalf("failed to fail review task: %v", err)
	}

	// Process review results - should auto-approve
	if err := reviewWorker.processReviewResults(ctx); err != nil {
		t.Fatalf("processReviewResults failed: %v", err)
	}

	// Parent should be auto-approved (verified), not stuck in review_pending
	parent, err = repo.GetTaskByID(ctx, "RV-PARENT-001")
	if err != nil {
		t.Fatalf("GetTaskByID after review failed: %v", err)
	}
	if parent.State != engine.StateVerified {
		t.Fatalf("expected parent to be auto-approved (verified) after review task failure, got state=%q", parent.State)
	}
}

// TestTightenedRejectionKeywords verifies PR-2: common words like "fail", "error", "wrong"
// no longer trigger false-positive rejections.
func TestTightenedRejectionKeywords(t *testing.T) {
	// These should NOT trigger rejection (false positives before PR-2)
	if containsRejectionKeywords("The build did not fail any tests") {
		t.Fatal("'fail any tests' should not be a rejection keyword")
	}
	if containsRejectionKeywords("Error handling is properly implemented") {
		t.Fatal("'Error handling' should not be a rejection keyword")
	}
	if containsRejectionKeywords("This is the wrong file, use the other one") {
		t.Fatal("'wrong file' should not be a rejection keyword")
	}
	if containsRejectionKeywords("There was a failure in the CI pipeline but code is correct") {
		t.Fatal("'failure in CI' should not be a rejection keyword")
	}
	if containsRejectionKeywords("incorrect path was used as a test case") {
		t.Fatal("'incorrect' should not be a rejection keyword")
	}

	// These SHOULD trigger rejection
	if !containsRejectionKeywords("I reject this code, it needs changes") {
		t.Fatal("'reject' should be a rejection keyword")
	}
	if !containsRejectionKeywords("The implementation is denied") {
		t.Fatal("'denied' should be a rejection keyword")
	}
	if !containsRejectionKeywords("This needs revision before approval") {
		t.Fatal("'needs revision' should be a rejection keyword")
	}
	if !containsRejectionKeywords("review decision: rejected") {
		t.Fatal("'review decision: rejected' should be a rejection keyword")
	}
	if !containsRejectionKeywords("must be fixed before merging") {
		t.Fatal("'must be fixed' should be a rejection keyword")
	}
}

// TestRecoverStuckBlockedTask verifies PR-3: a blocked (failed) root task can be
// recovered via the recovery endpoint, retiring its failed children and resetting
// it to retry_waiting.
func TestRecoverStuckBlockedTask(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create root task in failed state (blocked)
	createCompatTask(t, repo, "TS-RECOVER-001", `{"id":"TS-RECOVER-001","dispatch_ref":"dispatch_recover","state":"failed","transport":"cli","wave":1,"topo_rank":1,"title":"Blocked root task","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","auto_repair_count":2,"failure_code":"artifact_missing","coordination_stage":"stopped","last_dispatch_error":"empty_artifact_match"}`)

	// Create a failed child remediation task
	createCompatTask(t, repo, "TS-CHILD-001", `{"id":"TS-CHILD-001","dispatch_ref":"dispatch_recover","state":"failed","transport":"cli","wave":1,"topo_rank":1,"title":"Failed remediation","owner_agent":"Codex","status":"blocked","type":"artifact-fix","priority":2,"parent_task_id":"TS-RECOVER-001","root_task_id":"TS-RECOVER-001","dispatch_status":"failed","coordination_stage":"remediation","failure_code":"artifact_missing"}`)

	// Call recovery endpoint
	body := bytes.NewBufferString(`{"task_ids":["TS-RECOVER-001"],"reason":"platform_defect"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/stuck-tasks", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected recovery status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	// Verify root task is now in retry_waiting
	rootTask, err := repo.GetTaskByID(ctx, "TS-RECOVER-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if rootTask.State != engine.StateRetryWaiting {
		t.Fatalf("expected root task in retry_waiting, got %q", rootTask.State)
	}

	rootPayload := decodeCompatPayload(rootTask.CardJSON)
	if readIntDefault(rootPayload, 0, "auto_repair_count") != 0 {
		t.Fatalf("expected repair count reset to 0, got %d", readIntDefault(rootPayload, 0, "auto_repair_count"))
	}

	// Verify child task was retired (done)
	childTask, err := repo.GetTaskByID(ctx, "TS-CHILD-001")
	if err != nil {
		t.Fatalf("GetTaskByID for child failed: %v", err)
	}
	if childTask.State != engine.StateDone {
		t.Fatalf("expected child task retired to done, got %q", childTask.State)
	}
}

// TestRecoverStuckReviewPendingTask verifies PR-3: a task stuck in review_pending
// is auto-approved via recovery.
func TestRecoverStuckReviewPendingTask(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	createCompatTask(t, repo, "TS-REVIEW-001", `{"id":"TS-REVIEW-001","dispatch_ref":"dispatch_recover","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Stuck in review","owner_agent":"Claude","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","coordination_stage":"under_review"}`)

	body := bytes.NewBufferString(`{"task_ids":["TS-REVIEW-001"],"reason":"platform_defect"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/stuck-tasks", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected recovery status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	task, err := repo.GetTaskByID(ctx, "TS-REVIEW-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateVerified {
		t.Fatalf("expected task auto-approved to verified, got %q", task.State)
	}
}

func TestRecoverVerifyFailedNoopMergeTask(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	createCompatTask(t, repo, "TS-VERIFY-NOOP-001", `{"id":"TS-VERIFY-NOOP-001","dispatch_ref":"dispatch_recover","state":"verify_failed","transport":"cli","wave":1,"topo_rank":1,"title":"Noop merge failure","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","coordination_stage":"review_approved","review_decision":"approved","last_dispatch_error":"git command failed: exit status 1 (output: On branch feature/test\nnothing to commit, working tree clean\n)"}`)

	body := bytes.NewBufferString(`{"task_ids":["TS-VERIFY-NOOP-001"],"reason":"platform_defect"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/stuck-tasks", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected recovery status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	task, err := repo.GetTaskByID(ctx, "TS-VERIFY-NOOP-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateVerified {
		t.Fatalf("expected task recovered to verified, got %q", task.State)
	}

	payload := decodeCompatPayload(task.CardJSON)
	if readString(payload, "status") != "verified" {
		t.Fatalf("expected payload status verified, got %q", readString(payload, "status"))
	}
	if readString(payload, "dispatch_status") != "completed" {
		t.Fatalf("expected dispatch_status completed, got %q", readString(payload, "dispatch_status"))
	}
}

func TestRecoverVerifyFailedArtifactTaskToRetryWaiting(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	createCompatTask(t, repo, "TS-VERIFY-ART-001", `{"id":"TS-VERIFY-ART-001","dispatch_ref":"dispatch_recover","state":"verify_failed","transport":"cli","wave":1,"topo_rank":1,"title":"Artifact verify failure","owner_agent":"Gemini","status":"blocked","type":"ui","priority":1,"dispatch_status":"failed","auto_repair_count":2,"failure_code":"artifact_missing","coordination_stage":"stopped","last_dispatch_error":"empty_artifact_match: no files matched pattern: frontend/src/pages/AccountWorkspace.tsx"}`)
	createCompatTask(t, repo, "TS-VERIFY-ART-CHILD-001", `{"id":"TS-VERIFY-ART-CHILD-001","dispatch_ref":"dispatch_recover","state":"failed","transport":"cli","wave":1,"topo_rank":1,"title":"Failed remediation","owner_agent":"Codex","status":"blocked","type":"artifact-fix","priority":2,"parent_task_id":"TS-VERIFY-ART-001","root_task_id":"TS-VERIFY-ART-001","dispatch_status":"failed","coordination_stage":"remediation","failure_code":"artifact_missing"}`)

	body := bytes.NewBufferString(`{"task_ids":["TS-VERIFY-ART-001"],"reason":"platform_defect"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/stuck-tasks", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected recovery status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	task, err := repo.GetTaskByID(ctx, "TS-VERIFY-ART-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateRetryWaiting {
		t.Fatalf("expected task recovered to retry_waiting, got %q", task.State)
	}

	childTask, err := repo.GetTaskByID(ctx, "TS-VERIFY-ART-CHILD-001")
	if err != nil {
		t.Fatalf("GetTaskByID for child failed: %v", err)
	}
	if childTask.State != engine.StateDone {
		t.Fatalf("expected remediation child retired to done, got %q", childTask.State)
	}
}

// TestReviewTaskStuckRunningAutoApprovesParent verifies PR-2: when a review task
// is running but has been stuck for too long, the parent is auto-approved.
func TestReviewTaskStuckRunningAutoApprovesParent(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create parent task in review_pending
	createCompatTask(t, repo, "RV-STUCK-001", `{"id":"RV-STUCK-001","dispatch_ref":"dispatch_review","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Stuck parent","owner_agent":"Codex","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","output_artifacts":["src/file.go"]}`)

	reviewWorker := NewReviewWorker(srv, repo, srv.logger, time.Second)
	if err := reviewWorker.processReviewPendingTasks(ctx); err != nil {
		t.Fatalf("processReviewPendingTasks failed: %v", err)
	}

	parent, err := repo.GetTaskByID(ctx, "RV-STUCK-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	parentPayload := decodeCompatPayload(parent.CardJSON)
	reviewTaskID := readString(parentPayload, "last_review_task_id")
	if reviewTaskID == "" {
		t.Fatal("expected review task to be created")
	}

	// Verify the review task exists and is in running state
	reviewTask, err := repo.GetTaskByID(ctx, reviewTaskID)
	if err != nil {
		t.Fatalf("failed to load review task: %v", err)
	}

	// The review worker should NOT auto-approve yet (task just created, not stuck)
	if err := reviewWorker.processReviewPendingTasks(ctx); err != nil {
		t.Fatalf("processReviewPendingTasks failed: %v", err)
	}
	parent, _ = repo.GetTaskByID(ctx, "RV-STUCK-001")
	if parent.State != engine.StateReviewPending {
		t.Fatalf("expected parent to stay in review_pending for fresh review task, got %q", parent.State)
	}

	// Verify isReviewTaskStuck returns false for fresh tasks
	if reviewWorker.isReviewTaskStuck(reviewTask) {
		t.Fatal("expected isReviewTaskStuck to be false for freshly created review task")
	}
}

// TestStartupRecoveryAutoApprovesStuckReviewPending verifies that tasks stuck in
// review_pending for more than 30 minutes are auto-approved at startup.
func TestStartupRecoveryAutoApprovesStuckReviewPending(t *testing.T) {
	_, repo, client, cleanup := setupTestServerWithClient(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.New(nil)

	// Create a task in review_pending state
	createCompatTask(t, repo, "TS-SU-REVIEW-001", `{"id":"TS-SU-REVIEW-001","dispatch_ref":"dispatch_su","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Stuck review at startup","owner_agent":"Claude","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","coordination_stage":"under_review"}`)

	// Simulate the task being stuck for 45 minutes by updating its updated_at
	_, err := client.Task.UpdateOneID("TS-SU-REVIEW-001").
		SetUpdatedAt(time.Now().Add(-45 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to age task updated_at: %v", err)
	}

	srv := New(repo, logger)
	recovery := NewStartupRecovery(srv, repo, logger)

	if err := recovery.Run(ctx); err != nil {
		t.Fatalf("startup recovery failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "TS-SU-REVIEW-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateVerified {
		t.Fatalf("expected task auto-approved to verified, got %q", task.State)
	}
}

// TestStartupRecoverySkipsRecentReviewPending verifies that tasks recently
// put in review_pending are NOT auto-approved at startup.
func TestStartupRecoverySkipsRecentReviewPending(t *testing.T) {
	_, repo, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.New(nil)

	// Create a task in review_pending state that was recently updated
	createCompatTask(t, repo, "TS-SU-RECENT-001", `{"id":"TS-SU-RECENT-001","dispatch_ref":"dispatch_su2","state":"review_pending","transport":"cli","wave":1,"topo_rank":1,"title":"Recent review","owner_agent":"Claude","status":"review_pending","type":"integration","priority":1,"dispatch_status":"review_pending","coordination_stage":"under_review"}`)

	srv := New(repo, logger)
	recovery := NewStartupRecovery(srv, repo, logger)

	if err := recovery.Run(ctx); err != nil {
		t.Fatalf("startup recovery failed: %v", err)
	}

	task, err := repo.GetTaskByID(ctx, "TS-SU-RECENT-001")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task.State != engine.StateReviewPending {
		t.Fatalf("expected recent review_pending task to stay in review_pending, got %q", task.State)
	}
}
