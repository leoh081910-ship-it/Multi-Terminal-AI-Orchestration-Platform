---
phase: 01-foundation
verified: 2026-05-15T15:32:37Z
status: gaps_found
score: 6/7 must-haves verified
re_verification:
  previous_status: gaps_found
  previous_score: 5/7
  gaps_closed:
    - "Task Card upsert persists card_json as the business field source"
  gaps_remaining:
    - "Task CRUD persists end-to-end / API-01 full REST coverage"
  regressions: []
gaps:
  - truth: "Task CRUD and API-01 REST coverage persist end-to-end"
    status: partial
    reason: "Create, read, update, list, state transition, and wave endpoints are implemented, but task delete and event query endpoints required by API-01 are not present."
    artifacts:
      - path: "internal/store/repository.go"
        issue: "No DeleteTask repository method found."
      - path: "internal/server/server.go"
        issue: "No DELETE /api/tasks/{id} route and no REST event query route found."
    missing:
      - "Add repository delete support for tasks or explicitly narrow API-01/Phase 1 CRUD scope."
      - "Add DELETE /api/tasks/{id} if API-01 task CRUD means full create/read/update/delete."
      - "Add event query endpoint(s), such as GET /api/tasks/{id}/events or dispatch-scoped event listing, to satisfy API-01 event query."
---

# Phase 1: Foundation Verification Report

**Phase Goal:** Database persistence and project infrastructure operational
**Verified:** 2026-05-15T15:32:37Z
**Status:** gaps_found
**Re-verification:** Yes — after previous gap closure

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Go module initialized with required dependencies and generated ent code | ✓ VERIFIED | `go.mod`, `go.sum`, `ent/generate.go`, and generated `ent/ent.go` exist; gsd artifact verification passed 6/6 for Plan 01. |
| 2 | SQLite creates `tasks`, `events`, `waves` tables with required schemas | ✓ VERIFIED | `ent/schema/task.go`, `ent/schema/event.go`, and `ent/schema/wave.go` define required PERS-01/PERS-03 fields; `go test ./...` passed against generated ent code. |
| 3 | Server entry point wires SQLite, migrations, repository, and chi HTTP server | ✓ VERIFIED | `cmd/server/main.go` opens SQLite, runs `client.Schema.Create`, constructs `store.NewRepository`, then calls `server.New`; `go test ./...` passed. |
| 4 | Task Card upsert persists `card_json` as the business field source | ✓ VERIFIED | `repository.go` derives create/update values via `deriveTaskCard` from `CardJSON`; `BuildTaskView` reconstructs API views from `card_json`; tests cover create, update, invalid JSON rejection, get/list response mapping. |
| 5 | Event logging records state transitions atomically with task updates | ✓ VERIFIED | `UpdateTaskState` wraps task update and event create inside `WithTx`; tests verify transition state, retry count/terminal_at behavior, and event row attempt values. |
| 6 | Wave CRUD operations work with `(dispatch_ref, wave)` uniqueness enforcement | ✓ VERIFIED | `UpsertWave`, `GetWave`, and `SealWave` exist and are wired to `/api/dispatches/{dispatchRef}/waves`; `ent/schema/wave.go` defines a unique index on `(project_id, dispatch_ref, wave)`, which is a project-scoped superset of the required uniqueness. |
| 7 | Task CRUD and API-01 REST coverage persist end-to-end | ✗ FAILED | Task create/read/update/list and wave/state endpoints exist, but no task delete route/method and no event query REST route were found. |

