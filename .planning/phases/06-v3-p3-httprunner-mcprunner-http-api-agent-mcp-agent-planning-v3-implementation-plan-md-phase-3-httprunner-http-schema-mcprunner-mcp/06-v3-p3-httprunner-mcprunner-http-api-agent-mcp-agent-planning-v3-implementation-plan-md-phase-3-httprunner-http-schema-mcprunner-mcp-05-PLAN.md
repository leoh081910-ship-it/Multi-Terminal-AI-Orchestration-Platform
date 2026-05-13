---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 05
type: execute
wave: 3
depends_on:
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-01
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-02
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-03
files_modified:
  - docs/examples/README.md
  - docs/examples/http-runner-openai.json
  - docs/examples/http-runner-anthropic.json
  - docs/examples/http-runner-ollama.json
  - docs/examples/mcp-runner-local.json
  - docs/agent-onboarding.md
autonomous: true
requirements:
  - HTTP-01
  - MCP-01
must_haves:
  truths:
    - "Docs and examples target the existing org Agent API, not scheduler compatibility endpoints."
    - "Docs/examples preserve `runner_config` as a serialized JSON string in public API payloads."
    - "Docs describe actual `RunnerTask` template fields and do not claim `.Prompt` or `.TaskID` aliases are supported."
    - "Docs state MCP Phase 6 is HTTP-only, `transport:\"sse\"` is rejected, and `tool_name` defaults to `execute_task`."
    - "Docs distinguish optional manual real-service smoke tests from automated local fake-service validation via `httptest`."
  artifacts:
    - path: "docs/examples/README.md"
      provides: "copy-runnable runner example guide aligned to org Agent API"
      contains: "runner_config"
    - path: "docs/examples/http-runner-openai.json"
      provides: "HTTPRunner OpenAI runner_config example"
      contains: "body_template"
    - path: "docs/examples/http-runner-anthropic.json"
      provides: "HTTPRunner Anthropic runner_config example"
      contains: "toJSON"
    - path: "docs/examples/http-runner-ollama.json"
      provides: "HTTPRunner Ollama runner_config example"
      contains: "endpoint"
    - path: "docs/examples/mcp-runner-local.json"
      provides: "local MCP runner_config example"
      contains: "tool_name"
    - path: "docs/agent-onboarding.md"
      provides: "operator onboarding guide for HTTP/MCP agents"
      contains: "httptest"
  key_links:
    - from: "docs/examples/README.md"
      to: "internal/server/org_api.go"
      via: "curl examples use /api/v1/orgs/{orgID}/agents and runner_config string"
      pattern: "/api/v1/orgs/.*/agents"
    - from: "docs/agent-onboarding.md"
      to: "internal/runner/interface.go"
      via: "template variables match actual RunnerTask fields"
      pattern: ".FilesToModify"
---

<objective>
Reconcile HTTP/MCP Agent docs and examples with the Phase 6 code/API contract.

Purpose: users need copy-runnable examples that match actual org Agent API behavior, serialized `runner_config` strings, real `RunnerTask` template fields, and the HTTP-only MCP scope.
Output: updated examples README, existing HTTP runner examples, one local MCP example if missing, and onboarding docs aligned with code and validation strategy.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
@E:/04-Claude/Runtime/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/REQUIREMENTS.md
@.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-CONTEXT.md
@.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-RESEARCH.md
@CLAUDE.md
@internal/runner/interface.go
@internal/runner/http_runner.go
@internal/runner/mcp_runner.go
@internal/org/service.go
@internal/server/org_api.go
@docs/examples/README.md
@docs/examples/http-runner-openai.json
@docs/examples/http-runner-anthropic.json
@docs/examples/http-runner-ollama.json
@docs/agent-onboarding.md

<interfaces>
Document the existing org Agent API:
- Create: `POST /api/v1/orgs/{orgID}/agents`
- Request fields include `name`, `type`, optional `specialties`, `runner_type`, and serialized string `runner_config`.
- `runner_config` contents are JSON for the selected runner, but the public API field itself is a JSON string.

