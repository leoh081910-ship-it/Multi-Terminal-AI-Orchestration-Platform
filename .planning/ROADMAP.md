# Roadmap - 多终端 AI 编排平台

## Project Overview
- **Goal**: Multi-AI task orchestration platform enabling full automation from task submit to merge
- **Core**: Go backend + React frontend with SQLite, 13-state machine, CLI/API transport
- **Key Features**: Wave-based scheduling, dependency topology, reverse engineering loop, GSD Connector

## Phases

- [ ] **Phase 1: Foundation** - SQLite persistence and project setup
- [ ] **Phase 2: Core Engine** - State machine and orchestration logic
- [ ] **Phase 3: Execution Layer** - Transport and reverse engineering
- [ ] **Phase 4: Integration** - Merge queue and Agent Connector
- [ ] **Phase 5: Interface** - HTTP API and React Web UI

---

## Phase Details

### Phase 1: Foundation
**Goal**: Database persistence and project infrastructure operational
**Depends on**: Nothing (first phase)
**Requirements**: SETUP-01, SETUP-02, SETUP-03, PERSISTENCE-01, PERSISTENCE-02, PERSISTENCE-03, PERSISTENCE-04, PERSISTENCE-05
**Success Criteria** (what must be TRUE):
  1. Go server starts without errors via `go run cmd/server/main.go`
  2. SQLite database creates `tasks`, `events`, `waves` tables with correct schema
  3. Process restart recovers server state from SQLite (task CRUD, wave state)
  4. Task Card upsert correctly persists `card_json` as business field source
  5. Event logging records all state transitions atomically with task updates
  6. Wave CRUD operations work (create, query, seal) with `(dispatch_ref, wave)` uniqueness enforcement
**Plans**: TBD

### Phase 2: Core Engine
**Goal**: 13-state machine orchestration with dependency and wave management
**Depends on**: Phase 1
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07, CORE-08, CORE-09
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
**Requirements**: TRANSPORT-01, TRANSPORT-02, TRANSPORT-03, TRANSPORT-04, TRANSPORT-05, TRANSPORT-06, TRANSPORT-07, TRANSPORT-08, REVERSE-01, REVERSE-02, REVERSE-03, REVERSE-04, REVERSE-05, REVERSE-06, REVERSE-07, REVERSE-08, REVERSE-09, REVERSE-10, REVERSE-11, REVERSE-12, REVERSE-13
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
**Requirements**: MERGE-01, MERGE-02, MERGE-03, MERGE-04, MERGE-05, MERGE-06, AGENT-01, AGENT-02, AGENT-03, AGENT-04
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
**Requirements**: UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07, REVERSE-05
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
| 1. Foundation | 0/1 | Not started | - |
| 2. Core Engine | 0/1 | Not started | - |
| 3. Execution Layer | 0/1 | Not started | - |
| 4. Integration | 0/1 | Not started | - |
| 5. Interface | 0/1 | Not started | - |

---

## Coverage

**Total v1 Requirements**: 56

| Category | Requirements | Phase |
|----------|--------------|-------|
| SETUP | SETUP-01, SETUP-02, SETUP-03 | Phase 1 |
| PERSISTENCE | PERSISTENCE-01, PERSISTENCE-02, PERSISTENCE-03, PERSISTENCE-04, PERSISTENCE-05 | Phase 1 |
| CORE | CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07, CORE-08, CORE-09 | Phase 2 |
| TRANSPORT | TRANSPORT-01, TRANSPORT-02, TRANSPORT-03, TRANSPORT-04, TRANSPORT-05, TRANSPORT-06, TRANSPORT-07, TRANSPORT-08 | Phase 3 |
| REVERSE | REVERSE-01, REVERSE-02, REVERSE-03, REVERSE-04, REVERSE-05, REVERSE-06, REVERSE-07, REVERSE-08, REVERSE-09, REVERSE-10, REVERSE-11, REVERSE-12, REVERSE-13 | Phase 3 |
| MERGE | MERGE-01, MERGE-02, MERGE-03, MERGE-04, MERGE-05, MERGE-06 | Phase 4 |
| AGENT | AGENT-01, AGENT-02, AGENT-03, AGENT-04 | Phase 4 |
| UI | UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07, REVERSE-05 | Phase 5 |

**Coverage: 56/56 requirements mapped ✓**

---

*Generated: 2026-04-06*