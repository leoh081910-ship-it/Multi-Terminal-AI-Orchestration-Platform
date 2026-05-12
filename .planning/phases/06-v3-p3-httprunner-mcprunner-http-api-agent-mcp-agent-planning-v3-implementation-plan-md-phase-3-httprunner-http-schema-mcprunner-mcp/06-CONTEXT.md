# Phase 6: v3-P3 HTTPRunner + MCPRunner - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 6 delivers production-ready HTTP API Agent and MCP protocol Agent support for the v3 Runner system. The phase should close the gap between the current first-pass implementation and a reliable registration/execution contract: HTTPRunner config/schema behavior, MCPRunner execution semantics, example configs, basic AgentWorkbench registration usability, and repeatable local end-to-end tests.

This phase does not expand into advanced routing/capability scoring, full UI configuration editors, Prometheus/trace observability, or production-grade SSE lifecycle support; those belong to later v3 phases or existing downstream roadmap items.

</domain>

<decisions>
## Implementation Decisions

### Registration and Validation
- **D-01:** Creating or updating an HTTP/MCP Agent must fail with `400` when `runner_config` is invalid or the Runner cannot be built. Do not persist a DB record that cannot construct a usable Runner.
- **D-02:** The public API contract keeps `runner_config` as a serialized JSON string, matching current `AgentView`, frontend form behavior, and storage shape.
- **D-03:** Registration performs synchronous construction/config validation only. It should not synchronously call external health checks or capability discovery during create/update, so registration is not blocked by network/API availability.

### HTTPRunner Template Contract
- **D-04:** `body_template` officially uses `RunnerTask` fields: `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout` where applicable. Documentation and examples must be aligned to this contract.
- **D-05:** Go templates must support a `toJSON` helper for safe JSON embedding. Existing examples already rely on this and the phase should make the code match the documented pattern.
- **D-06:** If `body_template` is omitted, HTTPRunner should keep the current simple default payload shape: task metadata (`task_id`, `task_type`) plus flattened `RunnerTask.Context` values.

### MCPRunner Execution Semantics
- **D-07:** MCPRunner must support a configured `tool_name` in `runner_config`, defaulting to `execute_task`.
- **D-08:** MCP `tools/call` params must use the standard shape: `{ "name": tool_name, "arguments": { ...task payload... } }` rather than directly placing task fields at the top level of `params`.
- **D-09:** Phase 6 supports MCP over HTTP/JSON-RPC only. `transport: "sse"` should not be silently accepted as if implemented; either reject it during validation or clearly mark it unsupported until a later phase.

### Verification and Acceptance
- **D-10:** Automated end-to-end validation should use local fake HTTP and MCP JSON-RPC services, not real OpenAI/Anthropic calls. Real API examples may remain manual smoke-test documentation only.
- **D-11:** The existing AgentWorkbench registration form is in scope at a basic usability level: fields should match the locked API contract and errors should be visible, but a full schema-driven config editor is out of scope.
- **D-12:** Example docs/configs must be copy-runnable and consistent with actual API behavior, including environment variable usage and serialized `runner_config` expectations.

### Claude's Discretion
- Exact internal package/file split is open to planner discretion. The roadmap references `http_config.go` and `mcp_capability.go`, but existing code currently co-locates config/capability behavior in `http_runner.go` and `mcp_runner.go`; planner may keep or split files based on maintainability.
- Exact local e2e test harness shape is open, as long as it is repeatable without external API keys and exercises registration plus Runner execution semantics.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope and Roadmap
- `.planning/ROADMAP.md` — Formal Phase 6 entry and v3 phase mapping.
- `.planning/v3-implementation-plan.md` §Phase 3 — Original HTTPRunner/MCPRunner task list and acceptance targets.
- `.planning/v3-multi-agent-roadmap.md` — v3 universal multi-agent orchestration vision and HTTP/MCP requirements.

