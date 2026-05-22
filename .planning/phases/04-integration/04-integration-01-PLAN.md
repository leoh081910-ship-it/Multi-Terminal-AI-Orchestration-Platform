---
phase: 04-integration
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/connector/gsd.go
  - internal/connector/gsd_test.go
  - internal/mergequeue/queue.go
  - internal/mergequeue/queue_test.go
autonomous: true
requirements:
  - CONN-01
  - CONN-02
  - CONN-03
  - MERG-01
  - MERG-02
  - MERG-03
  - MERG-05
  - MERG-06
must_haves:
  truths:
    - "GSD connector still discovers tasks and hydrates execution context, but write-back now persists both artifacts and planning-document updates instead of leaving the lifecycle half-finished."
    - "Merge queue remains single-consumer and deterministic: verified tasks are selected only when dependencies are done, then sorted by topo rank and created time before git work."
    - "Merge execution still copies artifacts into the main checkout, runs git add and git commit, and advances task state only after a successful merge path."
  artifacts:
    - path: "internal/connector/gsd.go"
      provides: "GSD connector discovery, hydration, ack, and write-back lifecycle"
      contains: "func (c *GSDConnector) WriteBackArtifacts"
    - path: "internal/connector/gsd_test.go"
      provides: "connector contract and write-back regressions"
      contains: "WriteBackArtifacts"
    - path: "internal/mergequeue/queue.go"
      provides: "single-consumer merge queue with dependency gating and git merge flow"
      contains: "func (mq *MergeQueue) processNextTask"
    - path: "internal/mergequeue/queue_test.go"
      provides: "merge ordering and merge execution regressions"
      contains: "TestExecuteMergeCopiesArtifactsAndCommitsBeforeDone"
  key_links:
    - from: "internal/connector/gsd.go"
      to: "internal/mergequeue/queue.go"
      via: "write-back artifacts become merge inputs after verification"
      pattern: "WriteBackArtifacts|MergeQueue"
    - from: "internal/mergequeue/queue.go"
      to: "internal/server/mergequeue_adapter.go"
      via: "repository-backed task selection and dependency checks stay compatible with adapter inputs"
      pattern: "GetVerifiedTasksReadyForMerge|CheckTasksInState"
---

<objective>
Stabilize the core integration seam between the GSD connector and the merge queue.

Purpose: the platform already knows how to discover tasks, write artifacts, order verified work, and commit merges; this plan closes the remaining lifecycle gaps so the wave can move from isolated components to a reliable merge pipeline.
Output: a non-stub GSD write-back path, deterministic merge ordering/gating, and regression tests that prove the connector-to-merge handoff behaves correctly.
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
@internal/connector/interface.go
@internal/connector/gsd.go
@internal/connector/interface_test.go
@internal/mergequeue/queue.go
@internal/mergequeue/queue_test.go
@internal/server/mergequeue_adapter.go
@CLAUDE.md

<interfaces>
Use the existing `Connector` interface from `internal/connector/interface.go`:
- `DiscoverTasks(ctx context.Context) ([]TaskCard, error)`
- `HydrateContext(ctx context.Context, taskID string, baseContext map[string]interface{}) (map[string]interface{}, error)`
- `AckResult(ctx context.Context, result TaskResult) error`
- `WriteBackArtifacts(ctx context.Context, taskID string, artifacts []Artifact) error`
- `GetConnectorType() string`
- `GetConnectorVersion() string`
- `HealthCheck(ctx context.Context) error`

Use the existing merge queue repository contract from `internal/mergequeue/queue.go`:
- `GetVerifiedTasksReadyForMerge(ctx context.Context) ([]TaskInfo, error)`
- `GetTaskDependencies(ctx context.Context, taskID string) ([]string, error)`
- `CheckTasksInState(ctx context.Context, taskIDs []string, state string) (bool, error)`
- `UpdateTaskState(ctx context.Context, taskID, fromState, toState, reason string) error`

