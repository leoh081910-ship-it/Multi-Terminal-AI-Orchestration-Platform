---
phase: 06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/runner/http_runner.go
  - internal/runner/http_runner_test.go
autonomous: true
requirements:
  - HTTP-01
must_haves:
  truths:
    - "HTTPRunner hardens the existing canonical Runner implementation; it does not create a parallel Runner interface or new http.go/runner.go path."
    - "HTTP runner configs are validated locally without outbound network calls before registry/API persistence can rely on them."
    - "HTTP body templates support `toJSON` before Parse and render only the real `RunnerTask` fields from `internal/runner/interface.go`."
    - "The default HTTP payload remains `task_id`, `task_type`, plus flattened `RunnerTask.Context`."
  artifacts:
    - path: "internal/runner/http_runner.go"
      provides: "canonical HTTPRunner config validation, template parsing, and execution behavior"
      contains: "type HTTPRunnerConfig struct"
    - path: "internal/runner/http_runner_test.go"
      provides: "httptest-only HTTPRunner contract regressions"
      contains: "httptest.NewServer"
  key_links:
    - from: "internal/runner/http_runner.go"
      to: "internal/registry/registry.go"
      via: "HTTPRunnerConfig.Validate is called by registry.BuildRunner before API persistence validation"
      pattern: "cfg.Validate()"
    - from: "internal/runner/http_runner.go"
      to: "docs/examples and AgentWorkbench body_template examples"
      via: "toJSON registered in template FuncMap before Parse"
      pattern: "Funcs(template.FuncMap"
---

<objective>
Harden the existing canonical HTTPRunner implementation for Phase 6.

Purpose: HTTP agents must have a locally validatable configuration and a stable request-body contract before API create/update can safely persist and register them.
Output: stricter `HTTPRunnerConfig.Validate`, `toJSON` template support, official RunnerTask field rendering, default payload regression coverage, and local fake-service tests.
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
@internal/runner/http_runner.go
@internal/runner/http_runner_test.go
@internal/registry/registry.go

<interfaces>
Do not create `internal/runner/runner.go`, `internal/runner/http.go`, or a `Runner.Run(...)` interface. Use the existing contract in `internal/runner/interface.go`:
- `Runner.Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error)`
- `RunnerTask.ID`, `Type`, `Workspace`, `Command`, `Shell`, `FilesToModify`, `Env`, `Context`, `Timeout`
- `HTTPRunnerConfig` in `internal/runner/http_runner.go`

