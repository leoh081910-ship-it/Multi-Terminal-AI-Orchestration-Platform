---
phase: 04-integration
plan: "02"
type: execute
wave: 1
depends_on:
  - 04-integration-01
files_modified:
  - internal/server/mergequeue_adapter.go
  - internal/server/server_test.go
  - internal/store/repository_test.go
autonomous: true
requirements:
  - AGNT-01
  - AGNT-02
  - AGNT-03
  - MERG-06
must_haves:
  truths:
    - "The merge queue adapter remains the bridge between repository-backed task views and queue selection logic; it does not become a second queue implementation."
    - "Repository-driven dependency and state checks stay compatible with the merge queue's expectations for verified tasks, dependency lookup, and state promotion."
    - "The integration surface can be tested locally without external services by using repository fixtures and the existing server test harness."
  artifacts:
    - path: "internal/server/mergequeue_adapter.go"
      provides: "repository-to-mergequeue bridge for verified task selection, dependency lookup, and state sync"
      contains: "type MergeQueueRepositoryAdapter"
    - path: "internal/server/server_test.go"
      provides: "integration coverage for adapter-backed merge path and task lifecycle behavior"
      contains: "MergeQueueRepositoryAdapter"
    - path: "internal/store/repository_test.go"
      provides: "repository fixtures for adapter and merge integration"
      contains: "UpdateTaskState"
  key_links:
    - from: "internal/server/mergequeue_adapter.go"
      to: "internal/mergequeue/queue.go"
      via: "GetVerifiedTasksReadyForMerge / GetTaskDependencies / CheckTasksInState / UpdateTaskState"
      pattern: "MergeQueueRepositoryAdapter|GetVerifiedTasksReadyForMerge|CheckTasksInState"
    - from: "internal/server/server_test.go"
      to: "internal/server/mergequeue_adapter.go"
      via: "server-level integration can exercise the adapter without introducing a new transport layer"
      pattern: "MergeQueueRepositoryAdapter|NewMergeQueueRepositoryAdapter"
---

<objective>
Prove the server-side adapter keeps the merge queue aligned with repository-backed task state.

Purpose: the queue already knows how to order and merge tasks, but the server-facing bridge must reliably supply verified tasks, dependency state, and state updates so the integration path remains coherent.
Output: adapter-focused regression coverage showing that merge queue inputs and state transitions stay consistent with repository data.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
@E:/04-Claude/Runtime/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/04-integration/04-RESEARCH.md
@.planning/phases/04-integration/04-VALIDATION.md
@.planning/phases/04-integration/04-integration-01-PLAN.md
@internal/server/mergequeue_adapter.go
@internal/server/server.go
@internal/server/server_test.go
@internal/store/repository.go
@internal/store/repository_test.go
@internal/mergequeue/queue.go
@CLAUDE.md

<interfaces>
Use the existing adapter contract from `internal/server/mergequeue_adapter.go`:
- `GetVerifiedTasksReadyForMerge(ctx context.Context) ([]mergequeue.TaskInfo, error)`
- `GetTaskDependencies(ctx context.Context, taskID string) ([]string, error)`
- `CheckTasksInState(ctx context.Context, taskIDs []string, state string) (bool, error)`
- `UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string) error`

