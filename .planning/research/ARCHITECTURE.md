# Architecture Research

**Domain:** AI Task Orchestration Platform
**Researched:** 2026-04-06
**Confidence:** MEDIUM

## Standard Architecture

### System Overview

The multi-terminal AI orchestration platform is a local-first, single-user system that coordinates task execution across multiple AI agents with state persistence, dependency management, and specialized reverse engineering workflows.

```
┌─────────────────────────────────────────────────────────────────┐
│                     External Interfaces                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐              ┌──────────────────┐        │
│  │   CLI Adapter    │              │   API Adapter    │        │
│  │ (git worktree)   │              │ (artifact files) │        │
│  └────────┬─────────┘              └────────┬─────────┘        │
│           │                                   │                 │
├───────────┴───────────────────────────────────┴─────────────────┤
│                     Connector Interface Layer                   │
├─────────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Connector (discoverTasks/hydrateContext/ackResult/    │    │
│  │            writeBackArtifacts)                         │    │
│  └─────────────────────────┬──────────────────────────────┘    │
│                            │                                     │
├────────────────────────────┴────────────────────────────────────┤
│                     Control Plane                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐ ┌──────────────┐ ┌────────────────────────┐  │
│  │ State Engine │ │ Wave Manager │ │ Dependency Resolver    │  │
│  │ (13 states)  │ │ (batch/group)│ │ (topo_rank/conflicts)  │  │
│  └──────┬───────┘ └──────┬───────┘ └───────────┬────────────┘  │
│         │                │                      │                │
│  ┌──────┴────────────────┴──────────────────────┴─────────┐    │
│  │              Task Queue & Dispatcher                    │    │
│  └────────────────────────┬────────────────────────────────┘    │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐    │
│  │              Merge Queue (serial consumer)              │    │
│  └────────────────────────┬────────────────────────────────┘    │
│                           │                                      │
├────────────────────────────┴────────────────────────────────────┤
│                     Agent Runner Layer                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Agent Runner (abstract execution boundary)            │    │
│  │  └─ Reverse Engineering Loop Engine (IDA+Frida)        │    │
│  └─────────────────────────┬──────────────────────────────┘    │
│                            │                                     │
├────────────────────────────┴────────────────────────────────────┤
│                     Transport & Workspace                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │
│  │ Git Worktree │ │ Artifact Dir │ │ Workspace Manager    │   │
│  │  (CLI mode)  │ │ (API mode)   │ │ (path validation)    │   │
│  └──────────────┘ └──────────────┘ └──────────────────────┘   │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                     Persistence Layer                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    SQLite Database                        │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐                   │  │
│  │  │ tasks   │  │ events  │  │ waves   │                   │  │
│  │  └─────────┘  └─────────┘  └─────────┘                   │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **Connector Interface** | Abstracts external task sources; provides discoverTasks, hydrateContext, ackResult, writeBackArtifacts | TypeScript interface with implementation for each source (GSD, manual, API) |
| **State Engine** | Enforces 13-state finite state machine with strict transitions | Event-driven FSM with transition validation |
| **Wave Manager** | Manages task batching, wave sealing, routing gates | Component tracking (dispatch_ref, wave) pairs |
| **Dependency Resolver** | Computes topo_rank, detects conflicts_with | Topological sort with conflict graph |
| **Task Queue** | Schedules tasks for execution, manages routing | Priority queue ordered by topo_rank + created_at |
| **Merge Queue** | Serial consumer for verified patches; dependency-checked | Single-threaded queue with pre-condition validation |
| **Agent Runner** | Abstract execution boundary for task agents | Interface with implementations per agent type |
| **Reverse Loop Engine** | IDA + Frida quantifiable verification loop | Specialized state machine within running state |
| **Transport Layer** | Normalizes CLI (git worktree) vs API (artifacts) | Adapter pattern with unified interface |
| **Workspace Manager** | Validates paths, manages git worktrees, handles Windows edge cases | Path validation, symlink handling |

## Recommended Project Structure

```
src/
├── core/                          # Platform kernel
│   ├── state/                     # State machine engine
│   │   ├── StateEngine.ts         # FSM implementation
│   │   ├── transitions.ts         # Transition rules
│   │   └── validators.ts          # State validation
│   ├── queue/                     # Task scheduling
│   │   ├── TaskQueue.ts           # Priority queue
│   │   ├── MergeQueue.ts          # Serial merge consumer
│   │   └── dispatcher.ts          # Routing logic
│   ├── wave/                      # Wave management
│   │   ├── WaveManager.ts         # Batch grouping
│   │   ├── WaveSealer.ts          # Seal enforcement
│   │   └── routing-gates.ts       # Routing gates
│   ├── dependency/                # Dependency resolution
│   │   ├── DependencyResolver.ts  # topo_rank computation
│   │   ├── ConflictDetector.ts    # conflicts_with detection
│   │   └── topo-sorter.ts         # Topological sort
│   └── recovery/                  # Retry & recovery
│       ├── RetryPolicy.ts         # Backoff, max_retries
│       ├── RecoveryManager.ts     # Process recovery
│       └── failure-propagation.ts # Dependency failure handling
│
├── persistence/                   # Data layer
│   ├── database.ts                # SQLite connection pool
│   ├── repositories/
│   │   ├── TaskRepository.ts      # tasks table CRUD
│   │   ├── EventRepository.ts     # events table CRUD
│   │   └── WaveRepository.ts      # waves table CRUD
│   ├── migrations/                # Schema migrations
│   └── transaction.ts             # Transaction utilities
│
├── transport/                     # Transport adapters
│   ├── Transport.ts               # Interface definition
│   ├── CliTransport.ts            # Git worktree implementation
│   ├── ApiTransport.ts            # Artifact file implementation
│   └── artifact-manager.ts        # Artifact storage
│
├── runner/                        # Agent execution
│   ├── AgentRunner.ts             # Abstract runner
│   ├── Runner.ts                  # Concrete implementations
│   └── workers/
│       └── reverse-engineering/   # Reverse loop engine
│           ├── LoopEngine.ts      # Quantifiable verification
│           ├── IdaIntegration.ts  # IDA MCP client
│           └── FridaOracle.ts     # Black-box validation
│
├── connector/                     # Connector interface
│   ├── Connector.ts               # Interface definition
│   ├── GsdConnector.ts            # GSD implementation
│   └── ManualConnector.ts         # Manual/API source
│
├── workspace/                     # Workspace management
│   ├── WorkspaceManager.ts        # Path validation
│   ├── GitWorktree.ts             # Worktree operations
│   ├── WindowsPathValidator.ts    # Windows edge cases
│   └── cleanup.ts                 # TTL-based cleanup
│
└── cli/                           # Command-line interface
    └── index.ts                   # CLI entry point
