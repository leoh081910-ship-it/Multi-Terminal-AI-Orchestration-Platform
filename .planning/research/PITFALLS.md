# Pitfalls Research

**Domain:** AI Task Orchestration Platform (Multi-terminal)
**Researched:** 2026-04-06
**Confidence:** MEDIUM to HIGH (based on established patterns in SQLite/git/state-machine concurrency; some areas would benefit from additional verification)

## Critical Pitfalls

### Pitfall 1: SQLite Concurrency Locking Under Multi-goroutine Access

**What goes wrong:**
SQLite database locked errors, particularly "database is locked" during high-concurrency task processing. The `tasks` table receives simultaneous writes from multiple goroutines (router, executor, merger, recovery daemon) leading to SQLITE_BUSY failures.

**Why it happens:**
SQLite uses file-level locking. Default configuration allows only one writer at a time. With the PRD's design requiring concurrent state transitions across multiple components (Control Plane, Worker Pool, Merge Queue Consumer), the default single-writer model becomes a bottleneck. Additionally, long-running transactions hold locks, blocking other writers.

**How to avoid:**
1. Enable WAL (Write-Ahead Logging) mode: `PRAGMA journal_mode=WAL`
2. Set appropriate busy_timeout: `PRAGMA busy_timeout=5000` (5 seconds)
3. Use connection pooling with limited pool size
4. Implement write serialization at application level with sync.Mutex for critical state transitions
5. Keep transactions short - never hold db connections across await points

**Warning signs:**
- "database is locked" in logs occurring more than 1% of write attempts
- Transaction timeout errors increasing under load
- High latency in state transition operations when multiple tasks process in parallel

**Phase to address:** Phase 2 (Core Database) — SQLite configuration and connection pooling must be established before task execution components

---

### Pitfall 2: Git Worktree Management on Windows (Long Paths, Chinese Characters, Symlink Permissions)

**What goes wrong:**
- Git worktree creation fails on Windows due to 260-character MAX_PATH limit
- File paths containing Chinese characters cause encoding issues in git
- Worktrees created in deep nested directories exceed Windows path limits
- Symlink operations fail without administrator privileges on Windows

**Why it happens:**
Windows has fundamental limitations that Linux/macOS don't have:
- 260 character hard limit on MAX_PATH by default
- NTFS symlinks require SeCreateSymbolicLinkPrivilege (admin rights)
- Git's default behavior on Windows doesn't handle long paths well
- Chinese/Japanese characters in paths may cause UTF-8 encoding mismatches

The CLI transport in PRD Section 3.2.1 relies heavily on git worktrees, making this a critical Windows compatibility issue.

**How to avoid:**
1. Global git config: `git config --global core.longpaths true`
2. Enable Windows Long Path support via Group Policy or registry
3. Never create worktrees in nested paths > 200 characters from drive root
4. Avoid Chinese characters in workspace paths - use ASCII placeholders
5. For symlinks: use junction points (directories) or require admin privilege check
6. If platform requires symlink for artifacts, detect Windows and fallback to file copy

**Warning signs:**
- `error: unable to create directory` during worktree add
- `invalid symlink` or `permission denied` errors
- File operations succeed but git status shows nothing
- Path length errors when artifact_path exceeds 200 chars

**Phase to address:** Phase 3 (Transport Layer) — Windows path handling must be implemented before CLI transport testing

---

### Pitfall 3: State Machine Race Conditions in Concurrent Transitions

**What goes wrong:**
Tasks transition through states in unpredictable order. A task marked "done" might have been stuck in "running", or two tasks transition to "routed" simultaneously creating duplicate work.

**Why it happens:**
The PRD defines a complex state machine with multiple entry points:
- Dependency failure propagation triggers concurrent transitions
- Recovery daemon may race with normal execution
- WebSocket callbacks update state while worker processes update the same task
- No optimistic locking or version контроль on state transitions

**How to avoid:**
1. Implement optimistic locking using a `version` column on tasks table
2. Require `expected_state` on every state transition query: `UPDATE tasks SET state = ? WHERE id = ? AND state = ?`
3. Use database transactions for multi-table updates (state + event atomically)
4. Implement a state transition validator that rejects invalid state pairs
5. Add mutex protection for in-memory state caching

**Warning signs:**
- State machine reaches invalid states (e.g., merged before running)
- Events show impossible sequences (queued → merged without intermediate states)
- Race condition errors in logs
- Duplicate merge attempts for same task

