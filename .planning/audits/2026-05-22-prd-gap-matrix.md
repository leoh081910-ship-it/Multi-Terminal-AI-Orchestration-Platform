# PRD Gap Matrix — 2026-05-22

Status values:

- `done_verified`: code, tests, and current validation evidence exist.
- `implemented_unverified`: code exists but tests or current validation evidence are missing.
- `partial`: behavior exists only in part.
- `missing`: no implementation found.
- `manual_followup`: real environment or human validation required.

## Matrix

| Area | Requirement IDs | Code evidence | Test evidence | Current validation | Status | Next action |
|------|-----------------|---------------|---------------|--------------------|--------|-------------|
| Persistence | PERS-01..PERS-06 | `internal/store/repository.go`, ent schema files | `internal/store/repository_test.go`, phase verification docs | pending current `go test` | implemented_unverified | Verify with backend tests. |
| Core state machine | CORE-01..CORE-06, STAT-01..STAT-06 | `internal/engine/state.go`, `internal/engine/orchestrator.go` | `internal/engine/state_test.go`, `internal/engine/orchestrator_test.go` | pending current `go test` | implemented_unverified | Verify tests and add targeted tests only if gaps fail. |
| Wave/dependency/retry | WAVE-01..WAVE-05, DEPD-01..DEPD-05, RETR-01..RETR-07 | `internal/engine/wave.go`, `internal/engine/dependency.go`, `internal/engine/retry.go` | `internal/engine/wave_test.go`, `internal/engine/dependency_test.go`, `internal/engine/retry_test.go` | pending current `go test` and wave smoke | implemented_unverified | Run tests and wave smoke. |
| Transport/execution | TRAN-01..TRAN-07 | `internal/transport/*.go`, `internal/executor/*.go`, runner files | `internal/transport/transport_test.go`, `internal/executor/coordinator_test.go`, runner tests | pending current `go test` and Windows path smoke | implemented_unverified | Verify tests and Windows path smoke. |
| Reverse | REVR-01..REVR-14 | `internal/reverse/config.go`, `internal/reverse/executor.go`, `internal/reverse/artifacts.go` | `internal/reverse/executor_test.go`, `internal/reverse/artifacts_test.go` | artifact contract, placeholder removal, IDA classification source changes done; Go tests blocked | implemented_unverified | Run `go test ./internal/reverse` when Go is available; real IDA/Frida manual follow-up. |
| Merge queue | MERG-01..MERG-06 | `internal/mergequeue/queue.go`, `internal/server/mergequeue_adapter.go` | `internal/mergequeue/queue_test.go` | pending current `go test` and merge smoke | implemented_unverified | Verify tests and add smoke if behavior fails. |
| Agent/runner/connector | AGNT-01..AGNT-03, CONN-01..CONN-03 | `internal/connector/interface.go`, `internal/connector/gsd.go`, `internal/runner/*.go`, `internal/registry/*.go` | connector, runner, registry tests | pending current `go test` | implemented_unverified | Verify tests. |
| API | API-01..API-03 | `internal/server/server.go`, `internal/server/compat_scheduler.go` | `internal/server/server_test.go` | pending API smoke | implemented_unverified | Verify service/API smoke. |
| UI | UI-01..UI-07 | `web/src/App.tsx`, scheduler pages, wave page, event log, websocket hook | frontend build and lint | frontend build and lint pass; browser smoke blocked by Go/service startup | implemented_unverified | Run browser smoke after service can start. |

## Confirmed gaps before implementation

| Gap | Evidence | Track |
|-----|----------|-------|
| Reverse generated `final.c` can contain TODO placeholder logic | `internal/reverse/executor.go` writes TODO in generated main per plan evidence. | Reverse Capability |
| Real browser validation is not current evidence | Current build can pass without UI interaction. | Release Hardening |
| Current Go test result is unknown in this shell | `go mod download` returned `go: command not found`; Task 3 must run `go version` and Windows-shell diagnosis. | Release Hardening |
| Worktree baseline differs from original repository checkout | Isolated worktree was created from committed `main`; original uncommitted planning/product files were copied into it after user approval. | Audit/Release Hardening |
