# Phase 1: Foundation - Research

**Researched:** 2026-04-06
**Domain:** Go server infrastructure with SQLite persistence
**Confidence:** HIGH (verified versions, clear requirements from PRD/CONTEXT)

## Summary

Phase 1 establishes the foundation for the AI task orchestration platform: a Go server that starts cleanly, uses SQLite for persistence, and correctly models the task/event/wave data structures. Key finding: use pure Go SQLite driver (`modernc.org/sqlite` v1.48.1) since no C compiler is available on the system. The project structure follows standard Go layout with functional layer packaging in `internal/`.

**Primary recommendation:** Use `modernc.org/sqlite` for database, `viper` for configuration, `zerolog` for logging. Implement store layer first, then wire to cmd/server/main.go. Ensure WAL mode and busy_timeout per CONTEXT.md decisions.

---

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Use Standard Go Layout: `cmd/server`, `cmd/cli`, `internal/` packages
- Functional layer packaging: `store`, `engine`, `transport`, `api`, `connector`
- SQLite with WAL mode + busy_timeout
- Phase 1: Foundation limited to basic CRUD - no routing logic, no wave processing

### Claude's Discretion
- **Migration tool:** Choose between embedded migrations, migrate tool, or golang-migrate
- **Config format:** JSON vs TOML vs YAML - recommend YAML for PRD simplicity
- **Logging framework:** zerolog vs slog vs log - recommend zerolog for structured logging

### Deferred Ideas (OUT OF SCOPE)
- CLI transport implementation
- GSD connector integration
- IDA Pro MCP integration
- WebSocket API
- Reverse engineering tasks
- Cross-platform path handling for Windows

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETUP-01 | Go server starts without errors via `go run cmd/server/main.go` | Standard Go layout research; verified Go 1.26.1 available |
| SETUP-02 | SQLite database creates `tasks`, `events`, `waves` tables with correct schema | Schema from PRD sections 3.3, 3.4, 4.1; verified modernc.org/sqlite driver |
| SETUP-03 | Process restart recovers server state from SQLite (task CRUD, wave state) | Recovery design from PITFALLS.md section 7 |
| PERSISTENCE-01 | Task Card upsert correctly persists `card_json` as business field source | Verified by schema - card_json TEXT NOT NULL in tasks table |
| PERSISTENCE-02 | Event logging records all state transitions atomically with task updates | Transaction design from PRD section 3.3 |
| PERSISTENCE-03 | Wave CRUD operations work (create, query) | Schema includes waves table with unique constraint on (dispatch_ref, wave) |
| PERSISTENCE-04 | Wave seal operation works | Verified by schema - sealed_at column in waves table |
| PERSISTENCE-05 | (dispatch_ref, wave) uniqueness enforcement | Unique constraint verified in PRD section 3.3 |

</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.1 | Runtime | Available on system, recent stable |
| modernc.org/sqlite | v1.48.1 | SQLite driver | Pure Go (no CGO), database/sql compatible, latest from 2026-04-03 |
| github.com/spf13/viper | v1.21.0 | Configuration | Supports YAML/JSON/TOML/ENV, case-insensitive keys, watches config files |
| github.com/rs/zerolog | v1.35.0 | Logging | Zero-allocation JSON logging, structured logging best practice |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/golang-migrate/migrate | v4.18.0 | Database migrations | When schema versioning needed |
| github.com/stretchr/testify | v1.10.0 | Test assertions | For unit testing |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| modernc.org/sqlite | github.com/mattn/go-sqlite3 | go-sqlite3 requires CGO (GCC) which is not available |
| modernc.org/sqlite | github.com/glebarez/sqlite | Pure Go alternative (based on modernc), but modernc is more actively maintained |
| Viper | standard library json.Decoder | More boilerplate but zero dependency; use for minimal config |
| zerolog | log/slog (std) | slog is built-in (Go 1.21+), but zerolog has better ergonomics and benchmarking |