Do not document `/api/v1/projects/default/scheduler/agents` as the Phase 6 create path.
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Update copy-runnable examples for serialized runner_config</name>
  <files>docs/examples/README.md, docs/examples/http-runner-openai.json, docs/examples/http-runner-anthropic.json, docs/examples/http-runner-ollama.json, docs/examples/mcp-runner-local.json</files>
  <read_first>
    <file>docs/examples/README.md</file>
    <file>docs/examples/http-runner-openai.json</file>
    <file>docs/examples/http-runner-anthropic.json</file>
    <file>docs/examples/http-runner-ollama.json</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/org/service.go</file>
    <file>internal/server/org_api.go</file>
  </read_first>
  <action>
    Update `docs/examples/README.md` and existing HTTP example JSON files so they match the actual runner config structs and API contract:
    - Keep `docs/examples/http-runner-openai.json`, `http-runner-anthropic.json`, and `http-runner-ollama.json` as runner_config examples unless execution finds they are obsolete beyond repair.
    - Ensure HTTP examples use `endpoint`, supported `method`, `headers`, optional `auth_token`, `body_template`, `output_path`, `timeout_ms`, and `model` according to `HTTPRunnerConfig`.
    - Ensure any template examples use actual fields such as `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout`, and `toJSON`.
    - Add `docs/examples/mcp-runner-local.json` if missing, containing an MCP runner_config object with `endpoint`, `transport:"http"`, and `tool_name:"execute_task"`.
    - In README, show how to pass the runner_config object as a serialized JSON string in `runner_config` when creating an Agent through `/api/v1/orgs/{orgID}/agents`.
    - Prefer inline escaped JSON and/or Windows-friendly PowerShell examples over assuming `jq` is installed. If `jq` is mentioned, provide a no-jq alternative.
    - State real API/MCP smoke tests are optional manual checks after local services are running; automated validation uses Go `httptest` fake services.

    Validate example JSON syntax with `python -m json.tool` or an equivalent local parser during execution.
  </action>
  <verify>
    <automated>python -m json.tool docs/examples/http-runner-openai.json >/dev/null && python -m json.tool docs/examples/http-runner-anthropic.json >/dev/null && python -m json.tool docs/examples/http-runner-ollama.json >/dev/null && python -m json.tool docs/examples/mcp-runner-local.json >/dev/null</automated>
  </verify>
  <acceptance_criteria>
    - `docs/examples/README.md` contains `/api/v1/orgs/`.
    - `docs/examples/README.md` states `runner_config` is a serialized JSON string.
    - `docs/examples/README.md` contains `httptest`.
    - `docs/examples/http-runner-openai.json` contains `body_template`.
    - `docs/examples/http-runner-anthropic.json` or another HTTP example contains `toJSON`.
    - `docs/examples/mcp-runner-local.json` contains `"transport": "http"`.
    - `docs/examples/mcp-runner-local.json` contains `"tool_name": "execute_task"`.
    - All touched JSON example files parse successfully.
  </acceptance_criteria>
  <done>Examples are valid JSON and README shows copy-runnable org Agent creation with serialized runner_config strings and local fake-service validation guidance.</done>
</task>

<task type="auto">
  <name>Task 2: Align onboarding guide with HTTP/MCP runner contracts</name>
  <files>docs/agent-onboarding.md</files>
  <read_first>
    <file>docs/agent-onboarding.md</file>
    <file>docs/examples/README.md</file>
    <file>internal/runner/interface.go</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/org/service.go</file>
    <file>internal/server/org_api.go</file>
  </read_first>
  <action>
    Update `docs/agent-onboarding.md` with sections covering:
    - Public API contract: org Agent API path, `runner_config` is a serialized JSON string, create/update perform strict local runner construction/config validation, and registration does not require network health/capability checks.
    - HTTP runner contract: endpoint scheme `http`/`https`, supported methods `GET`, `POST`, `PUT`, `PATCH`, default method `POST`, timeout default matching code, default payload `task_id`, `task_type`, plus flattened `RunnerTask.Context`.
    - HTTP template variables: list only `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout`, and `toJSON`. State `toJSON` is registered before template Parse.
    - Unsupported aliases: state `.Prompt` and `.TaskID` are not supported aliases, and mention them only in that unsupported-alias sentence.
    - MCP runner contract: endpoint scheme `http`/`https`, `transport` must be `http` in Phase 6, `transport:"sse"` is rejected in Phase 6, `tool_name` defaults to `execute_task`, and `tools/call` params shape is `{ "name": "execute_task", "arguments": { "task_id": "...", "task_type": "..." } }`.
    - Validation: automated validation uses Go `httptest` fake HTTP/MCP services and the full command is `go test ./... && npm --prefix web run lint && npm --prefix web run build`.
    - Optional manual smoke tests: link to `docs/examples/README.md` and state real API calls require local server/fake services or user-provided services to be running.

    Remove or rewrite stale object-shaped `runner_config` request examples, `agent_id` create fields if they do not match `org.CreateAgentInput`, `.Prompt`/`.TaskID` supported-variable tables, and any claim that MCP SSE is currently supported.
  </action>
  <verify>
    <automated>go test ./internal/runner ./internal/server -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `docs/agent-onboarding.md` contains `runner_config`.
    - `docs/agent-onboarding.md` contains `/api/v1/orgs/`.
    - `docs/agent-onboarding.md` contains `.FilesToModify`.
    - `docs/agent-onboarding.md` contains `.Context.*`.
    - `docs/agent-onboarding.md` contains `toJSON`.
    - `docs/agent-onboarding.md` contains `transport:"sse" is rejected in Phase 6` or an equivalent exact statement.
    - `docs/agent-onboarding.md` contains `go test ./... && npm --prefix web run lint && npm --prefix web run build`.
    - `docs/agent-onboarding.md` contains `.Prompt` only in the unsupported-alias sentence.
    - `docs/agent-onboarding.md` contains `.TaskID` only in the unsupported-alias sentence.
    - `go test ./internal/runner ./internal/server -count=1` exits 0.
  </acceptance_criteria>
  <done>Onboarding docs describe exact HTTP/MCP contracts, unsupported aliases, httptest validation, and optional manual smoke tests aligned to actual code.</done>
</task>

</tasks>

<verification>
```bash
python -m json.tool docs/examples/http-runner-openai.json >/dev/null && python -m json.tool docs/examples/http-runner-anthropic.json >/dev/null && python -m json.tool docs/examples/http-runner-ollama.json >/dev/null && python -m json.tool docs/examples/mcp-runner-local.json >/dev/null
go test ./internal/runner ./internal/server -count=1
```
</verification>

<success_criteria>
- Docs and examples use serialized `runner_config` strings in public API payloads.
- Docs/examples use actual RunnerTask fields, not `.Prompt` or `.TaskID` aliases.
- MCP examples include `tool_name:"execute_task"` and `transport:"http"`.
- Validation docs distinguish automated `httptest` fake services from optional manual real-service smoke tests.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-05-SUMMARY.md`
</output>
