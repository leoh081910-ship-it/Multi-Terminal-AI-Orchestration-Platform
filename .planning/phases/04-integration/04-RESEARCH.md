# Phase 4: Integration - Research

**Researched:** 2026-05-16  
**Domain:** Connector orchestration, merge queue integration, server bridge wiring, and end-to-end task lifecycle
**Confidence:** HIGH (direct codebase analysis; phase-specific gaps observed in source)

---

## Executive Summary

Phase 4 is the integration layer that connects task discovery, context hydration, execution acknowledgements, artifact write-back, and merge queue processing. The codebase already contains most of the core abstractions: `Connector`, `GSDConnector`, `MergeQueue`, and the server adapter layer that bridges repository data into merge scheduling. The main work in this phase is not inventing new primitives; it is closing the remaining wiring gaps and turning existing scaffolding into a reliable end-to-end flow.

**Primary recommendation:** Treat this phase as a contract-hardening and integration phase. Keep the existing abstractions, add missing write-back persistence behavior, verify merge ordering and gating through tests, and complete the connector → queue → merge → write-back lifecycle with Windows-safe validation coverage.

---

## User / Project Constraints

### Locked decisions from project docs
- Go backend + React/Vite frontend.
- SQLite only for v1.
- Windows 11 is the primary platform.
- The platform must support 10–20 concurrent tasks.
- Claude Code CLI is the first Runner, but the architecture must stay extensible.

### Phase-specific constraints
- Connector contracts should remain generic and pluggable.
- Merge queue must stay serialized at merge time, even if routing/discovery is parallelized.
- Windows long paths and Chinese paths matter for artifact copy and git operations.
- End-to-end validation should be achievable with local Go tests where possible.

---

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CONN-01 | `Connector` interface supports discover / hydrate / ack / write-back lifecycle. | `internal/connector/interface.go` already defines the shape. |
| CONN-02 | GSD connector can discover tasks and hydrate execution context. | `internal/connector/gsd.go` already implements the main flow. |
| CONN-03 | GSD connector writes artifacts back and updates planning documents. | `updatePlanningDocuments()` is still a stub. |
| MERG-01 | Merge queue selects verified tasks ready for merge. | `internal/mergequeue/queue.go` already contains repo-driven selection logic. |
| MERG-02 | Merge queue respects dependency order and topo rank. | Sorting by topo rank + created time is already present. |
| MERG-03 | Merge queue applies git merge/commit flow. | `executeMerge()` already stages, commits, and updates state. |
| MERG-04 | Windows path behavior is manually validated. | Automated coverage is limited for real Windows path semantics. |
| MERG-05 | Merge queue updates task state through verified → merged → done. | Current queue logic already transitions states after commit. |
| MERG-06 | Merge queue integration is testable end-to-end. | Adapter and queue are present, but full lifecycle tests are still needed. |
| AGNT-01 | Server adapter bridges repository data into merge queue inputs. | `internal/server/mergequeue_adapter.go` already exists. |
| AGNT-02 | Server can feed merge queue dependency/state checks. | Adapter methods already expose dependency and state checks. |
| AGNT-03 | Server + queue integration can be validated without external services. | Existing repository-driven tests make this feasible. |

---

## What Exists

### Connector abstraction
`internal/connector/interface.go` already defines the contract expected by the phase:

```go
type Connector interface {
	DiscoverTasks(ctx context.Context) ([]TaskCard, error)
	HydrateContext(ctx context.Context, taskID string, baseContext map[string]interface{}) (map[string]interface{}, error)
	AckResult(ctx context.Context, result TaskResult) error
	WriteBackArtifacts(ctx context.Context, taskID string, artifacts []Artifact) error
	GetConnectorType() string
	GetConnectorVersion() string
	HealthCheck(ctx context.Context) error
}
```

This gives the phase a stable seam for multiple connector implementations.

### GSD connector
`internal/connector/gsd.go` already covers the main execution-facing methods:
- task discovery
- context hydration
- result acknowledgement
- artifact write-back into workspace paths

The main gap is the planning-document update step. `WriteBackArtifacts()` calls `updatePlanningDocuments()`, but that method is still a stub and returns `nil`.

### Merge queue
`internal/mergequeue/queue.go` already contains the core merge pipeline:
- repository-driven selection of verified tasks
- dependency-aware ordering
- git copy / add / commit flow
- task state transitions after merge

The merge queue is therefore structurally present; the main need is stronger integration coverage and Windows-safe verification of the artifact copy + git flow.

### Server adapter
`internal/server/mergequeue_adapter.go` already bridges repository data into the merge queue interfaces. This is important because it means the merge queue does not need to know ent/repository details directly.

---

## Gaps and Risks

### GAP 1: Planning document write-back is not implemented
**Severity:** HIGH

`updatePlanningDocuments()` in `internal/connector/gsd.go` is a stub. That means artifacts can be copied into the workspace, but the phase cannot yet persist updates back into the planning documents that track state, verification, and roadmap progress.

**Impact:** The connector lifecycle is incomplete; write-back is only partially effective.

### GAP 2: End-to-end lifecycle coverage is incomplete
**Severity:** HIGH

The code paths for discovery, merge selection, and merge execution exist, but the full connector → queue → merge → write-back lifecycle still needs a dedicated integration test to prove the pieces work together.

**Impact:** The phase may look complete at the unit level but still fail as a system.

### GAP 3: Windows path behavior needs manual proof
**Severity:** MEDIUM

The project explicitly targets Windows 11 and Chinese paths. Git/artifact operations can behave differently there than in Linux-only test environments.

**Impact:** Artifact copy and merge steps may pass in CI but fail on the intended local platform.

### GAP 4: Merge ordering is sensitive to repository state
**Severity:** MEDIUM

