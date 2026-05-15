---
phase: 01-foundation
plan: "03"
type: execute
wave: 3
depends_on:
  - 01-foundation-02
files_modified:
  - internal/store/repository.go
  - internal/store/repository_test.go
  - internal/server/server.go
  - internal/server/server_test.go
autonomous: true
requirements:
  - API-01
gap_closure: true
must_haves:
  truths:
    - "Client can delete an existing task through REST and the task is no longer returned by GET /api/tasks/{id}"
    - "Client can query persisted state-transition events for a task through REST and receive real database rows in stable order"
    - "Phase 01 task API now covers create, read, update, delete, wave operations, state actions, and persisted event query for API-01"
  artifacts:
    - path: "internal/store/repository.go"
      provides: "Repository delete + task-event query methods for API-01 gap closure"
      exports:
        - "DeleteTask"
        - "ListEventsByTaskID"
    - path: "internal/server/server.go"
      provides: "DELETE /api/tasks/{id} and GET /api/tasks/{id}/events routes and handlers"
    - path: "internal/store/repository_test.go"
      provides: "Repository regression coverage for task delete semantics and event ordering"
    - path: "internal/server/server_test.go"
      provides: "HTTP regression coverage for delete and persisted event query routes"
  key_links:
    - from: "internal/server/server.go"
      to: "internal/store/repository.go"
      via: "handleDeleteTask -> repo.DeleteTask"
      pattern: "handleDeleteTask|DeleteTask"
    - from: "internal/server/server.go"
      to: "internal/store/repository.go"
      via: "handleListTaskEvents -> repo.ListEventsByTaskID"
      pattern: "handleListTaskEvents|ListEventsByTaskID"
    - from: "internal/store/repository.go"
      to: "ent event rows"
      via: "ListEventsByTaskID ordered query"
      pattern: "FieldTimestamp|FieldEventID|task_id"
---

<objective>
Close the remaining Phase 01/API-01 verification gaps without replanning already-complete persistence work.

Purpose: satisfy full REST coverage for the existing task API by adding task deletion and persisted event query on top of the verified repository/server baseline.
Output: one repository extension, one server route extension, and matching regression tests proving delete + event query behavior end-to-end.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
@E:/04-Claude/Runtime/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/REQUIREMENTS.md
@.planning/phases/01-foundation/01-CONTEXT.md
@.planning/phases/01-foundation/01-VERIFICATION.md
@.planning/phases/01-foundation/01-foundation-02-SUMMARY.md
@internal/store/repository.go
@internal/store/repository_test.go
@internal/server/server.go
@internal/server/server_test.go
@ent/schema/task.go
@ent/schema/event.go
@ent/schema/agent_call.go

<interfaces>
From `internal/store/repository.go`:
```go
type EventData struct {
    EventID   string
    ProjectID string
    TaskID    string
    EventType string
    FromState string
    ToState   string
    Timestamp time.Time
    Reason    string
    Attempt   int
    Transport string
    RunnerID  string
    Details   string
}

func (r *Repository) GetTaskByID(ctx context.Context, taskID string) (*ent.Task, error)
func (r *Repository) ListTasksByDispatchRef(ctx context.Context, dispatchRef string) ([]*ent.Task, error)
func (r *Repository) UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string, eventData *EventData) error
func (r *Repository) CreateEvent(ctx context.Context, eventData *EventData) error
```

