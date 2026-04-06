# Project Research Summary

**Project:** 多终端 AI 编排平台
**Domain:** AI Task Orchestration / Multi-Agent Dispatch Platform
**Researched:** 2026-04-06
**Confidence:** MEDIUM-HIGH

## Executive Summary

This is an AI task orchestration platform enabling single-user local execution of complex multi-agent workflows. Users submit Task Cards via CLI (git worktree) or API (isolated directories), and the platform handles wave-based batching, dependency-aware scheduling, workspace isolation, artifact collection, and result merging back to the main repository. The platform targets Windows 11 with Go backend + React frontend, using SQLite for state persistence.

The research identifies this as a novel category combining task queuing with specialized reverse engineering support. The key differentiator is the `reverse_static_c_rebuild` task type requiring IDA Pro + Frida integration with a quantitative verification loop (100% match_rate as exit condition). Expert recommendations center on building the core orchestration engine first before adding visual interfaces, and prioritizing SQLite-backed state recovery over real-time dashboards.

Key risks include: reverse engineering task integration complexity (VERY HIGH), wave sealing semantics requiring careful state Machine implementation, and cross-platform path handling (Windows long paths, Chinese characters). Mitigate by prioritizing foundational components and deferring IDA/Frida integration until core workflow is validated.

## Key Findings

### Recommended Stack

The technology choices derive from PRD v24 requirements plus domain expertise in AI agent orchestration. The Go + React/Vite stack aligns with the project's need for concurrent task handling and rich UI management.

**Core technologies:**
- **Go** — Backend runtime: Strong typing, built-in concurrency (goroutines), mature CLI ecosystem, excellent SQLite drivers (modernc.org/sqlite). Chosen for task dispatch performance.
- **React + Vite** — Frontend: Full management UI requires complex forms/tables/real-time updates; React ecosystem is most mature. Vite for fast bundling.
- **SQLite** — Storage: Single-file, zero-configuration, ACID compliant. Single-user local deployment makes PostgreSQL/MySQL unnecessary. V1 fixed backend (no configurability).
- **Git Worktree** — CLI Transport isolation: Standard git mechanism for workspace isolation. Natural fit for CLI-first users who already use git.
- **WebSocket** — Real-time updates: Web UI requires live task state; WebSocket preferred over SSE for bidirectional flexibility.
- **IDA Pro MCP + Frida** — Reverse engineering integration: Specialized tooling for `reverse_static_c_rebuild` task type (quantitative verification loop).

### Expected Features

**Must have (table stakes):**
- Task Discovery & Registration — Core entry point for receiving tasks from Connectors/APIs/manual input
- SQLite-backed State Machine (13-state: 8 normal + 5 exceptional) — Users expect visibility into task lifecycle with audit trails
- Wave-Based Batching with Seal Semantics — Once sealed, no modifications allowed; provides strong consistency
- Dependency Management — `depends_on` relations, topological ranking, cross-wave validation
- Workspace Isolation — Git worktree (CLI) / isolated directory (API); prevents cross-task contamination
- Artifact Collection — Whitelist-based (`files_to_modify`) capture to `artifacts/{task_id}/`
- Result Merging — Git add + commit pipeline with single-consumer serial queue
- Error Handling & Recovery — Retry logic with configurable max_retries, SQLite-based resume after crashes
- Reverse Engineering Task Type (`reverse_static_c_rebuild`) — Specialized handling with quantitative verification loop (100% match_rate)

**Should have (competitive):**
- Wave-Based Batching with Sealed Units — Granular control; tasks grouped and sealed atomically
- Quantitative Verification Loop — Automated diff between static C rebuild and Frida oracle output; 100% match required
- Context-Aware Recovery — Maintains `loop_iteration_count`, preserves artifacts, reloads state from SQLite
- Multi-Transport Normalization — CLI and API converge at `patch_ready` state
- Analysis State Persistence — `.md` state file serves as single source of truth across iterations
- Conflict Detection — `conflicts_with` computed per wave; prevents resource contention

**Defer (v2+):**
- Real-Time Web Dashboard (adds infrastructure complexity; use CLI status for v1)
- Multi-User Collaboration
- Cloud Storage Backends
- Visual Dependency Graph Builder
- Marketplace for Pre-built Connectors

### Architecture Approach

The architecture follows a transport-normalized pipeline: CLI (git worktree) and API (isolated directory) inputs converge at `patch_ready` state, then proceed through unified routing, execution, and merging stages. SQLite is the single source of truth for task state, with events table for audit only.

**Major components:**
1. **Transport Layer** — Normalizes CLI (git worktree, `discoverTasks`) and API (REST endpoints, isolated dirs) to same internal representation
2. **Task Engine** — State machine transitions, dependency analysis, wave assignment, topological ranking
3. **Execution Layer** — Workspace isolation, artifact collection, Agent abstraction (Claude Code CLI as first implementation)
4. **Merge Queue** — Serial single-consumer queue ordered by `topo_rank + created_at`; waits for all dependencies to complete
5. **Recovery Manager** — SQLite-based state reload, preserves iteration counts and intermediate artifacts

### Critical Pitfalls

1. **Wave Seal Race Condition** — Adding task to sealed wave causes routing to fail. Prevention: Check `wave.sealed_at IS NOT NULL` before routing; return `reason = "wave_already_sealed"` if attempted.

