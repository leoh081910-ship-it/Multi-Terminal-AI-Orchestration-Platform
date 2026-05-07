# v3 Architecture — Universal Multi-Agent Orchestration Platform

## Overview

v3 extends the platform from three hardcoded AI agents (Claude/Gemini/Codex) to a universal orchestration system supporting any agent via a pluggable `Runner` interface. Agents can be CLI tools, HTTP APIs, or MCP protocol servers.

## Core Components

### Runner Interface

The `Runner` interface (`internal/runner/interface.go`) is the universal execution contract:

```go
type Runner interface {
    Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error)
    HealthCheck(ctx context.Context) error
    Cancel(ctx context.Context, taskID string) error
    GetCapabilities(ctx context.Context) (*CapabilityManifest, error)
    Type() RunnerType
    String() string
}
```

Three concrete implementations:

| Runner | Type Constant | Description |
|--------|--------------|-------------|
| `CLIRunner` | `"cli"` | Wraps `transport.CLITransport` for local CLI agents |
| `HTTPRunner` | `"http"` | Template-based HTTP API calls (OpenAI, Anthropic, etc.) |
| `MCPRunner` | `"mcp"` | JSON-RPC MCP protocol client |

### CapabilityManifest

Each Runner self-describes its capabilities via a 16-field manifest:

```go
type CapabilityManifest struct {
    Name              string
    TaskTypes         []string   // ["feature", "bugfix", "review"]
    MaxInputSize      int64
    OutputGlobs       []string
    ModelFamily       string     // "claude", "gpt", "local"
    ContextWindow     int
    SupportsThinking  bool
    SupportsMultimodal bool
    SupportsStreaming bool
    LatencyP50        int
    CostPer1KInput    float64
    CostPer1KOutput   float64
    Runtime           string     // "cli", "http", "mcp"
    Version           string
    Tags              []string
}
```

### AgentRegistry

`internal/registry/registry.go` provides runtime agent management:

- **Register/Unregister**: Dynamic agent lifecycle
- **Manifest caching**: Lazy + proactive capability refresh during heartbeat
- **Heartbeat monitoring**: Periodic `HealthCheck` with status tracking
- **Factory**: `BuildRunner(BuilderInput)` creates Runner instances from DB config

Agent statuses: `online`, `idle`, `running`, `offline`, `error`

### Dispatch Flow

```
Task arrives
    │
    ▼
dispatchCompatTask()
    │
    ├── Router selects agent (capability-based scoring)
    │       or fallback to owner_agent field
    │
    ▼
runCompatExecution() / RunCompatExecutionViaRunner()
    │
    ├── getRunnerForAgent(agentID)
    │       → Registry.Get(agentID)
    │
    ├── Runner.Execute(ctx, RunnerTask)
    │       → CLIRunner: transport.CLITransport
    │       → HTTPRunner: HTTP request with template
    │       → MCPRunner: JSON-RPC tools/call
    │
    ├── Record agent_call (async, non-blocking)
    │
    ├── Prometheus metrics (agent_requests_total, agent_duration_seconds)
    │
    └── Result → finishCompatExecutionSuccess/Failure
```

### Intelligent Routing

`internal/router/capability_matcher.go` performs 4-dimension scoring:

| Dimension | Weight | Criteria |
|-----------|--------|----------|
| Task Type | 50% | Manifest.TaskTypes matches requested type |
| Context Window | 20% | Sufficient context for task complexity |
| Features | 20% | Thinking, multimodal, streaming support |
| Cost/Latency | 10% | Lower cost and latency preferred |

### Observability

**Structured logging**: Every execution carries `trace_id` (from `execution_session_id`), propagated through both compat and Runner dispatch paths.

**Prometheus metrics** (namespace `ai_orchestrator`):

| Metric | Type | Labels |
|--------|------|--------|
| `agent_requests_total` | Counter | agent_id, runner_type, task_type, status |
| `agent_duration_seconds` | Histogram | agent_id, runner_type |
| `agent_heartbeat_timestamp` | Gauge | agent_id |
| `registry_agent_count` | Gauge | status |

**Agent call persistence**: `agent_calls` table records every execution with task_id, agent_id, runner_type, status, duration, error_message, output_summary.

### Database Schema

Key ent entities:

- **Agent** (`ent/schema/agent.go`): agent_id, name, runner_type, runner_config (JSON), status, org_id
- **AgentCall** (`ent/schema/agent_call.go`): id, task_id, agent_id, runner_type, task_type, trace_id, status, exit_code, error_message, output_summary, duration_ms, started_at, finished_at
- **Task**: existing task entity with new assigned_agent_id field

### Migration Tool

`cmd/migrate/main.go` migrates v2 `config.yaml` to v3 agent table:

```bash
go run cmd/migrate/main.go --config config.yaml --db ai-orchestration.db
# --dry-run: preview without writing
```

Reads claude/gemini/codex runtime blocks, creates CLI Runner entries in the agent table. Idempotent and creates backup of config.yaml.

## Zero-Breaking-Change Strategy

v3 maintains full backward compatibility:

1. `CLIRunner` wraps existing `transport.CLITransport` without modification
2. Compat dispatch path (`runCompatExecution`) falls back to v2 execution if no Runner is registered
3. `owner_agent` field continues to work; `"auto"` triggers intelligent routing
4. All existing API endpoints remain functional

## File Map

```
internal/
├── runner/
│   ├── interface.go          # Runner interface, RunnerTask, RunnerResult
│   ├── capability.go         # CapabilityManifest
│   ├── cli_runner.go         # CLIRunner implementation
│   ├── http_runner.go        # HTTPRunner implementation
│   └── mcp_runner.go         # MCPRunner implementation
├── registry/
│   ├── registry.go           # AgentRegistry + BuildRunner factory
│   └── heartbeat.go          # Heartbeat monitoring + capability refresh
├── router/
│   ├── router.go             # Intelligent task router
│   └── capability_matcher.go # 4-dimension scoring
├── server/
│   ├── compat_dispatch.go    # Compat dispatch with Runner bridge
│   ├── runner_dispatch.go    # Runner-native dispatch + agent call recording
│   └── org_api.go            # Agent management API endpoints
├── store/
│   └── repository.go         # AgentCall CRUD methods
└── telemetry/
    └── metrics.go            # Prometheus metrics
ent/schema/
├── agent.go                  # Agent entity
└── agent_call.go             # AgentCall entity
docs/examples/
├── http-runner-openai.json
├── http-runner-anthropic.json
└── http-runner-ollama.json
```
