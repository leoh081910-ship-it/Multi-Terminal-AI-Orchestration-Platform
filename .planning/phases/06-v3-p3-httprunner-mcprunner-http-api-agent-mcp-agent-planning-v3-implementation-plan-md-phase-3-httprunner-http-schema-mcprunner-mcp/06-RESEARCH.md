# Phase 6: v3-P3 HTTPRunner + MCPRunner - Research

**Researched:** 2026-05-13
**Domain:** Go Runner abstractions, HTTP API agents, MCP HTTP/JSON-RPC agents, React Agent Workbench
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Registration and Validation
- **D-01:** Creating or updating an HTTP/MCP Agent must fail with `400` when `runner_config` is invalid or the Runner cannot be built. Do not persist a DB record that cannot construct a usable Runner.
- **D-02:** The public API contract keeps `runner_config` as a serialized JSON string, matching current `AgentView`, frontend form behavior, and storage shape.
- **D-03:** Registration performs synchronous construction/config validation only. It should not synchronously call external health checks or capability discovery during create/update, so registration is not blocked by network/API availability.

#### HTTPRunner Template Contract
- **D-04:** `body_template` officially uses `RunnerTask` fields: `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout` where applicable. Documentation and examples must be aligned to this contract.
- **D-05:** Go templates must support a `toJSON` helper for safe JSON embedding. Existing examples already rely on this and the phase should make the code match the documented pattern.
- **D-06:** If `body_template` is omitted, HTTPRunner should keep the current simple default payload shape: task metadata (`task_id`, `task_type`) plus flattened `RunnerTask.Context` values.

#### MCPRunner Execution Semantics
- **D-07:** MCPRunner must support a configured `tool_name` in `runner_config`, defaulting to `execute_task`.
- **D-08:** MCP `tools/call` params must use the standard shape: `{ "name": tool_name, "arguments": { ...task payload... } }` rather than directly placing task fields at the top level of `params`.
- **D-09:** Phase 6 supports MCP over HTTP/JSON-RPC only. `transport: "sse"` should not be silently accepted as if implemented; either reject it during validation or clearly mark it unsupported until a later phase.

#### Verification and Acceptance
- **D-10:** Automated end-to-end validation should use local fake HTTP and MCP JSON-RPC services, not real OpenAI/Anthropic calls. Real API examples may remain manual smoke-test documentation only.
- **D-11:** The existing AgentWorkbench registration form is in scope at a basic usability level: fields should match the locked API contract and errors should be visible, but a full schema-driven config editor is out of scope.
- **D-12:** Example docs/configs must be copy-runnable and consistent with actual API behavior, including environment variable usage and serialized `runner_config` expectations.

### Claude's Discretion
- Exact internal package/file split is open to planner discretion. The roadmap references `http_config.go` and `mcp_capability.go`, but existing code currently co-locates config/capability behavior in `http_runner.go` and `mcp_runner.go`; planner may keep or split files based on maintainability.
- Exact local e2e test harness shape is open, as long as it is repeatable without external API keys and exercises registration plus Runner execution semantics.

### Deferred Ideas (OUT OF SCOPE)
- Full schema-driven frontend config editor with field-level validation/help is deferred beyond this phase.
- MCP SSE transport support is deferred beyond this phase.
- Automatic MCP tool selection from `tools/list` based on task type is deferred to capability/routing enhancement phases.
- Real external API smoke tests may be documented as optional manual checks, but not required for automated Phase 6 acceptance.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HTTP-01 | HTTPRunner HTTP API invocation for v3 Agent execution. | Existing `internal/runner/http_runner.go` is mostly present; plan must add template helper validation, registration pre-validation, tests, and doc/example alignment. |
| MCP-01 | MCPRunner MCP protocol support for v3 Agent execution. | Existing `internal/runner/mcp_runner.go` is mostly present; plan must add `tool_name`, standard `tools/call` params, reject unsupported SSE, and fake MCP tests. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Use Go backend + React/Vite frontend.
- Keep SQLite as the only v1 storage backend.
- Windows 11 is the primary platform; handle long paths, Chinese paths, spaces, and symlink permissions when execution touches filesystem/worktrees.
- Expected scale is 10-20 concurrent tasks; registry/runner changes must remain concurrency-safe.
- Claude Code CLI is the first Runner, but Runner abstractions must support extension.
- Follow existing codebase patterns because conventions and architecture are still emergent.
- GSD workflow says file-changing work should run through GSD entry points; this research was invoked by a GSD phase flow.
- Do not introduce a separate Python or Node toolchain for Claude; use project/global tooling already present.

## Summary

Phase 6 is not greenfield. `HTTPRunner`, `MCPRunner`, `Registry.BuildRunner`, org Agent APIs, AgentWorkbench basics, and runner docs/examples already exist. The phase should be planned as a hardening and contract-alignment phase: preserve the serialized JSON string `runner_config` API contract, make create/update fail before persistence for invalid HTTP/MCP configs, align HTTP templates with the documented `RunnerTask` data object and `toJSON`, and make MCP calls conform to the official `tools/call` `{name, arguments}` shape.

