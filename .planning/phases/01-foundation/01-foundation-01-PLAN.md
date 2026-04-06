---
phase: 01-foundation
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: []
autonomous: true
requirements:
  - SETUP-01
  - SETUP-02
  - SETUP-03
  - PERSISTENCE-01
  - PERSISTENCE-02
  - PERSISTENCE-03
  - PERSISTENCE-04
  - PERSISTENCE-05
must_haves:
  truths:
    - "Server can start and listen on port 8080"
    - "CLI can execute commands and connect to database"
    - "Task state persists across process restart"
    - "Tasks table stores all required fields"
    - "Events table stores all required fields"
    - "Waves table stores all required fields"
    - "card_json stores complete Task Card"
    - "State transitions and events written in same atomic transaction"
  artifacts:
    - path: "cmd/server/main.go"
      provides: "HTTP API server entrypoint"
    - path: "cmd/cli/main.go"
      provides: "CLI entrypoint"
    - path: "internal/store/store.go"
      provides: "SQLite persistence layer"
    - path: "internal/store/models.go"
      provides: "Go types matching PRD schema"
    - path: "internal/store/testutil_test.go"
      provides: "Test helpers for in-memory DB"
  key_links:
    - from: "cmd/server/main.go"
      to: "internal/store/store.go"
      pattern: "NewStore.*data/orchestrator\\.db"
    - from: "cmd/cli/main.go"
      to: "internal/store/store.go"
      pattern: "NewStore.*data/orchestrator\\.db"
    - from: "internal/store/store.go"
      to: "SQLite database"
      pattern: "journal_mode=WAL"
---

<objective>
Initialize Go project with Standard Layout and implement SQLite persistence layer with three-table schema. Create test infrastructure and verify both server/CLI entrypoints can start successfully.

Purpose: Establish foundation for all subsequent phases - store layer is the single source of truth for task state
Output: Runnable Go project with operational SQLite persistence
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@.planning/ROADMAP.md
@.planning/phases/01-foundation/01-CONTEXT.md
@.planning/phases/01-foundation/01-RESEARCH.md
@.planning/research/STACK.md
@.planning/research/PITFALLS.md

# PRD Schema Requirements (from 01-CONTEXT.md canonical refs)
# tasks: id, dispatch_ref, state, retry_count, loop_iteration_count, transport, wave, topo_rank, workspace_path, artifact_path, last_error_reason, created_at, updated_at, terminal_at, card_json
# events: event_id, task_id, event_type, from_state, to_state, timestamp, reason, attempt, transport, runner_id, details
# waves: dispatch_ref, wave, sealed_at, created_at with unique (dispatch_ref, wave)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Initialize Go project structure</name>
  <files>go.mod, cmd/server/main.go, cmd/cli/main.go, internal/store/store.go, internal/store/models.go</files>
  <read_first>
    - .planning/phases/01-foundation/01-CONTEXT.md (D-01, D-02, D-03 decisions)
  </read_first>
  <action>
    1. Create go.mod with module name "ai-orchestrator" (adjust per D-05)
    2. Create directory structure:
       - cmd/server/main.go
       - cmd/cli/main.go
       - internal/store/store.go
       - internal/store/models.go
    3. Create minimal main.go in cmd/server that:
       - Creates SQLite DB at ./data/orchestrator.db
       - Opens connection, runs migrations, creates tables
       - Starts HTTP server on port 8080
       - Logs "Server started on :8080"
       - Handles SIGINT/SIGTERM graceful shutdown
       - Adds /health endpoint returning 200 OK
    4. Create minimal main.go in cmd/cli that:
       - Opens SQLite DB at ./data/orchestrator.db
       - Provides subcommands: task, wave, server
       - Logs "CLI ready"
    5. Run `go mod tidy` to fetch dependencies (mattn/go-sqlite3)
  </action>
  <verify>go build ./cmd/...</verify>
  <done>Project compiles with both server and CLI entrypoints</done>
</task>

<task type="auto">
  <name>Task 2: Define Go models matching SQLite schema</name>
  <files>internal/store/models.go</files>
  <read_first>
    - 多终端 AI 编排平台 PRD.md (Section 3 - SQLite persistence schema)
  </read_first>
  <action>
    Create internal/store/models.go with:

    1. TaskState string type with constants:
       - TaskStateQueued = "queued"


TaskStateRouted = "routed"


TaskStateWorkspacePrepared = "workspace_prepared"