2. **Dependency After Routing** — Task gains new `depends_on` after being routed to a wave. Prevention: Validate no new dependencies can be added post-routing; reject with `reason = "dependency_added_post_routing"`.

3. **Conflict Within Wave** — Two tasks in same wave have `conflicts_with` relation. Prevention: Validate `conflicts_with` empty set at wave seal time; compute conflicts per wave during assignment phase.

4. **Transient State Divergence** — In-memory state drifts from SQLite (dual-state problem). Prevention: NEVER cache task state in memory; always read/write from SQLite; events table is append-only audit, not source of truth.

5. **Reverse Task Loop Infinite** — Verification loop never reaches 100% match_rate due to flawed static reconstruction. Prevention: Enforce exit condition `match_rate == 100%`; verify state persistence (loop iteration count preserved across process restarts).

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation & Task Engine
**Rationale:** Core orchestration engine must work before any UI or specialized task types. This phase validates the fundamental workflow: task submission → wave assignment → routing → execution → merge.
**Delivers:** SQLite schema (tasks/events/waves tables), state machine implementation (13 states), basic task CRUD, dependency analysis with topological ranking.
**Addresses:** Task Discovery, State Management, Dependency Management, Basic Retry Logic.
**Avoids:** Transient state divergence pitfall by enforcing SQLite-as-source-of-truth from day one.

### Phase 2: Transport & Execution
**Rationale:** With core engine ready, connect the two transport paths (CLI git worktree + API isolated directory) and implement workspace isolation. This validates transport normalization.
**Delivers:** CLI Transport (discoverTasks → hydrateContext → writeBackArtifacts), API Transport (REST endpoints), workspace isolation implementation, Agent abstraction interface.
**Addresses:** Workspace Isolation, Artifact Collection, Result Merging, Connector interface.
**Avoids:** Wave sealing and conflict detection pitfalls by implementing proper validation at wave seal time.

### Phase 3: Reverse Engineering Support
**Rationale:** This is the key differentiator, but requires the foundation and execution layer to be stable first. IDA Pro + Frida integration is complex and should only be attempted after core workflow is validated.
**Delivers:** `reverse_static_c_rebuild` task type, IDA Pro MCP integration, Frida dynamic analysis hooks, quantitative diff engine, analysis state persistence (`.md` files).
**Addresses:** Reverse Engineering Task, Quantitative Verification Loop, Analysis State Persistence, Context-Aware Recovery.
**Avoids:** Infinite loop pitfall by enforcing 100% match_rate exit condition from design phase.

### Phase 4: Frontend & Real-Time
**Rationale:** With all backend capabilities in place, add the management UI. WebSocket pushes ensure the UI reflects live state. This is the lowest risk phase as backend is fully functional.
**Delivers:** React + Vite Web UI, WebSocket state push, task CRUD UI, wave management panel, event log viewer.
**Addresses:** CLI Status Interface, Rich Status Queries, Wave-level visibility.
**Avoids:** Dashboard complexity pitfall by keeping v1 dashboard simple; focus on functional task management over visual polish.

### Phase Ordering Rationale

- **Why Platform Core First:** Every other component depends on the task engine and SQLite state machine. Building this first validates the entire approach.
- **Why Transport Second:** Execution requires inputs; transport brings tasks into the system. Validates normalization principle.
- **Why Reverse Engineering Third:** HIGHEST complexity component requires stable foundation. Defer until core is proven.
- **Why Frontend Last:** UI can always be added later; backend capabilities are the hard part. Validates CLI-first approach.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Reverse Engineering):** IDA Pro MCP server capabilities, Frida script stability on Windows, quantitative diff format specification. Domain is niche with limited documentation.
- **Phase 4 (Frontend):** WebSocket reconnection strategies, large state set pagination, concurrent UI update conflict handling.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** SQLite schemas, state machines, and DAG topological sorting are well-documented patterns.
- **Phase 2 (Transport):** Git worktree commands and REST API design follow standard conventions.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | PRD v24 requirements specify Go/React/SQLite; domain expertise confirms these choices |
| Features | MEDIUM-HIGH | Derived from PRD; web research returned limited results; domain expertise fills gaps |
| Architecture | HIGH | PRD v24 provides detailed architecture; transport normalization is sound pattern |
| Pitfalls | MEDIUM | Identified through architecture analysis; need implementation to validate |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **IDA Pro MCP integration:** No official documentation on MCP server capabilities for IDA Pro. Need to validate during Phase 3 planning.
- **Frida on Windows:** Dynamic instrumentation behavior on Windows 11 may differ from Linux. Need testing during Phase 3.
- **Git worktree on Windows:** Symbolic link permissions and long path handling may cause issues. PRD notes Windows compatibility requirement but details TBD during Phase 2.

## Sources

### Primary (HIGH confidence)
- PRD v24: Multi-terminal AI Orchestration Platform — Complete specification of data model, state machine, transport flows, reverse task rules
- PROJECT.md (this project) — Requirements, constraints, key decisions

### Secondary (MEDIUM confidence)
- Domain expertise — AI agent orchestration patterns, task queuing systems, reverse engineering workflows
- Comparison with AutoGen, CrewAI, LangChain — General capability mapping to understand differentiation

### Tertiary (LOW confidence)
- STACK research results — General web search; limited domain-specific results for "AI task orchestration platform" as category

---

*Research completed: 2026-04-06*
*Ready for roadmap: yes*