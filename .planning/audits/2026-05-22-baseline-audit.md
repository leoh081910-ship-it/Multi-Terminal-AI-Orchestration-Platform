# Baseline Audit — 2026-05-22

## Working tree summary

Captured in isolated worktree:

```bash
git status --short
git log --oneline -10
git diff --stat
```

Current branch: `worktree-prd-completion-2026-05-22`

Recent commits:

```text
0e18a54 docs: rewrite README with architecture overview and quick start
b9266ee feat: multi-terminal AI orchestration platform — initial release
20226ec docs: sync roadmap + complete UAT (5/5 pass)
74dec38 fix(260407-w7l): enforce task card_json source of truth
f950386 test(260407-w7l): lock card_json repository source of truth
742c952 docs(01): create execution plans (2 plans)
4b35fe2 docs(01-foundation): create phase 1 plan
60b941c docs(phase-1): add validation strategy
2a4dbc6 docs(01-foundation): research Phase 1 foundation - Go + SQLite stack
cd39e48 docs(state): record phase 1 context session
```

Tracked diff summary:

```text
.planning/REQUIREMENTS.md           |   38 +-
.planning/ROADMAP.md                |   79 +-
.planning/STATE.md                  |  197 +-
internal/server/compat_scheduler.go |  443 +-
internal/server/server.go           |  246 +-
internal/server/server_test.go      |  611 ++-
internal/server/static_web.go       |   18 +-
web/src/api/schedulerApi.ts         |  109 +-
web/src/pages/OrchestratorHome.tsx  |   89 +-
web/src/pages/SchedulerBoard.tsx    |  127 +-
web/src/types/scheduler.ts          |   66 +
11 files changed, 1908 insertions(+), 899 deletions(-)
```

Git reported LF-to-CRLF warnings for several modified files when running `diff --stat`; do not normalize line endings as part of this audit.

### Modified tracked files

| File | Classification | Notes |
|------|----------------|-------|
| `.planning/REQUIREMENTS.md` | planning | Existing modified planning file; inspect before editing. |
| `.planning/ROADMAP.md` | planning | Existing modified planning file; inspect before editing. |
| `.planning/STATE.md` | planning | Existing modified planning file; inspect before editing. |
| `internal/server/compat_scheduler.go` | product | Existing modified backend compatibility scheduler file; inspect before editing. |
| `internal/server/server.go` | product | Existing modified backend route/server file; inspect before editing. |
| `internal/server/server_test.go` | validation | Existing modified backend tests; inspect before editing. |
| `internal/server/static_web.go` | product | Existing modified static serving file; inspect before editing. |
| `web/src/api/schedulerApi.ts` | product | Existing modified frontend scheduler API contract file; inspect before editing. |
| `web/src/pages/OrchestratorHome.tsx` | product | Existing modified home/control page; inspect before editing. |
| `web/src/pages/SchedulerBoard.tsx` | product | Existing modified scheduler board page; inspect before editing. |
| `web/src/types/scheduler.ts` | product | Existing modified frontend scheduler type contract; inspect before editing. |

### Untracked files

| Path | Classification | Handling |
|------|----------------|----------|
| `.planning/audits/` | planning/audit | Contains PRD completion audit documents; inspect before editing. |
| `.planning/phases/03-execution-layer/` | planning | Copied from original workspace; do not overwrite unrelated files. |
| `.planning/phases/04-integration/` | planning | Copied from original workspace; do not overwrite unrelated files. |
| `.planning/phases/05-interface/` | planning | Copied from original workspace; do not overwrite unrelated files. |
| `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/` | planning/validation | Copied from original workspace; inspect before editing. |
| `docs/superpowers/` | planning/spec | Contains implementation plan/spec context copied from original workspace. |
| `internal/router/` | validation | Contains untracked router tests copied from original workspace. |
| `web/src/hooks/useWebSocket.ts` | product | Untracked in this branch but copied from original workspace; inspect before editing. |
| `web/src/pages/EventLogPage.tsx` | product | Untracked in this branch but copied from original workspace; inspect before editing. |
| `web/src/pages/TaskDetailPage.tsx` | product | Untracked in this branch but copied from original workspace; inspect before editing. |

## Validation baseline