TaskStateRunning = "running"


TaskStatePatchReady = "patch_ready"


TaskStateVerified = "verified"


TaskStateMerged = "merged"


TaskStateDone = "done"


TaskStateRetryWaiting = "retry_waiting"


TaskStateVerifyFailed = "verify_failed"


TaskStateApplyFailed = "apply_failed"


TaskStateFailed = "failed"

    2. Task struct (per PERSISTENCE-01):
       ```go
       type Task struct {
           ID                    string
           DispatchRef           string
           State                 string
           RetryCount            int
           LoopIterationCount    int
           Transport             string
           Wave                  int
           TopoRank              int
           WorkspacePath         string
           ArtifactPath          string
           LastErrorReason       string
           CreatedAt             time.Time
           UpdatedAt             time.Time
           TerminalAt            *time.Time
           CardJSON              string  // TEXT NOT NULL per PRD
       }
       ```

    3. TaskEvent struct (per PERSISTENCE-02):
       ```go
       type TaskEvent struct {
           EventID      string
           TaskID       string
           EventType    string
           FromState    string
           ToState      string
           Timestamp    time.Time
           Reason       string
           Attempt      int
           Transport    string
           RunnerID     string
           Details      string
       }
       ```

    4. Wave struct (per PERSISTENCE-03):
       ```go
       type Wave struct {
           DispatchRef string
           Wave        int
           SealedAt    *time.Time
           CreatedAt   time.Time
       }
       ```

    5. TaskCard struct with JSON tags for unmarshaling

    6. Repository and table constants for SQL queries
  </action>
  <verify>go build ./internal/store/...</verify>
  <done>Go types match PRD schema exactly</done>
</task>

<task type="auto">
  <name>Task 3: Implement SQLite store with migrations and WAL mode</name>
  <files>internal/store/store.go</files>
  <read_first>
    - internal/store/models.go (just created in Task 2)
    - .planning/research/PITFALLS.md (Critical Pitfall 1: SQLite concurrency)
  </read_first>
  <action>
    Create internal/store/store.go with:

    1. Store struct with sql.DB

    2. NewStore(path string) (*Store, error) that:
       - Creates directory if not exists
       - Opens SQLite database using github.com/mattn/go-sqlite3
       - Executes PRAGMA journal_mode=WAL (per D-05 and PERSISTENCE-06)
       - Executes PRAGMA busy_timeout=5000 (per PITFALLS research)
       - Runs migrations (create tables if not exist)

    3. Migration SQL:
       ```sql
       CREATE TABLE IF NOT EXISTS tasks (
           id TEXT PRIMARY KEY,
           dispatch_ref TEXT NOT NULL,
           state TEXT NOT NULL,
           retry_count INTEGER DEFAULT 0,
           loop_iteration_count INTEGER DEFAULT 0,
           transport TEXT,
           wave INTEGER NOT NULL,
           topo_rank INTEGER DEFAULT 0,
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
           attempt INTEGER,
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
       ```

    4. Indexes:
       - CREATE INDEX IF NOT EXISTS idx_tasks_dispatch_ref ON tasks(dispatch_ref)
       - CREATE INDEX IF NOT EXISTS idx_events_task_id ON events(task_id)
       - CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state)

    5. Close() method to clean up DB connection
  </action>
  <verify>go build ./internal/store/...</verify>
  <done>Database opens with correct configuration, tables created</done>
</task>

<task type="auto">
  <name>Task 4: Implement CRUD with atomic transactions</name>
  <files>internal/store/store.go</files>
  <read_first>
    - internal/store/store.go (created in Task 3)
  </read_first>
  <action>
    Extend internal/store/store.go with CRUD operations:

    1. CreateTask(ctx, task *Task) error:
       - INSERT INTO tasks
       - UPSERT INTO waves (dispatch_ref, wave) (per WAVE-01)
       - Use transaction for both

    2. GetTask(ctx, id string) (*Task, error)

    3. UpdateTaskState(ctx, id, state, reason string) error

    4. ListTasksByDispatchRef(ctx, dispatchRef string) ([]Task, error)

    5. ListTasksByWave(ctx, dispatchRef string, wave int) ([]Task, error)

    6. ListTasksByState(ctx, state string) ([]Task, error)

    7. CreateEvent(ctx, event *TaskEvent) error

    8. ListEventsByTaskID(ctx, taskID string) ([]TaskEvent, error)

    9. CreateWave(ctx, wave *Wave) error

    10. GetWave(ctx, dispatchRef string, wave int) (*Wave, error)

    11. SealWave(ctx, dispatchRef string, wave int) error

    12. ListWaves(ctx, dispatchRef string) ([]Wave, error)

    13. AtomicTaskStateTransition - critical per PERSISTENCE-05:
       - BEGIN TRANSACTION
       - UPDATE tasks SET state=?, updated_at=?, last_error_reason=? WHERE id=? AND state=?
       - INSERT INTO events
       - COMMIT
  </action>
  <verify>go test ./internal/store/... -count=1 -short</verify>
  <done>All CRUD operations functional with atomic transactions</done>
