package server

import (
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