The most important implementation gap is transactional/ordering behavior in `internal/server/org_api.go`: today `handleCreateAgent` persists first and then best-effort builds/registers the Runner. If build fails, a broken Agent can remain in the DB, violating D-01. `handleUpdateAgent` has the same risk: it updates DB first, then best-effort rebuilds. Plan validation before DB writes, and for updates derive the effective post-update runner type/config before persisting.

MCP should be scoped to HTTP/JSON-RPC only in this phase. Official current MCP docs define Streamable HTTP as the standard HTTP transport replacing older HTTP+SSE; because D-09 defers SSE, validation should reject `transport: "sse"` rather than silently accepting it. Local fake services should drive automated e2e tests; real OpenAI/Anthropic/Ollama examples should stay documentation/manual only.

**Primary recommendation:** Plan a focused contract-hardening sequence: backend validation first, HTTP template helper second, MCP request shape third, local e2e fourth, then AgentWorkbench/docs/examples cleanup.

## Current Implementation Status and Gaps

| Area | Current Status | Gap vs Locked Decisions | Recommended Planning Action | Confidence |
|------|----------------|-------------------------|-----------------------------|------------|
| Runner interface | `internal/runner/interface.go` defines `Runner`, `RunnerTask`, `RunnerResult`, `Workspace`, `RunnerType` values. | No phase blocker. Template docs must use actual `RunnerTask` fields. | Reference this contract directly in docs/tests. | HIGH |
| HTTPRunner config validation | `HTTPRunnerConfig.Validate()` requires endpoint and non-negative timeout. `registry.BuildRunner` calls it. | Does not parse `body_template` at registration, so an unusable template can persist. URL parsing allows relative/hostless strings if `url.Parse` succeeds. Method is not validated. | Extend `Validate()` to parse template with FuncMap and require absolute `http`/`https` URL. Optionally validate HTTP method against allowed methods. | HIGH |
| HTTPRunner templating | `renderBody()` uses `template.New("body").Parse(...)` without FuncMap. Default payload is `task_id`, `task_type`, flattened context. | `toJSON` is missing; examples and AgentWorkbench placeholder use `toJSON` and currently fail at parse/execute. | Add `template.FuncMap{"toJSON": toJSON}` before `Parse`. Implement `toJSON(any) (string, error)` with `json.Marshal`. Add tests for `.Context.prompt | toJSON`, arrays/maps, and parse failure. | HIGH |
| HTTPRunner default body | Default body already matches D-06: task metadata plus flattened `Context`. | No gap except tests should lock behavior. | Add regression test asserting omitted body_template sends only `task_id`, `task_type`, and flattened context. | HIGH |
| MCPRunner config | `MCPRunnerConfig` has endpoint, transport, headers, auth_token, timeout_ms, tools_enabled. | Missing `tool_name`; accepts `transport: "sse"` despite phase excluding SSE. Comments say SSE supported although code only posts JSON-RPC. | Add `ToolName string json:"tool_name,omitempty"`; default to `execute_task`. Change validation to allow empty/http only and reject sse with unsupported message. Update comments/docs. | HIGH |
| MCP `tools/call` request | `Execute()` sends `method:"tools/call"` with task fields directly in `params`. | Violates official MCP shape and D-08. | Build `arguments` payload from task fields, then marshal params as `{ "name": toolName, "arguments": arguments }`. | HIGH |
| MCP HTTP headers | Current MCP calls set `Content-Type` but not `Accept`; headers only partly applied in `discoverTools` (auth only, custom headers omitted). | Official Streamable HTTP says POST should include `Accept: application/json, text/event-stream` and protocol version header on subsequent requests. D-09 does not require full Streamable HTTP lifecycle, but local JSON-RPC should be more compliant. | Add shared request helper setting `Content-Type`, `Accept`, auth, custom headers. Consider `MCP-Protocol-Version: 2025-06-18` after initialization if implementing session later; for this phase document not full session lifecycle. | MEDIUM |
| MCP capability discovery | `GetCapabilities()` calls `tools/list` and uses tool names as `TaskTypes`. | Capability discovery is allowed outside registration. It currently ignores custom headers in `discoverTools`; it does not expose tool schemas. | Fix headers. Keep basic discovery; do not add auto tool selection. Add fake `tools/list` test. | MEDIUM |
| Registry factory | `registry.BuildRunner()` constructs CLI/HTTP/MCP and validates configs. | Good reuse point for API pre-validation. CLI unmarshal ignores errors; HTTP/MCP strict. | Add server helper `validateRunnerInput` that calls BuildRunner with effective config before DB persistence for HTTP/MCP and returns 400 errors. | HIGH |
| Org service | `CreateAgent` stores `RunnerConfig` string; defaults CLI config if empty; update sets fields directly. | Service is storage-only and does not validate JSON; OK if server validates. But if used elsewhere, it can persist broken HTTP/MCP configs. | Keep public contract string. Prefer validation in server/API layer for this phase; optionally add service-level validation callback only if it does not import registry and create cycles. | MEDIUM |
| Org API create/update | `handleCreateAgent` persists first, then best-effort registers; `buildRunnerFromAgentView` returns nil on errors. `handleUpdateAgent` persists first, then best-effort re-registers. | Direct D-01 violation: broken HTTP/MCP Agent records can be persisted and update can leave DB and registry inconsistent. | Validate/build before `CreateAgent`; for update, load existing Agent, compute effective runner_type/config, validate/build before `UpdateAgent`, then register the pre-built Runner after DB save. If register cannot fail, DB stays consistent. | HIGH |
| AgentWorkbench | Basic form exists for HTTP/MCP; errors visible via mutation; runner_config is stringified JSON. It always uses MCP transport http and tools_enabled true. | No `tool_name` input. Body template shown for MCP even though MCP does not use it. Error messages depend on backend. No update form. | Add basic `tool_name` field when MCP selected; hide or clarify HTTP-only body_template/output_path/model fields where appropriate. Preserve stringified `runner_config`. | HIGH |
| Docs/examples | Examples use serialized runner_config files, `RunnerTask` fields, env expansion. Onboarding still shows object-shaped runner_config, `agent_id`, `.Prompt`/`.TaskID`, default 60000ms, SSE. | D-02/D-04/D-09/D-12 drift. README curl cannot directly pass file content as escaped JSON string. | Rework examples to include copy-runnable curl using `jq -Rs` or documented escaping; update onboarding to serialized `runner_config`, actual endpoints, 300000ms default, `.ID`/`.Type`/`.Command`/`.Context.*`, and HTTP-only MCP. | HIGH |
| Tests | Unit tests exist for HTTP/MCP, registry factory, server. No local fake end-to-end that covers API registration + Runner execution semantics. Existing MCP tests currently expect old direct params and accept SSE. | D-10 not satisfied; tests will need updates for new contracts. | Add local fake HTTP and MCP httptest services. Include API create failure tests and direct Runner execution tests. Optional server e2e via org API plus registry execution if test server exposes registry injection. | HIGH |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.2 installed, module declares 1.25.0 | Backend implementation and tests | Current project backend is Go; `net/http`, `httptest`, `text/template`, and `encoding/json` cover Phase 6 without new dependencies. |
| `net/http` / `httptest` | Go stdlib | HTTPRunner, MCP HTTP JSON-RPC, local fake services | Avoids new runtime/tooling; existing code already uses it. |
| `text/template` | Go stdlib | HTTP request body templating | Existing implementation uses Go templates; official stdlib supports FuncMap for `toJSON`. |
| `encoding/json` | Go stdlib | Config parsing, `toJSON`, JSON-RPC encoding | Existing runner_config and JSON-RPC structs already use it. |
| chi | Project uses v5.2.1; latest checked v5.2.5 | HTTP API routing | Existing server uses chi; no route architecture change needed. |
| ent | v0.14.6 | Agent persistence | Existing org service and schema are ent-based. |
| React | Project uses ^19.2.4; latest checked 19.2.6 | AgentWorkbench UI | Existing frontend stack; only minor form UX changes needed. |
| TanStack Query | Project uses ^5.96.2; latest checked 5.100.10 | Agent API mutations/query caching | Existing AgentWorkbench uses it; no alternative needed. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.11.1 | Assertions in new Go tests if desired | Existing tests mostly use stdlib; use only if nearby patterns already do. |
| `modernc.org/sqlite` | v1.37.1 | In-memory ent API tests | Existing server tests use it for `file:...mode=memory`. |
| TypeScript | Project uses ~6.0.2; latest checked 6.0.3 | Frontend type safety | Needed only for AgentWorkbench and orgApi type changes. |
| Vite | Project uses ^5.4.21; latest checked 8.0.12 | Frontend build | Do not upgrade in this phase; just keep lint/build green. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go stdlib JSON-RPC structs | MCP Go SDK | Roadmap originally mentioned `mcp-go`, but current code already implements enough HTTP/JSON-RPC. Adding an SDK risks dependency churn and broader lifecycle work outside D-09. |
| Dot-notation `NavigateJSON` | JSONPath library | Current extractor exists and is covered by tests; a new JSONPath dependency is unnecessary for Phase 6. |
| Handwritten AgentWorkbench form | Schema-driven config editor | Explicitly deferred; basic usability only is in scope. |
| Real OpenAI/Anthropic e2e | Local `httptest` fake service | Real calls require keys/network/cost and violate D-10 for automated e2e. |