**Installation:**
```bash
go mod init github.com/mCP-DevOS/ai-orchestration-platform
go get modernc.org/sqlite@v1.48.1
go get github.com/spf13/viper@v1.21.0
go get github.com/rs/zerolog@v1.35.0
go get github.com/stretchr/testify@v1.10.0
```

## Architecture Patterns

### Recommended Project Structure
```
ai-orchestration-platform/
├── cmd/
│   ├── server/main.go          # Server entry point (minimal)
│   └── cli/main.go             # CLI entry point
├── internal/
│   ├── store/                  # Database layer
│   │   ├── tasks.go           # Task CRUD operations
│   │   ├── events.go          # Event logging
│   │   ├── waves.go           # Wave CRUD
│   │   └── migrations/        # Schema migrations
│   ├── engine/                 # Business logic (not needed in Phase 1)
│   ├── transport/              # Transport protocols (not needed in Phase 1)
│   ├── api/                    # HTTP handlers (not needed in Phase 1)
│   └── connector/              # External connectors (not needed in Phase 1)
├── config.yaml                 # Application config (YAML per recommendation)
├── go.mod
└── go.sum
```

### Pattern 1: Store Layer with Database Transactions
**What:** All database operations use `database/sql` with explicit transactions for atomic operations
**When to use:** When updating tasks + events together, or waves + tasks
**Example:**
```go
// Wraps a transaction with automatic rollback on error
func (s *Store) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
        }
        return err
    }
    return tx.Commit()
}
```

### Pattern 2: SQLite WAL Mode Configuration
**What:** Enable Write-Ahead Logging for concurrent access
**When to use:** On every new database connection
**Example:**
```go
// PRAGMA settings must be set after opening connection
func configureSQLite(db *sql.DB) error {
    // Enable WAL mode for concurrent writes
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        return fmt.Errorf("set WAL mode: %w", err)
    }
    // Set busy timeout to 5 seconds
    if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
        return fmt.Errorf("set busy_timeout: %w", err)
    }
    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
        return fmt.Errorf("enable foreign keys: %w", err)
    }
    return nil
}
```

### Pattern 3: Task Upsert with card_json
**What:** Insert or update task, always persisting full card_json as source of truth
**When to use:** When enqueuing new tasks or updating existing ones
**Example:**
```go
func (s *Store) UpsertTask(ctx context.Context, task *Task) error {
    query := `
        INSERT INTO tasks (id, dispatch_ref, state, retry_count, loop_iteration_count,
            transport, wave, topo_rank, workspace_path, artifact_path,
            last_error_reason, created_at, updated_at, terminal_at, card_json)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            state = excluded.state,
            retry_count = excluded.retry_count,
            updated_at = excluded.updated_at,
            terminal_at = excluded.terminal_at,
            card_json = excluded.card_json
    `
    _, err := s.db.ExecContext(ctx, query,
        task.ID, task.DispatchRef, task.State, task.RetryCount, task.LoopIterationCount,
        task.Transport, task.Wave, task.TopoRank, task.WorkspacePath, task.ArtifactPath,
        task.LastErrorReason, task.CreatedAt, task.UpdatedAt, task.TerminalAt, task.CardJSON,
    )
    return err
}
```

### Pattern 4: Atomic Task + Event Update
**What:** Use transaction to ensure task state change and event log are written atomically
**When to use:** For every state transition
**Example:**
```go
func (s *Store) TransitionTask(ctx context.Context, taskID, fromState, toState, reason string) error {
    return s.withTx(ctx, func(tx *sql.Tx) error {
        // Update task state
        _, err := tx.ExecContext(ctx,
            "UPDATE tasks SET state = ?, updated_at = ? WHERE id = ? AND state = ?",
            toState, time.Now().UTC(), taskID, fromState)
        if err != nil {
            return err
        }
        // Insert event atomically
        _, err = tx.ExecContext(ctx,
            `INSERT INTO events (event_id, task_id, event_type, from_state, to_state, timestamp, reason)
             VALUES (?, ?, ?, ?, ?, ?, ?)`,
            uuid.New().String(), taskID, "STATE_TRANSITION", fromState, toState, time.Now().UTC(), reason)
        return err
    })
}
```

