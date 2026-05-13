---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 03
type: execute
wave: 2
depends_on:
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-01
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-02
files_modified:
  - internal/server/org_api.go
  - internal/org/service.go
  - internal/registry/registry.go
  - internal/server/server_test.go
autonomous: true
requirements:
  - HTTP-01
  - MCP-01
must_haves:
  truths:
    - "The existing org Agent API is the authoritative create/update path for Phase 6; do not implement scheduler compatibility agent endpoints."
    - "Create HTTP/MCP Agent requests return 400 before persistence when the effective runner config cannot build a runner."
    - "Update HTTP/MCP Agent requests validate the effective post-update config before DB mutation and preserve the previous DB config and registry entry on invalid input."
    - "The public API keeps `runner_config` as a serialized JSON string, never an object."
    - "Agent registration performs local runner construction/config validation only; it does not call health checks, tools/list, or external endpoints."
  artifacts:
    - path: "internal/server/org_api.go"
      provides: "pre-persistence Agent create/update validation and registry registration ordering"
      contains: "handleCreateAgent"
    - path: "internal/org/service.go"
      provides: "serialized runner_config storage and AgentView contract"
      contains: "RunnerConfig string"
    - path: "internal/registry/registry.go"
      provides: "canonical BuildRunner factory reused for validation"
      contains: "func BuildRunner"
    - path: "internal/server/server_test.go"
      provides: "org API regression tests for no-persist invalid config and update preservation"
      contains: "runner_config"
  key_links:
    - from: "internal/server/org_api.go"
      to: "internal/registry/registry.go"
      via: "pre-persistence validation reuses registry.BuildRunner"
      pattern: "registry.BuildRunner"
    - from: "internal/server/org_api.go"
      to: "internal/org/service.go"
      via: "validate before CreateAgent/UpdateAgent mutates SQLite"
      pattern: "orgSvc.CreateAgent|orgSvc.UpdateAgent"
---

<objective>
Wire strict HTTP/MCP runner config validation into the existing org Agent API.

Purpose: users must not be able to create or update an HTTP/MCP Agent into a non-runnable state, while preserving the serialized `runner_config` public contract and avoiding registration-time network checks.
Output: pre-persistence validation in `internal/server/org_api.go`, update preservation behavior, registry consistency, and server tests proving invalid configs return 400 without mutating state.
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
@.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-VALIDATION.md
@CLAUDE.md
@internal/server/org_api.go
@internal/org/service.go
@internal/registry/registry.go
@internal/runner/http_runner.go
@internal/runner/mcp_runner.go
@internal/server/server_test.go

<interfaces>
Use the current org API routes in `internal/server/org_api.go`:
- `POST /api/v1/orgs/{orgID}/agents` handled by `handleCreateAgent`
- `PATCH /api/v1/orgs/{orgID}/agents/{agentID}` handled by `handleUpdateAgent`
- `GET /api/v1/orgs/{orgID}/agents` handled by `handleListAgents`

Do not implement `/api/v1/projects/{project}/scheduler/agents` for this phase. That was a rejected planning path.