**Installation:**

No new production dependency is required. Use existing commands:

```bash
go test ./internal/runner ./internal/registry ./internal/server
npm --prefix web run lint
npm --prefix web run build
```

**Version verification:**

Checked 2026-05-13:
- `go version`: `go1.26.2 windows/amd64`
- `npm view @tanstack/react-query version`: `5.100.10`, modified `2026-05-11T22:14:14.975Z`
- `npm view react version`: `19.2.6`, modified `2026-05-08T16:46:20.455Z`
- `npm view vite version`: `8.0.12`, modified `2026-05-11T07:11:54.747Z`
- `npm view typescript version`: `6.0.3`, modified `2026-04-16T23:38:57.055Z`
- `go list -m -versions github.com/go-chi/chi/v5`: latest seen `v5.2.5`
- `go list -m -versions entgo.io/ent`: latest seen `v0.14.6`

Do not plan dependency upgrades as part of this phase unless tests require it.

## Architecture Patterns

### Recommended Project Structure

Use the current package layout; split files only where it improves clarity.

```text
internal/
├── runner/
│   ├── interface.go          # RunnerTask/RunnerResult contract
│   ├── http_runner.go        # HTTPRunner config, validation, templating, execution
│   ├── http_runner_test.go   # HTTPRunner fake-service unit/e2e tests
│   ├── mcp_runner.go         # MCP config, JSON-RPC helpers, execution, discovery
│   └── mcp_runner_test.go    # MCP fake-service tests
├── registry/
│   ├── registry.go           # BuildRunner factory and live registry
│   └── registry_test.go      # validation/factory tests
├── server/
│   ├── org_api.go            # pre-persistence validation for Agent create/update
│   └── server_test.go        # org API 400/no-persist tests
web/src/
├── api/orgApi.ts             # serialized runner_config API type
└── pages/AgentWorkbenchPage.tsx # basic HTTP/MCP form updates
docs/
├── agent-onboarding.md       # contract-correct registration guide
└── examples/                 # copy-runnable configs and README
```

