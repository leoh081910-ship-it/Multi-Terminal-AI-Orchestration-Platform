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

func TestCompatTriageLineageReturnsRootTimeline(t *testing.T) {
	srv, repo, cleanup := setupTestServer(t)
	defer cleanup()

	createCompatTask(t, repo, "TRIAGE-LINEAGE-ROOT", `{"id":"TRIAGE-LINEAGE-ROOT","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":1,"title":"Root task","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"root_task_id":"TRIAGE-LINEAGE-ROOT","dispatch_status":"failed","result_summary":"root summary"}`)
	createCompatTask(t, repo, "TRIAGE-LINEAGE-REVIEW", `{"id":"TRIAGE-LINEAGE-REVIEW","project_id":"default","dispatch_ref":"dispatch_triage","state":"done","transport":"cli","wave":1,"topo_rank":2,"title":"Review task","owner_agent":"Reviewer","status":"done","type":"code-review","priority":1,"parent_task_id":"TRIAGE-LINEAGE-ROOT","root_task_id":"TRIAGE-LINEAGE-ROOT","review_decision":"rejected","result_summary":"review found missing validation"}`)
	createCompatTask(t, repo, "TRIAGE-LINEAGE-REWORK", `{"id":"TRIAGE-LINEAGE-REWORK","project_id":"default","dispatch_ref":"dispatch_triage","state":"retry_waiting","transport":"cli","wave":1,"topo_rank":3,"title":"Rework task","owner_agent":"Claude","status":"ready","type":"rework","priority":1,"parent_task_id":"TRIAGE-LINEAGE-ROOT","root_task_id":"TRIAGE-LINEAGE-ROOT","result_summary":"rework generated"}`)
	createCompatTask(t, repo, "TRIAGE-LINEAGE-TRIAGE", `{"id":"TRIAGE-LINEAGE-TRIAGE","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":4,"title":"Manual triage","owner_agent":"Claude","status":"blocked","type":"triage","priority":1,"parent_task_id":"TRIAGE-LINEAGE-ROOT","root_task_id":"TRIAGE-LINEAGE-ROOT","result_summary":"manual escalation"}`)
	createCompatTask(t, repo, "TRIAGE-LINEAGE-OTHER", `{"id":"TRIAGE-LINEAGE-OTHER","project_id":"default","dispatch_ref":"dispatch_triage","state":"blocked","transport":"cli","wave":1,"topo_rank":5,"title":"Other root","owner_agent":"Claude","status":"blocked","type":"integration","priority":1,"root_task_id":"TRIAGE-LINEAGE-OTHER"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/default/triage/tasks/TRIAGE-LINEAGE-REVIEW/lineage", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected lineage status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}

	var got triageLineageResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal lineage response: %v", err)
	}
	if got.Original == nil {
		t.Fatalf("expected original timeline entry, got nil; body=%s", res.Body.String())
	}
	if got.Original.TaskID != "TRIAGE-LINEAGE-ROOT" {
		t.Fatalf("original task_id = %q, want TRIAGE-LINEAGE-ROOT", got.Original.TaskID)
	}
	if len(got.Timeline) != 4 {
		t.Fatalf("timeline length = %d, want 4; timeline=%#v", len(got.Timeline), got.Timeline)
	}

	wantTypes := map[string]string{
		"TRIAGE-LINEAGE-ROOT":   "original",
		"TRIAGE-LINEAGE-REVIEW": "review",
		"TRIAGE-LINEAGE-REWORK": "rework",
		"TRIAGE-LINEAGE-TRIAGE": "triage",
	}
	seen := make(map[string]triageTimelineEntry, len(got.Timeline))
	for _, entry := range got.Timeline {
		seen[entry.TaskID] = entry
		if wantType, ok := wantTypes[entry.TaskID]; ok && entry.Type != wantType {
			t.Fatalf("entry %s type = %q, want %q", entry.TaskID, entry.Type, wantType)
		}
		if entry.TaskID == "TRIAGE-LINEAGE-OTHER" {
			t.Fatalf("lineage response included unrelated root task")
		}
		if entry.CreatedAt == "" || entry.UpdatedAt == "" {
			t.Fatalf("entry %s missing timestamps: %#v", entry.TaskID, entry)
		}
	}
	for id := range wantTypes {
		if _, ok := seen[id]; !ok {
			t.Fatalf("timeline missing %s; timeline=%#v", id, got.Timeline)
		}
	}
	if seen["TRIAGE-LINEAGE-REVIEW"].Decision != "rejected" {
		t.Fatalf("review decision = %q, want rejected", seen["TRIAGE-LINEAGE-REVIEW"].Decision)
	}
	if seen["TRIAGE-LINEAGE-REVIEW"].Summary != "review found missing validation" {
		t.Fatalf("review summary = %q, want review found missing validation", seen["TRIAGE-LINEAGE-REVIEW"].Summary)
	}
}

func TestCompatTriageLineageMissingTaskReturnsNotFound(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/default/triage/tasks/TRIAGE-LINEAGE-MISSING/lineage", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected missing lineage status %d, got %d body=%s", http.StatusNotFound, res.Code, res.Body.String())
	}
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
