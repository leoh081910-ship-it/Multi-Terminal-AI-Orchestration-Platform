---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 06
type: execute
wave: 4
depends_on:
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-03
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-04
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-05
files_modified:
  - internal/runner/http_runner_test.go
  - internal/runner/mcp_runner_test.go
  - internal/server/server_test.go
  - docs/examples/README.md
  - docs/agent-onboarding.md
autonomous: true
requirements:
  - HTTP-01
  - MCP-01
must_haves:
  truths:
    - "All HTTP/MCP runner and API validation is automated with local fake services only."
    - "The full phase validation command must pass with exit 0: `go test ./... && npm --prefix web run lint && npm --prefix web run build`."
    - "No unrelated failure waiver can mark this phase complete; scoped fixes or upstream cleanup are required until the full command is green."
    - "Tests cover no-network registration, invalid config no-persist/preservation, serialized runner_config, default/custom tool_name, tools/list, tools/call params, SSE rejection, and HTTP template/default payload behavior."
  artifacts:
    - path: "internal/runner/http_runner_test.go"
      provides: "HTTPRunner regression tests"
      contains: "httptest.NewServer"
    - path: "internal/runner/mcp_runner_test.go"
      provides: "MCPRunner regression tests"
      contains: "httptest.NewServer"
    - path: "internal/server/server_test.go"
      provides: "org Agent API strict validation regression tests"
      contains: "runner_config"
    - path: "docs/examples/README.md"
      provides: "manual example review target"
      contains: "runner_config"
    - path: "docs/agent-onboarding.md"
      provides: "operator contract review target"
      contains: "httptest"
  key_links:
    - from: "internal/server/server_test.go"
      to: "internal/runner validation"
      via: "org Agent create/update APIs reject invalid effective configs before mutation"
      pattern: "StatusBadRequest"
    - from: "validation command"
      to: "frontend and backend"
      via: "full test/lint/build"
      pattern: "go test ./... && npm --prefix web run lint && npm --prefix web run build"
---

<objective>
Close Phase 6 with full local validation coverage and a green full-suite command.