### Pattern 1: Pre-persistence Runner validation

**What:** Decode the serialized `runner_config` string and call `registry.BuildRunner` before writing HTTP/MCP Agent records.

**When to use:** `POST /api/v1/orgs/{orgID}/agents` and `PATCH /api/v1/orgs/{orgID}/agents/{agentID}` whenever the effective runner type is `http` or `mcp`.

**Example:**

```go
// Source: existing registry.BuildRunner contract and D-01/D-03
func (s *Server) validateRunnerBuild(agentID, name, runnerType, cfgString string) (runner.Runner, error) {
    rt := registry.RunnerTypeFromString(runnerType)
    var raw json.RawMessage
    if cfgString != "" && cfgString != "{}" {
        if err := json.Unmarshal([]byte(cfgString), &raw); err != nil {
            return nil, fmt.Errorf("runner_config must be valid JSON string: %w", err)
        }
    }
    return registry.BuildRunner(registry.BuilderInput{
        AgentID:    agentID,
        AgentName:  name,
        RunnerType: rt,
        Config:     raw,
    })
}
```

Planner note: the exact helper should avoid cyclic imports and use existing server patterns. For create, the final DB ID is generated inside `org.Service.CreateAgent`, so either validate with a temporary ID or move ID generation/transaction boundary carefully. Runner ID only affects `String()` and registry key is supplied separately, so a temporary ID is acceptable for validation; build the final Runner again after create, or add a service input ID if clean.

### Pattern 2: Template functions before Parse

**What:** Add a `toJSON` function to the template FuncMap before parsing `body_template`.

**When to use:** both validation and rendering of HTTP `body_template`.

**Example:**

```go
// Source: Go text/template docs: New -> Funcs -> Parse -> Execute
func toJSON(v any) (string, error) {
    b, err := json.Marshal(v)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

tmpl, err := template.New("body").Funcs(template.FuncMap{
    "toJSON": toJSON,
}).Parse(bodyTemplate)
```

### Pattern 3: MCP `tools/call` params shape

**What:** MCP `tools/call` request params must be an object with `name` and `arguments`.

**When to use:** every `MCPRunner.Execute` call.

**Example:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "execute_task",
    "arguments": {
      "task_id": "task-1",
      "task_type": "feature",
      "workspace_path": "C:/work/task-1",
      "files_to_modify": ["src/main.go"],
      "prompt": "implement feature X"
    }
  }
}
```

### Pattern 4: Local fake-service e2e

**What:** Use `httptest.NewServer` to emulate HTTP and MCP services in Go tests.

**When to use:** runner execution tests and server API registration tests. Avoid real APIs in automated CI.

**Example:**

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    var rpc mcpRequest
    _ = json.NewDecoder(r.Body).Decode(&rpc)
    if rpc.Method == "tools/call" {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(mcpResponse{
            JSONRPC: "2.0",
            ID: rpc.ID,
            Result: json.RawMessage(`{"content":[{"type":"text","text":"ok"}],"isError":false}`),
        })
        return
    }
    w.WriteHeader(http.StatusBadRequest)
}))
defer srv.Close()
```

### Anti-Patterns to Avoid

- **Persist then best-effort register:** creates unusable Agent rows and violates D-01. Validate/build before DB mutation.
- **Object-shaped public `runner_config`:** docs and UI must keep `runner_config` as a serialized JSON string.
- **Accepting `transport:"sse"` in MCP config:** current code does not implement SSE lifecycle; reject it or tests will encode false support.
- **Template string interpolation without `toJSON`:** unsafe for JSON values and currently incompatible with examples.
- **Real API e2e in automated tests:** violates D-10 and creates flaky/costly validation.
- **Introducing MCP SDK mid-phase:** unnecessary for the locked HTTP/JSON-RPC subset and increases scope.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP fake services | External mock server binary | Go `httptest.NewServer` | Zero external deps, parallel-safe, easy request assertions. |
| JSON serialization in templates | Manual quote escaping | `toJSON` using `encoding/json.Marshal` | Handles quotes, arrays, maps, numbers, booleans safely. |
| Agent config validation path | Duplicate validation in each handler | `registry.BuildRunner` plus RunnerConfig `Validate()` | Keeps startup/load/API validation consistent. |
| MCP request encoding | String-concatenated JSON | Typed structs + `encoding/json` | JSON-RPC shapes are exact and testable. |
| API error shape | New response envelope | Existing `APIResponse{Success:false, Error:...}` | Maintains frontend `unwrap()` behavior. |
| Frontend state management | New store library | Existing TanStack Query mutations | Keeps AgentWorkbench simple and consistent. |