The queue sorts by topo rank and creation time. That is correct structurally, but it must be preserved under integration tests to avoid regressions in dependency gating.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.x | Backend implementation and tests | Existing project runtime. |
| `net/http` / `httptest` | stdlib | Server and integration tests | No extra runtime cost. |
| `os`, `filepath`, `io` | stdlib | Artifact copy and path handling | Required for Windows-safe file work. |
| `testing` | stdlib | Unit and integration tests | Existing test approach. |
| `zerolog` | project standard | Logging | Already used in services. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | existing project use | Assertions | Only if nearby tests already rely on it. |

### Avoid introducing
- new merge orchestration frameworks
- new path libraries
- new queue abstractions
- additional persistence layers

---

## Architecture Patterns

### Recommended flow

```text
Connector
  ↓
Hydrate context / discover tasks
  ↓
Execution / ack / artifact write-back
  ↓
Server adapter
  ↓
MergeQueue
  ↓
Git copy → add → commit
  ↓
Task state transitions
```

### Pattern 1: Keep connector contract generic
The `Connector` interface should stay transport-agnostic. GSD-specific behavior belongs in the implementation, not in the interface.

### Pattern 2: Keep merge ordering deterministic
Merge selection should remain deterministic under test:
- verified first
- dependency-safe second
- topo rank third
- creation time as a stable tiebreaker

### Pattern 3: Treat write-back as part of the lifecycle
Artifact copy is not enough. If the connector reports success, the planning/document state must also be updated or the lifecycle remains incomplete.

### Pattern 4: Integrate through server adapters
The server adapter should remain the seam between repository data and merge queue inputs. That keeps the merge queue isolated from ent and request-layer concerns.

---

## Validation Architecture

### Test framework
| Property | Value |
|----------|-------|
| Framework | Go testing |
| Quick run command | `go test ./internal/connector ./internal/mergequeue ./internal/server -count=1` |
| Full suite command | `go test ./... -count=1` |
| Phase gate | Full suite green before verify |

### Phase requirements → test map
| Req ID | Behavior | Test Type | Automated Command | File Exists |
|--------|----------|-----------|-------------------|-------------|
| CONN-01 | Connector interface contract preserved | unit | `go test ./internal/connector -count=1` | yes |
| CONN-02 | GSD discover/hydrate/ack flow works | unit/integration | `go test ./internal/connector -count=1` | partial |
| CONN-03 | Write-back updates planning docs | integration | `go test ./internal/connector -count=1` | missing coverage |
| MERG-01/02/05 | Merge selection and ordering are correct | integration | `go test ./internal/mergequeue -count=1` | yes |
| MERG-03 | Git merge/commit flow succeeds | integration | `go test ./internal/mergequeue -count=1` | yes |
| MERG-04 | Windows path artifact behavior | manual | local Windows validation | manual only |
| MERG-06 | End-to-end lifecycle completes | integration | `go test ./internal/connector ./internal/mergequeue ./internal/server -count=1` | needs new coverage |
| AGNT-01/02/03 | Adapter bridges repo to queue | integration | `go test ./internal/server ./internal/mergequeue -count=1` | partial |

### Wave 0 requirements
- [ ] `internal/connector/gsd_test.go` — connector contract and write-back behavior
- [ ] end-to-end integration coverage for connector -> queue -> merge -> write-back lifecycle
- [ ] integration test for server adapter + merge queue dependency gating
- [ ] Windows manual validation checklist for artifact copy and git merge flow

---

## Recommended Implementation Approach

### 1. Complete GSD write-back behavior
Implement `updatePlanningDocuments()` so it performs the actual document updates expected by the connector lifecycle. The precise update rules should follow the existing planning file structure and avoid broad refactors.

### 2. Add end-to-end lifecycle tests
Cover the full flow with repository-backed tests:
- connector discovers tasks
- context is hydrated
- artifacts are written back
- merge queue receives eligible tasks
- merge queue processes them in dependency order
- task state transitions are persisted

### 3. Keep merge queue deterministic
Lock ordering and state progression with tests so later integration changes do not break merge sequencing.

### 4. Validate Windows behavior explicitly
Use a manual verification step on Windows 11 for path-sensitive artifact copy and git merge behavior.

---

## Risks and Sequencing Advice

### Recommended sequence
1. Finish connector write-back behavior.
2. Add connector and merge queue integration tests.
3. Add server-adapter/queue wiring coverage.
4. Run full regression.
5. Validate Windows-specific merge behavior manually.

### Risk matrix
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Planning document updates are underspecified | Medium | High | Limit changes to current document format and update only the known planning artifacts. |
| Merge queue tests become flaky due to git state | Medium | High | Use isolated temp workspaces and deterministic fixtures. |
| Windows path issues differ from CI | High | Medium | Add manual validation on a native Windows 11 path with Chinese characters. |
| Adapter changes leak repository concerns into queue | Low | Medium | Keep all repository access inside adapter boundaries. |

---

## Sources

### Primary
- `internal/connector/interface.go`
- `internal/connector/gsd.go`
- `internal/mergequeue/queue.go`
- `internal/server/mergequeue_adapter.go`
- `.planning/STATE.md`
- `.planning/ROADMAP.md`
- `.planning/phases/04-integration/04-VALIDATION.md`
- `.planning/v3-implementation-plan.md`

### Secondary
- Project-wide GSD workflow docs and phase research results already present in the repository

---

## Metadata

**Confidence breakdown:**
- Connector abstraction: HIGH
- Merge queue structure: HIGH
- Write-back gap: HIGH
- End-to-end coverage: MEDIUM until tests are added
- Windows validation need: HIGH

**Research date:** 2026-05-16
**Valid until:** 2026-06-16