Purpose: the phase is complete only when local fake services prove HTTP/MCP behavior, org Agent API mutation guarantees are covered, docs match the contract, and the entire backend/frontend validation command exits 0.
Output: any missing regression assertions filled, docs/examples sanity checked, and a clean full validation run.
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
@internal/runner/http_runner.go
@internal/runner/http_runner_test.go
@internal/runner/mcp_runner.go
@internal/runner/mcp_runner_test.go
@internal/server/org_api.go
@internal/server/server_test.go
@web/package.json
@docs/examples/README.md
@docs/agent-onboarding.md
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Audit and fill local fake-service regression coverage</name>
  <files>internal/runner/http_runner_test.go, internal/runner/mcp_runner_test.go, internal/server/server_test.go</files>
  <read_first>
    <file>internal/runner/http_runner_test.go</file>
    <file>internal/runner/mcp_runner_test.go</file>
    <file>internal/server/server_test.go</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/mcp_runner.go</file>
    <file>internal/server/org_api.go</file>
    <file>internal/org/service.go</file>
  </read_first>
  <behavior>
    - Test 1: HTTPRunner default payload test asserts `task_id`, `task_type`, and a flattened context key.
    - Test 2: HTTPRunner template test asserts `toJSON` works and official fields render.
    - Test 3: HTTPRunner validation tests assert invalid endpoint scheme/host, invalid method, invalid template, and valid `toJSON` template behavior.
    - Test 4: MCPRunner tests assert default `execute_task`, custom `run_agent`, `tools/list`, `tools/call`, `params.arguments`, and `transport:"sse"` rejection.
    - Test 5: Org API tests assert invalid create returns 400 before add, invalid update preserves old config/registry state, `runner_config` remains a string, and unreachable local endpoints still register when config syntax is valid.
    - Test 6: No automated test depends on real external API services, API keys, sleeps, or fixed external ports.
  </behavior>
  <action>
    Review tests added or updated by Plans 01-05. Add only missing assertions or narrowly scoped tests to cover every item below:
    - HTTPRunner: default payload contains top-level `task_id`, `task_type`, and flattened Context key; template uses `toJSON`; official fields `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout` render; invalid URL/method/template errors are covered.
    - MCPRunner: `ToolName` defaults to `execute_task`; custom `ToolName` `run_agent` is used as `params.name`; `GetCapabilities`/discovery sends `method:"tools/list"`; `Execute` sends `method:"tools/call"`; `params.arguments` contains `task_id`, `task_type`, and flattened Context; `transport:"sse"` returns the exact Phase 6 rejection error.
    - Org Agent API: create invalid HTTP/MCP config returns 400 before Agent appears in list; valid create response has `runner_config` decoded as Go `string`; invalid update returns 400 and subsequent GET/list still returns the old `runner_config`; valid create with endpoint `http://127.0.0.1:1/mcp` or `http://127.0.0.1:1/run` succeeds without dialing it.

    All new HTTP/MCP service behavior tests must use `httptest.NewServer`. Do not use real API services, sleeps, external ports, or manual smoke tests in automated tests.
  </action>
  <verify>
    <automated>go test ./internal/runner ./internal/registry ./internal/server -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/runner/http_runner_test.go` contains `httptest.NewServer`.
    - `internal/runner/http_runner_test.go` contains `toJSON`.
    - `internal/runner/mcp_runner_test.go` contains `"tools/list"`.
    - `internal/runner/mcp_runner_test.go` contains `"tools/call"`.
    - `internal/runner/mcp_runner_test.go` contains `execute_task`.
    - `internal/runner/mcp_runner_test.go` contains `run_agent`.
    - `internal/server/server_test.go` contains an invalid update preservation test for Agent runner_config.
    - `internal/server/server_test.go` contains `runner_config`.
    - `go test ./internal/runner ./internal/registry ./internal/server -count=1` exits 0.
  </acceptance_criteria>
  <done>Regression tests cover every required HTTP/MCP runner and org API validation behavior using local fake services only.</done>
</task>

<task type="auto">
  <name>Task 2: Run full phase validation command and fix scoped failures until green</name>
  <files>internal/runner/http_runner_test.go, internal/runner/mcp_runner_test.go, internal/server/server_test.go, web/src/pages/AgentWorkbenchPage.tsx, web/src/api/orgApi.ts, docs/examples/README.md, docs/agent-onboarding.md</files>
  <read_first>
    <file>web/package.json</file>
    <file>go.mod</file>
    <file>.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-VALIDATION.md</file>
  </read_first>
  <action>
    Run the required full validation command exactly:
    `go test ./... && npm --prefix web run lint && npm --prefix web run build`.

    If it fails, fix the failing files or validation blockers needed to make the exact full command pass. Do not mark Phase 6 complete with an unrelated-failure waiver. If the failure originates outside files touched by this phase, make the smallest safe cleanup needed for the full command to exit 0, or stop and report the precise blocker without creating the SUMMARY file.

    Also review docs/examples manually enough to confirm:
    - public API examples use `/api/v1/orgs/{orgID}/agents`, not scheduler agent endpoints;
    - `runner_config` is documented as a serialized JSON string;
    - no automated test requires real external HTTP/MCP services;
    - onboarding does not claim `.Prompt`, `.TaskID`, or MCP SSE support.
  </action>
  <verify>
    <automated>go test ./... && npm --prefix web run lint && npm --prefix web run build</automated>
  </verify>
  <acceptance_criteria>
    - `go test ./...` exits 0.
    - `npm --prefix web run lint` exits 0.
    - `npm --prefix web run build` exits 0.
    - No test file contains real external service hostnames for automated validation other than `127.0.0.1` or `httptest` server URLs.
    - The phase SUMMARY records the exact full command and that it passed.
  </acceptance_criteria>
  <done>The complete phase validation command has run exactly and passed; Phase 6 is ready for verification.</done>
</task>

</tasks>

<verification>
```bash
go test ./... && npm --prefix web run lint && npm --prefix web run build
```
</verification>

<success_criteria>
- All phase-required automated tests use local fake services via `httptest` only.
- Full validation command is run exactly and exits 0.
- Runner/API/frontend/docs plans are integration-safe and target canonical code paths.
- No unrelated-failure waiver is used as a completion condition.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-06-SUMMARY.md`
</output>
