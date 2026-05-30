package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/engine"
)

func TestCompatTriageApproveTransitionsToVerified(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-APPROVE-001", `{"id":"TRIAGE-APPROVE-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Approve blocked task","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","coordination_stage":"stopped"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/tasks/TRIAGE-APPROVE-001/approve", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected approve status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	assertCompatTaskStateAndPayload(t, repo, "TRIAGE-APPROVE-001", engine.StateVerified, map[string]interface{}{
		"status":             "verified",
		"dispatch_status":    "completed",
		"coordination_stage": "manually_approved",
	})
}

func TestCompatTriageRetryTransitionsToRetryWaitingAndResetsRepairCount(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-RETRY-001", `{"id":"TRIAGE-RETRY-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Retry blocked task","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","coordination_stage":"stopped","auto_repair_count":2,"last_rejection_reason":"same failure"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/tasks/TRIAGE-RETRY-001/retry", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected retry status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	assertCompatTaskStateAndPayload(t, repo, "TRIAGE-RETRY-001", engine.StateRetryWaiting, map[string]interface{}{
		"status":                "ready",
		"dispatch_status":       "failed",
		"coordination_stage":    "manual_retry",
		"auto_repair_count":     float64(0),
		"last_rejection_reason": "",
	})
}

func TestCompatTriageWonFixTransitionsToDone(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-WONTFIX-001", `{"id":"TRIAGE-WONTFIX-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Wonfix blocked task","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","coordination_stage":"stopped"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/tasks/TRIAGE-WONTFIX-001/wontfix", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected wontfix status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	assertCompatTaskStateAndPayload(t, repo, "TRIAGE-WONTFIX-001", engine.StateDone, map[string]interface{}{
		"status":             "done",
		"dispatch_status":    "completed",
		"coordination_stage": "wont_fix",
	})
}

func TestCompatTriageBatchApproveTransitionsPayloadState(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-BATCH-APPROVE-001", `{"id":"TRIAGE-BATCH-APPROVE-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Batch approve one","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed"}`)
	createCompatTask(t, repo, "TRIAGE-BATCH-APPROVE-002", `{"id":"TRIAGE-BATCH-APPROVE-002","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":2,"title":"Batch approve two","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/batch", bytes.NewBufferString(`{"task_ids":["TRIAGE-BATCH-APPROVE-001","TRIAGE-BATCH-APPROVE-002"],"action":"approve"}`))
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected batch approve status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	for _, id := range []string{"TRIAGE-BATCH-APPROVE-001", "TRIAGE-BATCH-APPROVE-002"} {
		assertCompatTaskStateAndPayload(t, repo, id, engine.StateVerified, map[string]interface{}{
			"status":             "verified",
			"dispatch_status":    "completed",
			"coordination_stage": "batch_approve",
		})
	}
}

func TestCompatTriageBatchRetryTransitionsPayloadStateAndResetsRepairCount(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-BATCH-RETRY-001", `{"id":"TRIAGE-BATCH-RETRY-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Batch retry","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed","auto_repair_count":2,"last_rejection_reason":"same failure"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/batch", bytes.NewBufferString(`{"task_ids":["TRIAGE-BATCH-RETRY-001"],"action":"retry"}`))
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected batch retry status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	assertCompatTaskStateAndPayload(t, repo, "TRIAGE-BATCH-RETRY-001", engine.StateRetryWaiting, map[string]interface{}{
		"status":                "ready",
		"dispatch_status":       "failed",
		"coordination_stage":    "batch_retry",
		"auto_repair_count":     float64(0),
		"last_rejection_reason": "",
	})
}

func TestCompatTriageBatchWonFixTransitionsPayloadState(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-BATCH-WONTFIX-001", `{"id":"TRIAGE-BATCH-WONTFIX-001","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Batch wontfix","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"dispatch_status":"failed"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/default/triage/batch", bytes.NewBufferString(`{"task_ids":["TRIAGE-BATCH-WONTFIX-001"],"action":"wontfix"}`))
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected batch wontfix status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	assertCompatTaskStateAndPayload(t, repo, "TRIAGE-BATCH-WONTFIX-001", engine.StateDone, map[string]interface{}{
		"status":             "done",
		"dispatch_status":    "completed",
		"coordination_stage": "batch_wontfix",
	})
}

func assertCompatTaskStateAndPayload(t *testing.T, repo compatTaskGetter, taskID string, wantState string, wantPayload map[string]interface{}) {
	t.Helper()

	task, err := repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if task == nil {
		t.Fatalf("task %s not found", taskID)
	}
	if task.State != wantState {
		t.Fatalf("task state = %q, want %q; card_json=%s", task.State, wantState, task.CardJSON)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.CardJSON), &payload); err != nil {
		t.Fatalf("unmarshal card_json: %v", err)
	}
	for key, want := range wantPayload {
		if got := payload[key]; got != want {
			t.Fatalf("payload[%s] = %#v, want %#v; card_json=%s", key, got, want, task.CardJSON)
		}
	}
}

type compatTaskGetter interface {
	GetTaskByID(context.Context, string) (*ent.Task, error)
}
