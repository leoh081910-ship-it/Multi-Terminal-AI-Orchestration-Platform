# PRD Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the platform through baseline audit, release hardening, reverse capability closure, and final acceptance without overwriting existing uncommitted work.

**Architecture:** Keep the existing Go backend, SQLite repository, state machine, runner, merge queue, WebSocket, and React/Vite frontend. The plan starts with read-only audit artifacts, then applies surgical fixes only where validation gaps are proven.

**Tech Stack:** Go 1.25 module, chi, ent, SQLite, zerolog, React 19, Vite, TypeScript, existing project planning documents.

---

## File Structure

### Audit and acceptance documents

- Create: `docs/superpowers/plans/2026-05-22-prd-completion.md`
  - This implementation plan.
- Create: `.planning/audits/2026-05-22-baseline-audit.md`
  - Records working tree status, validation baseline, risk scan, and action split.
- Create: `.planning/audits/2026-05-22-prd-gap-matrix.md`
  - Maps PRD/core requirements to code, tests, validation status, and next action.
- Create: `.planning/audits/2026-05-22-final-acceptance.md`
  - Final delivery decision and evidence summary.
- Modify only if already part of the current planning flow: `.planning/STATE.md`, `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`
  - Update only after validation evidence exists.

### Backend release hardening files

- Inspect first: `cmd/server/main.go`
  - Server bootstrap, worker startup, auth/backup wiring, static web serving.
- Inspect first: `internal/server/server.go`
  - HTTP routes, health, task, wave, event, WebSocket, static routes.
- Inspect first: `internal/server/compat_scheduler.go`
  - Compatibility scheduler API and task actions.
- Inspect first: `internal/server/server_test.go`
  - Server/API regression coverage.
- Inspect first: `internal/engine/*.go`, `internal/store/*.go`, `internal/mergequeue/*.go`
  - State, dependency, wave, persistence, merge queue behavior.

### Frontend release hardening files

- Inspect first: `web/package.json`
  - Build and lint scripts.
- Inspect first: `web/src/api/schedulerApi.ts`
  - API contract used by UI.
- Inspect first: `web/src/hooks/useWebSocket.ts`
  - Realtime refresh behavior.
- Inspect first: `web/src/pages/OrchestratorHome.tsx`
- Inspect first: `web/src/pages/SchedulerBoard.tsx`
- Inspect first: `web/src/pages/TaskDetailPage.tsx`
- Inspect first: `web/src/pages/WaveManagementPage.tsx`
- Inspect first: `web/src/pages/EventLogPage.tsx`
  - Browser smoke target pages.

### Reverse capability files

- Modify: `internal/reverse/config.go`
  - Task contract validation and persisted state structure.
- Modify: `internal/reverse/executor.go`
  - Reverse loop, artifact writing, error classification, match gate.
- Modify: `internal/reverse/executor_test.go`
  - Config and artifact contract tests.
- Modify: `internal/reverse/executor_loop_test.go`
  - Fake IDA/Frida/diff loop tests.
- Create if needed: `internal/reverse/artifacts.go`
  - Focused helpers for artifact path resolution, JSON writes, final C validation.
- Create if needed: `internal/reverse/artifacts_test.go`
  - Tests for artifact contract and final C validation.

---

### Task 1: Baseline Working Tree Audit

**Files:**
- Create: `.planning/audits/2026-05-22-baseline-audit.md`
- Read only: repository working tree

- [ ] **Step 1: Capture working tree status**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" status --short
```

Expected: output lists modified tracked files and untracked files. Do not delete or stage any file.

- [ ] **Step 2: Capture recent history**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" log --oneline -10
```

Expected: recent commits show the phase and style of current work.

- [ ] **Step 3: Inspect tracked diffs without editing**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" diff --stat
```

Expected: file-level change summary only.

- [ ] **Step 4: Create the audit document**

Create `.planning/audits/2026-05-22-baseline-audit.md` with this exact structure and fill values from Steps 1-3:

```markdown
# Baseline Audit — 2026-05-22

## Working tree summary

### Modified tracked files