Preserve these existing public shapes:
- `org.CreateAgentInput.RunnerConfig string `json:"runner_config,omitempty"``
- `org.UpdateAgentInput.RunnerConfig *string `json:"runner_config,omitempty"``
- `org.AgentView.RunnerConfig string `json:"runner_config"``
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add org API pre-persistence runner validation helper</name>
  <files>internal/server/org_api.go, internal/registry/registry.go, internal/server/server_test.go</files>
  <read_first>
    <file>internal/server/org_api.go</file>
    <file>internal/org/service.go</file>
    <file>internal/registry/registry.go</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/server/server_test.go</file>
  </read_first>
  <behavior>
    - Test 1: POST an HTTP Agent with missing/invalid endpoint returns HTTP 400 and response error contains the runner validation error.
    - Test 2: POST an MCP Agent with `transport:"sse"` returns HTTP 400 and response error contains `transport sse is not supported in Phase 6; use http`.
    - Test 3: after an invalid create response, listing Agents for the org does not include the requested Agent.
    - Test 4: POST valid HTTP/MCP Agents with unreachable local endpoints such as `http://127.0.0.1:1/run` or `/mcp` returns 201 because validation builds runners but does not dial endpoints.
    - Test 5: POST with `runner_config` as a JSON object, not a JSON string, returns HTTP 400 from request decoding or validation and does not persist.
  </behavior>
  <action>
    Update `internal/server/org_api.go` so `handleCreateAgent` validates the effective runner config before calling `s.orgSvc.CreateAgent`.

    Implementation guidance:
    - Add a server helper such as `validateAgentRunnerBuild(agentID, agentName, runnerType, runnerConfig string) (runner.Runner, error)` in `org_api.go` or a nearby existing server file.
    - Scope strict pre-validation to effective runner types `http` and `mcp`; preserve CLI compatibility unless already safely validated by existing code.
    - Decode `runner_config` by unmarshalling the serialized string into `json.RawMessage`; if the string is malformed JSON, return 400 with a clear error.
    - Call `registry.BuildRunner` with `BuilderInput{AgentID: ..., AgentName: ..., RunnerType: registry.RunnerTypeFromString(runnerType), Config: raw}`.
    - Do not call `HealthCheck`, `GetCapabilities`, MCP `discoverTools`, HTTP endpoint requests, or any other network path during validation.
    - For create, using a temporary validation ID before the final DB ID is acceptable because ID/name do not affect HTTP/MCP config validity. After `CreateAgent` succeeds, build/register the final runner from the returned `AgentView` as the current code does, but invalid config must never reach that point.
    - Keep API error envelope consistent with existing `APIResponse{Success:false, Error:...}` behavior.

    Update or add server tests in `internal/server/server_test.go` using existing in-memory test setup. Tests should drive the real org API handlers and inspect list results through public API/service calls, not private maps.
  </action>
  <verify>
    <automated>go test ./internal/server -run 'TestHandleCreateAgent_Invalid.*RunnerConfig|TestHandleCreateAgent_Valid.*RunnerConfigDoesNotDial|TestHandleCreateAgent_RunnerConfigMustBeString' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/server/org_api.go` validates HTTP/MCP runner config before calling `s.orgSvc.CreateAgent`.
    - `internal/server/org_api.go` contains `registry.BuildRunner` or calls a helper that does.
    - `internal/server/org_api.go` does not call `HealthCheck` or `GetCapabilities` during create/update validation.
    - `internal/server/server_test.go` contains a create-invalid HTTP runner config test.
    - `internal/server/server_test.go` contains a create-invalid MCP runner config test.
    - `internal/server/server_test.go` asserts invalid create does not persist the Agent.
    - `go test ./internal/server -run 'TestHandleCreateAgent_Invalid.*RunnerConfig|TestHandleCreateAgent_Valid.*RunnerConfigDoesNotDial|TestHandleCreateAgent_RunnerConfigMustBeString' -count=1` exits 0.
  </acceptance_criteria>
  <done>Org API create rejects invalid HTTP/MCP configs before SQLite persistence and valid unreachable local endpoints still register without network checks.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Preserve existing config and registry on invalid Agent update</name>
  <files>internal/server/org_api.go, internal/org/service.go, internal/server/server_test.go</files>
  <read_first>
    <file>internal/server/org_api.go</file>
    <file>internal/org/service.go</file>
    <file>internal/registry/registry.go</file>
    <file>internal/server/server_test.go</file>
  </read_first>
  <behavior>
    - Test 1: create a valid HTTP Agent, then PATCH it with an invalid HTTP runner_config; PATCH returns HTTP 400.
    - Test 2: after the failed PATCH, GET/list still returns the old `runner_config` string exactly as previously stored.
    - Test 3: after the failed PATCH, the live runner registry still has a runner for the Agent that was built from the old valid config rather than being replaced or removed.
    - Test 4: PATCH changing a valid HTTP Agent to a valid MCP config succeeds if the MCP config is locally buildable.
    - Test 5: PATCH with `runner_config` as an object is rejected and preserves old state.
    - Test 6: PATCH changing only a non-runner field on an Agent whose effective stored HTTP/MCP config is invalid returns HTTP 400 and does not mutate the Agent.
  </behavior>
  <action>
    Update `handleUpdateAgent` in `internal/server/org_api.go`:
    - Load the existing Agent before mutation using the existing org service getter.
    - Compute the effective post-update runner type and runner_config by applying non-nil patch fields to the existing `AgentView`.
    - If the effective runner type is `http` or `mcp`, validate/build the effective post-update config before calling `s.orgSvc.UpdateAgent`, regardless of which request fields changed.
    - If validation fails, return HTTP 400 and do not call `UpdateAgent`.
    - After a successful DB update, register the already-validated final runner or rebuild from returned `AgentView` only after DB mutation succeeds.
    - Do not remove or overwrite the existing registry entry before validation succeeds.

    Update `internal/org/service.go` only if a small service method or behavior adjustment is necessary to load existing state or preserve string fields cleanly. Do not move registry imports into `internal/org` if that would create coupling or import cycles.

    Add server tests to `internal/server/server_test.go` using the existing API/test server patterns. Assert `runner_config` decodes as a Go string in API responses, and include a regression proving non-runner-field PATCH cannot mutate an existing HTTP/MCP Agent whose effective stored runner_config is invalid.
  </action>
  <verify>
    <automated>go test ./internal/server -run 'TestHandleUpdateAgent_InvalidRunnerConfigPreservesExisting|TestHandleUpdateAgent_ValidRunnerConfig' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/server/org_api.go` computes effective update config before calling `s.orgSvc.UpdateAgent`.
    - `internal/server/server_test.go` contains `TestHandleUpdateAgent_InvalidRunnerConfigPreservesExisting` or an equivalently named test.
    - `internal/server/server_test.go` asserts a failed update leaves old `runner_config` unchanged.
    - `internal/server/server_test.go` asserts API `runner_config` decodes as Go `string`.
    - `go test ./internal/server -run 'TestHandleUpdateAgent_InvalidRunnerConfigPreservesExisting|TestHandleUpdateAgent_ValidRunnerConfig' -count=1` exits 0.
  </acceptance_criteria>
  <done>Org API update validates before mutation, preserves persisted and live registry state on invalid HTTP/MCP config, and keeps `runner_config` serialized as a string.</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/server -run 'TestHandle(Create|Update)Agent_.*RunnerConfig' -count=1
go test ./internal/registry ./internal/server -count=1
```
</verification>

<success_criteria>
- Invalid HTTP/MCP Agent create/update returns 400 before persistence or registry mutation.
- Valid registrations do not require live HTTP/MCP services.
- Public API keeps `runner_config` as serialized JSON string.
- Existing org Agent API is hardened; no scheduler-agent compatibility endpoint is introduced for this phase.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-03-SUMMARY.md`
</output>