### Anti-Patterns to Avoid
- **Skip WAL mode:** Will cause "database is locked" errors under concurrent writes
- **Skip busy_timeout:** Default 0 means immediate failure on lock
- **No foreign keys:** Risk of orphan records in events table
- **Separate queries for task+event:** Not atomic - can lose event if task update succeeds but event fails
- **Store card_json as NULL:** Per PRD, card_json is TEXT NOT NULL - always persist full JSON

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQLite driver | Custom wrapper | modernc.org/sqlite | Battle-tested, pure Go, actively maintained |
| Configuration parsing | Manual JSON/YAML parsing | spf13/viper | Handles all formats, env vars, defaults, watches |
| Logging boilerplate | Custom log formatting | rs/zerolog | Zero-allocation, structured, hooks, console output |
| UUID generation | Custom ID generation | github.com/google/uuid | Standard, collision-resistant |

**Key insight:** Building custom versions of these components adds maintenance burden without value. The only decision is *which* established library - not whether to use one.

---

## Runtime State Inventory

> This section is for rename/refactor/migration phases. Phase 1 is greenfield - no existing runtime state exists.

**Not applicable:** Phase 1 creates the database from scratch. No existing data to migrate.

---

## Common Pitfalls

### Pitfall 1: SQLite Concurrency Locking Under Multi-goroutine Access
**What goes wrong:** "database is locked" errors during concurrent task processing
**Why it happens:** SQLite default journal mode (DELETE) allows only one writer; no busy_timeout
**How to avoid:** Enable WAL mode `PRAGMA journal_mode=WAL`, set `PRAGMA busy_timeout=5000`
**Warning signs:** "database is locked" in logs during high-concurrency writes

**This phase addresses:** SETUP-02 (WAL mode on table creation)

### Pitfall 2: State Machine Race Conditions in Concurrent Transitions
**What goes wrong:** Two goroutines transition same task simultaneously, state becomes inconsistent
**Why it happens:** No optimistic locking on state transitions
**How to avoid:** Use `UPDATE ... WHERE state = expected` pattern - require current state matches
**Warning signs:** Tasks in impossible states, duplicate events

**This phase addresses:** PERSISTENCE-02 (ensure event + task updated atomically)

### Pitfall 3: Process Crash Fails to Restore In-Flight Tasks
**What goes wrong:** After restart, tasks in "running" state remain stuck
**Why it happens:** Process exited during task execution, state not updated
**How to avoid:** Implement recovery on startup - scan non-terminal tasks, resume or mark as retry_waiting
**Warning signs:** Tasks stuck in running after restart, wave state inconsistent

**This phase addresses:** SETUP-03 (basic recovery, but full implementation may extend to Phase 2)

### Pitfall 4: card_json NULL or Empty
**What goes wrong:** Business logic fails because card_json is the source of truth
**Why it happens:** Upsert doesn't properly serialize full Task Card
**How to avoid:** Enforce NOT NULL constraint, validate on write
**Warning signs:** Queries for business fields return NULL when they shouldn't

**This phase addresses:** PERSISTENCE-01 (ensure card_json always stored)

---

## Code Examples

### Server Main Entry (Minimal Phase 1)
```go
// cmd/server/main.go
// Source: Standard Go project layout + research
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/mCP-DevOS/ai-orchestration-platform/internal/store"
    "github.com/rs/zerolog"
    "github.com/spf13/viper"
)

func main() {
    // Initialize logger
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    logger := zerolog.New(os.Stderr).With().Timestamp().Str("component", "server").Logger()

    // Load configuration (YAML)
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    if err := viper.ReadInConfig(); err != nil {
        logger.Fatal().Err(err).Msg("failed to read config")
    }

    // Initialize SQLite store
    dbPath := viper.GetString("database.path")
    if dbPath == "" {
        dbPath = "./data/orchestrator.db"
    }
    s, err := store.NewStore(dbPath)
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to initialize store")
    }
    defer s.Close()

    logger.Info().Str("db", dbPath).Msg("server initialized")

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    <-ctx.Done()
    logger.Info().Msg("shutting down")
}
```

