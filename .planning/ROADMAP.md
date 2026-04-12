# Roadmap - 多终端 AI 编排平台

## Project Overview
- **Goal**: Multi-AI task orchestration platform enabling full automation from task submit to merge
- **Core**: Go backend + React frontend with SQLite, 13-state machine, CLI/API transport
- **Key Features**: Wave-based scheduling, dependency topology, reverse engineering loop, GSD Connector

## Phases

- [x] **Phase 1: Foundation** - SQLite persistence and project setup ✓
- [x] **Phase 2: Core Engine** - State machine and orchestration logic ✓
- [x] **Phase 3: Execution Layer** - Transport and reverse engineering ✓
- [x] **Phase 4: Integration** - Merge queue and Agent Connector ✓
- [x] **Phase 5: Interface** - HTTP API and React Web UI ✓

---

## Phase Details

### Phase 1: Foundation
**Goal**: Database persistence and project infrastructure operational
**Depends on**: Nothing (first phase)
**Requirements**: PERS-01, PERS-02, PERS-03, PERS-04, PERS-05, PERS-06
**Success Criteria** (what must be TRUE):
  1. Go server starts without errors via `go run cmd/server/main.go`
  2. SQLite database creates `tasks`, `events`, `waves` tables with correct schema
  3. Process restart recovers server state from SQLite (task CRUD, wave state)
  4. Task Card upsert correctly persists `card_json` as business field source
  5. Event logging records all state transitions atomically with task updates (PERS-05)
  6. Wave CRUD operations work (create, query, seal) with `(dispatch_ref, wave)` uniqueness enforcement
  7. SQLite uses WAL mode + busy_timeout for concurrent access (PERS-06)
**Plans:** 2 plans
- [x] 01-foundation-01-PLAN.md — Go module + ent schemas + code generation ✓
- [x] 01-foundation-02-PLAN.md — Repository + server + health check ✓

### Phase 2: Core Engine
**Goal**: 13-state machine orchestration with dependency and wave management
**Depends on**: Phase 1
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, WAVE-01, WAVE-02, WAVE-03, WAVE-04, WAVE-05, STAT-01, STAT-02, STAT-03, STAT-04, STAT-05, STAT-06, DEPD-01, DEPD-02, DEPD-03, DEPD-04, DEPD-05, RETR-01, RETR-02, RETR-03, RETR-04, RETR-05, RETR-06, RETR-07
**Success Criteria** (what must be TRUE):
  1. Tasks can transition through all 8 main states (queued → routed → workspace_prepared → running → patch_ready → verified → merged → done)
  2. Tasks transition through 5 exception states correctly (retry_waiting, verify_failed, apply_failed, failed)
  3. Wave seal prevents tasks entering `routed` before wave is sealed
  4. `depends_on` correctly validates at enqueue (rejects cross-wave forward dependencies)
  5. `conflicts_with` calculated only within same wave, skipped if `depends_on` exists
  6. `topo_rank` correctly computed from dependency graph
  7. Dependency failure propagation marks dependent tasks as failed immediately
  8. TTL handler correctly expires tasks after configured duration from `terminal_at`
**Plans**: TBD

### Phase 3: Execution Layer
**Goal**: Task execution via CLI/API transport with reverse engineering support
**Depends on**: Phase 2
**Requirements**: TRAN-01, TRAN-02, TRAN-03, TRAN-04, TRAN-05, TRAN-06, TRAN-07, REVR-01, REVR-02, REVR-03, REVR-04, REVR-05, REVR-06, REVR-07, REVR-08, REVR-09, REVR-10, REVR-11, REVR-12, REVR-13, REVR-14
**Success Criteria** (what must be TRUE):
  1. CLI transport creates git worktree at `workspace_path` with correct isolation
  2. CLI transport extracts artifact files matching `files_to_modify` glob patterns
  3. CLI transport writes extracted artifacts to `artifacts/{task_id}/`
  4. API transport creates isolated workspace directory and writes full file artifacts
  5. Both CLI and API transport normalize to `patch_ready` state
  6. Windows paths validated for length (>260), spaces, and Chinese characters
  7. Windows worktree operations handle symlink permissions correctly
  8. `reverse_static_c_rebuild` task type accepted and enters `routed` with valid context
  9. Reverse tasks rejected at enqueue if required fields missing (`target_so_path`, `frida_hook_spec`, etc.)
  10. IDA MCP integration fetches static analysis in first loop iteration
  11. Frida oracle runs black-box hook on target device and captures output
  12. Static C rebuild compiles and runs to produce `static_output`
  13. Diff calculation produces `diff_report.json` with `match_rate`
  14. Loop continues until `match_rate = 100%`, then transitions to `patch_ready`
  15. Analysis state persisted to `RE-STATE.md` across loop iterations
  16. Recovery resumes from `analysis_state_md_path`, not mid-loop position
  17. `loop_iteration_count` correctly increments each cycle and resets on retry
  18. `max_loop_iterations` limit triggers `retry_waiting` with `reverse_loop_exhausted`
  19. Final artifact `final.c` is standalone compilable C code without unresolved offsets
  20. Environment unavailability (IDA/Frida) triggers retry with `reverse_env_unavailable`
  21. Reverse tasks with `reverse_final_artifact_invalid` fail validation at `verify_failed`
