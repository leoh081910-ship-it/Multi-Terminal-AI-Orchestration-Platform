# Phase 3 Research: Execution Layer

**Phase:** 3
**Goal:** Task execution via CLI/API transport with reverse engineering support
**Date:** 2026-05-07
**Status:** RESEARCH COMPLETE

---

## Executive Summary

Phase 3 Execution Layer is **partially implemented**. The foundation (transport, reverse executor, coordinator) exists as scaffolded code, but **critical integration gaps remain** between the transport/reverse packages and the actual dispatch flow. Most notably, the reverse execution path is defined architecturally but **not wired into the dispatch system**. The code is well-structured — it follows Go best practices, has proper error handling, and the reverse executor loop logic is sound — but it's disconnected from where it matters: the server dispatch layer.

---

## What Exists

### CLI Transport (`internal/transport/cli.go`)
- **Status:** ✅ Mostly complete
- Creates git worktree via `git worktree add`
- Extracts artifacts by glob patterns (`FilesToModify`)
- Writes artifacts to `artifacts/{task_id}/`
- Handles worktree cleanup via `defer removeWorktree`
- Basic Windows path validation (length only, 260 char limit)
- **Comment:** TRAN-01~03 references are in the Execute function docstring — implementation matches

### API Transport (`internal/transport/api.go`)
- **Status:** ⚠️ Incomplete (TRAN-04/05)
- Creates isolated workspace directory (not git worktree)
- Collects artifacts matching `FilesToModify`
- Writes artifacts to `ArtifactPath`
- **Gap:** TRAN-04 requires "syncs to API isolated directory" — current impl just writes to `config.ArtifactPath`. The "API isolated directory" sync destination is not clear in the code.

### Transport Core (`internal/transport/`)
| File | What's In It |
|------|-------------|
| `transport.go` | `Executor` interface, `TransportType` (CLI/API), `TaskConfig`, `Artifact`, `ExecutionResult`, `PathValidator` (length only), `ArtifactManager` |
| `command.go` | `BuildShellCommand` — supports cmd/powershell/pwsh/sh/bash |
| `env.go` | `mergeCommandEnv` — merges extra env with `os.Environ()` |
| `artifacts.go` | `writeArtifactsToPath` — creates dirs, writes files preserving relative paths |
| `process.go` | `executeCommand` — buffered stdout/stderr, log file support, execution banners |
| `transport_test.go` | Tests exist |

### Path Validation (`internal/transport/transport.go`)
- `PathValidator` checks length > 260 → error
- **TRAN-07 says:** "check root path length, spaces, Chinese characters; CLI extra checks symlink permissions and worktree path length"
- **Reality:** Only length check. No space checking (spaces are valid in Windows paths). No Chinese char handling (UTF-8 is fine). Symlink permissions not checked explicitly.
- **Note:** The spec language is vague — "check spaces" likely means "don't break on spaces" not "reject paths with spaces"

### Reverse Executor (`internal/reverse/executor.go`)
- **Status:** ⚠️ Implemented but disconnected
- Full verification loop: IDA → C generate → compile → run → Frida → diff → check match_rate
- `calculateDiff`: line-by-line comparison, normalizes whitespace, computes match_rate percentage
- `generateCCode`: generates stub C code with struct/function definitions
- Writes artifacts: `final.c`, `static_output.json`, `frida_oracle_output.json`, `diff_report.json` to `artifacts/{task_id}/reverse/`
- Loop state persistence via `RE-STATE.md` (`LoadAnalysisState`/`SaveAnalysisState`)
- Error types: `MaxLoopIterationsError`, `EnvironmentUnavailableError`
- **Gap:** `generateCCode` is a stub — it outputs TODO comments in the main function

### Reverse Config (`internal/reverse/config.go`)
- `ReverseTaskConfig` struct with all required fields (REVR-01)
- `ReverseTaskConfig.Validate()` checks all required fields (REVR-02)
- `AnalysisState` struct tracks `loop_iteration_count`, `current_phase`, `last_match_rate`, etc. (REVR-10/11)
- `LoadAnalysisState`/`SaveAnalysisState` for persistence (REVR-15/16)

### Reverse Tests (`internal/reverse/executor_test.go`)
- Config validation tests only — no integration tests for the full loop

### Executor Coordinator (`internal/executor/coordinator.go`)
- `Coordinator` holds `cliTransport`, `apiTransport`, `reverseExecutor`, `orchestrator`
- `ExecuteTask` dispatches to CLI or API transport
- `ExecuteReverseTask` calls reverse executor
- `ExecutionHandler.HandleReverseTaskExecution` bridges from map config to `ReverseTaskConfig`
- **Gap:** No actual invocation of `HandleReverseTaskExecution` in the dispatch flow

---

## Critical Gaps (Must Fix)

### GAP 1: Reverse Dispatch Not Wired to Server
**Severity:** CRITICAL — reverse tasks can't actually execute

