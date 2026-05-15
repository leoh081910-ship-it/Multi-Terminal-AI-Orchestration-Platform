---
phase: 01-foundation
plan: "03"
subsystem: api
tags: [go, sqlite, ent, chi, persistence, rest-api, gap-closure]

requires:
  - phase: 01-foundation
    provides: Repository, server routes, and verification baseline from 01-foundation-02
provides:
  - Repository delete support for tasks and related rows
  - Repository query support for persisted task events in stable order
  - chi REST routes for task delete and task event query
  - Focused repository and HTTP regression coverage for API-01 gap closure
affects: [api, database, server, orchestration, phase-01]

tech-stack:
  added: []
  patterns:
    - Repository deletes related rows inside WithTx before deleting the task row
    - Task event queries return persisted rows ordered by timestamp then event_id for stable API output
    - HTTP handlers preserve the existing APIResponse envelope and route grouping style

key-files:
  created:
    - .planning/phases/01-foundation/01-foundation-03-SUMMARY.md
  modified:
    - internal/store/repository.go
    - internal/store/repository_test.go
    - internal/server/server.go
    - internal/server/server_test.go

key-decisions:
  - "Delete semantics remain in the existing repository layer instead of introducing a new persistence abstraction."
  - "Task event query output is stabilized by ordering on timestamp ascending and then event_id ascending."
  - "The API gap was closed inside the existing /api/tasks/{id} route group to preserve current surface shape."

patterns-established:
  - "Repository delete: DeleteTask removes related event and agent_call rows, then deletes the task row in one transaction."
  - "REST delete contract: DELETE /api/tasks/{id} returns {id, deleted:true} on success and the standard 404 envelope when missing."
  - "REST event query contract: GET /api/tasks/{id}/events returns persisted event rows for that task only, in stable chronological order."

requirements-completed: [API-01]

duration: 27min
completed: 2026-05-15
---

# Phase 01 Plan 03: API-01 Gap Closure Summary

**Closed the remaining Phase 01 REST coverage gaps by adding task deletion, persisted task-event query support, and focused regression coverage.**

## Performance

- **Duration:** 27 min
- **Tasks:** 2
- **Files modified:** 4 source files + 1 summary file

## Accomplishments

- Added `DeleteTask` and `ListEventsByTaskID` to `internal/store/repository.go` so the repository can remove a task plus related rows and return persisted event rows in stable order.
- Added repository regression tests in `internal/store/repository_test.go` covering delete semantics, missing-task delete behavior, and chronological task-event ordering.
- Added `DELETE /api/tasks/{id}` and `GET /api/tasks/{id}/events` to `internal/server/server.go` using the existing `/api/tasks/{id}` route group and `APIResponse` envelope.
- Added HTTP regression tests in `internal/server/server_test.go` covering delete success, delete 404, task-event query success, and task-event query 404.
- Re-ran Phase 01 verification and confirmed the previous API-01 gaps are now closed.

## Files Created/Modified

- `internal/store/repository.go` - Added `DeleteTask` and `ListEventsByTaskID`.
- `internal/store/repository_test.go` - Added focused delete and ordered-event query tests.
- `internal/server/server.go` - Added delete and task-event query routes and handlers.
- `internal/server/server_test.go` - Added focused HTTP regression coverage for the new routes.
- `.planning/phases/01-foundation/01-foundation-03-SUMMARY.md` - Created this execution summary.

## Verification

- `go test ./internal/store -run 'Test(DeleteTaskRemovesTaskAndRelatedRows|DeleteTaskReturnsFalseWhenMissing|ListEventsByTaskIDReturnsOrderedEvents)$' -count=1` passed.
- `go test ./internal/server -run 'Test(HandleDeleteTaskRemovesTaskAndReturns404Afterwards|HandleDeleteTaskReturns404WhenMissing|HandleListTaskEventsReturnsPersistedTransitions|HandleListTaskEventsReturns404WhenTaskMissing)$' -count=1` passed.
- `go test ./internal/store ./internal/server -count=1` passed.
- `.planning/phases/01-foundation/01-VERIFICATION.md` was updated to `status: passed` with `score: 7/7 must-haves verified`.

## Issues Encountered

- The initial `DeleteTask` implementation used single-value assignment for ent delete builders whose `Exec` methods return `(int, error)`, causing a compile failure; this was corrected before re-running tests.
- A previously failed subagent execution had already left partial test coverage in place, so the remaining work was completed manually and verified in the main worktree.

## Deviations from Plan

None. The implementation stayed inside the planned repository and server layers and preserved the existing route shape.

## Next Phase Readiness

- Phase 01 now has full planned repository and REST coverage for task CRUD, state actions, wave operations, and persisted task-event query.
- Phase 01 verification is green and no remaining blockers were reported in the updated verification file.

---
*Phase: 01-foundation*
*Completed: 2026-05-15*