**Key insight:** The complex part is not making HTTP calls; it is keeping registration, DB persistence, live registry, docs, and UI aligned to one contract. Reuse the existing factory and API envelope rather than inventing parallel validation paths.

## Common Pitfalls

### Pitfall 1: Broken Agent row persisted after validation failure
**What goes wrong:** `POST` returns success even though registry build failed, or DB contains a non-runnable HTTP/MCP Agent.
**Why it happens:** current `handleCreateAgent` writes DB first, then logs and ignores `buildRunnerFromAgentView` failures.
**How to avoid:** validate/build before `orgSvc.CreateAgent`; add server tests that assert 400 and zero persisted Agents for invalid HTTP/MCP config.
**Warning signs:** log contains `failed to build runner` but HTTP status is 201.

### Pitfall 2: PATCH update leaves DB/registry inconsistent
**What goes wrong:** update persists invalid config; registry keeps old runner or silently skips new runner.
**Why it happens:** update writes DB first and re-registers only if build returns non-nil.
**How to avoid:** load existing Agent, compute effective runner type/config, validate/build before `UpdateAgent`; only register after successful save.
**Warning signs:** GET Agent shows invalid `runner_config`, capabilities endpoint still reflects old runner.

### Pitfall 3: `toJSON` parse failure at runtime
**What goes wrong:** templates from docs/UI fail with `function "toJSON" not defined`.
**Why it happens:** `template.New(...).Parse` is called without `Funcs`, or Funcs is called after Parse.
**How to avoid:** centralize template creation with `Funcs` before `Parse`; use same path in validation and render.
**Warning signs:** HTTPRunner tests with `{{.Context.prompt | toJSON}}` fail before request is sent.

### Pitfall 4: MCP server rejects `tools/call` params
**What goes wrong:** MCP fake/real server returns `Invalid params` or `Unknown tool`.
**Why it happens:** current code places `task_id`/`task_type` directly in `params` instead of `{name, arguments}`.
**How to avoid:** implement standard params shape and test that fake MCP server receives `params.name == tool_name` and arguments contains task payload.
**Warning signs:** JSON-RPC error `-32602 Invalid params`.

### Pitfall 5: False SSE support
**What goes wrong:** users configure `transport:"sse"`; registration succeeds but execution uses plain POST and later fails unpredictably.
**Why it happens:** validation currently allows `sse`, and comments claim support.
**How to avoid:** reject `sse` with a clear validation error in Phase 6; docs say SSE is deferred.
**Warning signs:** tests still have "valid sse transport" as a success case.

### Pitfall 6: Docs drift from API contract
**What goes wrong:** users copy onboarding curl with object `runner_config` or `agent_id` and registration fails or behaves differently than docs.
**Why it happens:** `docs/agent-onboarding.md` predates serialized `runner_config` decision and current `org.CreateAgentInput`.
**How to avoid:** update docs/examples after backend contract is stable; include copy-runnable curl using escaped JSON string.
**Warning signs:** docs mention `.Prompt`, `.TaskID`, default `60000`, or MCP SSE.

### Pitfall 7: Fake e2e tests accidentally only test direct Runner code
**What goes wrong:** Runner unit tests pass, but create/update API still persists invalid configs.
**Why it happens:** no server-level test drives org API registration.
**How to avoid:** include both direct Runner fake-service tests and org API tests for 400/no-persist behavior.
**Warning signs:** tests do not instantiate `Server.Handler()` for `/api/v1/orgs/.../agents`.

## Recommended Implementation Approach by Gap

### Gap A: API create/update validation

1. Add a server-level helper that validates a serialized `runner_config` using `registry.BuildRunner`.
2. On create:
   - Decode request body.
   - Determine `runner_type` default (`cli` if empty).
   - For `http`/`mcp`, validate config before persistence.
   - Return `400` with existing `APIResponse` on invalid JSON/config/template/build failure.
   - Persist only after validation passes.
   - Register a final built Runner after save.
3. On update:
   - Load existing Agent before mutation.
   - Compute effective runner_type and runner_config after applying patch.
   - Validate/build before persistence if effective type is HTTP/MCP or runner fields changed.
   - Save, then register the validated Runner.
4. Add tests:
   - Create HTTP with missing endpoint returns 400 and no Agent persisted.
   - Create HTTP with invalid template returns 400 and no Agent persisted.
   - Update existing valid HTTP Agent to invalid MCP config returns 400 and old Agent remains unchanged.

### Gap B: HTTP template helper and validation

1. Add `toJSON` helper in `internal/runner/http_runner.go` or new `http_config.go`.
2. Use shared `parseBodyTemplate` in both `Validate()` and `renderBody()`.
3. Validate URL is absolute with scheme `http` or `https` and non-empty host. Current `url.Parse` alone is too permissive for strings like `not a url`.
4. Optionally validate method against common HTTP verbs; default remains POST.
5. Add tests for:
   - `{{.Context.prompt | toJSON}}` preserves quotes/newlines.
   - `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace.Path`, `.Env`, `.Timeout` template access.
   - default body unchanged.