`ReverseTaskConfig`, `IDAMCPClient`, and `FridaClient` appear **zero times** in `internal/server/`. The reverse execution path is architecturally defined but completely disconnected from the dispatch system.

**What's needed:**
1. `ReverseTaskConfig` must be instantiated from the task's `card_json` context when `task.type == "reverse_static_c_rebuild"`
2. `reverse.Executor` must be instantiated in server startup with real `IDAMCPClient` and `FridaClient` implementations
3. Dispatch path must check task type and route to `ExecuteReverseTask` instead of regular transport
4. On `match_rate == 100%`: transition task `running → patch_ready` and store `final.c` artifact
5. On `MaxLoopIterationsError`: transition `running → retry_waiting` with `reverse_loop_exhausted`
6. On `EnvironmentUnavailableError`: transition `running → retry_waiting` with `reverse_env_unavailable`
7. On any internal retry (compile_failed, frida_oracle_failed, etc.): log `loop_iteration` event, don't change outer state

### GAP 2: Reverse Task Validation at Enqueue
**Severity:** HIGH — REVR-02 requires rejection at enqueue

The engine/enqueue path must call `ReverseTaskConfig.Validate()` before transitioning to `routed` state. Currently no such validation exists. Tasks with missing `target_so_path`, `frida_hook_spec`, etc. could enter `routed`.

**What's needed:**
- In `internal/engine/` or wherever enqueue validation happens: check if `task.type == "reverse_static_c_rebuild"`, extract reverse context fields, call `Validate()`, reject with `reverse_missing_field` reason if invalid.

### GAP 3: IDAMCPClient & FridaClient Implementations
**Severity:** CRITICAL — the reverse executor has interfaces but no real implementations

`reverse.Executor` expects `IDAMCPClient` (interface: `GetStaticAnalysis`, `GetFunctionInfo`, `GetStructInfo`) and `FridaClient` (interface: `RunHook`, `IsDeviceAvailable`).

**What's needed:**
1. **IDAMCPClient** — connects to IDA Pro MCP server (HTTP/SSE?). The `ida_mcp_endpoint` from config suggests an HTTP endpoint. Need to implement HTTP client that calls IDA MCP tools.
2. **FridaClient** — runs `frida` CLI or uses frida Python/Go bindings. The `frida_hook_spec` describes what to hook; `IsDeviceAvailable` checks if device is connected.

These are protocol clients, not trivial to implement. See implementation plan for approach.

### GAP 4: Loop Iteration Event Logging
**Severity:** MEDIUM — REVR-05 requires events table logging

Each internal retry (compile_failed, frida_oracle_failed, etc.) should log a `loop_iteration` event with `from_state = "running"`, `to_state = "running"`. Currently `executor.go` has `Logger.Info/Error` calls but no events table writes.

**What's needed:**
- The reverse executor needs a reference to the events writer (or the `repo` interface) to write events during loop iterations.
- This requires passing a `func(logEvent)` callback or a `Repo` interface into the executor.

### GAP 5: REVR-14 Acceptance Validation
**Severity:** MEDIUM — `final.c` validation is defined but not called

The `FinalArtifact` acceptance criteria (REVR-14): "final.c is standalone compilable C code without unresolved offsets." The `generateCCode` function produces a stub with TODO comments. There's no validation pass that checks:
- `gcc -o /dev/null final.c` compiles without errors
- No `TODO` or placeholder comments in the output
- All referenced structs are defined

**What's needed:**
- A `ValidateFinalArtifact(final.c path)` function that runs compile check and pattern checks
- Called before transitioning `running → patch_ready`

### GAP 6: TRAN-07 Windows Symlink Permissions
**Severity:** LOW — `git worktree` handles this, but not explicitly

The `createWorktree` function doesn't explicitly check or handle symlink permissions beyond `git worktree add`. On Windows, `git worktree` works fine without special handling. However, if git is configured with `core.symlinks=false`, worktrees may not work correctly.

**What's needed:**
- Check `git config core.symlinks` and warn if `false` on Windows
- Or document that `core.symlinks=true` is required for the platform

### GAP 7: API Transport "Sync" Requirement
**Severity:** LOW — TRAN-04 wording is ambiguous

TRAN-04 says: "API Transport returns full file artifacts, writes to artifacts/{task_id}/ **and syncs to API isolated directory**."

Current implementation: writes to `config.ArtifactPath` only. "API isolated directory" is not defined in the codebase. This likely means: write artifacts to the workspace dir AND to the artifact dir (two copies). The current `APITransport` only writes to `ArtifactPath`.

**What's needed:**
- Clarify: is "API isolated directory" a separate path from `ArtifactPath`?
- If yes: add second write to that path
- If no: the current implementation is sufficient

---

## Requirements Coverage Analysis

