# Agent Onboarding Guide

This guide covers adding a new AI agent to the orchestration platform. Three runner types are supported: CLI, HTTP, and MCP.

## Prerequisites

- Platform v3+ deployed
- Access to the platform API (default: `http://localhost:8080/api/v1`)
- Organization ID for scoping

## Option 1: CLI Agent

Use when the agent runs as a local CLI tool (e.g., `claude`, `gemini`, `codex`).

### Registration

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "agent_id": "my-agent",
    "name": "My Custom Agent",
    "runner_type": "cli",
    "runner_config": {
      "base_path": "/tmp/worktrees",
      "main_repo": "/path/to/main/repo"
    }
  }'
```

### runner_config fields

| Field | Required | Description |
|-------|----------|-------------|
| `base_path` | Yes | Parent directory for git worktrees |
| `main_repo` | Yes | Path to the main git repository |

### How it works

CLIRunner wraps the existing transport layer. It creates a worktree under `base_path/{taskID}`, executes the agent CLI, and collects artifacts. The agent CLI must support standard task execution patterns.

### Capability detection

CLIRunner auto-detects capabilities by running `{cli} --version`. Supported CLIs:

- `claude` — Detects model family (opus/sonnet/haiku) and version
- `gemini` — Detects version
- `codex` — Detects version

For custom CLIs, the agent ID is used as the binary name.

## Option 2: HTTP Agent

Use when the agent exposes an HTTP API (OpenAI, Anthropic, Ollama, custom endpoints).

### Registration

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "agent_id": "openai-gpt4",
    "name": "OpenAI GPT-4o",
    "runner_type": "http",
    "runner_config": {
      "endpoint": "https://api.openai.com/v1/chat/completions",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "auth_token": "Bearer ${OPENAI_API_KEY}",
      "body_template": "{\"model\":\"gpt-4o\",\"messages\":[{\"role\":\"user\",\"content\":{{.Prompt | toJSON}}}]}",
      "output_path": "choices[0].message.content",
      "timeout_ms": 300000,
      "model": "gpt-4o"
    }
  }'
```

### runner_config fields

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | Yes | Target HTTP URL |
| `method` | No | HTTP method (default: `POST`) |
| `headers` | No | Additional headers (JSON object) |
| `auth_token` | No | Bearer token or API key. Supports `${ENV_VAR}` expansion |
| `body_template` | No | Go template for request body. Available: `.Prompt`, `.TaskID`, `.Command`, `.FilesToModify` |
| `output_path` | No | Dot-notation path to extract from JSON response (e.g., `choices[0].message.content`) |
| `timeout_ms` | No | Request timeout in milliseconds (default: 60000) |
| `model` | No | Model identifier for capability manifest |

### Template variables

The `body_template` uses Go `text/template` syntax:

| Variable | Type | Description |
|----------|------|-------------|
| `.Prompt` | string | Task prompt/command content |
| `.TaskID` | string | Task identifier |
| `.Command` | string | Shell command to execute |
| `.FilesToModify` | []string | Files the task should modify |
| `.Context` | map[string]any | Additional context key-value pairs |

Use `{{.Prompt | toJSON}}` for safe JSON string embedding.

### Example configs

See `docs/examples/` for complete configurations:
- `http-runner-openai.json` — OpenAI GPT-4o
- `http-runner-anthropic.json` — Anthropic Claude
- `http-runner-ollama.json` — Ollama local LLM

## Option 3: MCP Agent

Use when the agent implements the Model Context Protocol (JSON-RPC over HTTP/SSE).

### Registration

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "agent_id": "my-mcp-server",
    "name": "My MCP Server",
    "runner_type": "mcp",
    "runner_config": {
      "endpoint": "http://localhost:3001/mcp",
      "transport": "http",
      "timeout_ms": 120000,
      "tool_name": "execute_task"
    }
  }'
```

### runner_config fields

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | Yes | MCP server URL |
| `transport` | No | `"http"` (Streamable HTTP) or `"sse"` (Server-Sent Events). Default: `"http"` |
| `timeout_ms` | No | Request timeout in milliseconds (default: 60000) |
| `tool_name` | No | Tool name for execution calls (default: `"execute_task"`) |

### How it works

MCPRunner performs:
1. `initialize` handshake on first connection
2. `tools/list` to discover available tools
3. `tools/call` with the configured `tool_name` for task execution

## Post-Registration

After registering an agent, verify it's online:

```bash
# Check agent status
curl http://localhost:8080/api/v1/orgs/{orgID}/agents/{agentID}

# Trigger heartbeat
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents/{agentID}/heartbeat

# View capabilities
curl http://localhost:8080/api/v1/orgs/{orgID}/agents/{agentID}/capabilities
```

## Routing

Agents are automatically routed to tasks based on their CapabilityManifest. To explicitly assign:

- Set `owner_agent` in the task payload to the agent ID
- Set `owner_agent: "auto"` for intelligent routing

## Monitoring

- Agent call history: `GET /api/tasks/{taskID}/agent-calls`
- Prometheus metrics: `agent_requests_total`, `agent_duration_seconds`
- Structured logs with `trace_id` throughout execution chain
