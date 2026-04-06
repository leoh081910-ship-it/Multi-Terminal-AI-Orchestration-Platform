# Feature Research

**Domain:** AI Task Orchestration / Multi-Agent Dispatch Platform
**Researched:** 2025-04-06
**Confidence:** MEDIUM-HIGH

*Note: Domain research via web search returned limited results. Analysis based on PRD v24 specification and domain expertise in AI agent orchestration systems.*

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Task Discovery & Registration | Every orchestration platform needs a way to receive tasks from external sources (Connectors, APIs, manual input) | MEDIUM | `discoverTasks()` is core entry point; must handle diverse input formats |
| Task State Management | Users need visibility into where each task is in its lifecycle | LOW | SQLite-backed state machine with clear transitions; users expect audit trails |
| Dependency Management | Complex work requires ordering - tasks that depend on other tasks must wait | HIGH | `depends_on` relations, topological ranking, wave-based batching |
| Workspace Isolation | Multiple tasks running simultaneously must not interfere with each other | HIGH | CLI uses git worktree, API uses isolated directories; prevents cross-task contamination |
| Artifact Collection | After task execution, need to capture what was produced | LOW | Files matching `files_to_modify` whitelist get captured to `artifacts/{task_id}/` |
| Result Merging | Individual task results must be combined into a coherent whole | MEDIUM | Git add + commit workflow; serial single-consumer queue |
| Error Handling & Recovery | Tasks fail; need sensible retry logic and resume capability | HIGH | `retry_count`, `retry_waiting` state, process recovery from SQLite |
| Deterministic Validation | Users need confidence that task outputs are correct | MEDIUM | Acceptance criteria, test commands, file existence checks |

### Differentiation (Project-Specific Advantages)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Wave-Based Batching | Granular control over task scheduling; can group related tasks and seal them as a unit | HIGH | Unique to this platform; enables "sealed waves" that cannot be modified post-submission |
| Reverse Engineering Task Type | Specialized handling for `reverse_static_c_rebuild` with quantifiable verification loops | VERY HIGH | Core differentiator; IDA Pro + Frida integration, 100% match_rate requirement |
| Quantitative Verification Loop | For reverse tasks, automated diff between static C rebuild and Frida oracle output | HIGH | `match_rate = 100%` as exit condition; automatic loop iteration |
| Context-Aware Recovery | After crashes/process restarts, tasks resume from correct state without data loss | HIGH | Maintains `loop_iteration_count`, preserves intermediate artifacts, reloads state from SQLite |
| Multi-Transport Support | CLI (git worktree) and API transport normalized to same workflow | MEDIUM | Unified `patch_ready` onward path regardless of transport origin |
| Analysis State Persistence | Reverse tasks maintain authoritative `.md` state file across iterations and sessions | MEDIUM | `analysis_state_md_path` as single source of truth; prevents context loss |
| Conflict Detection | Automatically identifies tasks that cannot run in parallel | MEDIUM | `conflicts_with` computed per wave; prevents resource contention |
| Transport Normalization | Different execution environments (CLI/API) converge to same processing pipeline | LOW | Key architectural principle from PRD |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-Time Web Dashboard | Users want live visibility into task progress | Adds significant complexity; requires WebSocket/polling infrastructure; distracts from CLI-first design | Defer to v2; use SQLite queries and CLI status commands for v1 |
| Cross-Wave Implicit Ordering | "Wave 2 tasks should naturally run after Wave 1" | Creates hidden dependencies; violates explicit `depends_on` principle; makes behavior non-deterministic | Use explicit `depends_on` to declare cross-wave dependencies |
| Automatic Conflict Resolution | "Platform should figure out how to resolve conflicts" | Conflicts are often semantic (not technical); auto-resolution could produce wrong results | Manual intervention required; platform rejects invalid routing, not resolves |
| Persistent Background Workers | "Keep workers running, don't restart for each task" | Adds stateful runtime complexity; complicates recovery; introduces resource leaks | Stateless task processing; each execution starts fresh from SQLite state |
| Cross-Wave Dependency Inference | "Platform should guess dependencies from file names" | Guessing leads to incorrect assumptions; hidden coupling breaks maintainability | Require explicit `depends_on` declaration |
| Wave Sealing Reversal | "Allow adding tasks to already-sealed wave" | Breaks sealed batch semantics; could cause missing dependencies | Reject with `reason = "wave_already_sealed"`; create new wave instead |
| Automatic Retry Scheduling | "Platform should retry failed tasks automatically without limit" | Could infinite-loop on broken tasks; wastes resources; hides underlying issues | Configurable `max_retries = 2` default; explicit retry waiting state |

## Feature Dependencies

```
[Task Discovery (discoverTasks)]
    └──generates──> [Task Card Creation]
                           └──requires──> [Wave Assignment]
                                              └──requires──> [Dependency Analysis]
                                                                     └──computes──> [Topological Ranking]

[Workspace Isolation]
    └──enables──> [Parallel Execution]
                       └──requires──> [Conflict Detection]

[Artifact Collection]
    └──feeds──> [Result Merging]
                    └──requires──> [Git Commit Pipeline]

[Reverse Task Execution]
    └──requires──> [IDA Integration (ida-pro-mcp)]
    └──requires──> [Frida Dynamic Analysis]
    └──requires──> [Quantitative Diff Engine]
    └──requires──> [Analysis State Persistence]
```