Do not introduce a new connector abstraction, a new queue abstraction, or a second merge pipeline.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Complete GSD connector write-back and planning-document persistence</name>
  <files>internal/connector/gsd.go, internal/connector/gsd_test.go</files>
  <read_first>
    <file>internal/connector/interface.go</file>
    <file>internal/connector/gsd.go</file>
    <file>internal/connector/interface_test.go</file>
    <file>internal/connector/gsd_test.go</file>
    <file>.planning/phases/04-integration/04-RESEARCH.md</file>
    <file>.planning/phases/04-integration/04-VALIDATION.md</file>
  </read_first>
  <behavior>
    - Test 1: `DiscoverTasks` still parses PLAN files and returns task cards with wave, dependency, file, and transport data intact.
    - Test 2: `HydrateContext` preserves the incoming base context and adds GSD-specific fields without mutating the source map.
    - Test 3: `AckResult` writes a JSON acknowledgment file under the workspace ack directory.
    - Test 4: `WriteBackArtifacts` writes all artifact files to the task artifact directory.
    - Test 5: `WriteBackArtifacts` updates the GSD planning documents instead of leaving `updatePlanningDocuments()` as a no-op.
  </behavior>
  <action>
    Finish the existing GSD connector lifecycle in `internal/connector/gsd.go` without adding a second connector implementation.

    Concrete rules:
    - Keep `DiscoverTasks`, `HydrateContext`, `AckResult`, and `WriteBackArtifacts` on the current `GSDConnector` type.
    - Replace the stubbed `updatePlanningDocuments()` with real file updates against the current GSD planning tree.
    - Preserve the existing artifact copy behavior into `workspaceDir/artifacts/{taskID}`.
    - Preserve the existing ack file behavior in `workspaceDir/acks/{taskID}.json`.
    - Keep filesystem handling Windows-safe and path-join based.
    - Keep all behavior deterministic for tests using temp directories and local fixture files.

    Add or update tests in `internal/connector/gsd_test.go` using only local fixtures and temp dirs. The tests should prove the connector lifecycle is intact and that write-back now mutates planning files rather than returning success without a side effect.
  </action>
  <verify>
    <automated>go test ./internal/connector -run 'Test(DiscoverTasks|HydrateContext|AckResult|WriteBackArtifacts)' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/connector/gsd.go` still contains `type GSDConnector struct`.
    - `internal/connector/gsd.go` still contains `func (c *GSDConnector) WriteBackArtifacts(`.
    - `internal/connector/gsd.go` no longer leaves `updatePlanningDocuments()` as a stubbed no-op.
    - `internal/connector/gsd_test.go` covers the connector lifecycle with temp dirs and local fixtures.
    - `go test ./internal/connector -run 'Test(DiscoverTasks|HydrateContext|AckResult|WriteBackArtifacts)' -count=1` exits 0.
  </acceptance_criteria>
  <done>GSD connector write-back now leaves a durable planning-tree side effect and the connector contract is protected by regression tests.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Lock merge queue ordering, dependency gating, and merge commit flow</name>
  <files>internal/mergequeue/queue.go, internal/mergequeue/queue_test.go</files>
  <read_first>
    <file>internal/mergequeue/queue.go</file>
    <file>internal/mergequeue/queue_test.go</file>
    <file>.planning/phases/04-integration/04-RESEARCH.md</file>
    <file>.planning/phases/04-integration/04-VALIDATION.md</file>
  </read_first>
  <behavior>
    - Test 1: verified tasks are selected in deterministic order by `topo_rank` ascending, then `created_at` ascending.
    - Test 2: tasks are skipped when dependencies are not in `done` state.
    - Test 3: `copyArtifactsToMainCheckout` copies nested artifacts into the main checkout.
    - Test 4: `executeMerge` advances a verified task to `merged` and then `done` after a successful git path.
    - Test 5: merge execution remains serial and deterministic under local repository fixtures.
  </behavior>
  <action>
    Harden the existing merge queue implementation in `internal/mergequeue/queue.go` rather than redesigning it.

    Concrete rules:
    - Keep the single-consumer processor loop.
    - Keep dependency gating based on repository state and task dependency lookup.
    - Keep merge ordering by `topo_rank` and then `created_at`.
    - Keep artifact copy, `git add`, and `git commit` as the merge execution path.
    - Preserve the `verified -> merged -> done` success progression.
    - Preserve the apply-failed path when artifact copy or git work fails.
    - Keep tests isolated to temporary git repositories and temp artifact trees.

    Update or extend `internal/mergequeue/queue_test.go` so the dependency check, ordering, artifact copy, and merge commit flow are all locked down by direct regression tests.
  </action>
  <verify>
    <automated>go test ./internal/mergequeue -run 'Test(MergeQueueSortOrder|MergeQueueDependencyCheck|CopyArtifactsToMainCheckoutCopiesNestedFiles|CopyArtifactsToMainCheckoutRejectsEmptyDirectory|ExecuteMergeTransitionsVerifiedToApplyFailedWhenCopyFails|ExecuteMergeCopiesArtifactsAndCommitsBeforeDone|ExecuteMergeTreatsNothingToCommitAsSuccess)' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/mergequeue/queue.go` still contains `func (mq *MergeQueue) processNextTask(`.
    - `internal/mergequeue/queue.go` still sorts by `topo_rank` and `created_at`.
    - `internal/mergequeue/queue.go` still runs `git add` and `git commit` in `executeMerge`.
    - `internal/mergequeue/queue_test.go` contains deterministic ordering and merge regression tests.
    - `go test ./internal/mergequeue -run 'Test(MergeQueueSortOrder|MergeQueueDependencyCheck|CopyArtifactsToMainCheckoutCopiesNestedFiles|CopyArtifactsToMainCheckoutRejectsEmptyDirectory|ExecuteMergeTransitionsVerifiedToApplyFailedWhenCopyFails|ExecuteMergeCopiesArtifactsAndCommitsBeforeDone|ExecuteMergeTreatsNothingToCommitAsSuccess)' -count=1` exits 0.
  </acceptance_criteria>
  <done>Merge queue ordering and git-backed merge transitions are locked by tests and remain compatible with dependency gating.</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/connector -run 'Test(DiscoverTasks|HydrateContext|AckResult|WriteBackArtifacts)' -count=1
go test ./internal/mergequeue -run 'Test(MergeQueueSortOrder|MergeQueueDependencyCheck|CopyArtifactsToMainCheckoutCopiesNestedFiles|CopyArtifactsToMainCheckoutRejectsEmptyDirectory|ExecuteMergeTransitionsVerifiedToApplyFailedWhenCopyFails|ExecuteMergeCopiesArtifactsAndCommitsBeforeDone|ExecuteMergeTreatsNothingToCommitAsSuccess)' -count=1
go test ./internal/connector ./internal/mergequeue -count=1
```
</verification>

<success_criteria>
The core integration seam is no longer partial: GSD connector write-back persists real planning-tree changes, merge selection is deterministic, merge execution still goes through the git path, and the wave has regression coverage for the connector-to-merge boundary.
</success_criteria>

<output>
After completion, create `.planning/phases/04-integration/04-integration-01-SUMMARY.md`
</output>
