## Plan 01: HTTPRunner hardened config validation and template support

**Status:** Complete

**Changes:**
- `internal/runner/http_runner.go`: Added strict `Validate()` (endpoint scheme/host, method whitelist, template parse, timeout sign), `toJSON` template func, `newBodyTemplate` helper, default body payload with `task_id`/`task_type` + flattened Context.
- `internal/runner/http_runner_test.go`: Extended `TestHTTPRunnerConfig_Validate`, added `TestHTTPRunner_TemplateToJSON`, rewrote `TestHTTPRunner_Execute_TemplateBody`, added `TestHTTPRunner_Execute_DefaultBody`.

**Verification:** `go test ./internal/runner -count=1` exits 0.
