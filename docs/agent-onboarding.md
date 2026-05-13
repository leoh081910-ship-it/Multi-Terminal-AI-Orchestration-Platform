# Agent Onboarding Guide

This guide covers registering HTTP and MCP agents via the org Agent API.

## Prerequisites

- Platform v3+ deployed
- Access to the platform API (default: `http://localhost:8080/api/v1`)
- Organization ID for scoping

## Register an HTTP Agent

HTTP agents call external AI APIs (OpenAI, Anthropic, Ollama, etc.) via templated HTTP requests.

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "OpenAI GPT-4o",
    "type": "http-agent",
    "runner_type": "http",
    "runner_config": "{\"endpoint\":\"https://api.openai.com/v1/chat/completions\",\"method\":\"POST\",\"auth_token\":\"Bearer ${OPENAI_API_KEY}\",\"body_template\":\"{\\\"model\\\":\\\"gpt-4o\\\",\\\"messages\\\":[{\\\"role\\\":\\\"user\\\",\\\"content\\\":{{.Context.prompt | toJSON}}}]}\"}\",\"output_path\":\"choices[0].message.content\",\"timeout_ms\":300000,\"model\":\"gpt-4o\"}"
  }'
```

Key points:
- `runner_config` is a **JSON string**, not a nested object.
- Validation rejects missing/invalid endpoints, unsupported methods, and malformed templates before the agent is persisted.
- No network calls are made during registration; unreachable endpoints are accepted.

### runner_config fields

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | Yes | Target HTTP URL (http or https) |
| `method` | No | HTTP method: GET, POST, PUT, PATCH (default: POST) |
| `headers` | No | Additional headers |
| `auth_token` | No | Bearer token or API key. Supports `${ENV_VAR}` expansion |
| `body_template` | No | Go `text/template` for request body |
| `output_path` | No | Dot-notation path to extract from response (e.g., `choices[0].message.content`) |
| `timeout_ms` | No | Timeout in milliseconds (default: 300000) |
| `model` | No | Model identifier for capability manifest |

### Template variables

The `body_template` uses Go `text/template` syntax with a `toJSON` filter:

| Variable | Type | Description |
|----------|------|-------------|
| `{{.ID}}` | string | Task ID |
| `{{.Type}}` | string | Task type |
| `{{.Command}}` | string | Shell command |
| `{{.Shell}}` | string | Shell name (bash, powershell) |
| `{{.Context.field}}` | any | Task context field |
| `{{.FilesToModify \| toJSON}}` | JSON | Files list serialized |
| `{{.Workspace.Path}}` | string | Workspace directory path |
| `{{index .Env "KEY"}}` | string | Environment variable |
| `{{.Timeout}}` | duration | Task timeout |

When `body_template` is omitted, the default payload is `{"task_id":..., "task_type":..., ...context fields}`.

### Example configs

See `docs/examples/` for complete configurations:
- `http-runner-openai.json` — OpenAI GPT-4o
- `http-runner-anthropic.json` — Anthropic Claude
- `http-runner-ollama.json` — Ollama local LLM

## Register an MCP Agent

MCP agents communicate via JSON-RPC 2.0 over HTTP.

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Local MCP Agent",
    "type": "mcp-agent",
    "runner_type": "mcp",
    "runner_config": "{\"endpoint\":\"http://localhost:3000/mcp\",\"transport\":\"http\",\"tool_name\":\"execute_task\",\"tools_enabled\":true}"
  }'
```

Key points:
- `runner_config` is a **JSON string**.
- `transport` must be `http`. Phase 6 rejects `sse` at validation.
- `tool_name` defaults to `execute_task` when omitted.
- The runner sends `tools/call` with `{"name": tool_name, "arguments": {task_id, task_type, ...context}}`.

### runner_config fields

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | Yes | MCP server URL (http or https) |
| `transport` | No | Must be `"http"` (Phase 6). Default: `"http"` |
| `tool_name` | No | Tool invoked via `tools/call` (default: `execute_task`) |
| `tools_enabled` | No | Enable MCP tool use |
| `auth_token` | No | Bearer token. Supports `${ENV_VAR}` |
| `timeout_ms` | No | Timeout in milliseconds (default: 300000) |

## Validation Errors

If `runner_config` is invalid, the API returns HTTP 400 before persisting:

```json
{"success": false, "error": "HTTP runner config validation failed: endpoint is required"}
```

The agent is **not created** in the database when validation fails. Update requests with invalid config also return 400 and preserve the existing agent state.

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

## Testing

All HTTP/MCP runner tests use local `httptest.NewServer` fake services — no external API calls, no network dependencies.

```bash
go test ./internal/runner ./internal/registry ./internal/server -count=1
```
