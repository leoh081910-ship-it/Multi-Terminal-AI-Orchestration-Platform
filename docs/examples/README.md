# HTTPRunner Example Configurations

These JSON files are `runner_config` values for registering HTTP-based AI agents via the API.

## Quick Start

1. Set the required environment variable (for OpenAI/Anthropic examples)
2. Start the server: `go run cmd/server/main.go`
3. Create an agent via the API:

```bash
# OpenAI example
curl -X POST http://localhost:8080/api/v1/orgs/default/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4o Agent",
    "runner_type": "http",
    "runner_config": "<contents of http-runner-openai.json>"
  }'
```

4. The agent is immediately available for task routing and execution.

## Files

| File | Agent | API Key Env Var |
|------|-------|-----------------|
| `http-runner-openai.json` | OpenAI GPT-4o | `OPENAI_API_KEY` |
| `http-runner-anthropic.json` | Anthropic Claude | `ANTHROPIC_API_KEY` |
| `http-runner-ollama.json` | Ollama (local) | None needed |

## Template Variables

The `body_template` field uses Go `text/template` syntax. Available variables:

| Variable | Description |
|----------|-------------|
| `{{.ID}}` | Task ID |
| `{{.Type}}` | Task type (feature, bugfix, etc.) |
| `{{.Command}}` | Shell command string |
| `{{.Shell}}` | Shell name (bash, powershell) |
| `{{.Context.field}}` | Any field from the task context map |
| `{{.FilesToModify \| toJSON}}` | Files list serialized as JSON |
| `{{.Workspace.Path}}` | Workspace directory path |
| `{{index .Env "KEY"}}` | Environment variable by key |
| `{{.Timeout}}` | Task timeout duration |

## Output Extraction

The `output_path` field uses dot-notation to extract the agent's response from the JSON API response:

| Provider | Output Path |
|----------|-------------|
| OpenAI | `choices[0].message.content` |
| Anthropic | `content[0].text` |
| Ollama | `response` |

## MCP Runner Config

For MCP protocol agents, use `runner_type: "mcp"` with this config format:

```json
{
  "endpoint": "http://localhost:3000/mcp",
  "transport": "http",
  "tool_name": "execute_task",
  "tools_enabled": true
}
```

Phase 6 supports HTTP transport only; `transport: "sse"` is rejected at validation.
The `tool_name` field defaults to `execute_task` when omitted.

See `mcp-runner-local.json` for a complete example.