```

### Structure Rationale

- **core/:** Contains the essential orchestration logic. Separated by concern (state, queue, wave, dependency, recovery) with clear boundaries.
- **persistence/:** SQLite access is encapsulated in repositories. Migrations live separately for schema evolution.
- **transport/:** CLI vs API transport differ in execution model but share interface, enabling transparent switching.
- **runner/:** Agent execution is abstracted. Reverse engineering gets specialized treatment due to its unique loop pattern.
- **connector/:** New task sources can be added by implementing the Connector interface without touching core logic.
- **workspace/:** Windows-specific path handling is isolated due to length/space/Unicode requirements.
- **cli/:** Thin entry point that delegates to core logic.

## Architectural Patterns

### Pattern 1: Event-Sourced State Machine

**What:** Task state is stored as immutable events (events table) with current state derived from event stream.

**When to use:** When audit trails and recovery are critical. This platform requires full state history for recovery and debugging.

**Trade-offs:**
- Pros: Complete audit trail, easy replay, natural fit for recovery
- Cons: Additional storage, queries need reconstruction or materialized state

**Example:**
```typescript
// Event-driven transition
await db.transaction(async (tx) => {
  await tx.execute(
    'INSERT INTO events (event_id, task_id, event_type, from_state, to_state, ...) VALUES (?, ?, ?, ?, ?, ...)',
    [eventId, taskId, 'STATE_TRANSITION', fromState, toState, ...]
  );
  await tx.execute(
    'UPDATE tasks SET state = ?, updated_at = ? WHERE id = ?',
    [toState, now, taskId]
  );
});
```

### Pattern 2: Wave Sealed Batching

**What:** Tasks are grouped into waves; a wave must be sealed (no more tasks can be added) before any task in that wave can route to execution.

**When to use:** When you need deterministic batch boundaries for parallel execution without conflict races.

**Trade-offs:**
- Pros: Clear boundaries, enables conflict-free parallel execution
- Cons: Requires explicit seal call; late additions must wait for next wave

### Pattern 3: Serial Merge Consumer

**What:** Only verified patches enter the merge queue, which processes one task at a time with dependency pre-validation.

**When to use:** When merge conflicts are expensive and must be prevented at all costs.

**Trade-offs:**
- Pros: No merge conflicts, simple conflict resolution
- Cons: Slow throughput if many tasks have dependencies

### Pattern 4: Quantifiable Verification Loop

**What:** Used for reverse engineering tasks - static还原 must match Frida oracle output 100%.

**When to use:** When task correctness can only be verified through deterministic comparison with ground truth.

**Trade-offs:**
- Pros: Mathematically rigorous verification
- Cons: Only applicable to deterministic, observable outputs

### Pattern 5: Transport Abstraction

**What:** CLI (git worktree) and API (artifact files) transports share interface but differ in execution model.

**When to use:** When execution environment varies but processing logic is identical.

**Trade-offs:**
- Pros: Unified code path after task completion, easy to add new transports
- Cons: Must maintain interface compatibility across different execution models

## Data Flow

### Request Flow: Task Discovery to Completion

```
[Connector.discoverTasks()]
         │
         ▼