| REQ | Status | Notes |
|-----|--------|-------|
| TRAN-01 | ✅ Done | cli.go: worktree creation, artifact extraction |
| TRAN-02 | ✅ Done | cli.go: empty_artifact_match error |
| TRAN-03 | ✅ Done | cli.go: warns only, no fail on extra files |
| TRAN-04 | ⚠️ Gap 7 | API transport exists, "sync" ambiguous |
| TRAN-05 | ✅ Done | api.go: workspace write failure → error |
| TRAN-06 | ✅ Done | Both transports normalize to patch_ready |
| TRAN-07 | ⚠️ Gap 6 | Length check only, symlink check incomplete |
| REVR-01 | ⚠️ Gap 3 | Config exists, real clients don't |
| REVR-02 | ⚠️ Gap 2 | Validation exists, not called at enqueue |
| REVR-03 | ⚠️ Gap 1 | Loop logic exists, not wired to dispatch |
| REVR-04 | ⚠️ Gap 4 | Internal errors handled, events not logged |
| REVR-05 | ⚠️ Gap 4 | Logger calls exist, events table not written |
| REVR-06 | ✅ Done | loop_iteration_count incremented |
| REVR-07 | ✅ Done | MaxLoopIterationsError implemented |
| REVR-08 | ✅ Done | State persistence, not reset on retry |
| REVR-09 | ✅ Done | match_rate >= 100.0 check exists |
| REVR-10 | ✅ Done | LoopIterationCount persisted |
| REVR-11 | ✅ Done | LoadAnalysisState at loop start |
| REVR-12 | ✅ Done | Artifacts written to reverse/ dir |
| REVR-13 | ✅ Done | DiffReport has match_rate, mismatch_cases, normalization_rules |
| REVR-14 | ⚠️ Gap 5 | generateCCode is stub, no validation |

**Summary:** 9/21 requirements are fully satisfied. 12 have gaps of varying severity.

---

## Implementation Approach

### Architecture
The reverse executor is well-designed as a standalone package. The key is integration, not redesign:

```
compat_dispatch.go or server.go
    ↓
    Check task.type == "reverse_static_c_rebuild"
    ↓
    Build ReverseTaskConfig from card_json context
    ↓
    ReverseTaskConfig.Validate() ← reject before routed if invalid
    ↓
    ExecuteReverseTask(ctx, config)
    ↓
    reverse.Executor.Execute()
        → IDA MCP call (IDAMCPClient impl)
        → C generation (stub → real implementation)
        → compile → run → Frida (FridaClient impl)
        → diff → match_rate check
        → on 100%: transition running→patch_ready
        → on error: transition appropriate state
        → each iteration: write loop_iteration event
```

### IDAMCPClient Implementation Approach
Based on `ida_mcp_endpoint` field, this is an HTTP endpoint. MCP typically uses JSON-RPC over HTTP SSE. Options:
1. **HTTP SSE client** — send JSON-RPC requests, receive SSE events
2. **Direct MCP library** — use `nhooyr/mcp` or similar (check if Go MCP lib exists)
3. **HTTP POST/GET** — if IDA MCP exposes REST-like endpoints

For planning purposes: implement as HTTP client that sends tool_call JSON-RPC requests to the endpoint. The MCP spec uses `application/json` for requests and SSE for responses.

### FridaClient Implementation Approach
Based on `frida_hook_spec` — this describes WHAT to hook. Options:
1. **frida CLI** — `frida -H <device> -l <script.js> -f <binary> -o output.json`
2. **frida-python bindings** — run `python -m frida` or use frida Python API
3. **frida-go bindings** — check for `github.com/frida/frida-go`

For planning purposes: implement as a Go wrapper around the `frida` CLI binary. Check if `frida` is in PATH, run the appropriate command, parse JSON output.

---

## Windows-Specific Notes

- Git worktree works on Windows 11 with default git config
- Long paths (>260) require registry key `HKLM\SYSTEM\CurrentControlSet\Control\FileSystem\LongPathsEnabled = 1` OR use `\\?\` prefix
- Symlink permissions: `git worktree add` requires either admin elevation or `core.symlinks = true`
- The codebase already uses `filepath` package consistently — cross-platform paths handled
- Chinese characters in paths work fine on NTFS with Go's `os` package (UTF-16 internally)

---

## Validation Architecture

For REVR requirements, validation is quantifiable:
- TRAN-01: `git worktree list` shows worktree exists during execution
- TRAN-02: unit test with empty `FilesToModify` glob
- TRAN-04: `ls artifacts/{task_id}/` shows files
- REVR-09: `jq '.match_rate' artifacts/{task_id}/reverse/diff_report.json >= 100`
- REVR-14: `gcc -o /dev/null artifacts/{task_id}/reverse/final.c` exit 0

---

## Dependencies

1. **Reverse executor** depends on `internal/engine` (for state transitions) and `internal/store` (for event logging)
2. **IDAMCPClient** depends on external IDA Pro installation with MCP server running
3. **FridaClient** depends on `frida` CLI or Python bindings
4. **State machine integration** requires coordination with `internal/engine/` state transition logic

---

*Research by direct codebase analysis — gsd-phase-researcher agent timed out on Cloudflare*
