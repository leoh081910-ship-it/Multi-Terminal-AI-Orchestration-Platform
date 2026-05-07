# Migration Guide: v2 → v3

This guide covers migrating from the hardcoded 3-agent platform (v2) to the universal multi-agent platform (v3).

## What Changes

| Area | v2 | v3 |
|------|----|----|
| Agent support | Hardcoded Claude/Gemini/Codex | Dynamic, any number of agents |
| Execution | Direct transport dispatch | Runner interface (CLI/HTTP/MCP) |
| Routing | Fixed `owner_agent` field | Capability-based intelligent routing |
| Monitoring | Basic logging | Structured logging + Prometheus + call history |
| Configuration | `config.yaml` per project | `agent` table + API-driven |

## Automated Migration

The migration tool (`cmd/migrate/main.go`) handles the most common case:

```bash
# Preview changes
go run cmd/migrate/main.go --config config.yaml --db ai-orchestration.db --dry-run

# Execute migration
go run cmd/migrate/main.go --config config.yaml --db ai-orchestration.db
```

### What the tool does

1. Reads each project block from `config.yaml`
2. Extracts `claude`, `gemini`, `codex` runtime configurations
3. Creates entries in the `agent` table with `runner_type: "cli"` and appropriate `runner_config`
4. Backs up `config.yaml` as `config.yaml.v3-migration-bak`

The tool is idempotent — running it multiple times produces the same result.

### CLI flags

| Flag | Required | Description |
|------|----------|-------------|
| `--config` | Yes | Path to v2 config.yaml |
| `--db` | Yes | Path to SQLite database |
| `--dry-run` | No | Preview without writing to database |

## Manual Steps

### 1. Database migration

The `agent` and `agent_call` tables are created automatically by ent on first startup. No manual DDL required.

### 2. Agent registration

For agents not covered by the migration tool (custom agents):

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "agent_id": "my-custom-agent",
    "name": "My Custom Agent",
    "runner_type": "http",
    "runner_config": {
      "endpoint": "https://api.example.com/v1/execute",
      "auth_token": "Bearer ${API_KEY}",
      "output_path": "result.content"
    }
  }'
```

### 3. Update task payloads

Existing tasks continue to work without changes. To opt into intelligent routing:

```json
{
  "owner_agent": "auto",
  "type": "feature",
  "capabilities": ["can_execute"]
}
```

When `owner_agent` is `"auto"` or empty, the router selects the best agent based on capability matching.

### 4. Configure routing (optional)

If you want to customize routing weights:

```bash
curl -X PUT http://localhost:8080/api/routing/config \
  -H 'Content-Type: application/json' \
  -d '{
    "strategy": "capability",
    "weights": {
      "task_type": 0.5,
      "context_window": 0.2,
      "features": 0.2,
      "cost_latency": 0.1
    }
  }'
```

## Backward Compatibility

v3 maintains full backward compatibility:

- **`owner_agent: "Claude"`** still works — routes directly to the Claude CLI runner
- **Compat dispatch path** is preserved — tasks without a registered Runner fall back to v2 execution
- **`config.yaml`** is still read as a fallback for project configuration
- **All v2 API endpoints** remain functional unchanged

## Verification

After migration, verify:

1. **Agent status**: `GET /orgs/{orgID}/agents` — all agents should show `"status": "online"`
2. **Capabilities**: `GET /orgs/{orgID}/agents/{agentID}/capabilities` — manifest should be populated
3. **Execute a test task**: Create a simple task and verify it dispatches correctly
4. **Check agent calls**: `GET /api/tasks/{taskID}/agent-calls` — should show execution records
5. **Prometheus metrics**: `GET /metrics` — should include `ai_orchestrator_agent_requests_total`

## Rollback

Each v3 phase has a git tag (`v3-phase1` through `v3-phase6`). To rollback:

1. Stop the server
2. Checkout the desired tag: `git checkout v3-phase2`
3. Rebuild and restart

The `agent` table is additive — rollback does not require database changes. The v2 code ignores unknown tables.

## Common Issues

### Agent shows "offline" after registration

The heartbeat check failed. Verify:
- CLI agents: binary is in PATH and executable
- HTTP agents: endpoint is reachable from the server
- MCP agents: MCP server is running and accepting connections

### Task routes to wrong agent

Check the capability manifest and routing weights. Use the routing preview endpoint to debug:

```bash
curl -X POST http://localhost:8080/api/v1/orgs/{orgID}/routing/preview \
  -d '{"task_type": "feature"}'
```

### Migration tool fails on config.yaml

Ensure the config file follows the v2 format with `claude`, `gemini`, or `codex` blocks. The tool skips projects without recognized runtime configurations.