| Check | Command | Result | Notes |
|-------|---------|--------|-------|
| Frontend build | `npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run build` | pass | PASS after fixing `web/package.json` hard-coded Vite paths and removing missing `LogViewer` import: Vite transformed 1857 modules, built `dist/index.html`, CSS and JS assets in 24.92s. |
| Frontend lint | `npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run lint` | pass | PASS: ESLint completed with no reported issues. |
| Go availability | `go version` | GO_NOT_FOUND_IN_BASH | FAIL: `/usr/bin/bash: line 1: go: command not found`; PowerShell `Get-Command go` returned no command. |
| Backend tests | `go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./... -count=1` | blocked | Blocked because Go is unavailable in bash and Windows shell diagnosis did not find `go`. |
| Service startup | `go -C "E:/vibe coding/Projects/多终端 AI 编排平台" run ./cmd/server --config config.yaml` | blocked | Blocked until Go is installed or added to PATH. |

## Risk scan summary

| Risk | Evidence | Action |
|------|----------|--------|
| Isolated worktree started from older committed base | Plan file and original uncommitted changes were copied from the original workspace after worktree creation. | Treat copied files as baseline; inspect diffs before editing each touched file. |
| Existing worktree has many uncommitted files | `git status --short` lists modified tracked files and multiple untracked planning/product files. | Do not delete or commit without explicit approval; stage exact files only if later approved. |
| Reverse generated C contains placeholder TODO | `internal/reverse/executor.go` currently writes TODO in generated main logic per plan evidence. | Address in Reverse Capability tasks. |
| Go is unavailable in bash PATH | `go mod download` returned `/usr/bin/bash: line 1: go: command not found`. | Record as environment blocker if `go version` still fails in Task 3. |
| LF/CRLF churn could pollute diffs | `git diff --stat` emitted LF-to-CRLF warnings. | Avoid broad formatting or line-ending normalization. |

## Phase split

- Release Hardening: validate frontend build, frontend lint, Go tests, service/API smoke, browser smoke, Windows path smoke, and verification records.
- Reverse Capability: close reverse task contract, artifact contract, placeholder-free `final.c`, match-rate gate, environment classification, fake-loop tests, and real IDA/Frida manual follow-up.
- Final Acceptance: summarize automated evidence, manual follow-ups, remaining risks, and delivery verdict.

## Release hardening smoke results

| Scenario | Result | Evidence | Follow-up |
|----------|--------|----------|-----------|
| Service startup | blocked | `go run ./cmd/server` requires Go runtime not available in current shell. | Install Go or add to PATH; then run smoke. |
| Health endpoint | blocked | Depends on service startup. | Start service then `curl http://localhost:8080/health`. |
| Task list | blocked | Depends on service startup. | Start service then `curl /api/v1/scheduler/tasks`. |
| Task create | blocked | Depends on service startup. | Start service then POST task. |
| Wave seal | blocked | Depends on service startup. | Start service then POST/PUT wave. |
| WebSocket/browser refresh | manual_followup | Cannot start service; frontend build passes but browser interaction requires running backend. | Start service, open frontend, perform actions, observe WebSocket refresh. |
| Windows path smoke | manual_followup | Repository path contains spaces and Chinese characters; artifact copy and git operations untested at runtime. | Manual checklist: create task with artifact path containing Chinese, verify git add/commit. |

## Reverse capability source changes

| Change | File | Status | Notes |
|--------|------|--------|-------|
| Artifact path helper | `internal/reverse/artifacts.go` | created | `reverseArtifactPaths()` and `validateFinalC()` implemented. |
| Artifact contract tests | `internal/reverse/artifacts_test.go` | created | Tests for placeholder rejection, valid program acceptance, path layout. |
| Placeholder removal | `internal/reverse/executor.go` | modified | `generateCCode()` main block no longer emits TODO. |
| Final artifact gate | `internal/reverse/executor.go` | modified | `validateFinalC()` called before returning success at 100% match rate. |
| Artifact path consolidation | `internal/reverse/executor.go` | modified | Ad-hoc paths replaced with `reverseArtifactPaths()` helper. |
| IDA error classification | `internal/reverse/executor.go` | modified | IDA failure returns `EnvironmentUnavailableError{Reason: "ida_mcp_unavailable"}`. |
| Generate C test | `internal/reverse/executor_test.go` | modified | `TestGenerateCCodeDoesNotEmitPlaceholderTODO` added. |
| IDA classification test | `internal/reverse/executor_test.go` | modified | `TestExecutorClassifiesIDAUnavailable` added with mocks. |

All Go source changes are **unverified** — `go test ./internal/reverse` is blocked by Go runtime unavailability.