**Plans**: TBD

### Phase 4: Integration
**Goal**: Merge queue and Agent Connector with GSD implementation
**Depends on**: Phase 3
**Requirements**: MERG-01, MERG-02, MERG-03, MERG-04, MERG-05, MERG-06, AGNT-01, AGNT-02, AGNT-03, CONN-01, CONN-02, CONN-03
**Success Criteria** (what must be TRUE):
  1. Verified tasks automatically enqueue to merge queue single-consumer
  2. Merge queue consumes tasks ordered by `topo_rank` asc, then `created_at` asc
  3. Merge queue only consumes tasks with all dependencies in `done` state
  4. Merge operation copies artifact to main checkout, runs `git add` and `git commit`
  5. `apply_failed` state requires manual intervention, no auto-retry
  6. Merge status correctly updates task to `merged` then `done`
  7. Retry logic reschedules failed merges up to `max_retries` with 30s/60s backoff
  8. Connector interface defines `discoverTasks`, `hydrateContext`, `ackResult`, `writeBackArtifacts`
  9. GSD Connector implements full interface, generates Task Cards from PLAN
  10. GSD Connector fills `wave`, explicit `depends_on`, and `files_to_modify` paths
**Plans**: TBD

### Phase 5: Interface
**Goal**: HTTP API server and React Web UI for full platform control
**Depends on**: Phase 4
**Requirements**: API-01, API-02, API-03, UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07
**Success Criteria** (what must be TRUE):
  1. HTTP server starts on configured port with chi router
  2. Task CRUD API endpoints work (create, read, update, delete)
  3. Wave management API endpoints work (list, seal status, tasks by wave)
  4. WebSocket delivers real-time state transitions to connected clients
  5. React+Vite frontend builds without errors
  6. Web UI displays task list, wave board, state machine viewer, event log
  7. Web UI provides real-time updates via WebSocket
  8. Reverse validation tasks correctly check algorithm correctness from `artifacts/{task_id}/reverse/`
  9. Reverse validation ensures `match_rate = 100%` before allowing state progression
**Plans**: TBD
**UI hint**: yes

---

## Phase Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | ✓ Complete | 2026-04-07 |
| 2. Core Engine | — | ✓ Complete | 2026-04-09 |
| 3. Execution Layer | — | ✓ Complete | 2026-04-10 |
| 4. Integration | — | ✓ Complete | 2026-04-11 |
| 5. Interface | — | ✓ Complete | 2026-04-11 |

---

## Coverage

**Total v1 Requirements**: 56

| Category | Requirements | Phase |
|----------|--------------|-------|
| Persistence | PERS-01, PERS-02, PERS-03, PERS-04, PERS-05, PERS-06 | Phase 1 |
| Core | CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06 | Phase 2 |
| Wave | WAVE-01, WAVE-02, WAVE-03, WAVE-04, WAVE-05 | Phase 2 |
| State | STAT-01, STAT-02, STAT-03, STAT-04, STAT-05, STAT-06 | Phase 2 |
| Dependency | DEPD-01, DEPD-02, DEPD-03, DEPD-04, DEPD-05 | Phase 2 |
| Retry | RETR-01, RETR-02, RETR-03, RETR-04, RETR-05, RETR-06, RETR-07 | Phase 2 |
| Transport | TRAN-01, TRAN-02, TRAN-03, TRAN-04, TRAN-05, TRAN-06, TRAN-07 | Phase 3 |
| Reverse | REVR-01, REVR-02, REVR-03, REVR-04, REVR-05, REVR-06, REVR-07, REVR-08, REVR-09, REVR-10, REVR-11, REVR-12, REVR-13, REVR-14 | Phase 3 |
| Merge Queue | MERG-01, MERG-02, MERG-03, MERG-04, MERG-05, MERG-06 | Phase 4 |
| Agent | AGNT-01, AGNT-02, AGNT-03 | Phase 4 |
| Connector | CONN-01, CONN-02, CONN-03 | Phase 4 |
| API | API-01, API-02, API-03 | Phase 5 |
| UI | UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07 | Phase 5 |

**Coverage: 56/56 requirements mapped**

---

*Generated: 2026-04-06*
*Last updated: 2026-04-12 after roadmap sync — all 5 phases confirmed complete from codebase*