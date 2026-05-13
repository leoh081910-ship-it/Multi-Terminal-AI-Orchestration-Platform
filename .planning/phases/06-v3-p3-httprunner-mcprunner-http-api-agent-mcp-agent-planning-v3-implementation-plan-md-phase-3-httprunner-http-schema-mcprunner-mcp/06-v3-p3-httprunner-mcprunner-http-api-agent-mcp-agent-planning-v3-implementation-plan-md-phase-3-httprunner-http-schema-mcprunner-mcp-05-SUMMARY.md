## Plan 05: Docs/examples alignment with Phase 6 contract

**Status:** Complete

**Changes:**
- `docs/examples/README.md`: Updated template variables table (added `.Command`, `.Shell`, `toJSON`, `index .Env`, `.Timeout`), updated MCP section with `tool_name`, Phase 6 HTTP-only note, reference to new example file.
- `docs/examples/mcp-runner-local.json`: New file with complete MCP runner_config example including `tool_name`.
- `docs/agent-onboarding.md`: Rewritten to reflect serialized `runner_config` string contract, correct template variables (no `.Prompt`/`.TaskID`), MCP HTTP-only transport, `tool_name` defaults, validation error behavior, `httptest` testing.

**Verification:** Manual review against Phase 6 truths.