### Dependency Notes

- **Task Discovery requires Wave Assignment:** Before tasks can be routed, they must be assigned to a wave (batch)
- **Wave Assignment requires Dependency Analysis:** Cannot finalize wave membership without understanding task relationships
- **Dependency Analysis computes Topological Ranking:** Used for merge queue ordering (`topo_rank` field)
- **Workspace Isolation enables Parallel Execution:** Each task gets isolated environment (git worktree or API dir)
- **Conflict Detection prevents Parallel Execution issues:** Automatically identifies tasks that cannot share execution context
- **Reverse Task requires multiple integrations:** IDA Pro MCP + Frida + Diff Engine are all mandatory for `reverse_static_c_rebuild`

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed to validate the concept.

- [x] Task Discovery & Registration via Connector interface
- [x] SQLite-backed state machine (queued → routed → running → done/failed)
- [x] Wave-based task batching with seal semantics
- [x] Dependency management (`depends_on`, topological ranking)
- [x] Workspace isolation (CLI: git worktree, API: isolated directory)
- [x] Artifact collection via `files_to_modify` whitelist
- [x] Result merging to main checkout via git add + commit
- [x] Basic retry logic with configurable max_retries
- [x] Reverse engineering task type (`reverse_static_c_rebuild`) with quantifiable verification loop

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] Rich CLI status interface (beyond basic state query)
- [ ] Task cancellation (graceful termination)
- [ ] Wave-level pause/resume
- [ ] Configurable retry backoff strategies
- [ ] Workspace TTL management UI/CLI

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] Real-time web dashboard with WebSocket updates
- [ ] Multi-user collaboration support
- [ ] Cloud storage backend (currently SQLite-only)
- [ ] Marketplace for pre-built Connectors
- [ ] Visual dependency graph builder
- [ ] SLA monitoring and alerting

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Task Discovery & Registration | HIGH | MEDIUM | P1 |
| State Management (SQLite) | HIGH | LOW | P1 |
| Wave-Based Batching | HIGH | HIGH | P1 |
| Dependency Management | HIGH | HIGH | P1 |
| Workspace Isolation (worktree/dir) | HIGH | HIGH | P1 |
| Artifact Collection | HIGH | LOW | P1 |
| Result Merging | HIGH | MEDIUM | P1 |
| Error Handling & Recovery | HIGH | HIGH | P1 |
| Reverse Task with Verification Loop | HIGH | VERY HIGH | P1 |
| Conflict Detection | MEDIUM | MEDIUM | P1 |
| Analysis State Persistence | MEDIUM | MEDIUM | P1 |
| CLI Status Interface | MEDIUM | LOW | P2 |
| Task Cancellation | MEDIUM | MEDIUM | P2 |
| Configurable Backoff | LOW | LOW | P3 |
| Web Dashboard | MEDIUM | HIGH | P3 |
| Multi-User Support | MEDIUM | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | AutoGen | CrewAI | LangChain Agents | Our Approach |
|---------|---------|--------|------------------|--------------|
| Task Orchestration | Agent-to-agent conversation | Sequential/parallel workflows | Chain/LangGraph | Wave-based batching with explicit dependencies |
| State Persistence | In-memory | In-memory + DB optional | Varies | SQLite-native with event audit log |
| Workspace Isolation | Shared context | Shared environment | Shared environment | Git worktree (CLI) / isolated dir (API) |
| Dependency Management | Implicit via conversation | Explicit crew roles | Explicit chain definitions | Explicit `depends_on` relation + topological ranking |
| Reverse Engineering Support | None | None | None | Specialized `reverse_static_c_rebuild` with quantitative loop |
| Verification Loop | LLM-based eval | Task-specific | Varies | Quantitative diff (100% match_rate) for reverse tasks |
| Conflict Detection | None | None | None | `conflicts_with` computed per wave |
| Recovery/Resume | Limited | Limited | Limited | Full SQLite-based recovery with state preservation |
| Merge Strategy | N/A | N/A | N/A | Git add + commit with serial queue |

## Domain-Specific Notes

### What Makes This Platform Unique

1. **Quantitative Verification for Reverse Tasks**: Unlike general AI agent frameworks that use LLM-based evaluation, this platform requires byte-level exact match (100% match_rate) between static C rebuild and Frida oracle output for reverse engineering tasks.

2. **Wave Sealing Semantics**: The sealed wave model is distinct - once a wave is sealed (committed to routing), no new tasks can be added. This provides stronger consistency guarantees than typical task queue batching.

3. **Single Source of Truth**: SQLite `tasks` table as definitive state store, with `events` for audit only. This design decision simplifies recovery and avoids dual-state synchronization problems.

4. **Transport Normalization**: CLI and API transports converge at `patch_ready` state, abstracting away execution environment differences.

5. **Reverse Task State Persistence**: Unlike typical agent frameworks that lose context on restart, this platform maintains `analysis_state_md_path` and preserves `loop_iteration_count` across process restarts.

### Sources

- PRD v24: Multi-terminal AI Orchestration Platform (detailed specification analysis)
- Domain expertise in AI agent orchestration, task queuing systems, and reverse engineering workflows
- Comparison with AutoGen, CrewAI, LangChain agent frameworks (general capability mapping)

---

*Feature research for: AI Task Orchestration Platform*
*Researched: 2025-04-06*