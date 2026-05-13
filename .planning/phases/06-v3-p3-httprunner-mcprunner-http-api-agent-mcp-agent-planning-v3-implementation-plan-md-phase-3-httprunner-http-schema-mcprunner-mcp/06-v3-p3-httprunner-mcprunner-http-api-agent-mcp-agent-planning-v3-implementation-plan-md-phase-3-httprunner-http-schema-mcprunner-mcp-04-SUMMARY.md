## Plan 04: AgentWorkbench frontend Phase 6 alignment

**Status:** Complete

**Changes:**
- `web/src/api/orgApi.ts`: Already correct — `runner_config?: string`, org agent routes. No changes needed.
- `web/src/pages/AgentWorkbenchPage.tsx`: Added `mcpToolName` state defaulting to `execute_task`, MCP tool_name input field, read-only config preview showing the serialized runner_config that will be sent, `queryClient.invalidateQueries` on mutation success.

**Verification:** `npm --prefix web run lint && npm --prefix web run build` exits 0.
