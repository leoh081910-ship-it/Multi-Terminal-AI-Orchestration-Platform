# Multi-Terminal AI Orchestration Platform

A unified orchestration layer that schedules and coordinates multiple AI coding agents (Claude Code, Codex CLI, Gemini CLI) across projects with parallel wave execution, failure recovery, and real-time observability.

## Problem

AI coding assistants operate as isolated terminals — one project might be worked on by Claude, Gemini, and Codex simultaneously, but each runs in a separate process with no shared context, task routing, or conflict resolution. Developers end up manually coordinating agents, duplicating context, and losing visibility into who did what.

## Solution

This platform sits **between your project and your AI agents** — it accepts work items via API, decomposes them into execution waves with dependency awareness, dispatches tasks to the right agent runtime in parallel, monitors outcomes, retries failures, and surfaces everything on a real-time dashboard.

```
                    ┌──────────────────────────────┐
                    │     Web Dashboard (:8080)     │
                    │   Wave board · Task timeline  │
                    └──────────────┬───────────────┘
                                   │
                    ┌──────────────▼───────────────┐
                    │      Orchestration Engine     │
                    │  ┌─────────┐  ┌───────────┐  │
                    │  │ Auto    │  │ Failure   │  │
                    │  │Dispatch │  │Orchestr.  │  │
                    │  └─────────┘  └───────────┘  │
                    │  ┌─────────┐  ┌───────────┐  │
                    │  │ Retry   │  │ Merge     │  │
                    │  │ Worker  │  │Queue      │  │
                    │  └─────────┘  └───────────┘  │
                    └──────┬──────────┬────────────┘
                           │          │
              ┌────────────▼──┐  ┌────▼─────────────┐
              │ Connector     │  │ Reverse Executor  │
              │ Claude·Codex  │  │ output→input loop │
              │ ·Gemini       │  │                   │
              └───────┬───────┘  └───────────────────┘
                      │
           ┌──────────┼──────────┐
           ▼          ▼          ▼
        Claude     Codex      Gemini
        (pwsh)     (pwsh)     (pwsh)
```

## Key Capabilities

### Wave-Based Parallel Scheduling

Tasks are organized into **waves** based on dependency graphs. All tasks within a wave execute concurrently across available AI runtimes. Waves are sequential — downstream waves only start after upstream waves complete.

### Long-Chain Task Execution

Each task moves through a full lifecycle:

```
Plan → Research → Implement → Review → Merge
```

A **Coordinator-Reviewer** dual-agent model enforces quality gates — the coordinator produces work, and an independent reviewer validates it before merging.

### Failure Orchestrator

Failures are automatically **classified** (compile error, test failure, dependency conflict, timeout), matched against **policies** (retry, escalate, skip), and routed through a retry worker that can re-dispatch with adjusted parameters.

### Reverse Executor

CLI-based AI agents don't natively speak to each other. The **reverse executor** captures output artifacts from one agent, transforms them into input context for another, enabling cross-agent feedback loops without manual intervention.

### Multi-AI Runtime Connectors

A unified `Transport` abstraction layer handles process lifecycle, stdin/stdout protocol, and environment injection for Claude Code, Codex CLI, and Gemini CLI — each with its own shell invocation strategy.

### Project Registry

Multiple projects can be managed simultaneously. Each project gets:
- Isolated git worktrees for safe concurrent execution
- Independent workspace and artifact directories
- Per-project agent runtime configuration
- Queue browsing at `/api/v1/projects/{id}/...`

### Real-Time Dashboard

React + TypeScript frontend with WebSocket-driven live updates:
- **Orchestrator Home** — project overview, active waves, system health
- **Scheduler Board** — task timeline, wave dependencies, failure drill-down
- Single-process deployment (Go serves the built frontend directly)

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.24, Chi router, Zerolog |
| **ORM** | Ent (code-generated type-safe ORM) |
| **Database** | SQLite (modernc.org/sqlite) |
| **Frontend** | React 19, TypeScript, Vite |
| **Realtime** | WebSocket (Gorilla) |
| **Config** | YAML (Viper) |
| **Agent Runtimes** | PowerShell-hosted Claude Code, Codex CLI, Gemini CLI |

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+ (for frontend only)
- One or more AI CLI tools installed (Claude Code, Codex CLI, Gemini CLI)

### Single-Process Server

```powershell
go run ./cmd/server -config config.yaml
```

The server listens on `http://127.0.0.1:8080`. The dashboard is served at `/board` directly from the Go process — no separate frontend process needed.

### Frontend Dev Mode

```powershell
cd web
npm install
npm run dev        # http://127.0.0.1:5173
```

### Build Frontend

```powershell
cd web
npm run build      # output → web/dist (served by Go)
```

## Project Structure

```
.
├── cmd/server/              # Single binary entrypoint
├── internal/
│   ├── api/                 # HTTP handlers (Chi router)
│   ├── connector/           # Multi-AI runtime adapters (GSD protocol)
│   ├── engine/              # Wave scheduler, dependency engine, retry engine
│   ├── executor/            # Coordinator-Reviewer agent orchestrator
│   ├── mergequeue/          # Artifact merge queue with conflict detection
│   ├── reverse/             # Output→input reverse executor
│   ├── server/              # Server composition (dispatch, failure, recovery)
│   ├── store/               # Ent-backed data repository
│   └── transport/           # CLI and API transport abstractions
├── ent/                     # Ent ORM schemas & generated code (Task, Wave, Event)
├── web/                     # React + TypeScript frontend
│   ├── src/pages/           # OrchestratorHome, SchedulerBoard
│   ├── src/api/             # API + WebSocket clients
│   └── src/components/      # Layout, shared UI
├── docs/                    # PRDs, plans, API contracts
├── scripts/runtime/         # Per-agent PowerShell launchers
├── config.yaml              # Platform configuration
└── start.ps1                # Startup script
```

## Configuration

`config.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080

auto_dispatch:
  enabled: true
  interval_ms: 2000

projects:
  items:
    - id: my-project
      name: My Project
      repo_root: E:/path/to/repo
      claude:
        command: '& ''path/to/claude.ps1'''
        shell: powershell
      gemini:
        command: '& ''path/to/gemini.ps1'''
        shell: powershell
      codex:
        command: '& ''path/to/codex.ps1'''
        shell: powershell
```

Each project declares its own agent runtime commands. The platform manages git isolation (worktrees), workspace directories, and artifact storage per project automatically.

## API

| Endpoint | Description |
|----------|------------|
| `GET /api/v1/health` | System health check |
| `GET /api/v1/projects` | List registered projects |
| `POST /api/v1/projects/{id}/tasks` | Create a task for execution |
| `GET /api/v1/projects/{id}/tasks` | List tasks with filters |
| `GET /api/v1/projects/{id}/waves` | Current wave execution state |
| `GET /api/v1/projects/{id}/queue` | Pending queue status |
| `WS /api/v1/ws` | Real-time event stream |

## Documentation

- [API Contract](docs/project-api-contract.md)
- [Coordinator-Reviewer PRD](docs/prd/PRD-DA-001-dynamic-coordinator-reviewer.md)
- [Coordinator-Reviewer Plan](docs/plans/PLAN-DA-001-dynamic-coordinator-reviewer.md)
- [Runtime Reliability PRD](docs/prd/PRD-OPS-001-platform-runtime-reliability.md)
- [Runtime Reliability Plan](docs/plans/PLAN-OPS-001-platform-runtime-reliability.md)
- [PRD Registry](docs/PRD_REGISTRY.md)

## License

MIT