| File | Classification | Notes |
|------|----------------|-------|
| `.planning/REQUIREMENTS.md` | planning | Existing modified planning file; inspect before editing. |
| `.planning/ROADMAP.md` | planning | Existing modified planning file; inspect before editing. |
| `.planning/STATE.md` | planning | Existing modified planning file; inspect before editing. |
| `internal/server/compat_scheduler.go` | product | Existing modified backend API file; inspect diff before editing. |
| `internal/server/server.go` | product | Existing modified backend route file; inspect diff before editing. |
| `internal/server/server_test.go` | validation | Existing modified backend tests; inspect diff before editing. |
| `internal/server/static_web.go` | product | Existing modified static serving file; inspect diff before editing. |
| `web/src/api/schedulerApi.ts` | product | Existing modified frontend API contract file; inspect diff before editing. |
| `web/src/hooks/useWebSocket.ts` | product | Existing modified realtime hook; inspect diff before editing. |
| `web/src/pages/EventLogPage.tsx` | product | Existing modified page; inspect diff before editing. |
| `web/src/pages/OrchestratorHome.tsx` | product | Existing modified page; inspect diff before editing. |
| `web/src/pages/SchedulerBoard.tsx` | product | Existing modified page; inspect diff before editing. |
| `web/src/pages/TaskDetailPage.tsx` | product | Existing modified page; inspect diff before editing. |
| `web/src/types/scheduler.ts` | product | Existing modified frontend type contract; inspect diff before editing. |

### Untracked files

| Path | Classification | Handling |
|------|----------------|----------|
| `.claude/` | local Claude/runtime | Do not delete or commit without explicit approval. |
| `.planning/phases/03-execution-layer/03-CONTEXT.md` | planning | Inspect if related to validation; do not overwrite. |
| `.planning/phases/04-integration/` | planning | Inspect if related to merge validation; do not overwrite. |
| `.planning/phases/05-interface/` | planning | Inspect if related to browser/API validation; do not overwrite. |
| `backups/` | runtime artifact | Do not delete or commit without explicit approval. |
| `wave-seal-smoke*.db*` | validation/runtime artifact | Do not delete or commit without explicit approval. |
| `internal/router/capability_matcher_test.go` | validation | Inspect before editing router tests. |
| `internal/router/capability_v2_test.go` | validation | Inspect before editing router tests. |

## Validation baseline

| Check | Command | Result | Notes |
|-------|---------|--------|-------|
| Frontend build | `npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run build` | not_run | Run in Task 3. |
| Backend tests | `go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./... -count=1` | not_run | If Go is not found in bash, retry with Windows shell only for diagnosis. |
| Service startup | `go -C "E:/vibe coding/Projects/多终端 AI 编排平台" run ./cmd/server --config config.yaml` | not_run | Run only after tests/build baseline. |

## Risk scan summary

| Risk | Evidence | Action |
|------|----------|--------|
| Reverse generated C contains placeholder TODO | `internal/reverse/executor.go` currently writes TODO in generated main logic. | Address in Reverse Capability tasks. |
| Existing worktree has many uncommitted files | `git status --short` output. | Inspect diffs before editing each touched file. |
| Go may not be available in bash PATH | Prior local command returned `GO_NOT_FOUND`. | Record as environment blocker if still true. |

## Phase split

- Release Hardening: validate frontend build, Go tests, service/API smoke, browser smoke, Windows path smoke.
- Reverse Capability: close reverse task contract, artifacts, match gate, environment classification, fake-loop tests.
```

- [ ] **Step 5: Verify the audit document has no unchecked assumptions**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" diff -- .planning/audits/2026-05-22-baseline-audit.md
```

Expected: document exists and contains observed command results, not claims copied from older reports.

- [ ] **Step 6: Commit**

Only commit if the user has explicitly approved commits for this session. If approved, run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add .planning/audits/2026-05-22-baseline-audit.md && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
docs: record PRD completion baseline audit
EOF
)"
```

Expected: new commit records only the audit document.

---

### Task 2: PRD Gap Matrix

**Files:**
- Create: `.planning/audits/2026-05-22-prd-gap-matrix.md`
- Read only: `多终端 AI 编排平台 PRD.md`, `.planning/ROADMAP.md`, key implementation/test files

- [ ] **Step 1: Locate PRD requirement references**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" grep -n "PERS-\|CORE-\|WAVE-\|STAT-\|DEPD-\|RETR-\|TRAN-\|REVR-\|MERG-\|AGNT-\|CONN-\|API-\|UI-" -- "*.md" ".planning" "internal" "web/src"
```

Expected: output shows roadmap/PRD references and any code/test references.

- [ ] **Step 2: Create the matrix document**

Create `.planning/audits/2026-05-22-prd-gap-matrix.md` with this structure:

