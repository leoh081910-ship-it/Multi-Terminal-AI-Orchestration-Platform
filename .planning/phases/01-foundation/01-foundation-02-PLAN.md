---
phase: 01-foundation
plan: "02"
type: execute
wave: 2
depends_on:
  - 01-foundation-01
files_modified:
  - internal/store/repository.go
  - internal/server/server.go
  - cmd/server/main.go
autonomous: true
requirements:
  - PERS-05
  - PERS-06
  - API-01
must_haves:
  truths:
    - "SQLite repository with transaction support for all CRUD operations"
    - "HTTP server with chi router exposes REST API"
    - "Health check endpoint with proper retry logic"
  artifacts:
    - path: "internal/store/repository.go"
      provides: "Repository with transaction support per PERS-05"
      methods:
        - TaskCreate
        - TaskUpdate
        - TaskGetByID
        - TaskListByDispatchRef
        - EventCreate
        - WaveUpsert
        - WaveGet
    - path: "internal/server/server.go"
      provides: "HTTP server with chi router"
      routes:
        - GET /health (with retry)
        - GET /api/tasks
        - POST /api/tasks
        - GET /api/tasks/:id
        - PUT /api/tasks/:id
        - GET /api/dispatches/:dispatch_ref/tasks
        - POST /api/dispatches/:dispatch_ref/waves
    - path: "cmd/server/main.go"
      provides: "Server entry point with SQLite WAL mode config"
  key_links:
    - from: "internal/server/server.go"
      to: "internal/store/repository.go"
      via: "repository passed as dependency"
    - from: "cmd/server/main.go"
      to: "internal/server/server.go"
      via: "server.New() constructor"
---

<objective>
Implement SQLite repository layer and HTTP server with chi router.
Establishes persistence and API layer for the platform.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
# From Plan 01-foundation-01:
- Ent schemas for tasks, events, waves exist
- Dependencies: ent, modernc.org/sqlite, chi/v5, zerolog, viper

# Key requirements:
- PERS-05: Event and task state updates in same SQLite transaction
- PERS-06: SQLite uses WAL mode + busy_timeout (5000ms)
- API-01: RESTful API for task CRUD, wave operations, state query
</context>

<interfaces>
<!-- From ent generated code (will exist after plan 01 runs) -->
```go
// ent/client.go - Expected exports
type Client struct {
  Task *TaskClient
  Event *EventClient
  Wave *WaveClient
}
func NewClient(opts ...Option) (*Client, error)
func (c *Client) Tx(ctx context.Context) (*Tx, error)

// ent/task.go - Expected methods
func (c *TaskClient) Create() *TaskCreate
func (c *TaskClient) Get(ctx context.Context, id string) (*Task, error)
func (c *TaskClient) Query() *TaskQuery
type Task struct {
  ID string
  DispatchRef string
  State string
  // ... all PERS-01 fields
}
```
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Implement SQLite repository with transaction support</name>
  <files>internal/store/repository.go</files>
  <action>
Create internal/store/repository.go implementing:

1. Repository struct wrapping ent.Client
2. NewRepository(client *ent.Client) *Repository
3. WithTx() method for transaction context
4. Task methods:
   - CreateTask(ctx, *TaskCard) (string, error) - generates UUID, sets initial state "queued"
   - UpdateTaskState(ctx, taskID, fromState, toState, reason) error - wraps in tx per PERS-05
   - GetTaskByID(ctx, taskID) (*Task, error)
   - ListTasksByDispatchRef(ctx, dispatchRef) ([]*Task, error)
   - UpdateTask(ctx, taskID, updates) error
5. Event methods:
   - CreateEvent(ctx, event) error - called within same tx as state update
6. Wave methods:
   - UpsertWave(ctx, dispatchRef, wave) error - creates or updates
   - GetWave(ctx, dispatchRef, wave) (*Wave, error)
   - SealWave(ctx, dispatchRef, wave) error

Key: Event writes and task state updates MUST be in same SQLite transaction (PERS-05).
  </action>
  <verify>
<automated>go build -o /dev/null ./internal/store/...</automated>
  </verify>
  <done>Repository implements all CRUD methods with transaction support per PERS-05</done>
</task>

<task type="auto">
  <name>Task 2: Build HTTP server with chi router</name>
  <files>internal/server/server.go</files>
  <action>
Create internal/server/server.go:

1. Server struct with repository and logger
2. New(repo *store.Repository, logger *zerolog.Logger) *Server
3. Routes setup with chi:
   - GET /health - returns {"status": "ok"} (handles health check)
   - GET /api/tasks - list all tasks
   - POST /api/tasks - create task (accepts JSON TaskCard)
   - GET /api/tasks/:id - get task by ID
   - PUT /api/tasks/:id - update task
   - GET /api/dispatches/:dispatch_ref/tasks - list tasks by dispatch
   - POST /api/dispatches/:dispatch_ref/waves - create/upsert wave
   - GET /api/dispatches/:dispatch_ref/waves/:wave - get wave status

4. Each handler parses request, calls repository, returns JSON response
5. Use zerolog for request logging (middleware)
6. Return proper HTTP status codes (200, 201, 400, 404, 500)
  </action>
  <verify>
<automated>go build -o /dev/null ./internal/server/...</automated>
  </verify>
  <done>HTTP server with chi router exposes all API-01 endpoints</done>
</task>

<task type="auto">
  <name>Task 3: Create server entry point with SQLite WAL mode</name>
  <files>cmd/server/main.go</files>
  <action>
Create cmd/server/main.go:

1. Load config via viper (config.yaml or env vars):
   - database.path (SQLite file path)
   - server.host
   - server.port
2. Open SQLite database with modernc.org/sqlite driver
3. Configure SQLite with WAL mode and busy_timeout:
   ```go
   db, err := sqlite.Open("file:" + cfg.Database.Path + "?_journal_mode=WAL&_busy_timeout=5000")
   ```
4. Run ent generated code to create tables if not exist: client.Schema.Create(ctx)
5. Initialize repository: store.NewRepository(client)
6. Initialize server: server.New(repo, logger)
7. Start HTTP server on configured host:port
8. Handle graceful shutdown (SIGINT/SIGTERM)
  </action>
  <verify>
<automated>go build -o /tmp/ai编排-platform-server ./cmd/server/...</automated>
  </verify>
  <done>Server binary compiles, config loads, SQLite opens with WAL mode</done>
</task>

</tasks>

<verification>
- [ ] Repository has transaction support (PERS-05)
- [ ] SQLite uses WAL mode + 5000ms busy_timeout (PERS-06)
- [ ] All API-01 endpoints implemented
- [ ] Server compiles without error
</verification>

<success_criteria>
Platform foundation complete: persistence layer with transaction support and REST API server running.
</success_criteria>

<output>
After completion, create `.planning/phases/01-foundation/{phase}-02-SUMMARY.md`
</output>