Official HTTP template variables are the existing `RunnerTask` fields: `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, `.Timeout`. Do not add `.Prompt`, `.TaskID`, `prompt`, or `taskID` aliases.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Strengthen canonical HTTPRunner config validation</name>
  <files>internal/runner/http_runner.go, internal/runner/http_runner_test.go</files>
  <read_first>
    <file>internal/runner/interface.go</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/http_runner_test.go</file>
    <file>internal/registry/registry.go</file>
  </read_first>
  <behavior>
    - Test 1: `HTTPRunnerConfig{Endpoint:""}.Validate()` returns an error containing `endpoint is required`.
    - Test 2: hostless or relative endpoints such as `not-a-url` or `/run` return an error containing `endpoint scheme must be http or https` or `endpoint host is required`.
    - Test 3: non-http schemes such as `ftp://example.test/run` return an error containing `endpoint scheme must be http or https`.
    - Test 4: unsupported methods such as `TRACE` return an error containing `method must be one of GET, POST, PUT, PATCH`.
    - Test 5: malformed `body_template` returns an error during `Validate()` before any HTTP request can be made.
    - Test 6: `body_template` using `toJSON`, such as `{{.Context.prompt | toJSON}}`, parses successfully during `Validate()`.
  </behavior>
  <action>
    Update `HTTPRunnerConfig.Validate()` in `internal/runner/http_runner.go` only. Keep the existing `HTTPRunnerConfig` type name, `NewHTTPRunner(id, name string, config HTTPRunnerConfig) *HTTPRunner`, and `Execute(...)` signature.

    Validation rules:
    - Trim and parse `Endpoint`.
    - Require a non-empty endpoint.
    - Require an absolute URL with scheme exactly `http` or `https` and non-empty host.
    - Default empty method to `POST`; otherwise uppercase it for validation and accept only `GET`, `POST`, `PUT`, `PATCH`.
    - Preserve the existing timeout field (`TimeoutMs`) and reject negative values.
    - If `BodyTemplate` is non-empty, parse it with the same FuncMap used at execution time, including `toJSON`, before returning success.
    - Do not perform health checks, HEAD/GET/POST calls, tools/list, sleeps, or any outbound network validation.

    Add or update tests in `internal/runner/http_runner_test.go` under the existing HTTPRunner test style. Prefer extending `TestHTTPRunnerConfig_Validate` and adding a focused `TestHTTPRunner_TemplateToJSON` if clearer. Tests must not depend on external services.
  </action>
  <verify>
    <automated>go test ./internal/runner -run 'TestHTTPRunnerConfig_Validate|TestHTTPRunner_TemplateToJSON' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/runner/http_runner.go` still contains `type HTTPRunnerConfig struct`.
    - `internal/runner/http_runner.go` does not introduce `func (r *HTTPRunner) Run(`.
    - `internal/runner/http_runner.go` contains `endpoint scheme must be http or https`.
    - `internal/runner/http_runner.go` contains `method must be one of GET, POST, PUT, PATCH`.
    - `internal/runner/http_runner.go` parses `BodyTemplate` with a template FuncMap containing `toJSON` during validation.
    - `go test ./internal/runner -run 'TestHTTPRunnerConfig_Validate|TestHTTPRunner_TemplateToJSON' -count=1` exits 0.
  </acceptance_criteria>
  <done>HTTPRunner configs fail fast locally for invalid endpoint/method/template values and still require no network availability.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Lock HTTPRunner payload rendering and toJSON execution behavior</name>
  <files>internal/runner/http_runner.go, internal/runner/http_runner_test.go</files>
  <read_first>
    <file>internal/runner/interface.go</file>
    <file>internal/runner/http_runner.go</file>
    <file>internal/runner/http_runner_test.go</file>
  </read_first>
  <behavior>
    - Test 1: with empty `BodyTemplate`, an `httptest.NewServer` handler receives JSON with top-level `task_id`, `task_type`, and flattened `RunnerTask.Context` keys.
    - Test 2: with `BodyTemplate` containing `{{.Context.payload | toJSON}}`, the fake server receives valid JSON preserving nested strings, arrays, numbers, booleans, and objects.
    - Test 3: a template can render the official fields `.ID`, `.Type`, `.Command`, `.Shell`, `.FilesToModify`, `.Workspace`, `.Env`, `.Context.*`, and `.Timeout` from the existing `RunnerTask` shape.
    - Test 4: no test or implementation relies on `.Prompt` or `.TaskID` aliases.
  </behavior>
  <action>
    Update `internal/runner/http_runner.go` to use a shared template creation path for validation and execution:
    - Add `toJSON(value any) (string, error)` using `encoding/json.Marshal`.
    - Add or reuse a helper that calls `template.New("body").Funcs(template.FuncMap{"toJSON": toJSON}).Parse(...)` before any Execute-time render.
    - Keep the default payload behavior currently present in `renderBody`: `task_id`, `task_type`, then flatten `task.Context` into the top-level map.
    - Keep the existing HTTP execution behavior, output extraction, env expansion, and Runner interface implementation.

    Update `internal/runner/http_runner_test.go` using `httptest.NewServer` only. Server handlers should decode the actual request body and assert request method, headers where relevant, and body shape. Use the existing `RunnerTask` struct; if testing workspace, use the real `Workspace` field shape from `interface.go` instead of inventing a string-only workspace contract.
  </action>
  <verify>
    <automated>go test ./internal/runner -run 'TestHTTPRunner_Execute|TestHTTPRunner_TemplateToJSON|TestHTTPRunner_Execute_DefaultBody' -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `internal/runner/http_runner.go` contains `template.FuncMap{"toJSON": toJSON}` or an equivalent FuncMap assignment before `Parse`.
    - `internal/runner/http_runner.go` contains `"task_id"` and `"task_type"` in the default payload path.
    - `internal/runner/http_runner.go` flattens `RunnerTask.Context` into the default payload.
    - `internal/runner/http_runner_test.go` contains `httptest.NewServer`.
    - `internal/runner/http_runner_test.go` contains `toJSON`.
    - `internal/runner/http_runner_test.go` does not contain `.Prompt` or `.TaskID` as supported template variables.
    - `go test ./internal/runner -run 'TestHTTPRunner_Execute|TestHTTPRunner_TemplateToJSON|TestHTTPRunner_Execute_DefaultBody' -count=1` exits 0.
  </acceptance_criteria>
  <done>HTTPRunner request rendering is covered with local fake services, supports `toJSON`, and preserves the locked default payload contract.</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/runner -run 'TestHTTPRunnerConfig_Validate|TestHTTPRunner_Execute|TestHTTPRunner_TemplateToJSON|TestHTTPRunner_Execute_DefaultBody' -count=1
```
</verification>

<success_criteria>
- Existing canonical `HTTPRunner` is hardened; no parallel runner files or interfaces are introduced.
- Invalid HTTP configs and invalid templates fail local validation before API persistence can depend on them.
- HTTP templates support `toJSON` and official `RunnerTask` fields only.
- Default HTTP payload remains `task_id`, `task_type`, plus flattened context.
</success_criteria>

<output>
After completion, create `.planning/phases/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp/06-v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp-01-SUMMARY.md`
</output>
