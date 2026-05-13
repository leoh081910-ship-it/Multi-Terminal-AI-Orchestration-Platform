---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 04
type: execute
wave: 3
depends_on:
  - 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-03
files_modified:
  - web/src/api/orgApi.ts
  - web/src/pages/AgentWorkbenchPage.tsx
autonomous: true
requirements:
  - HTTP-01
  - MCP-01
must_haves:
  truths:
    - "The existing `AgentWorkbenchPage` is the Phase 6 UI integration point; do not create a parallel AgentWorkbench route/page."
    - "The existing `orgApi` is the frontend Agent create/update API; do not route this work through schedulerApi."
    - "AgentWorkbench creates HTTP and MCP Agents with serialized `runner_config` strings."
    - "MCP Agent creation includes `tool_name`, defaulting to `execute_task`, and keeps `transport` fixed to `http` for Phase 6."
    - "Backend validation errors from create/update are visible in the UI without adding a complex JSON/schema editor."
    - "Frontend lint and build pass."
  artifacts:
    - path: "web/src/api/orgApi.ts"
      provides: "frontend Agent API types with serialized runner_config contract"
      contains: "runner_config?: string"
    - path: "web/src/pages/AgentWorkbenchPage.tsx"
      provides: "existing AgentWorkbench form for HTTP/MCP registration usability"
      contains: "orgApi.createAgent"
  key_links:
    - from: "web/src/pages/AgentWorkbenchPage.tsx"
      to: "web/src/api/orgApi.ts"
      via: "create mutation sends runner_config as JSON.stringify(...) string"
      pattern: "runner_config"
    - from: "web/src/pages/AgentWorkbenchPage.tsx"
      to: "backend validation errors"
      via: "mutation error text rendered in page"
      pattern: "error"
---

<objective>
Align the existing AgentWorkbench frontend with the Phase 6 HTTP/MCP Agent contract.

Purpose: users need a basic UI path to create HTTP/MCP Agents using serialized runner configs, including MCP `tool_name`, and to see backend validation errors when configs are invalid.
Output: small updates to `orgApi` types if needed and `AgentWorkbenchPage` form behavior; no new UI library, no separate page, no complex JSON editor.
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
@web/src/api/orgApi.ts
@web/src/pages/AgentWorkbenchPage.tsx
@web/package.json

<interfaces>
Use the existing frontend API and page:
- `web/src/api/orgApi.ts` exports `Agent`, `createAgent`, `updateAgent`.
- `web/src/pages/AgentWorkbenchPage.tsx` already renders Agent cards, routing preview, and HTTP/MCP registration form.

Do not create `web/src/pages/AgentWorkbench.tsx`, do not add a new route in `App.tsx`, and do not use `schedulerApi.ts` for this org Agent work.
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Ensure orgApi types preserve serialized runner_config</name>
  <files>web/src/api/orgApi.ts</files>
  <read_first>
    <file>web/src/api/orgApi.ts</file>
    <file>internal/org/service.go</file>
    <file>internal/server/org_api.go</file>
  </read_first>
  <action>
    Review `web/src/api/orgApi.ts` and update only if needed:
    - `Agent.runner_config` must remain `string | undefined`, not an object.
    - `createAgent` input must allow `runner_type?: string` and `runner_config?: string`.
    - `updateAgent` input must allow `runner_type?: string` and `runner_config?: string` if update UI or future mutation uses it.
    - Keep the existing `unwrap`/client behavior and API route `/orgs/{orgId}/agents`.
    - Do not add scheduler Agent types or duplicate API methods.
  </action>
  <verify>
    <automated>npm --prefix web run lint</automated>
  </verify>
  <acceptance_criteria>
    - `web/src/api/orgApi.ts` contains `runner_config?: string`.
    - `web/src/api/orgApi.ts` create/update input types do not parse `runner_config` into an object.
    - `web/src/api/orgApi.ts` uses `/orgs/${orgId}/agents` routes, not scheduler agent routes.
    - `npm --prefix web run lint` exits 0.
  </acceptance_criteria>
  <done>Frontend API types keep the public serialized runner_config contract aligned with backend AgentView.</done>
</task>

<task type="auto">
  <name>Task 2: Add MCP tool_name and clearer serialized config preview to existing AgentWorkbench</name>
  <files>web/src/pages/AgentWorkbenchPage.tsx</files>
  <read_first>
    <file>web/src/pages/AgentWorkbenchPage.tsx</file>
    <file>web/src/api/orgApi.ts</file>
    <file>internal/org/service.go</file>
    <file>internal/server/org_api.go</file>
  </read_first>
  <action>
    Update the existing `HTTPAgentForm` or equivalent form inside `web/src/pages/AgentWorkbenchPage.tsx`.

    Required behavior:
    - Preserve basic inline style patterns already used in the file.
    - Preserve `orgApi.createAgent(orgId, payload)` via TanStack Query mutation.
    - For HTTP runner config, generate `runner_config` with `JSON.stringify(...)` and current HTTP fields such as `endpoint`, `method`, headers/auth/body_template/output_path/model as already supported by `HTTPRunnerConfig`.
    - For MCP runner config, generate `runner_config` with `JSON.stringify({ endpoint, transport: 'http', tool_name: mcpToolName || 'execute_task', ...existing supported fields where appropriate })`.
    - Add a visible MCP `tool_name` input with default `execute_task`.
    - Keep MCP `transport` displayed or generated as `http`; do not offer `sse` as a selectable successful option in this phase.
    - Display a read-only preview of the exact serialized `runner_config` string that will be sent.
    - Ensure backend create errors remain visible. If current error rendering is weak, add a small helper that extracts useful text from the existing API error object/message; do not add a new error framework.
    - On success, invalidate the existing `['agents', orgId]` query and show success feedback consistent with current page style.
    - Do not add a raw JSON textarea editor, schema editor, capability discovery UI, new UI library, or new route.
  </action>
  <verify>
    <automated>npm --prefix web run lint && npm --prefix web run build</automated>
  </verify>
  <acceptance_criteria>
    - `web/src/pages/AgentWorkbenchPage.tsx` contains `mcpToolName` or an equivalently named state field.
    - `web/src/pages/AgentWorkbenchPage.tsx` contains `execute_task`.
    - `web/src/pages/AgentWorkbenchPage.tsx` contains `runner_config` generated from `JSON.stringify`.
    - `web/src/pages/AgentWorkbenchPage.tsx` renders backend mutation error text visibly.
    - `web/src/pages/AgentWorkbenchPage.tsx` does not add a raw JSON editing `textarea`.
    - `web/src/pages/AgentWorkbenchPage.tsx` does not import `schedulerApi`.
    - `npm --prefix web run lint && npm --prefix web run build` exits 0.
  </acceptance_criteria>
  <done>Existing AgentWorkbench can create HTTP/MCP Agents with serialized runner_config, default MCP tool_name, fixed HTTP transport, and visible backend validation errors.</done>
</task>

</tasks>

<verification>
```bash
npm --prefix web run lint && npm --prefix web run build
```
</verification>

<success_criteria>
- Existing AgentWorkbench page, not a duplicate page, handles the Phase 6 UI scope.
- HTTP/MCP create payloads keep `runner_config` serialized as a string.
- MCP generated config includes `tool_name` and defaults it to `execute_task`.
- Backend validation errors are visible to users.
- No complex JSON/schema editor is introduced.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-04-SUMMARY.md`
</output>
