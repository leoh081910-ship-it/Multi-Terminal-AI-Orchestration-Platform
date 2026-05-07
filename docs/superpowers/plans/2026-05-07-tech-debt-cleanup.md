# Tech Debt Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Systematically eliminate all remaining technical debt in the Go backend, organized by severity (HIGH → MEDIUM → LOW), with build verification after each round.

**Architecture:** Three-layer cleanup: Round 1 fixes runtime risks (nil panics, silent data loss, raw state strings), Round 2 fixes maintainability issues (magic numbers, code duplication, naming), Round 3 cleans hygiene (v3 comments, dead code). Each round ends with `go build ./...` and `go vet ./...`.

**Tech Stack:** Go 1.22+, ent ORM, SQLite, chi router, zerolog

---

## Round 1: HIGH Severity (Runtime Risks)

### Task 1: Fix nil pointer in buildRunnerTask

**Files:**
- Modify: `internal/server/runner_dispatch.go:32-83`

`executionManager.mainRepoPath` is accessed on line 75 without nil guard. Lines 41 and 47 check `executionManager != nil`, but line 75 does not.

- [ ] **Step 1: Write the failing test**

Create `internal/server/runner_dispatch_test.go`:

```go
package server

import (
	"testing"
)

func TestBuildRunnerTask_NilExecutionManager(t *testing.T) {
	s := &Server{}
	task := s.buildRunnerTask("test-task", map[string]interface{}{}, nil)
	if task.Workspace.MainRepo != "" {
		t.Errorf("expected empty MainRepo when executionManager is nil, got %q", task.Workspace.MainRepo)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go test ./internal/server/ -run TestBuildRunnerTask_NilExecutionManager -v`
Expected: PANIC on nil pointer dereference

- [ ] **Step 3: Fix the nil dereference**

In `runner_dispatch.go` line 75, wrap `executionManager.mainRepoPath`:

```go
// Before:
MainRepo:     executionManager.mainRepoPath,

// After:
MainRepo:     safeMainRepoPath(executionManager),
```

Add helper:
```go
func safeMainRepoPath(em *compatExecutionManager) string {
	if em == nil {
		return ""
	}
	return em.mainRepoPath
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go test ./internal/server/ -run TestBuildRunnerTask_NilExecutionManager -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/runner_dispatch.go internal/server/runner_dispatch_test.go
git commit -m "fix: guard nil executionManager in buildRunnerTask"
```

---

### Task 2: Fix nil pointer in RunCompatExecutionViaRunner fallback

**Files:**
- Modify: `internal/server/runner_dispatch.go:158-165`

`s.runnerRegistry.List()` is called on line 160 without checking if `s.runnerRegistry` is nil.

- [ ] **Step 1: Add nil guard**

```go
// Before (line 158-164):
if run == nil {
    runners := s.runnerRegistry.List()
    if len(runners) > 0 {
        run = runners[0]
    }
}

// After:
if run == nil && s.runnerRegistry != nil {
    runners := s.runnerRegistry.List()
    if len(runners) > 0 {
        run = runners[0]
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go build ./...`
Expected: exit 0

- [ ] **Step 3: Commit**

```bash
git add internal/server/runner_dispatch.go
git commit -m "fix: guard nil runnerRegistry in RunCompatExecutionViaRunner fallback"
```

---

### Task 3: Replace raw state strings in store/repository.go

**Files:**
- Modify: `internal/store/repository.go:440,454,858`
- Import: `github.com/mCP-DevOS/ai-orchestration-platform/internal/engine` (add if missing)

- [ ] **Step 1: Replace line 440**

```go
// Before:
if c.State != "done" {

// After:
if c.State != string(engine.StateDone) {
```

- [ ] **Step 2: Replace line 454**

```go
// Before:
SetState("verified").

// After:
SetState(string(engine.StateVerified)).
```

- [ ] **Step 3: Replace line 858**

```go
// Before:
state == "done" || state == "failed"

// After:
state == string(engine.StateDone) || state == string(engine.StateFailed)
```

