---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/runner/mcp_runner.go
  - internal/runner/mcp_runner_test.go
autonomous: true
requirements:
  - MCP-01
must_haves:
  truths:
    - "MCPRunner hardens the existing canonical Runner implementation; it does not create mcp.go, runner.go, or a parallel Run interface."
    - "MCP Phase 6 is HTTP/JSON-RPC only and rejects `transport:\"sse\"` explicitly."
    - "MCPRunner supports `tool_name`, defaults it to `execute_task`, and uses it as `params.name`."
    - "MCPRunner sends `tools/call` params as `{name, arguments}` where arguments contains the task payload."
    - "MCP tool discovery uses `tools/list` with local `httptest` fake services only."
  artifacts:
    - path: "internal/runner/mcp_runner.go"
      provides: "canonical MCPRunner config validation, JSON-RPC execution, and discovery behavior"
      contains: "type MCPRunnerConfig struct"
    - path: "internal/runner/mcp_runner_test.go"
      provides: "httptest-only MCPRunner contract regressions"
      contains: "httptest.NewServer"
  key_links:
    - from: "internal/runner/mcp_runner.go"
      to: "MCP tools/call spec"
      via: "params object includes name and arguments"
      pattern: "\"tools/call\""
    - from: "internal/runner/mcp_runner.go"
      to: "Phase 6 HTTP-only constraint"
      via: "Validate rejects transport sse"
      pattern: "transport sse is not supported in Phase 6; use http"
---

<objective>
Harden the existing canonical MCPRunner implementation for Phase 6 HTTP/JSON-RPC MCP agents.

Purpose: MCP agents need deterministic local validation and standard MCP `tools/call` semantics before API registration can persist and register MCP configs safely.
Output: `tool_name` config support, `execute_task` defaulting, explicit SSE rejection, standard `{name, arguments}` params, and fake-service tests for `tools/list` and `tools/call`.
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
@internal/runner/interface.go
@internal/runner/mcp_runner.go
@internal/runner/mcp_runner_test.go
@internal/registry/registry.go

<interfaces>
Do not create `internal/runner/runner.go`, `internal/runner/mcp.go`, or a `Runner.Run(...)` interface. Use the existing contract in `internal/runner/interface.go` and the existing `MCPRunner` in `internal/runner/mcp_runner.go`.

