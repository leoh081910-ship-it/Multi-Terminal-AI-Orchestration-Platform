## Plan 03: Org Agent API pre-persistence runner validation

**Status:** Complete

**Changes:**
- `internal/server/org_api.go`: Added `validateRunnerBuild` helper using `registry.BuildRunner`. `handleCreateAgent` validates HTTP/MCP config before `CreateAgent` call. `handleUpdateAgent` loads existing agent, computes effective post-update config, validates before `UpdateAgent` call. Invalid config returns 400 without mutating DB or registry.
- `internal/server/server_test.go`: Added `createTestOrg` helper and 5 tests: invalid HTTP create, invalid MCP SSE create, valid unreachable endpoint create, invalid update preserves existing config, valid HTTP-to-MCP update.

**Verification:** `go test ./internal/registry ./internal/server -count=1` exits 0.