- [ ] **Step 4: Verify build**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go build ./...`
Expected: exit 0

- [ ] **Step 5: Commit**

```bash
git add internal/store/repository.go
git commit -m "fix: replace raw state strings with engine.State* constants in repository"
```

---

### Task 4: Replace raw state strings in mergequeue/queue.go

**Files:**
- Modify: `internal/mergequeue/queue.go:154,179,187,197,208,213`
- Import: `github.com/mCP-DevOS/ai-orchestration-platform/internal/engine` (add if missing)

- [ ] **Step 1: Replace all 6 occurrences**

```go
// Line 154:
// Before: mq.repo.CheckTasksInState(ctx, deps, "done")
// After:  mq.repo.CheckTasksInState(ctx, deps, string(engine.StateDone))

// Line 179:
// Before: mq.repo.UpdateTaskState(ctx, task.ID, "verified", "apply_failed", ...)
// After:  mq.repo.UpdateTaskState(ctx, task.ID, string(engine.StateVerified), string(engine.StateApplyFailed), ...)

// Line 187: same pattern as 179

// Line 197: same pattern as 179

// Line 208:
// Before: mq.repo.UpdateTaskState(ctx, task.ID, "verified", "merged", "")
// After:  mq.repo.UpdateTaskState(ctx, task.ID, string(engine.StateVerified), string(engine.StateMerged), "")

// Line 213:
// Before: mq.repo.UpdateTaskState(ctx, task.ID, "merged", "done", "")
// After:  mq.repo.UpdateTaskState(ctx, task.ID, string(engine.StateMerged), string(engine.StateDone), "")
```

- [ ] **Step 2: Verify build**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go build ./...`
Expected: exit 0

- [ ] **Step 3: Commit**

```bash
git add internal/mergequeue/queue.go
git commit -m "fix: replace raw state strings with engine.State* constants in mergequeue"
```

---

### Task 5: Add error logging to silent persistCompatPayload calls

**Files:**
- Modify: `internal/server/server.go:930,1041,1068,1085,1095,1114,1170`

All 7 occurrences use `_ = s.persistCompatPayload(...)`. Replace with error logging.

- [ ] **Step 1: Create a helper method**

Add to `internal/server/server.go`:

```go
func (s *Server) persistCompatPayloadLogged(ctx context.Context, taskID string, payload map[string]interface{}) {
	if err := s.persistCompatPayload(ctx, taskID, payload); err != nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to persist compat payload")
	}
}
```

- [ ] **Step 2: Replace all 7 occurrences**

```go
// Before:
_ = s.persistCompatPayload(ctx, taskID, payload)

// After:
s.persistCompatPayloadLogged(ctx, taskID, payload)
```

- [ ] **Step 3: Verify build**

Run: `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go build ./...`
Expected: exit 0

- [ ] **Step 4: Commit**

```bash
git add internal/server/server.go
git commit -m "fix: add error logging to persistCompatPayload calls"
```

---

### Round 1 Verification

- [ ] Run full build: `go build ./...`
- [ ] Run vet: `go vet ./...`
- [ ] Run tests: `go test ./...`

---

## Round 2: MEDIUM Severity (Maintainability)

### Task 6: Extract magic numbers into constants

**Files:**
- Modify: `internal/server/compat_dispatch.go`
- Modify: `internal/server/runner_dispatch.go`
- Modify: `internal/server/execution_reaper.go`
- Modify: `internal/runner/capability.go`
- Modify: `internal/runner/cli_runner.go`

- [ ] **Step 1: Add constants to compat_dispatch.go (top of file)**

```go
const (
	defaultExecutionTimeout      = 30 * time.Minute
	executionHeartbeatInterval   = 30 * time.Second
	stalledHeartbeatThreshold    = 5 * time.Minute
	sessionIDPrefix              = "SE-"
	sessionIDLength              = 10
	taskIDPrefix                 = "TS-"
	taskIDLength                 = 8
)
```

- [ ] **Step 2: Replace hardcoded values in compat_dispatch.go**

- Line 131: `30 * time.Minute` → `defaultExecutionTimeout`
- Line 119: `strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:10]` → use `sessionIDPrefix` and `sessionIDLength`
- Line 366: `30 * time.Second` → `executionHeartbeatInterval`

- [ ] **Step 3: Replace hardcoded values in runner_dispatch.go**

- Line 81: `30 * time.Minute` → `defaultExecutionTimeout`

- [ ] **Step 4: Replace hardcoded values in execution_reaper.go**

- Line 162: `5 * time.Minute` → `stalledHeartbeatThreshold`

- [ ] **Step 5: Add context window constant to runner package**