Extend the existing `MCPRunnerConfig` with `ToolName string `json:"tool_name,omitempty"``. Keep the existing constructor shape `NewMCPRunner(id, name string, config MCPRunnerConfig) *MCPRunner` and the existing `Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error)` method.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add MCP tool_name config and reject unsupported SSE transport</name>
  <files>internal/runner/mcp_runner.go, internal/runner/mcp_runner_test.go</files>
  <read_first>
    <file>internal/runner/interface.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/runner/mcp_runner_test.go</file>
    <file>internal/registry/registry.go</file>
  </read_first>
  <behavior>
    - Test 1: `MCPRunnerConfig{Endpoint:"http://127.0.0.1:1/mcp"}.Validate()` succeeds.
    - Test 2: a runner created with no `ToolName` uses `execute_task` for execution.
    - Test 3: a runner created with `ToolName:"run_agent"` uses `run_agent` for execution.
    - Test 4: `MCPRunnerConfig{Endpoint:"http://127.0.0.1:1/mcp", Transport:"sse"}.Validate()` returns an error containing exactly `transport sse is not supported in Phase 6; use http`.
    - Test 5: whitespace-only `ToolName` fails validation with an error containing `tool_name is required`.
    - Test 6: invalid/hostless/non-http endpoints fail local validation without dialing the endpoint.
  </behavior>
  <action>
    Update `internal/runner/mcp_runner.go`:
    - Add `ToolName string `json:"tool_name,omitempty"`` to the existing `MCPRunnerConfig`.
    - Add a small helper such as `toolName()` or normalized config logic returning `execute_task` when `ToolName` is empty.
    - Update `Validate()` so empty `Transport` and `http` are accepted, `sse` returns the exact Phase 6 rejection text, and other transports return a clear `transport must be http`-style error.
    - Update endpoint validation to require absolute `http`/`https` URL with non-empty host.
    - Reject whitespace-only `ToolName`.
    - Keep timeout validation on `TimeoutMs`; do not add network checks.
    - Update comments to remove any claim that SSE is supported in Phase 6.

    Update existing MCP tests in `internal/runner/mcp_runner_test.go`. Remove or invert any test that currently treats `transport:"sse"` as valid.
  </action>
  <verify>
    <automated>go test ./internal/runner -run 'TestMCPRunnerConfig_Validate|TestMCPRunner_Execute' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/runner/mcp_runner.go` contains `ToolName string` and `json:"tool_name,omitempty"`.
    - `internal/runner/mcp_runner.go` contains `execute_task`.
    - `internal/runner/mcp_runner.go` contains the exact string `transport sse is not supported in Phase 6; use http`.
    - `internal/runner/mcp_runner.go` does not claim SSE is supported in comments.
    - `internal/runner/mcp_runner_test.go` contains assertions for both `execute_task` and `run_agent`.
    - `go test ./internal/runner -run 'TestMCPRunnerConfig_Validate|TestMCPRunner_Execute' -count=1` exits 0.
  </acceptance_criteria>
  <done>MCP configs are locally validated for the Phase 6 HTTP-only subset and runner execution has deterministic default/custom tool names.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Standardize MCP tools/list and tools/call JSON-RPC request shape</name>
  <files>internal/runner/mcp_runner.go, internal/runner/mcp_runner_test.go</files>
  <read_first>
    <file>internal/runner/interface.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/runner/mcp_runner_test.go</file>
    <file>internal/runner/capability.go</file>
  </read_first>
  <behavior>
    - Test 1: `GetCapabilities(ctx)` or the existing discovery path POSTs JSON-RPC method `tools/list` to an `httptest.NewServer` and includes configured custom headers/auth.
    - Test 2: `Execute(ctx, task)` POSTs method `tools/call`.
    - Test 3: `tools/call` request `params.name` equals default `execute_task` when `tool_name` is omitted.
    - Test 4: `tools/call` request `params.name` equals custom `run_agent` when configured.
    - Test 5: `tools/call` request `params.arguments` contains `task_id`, `task_type`, flattened `RunnerTask.Context`, and current useful task metadata already sent by MCPRunner such as `workspace_path` and `files_to_modify` when present.
    - Test 6: JSON-RPC error responses still produce a failed RunnerResult/error path consistent with existing tests.
  </behavior>
  <action>
    Update `MCPRunner.Execute` in `internal/runner/mcp_runner.go`:
    - Build a task arguments map from the existing task payload behavior: `task_id`, `task_type`, flattened `task.Context`, and existing optional metadata such as workspace path and files_to_modify.
    - Marshal JSON-RPC `params` as `{ "name": r.toolName(), "arguments": arguments }`.
    - Do not place task fields directly at the top level of `params`.
    - Keep `method:"tools/call"`, HTTP POST behavior, result parsing, output extraction, and RunnerResult shape compatible with existing code.

    Update `discoverTools`/capability discovery if needed so `tools/list` requests use the same HTTP header helper as Execute, including custom `Headers` and expanded `AuthToken`. Keep discovery outside registration; do not call it from Validate or API create/update.

    Update `internal/runner/mcp_runner_test.go` with local `httptest.NewServer` handlers that decode JSON-RPC request bodies and assert `method`, `params.name`, and `params.arguments`. Do not use real MCP servers, sleeps, or external ports.
  </action>
  <verify>
    <automated>go test ./internal/runner -run 'TestMCPRunner_Execute|TestMCPRunner_GetCapabilities' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/runner/mcp_runner.go` contains `"tools/list"`.
    - `internal/runner/mcp_runner.go` contains `"tools/call"`.
    - `internal/runner/mcp_runner.go` contains `"arguments"`.
    - `internal/runner/mcp_runner.go` assigns or encodes `"name"` using the configured/default tool name.
    - `internal/runner/mcp_runner_test.go` contains `httptest.NewServer`.
    - `internal/runner/mcp_runner_test.go` asserts both `execute_task` and `run_agent` request names.
    - `go test ./internal/runner -run 'TestMCPRunner_Execute|TestMCPRunner_GetCapabilities' -count=1` exits 0.
  </acceptance_criteria>
  <done>MCPRunner uses local fake services to prove `tools/list` discovery and standard `tools/call` params with `{name, arguments}`.</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/runner -run 'TestMCPRunnerConfig_Validate|TestMCPRunner_Execute|TestMCPRunner_GetCapabilities' -count=1
```
</verification>

<success_criteria>
- Existing canonical `MCPRunner` is hardened; no parallel runner files or interfaces are introduced.
- MCP config includes `tool_name` with default `execute_task`.
- `transport:"sse"` fails validation with the exact Phase 6 HTTP-only error.
- `tools/call` params are `{ "name": tool_name, "arguments": { ...task payload... } }`.
- `tools/list` discovery is covered by `httptest`, not external MCP services.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-02-SUMMARY.md`
</output>