**Phase to address:** Phase 2 (Core Database) — State machine correctness must be verified before any task execution

---

### Pitfall 4: Glob Pattern Mismatches Across Platforms

**What goes wrong:**
- Artifacts not extracted because glob patterns work differently on Windows
- `files_to_modify` whitelist fails on paths with backslashes vs forward slashes
- Case sensitivity differences (Windows is case-insensitive, Linux is not)
- Path normalization issues when workspace path contains mixed separators

**Why it happens:**
Go's `filepath.Walk` and `path.Match` behave differently:
- Windows uses backslash (`\`) as separator, but git always uses forward slash (`/`)
- `filepath.Glob` respects OS case sensitivity
- Unicode normalization differences between platforms

Per PRD Section 3.2.1, CLI transport applies `files_to_modify` whitelist with glob at execution time, making this critical.

**How to avoid:**
1. Always normalize paths to forward slashes before applying glob patterns
2. Use case-insensitive matching on Windows, or explicitly document platform requirements
3. Test glob patterns on both Windows and Linux before deployment
4. Provide clear error messages when glob matches empty set (per PRD's `empty_artifact_match` handling)
5. Consider using filepath.Glob with custom WalkDir that normalizes separators

**Warning signs:**
- CLI transport returns empty artifact set on Windows but works on Linux
- `empty_artifact_match` errors occurring on specific platforms
- Files exist in worktree but aren't captured in artifact extraction

**Phase to address:** Phase 3 (Transport Layer) — Cross-platform glob testing before task execution phase

---

### Pitfall 5: Reverse Engineering Verification Loop Premature Completion

**What goes wrong:**
- `reverse_static_c_rebuild` tasks marked as completed with `match_rate < 100%`
- Intermediate artifacts treated as final output after partial recovery
- `analysis_state_md_path` not properly restored after session reset
- Frida oracle failures incorrectly treated as task failure instead of loop retry

**Why it happens:**
The PRD Section 4 defines strict quantification rules (Rule R1):
- Static C code compilation output must match Frida hook output 100%
- The verification loop must continue until `match_rate = 100%`
- Recovery must restore loop iteration count and reload state from `.md` file

Common mistakes:
- Treating ANY non-100% match as success
- Loading `analysis_state_md_path` after starting loop instead of BEFORE
- Confusing `oracle_mismatch` (loop retry reason) with `verify_failed` (state transition)
- Not preserving intermediate artifacts for next iteration analysis

**How to avoid:**
1. Create explicit hard gate: `if matchRate < 100.0 { continue }` loop
2. Recovery procedure MUST load `.md` state file before any processing
3. Mark `oracle_mismatch` as loop-internal, never trigger state transition
4. Ensure `loop_iteration_count` persists across crashes
5. Verify `final.c` satisfies all criteria in PRD Section 4.7 before allowing `patch_ready` transition

**Warning signs:**
- `artifacts/{task_id}/reverse/final.c` exists with match_rate < 100 in `diff_report.json`
- `loop_iteration_count` not incrementing between iterations
- `analysis_state_md_path` modified after loop starts (should only be read, never written during loop)
- Tasks transitioning to `patch_ready` without all required artifacts in `reverse/` directory

**Phase to address:** Phase 4 (Reverse Tasks) — This is unique to reverse engineering, must have specialized validation

---

### Pitfall 6: Dependency Failure Cascading Ignored

**What goes wrong:**
- Upstream task fails, but downstream tasks continue execution
- `dependency_failed` reason code never appears
- Failed tasks pile up without triggering fail-fast on dependents
- Circular dependencies not detected early, causing infinite wait

**Why it happens:**
Per PRD Section 3.1.9:
- "依赖失败传播由 Control Plane 立即触发"
- "当前仍在 running 的后置任务可被直接标记为 failed"

This requires immediate cascading after any dependency failure. Common issues:
- Propagation only checks at task start, not during execution
- Event-driven propagation missed if Control Plane crashes
- Wave boundaries incorrectly blocking propagation (depends_on must cross waves per Section 2.5)

**How to avoid:**
1. Immediately query all dependents when any task enters failed/apply_failed state
2. Use database trigger or event-driven listener for propagation
3. At startup/recovery, re-evaluate all non-terminal tasks against their dependency graph
4. Detect circular dependencies at task enqueue time, not at execution
5. Propagate through wave boundaries (PRD explicitly allows depends_on between waves)

**Warning signs:**
- Tasks remain in `running` state after their dependencies are `failed`
- `dependency_failed` reason not appearing in events table
- Wave seal accepts tasks with impossible dependencies (later wave depends on earlier)
- Deadlock situations where task waits forever for completed-but-failed dependency

**Phase to address:** Phase 2 (Core Database) — Dependency graph and propagation must be correct before execution

---

### Pitfall 7: Process Crash Fails to Restore In-Flight Tasks

**What goes wrong:**
- Platform restarts after crash but in-flight tasks remain stuck in old states
- Tasks in `running` state never resume, causing workflow deadlocks
- `retry_waiting` tasks don't resume after timeout
- Wave state inconsistent after crash (some tasks in wave completed, some stuck)

**Why it happens:**
PRD Section 3.4 covers recovery, but common gaps:
- Not persisting current iteration count and loop state to database
- Process exit during critical section leaves DB in inconsistent state
- Not restoring transport-specific state (worktree references, API connection)
- Wave unsealed/consistency not verified on restart

**How to avoid:**
1. At minimum, persist every state transition immediately (don't batch)
2. For reverse tasks, must restore full loop iteration count from events table
3. Implement health check that queries all non-terminal tasks on startup
4. For `running` tasks, either:
   - Resume execution (if idempotent and safe)
   - Mark as `retry_waiting` with `process_resume` reason
5. Verify wave consistency: if any task in wave is non-terminal, ensure wave is not sealed
6. Use `events` table to rebuild full task state on recovery

**Warning signs:**
- Tasks stuck in `running` after process restart
- Wave shows inconsistent state (sealed but contains non-terminal tasks)
- Recovery daemon unable to determine what to resume
- Reverse tasks starting from iteration 0 after recovery (should preserve count)

**Phase to address:** Phase 2 (Core Database) — Recovery mechanisms must be solid before any "in production" deployment

---

### Pitfall 8: WebSocket Reconnection Loses State Synchronization

**What goes wrong:**
- WebSocket reconnects but misses state updates that occurred during disconnect
- Client shows stale task states while server has progressed
- Duplicate events processed after reconnection
- Race between server push and client re-fetch

**Why it happens:**
WebSocket-only state management without compensation:
- No sequence ID tracking for state updates
- Client relies purely on push without periodic reconciliation
- Connection drop during state transition leads to inconsistent views

**How to avoid:**
1. Implement event sequence numbering (monotonic counter per task or globally)
2. On reconnect, client sends last known sequence ID, server sends delta
3. Implement periodic full state reconciliation (every 30-60 seconds) regardless of events
4. Use optimistic UI with "at-least-once" delivery acknowledgment
5. Server maintains last-N events in memory or database for missed message recovery

**Warning signs:**
- Clients showing wrong task states after network hiccup
- Multiple clicks required to refresh to correct state
- Events appear duplicated or missing after reconnect
- Users report "stale" data despite successful operations

**Phase to address:** Phase 5 (Connector Integration) — Likely not in v1 scope (PRD says single user, local), but plan for multi-client future

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip WAL mode, use default journal | Simpler config, fewer files | Severe write contention under load | Never - WAL is essential for concurrency |
| No version column on tasks | Simpler schema | State race conditions, data corruption | Never for this project |
| Synchronous state transitions | Simpler code | Performance bottleneck | Only for single-task demo |
| Skip wave consistency check | Faster wave sealing | Invalid state after crash | Never - consistency is fundamental |
| No reverse loop iteration persistence | Faster development | Loss of analysis progress after crash | Never for reverse tasks |
| Skip dependency propagation enforcement | Simpler Control Plane | Cascading failures missed | Never - per PRD Requirement |
| Use file copy instead of symlink on Windows | Works without admin | Broken artifact link semantics | Acceptable as fallback, document clearly |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GSD Connector | Not populating `files_to_modify` with expected new file paths | Must call GSD with all expected output paths pre-known |
| IDA Pro MCP | Assuming IDA MCP always available | Must check `ida_mcp_endpoint` before routing reverse task |
| Frida | Not handling injection failures gracefully | Must treat as recoverable environment error |
| git worktree | Worktree not cleaned up after merge failure | Must implement cleanup on failed/apply_failed transitions |
| SQLite | Connection not closed after use | Use connection pool with proper lifecycle management |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Single DB writer bottleneck | All task operations serialize | WAL mode + connection pool with write serialization | > 3 concurrent goroutines writing |
| Large card_json serialization | Slow state transitions | Store only JSON, optimize serialization only if measured | > 10KB per `card_json` |
| Reverse task too many iterations | Platform hangs on single task | `max_loop_iterations = 50` per PRD | Unbounded iterations |
| Wave with too many tasks | Memory pressure during routing | Cap wave size or use streaming | > 1000 tasks in single wave |
| Event table without cleanup | Database grows unbounded | TTL-based cleanup for events | > 1 year of continuous operation |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Arbitrary file write to artifacts/{task_id} | Path traversal attack via crafted task_id | Validate task_id matches `[a-z0-9_-]{1,16}` before any file operation |
| Executing test_command from untrusted source | Command injection | Never execute user-provided commands; only use pre-approved command patterns |
| Storing full context including credentials in card_json | Credential leakage | Exclude sensitive fields from card_json persistence or encrypt |
| Worktree in user-controlled directory | Symlink attack | Restrict worktree root to platform-controlled paths |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent state transition failures | User thinks task completed but it's stuck | Always log state transitions with clear reason codes |
| No visibility into reverse iteration progress | User waits forever without progress indication | Show loop count and last match_rate in task status |
| apply_failed requires manual intervention but no UI | User doesn't know what to do | Surface apply_failed tasks prominently with resolution guidance |
| Wave sealed but tasks still pending | Confusing overall status | Show wave seal status separately from task completion |

---

## "Looks Done But Isn't" Checklist

- [ ] **State Transition:** Transitions appear in logs but may not have persisted to database - verify `SELECT state FROM tasks` returns expected value
- [ ] **Artifacts:** Files copied to main checkout but merge not committed - verify git commit exists
- [ ] **Reverse Loop:** Iteration appears to advance but `loop_iteration_count` not updated in DB - query database directly
- [ ] **Dependency Propagation:** Downstream tasks marked failed but upstream still running - check events table for `dependency_failed` event
- [ ] **Wave Seal:** Wave appears sealed but new tasks still being added - query `waves.sealed_at IS NOT NULL`
- [ ] **Connection State:** WebSocket connected but event stream stopped - verify sequence IDs incrementing

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| SQLite locked at startup | LOW | Restart platform, WAL cleanup if needed |
| Worktree path too long | MEDIUM | Move workspace to shorter root path |
| State machine in invalid state | HIGH | Restore from events table, verify each task's history |
| Reverse task loop stuck at < 100% | HIGH | Check Frida accessibility, verify analysis_state_md_path is readable |
| In-flight task lost after crash | MEDIUM | Query events for last known state, determine resume point |
| Wave inconsistency | MEDIUM | Unseal wave if tasks stuck, re-evaluate dependency graph |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification Method |
|---------|------------------|---------------------|
| SQLite concurrency | Phase 2: Core Database | Write test with 10 concurrent goroutines, measure lock contention |
| Windows path issues | Phase 3: Transport Layer | Test worktree creation on Windows with Chinese characters in project path |
| State machine races | Phase 2: Core Database | Unit test all state transitions with mock concurrent access |
| Glob patterns | Phase 3: Transport Layer | Cross-platform test matrix - verify extract results match on Windows/Linux |
| Reverse verification loop | Phase 4: Reverse Tasks | Load test with broken C code, verify loop continues until 100% match |
| Dependency propagation | Phase 2: Core Database | Create dependency graph failure scenario, verify all dependents fail fast |
| Crash recovery | Phase 2: Core Database | Simulate SIGKILL during task execution, verify in-flight tasks recover |
| WebSocket sync | Phase 5: Connectors | Disconnect/reconnect test, verify no state loss |

---

## Sources

- SQLite concurrency: go-sqlite3 documentation, SQLite WAL mode specifications
- Windows long paths: Microsoft docs on MAX_PATH limitations and Group Policy settings
- Git worktree: git-worktree manual, known Windows compatibility issues
- State machine concurrency: Go memory model, optimistic locking patterns
- Glob patterns: Go filepath package documentation, cross-platform path handling
- Reverse engineering verification: Industry practice in binary analysis (Rule R1 from PRD)
- Dependency propagation: Workflow engine failure handling patterns
- Process recovery: Event sourcing, checkpoint-restart patterns
- WebSocket reconnection: RFC 6455, state synchronization best practices

---

*Pitfalls research for: Multi-terminal AI Orchestration Platform*
*Researched: 2026-04-06*