Use the existing repository contract from `internal/store/repository.go` rather than inventing a new bridge or scheduler.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Verify adapter-backed merge queue inputs against repository fixtures</name>
  <files>internal/server/mergequeue_adapter.go, internal/store/repository_test.go</files>
  <read_first>
    <file>internal/server/mergequeue_adapter.go</file>
    <file>internal/store/repository.go</file>
    <file>internal/store/repository_test.go</file>
    <file>internal/mergequeue/queue.go</file>
    <file>.planning/phases/04-integration/04-RESEARCH.md</file>
    <file>.planning/phases/04-integration/04-VALIDATION.md</file>
  </read_first>
  <behavior>
    - Test 1: verified tasks returned by the adapter only include tasks in `verified` state.
    - Test 2: dependency lookup from the adapter matches the repository-backed task card data.
    - Test 3: state checks through the adapter only return true when every referenced task is in the requested state.
    - Test 4: adapter state updates persist through the repository and remain visible to downstream queue selection.
  </behavior>
  <action>
    Keep the adapter thin and repository-backed; do not move merge logic into the server layer.

    Concrete rules:
    - Preserve `MergeQueueRepositoryAdapter` as a repository bridge.
    - Preserve the adapter method signatures expected by the merge queue.
    - Keep project filtering and compat payload synchronization behavior intact.
    - Use repository fixtures and task rows to prove adapter inputs are derived from the persisted data model.
    - Avoid external services; the entire proof should be local.

    Add or extend tests so the adapter's selection and state-check behavior are locked against regressions.
  </action>
  <verify>
    <automated>go test ./internal/server ./internal/store -run 'Test(MergeQueueRepositoryAdapter|UpdateTaskState|ListAllTasks|GetTaskByID)' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/server/mergequeue_adapter.go` still contains `type MergeQueueRepositoryAdapter struct`.
    - `internal/server/mergequeue_adapter.go` still exposes the four merge-queue bridge methods.
    - `internal/store/repository_test.go` covers the repository behavior needed by the adapter and queue.
    - `go test ./internal/server ./internal/store -run 'Test(MergeQueueRepositoryAdapter|UpdateTaskState|ListAllTasks|GetTaskByID)' -count=1` exits 0.
  </acceptance_criteria>
  <done>Adapter-backed merge queue inputs are proven against repository data and remain safe for queue consumption.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Add server-level integration coverage for the adapter-to-queue boundary</name>
  <files>internal/server/server_test.go, internal/server/mergequeue_adapter.go</files>
  <read_first>
    <file>internal/server/server.go</file>
    <file>internal/server/server_test.go</file>
    <file>internal/server/mergequeue_adapter.go</file>
    <file>internal/mergequeue/queue.go</file>
    <file>.planning/phases/04-integration/04-RESEARCH.md</file>
    <file>.planning/phases/04-integration/04-VALIDATION.md</file>
  </read_first>
  <behavior>
    - Test 1: the server-level integration harness can construct the adapter and query verified tasks ready for merge.
    - Test 2: dependency-gated tasks remain blocked until all dependencies are done.
    - Test 3: state promotion through the adapter remains visible to downstream merge queue selection.
    - Test 4: the integration test stays local, deterministic, and isolated to temp repositories or in-memory fixtures.
  </behavior>
  <action>
    Use the existing server test harness to validate the adapter-to-queue boundary without introducing a new transport or scheduler.

    Concrete rules:
    - Keep the integration test local and deterministic.
    - Drive the adapter with repository-backed tasks.
    - Confirm verified selection, dependency gating, and state propagation are all consistent.
    - Reuse existing server test patterns instead of creating a new test harness.
    - Do not add network calls or external services.

    Add a focused test path in `internal/server/server_test.go` that proves the merge queue can consume repository-backed data through the adapter as intended.
  </action>
  <verify>
    <automated>go test ./internal/server -run 'Test(MergeQueueRepositoryAdapter|MergeQueue.*Adapter|Adapter.*MergeQueue)' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/server/server_test.go` contains an adapter-to-queue integration regression.
    - The test proves dependency gating and state propagation stay intact.
    - The test does not rely on external services or live git remotes.
    - `go test ./internal/server -run 'Test(MergeQueueRepositoryAdapter|MergeQueue.*Adapter|Adapter.*MergeQueue)' -count=1` exits 0.
  </acceptance_criteria>
  <done>Server-side adapter integration is proven locally and continues to feed the merge queue with correct repository-backed data.</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/server ./internal/store -run 'Test(MergeQueueRepositoryAdapter|UpdateTaskState|ListAllTasks|GetTaskByID)' -count=1
go test ./internal/server -run 'Test(MergeQueueRepositoryAdapter|MergeQueue.*Adapter|Adapter.*MergeQueue)' -count=1
go test ./internal/server ./internal/store -count=1
```
</verification>

<success_criteria>
The server-to-queue bridge is covered by regression tests, and repository-backed merge inputs remain consistent with queue expectations.
</success_criteria>

<output>
After completion, create `.planning/phases/04-integration/04-integration-02-SUMMARY.md`
</output>