**Score:** 6/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `go.mod` | Go module definition with required backend dependencies | ✓ VERIFIED | Plan 01 artifact verification passed; full Go test suite passed. |
| `ent/schema/task.go` | Task schema per PERS-01/PERS-04 | ✓ VERIFIED | Required persistence columns exist, including `card_json`; additional later-phase columns are present and non-blocking. |
| `ent/schema/event.go` | Event schema per PERS-02 | ✓ VERIFIED | Required event fields exist, including event id, state fields, attempt, transport, runner, details. |
| `ent/schema/wave.go` | Wave schema per PERS-03 | ✓ VERIFIED | Required wave fields exist; uniqueness is enforced with project scoping. |
| `ent/generate.go` / `ent/ent.go` | Ent generation directive and generated client | ✓ VERIFIED | Both artifacts exist; generated code compiles under `go test ./...`. |
| `internal/store/repository.go` | Repository, transaction support, task/event/wave operations | ⚠️ PARTIAL | Core persistence and PERS-04/PERS-05 are implemented; task delete is missing if API-01 CRUD is interpreted literally. |
| `internal/server/server.go` | chi router exposing health, task, dispatch, wave APIs | ⚠️ PARTIAL | Health, create/read/update/list, retry/cancel, dispatch task list, and wave create/get/seal routes are present; no task delete route and no event query route. |
| `cmd/server/main.go` | Server entry point with SQLite WAL/busy_timeout and graceful shutdown | ✓ VERIFIED | SQLite DSN includes `journal_mode(WAL)` and `busy_timeout(5000)`; repository is injected into `server.New`; HTTP server starts with graceful shutdown wiring. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `ent/generate.go` | generated ent package | go generate / generated files | ✓ WIRED | Static gsd key-link matcher could not infer generation, but generated ent files exist and compile. |
| `cmd/server/main.go` | SQLite database | `sql.Open("sqlite", "file:...?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")` | ✓ WIRED | WAL and busy timeout are configured in the actual server entry point. |
| `cmd/server/main.go` | `internal/store/repository.go` | `store.NewRepository(client, db, &log.Logger)` | ✓ WIRED | Repository is constructed after migration and before server setup. |
| `cmd/server/main.go` | `internal/server/server.go` | `server.New(repo, log.Logger)` | ✓ WIRED | Server receives repository dependency and exposes the HTTP handler. |
| `internal/server/server.go` | `internal/store/repository.go` | Handler calls into repository methods | ✓ WIRED | Task and wave handlers call `CreateTask`, `GetTaskByID`, `UpdateTask`, `ListTasksByDispatchRef`, `UpsertWave`, `GetWave`, and `SealWave`. |
| `internal/store/repository.go` | `events` table | `UpdateTaskState` transaction | ✓ WIRED | Event creation happens in the same transaction closure as task state update. |
| `internal/server/server.go` | event query API | REST route for persisted events | ✗ NOT WIRED | No event query route was found in `server.go` or related server route files. |
| `internal/server/server.go` | task delete API | DELETE `/api/tasks/{id}` | ✗ NOT WIRED | No task delete route was found in task routes. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `internal/store/repository.go` task create/update | `TaskCard` business fields | `deriveTaskCard` parses `card.CardJSON` | Yes | ✓ FLOWING |
| `internal/server/server.go` task responses | `store.TaskView` | `store.BuildTaskView(task)` parsing persisted `CardJSON` plus runtime state columns | Yes | ✓ FLOWING |
| `internal/server/server.go` wave responses | `ent.Wave` rows | Repository queries SQLite through ent | Yes | ✓ FLOWING |
| `internal/store/repository.go` event writes | `EventData` | `UpdateTaskState` transaction inserts `Event` row with updated task retry count | Yes | ✓ FLOWING |
| API-01 event query | persisted `Event` rows | No REST endpoint found | No | ✗ DISCONNECTED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go regression suite | `go test ./...` | Passed across backend packages, including `internal/store` and `internal/server`. | ✓ PASS |
| Plan 01 artifact contract | `node ... gsd-tools.cjs verify artifacts ...01-foundation-01-PLAN.md` | 6/6 artifacts passed. | ✓ PASS |
| Plan 02 artifact contract | `node ... gsd-tools.cjs verify artifacts ...01-foundation-02-PLAN.md` | 3/3 artifacts passed. | ✓ PASS |
| Static key-link helper | `node ... gsd-tools.cjs verify key-links ...` | Helper reported 0/2 for both plans because it only checks direct source references; manual link verification above confirms real wiring for runtime links. | ⚠️ MANUAL REVIEWED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| PERS-01 | 01 | `tasks` table contains required fields | ✓ SATISFIED | `ent/schema/task.go` defines id, dispatch_ref, state, retry_count, loop_iteration_count, transport, wave, topo_rank, workspace_path, artifact_path, last_error_reason, created_at, updated_at, terminal_at, and card_json. |
| PERS-02 | 01 | `events` table contains required fields | ✓ SATISFIED | `ent/schema/event.go` defines event_id, task_id, event_type, from_state, to_state, timestamp, reason, attempt, transport, runner_id, and details. |
| PERS-03 | 01 | `waves` table contains required fields and uniqueness | ✓ SATISFIED | `ent/schema/wave.go` defines dispatch_ref, wave, sealed_at, created_at and a project-scoped unique index over dispatch_ref/wave. |
| PERS-04 | 01 | `card_json` TEXT NOT NULL stores full Task Card JSON, business fields default from it | ✓ SATISFIED | `deriveTaskCard` requires valid non-empty JSON and drives create/update fields from it; `BuildTaskView` reconstructs API projections from `card_json`; tests cover source-of-truth behavior. |
| PERS-05 | 02 | Event write and task state update occur in same SQLite transaction | ✓ SATISFIED | `UpdateTaskState` uses `WithTx`; task update and event create are in the same transaction closure. |
| PERS-06 | 01, 02 | SQLite uses WAL mode + busy_timeout | ✓ SATISFIED | `cmd/server/main.go` opens SQLite with `_pragma=journal_mode(WAL)` and `_pragma=busy_timeout(5000)`. |
| API-01 | 02, user-provided phase IDs | RESTful API exposes task CRUD, Wave operations, state query, event query | ✗ BLOCKED | Health/task create/read/update/list, retry/cancel state actions, dispatch task listing, and wave create/get/seal exist; no task delete endpoint/method and no persisted event query endpoint were found. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/store/repository.go` | n/a | Missing `DeleteTask` / equivalent task deletion method | Warning | Blocks full API-01 CRUD if delete is in scope. |
| `internal/server/server.go` | 190-202 | Task route group omits DELETE `/api/tasks/{id}` | Warning | Blocks full API-01 CRUD if delete is in scope. |
| `internal/server/server.go` and related route files | n/a | No REST route for querying persisted events | Blocker | Blocks API-01 event query requirement. |

No TODO/FIXME/placeholder blockers were identified in the core Phase 1 files. Empty-return/static patterns encountered in schema edge methods and defensive fallbacks are normal Go/ent implementations, not stubs.

### Human Verification Required

None for the backend persistence and route coverage verified here.

### Gaps Summary

The previous PERS-04 gap is closed: `card_json` is now validated and used to drive create/update persistence, and API task views are rebuilt from `card_json` with runtime state fields overlaid. The remaining blocker is API-01 coverage. The implemented server exposes health, task create/read/update/list, state actions (`cancel`, `retry`), dispatch task listing, and wave create/get/seal endpoints, but it does not expose task deletion or persisted event query. Because the user-provided phase requirement IDs explicitly include API-01 and REQUIREMENTS.md defines API-01 as task CRUD, Wave operations, state query, and event query, API-01 cannot be marked fully satisfied.

---

_Verified: 2026-05-15T15:32:37Z_
_Verifier: Claude (gsd-verifier)_