### Store Initialization with WAL
```go
// internal/store/store.go
// Source: modernc.org/sqlite documentation + PRD requirements
package store

import (
    "database/sql"
    "fmt"

    _ "modernc.org/sqlite"
)

type Store struct {
    db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    // Configure WAL mode
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        return nil, fmt.Errorf("set WAL mode: %w", err)
    }
    // Set busy timeout to 5 seconds
    if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
        return nil, fmt.Errorf("set busy_timeout: %w", err)
    }
    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
        return nil, fmt.Errorf("enable foreign keys: %w", err)
    }

    s := &Store{db: db}
    if err := s.migrate(); err != nil {
        return nil, fmt.Errorf("migrate: %w", err)
    }

    return s, nil
}

func (s *Store) Close() error {
    return s.db.Close()
}
```

### Database Schema Migration
```go
// internal/store/migrations/001_initial.go
package migrations

const initialSchema = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    dispatch_ref TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    loop_iteration_count INTEGER NOT NULL DEFAULT 0,
    transport TEXT NOT NULL,
    wave INTEGER NOT NULL,
    topo_rank INTEGER NOT NULL DEFAULT 0,
    workspace_path TEXT,
    artifact_path TEXT,
    last_error_reason TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    terminal_at TEXT,
    card_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    event_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    from_state TEXT,
    to_state TEXT,
    timestamp TEXT NOT NULL,
    reason TEXT,
    attempt INTEGER NOT NULL DEFAULT 0,
    transport TEXT,
    runner_id TEXT,
    details TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS waves (
    dispatch_ref TEXT NOT NULL,
    wave INTEGER NOT NULL,
    sealed_at TEXT,
    created_at TEXT NOT NULL,
    PRIMARY KEY (dispatch_ref, wave)
);

CREATE INDEX IF NOT EXISTS idx_tasks_dispatch_ref ON tasks(dispatch_ref);
CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state);
CREATE INDEX IF NOT EXISTS idx_tasks_wave ON tasks(wave);
CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
`