### Gap C: MCP tool_name and standard params

1. Extend `MCPRunnerConfig` with `ToolName string`.
2. Add method `toolName()` returning config value or `execute_task`.
3. Change `Execute()` params to:
   - `name`: tool name
   - `arguments`: map containing `task_id`, `task_type`, flattened context, optional `workspace_path`, `files_to_modify`, `command`, `shell`, maybe `env`.
4. Treat `result.isError == true` as `RunnerResult.Success=false` even without JSON-RPC error.
5. Add tests for custom tool_name and default tool_name.
6. Reject `transport:"sse"` in `Validate()`.

### Gap D: Local fake HTTP/MCP e2e

1. Put fake service tests in `internal/runner` for direct runner execution.
2. Put API validation/no-persist tests in `internal/server` using existing in-memory SQLite test setup.
3. If feasible, add an API-to-registry test by calling org create with a fake endpoint, then `runnerRegistry.Get(agentID).Execute()` or capabilities. Avoid relying on real network except `httptest` URL.
4. Keep e2e deterministic: no env API keys, no external services, no sleeping except existing polling patterns.

### Gap E: Docs and examples

1. Update `docs/examples/README.md` with exact serialized runner_config contract.
2. Provide copy-runnable curl pattern. Example with `jq` is portable on many dev machines but may not exist on Windows; include a PowerShell `Get-Content -Raw | ConvertTo-Json` equivalent if needed, or show inline escaped JSON.
3. Fix onboarding:
   - Remove `agent_id` field from create request unless API actually supports it.
   - `runner_config` value must be a string.
   - Template variables are `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout`.
   - HTTP timeout default is 300000ms.
   - MCP supports `transport:"http"` only in Phase 6 and `tool_name` defaults to `execute_task`.
4. Add MCP example JSON file if missing, e.g. `docs/examples/mcp-runner-local.json`.

## Code Examples

### HTTPRunner template helper

```go
// Source: Go text/template docs and Phase D-05
func toJSON(v any) (string, error) {
    b, err := json.Marshal(v)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

func parseBodyTemplate(bodyTemplate string) (*template.Template, error) {
    return template.New("body").Funcs(template.FuncMap{
        "toJSON": toJSON,
    }).Parse(bodyTemplate)
}
```

### MCP `tools/call` params

```go
// Source: MCP tools spec 2025-06-18
arguments := map[string]interface{}{
    "task_id":   task.ID,
    "task_type": task.Type,
}
for k, v := range task.Context {
    arguments[k] = v
}
params := map[string]interface{}{
    "name":      r.toolName(),
    "arguments": arguments,
}
paramsJSON, err := json.Marshal(params)
```

### Server validation before create

```go
// Source: current server APIResponse pattern and D-01
if input.RunnerType == "http" || input.RunnerType == "mcp" {
    if _, err := s.validateRunnerBuild("validation", input.Name, input.RunnerType, input.RunnerConfig); err != nil {
        s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
        return
    }
}
```

### Local fake MCP assertion