Add to `internal/runner/interface.go` or a new `internal/runner/constants.go`:

```go
const DefaultClaudeContextWindow = 200000
```

Replace all 5 occurrences of `200000` in `capability.go` and `cli_runner.go`.

- [ ] **Step 6: Add shared task type list constant**

```go
var DefaultTaskTypes = []string{"feature", "bugfix", "refactor", "documentation", "code-review", "analysis"}
```

Replace in `capability.go:85`, `http_runner.go:85`, `mcp_runner.go:95`.

- [ ] **Step 7: Verify build and commit**

```bash
go build ./... && git add -A && git commit -m "refactor: extract magic numbers into named constants"
```

---

### Task 7: Replace raw state strings in mergequeue_adapter.go

**Files:**
- Modify: `internal/server/mergequeue_adapter.go:40`

- [ ] **Step 1: Replace**

```go
// Before:
item.State != "verified"

// After:
item.State != string(engine.StateVerified)
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/mergequeue_adapter.go && git commit -m "fix: use engine.StateVerified constant in mergequeue_adapter"
```

---

### Task 8: Fix DefaultHTTPMManifest naming typo

**Files:**
- Modify: `internal/runner/capability.go:80`
- Modify: All callers of `DefaultHTTPMManifest`

- [ ] **Step 1: Rename function**

```go
// Before:
func DefaultHTTPMManifest() *CapabilityManifest {

// After:
func DefaultHTTPManifest() *CapabilityManifest {
```

- [ ] **Step 2: Update all callers**

Grep for `DefaultHTTPMManifest` and replace with `DefaultHTTPManifest`.

- [ ] **Step 3: Verify build and commit**

```bash
go build ./... && git add -A && git commit -m "fix: rename DefaultHTTPMManifest to DefaultHTTPManifest (typo)"
```

---

### Task 9: Add error handling to buildRunnerTask DB call

**Files:**
- Modify: `internal/server/runner_dispatch.go:34`

- [ ] **Step 1: Add error logging**

```go
// Before:
task, _ := s.repo.GetTaskByID(context.Background(), taskID)

// After:
task, err := s.repo.GetTaskByID(context.Background(), taskID)
if err != nil {
    s.logger.Warn().Err(err).Str("task_id", taskID).Msg("failed to fetch task for runner, using defaults")
}
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/runner_dispatch.go && git commit -m "fix: log error when buildRunnerTask fails to fetch task"
```

---

### Task 10: Fix silent persist error in mergequeue_adapter

**Files:**
- Modify: `internal/server/mergequeue_adapter.go:121`

- [ ] **Step 1: Add error logging**

```go
// Before:
_ = a.repo.CheckAndPromoteParent(ctx, taskID)

// After:
if err := a.repo.CheckAndPromoteParent(ctx, taskID); err != nil {
    a.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to promote parent task")
}
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/mergequeue_adapter.go && git commit -m "fix: log error on parent task promotion failure"
```

---

### Task 11: Add safe type assertion for execution_runtime

**Files:**
- Modify: `internal/server/compat_dispatch.go:156`

- [ ] **Step 1: Add safe assertion**

```go
// Before:
reason := payload["execution_runtime"].(string) + " runtime command is not configured"

// After:
runtime, _ := payload["execution_runtime"].(string)
if runtime == "" {
    runtime = "unknown"
}
reason := runtime + " runtime command is not configured"
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/compat_dispatch.go && git commit -m "fix: safe type assertion for execution_runtime"
```

---

### Task 12: Resolve dual defaultProjectID constant

**Files:**
- Modify: `cmd/server/main.go:43`
- Modify: `internal/server/project_registry.go:13`

- [ ] **Step 1: Keep the one in project_registry.go, export it**

```go
// In project_registry.go:
// Before:
const compatDefaultProjectID = "default"

// After:
const DefaultProjectID = "default"
```

- [ ] **Step 2: Use it in main.go**

```go
// Before:
const defaultProjectID = "default"

// After: remove this line, use server.DefaultProjectID
```

Update all references in main.go from `defaultProjectID` to `server.DefaultProjectID`.

- [ ] **Step 3: Verify build and commit**

```bash
go build ./... && git add -A && git commit -m "refactor: unify defaultProjectID constant in server package"
```

---

### Task 13: Fix dead logic in compat_scheduler.go

