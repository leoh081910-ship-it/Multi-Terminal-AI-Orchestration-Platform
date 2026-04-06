---
phase: 01-foundation
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - go.mod
  - go.sum
  - ent/schema/task.go
  - ent/schema/event.go
  - ent/schema/wave.go
  - ent/generate.go
autonomous: true
requirements:
  - PERS-01
  - PERS-02
  - PERS-03
  - PERS-04
  - PERS-06
must_haves:
  truths:
    - "Go module initialized with required dependencies"
    - "Ent ORM schemas defined for tasks, events, waves tables"
    - "Code generation produces ent generated code"
  artifacts:
    - path: "go.mod"
      provides: "Go module definition"
    - path: "ent/schema/task.go"
      provides: "Task entity schema per PERS-01"
    - path: "ent/schema/event.go"
      provides: "Event entity schema per PERS-02"
    - path: "ent/schema/wave.go"
      provides: "Wave entity schema per PERS-03"
    - path: "ent/generate.go"
      provides: "Ent code generation directive"
    - path: "ent/ent.go"
      provides: "Generated ent client code"
  key_links:
    - from: "ent/schema/task.go"
      to: "ent/schema/event.go"
      via: "Same ent generation"
    - from: "ent/generate.go"
      to: "ent/ent.go"
      via: "go generate"
---

<objective>
Initialize Go project with ent ORM schemas for tasks, events, and waves tables.
This establishes the foundation for persistence layer per PERS-01 through PERS-06.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
# From RESEARCH (key tech choices):
- Use `ent` v0.14.6 for ORM (not Prisma)
- Use `modernc.org/sqlite` for SQLite driver (pure Go, no CGO)
- Use `chi v5` for HTTP routing
- Use `zerolog` for logging
- Use `viper` for configuration

# From PRD requirements:
- PERS-01: tasks table fields (id, dispatch_ref, state, retry_count, loop_iteration_count, transport, wave, topo_rank, workspace_path, artifact_path, last_error_reason, created_at, updated_at, terminal_at, card_json)
- PERS-02: events table fields (event_id, task_id, event_type, from_state, to_state, timestamp, reason, attempt, transport, runner_id, details)
- PERS-03: waves table fields (dispatch_ref, wave, sealed_at, created_at)
- PERS-06: SQLite uses WAL mode + busy_timeout
</context>

<tasks>

<task type="auto">
  <name>Task 1: Initialize Go module with dependencies</name>
  <files>go.mod, go.sum</files>
  <action>
Initialize Go module and add required dependencies:
- ent v0.14.6 (ORM)
- modernc.org/sqlite (SQLite driver)
- chi/v5 (HTTP router)
- zerolog (logging)
- viper (configuration)
- github.com/google/uuid (UUID generation)

Create go.mod with module path and run go mod tidy.
  </action>
  <verify>
    <automated>go list -m all | grep -E "ent|sqlite|chi|zerolog|viper"</automated>
  </verify>
  <done>go.mod contains module with all 6 dependencies, go.sum populated</done>
</task>

<task type="auto">
  <name>Task 2: Create ent schemas for tasks, events, waves</name>
  <files>ent/schema/task.go, ent/schema/event.go, ent/schema/wave.go</files>
  <action>
Create ent schemas matching PRD requirements:

1. ent/schema/task.go - Task entity with fields:
   - ID (string, unique)
   - dispatch_ref (string, indexed)
   - state (string) - 13-state machine
   - retry_count (int)
   - loop_iteration_count (int)
   - transport (string)
   - wave (int)
   - topo_rank (int)
   - workspace_path (string)
   - artifact_path (string)
   - last_error_reason (string, nullable)
   - created_at (time)
   - updated_at (time)
   - terminal_at (time, nullable)
   - card_json (text) - stores full Task Card

2. ent/schema/event.go - Event entity:
   - event_id (string, unique)
   - task_id (string, indexed)
   - event_type (string)
   - from_state (string)
   - to_state (string)
   - timestamp (time)
   - reason (string, nullable)
   - attempt (int)
   - transport (string, nullable)
   - runner_id (string, nullable)
   - details (text, nullable)

3. ent/schema/wave.go - Wave entity:
   - dispatch_ref (string)
   - wave (int)
   - sealed_at (time, nullable)
   - created_at (time)
   - Add composite unique index on (dispatch_ref, wave)

Use ent.Field for each column, ent.Index for indexes.
  </action>
  <verify>
    <automated>ls ent/schema/*.go | wc -l | xargs -I {} test {} -eq 3 && echo "PASS" || echo "FAIL"</automated>
  </verify>
  <done>Three schema files exist with all required fields per PERS-01, PERS-02, PERS-03</done>
</task>

<task type="auto">
  <name>Task 3: Set up ent code generation</name>
  <files>ent/generate.go</files>
  <action>
Create ent/generate.go with:
```go
//go:build ignore

package main

import (
	"log"
	"entgo.io/ent/cmd/entc"
	"entgo.io/ent/cmd/entc/generate"
)

func main() {
	err := entc.Generate("./schema", &generate.Config{
		Target:   "./ent/generated",
	})
	if err != nil {
		log.Fatalf("running entc generate: %v", err)
	}
}
```

Then run `go generate ./ent/...` to generate ent client code.
The generated code will be in ent/generated/ or ent/ directory.
  </action>
  <verify>
<automated>ls ent/*.go 2>/dev/null | grep -q "ent.go" && echo "PASS" || echo "Need to run go generate"</automated>
  </verify>
  <done>Ent generated code exists (ent.go, entclient.go, etc.)</done>
</task>

</tasks>

<verification>
- [ ] go.mod shows ent v0.14.6, modernc.org/sqlite, chi/v5, zerolog, viper
- [ ] ent/schema/ contains task.go, event.go, wave.go
- [ ] Each schema has all fields from PERS-01, PERS-02, PERS-03
- [ ] go generate produces ent client code
</verification>

<success_criteria>
Go module with all dependencies and ent ORM schemas ready for repository implementation.
</success_criteria>

<output>
After completion, create `.planning/phases/01-foundation/{phase}-01-SUMMARY.md`
</output>