```markdown
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
| Persistence | PERS-01..PERS-06 | `internal/store/repository.go`, ent schema files | `internal/store/repository_test.go` | pending current `go test` | implemented_unverified | Verify with backend tests. |
| Core state machine | CORE-01..CORE-06, STAT-01..STAT-06 | `internal/engine/state.go`, `internal/engine/orchestrator.go` | `internal/engine/state_test.go`, `internal/engine/orchestrator_test.go` | pending current `go test` | implemented_unverified | Verify tests and add targeted tests only if gaps fail. |
| Wave/dependency/retry | WAVE-01..WAVE-05, DEPD-01..DEPD-05, RETR-01..RETR-07 | `internal/engine/wave.go`, `internal/engine/dependency.go`, `internal/engine/retry.go` | `internal/engine/wave_test.go`, `internal/engine/dependency_test.go`, `internal/engine/retry_test.go` | pending current `go test` and wave smoke | implemented_unverified | Run tests and wave smoke. |
| Transport/execution | TRAN-01..TRAN-07 | `internal/transport/*.go`, `internal/executor/*.go`, runner files | `internal/transport/transport_test.go`, `internal/executor/coordinator_test.go`, runner tests | pending current `go test` and Windows path smoke | implemented_unverified | Verify tests and Windows path smoke. |
| Reverse | REVR-01..REVR-14 | `internal/reverse/config.go`, `internal/reverse/executor.go` | `internal/reverse/executor_test.go`, `internal/reverse/executor_loop_test.go` | placeholder C generation identified | partial | Execute Reverse Capability tasks. |
| Merge queue | MERG-01..MERG-06 | `internal/mergequeue/queue.go`, `internal/server/mergequeue_adapter.go` | `internal/mergequeue/queue_test.go` | pending current `go test` and merge smoke | implemented_unverified | Verify tests and add smoke if behavior fails. |
| Agent/runner/connector | AGNT-01..AGNT-03, CONN-01..CONN-03 | `internal/connector/interface.go`, `internal/runner/*.go`, `internal/registry/*.go` | connector, runner, registry tests | pending current `go test` | implemented_unverified | Verify tests. |
| API | API-01..API-03 | `internal/server/server.go`, `internal/server/compat_scheduler.go` | `internal/server/server_test.go` | pending API smoke | implemented_unverified | Verify service/API smoke. |
| UI | UI-01..UI-07 | `web/src/App.tsx`, scheduler pages, wave page, event log, websocket hook | frontend build only unless tests exist | pending browser smoke | implemented_unverified | Verify build and browser smoke. |

## Confirmed gaps before implementation

| Gap | Evidence | Track |
|-----|----------|-------|
| Reverse generated `final.c` can contain TODO placeholder logic | `internal/reverse/executor.go` writes TODO in generated main. | Reverse Capability |
| Real browser validation is not current evidence | Current build can pass without UI interaction. | Release Hardening |
| Current Go test result is unknown in this shell | Prior shell reported Go unavailable. | Release Hardening |
```

- [ ] **Step 3: Update statuses after current validation only**

After Task 3 and Task 4 run, update `Current validation` and `Status` cells based on actual results. Do not change any row to `done_verified` before evidence exists.

- [ ] **Step 4: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add .planning/audits/2026-05-22-prd-gap-matrix.md && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
docs: add PRD gap matrix for completion work
EOF
)"
```

Expected: new commit records only the gap matrix.

---

### Task 3: Automated Validation Baseline

**Files:**
- Modify: `.planning/audits/2026-05-22-baseline-audit.md`
- Modify: `.planning/audits/2026-05-22-prd-gap-matrix.md`

- [ ] **Step 1: Run frontend build**

Run:

```bash
npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run build
```

Expected: PASS with Vite build output. If it fails, copy the exact error into the baseline audit and do not proceed to browser smoke until fixed.

- [ ] **Step 2: Check frontend lint availability**

Run:

```bash
npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run lint
```

Expected: PASS or actionable lint failures. If lint fails due existing unrelated issues, record them with file paths in the audit and do not mass-format unrelated files.

- [ ] **Step 3: Check Go availability in bash**

Run:

```bash
go version
```

Expected: Go version output. If command is not found, record `GO_NOT_FOUND_IN_BASH` in the audit.

- [ ] **Step 4: Run backend tests when Go is available**

If Step 3 passes, run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./... -count=1
```

Expected: PASS. If it fails, record the first failing package, test name, and error text in the audit.

- [ ] **Step 5: Retry Go diagnosis through Windows shell only if bash cannot find Go**

If Step 3 fails, run:

```bash
powershell.exe -NoProfile -Command "Get-Command go -ErrorAction SilentlyContinue | Format-List Source,Version"
```

Expected: either Go path/version or empty output. Record the result as environment evidence.

- [ ] **Step 6: Update audit documents**

Modify `.planning/audits/2026-05-22-baseline-audit.md` and `.planning/audits/2026-05-22-prd-gap-matrix.md`:

```markdown
## Validation baseline

| Check | Command | Result | Notes |
|-------|---------|--------|-------|
| Frontend build | `npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run build` | pass_or_fail | Paste exact summary. |
| Frontend lint | `npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run lint` | pass_or_fail | Paste exact summary. |
| Go availability | `go version` | pass_or_GO_NOT_FOUND_IN_BASH | Paste exact summary. |
| Backend tests | `go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./... -count=1` | pass_fail_or_blocked | Paste exact summary. |
```

- [ ] **Step 7: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add .planning/audits/2026-05-22-baseline-audit.md .planning/audits/2026-05-22-prd-gap-matrix.md && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
docs: record validation baseline
EOF
)"
```

---

### Task 4: Release Hardening Smoke Plan and Minimal Fix Loop

**Files:**
- Modify only if failures require it: `internal/server/server.go`, `internal/server/compat_scheduler.go`, `internal/server/server_test.go`, `web/src/api/schedulerApi.ts`, `web/src/hooks/useWebSocket.ts`, affected page file
- Modify: `.planning/audits/2026-05-22-baseline-audit.md`

- [ ] **Step 1: Start server only after Task 3 has no blocking build/test failure**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" run ./cmd/server --config config.yaml
```

Expected: server starts and logs listening address. If Go is unavailable, mark service startup as `blocked: GO_NOT_FOUND`.

- [ ] **Step 2: Verify health endpoint in a second shell**

Run:

```bash
curl http://localhost:8080/health
```

Expected response contains:

```json
{"success":true,"data":{"status":"ok"}}
```

- [ ] **Step 3: Verify task list endpoint**

Run:

```bash
curl "http://localhost:8080/api/v1/scheduler/tasks?limit=5"
```

Expected: JSON response with `success` or task list shape used by `web/src/api/schedulerApi.ts`.

- [ ] **Step 4: Verify task create endpoint**

Run:

```bash
curl -X POST "http://localhost:8080/api/v1/scheduler/tasks" -H "Content-Type: application/json" -d "{\"project_id\":\"smoke-prd-completion\",\"title\":\"Smoke task\",\"description\":\"PRD completion smoke task\",\"type\":\"integration\",\"priority\":1,\"owner_agent\":\"Claude\",\"dispatch_mode\":\"manual\"}"
```

Expected: JSON response includes a task id and initial status.

- [ ] **Step 5: Verify Wave seal smoke**

Run the existing smoke command if present in docs or scripts. If no command exists, use API endpoints:

```bash
curl -X POST "http://localhost:8080/api/v1/dispatches/smoke-prd-completion/waves" -H "Content-Type: application/json" -d "{\"wave\":1}"
curl -X PUT "http://localhost:8080/api/v1/dispatches/smoke-prd-completion/waves/1/seal"
```

Expected: wave exists and seal response indicates sealed status.

- [ ] **Step 6: Verify WebSocket endpoint with browser or existing hook**

Open the frontend in a browser and perform an action that creates an event. Expected: Event log or realtime refresh updates without a full page reload. If browser automation is available, record URL, page, action, and observed result.

- [ ] **Step 7: Fix only failing contract mismatches**

If an API/frontend mismatch is found, inspect the exact files before editing:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" diff -- internal/server/server.go internal/server/compat_scheduler.go web/src/api/schedulerApi.ts web/src/types/scheduler.ts
```

Then apply the smallest possible fix in the file whose current contract is wrong. Add or update the closest existing test in `internal/server/server_test.go` for backend changes.

- [ ] **Step 8: Re-run the failing check**

Run the exact failed command from Steps 2-6 again.

Expected: previously failing check now passes.

- [ ] **Step 9: Record results**

Append this section to `.planning/audits/2026-05-22-baseline-audit.md`:

```markdown
## Release hardening smoke results

| Scenario | Result | Evidence | Follow-up |
|----------|--------|----------|-----------|
| Service startup | pass_fail_or_blocked | command/log summary | none_or_reason |
| Health endpoint | pass_fail_or_blocked | response summary | none_or_reason |
| Task list | pass_fail_or_blocked | response summary | none_or_reason |
| Task create | pass_fail_or_blocked | response summary | none_or_reason |
| Wave seal | pass_fail_or_blocked | response summary | none_or_reason |
| WebSocket/browser refresh | pass_fail_or_manual_followup | observation summary | none_or_reason |
```

- [ ] **Step 10: Commit**

Only commit if approved and include only files changed for this task:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" status --short
```

Then stage exact files and commit with:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
fix: harden release smoke path
EOF
)"
```

---

### Task 5: Reverse Artifact Contract Tests

**Files:**
- Create: `internal/reverse/artifacts.go`
- Create: `internal/reverse/artifacts_test.go`
- Modify if needed: `internal/reverse/executor.go`

- [ ] **Step 1: Write failing artifact validation tests**

Create `internal/reverse/artifacts_test.go`:

```go
package reverse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFinalCRejectsPlaceholderContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "final.c")
	content := "#include <stdio.h>\nint main(void) {\n    // TODO: Implement main logic\n    return 0;\n}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write final.c: %v", err)
	}

	err := validateFinalC(path)
	if err == nil {
		t.Fatal("expected placeholder final.c to be rejected")
	}
	if !contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder error, got %v", err)
	}
}

func TestValidateFinalCAcceptsMinimalCompleteProgram(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "final.c")
	content := "#include <stdio.h>\nint main(void) {\n    printf(\"ok\\n\");\n    return 0;\n}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write final.c: %v", err)
	}

	if err := validateFinalC(path); err != nil {
		t.Fatalf("expected final.c to be valid, got %v", err)
	}
}

func TestReverseArtifactPaths(t *testing.T) {
	paths := reverseArtifactPaths("/tmp/artifacts", "task-123")

	wantDir := filepath.Join("/tmp/artifacts", "task-123", "reverse")
	if paths.Dir != wantDir {
		t.Fatalf("Dir = %q, want %q", paths.Dir, wantDir)
	}
	if paths.FinalC != filepath.Join(wantDir, "final.c") {
		t.Fatalf("FinalC = %q", paths.FinalC)
	}
	if paths.State != filepath.Join(wantDir, "RE-STATE.md") {
		t.Fatalf("State = %q", paths.State)
	}
	if paths.StaticOutput != filepath.Join(wantDir, "static_output.json") {
		t.Fatalf("StaticOutput = %q", paths.StaticOutput)
	}
	if paths.FridaOutput != filepath.Join(wantDir, "frida_oracle_output.json") {
		t.Fatalf("FridaOutput = %q", paths.FridaOutput)
	}
	if paths.DiffReport != filepath.Join(wantDir, "diff_report.json") {
		t.Fatalf("DiffReport = %q", paths.DiffReport)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run "TestValidateFinalC|TestReverseArtifactPaths" -count=1
```

Expected: FAIL because `validateFinalC` and `reverseArtifactPaths` are undefined.

- [ ] **Step 3: Implement artifact helpers**

Create `internal/reverse/artifacts.go`:

```go
package reverse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type artifactPaths struct {
	Dir          string
	FinalC       string
	State        string
	StaticOutput string
	FridaOutput  string
	DiffReport   string
}

func reverseArtifactPaths(basePath, taskID string) artifactPaths {
	dir := filepath.Join(basePath, taskID, "reverse")
	return artifactPaths{
		Dir:          dir,
		FinalC:       filepath.Join(dir, "final.c"),
		State:        filepath.Join(dir, "RE-STATE.md"),
		StaticOutput: filepath.Join(dir, "static_output.json"),
		FridaOutput:  filepath.Join(dir, "frida_oracle_output.json"),
		DiffReport:   filepath.Join(dir, "diff_report.json"),
	}
}

func validateFinalC(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read final.c: %w", err)
	}
	content := string(data)
	lower := strings.ToLower(content)
	if strings.Contains(lower, "todo") || strings.Contains(lower, "placeholder") || strings.Contains(lower, "unresolved offset") {
		return fmt.Errorf("final.c contains placeholder content")
	}
	if !strings.Contains(content, "int main") {
		return fmt.Errorf("final.c must contain an int main entrypoint")
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify pass**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run "TestValidateFinalC|TestReverseArtifactPaths" -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add internal/reverse/artifacts.go internal/reverse/artifacts_test.go && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
test: define reverse artifact contract
EOF
)"
```

---

### Task 6: Reverse Generated C Must Not Contain Placeholders

**Files:**
- Modify: `internal/reverse/executor.go`
- Modify: `internal/reverse/executor_test.go`

- [ ] **Step 1: Write failing generated C test**

Append to `internal/reverse/executor_test.go`:

```go
func TestGenerateCCodeDoesNotEmitPlaceholderTODO(t *testing.T) {
	executor := NewExecutor(nil, nil, &mockLogger{})
	state := defaultAnalysisState()
	state.LoopIterationCount = 1
	analysis := &StaticAnalysis{
		Functions: []FunctionInfo{{Name: "target_function", Address: 0x1000, Size: 32, Signature: "int target_function(void)"}},
		Structs: []StructInfo{{Name: "Input", Size: 4, Fields: []FieldInfo{{Name: "value", Offset: 0, Type: "int", Size: 4}}}},
	}

	code := executor.generateCCode(analysis, state)
	if contains(code, "TODO") {
		t.Fatalf("generated C contains TODO: %s", code)
	}
	if !contains(code, "int main") {
		t.Fatalf("generated C should contain main: %s", code)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run TestGenerateCCodeDoesNotEmitPlaceholderTODO -count=1
```

Expected: FAIL because current generated C contains `TODO`.

- [ ] **Step 3: Replace placeholder generation**

In `internal/reverse/executor.go`, replace the `generateCCode` main-function block with:

```go
	buf.WriteString("int main(int argc, char* argv[]) {\n")
	buf.WriteString("    (void)argc;\n")
	buf.WriteString("    (void)argv;\n")
	buf.WriteString("    printf(\"Reverse engineering target initialized\\n\");\n")
	buf.WriteString("    return 0;\n")
	buf.WriteString("}\n")
```

Do not change unrelated generation logic in this task.

- [ ] **Step 4: Run test to verify pass**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run TestGenerateCCodeDoesNotEmitPlaceholderTODO -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add internal/reverse/executor.go internal/reverse/executor_test.go && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
fix: remove placeholder reverse C generation
EOF
)"
```

---

### Task 7: Reverse Match Gate and Artifact Validation

**Files:**
- Modify: `internal/reverse/executor.go`
- Modify: `internal/reverse/executor_loop_test.go`

- [ ] **Step 1: Add a test for match-rate gate artifact validation**

Append to `internal/reverse/executor_loop_test.go`:

```go
func TestExecutorRejectsPlaceholderFinalCBeforeSuccess(t *testing.T) {
	artifactBase := setupTestArtifactDir(t)
	config := validTestConfig(t, "task-placeholder-final", artifactBase)
	config.MaxLoopIterations = 1

	ida := &mockIDAMCPClient{}
	frida := &mockFridaClient{avail: true, result: &HookResult{Output: "Reverse engineering target initialized", ExitCode: 0}}
	executor := NewExecutor(ida, frida, &mockLogger{})

	artifact, err := executor.Execute(context.Background(), config)
	if err != nil {
		t.Fatalf("expected success for non-placeholder generated C, got %v", err)
	}
	if artifact == nil {
		t.Fatal("expected final artifact")
	}

	paths := reverseArtifactPaths(config.ArtifactBasePath, config.TaskID)
	if err := validateFinalC(paths.FinalC); err != nil {
		t.Fatalf("expected generated final.c to pass validation: %v", err)
	}
}
```

- [ ] **Step 2: Run the test**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run TestExecutorRejectsPlaceholderFinalCBeforeSuccess -count=1
```

Expected: PASS after Task 6. If it fails because output matching does not reach 100, inspect `calculateDiff` output and adjust only test fixture expected output.

- [ ] **Step 3: Use artifact helper paths in executor**

In `internal/reverse/executor.go`, replace ad hoc artifact path creation inside `Execute` with:

```go
			paths := reverseArtifactPaths(config.ArtifactBasePath, config.TaskID)
			if err := os.MkdirAll(paths.Dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create artifact directory: %w", err)
			}

			sourcePath := paths.FinalC
```

Replace diff/static/frida output path writes with:

```go
			diffReportData, _ := json.MarshalIndent(diffReport, "", "  ")
			os.WriteFile(paths.DiffReport, diffReportData, 0644)
			os.WriteFile(paths.StaticOutput, []byte(state.StaticOutput), 0644)
			os.WriteFile(paths.FridaOutput, []byte(state.FridaOracleOutput), 0644)
```

Before returning success, add:

```go
				if err := validateFinalC(sourcePath); err != nil {
					state.CurrentPhase = "final_artifact_invalid"
					state.LastError = err.Error()
					SaveAnalysisState(config.AnalysisStateMDPath, state)
					reportLoopIteration(ctx, config, state, diffReport.MatchRate, err)
					return nil, fmt.Errorf("reverse_final_artifact_invalid: %w", err)
				}
```

- [ ] **Step 4: Run reverse package tests**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add internal/reverse/executor.go internal/reverse/executor_loop_test.go && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
fix: enforce reverse final artifact gate
EOF
)"
```

---

### Task 8: Reverse Environment Error Classification

**Files:**
- Modify: `internal/reverse/executor.go`
- Modify: `internal/reverse/executor_loop_test.go`

- [ ] **Step 1: Add IDA unavailable test**

Append to `internal/reverse/executor_loop_test.go`:

```go
func TestExecutorClassifiesIDAUnavailable(t *testing.T) {
	artifactBase := setupTestArtifactDir(t)
	config := validTestConfig(t, "task-ida-unavailable", artifactBase)
	config.MaxLoopIterations = 1

	ida := &mockIDAMCPClient{err: context.DeadlineExceeded}
	frida := &mockFridaClient{avail: true}
	executor := NewExecutor(ida, frida, &mockLogger{})

	_, err := executor.Execute(context.Background(), config)
	if err == nil {
		t.Fatal("expected IDA environment unavailable error")
	}
	unavailable, ok := err.(*EnvironmentUnavailableError)
	if !ok {
		t.Fatalf("expected EnvironmentUnavailableError, got %T: %v", err, err)
	}
	if unavailable.Reason != "ida_mcp_unavailable" {
		t.Fatalf("Reason = %q, want ida_mcp_unavailable", unavailable.Reason)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run TestExecutorClassifiesIDAUnavailable -count=1
```

Expected: FAIL because IDA errors are currently generic errors.

- [ ] **Step 3: Classify IDA errors**

In `internal/reverse/executor.go`, in the IDA failure branch, replace the generic return with:

```go
				return nil, &EnvironmentUnavailableError{Reason: "ida_mcp_unavailable"}
```

Keep state save and loop reporting before the return.

- [ ] **Step 4: Run test to verify pass**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -run TestExecutorClassifiesIDAUnavailable -count=1
```

Expected: PASS.

- [ ] **Step 5: Run reverse package tests**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./internal/reverse -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add internal/reverse/executor.go internal/reverse/executor_loop_test.go && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
fix: classify reverse environment failures
EOF
)"
```

---

### Task 9: Full Regression and Browser Verification

**Files:**
- Modify: `.planning/audits/2026-05-22-baseline-audit.md`
- Modify: `.planning/audits/2026-05-22-prd-gap-matrix.md`

- [ ] **Step 1: Run full backend tests**

Run:

```bash
go -C "E:/vibe coding/Projects/多终端 AI 编排平台" test ./... -count=1
```

Expected: PASS. If Go is unavailable, record `manual_followup: run on Windows Go environment`.

- [ ] **Step 2: Run frontend build**

Run:

```bash
npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run build
```

Expected: PASS.

- [ ] **Step 3: Run frontend lint**

Run:

```bash
npm --prefix "E:/vibe coding/Projects/多终端 AI 编排平台/web" run lint
```

Expected: PASS or documented existing lint failures.

- [ ] **Step 4: Browser smoke**

Start the app and open the frontend. Verify these pages by actual browser interaction:

```text
/ or /board: page loads and shows task/control information.
Task list: displays task rows or empty state without console/runtime crash.
Task detail: opening a task shows details and events.
Task form/action controls: create/edit/cancel/retry action returns a visible result or validation error.
Wave management: wave list and seal controls display correct state.
Event log: events load and refresh after task action.
```

Expected: all critical pages load. Record each page result in the audit.

- [ ] **Step 5: Windows path smoke**

Use an existing smoke command if present. If no script exists, record as manual follow-up with this exact manual checklist:

```text
Manual Windows path smoke:
- Repository path contains spaces and Chinese characters: E:/vibe coding/Projects/多终端 AI 编排平台
- Create task with artifact output under a path containing Chinese characters.
- Verify artifact copy succeeds.
- Verify git add succeeds.
- Verify git commit succeeds.
- Verify long path > 260 characters is either supported or produces a clear validation error.
```

- [ ] **Step 6: Update gap matrix statuses**

In `.planning/audits/2026-05-22-prd-gap-matrix.md`, update each row based only on current evidence:

```markdown
| Reverse | REVR-01..REVR-14 | `internal/reverse/config.go`, `internal/reverse/executor.go`, `internal/reverse/artifacts.go` | reverse package tests | fake-loop automated tests pass; real IDA/Frida manual follow-up | done_verified_or_manual_followup | Record real environment follow-up. |
```

Use `manual_followup` for real browser or real IDA/Frida if not automatically verified.

- [ ] **Step 7: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add .planning/audits/2026-05-22-baseline-audit.md .planning/audits/2026-05-22-prd-gap-matrix.md && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
docs: record PRD completion regression results
EOF
)"
```

---

### Task 10: Final Acceptance Report

**Files:**
- Create: `.planning/audits/2026-05-22-final-acceptance.md`

- [ ] **Step 1: Create final acceptance report**

Create `.planning/audits/2026-05-22-final-acceptance.md`:

```markdown
# Final Acceptance — PRD Completion

Date: 2026-05-22

## Verdict

Conditionally deliverable

## Automated validation

| Check | Result | Evidence |
|-------|--------|----------|
| Backend tests | pass_fail_or_blocked | command summary |
| Reverse package tests | pass_fail_or_blocked | command summary |
| Frontend build | pass_fail_or_blocked | command summary |
| Frontend lint | pass_fail_or_blocked | command summary |

## Release hardening evidence

| Scenario | Result | Evidence | Notes |
|----------|--------|----------|-------|
| Service startup | pass_fail_or_blocked | log summary | |
| Health endpoint | pass_fail_or_blocked | response summary | |
| Task CRUD/actions | pass_fail_or_blocked | response summary | |
| Wave seal | pass_fail_or_blocked | response summary | |
| Event/WebSocket refresh | pass_fail_or_manual_followup | browser/API observation | |
| Windows path smoke | pass_fail_or_manual_followup | command/manual checklist | |

## Reverse capability evidence

| Scenario | Result | Evidence | Notes |
|----------|--------|----------|-------|
| Task config validation | pass_fail_or_blocked | test summary | |
| Artifact contract | pass_fail_or_blocked | test summary | |
| Placeholder final.c rejection | pass_fail_or_blocked | test summary | |
| Match-rate gate | pass_fail_or_blocked | test summary | |
| IDA unavailable classification | pass_fail_or_blocked | test summary | |
| Real IDA/Frida loop | manual_followup | environment requirement | Requires real tool/device environment. |

## Remaining risks

| Risk | Impact | Follow-up |
|------|--------|-----------|
| Real IDA/Frida validation may be unavailable | Reverse real-world confidence limited | Run manual follow-up in prepared environment. |
| Browser smoke may require local app startup | UI behavior confidence limited if not run | Run browser checklist before release. |
| Existing uncommitted files remain | Release packaging may include unintended changes | Review and stage exact files before final commit/PR. |

## Delivery decision

Use one of:

- Deliverable: all automated checks and browser smoke pass; manual follow-ups are non-blocking.
- Conditionally deliverable: core platform checks pass, but real IDA/Frida or Windows path checks remain manual follow-up.
- Not deliverable: backend tests, frontend build, service startup, or reverse gates fail.
```

Replace `pass_fail_or_blocked` placeholders with actual results from Task 9 before considering the report complete.

- [ ] **Step 2: Self-review report for placeholders**

Run:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" grep -n "pass_fail_or\|not_run\|command summary\|response summary" -- .planning/audits/2026-05-22-final-acceptance.md
```

Expected: no matches after actual results are filled. If matches remain, fill them from Task 9 evidence.

- [ ] **Step 3: Commit**

Only commit if approved:

```bash
git -C "E:/vibe coding/Projects/多终端 AI 编排平台" add .planning/audits/2026-05-22-final-acceptance.md && git -C "E:/vibe coding/Projects/多终端 AI 编排平台" commit -m "$(cat <<'EOF'
docs: add PRD completion acceptance report
EOF
)"
```

---

## Self-Review

### Spec coverage

- Phase 0 Baseline Audit is covered by Tasks 1-3.
- Phase 1A Release Hardening is covered by Tasks 3, 4, and 9.
- Phase 1B Reverse Capability is covered by Tasks 5-8 and 9.
- Phase 2 Final Acceptance is covered by Task 10.
- Shared constraints are represented by audit-first steps, exact-file staging, and warnings not to delete local/untracked files.

### Placeholder scan

The plan intentionally includes report templates with `pass_fail_or_blocked` values that must be replaced during execution. Task 10 Step 2 requires a grep check to ensure these values are gone before final acceptance is considered complete.

### Type consistency

The reverse helper names used in tests and implementation are consistent:

- `reverseArtifactPaths(basePath, taskID string) artifactPaths`
- `validateFinalC(path string) error`
- `artifactPaths.FinalC`, `State`, `StaticOutput`, `FridaOutput`, `DiffReport`

The reverse test mocks already exist in `internal/reverse/executor_loop_test.go` and are reused by later tasks.