┌─────────────────────────────────────────────┐
│     Control Plane receives Task Cards       │
│  1. Generate dispatch_ref                   │
│  2. Upsert wave records                     │
│  3. Compute topo_rank                       │
│  4. Resolve conflicts_with                  │
│  5. Insert into tasks table                 │
│  6. Emit events (task_created)              │
└─────────────────────────────────────────────┘
         │
         ▼
[Connector.sealWave()] ──optional──► Seal wave (sealed_at = NOW)
         │
         ▼
┌─────────────────────────────────────────────┐
│     Task Queue picks up tasks               │
│  1. Query non-sealed tasks in ready state   │
│  2. Apply routing gates (wave sealed?)      │
│  3. Dispatch to Agent Runner                │
│  4. Update state → routed                   │
└─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│     Agent Runner executes task              │
│  1. Prepare workspace (CLI/API)             │
│  2. Execute agent logic                     │
│  3. Collect artifacts                       │
│  4. Update state → patch_ready              │
└─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│     Verification Pipeline                   │
│  1. Validate artifacts exist                │
│  2. Run test commands (if specified)        │
│  3. For reverse tasks: verify match_rate=100│
│  4. Update state → verified                 │
└─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│     Merge Queue (serial consumer)           │
│  1. Pull verified task                      │
│  2. Validate: all depends_on are done       │
│  3. For CLI: commit to main checkout        │
│  4. For API: copy artifacts to main         │
│  5. Update state → merged → done            │
└─────────────────────────────────────────────┘
```

### State Management

```
                    ┌─────────────────────────────────────┐
                    │         SQLite (Source of Truth)   │
                    │  tasks | events | waves            │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────┴──────────────────────┐
                    │    StateEngine (in-memory cache)    │
                    │  - Current task states              │
                    │  - Transition rules                 │
                    │  - Validators                       │
                    └──────────────┬──────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
┌────────┴────────┐      ┌────────┴────────┐      ┌────────┴────────┐
│   Wave Manager  │      │  Task Queue     │      │  Merge Queue    │
│ - Batch groups  │      │ - topo_rank     │      │ - Serial        │
│ - Seal status   │      │ - created_at    │      │ - Dependency    │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

### Key Data Flows

1. **Task Enqueuement Flow:** Connector discovers tasks → Control plane generates dispatch_ref, computes topo_rank, detects conflicts → Tasks written to SQLite with state=queued → Events emitted for audit trail.

2. **Routing Flow:** Task queue polls for queued tasks → Wave manager checks wave sealed status → Routing gates enforce seal before dispatch → Agent runner receives dispatch → State transitions to routed.

3. **Execution Flow:** Agent runner prepares workspace (CLI/API transport) → Executes task logic → Collects artifacts → Validates against acceptance criteria → Transitions to patch_ready.

4. **Merge Flow:** Verified task enters merge queue → Queue validates all depends_on are done → Serial commit to main checkout → State transitions merged → done.

5. **Recovery Flow:** Process restart → Load all tasks from SQLite → Reconstruct current state from events → Resume from last known state (not from beginning).

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single user, local | Simple monolithic process, single SQLite connection is sufficient |
| Future multi-user | Add connection pool, consider separating read/write paths |
| 1K+ concurrent tasks | Wave-based batching prevents scheduling explosion |
| Large artifact volume | Consider artifact cleanup TTL (default 72h), archival strategy |

### Scaling Priorities

1. **First bottleneck: Merge queue throughput**
   - Only serial processing, but dependencies are the real constraint
   -Mitigation: Wave-based parallel execution at runner level compensates

2. **Second bottleneck: Reverse engineering iteration speed**
   - Frida oracle requires device connectivity
   - Mitigation: Run non-reverse tasks in parallel with reverse tasks

### Scaling Assumptions

- v1 is single-user, local-first. Multi-user is out of scope.
- SQLite is adequate for local workload. No distributed database needed.
- Git worktree operations are I/O bound but acceptable for single-user.
- Reverse engineering tasks require specialized infrastructure (IDA, Frida device) that limits parallelism naturally.

## Anti-Patterns