From `internal/server/server.go`:
```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

Existing route shape to preserve:
```go
r.Route("/tasks", func(r chi.Router) {
    r.Get("/", s.handleListTasks)
    r.Get("/stats", s.handleTaskStats)
    r.Post("/", s.handleCreateTask)
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", s.handleGetTask)
        r.Put("/", s.handleUpdateTask)
        r.Post("/cancel", s.handleCancelTask)
        r.Post("/retry", s.handleRetryTask)
        r.Get("/agent-calls", s.handleGetTaskAgentCalls)
    })
})
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add repository delete and persisted event query support</name>
  <files>internal/store/repository.go, internal/store/repository_test.go</files>
  <read_first>
    - internal/store/repository.go
    - internal/store/repository_test.go
    - ent/schema/task.go
    - ent/schema/event.go
    - ent/schema/agent_call.go
    - .planning/phases/01-foundation/01-CONTEXT.md
    - .planning/phases/01-foundation/01-VERIFICATION.md
  </read_first>
  <behavior>
    - Test 1: deleting an existing task removes the task row and its related `events.task_id` and `agent_calls.task_id` rows, then returns `deleted=true`
    - Test 2: deleting a missing task returns `deleted=false` without raising an error
    - Test 3: listing task events returns persisted rows for one task only, ordered by `timestamp ASC, event_id ASC`
  </behavior>
  <action>
    Extend the existing `internal/store` layer per D-01 and D-02; do not introduce a new package or alternate persistence abstraction.

    In `internal/store/repository.go` add these exact repository methods:
    - `func (r *Repository) DeleteTask(ctx context.Context, taskID string) (bool, error)`
    - `func (r *Repository) ListEventsByTaskID(ctx context.Context, taskID string) ([]*ent.Event, error)`

    Implement `DeleteTask` with concrete semantics:
    - Load the task first with `r.client.Task.Get(ctx, taskID)`; if `ent.IsNotFound(err)`, return `(false, nil)`
    - Use `r.WithTx` so related row deletion and task deletion happen atomically
    - Inside the transaction, delete related rows in this order before deleting the task row:
      1. `tx.Event.Delete().Where(event.TaskID(taskID)).Exec(ctx)`
      2. `tx.AgentCall.Delete().Where(agentcall.TaskID(taskID)).Exec(ctx)`
      3. `tx.Task.DeleteOneID(taskID).Exec(ctx)`
    - If the loaded task has non-empty `workspace_path` or `artifact_path`, remove those filesystem paths with `os.RemoveAll` before the transaction delete, treating `os.ErrNotExist` as success
    - Return `(true, nil)` only when the task existed and deletion completed

    Implement `ListEventsByTaskID` with concrete query behavior:
    - Query `r.client.Event.Query().Where(event.TaskID(taskID))`
    - Order results by `timestamp` ascending and then `event_id` ascending for stable API output
    - Return an empty slice, not `nil`, when a task has no events

    Add repository tests in `internal/store/repository_test.go` using the existing in-memory SQLite harness:
    - `TestDeleteTaskRemovesTaskAndRelatedRows`
    - `TestDeleteTaskReturnsFalseWhenMissing`
    - `TestListEventsByTaskIDReturnsOrderedEvents`

    Seed events via `CreateEvent` and/or `UpdateTaskState`, then assert exact postconditions with repository queries so the API task in Wave 3 can rely on these contracts.
  </action>
  <acceptance_criteria>
    - `internal/store/repository.go` contains `func (r *Repository) DeleteTask(ctx context.Context, taskID string) (bool, error)`
    - `internal/store/repository.go` contains `func (r *Repository) ListEventsByTaskID(ctx context.Context, taskID string) ([]*ent.Event, error)`
    - `internal/store/repository.go` deletes `Event`, `AgentCall`, and `Task` rows for the same `taskID` inside `WithTx`
    - `internal/store/repository_test.go` contains `TestDeleteTaskRemovesTaskAndRelatedRows`
    - `internal/store/repository_test.go` contains `TestDeleteTaskReturnsFalseWhenMissing`
    - `internal/store/repository_test.go` contains `TestListEventsByTaskIDReturnsOrderedEvents`
  </acceptance_criteria>
  <verify>
    <automated>go test ./internal/store -run 'Test(DeleteTaskRemovesTaskAndRelatedRows|DeleteTaskReturnsFalseWhenMissing|ListEventsByTaskIDReturnsOrderedEvents)$' -count=1</automated>
  </verify>
  <done>Repository exposes stable delete + event-query contracts that cover the verification gap and are proven by focused store tests.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Expose DELETE task and GET task events routes in the chi API</name>
  <files>internal/server/server.go, internal/server/server_test.go</files>
  <read_first>
    - internal/server/server.go
    - internal/server/server_test.go
    - internal/store/repository.go
    - internal/store/repository_test.go
    - .planning/phases/01-foundation/01-CONTEXT.md
    - .planning/phases/01-foundation/01-VERIFICATION.md
  </read_first>
  <behavior>
    - Test 1: `DELETE /api/tasks/{id}` returns 200 with the deleted id for an existing task, and later `GET /api/tasks/{id}` returns 404
    - Test 2: `DELETE /api/tasks/{id}` returns 404 with `{"success":false,"error":"task not found"}` for a missing task
    - Test 3: `GET /api/tasks/{id}/events` returns persisted event rows for that task in stable chronological order
    - Test 4: `GET /api/tasks/{id}/events` returns 404 when the task id does not exist
  </behavior>
  <action>
    Extend the existing chi router in `internal/server/server.go` per D-01 and D-02; preserve the current `/api/tasks/{id}` route group rather than adding a parallel namespace.

    Make these concrete route additions under the existing `/api/tasks/{id}` block:
    - `r.Delete("/", s.handleDeleteTask)`
    - `r.Get("/events", s.handleListTaskEvents)`

    Add handler implementations with these exact behaviors:
    - `handleDeleteTask` calls `s.repo.DeleteTask(ctx, id)`
      - when repository returns `(false, nil)`, respond with `http.StatusNotFound` and `APIResponse{Success:false, Error:"task not found"}`
      - when repository returns `(true, nil)`, respond with `http.StatusOK` and `APIResponse{Success:true, Data: map[string]interface{}{"id": id, "deleted": true}}`
    - `handleListTaskEvents` first checks `s.repo.GetTaskByID(ctx, id)`
      - if task is missing, return the same 404 envelope as `handleGetTask`
      - otherwise call `s.repo.ListEventsByTaskID(ctx, id)` and return `APIResponse{Success:true, Data: events}` with `http.StatusOK`
    - Keep request/response style aligned with existing `handleGetTask`, `handleUpdateTask`, and `handleGetTaskAgentCalls`; do not add a new response envelope or pagination format for this gap closure

    Add focused HTTP tests in `internal/server/server_test.go`:
    - `TestHandleDeleteTaskRemovesTaskAndReturns404Afterwards`
    - `TestHandleDeleteTaskReturns404WhenMissing`
    - `TestHandleListTaskEventsReturnsPersistedTransitions`
    - `TestHandleListTaskEventsReturns404WhenTaskMissing`

    In the event-list test, create the task, write at least two persisted events with distinct timestamps and ids, call `GET /api/tasks/{id}/events`, and assert the returned JSON order and exact `event_type` / `from_state` / `to_state` values.
  </action>
  <acceptance_criteria>
    - `internal/server/server.go` route setup contains `r.Delete("/", s.handleDeleteTask)` under `/api/tasks/{id}`
    - `internal/server/server.go` route setup contains `r.Get("/events", s.handleListTaskEvents)` under `/api/tasks/{id}`
    - `internal/server/server.go` contains `func (s *Server) handleDeleteTask(`
    - `internal/server/server.go` contains `func (s *Server) handleListTaskEvents(`
    - `internal/server/server_test.go` contains `TestHandleDeleteTaskRemovesTaskAndReturns404Afterwards`
    - `internal/server/server_test.go` contains `TestHandleDeleteTaskReturns404WhenMissing`
    - `internal/server/server_test.go` contains `TestHandleListTaskEventsReturnsPersistedTransitions`
    - `internal/server/server_test.go` contains `TestHandleListTaskEventsReturns404WhenTaskMissing`
  </acceptance_criteria>
  <verify>
    <automated>go test ./internal/server -run 'Test(HandleDeleteTaskRemovesTaskAndReturns404Afterwards|HandleDeleteTaskReturns404WhenMissing|HandleListTaskEventsReturnsPersistedTransitions|HandleListTaskEventsReturns404WhenTaskMissing)$' -count=1</automated>
  </verify>
  <done>HTTP API exposes the missing delete and persisted event query routes with repository-backed behavior and regression tests proving API-01 coverage.</done>
</task>

</tasks>

<verification>
- Run `go test ./internal/store -run 'Test(DeleteTaskRemovesTaskAndRelatedRows|DeleteTaskReturnsFalseWhenMissing|ListEventsByTaskIDReturnsOrderedEvents)$' -count=1`
- Run `go test ./internal/server -run 'Test(HandleDeleteTaskRemovesTaskAndReturns404Afterwards|HandleDeleteTaskReturns404WhenMissing|HandleListTaskEventsReturnsPersistedTransitions|HandleListTaskEventsReturns404WhenTaskMissing)$' -count=1`
- Run `go test ./internal/store ./internal/server -count=1`
- Confirm `DELETE /api/tasks/{id}` and `GET /api/tasks/{id}/events` are present in `internal/server/server.go`
</verification>

<success_criteria>
API-01 no longer has Phase 01 coverage gaps: the repository can delete tasks and read persisted task events, and the chi REST surface exposes both operations with automated regression coverage.
</success_criteria>

<output>
After completion, create `.planning/phases/01-foundation/01-foundation-03-SUMMARY.md`
</output>
