# API Reference — v3

Base URL: `http://localhost:8080/api/v1`

## Agent Management

All agent endpoints are scoped under an organization.

### List Agents

```
GET /orgs/{orgID}/agents
```

Returns all registered agents for the organization.

**Response**: `{ "success": true, "data": [{ "agent_id": "...", "name": "...", "runner_type": "...", "status": "..." }] }`

### Create Agent

```
POST /orgs/{orgID}/agents
```

Register a new agent. The runner is instantiated immediately and a heartbeat is scheduled.

**Body**:
```json
{
  "agent_id": "my-agent",
  "name": "My Agent",
  "runner_type": "cli|http|mcp",
  "runner_config": { ... },
  "capabilities": ["can_execute"],
  "specialties": ["code"]
}
```

**Response**: `{ "success": true, "data": { "agent_id": "my-agent", "status": "online" } }`

### Get Agent

```
GET /orgs/{orgID}/agents/{agentID}
```

Returns agent details including current status and configuration.

### Update Agent

```
PATCH /orgs/{orgID}/agents/{agentID}
```

Update agent configuration. Changes take effect immediately (runner is re-instantiated).

**Body**: Partial update — only include fields to change.

### Delete Agent

```
DELETE /orgs/{orgID}/agents/{agentID}
```

Deregister an agent. Removes from the registry and stops heartbeat monitoring.

### Agent Heartbeat

```
POST /orgs/{orgID}/agents/{agentID}/heartbeat
```

Trigger an immediate heartbeat check. Returns current health status.

### Agent Capabilities

```
GET /orgs/{orgID}/agents/{agentID}/capabilities
```

Returns the agent's CapabilityManifest, refreshed from the live runner.

**Response**: `{ "success": true, "data": { "name": "...", "task_types": [...], "model_family": "...", ... } }`

## Agent Call History

### Get Task Agent Calls

```
GET /api/tasks/{taskID}/agent-calls
```

Returns all agent execution records for a task, ordered by `started_at` descending.

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "call-uuid",
      "task_id": "task-123",
      "agent_id": "Claude",
      "runner_type": "cli",
      "task_type": "feature",
      "trace_id": "session-abc",
      "status": "success",
      "exit_code": 0,
      "error_message": "",
      "output_summary": "Implemented feature X...",
      "duration_ms": 45000,
      "started_at": "2026-05-07T10:00:00Z",
      "finished_at": "2026-05-07T10:00:45Z"
    }
  ]
}
```

## Routing

### Preview Route

```
POST /orgs/{orgID}/routing/preview
```

Preview which agent would be selected for a task without creating it.

**Body**:
```json
{
  "task_type": "feature",
  "capabilities": ["can_execute"],
  "assigned_role_id": "executor"
}
```

### Get Routing Config

```
GET /routing/config
```

Returns the current routing configuration including strategy and weights.

### Update Routing Config

```
PUT /routing/config
```

Update routing strategy and capability matcher weights.

## Task Endpoints (existing)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/tasks` | List all tasks |
| GET | `/api/tasks/stats` | Task statistics |
| POST | `/api/tasks` | Create task |
| GET | `/api/tasks/{id}` | Get task |
| PUT | `/api/tasks/{id}` | Update task |
| POST | `/api/tasks/{id}/cancel` | Cancel task |
| POST | `/api/tasks/{id}/retry` | Retry task |

## System Health

```
GET /system/health
GET /system/workers
```

Returns platform health status and background worker states.

## Metrics

Prometheus metrics exposed at `/metrics` (standard endpoint):

| Metric | Type | Labels |
|--------|------|--------|
| `ai_orchestrator_http_requests_total` | Counter | method, path, status |
| `ai_orchestrator_tasks_created_total` | Counter | project_id, owner_agent, type |
| `ai_orchestrator_tasks_completed_total` | Counter | project_id, owner_agent, status |
| `ai_orchestrator_agent_requests_total` | Counter | agent_id, runner_type, task_type, status |
| `ai_orchestrator_agent_duration_seconds` | Histogram | agent_id, runner_type |
| `ai_orchestrator_active_workers` | Gauge | — |
| `ai_orchestrator_websocket_connections` | Gauge | — |

## Error Responses

All endpoints return errors in the standard envelope:

```json
{
  "success": false,
  "error": "descriptive error message"
}
```

Common HTTP status codes:
- `400` — Invalid request body or parameters
- `404` — Resource not found
- `409` — State conflict (e.g., retry non-retryable task)
- `500` — Internal server error
- `503` — Service unavailable (e.g., router not configured)