### Anti-Pattern 1: Skipping Wave Sealing

**What people do:** Sending tasks to execution without calling sealWave, trusting that "ready" means "can route."

**Why it's wrong:** Unsealed waves can have late-arriving tasks, causing race conditions in conflict detection.

**Do this instead:** Require explicit sealWave() call; reject routing attempts for unsealed waves with reason "wave_not_sealed".

### Anti-Pattern 2: Storing Derived State

**What people do:** Storing topo_rank in a separate column and updating it incrementally.

**Why it's wrong:** Derived state can drift from source. Topology changes (new dependencies) require recalculation of all dependent ranks.

**Do this instead:** Store only source of truth (relations in card_json). Compute topo_rank on read or cache with invalidation.

### Anti-Pattern 3: Parallel Merge Without Conflict Checking

**What people do:** Running multiple merges in parallel for throughput.

**Why it's wrong:** Git merge conflicts require manual resolution for apply_failed. Parallel merges guarantee conflicts.

**Do this instead:** Keep merge queue serial. The bottleneck is usually dependency waiting, not merge speed.

### Anti-Pattern 4: Mixing Internal and External Event Sources

**What people do:** Allowing Connector to emit state transition events directly, bypassing StateEngine.

**Why it's wrong:** Violates FSM invariant; external actors may attempt invalid transitions.

**Do this instead:** All state changes go through StateEngine. Connectors call platform methods, platform emits events.

### Anti-Pattern 5: Treating Reverse Loop as Regular Retry

**What people do:** Counting loop iteration failures as retry_count, triggering retry_waiting after each failed iteration.

**Why it's wrong:** Reverse task has iterative refinement process (match_rate: 0% → ... → 100%). Internal failures are expected; only exhaustion or environment failure triggers outer retry.

**Do this instead:** Separate loop_iteration_count from retry_count. Only trigger retry_waiting for max_loop_iterations exceeded or env unavailable.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| GSD (Get Shit Done) | First-party Connector | Discovers tasks from GSD plan output |
| Git (worktree) | Direct CLI via child_process | CLI transport uses git worktree for isolation |
| IDA Pro | MCP protocol (ida-pro-mcp) | Reverse tasks call IDA for static analysis |
| Frida | Black-box oracle | Reverse tasks run on rooted Android device |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Connector ↔ Control Plane | Method calls (discoverTasks, sealWave, etc.) | Connectors don't directly access state; all through Control Plane |
| Control Plane ↔ State Engine | Event-driven state transitions | Each transition emits event; state engine validates |
| Task Queue ↔ Agent Runner | Dispatch protocol (task payload) | Runner receives Task Card; returns result |
| Merge Queue ↔ Git | Command execution | Merge queue commits to main checkout |
| Reverse Loop Engine ↔ IDA/Frida | MCP + RPC | Specialized protocol; not HTTP REST |

## Build Order Considerations

From PRD analysis, the following build order minimizes circular dependencies:

### Phase 1: Persistence Foundation
- SQLite schema (tasks, events, waves)
- Repository layer with transaction support
- Migration framework

**Rationale:** All other components depend on persistence.

### Phase 2: State Machine Core
- StateEngine with transition validation
- Event emission on state changes
- Recovery from events table

**Rationale:** Dependencies and wave manager need state validation.

### Phase 3: Wave & Dependency
- WaveManager with seal enforcement
- DependencyResolver (topo_rank)
- ConflictDetector

**Rationale:** Task queue needs wave and dependency info to dispatch correctly.

### Phase 4: Task Queue & Execution
- TaskQueue with priority ordering
- AgentRunner abstraction
- Transport layer (CLI/API)

**Rationale:** These form the core execution path.

### Phase 5: Merge & Verification
- MergeQueue (serial consumer)
- Verification pipeline (including reverse task verification)

**Rationale:** Depends on queue and execution being functional.

### Phase 6: Connectors
- Connector interface
- GSDConnector implementation
- CLI entry point

**Rationale:** Connectors depend on all internal machinery.

### Phase 7: Reverse Engineering Loop
- LoopEngine
- IdaIntegration
- FridaOracle
- analysis_state_md persistence

**Rationale:** Specialized workflow; builds on execution and verification.

## Sources

- PRD v24: 多终端 AI 编排平台 PRD.md
- State machine patterns: Finite State Machines in Task Orchestration
- Wave-based batching: Known pattern from CI/CD systems (Jenkins stages, GitLab pipelines)
- Event sourcing: Domain-Driven Design patterns (Martin Fowler)
- SQLite single-writer: Transactional integrity for concurrent access

---

*Architecture research for: Multi-terminal AI Orchestration Platform*
*Researched: 2026-04-06*