```go
// Source: Go httptest + official MCP tools/call shape
var params struct {
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}
if err := json.Unmarshal(req.Params, &params); err != nil {
    t.Fatalf("params decode: %v", err)
}
if params.Name != "execute_task" {
    t.Fatalf("tool name = %q", params.Name)
}
if params.Arguments["task_id"] != "task-1" {
    t.Fatalf("missing task_id argument")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| MCP HTTP+SSE transport | Streamable HTTP transport, with optional SSE streams as part of the Streamable HTTP endpoint | MCP 2025-06-18 spec says Streamable HTTP replaces 2024-11-05 HTTP+SSE | Do not implement or advertise old SSE transport in Phase 6. Reject `transport:"sse"` until lifecycle support is deliberately planned. |
| MCP `tools/call` params as arbitrary task fields | `params: { name, arguments }` | Current MCP tools spec | Must change `MCPRunner.Execute()` and tests. |
| Template interpolation for JSON | Custom FuncMap with `toJSON` before Parse | Go stdlib documented pattern | Required for safe JSON embedding and existing examples. |
| Real API acceptance tests | Local fake HTTP/MCP services | User decision D-10 | Automated phase gate must not require API keys, network, or cost. |

**Deprecated/outdated:**

- `transport: "sse"` as a Phase 6 accepted config: not implemented by current code and deferred by user decision.
- Onboarding `.Prompt` / `.TaskID` template variables: not present in actual `RunnerTask`; use `.Command`, `.ID`, `.Context.*`.
- Object-shaped `runner_config` in public API docs: violates locked serialized JSON string contract.
- `timeout_ms` default `60000` in docs: code default is 300000ms for HTTP/MCP.

## Open Questions

1. **Should API validation apply to CLI runner_config too?**
   - What we know: D-01 explicitly mentions HTTP/MCP. CLI config is looser and current BuildRunner ignores CLI unmarshal errors.
   - What's unclear: Whether user expects CLI create/update to start failing on invalid JSON in this phase.
   - Recommendation: Scope strict 400 behavior to HTTP/MCP. Preserve CLI compatibility unless planner explicitly adds safe validation.

2. **Should HTTPRunner validate rendered body is valid JSON?**
   - What we know: Runner sends `Content-Type: application/json`, examples are JSON, default body is JSON.
   - What's unclear: Whether custom HTTP agents might intentionally accept non-JSON body with a custom content type.
   - Recommendation: Validate template parses at registration; do not require rendered JSON because validation lacks a concrete RunnerTask and phase decisions only require construction/config validation.

3. **Should MCPRunner implement initialize/session before every tool call?**
   - What we know: current `HealthCheck()` sends `initialize`, `GetCapabilities()` sends `tools/list`, `Execute()` sends `tools/call`. Official Streamable HTTP includes session support when server returns `Mcp-Session-Id`.
   - What's unclear: Whether target MCP servers for this project require full session lifecycle.
   - Recommendation: Keep Phase 6 to stateless HTTP/JSON-RPC fake-compatible support. Document that full Streamable HTTP session/SSE lifecycle is deferred if needed.

4. **Where should e2e tests live?**
   - What we know: roadmap says `tests/e2e/runner_http_test.go`, current project has package-local tests and server tests.
   - What's unclear: Whether repository prefers top-level `tests/e2e` for future orchestration.
   - Recommendation: Use package-local Go tests for now (`internal/runner`, `internal/server`) because they can access unexported structs and existing fixtures. Planner can add top-level e2e only if existing harness supports it.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Backend implementation/tests | Yes | go1.26.2 windows/amd64 | None needed |
| Node.js | Frontend lint/build | Yes | v24.13.0 | None needed |
| npm | Frontend package scripts/version checks | Yes | 11.6.2 | None needed |
| SQLite driver | In-memory server tests | Yes | `modernc.org/sqlite` v1.37.1 in go.mod | None needed |
| External OpenAI/Anthropic APIs | Manual docs only | Not required | N/A | Local fake HTTP services for automated tests |
| MCP external server | Manual docs only | Not required | N/A | Local fake MCP JSON-RPC service for automated tests |
| jq | Optional docs convenience | Not checked | N/A | Provide inline JSON or PowerShell copy-runnable alternative |

**Missing dependencies with no fallback:**
- None for automated Phase 6 validation.

**Missing dependencies with fallback:**
- Real external AI APIs and MCP services are intentionally not required; use local fake services.
- `jq` should not be assumed for Windows copy-runnable docs unless an alternative is included.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Backend framework | Go `testing` + `httptest`, Go 1.26.2 installed |
| Backend config files | `go.mod`; package tests under `internal/runner`, `internal/registry`, `internal/server` |
| Frontend framework | TypeScript/React build + ESLint via package scripts |
| Frontend config files | `web/package.json`, `web/vite.config.ts`, ESLint config in web project |
| Quick run command | `go test ./internal/runner ./internal/registry ./internal/server` |
| Full suite command | `go test ./... && npm --prefix web run lint && npm --prefix web run build` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| HTTP-01 | HTTPRunner validates config and rejects missing endpoint/invalid template. | unit | `go test ./internal/runner -run 'TestHTTPRunnerConfig_Validate|TestHTTPRunner_TemplateToJSON' -count=1` | Existing file, new/updated tests needed |
| HTTP-01 | HTTPRunner renders `toJSON` and actual RunnerTask fields. | unit | `go test ./internal/runner -run TestHTTPRunner_Execute_TemplateBody -count=1` | Existing file, update needed |
| HTTP-01 | HTTPRunner default body stays `task_id`, `task_type`, flattened context. | unit | `go test ./internal/runner -run TestHTTPRunner_Execute_DefaultBody -count=1` | New test needed |
| HTTP-01 | Creating invalid HTTP Agent returns 400 and does not persist DB record. | integration | `go test ./internal/server -run TestHandleCreateAgent_InvalidHTTPRunnerConfigDoesNotPersist -count=1` | New test needed |
| HTTP-01 | Updating Agent to invalid HTTP config returns 400 and preserves old config/registry. | integration | `go test ./internal/server -run TestHandleUpdateAgent_InvalidRunnerConfigPreservesExisting -count=1` | New test needed |
| MCP-01 | MCP config supports `tool_name` default/custom and rejects `sse`. | unit | `go test ./internal/runner -run TestMCPRunnerConfig_Validate -count=1` | Existing file, update needed |
| MCP-01 | MCPRunner sends `tools/call` params `{name, arguments}`. | unit/e2e fake | `go test ./internal/runner -run TestMCPRunner_Execute -count=1` | Existing file, update needed |
| MCP-01 | MCP capability discovery uses local fake `tools/list`. | unit/e2e fake | `go test ./internal/runner -run TestMCPRunner_GetCapabilities_DiscoverTools -count=1` | New test needed |
| MCP-01 | Creating invalid MCP Agent returns 400 and does not persist DB record. | integration | `go test ./internal/server -run TestHandleCreateAgent_InvalidMCPRunnerConfigDoesNotPersist -count=1` | New test needed |
| HTTP-01/MCP-01 | AgentWorkbench form keeps serialized `runner_config`, exposes errors, includes MCP `tool_name`. | frontend validation | `npm --prefix web run lint && npm --prefix web run build` | Existing page, update needed |
| HTTP-01/MCP-01 | Docs/examples match API contract and are copy-runnable. | manual/doc review | `go test ./...` plus review of docs examples | Existing docs, update needed |

### Sampling Rate

- **Per task commit:** `go test ./internal/runner ./internal/registry ./internal/server`
- **Per UI task commit:** `npm --prefix web run lint && npm --prefix web run build`
- **Per wave merge:** `go test ./... && npm --prefix web run lint && npm --prefix web run build`
- **Phase gate:** Full suite green, plus manual review that `docs/examples/README.md`, example JSON files, and `docs/agent-onboarding.md` reflect the serialized `runner_config` contract.

### Wave 0 Gaps

- [ ] `internal/server` tests for invalid HTTP/MCP create/update 400 and no persistence.
- [ ] `internal/runner/http_runner_test.go` additions for `toJSON`, RunnerTask field access, and default body regression.
- [ ] `internal/runner/mcp_runner_test.go` updates for `{name, arguments}`, `tool_name`, and SSE rejection.
- [ ] Optional shared server helper for runner validation before persistence.
- [ ] Docs/examples copy-runnable verification checklist.

## Risks and Sequencing Advice for Planner

### Recommended sequence

1. **Backend validation first:** Fix create/update API pre-validation so invalid configs cannot persist. This is the highest-risk user decision.
2. **HTTP template contract:** Add `toJSON`, template parse validation, and tests. This unblocks existing examples and AgentWorkbench placeholder.
3. **MCP contract:** Add `tool_name`, standard params, SSE rejection, and tests. Existing tests must be deliberately rewritten to the new expected shape.
4. **Local fake e2e:** Add `httptest` HTTP and MCP services. Keep tests fast and hermetic.
5. **AgentWorkbench basic UX:** Add MCP tool_name, reduce HTTP-only field confusion, rely on backend error visibility.
6. **Docs/examples last:** Once code behavior is stable, reconcile docs and make copy-runnable examples match exact API behavior.
7. **Full validation:** Run Go suite and frontend lint/build.

### Risk matrix

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Pre-validating create needs final Agent ID but ID is generated in service | Medium | Medium | Validate with temporary ID, then build final Runner post-save; Runner ID does not affect config validity. |
| Update validation accidentally drops partial fields | Medium | High | Load existing Agent first and compute effective config before validation. Add preservation test. |
| `url.Parse` validation remains too permissive | Medium | Medium | Check `u.IsAbs()`, scheme `http/https`, and `u.Host != ""`. |
| MCP full Streamable HTTP session support expands scope | Medium | Medium | Keep Phase 6 to direct HTTP JSON-RPC request/response; document session/SSE deferred. |
| Docs use shell features not available on Windows | Medium | Low | Provide Windows-friendly PowerShell or inline JSON alternative. |
| Frontend stores auth token in local component state/password field | Low | Medium | Existing behavior; do not expand secret management in this phase. Keep env var placeholders in docs. |

## Sources

### Primary (HIGH confidence)

- Project code: `internal/runner/interface.go` — RunnerTask fields and Runner interface.
- Project code: `internal/runner/http_runner.go` and `internal/runner/http_runner_test.go` — current HTTPRunner implementation and tests.
- Project code: `internal/runner/mcp_runner.go` and `internal/runner/mcp_runner_test.go` — current MCPRunner implementation and tests.
- Project code: `internal/registry/registry.go` and `internal/registry/registry_test.go` — BuildRunner validation path.
- Project code: `internal/server/org_api.go` — create/update persistence-before-registration gap.
- Project code: `internal/org/service.go` — serialized runner_config storage/API view.
- Project code: `web/src/pages/AgentWorkbenchPage.tsx` and `web/src/api/orgApi.ts` — current frontend registration shape.
- Project docs: `docs/examples/README.md`, example JSON files, `docs/agent-onboarding.md` — doc/code drift.
- Official Go docs: https://pkg.go.dev/text/template — `Funcs` must be called before `Parse`; FuncMap return rules.
- Official MCP tools spec 2025-06-18: https://modelcontextprotocol.io/specification/2025-06-18/server/tools — `tools/call` params `{name, arguments}` and tool result shape.
- Official MCP transports spec 2025-06-18: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports — Streamable HTTP replaces older HTTP+SSE; POST/GET behavior and headers.
- JSON-RPC 2.0 spec: https://www.jsonrpc.org/specification — request, params, response, and error object semantics.

### Secondary (MEDIUM confidence)

- Package registry checks via `npm view` and `go list -m -versions` on 2026-05-13 for current/latest versions.

### Tertiary (LOW confidence)

- None. No unverified community sources were needed.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — existing code and package manifests define the stack; versions verified locally/registry where useful.
- Architecture: HIGH — phase is mostly modifying known files and existing patterns.
- Pitfalls: HIGH — gaps are directly observed in code and cross-checked against locked decisions and official MCP/template specs.
- MCP full lifecycle details: MEDIUM — official spec is clear, but phase intentionally implements only the locked HTTP/JSON-RPC subset.

**Research date:** 2026-05-13
**Valid until:** 2026-06-12 for project-code findings; 2026-05-20 for MCP spec/current ecosystem assumptions.