</task>

<task type="auto">
  <name>Task 5: Create test infrastructure</name>
  <files>internal/store/testutil_test.go, internal/store/store_test.go</files>
  <read_first>
    - internal/store/store.go (CRUD operations from Task 4)
  </read_first>
  <action>
    Create test infrastructure:

    1. testutil_test.go:
       - NewInMemoryStore() *Store using ":memory:" SQLite
       - Automatic migrations on in-memory DB
       - Test helper functions

    2. store_test.go with tests:
       - TestTasksTable - verify Task CRUD
       - TestEventsTable - verify Event CRUD
       - TestWavesTable - verify Wave CRUD
       - TestCardJSON - verify JSON round-trip
       - TestAtomicity - verify transactions

    Quick: go test ./internal/store/... -count=1 -short
    Full: go test ./... -count=1 -v
  </action>
  <verify>go test ./internal/store/... -count=1 -short</verify>
  <done>Test infrastructure provides quick verification feedback</done>
</task>

<task type="auto">
  <name>Task 6: Verify server startup and restart recovery</name>
  <files>cmd/server/main.go</files>
  <read_first>
    - cmd/server/main.go (created in Task 1)
  </read_first>
  <action>
    Test server and recovery:

    1. Start server, verify:
       - Binary compiles: go build -o /tmp/aiserver ./cmd/server
       - Process starts and logs "Server started on :8080"
       - curl localhost:8080/health returns 200
       - Database file exists at ./data/orchestrator.db

    2. Verify restart recovery (SETUP-03):
       - Insert test task into database
       - Kill server (simulate crash: kill PID)
       - Start server again
       - Query task from DB - verify state persisted (PERSISTENCE-04)
       - Verify card_json can be deserialized after restart
  </action>
  <verify>go build -o /tmp/aiserver ./cmd/server && /tmp/aiserver & sleep 2 && curl localhost:8080/health</verify>
  <done>SETUP-01: Server starts on port 8080, SETUP-03: Process restart recovers state</done>
</task>

<task type="auto">
  <name>Task 7: Verify CLI execution</name>
  <files>cmd/cli/main.go</files>
  <read_first>
    - cmd/cli/main.go (created in Task 1)
  </read_first>
  <action>
    Test CLI:

    1. Build CLI: go build -o /tmp/aicli ./cmd/cli
       - Verify binary builds without error

    2. Test commands:
       - /tmp/aicli task list (should list tasks from DB)
       - /tmp/aicli wave list (should list waves from DB)

    3. Verify:
       - CLI connects to same DB as server (./data/orchestrator.db)
       - Commands execute successfully
  </action>
  <verify>go build -o /tmp/aicli ./cmd/cli</verify>
  <done>SETUP-02: CLI commands execute and connect to database</done>
</task>

</tasks>

<verification>
- All tasks complete without error
- go test ./internal/store/... -count=1 -short passes
- go test ./... -count=1 -v passes (full suite)
- Server starts and accepts HTTP requests
- CLI commands execute correctly
- State persists across process restart
</verification>

<success_criteria>
Phase 1 foundation complete when:
- [x] Go project compiles with both server and CLI entrypoints
- [x] SQLite with WAL mode enabled, busy_timeout configured
- [x] Three tables created: tasks, events, waves (per PERSISTENCE-01, PERSISTENCE-02, PERSISTENCE-03)
- [x] card_json stored as TEXT NOT NULL (per PERSISTENCE-04)
- [x] Atomic transaction for task+event writes (per PERSISTENCE-05)
- [x] SETUP-01: Server starts on port 8080
- [x] SETUP-02: CLI executes task/wave commands
- [x] SETUP-03: Process restart recovers task state
</success_criteria>

<output>
After completion, create .planning/phases/01-foundation/01-foundation-01-SUMMARY.md
</output>