**Files:**
- Modify: `internal/server/compat_scheduler.go:1186-1187`

- [ ] **Step 1: Simplify**

```go
// Before:
case "assigned":
    if dispatchStatus == "pending" {
        return engine.StateRouted
    }
    return engine.StateRouted

// After:
case "assigned":
    return engine.StateRouted
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/compat_scheduler.go && git commit -m "fix: remove dead logic in compat_scheduler state mapping"
```

---

### Round 2 Verification

- [ ] Run full build: `go build ./...`
- [ ] Run vet: `go vet ./...`
- [ ] Run tests: `go test ./...`

---

## Round 3: LOW Severity (Hygiene)

### Task 14: Clean up all v3 phase comments

**Files (30+ occurrences):**
- `internal/server/server.go:72,734`
- `internal/server/runner_dispatch.go:2,6,31,89,123,124`
- `internal/server/compat_dispatch.go:286,295`
- `internal/server/org_api.go:262,296,314,331`
- `internal/registry/registry.go:1,230`
- `internal/registry/heartbeat.go:1`
- `internal/router/router.go:24,63`
- `internal/router/capability_v2.go:2`
- `internal/runner/interface.go:2`
- `internal/runner/cli_runner.go:18`
- `internal/runner/http_runner.go:2`
- `internal/runner/mcp_runner.go:2`
- `internal/org/service.go:285,295,309,398`
- `ent/schema/agent.go:32,35,60`
- `cmd/server/main.go:193,198,256`
- `cmd/migrate/main.go:1`

- [ ] **Step 1: Batch replace using sed/replace-all**

Pattern: Remove `// v3:` / `// v3 ` / `// v3 Phase N:` prefixes from comments, keeping the descriptive text.

Examples:
```go
// Before: // v3: multi-agent runner registry
// After:  // multi-agent runner registry

// Before: // v3 Phase 3: HTTP runner implementation
// After:  // HTTP runner implementation

// Before: // v3: prefer dedicated runner_type/runner_config columns
// After:  // prefer dedicated runner_type/runner_config columns
```

For package-level doc comments that reference "v3" as historical context, convert to permanent description:
```go
// Before: // Package runner provides the v3 Runner interface...
// After:  // Package runner provides the Runner interface...
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add -A && git commit -m "chore: clean up v3 phase marker comments"
```

---

### Task 15: Remove dead code in compat_dispatch.go

**Files:**
- Modify: `internal/server/compat_dispatch.go:82`

- [ ] **Step 1: Remove dead assignment**

```go
// Before:
runtimeSpec := compatRuntimeSpec{Name: inferRuntimeFromAgent(ownerAgent, "")}
if executionManager != nil {
    runtimeSpec = executionManager.runtimeForAgent(ownerAgent)
}

// After:
var runtimeSpec compatRuntimeSpec
if executionManager != nil {
    runtimeSpec = executionManager.runtimeForAgent(ownerAgent)
} else {
    runtimeSpec = compatRuntimeSpec{Name: inferRuntimeFromAgent(ownerAgent, "")}
}
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/server/compat_dispatch.go && git commit -m "fix: clarify dead code path in dispatchCompatTask"
```

---

### Task 16: Remove unnecessary test suppress

**Files:**
- Modify: `internal/runner/interface_test.go:12`

- [ ] **Step 1: Remove**

```go
// Before:
_ = t // suppress unused warning

// After: delete the line
```

- [ ] **Step 2: Verify build and commit**

```bash
go build ./... && git add internal/runner/interface_test.go && git commit -m "chore: remove unnecessary unused variable suppression"
```

---

### Round 3 Verification

- [ ] Run full build: `go build ./...`
- [ ] Run vet: `go vet ./...`
- [ ] Run tests: `go test ./...`
- [ ] Final review: `git log --oneline -20` to verify all commits

---

## Summary

| Round | Tasks | Issues Fixed | Severity |
|-------|-------|-------------|----------|
| 1 | 5 | 5 HIGH (nil panics, silent data loss, raw state strings) | HIGH |
| 2 | 8 | ~20 MEDIUM (magic numbers, duplication, naming, dead logic) | MEDIUM |
| 3 | 3 | ~25 LOW (v3 comments, dead code, test cleanup) | LOW |
| **Total** | **16** | **~50 issues** | |

Each task: 2-5 minutes, TDD where applicable, build verification after each task.
