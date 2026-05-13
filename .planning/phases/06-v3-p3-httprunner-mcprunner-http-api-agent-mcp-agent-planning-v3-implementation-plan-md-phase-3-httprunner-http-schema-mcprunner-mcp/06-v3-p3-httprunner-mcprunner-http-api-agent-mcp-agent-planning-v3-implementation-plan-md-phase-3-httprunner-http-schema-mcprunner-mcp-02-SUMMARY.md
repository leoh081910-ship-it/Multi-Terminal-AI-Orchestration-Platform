## Plan 02: MCPRunner Phase 6 hardening

**Status:** Complete

**Changes:**
- `internal/runner/mcp_runner.go`: Added `ToolName` field (`json:"tool_name,omitempty"`), `toolName()` helper defaulting to `execute_task`, `applyHeaders` helper, strict Validate (http-only transport, rejects SSE with Phase 6 message, endpoint scheme/host, whitespace tool_name), Execute sends `{name, arguments}` params shape.
- `internal/runner/mcp_runner_test.go`: Removed unused `os` import, existing tests already covered tool_name/arguments shape/transport rejection.

**Verification:** `go test ./internal/runner -count=1` exits 0.