### Existing Runner and Registry Contracts
- `internal/runner/interface.go` — Runner interface, RunnerTask, RunnerResult, RunnerType values.
- `internal/runner/capability.go` — CapabilityManifest shape and default HTTP/MCP manifests.
- `internal/runner/http_runner.go` — Current HTTPRunner implementation, config, templating, output extraction, health/cancel behavior.
- `internal/runner/mcp_runner.go` — Current MCPRunner implementation, JSON-RPC structs, tools/list discovery, health/cancel behavior.
- `internal/registry/registry.go` — LoadFromDB and BuildRunner factory for CLI/HTTP/MCP runners.

### API and Frontend Integration
- `internal/org/service.go` — Agent create/update inputs and persisted runner_type/runner_config fields.
- `internal/server/org_api.go` — Agent create/update/capabilities endpoints and registry registration behavior.
- `web/src/api/orgApi.ts` — Frontend Agent API types and create/update contracts.
- `web/src/pages/AgentWorkbenchPage.tsx` — Current HTTP/MCP registration form and capabilities UI.

### Documentation and Examples
- `docs/examples/README.md` — Current runner example guide, template variables, output extraction, MCP config.
- `docs/examples/http-runner-openai.json` — OpenAI HTTPRunner example config.
- `docs/examples/http-runner-anthropic.json` — Anthropic HTTPRunner example config.
- `docs/examples/http-runner-ollama.json` — Ollama HTTPRunner example config.
- `docs/agent-onboarding.md` — Current Agent onboarding guide; contains contract drift that should be reconciled with code.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `runner.HTTPRunner` already implements request templating, headers/auth token environment expansion, output path extraction, health check, cancel, capabilities, and tests.
- `runner.MCPRunner` already implements JSON-RPC request/response handling, `initialize` health check, `tools/list` discovery, `tools/call` execution, output extraction, cancel, capabilities, and tests.
- `registry.BuildRunner` already constructs CLI/HTTP/MCP runners from stored JSON config and validates HTTP/MCP config.
- Org API already registers newly-created or updated runners into `runnerRegistry` immediately after DB writes.
- AgentWorkbench already has a basic HTTP/MCP registration form, runner badges, capabilities fetch, and routing preview.

### Established Patterns
- Backend API errors are returned as `APIResponse{Success:false, Error:...}` with HTTP status codes.
- Runner config is stored as `agent.runner_config` string and mirrored to frontend as `runner_config?: string`.
- Environment variable expansion uses `${VAR}` via `os.Expand` for auth tokens and headers.
- TanStack Query is used for frontend server state and mutations.

### Integration Points
- `internal/server/org_api.go` should enforce the locked validation behavior before or transactionally with DB persistence.
- `internal/org/service.go` should preserve the JSON-string API contract and not expose object-shaped runner_config as the official API.
- `internal/runner/http_runner.go` needs the `toJSON` template helper to align implementation with examples.
- `internal/runner/mcp_runner.go` needs `tool_name` config and standard MCP `tools/call` params.
- `docs/examples/` and `docs/agent-onboarding.md` must be reconciled with the actual template variables, timeout default, MCP fields, and serialized runner_config contract.

</code_context>

<specifics>
## Specific Ideas

- Keep the phase focused on making existing HTTP/MCP support reliable and coherent rather than adding advanced routing or a large UI editor.
- Prefer deterministic local fake-service tests over real API calls so validation is repeatable without network, API keys, or cost.
- Treat current doc/code drift as part of the phase: examples should be copy-runnable and match the locked API exactly.

</specifics>

<deferred>
## Deferred Ideas

- Full schema-driven frontend config editor with field-level validation/help is deferred beyond this phase.
- MCP SSE transport support is deferred beyond this phase.
- Automatic MCP tool selection from `tools/list` based on task type is deferred to capability/routing enhancement phases.
- Real external API smoke tests may be documented as optional manual checks, but not required for automated Phase 6 acceptance.

</deferred>

---

*Phase: 06-v3-p3-httprunner-mcprunner*
*Context gathered: 2026-05-13*