// Apply migrations (simplified for Phase 1)
func Migrate(db *sql.DB) error {
    if _, err := db.Exec(initialSchema); err != nil {
        return fmt.Errorf("exec initial schema: %w", err)
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| github.com/mattn/go-sqlite3 | modernc.org/sqlite | Now | Pure Go eliminates CGO dependency |
| Manual config parsing | spf13/viper | Long-standing | Standard library for configuration |
| log/std + custom formatting | rs/zerolog | v1.0 (2017) | Zero-allocation JSON logging |
| golang-migrate CLI | Embedded SQL in store | Optional | Phase 1 can use simple embedded migrations |
| In-memory state | SQLite persistence | Phase 1 | Enables crash recovery |

**Deprecated/outdated:**
- `github.com/mattn/go-sqlite3`: Not usable without CGO; switch to modernc.org/sqlite
- `github.com/mattn/go-sqlite3/libsqlite3`: macOS-only, no longer needed with pure Go driver

---

## Open Questions

1. **Config format:** YAML (recommended) or JSON?
   - What we know: Viper supports both; PRD mentions YAML in context
   - Recommendation: Use YAML for readability

2. **Migration strategy:** Simple embedded SQL or use golang-migrate?
   - What we know: Phase 1 schema is relatively simple
   - Recommendation: Embedded in store package for Phase 1; migrate to golang-migrate if >3 migrations

3. **Recovery behavior:** What to do with tasks in "running" state on restart?
   - What we know: PITFALLS.md suggests either resume or mark as retry_waiting
   - What's unclear: Which is safer? Resume risks duplicate execution
   - Recommendation: Mark as retry_waiting with reason "process_resume" for safety

4. **Wave recovery:** What if wave shows sealed but contains non-terminal tasks?
   - What's unclear: PRD does not specify
   - Recommendation: Unseal wave if inconsistency detected on startup

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Server runtime | YES | 1.26.1 | - |
| GCC/CC | CGO compilation | NO | - | Use pure Go driver (modernc.org/sqlite) |
| SQLite CLI | Debugging | ? | - | - |
| git | Worktree features (future) | YES | - | - |

**Missing dependencies with no fallback:**
- GCC/CC: No C compiler available - MUST use modernc.org/sqlite (pure Go)

**Missing dependencies with fallback:**
- SQLite CLI: Could install for debugging, but not required for server operation

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (testing package) + testify/assert |
| Config file | none |
| Quick run command | `go test ./internal/store/... -v` |
| Full suite command | `go test ./... -v` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|---------------|
| SETUP-01 | Server starts without error | Smoke | `go run cmd/server/main.go &` thencurl localhost:8080/health | No - create new |
| SETUP-02 | Tables created with correct schema | Unit | Verify tables exist via sqlite3 or Go query | No - create new |
| SETUP-03 | Recovery loads persisted state | Integration | Start server, kill, restart, verify state | No - create new |
| PERSISTENCE-01 | Task upsert persists card_json | Unit | `go test ./internal/store/... -run TestTaskUpsert` | No - create new |
| PERSISTENCE-02 | Events recorded with task updates | Integration | Transition task, verify both updated | No - create new |
| PERSISTENCE-03 | Wave CRUD operations | Unit | `go test ./internal/store/... -run TestWaveCRUD` | No - create new |
| PERSISTENCE-04 | Wave seal operation | Unit | Seal wave, verify sealed_at set | No - create new |
| PERSISTENCE-05 | Unique constraint enforced | Integration | Insert duplicate (dispatch_ref, wave), expect error | No - create new |

### Sampling Rate
- Per task commit: `go test -short ./...`
- Per wave merge: `go test ./... -v`
- Phase gate: Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/store/store_test.go` — tests for SETUP-02, PERSISTENCE-01, PERSISTENCE-03
- [ ] `internal/store/tasks_test.go` — tests for PERSISTENCE-01, PERSISTENCE-02
- [ ] `internal/store/waves_test.go` — tests for PERSISTENCE-03, PERSISTENCE-04, PERSISTENCE-05
- [ ] `internal/store/recovery_test.go` — test for SETUP-03
- [ ] `config.yaml` — minimal config file for testing
- [ ] Framework install: no additional install needed (Go testing is built-in)

---

## Sources

### Primary (HIGH confidence)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - Verified latest v1.48.1 from proxy.golang.org (2026-04-03)
- [spf13/viper](https://pkg.go.dev/github.com/spf13/viper) - Verified v1.21.0 (2025-09-08)
- [rs/zerolog](https://pkg.go.dev/github.com/rs/zerolog) - Verified v1.35.0 (2026-03-27)
- [Go project layout](https://github.com/golang-standards/project-layout) - Standard Go conventions
- [go-sqlite3 limitations](https://github.com/mattn/go-sqlite3) - Confirmed CGO requirement

### Secondary (MEDIUM confidence)
- [PITFALLS.md](./../../research/PITFALLS.md) - Local project-specific pitfalls research
- [CONTEXT.md](./01-CONTEXT.md) - Locked decisions from discuss phase

### Tertiary (LOW confidence)
- General Go + SQLite best practices (training data, not specifically verified)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all versions verified via pkg.go.dev, pure Go driver confirmed for non-CGO environment
- Architecture: HIGH - standard Go project layout, PRD schema precisely matches requirements
- Pitfalls: HIGH - PITFALLS.md provides detailed prevention strategies mapped to phases

**Research date:** 2026-04-06
**Valid until:** 2026-05-06 (30 days for stable stack; versions are current as of research date)