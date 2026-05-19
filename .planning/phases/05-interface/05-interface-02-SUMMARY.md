---
phase: 05-interface
plan: "02"
subsystem: ui
tags: [react, tanstack-query, websocket, scheduler, waves, events, go, chi]

requires:
  - phase: 03-execution-layer
    provides: scheduler task, event, and wave persistence plus project-scoped compatibility API surface
  - phase: 05-interface
    provides: formal dashboard layout and project selection context
provides:
  - Project-scoped Wave list/detail/seal API coverage and frontend client methods
  - Formal Wave management page with wave status, task counts, seal action, and navigation to board/events
  - Formal Event log page with task/dispatch filtering, historical query, and realtime WebSocket append
affects: [frontend, scheduler-api, event-observability, wave-management]

tech-stack:
  added: []
  patterns:
    - Project-aware TanStack Query keys for Wave and Event pages
    - HTTP history plus WebSocket append pattern for event observability

key-files:
  created: []
  modified:
    - internal/server/compat_wave_helpers.go
    - internal/server/server_test.go
    - web/src/types/scheduler.ts
    - web/src/api/schedulerApi.ts
    - web/src/pages/WaveManagementPage.tsx
    - web/src/pages/EventLogPage.tsx

key-decisions:
  - "Use the formal /api/v1/projects/{projectId}/scheduler routes for Wave and Event UI data instead of legacy dispatch-only surfaces."
  - "Keep realtime event append client-side by normalizing WebSocket messages into SchedulerEvent records and deduplicating against HTTP history."

patterns-established:
  - "Wave pages select a row, query detail by dispatch_ref and wave, then invalidate list/detail queries after seal."
  - "Event pages preserve filters in URL query params so Wave/task navigation lands on filtered history."

requirements-completed: [API-01, UI-03, UI-07]

duration: 5min
completed: 2026-05-19T09:54:36Z
---

# Phase 05-interface Plan 02: Wave and Event UI Summary

**Project-scoped Wave management and Event log UI backed by formal scheduler APIs and realtime WebSocket append**

## Performance

- **Duration:** 5 min in this continuation window; prior executor work completed Tasks 1-2 before compaction.
- **Started:** 2026-05-19T09:49:33Z
- **Completed:** 2026-05-19T09:54:36Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Added regression coverage for project-scoped Wave and Event API behavior, including wave URL param parsing, per-wave task counts, `sealed_at`, project isolation, and task/dispatch filters.
- Extended the frontend scheduler API and scheduler types with formal project-scoped Wave and Event methods.
- Replaced placeholder Wave and Event pages with working React/TanStack Query implementations for list/detail/seal, historical event filters, and realtime append.

## Task Commits

Each task was committed atomically:

1. **Task 1: 为正式 UI 补齐项目化 Wave / 事件 API** - `c0bcf4e` (fix)
2. **Task 2: 扩展 schedulerApi 为正式 Wave / 事件访问层** - `8c4f703` (feat)
3. **Task 3: 实现正式 Wave 管理页与事件日志页** - `1f44cb1` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified

- `internal/server/compat_wave_helpers.go` - Fixed project route wave param parsing, per-wave counts, and sealed timestamp mapping.
- `internal/server/server_test.go` - Added regression tests for project-scoped Wave routes and Event filters.
- `web/src/types/scheduler.ts` - Added `SchedulerWave` and `SchedulerEvent` frontend types.
- `web/src/api/schedulerApi.ts` - Added Wave list/detail/seal and Event history methods under project-scoped scheduler routes.
- `web/src/pages/WaveManagementPage.tsx` - Implemented Wave list/detail UI, seal action, status/count display, and board/event navigation.
- `web/src/pages/EventLogPage.tsx` - Implemented task/dispatch filters, historical Event query, WebSocket append, dedupe, and task/detail links.

## Decisions Made

- Used existing project-scoped compatibility endpoints instead of adding duplicate backend routes; the plan goal was satisfied by regression tests plus targeted fixes to discovered gaps.
- Event realtime append is normalized from the generic WebSocket envelope into `SchedulerEvent`, then merged with HTTP history by `event_id`.
- Filter state for the Event page is stored in URL query parameters so navigation from Wave management can land directly on a filtered dispatch event chain.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Wave route parameter parsing**
- **Found during:** Task 1 (project-scoped Wave API coverage)
- **Issue:** The project-scoped route uses `{wave}`, but `parseCompatWaveParam` only read `waveNum`, causing Wave detail/seal calls to return 404.
- **Fix:** Read `wave` first, then `waveNum`, then the query parameter fallback.
- **Files modified:** `internal/server/compat_wave_helpers.go`
- **Verification:** `go test ./internal/server ./internal/store -count=1`
- **Committed in:** `c0bcf4e`

**2. [Rule 1 - Bug] Fixed Wave summary counts and sealed timestamp**
- **Found during:** Task 1 (project-scoped Wave API coverage)
- **Issue:** Wave summaries counted all tasks passed to the helper rather than only tasks matching the wave row's dispatch ref and wave number, and sealed responses omitted `sealed_at`.
- **Fix:** Filtered summary counts by dispatch ref and wave, computed task count from filtered counts, and populated `sealed_at` when present.
- **Files modified:** `internal/server/compat_wave_helpers.go`
- **Verification:** `go test ./internal/server ./internal/store -count=1`
- **Committed in:** `c0bcf4e`

**3. [Rule 3 - Blocking] Restored dropped frontend type imports**
- **Found during:** Task 2 frontend build verification
- **Issue:** Adding Wave/Event imports accidentally removed existing `ScheduledTask` and `RuntimeSummary` imports, causing TypeScript build errors.
- **Fix:** Restored both imports in `schedulerApi.ts`.
- **Files modified:** `web/src/api/schedulerApi.ts`
- **Verification:** `npm run build`
- **Committed in:** `8c4f703`

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking issue)
**Impact on plan:** All fixes were required for correctness and did not add scope beyond project-scoped Wave/Event UI delivery.

## Issues Encountered

- Creating test tasks alone did not create Wave rows; Task 1 tests explicitly call `UpsertWave` to seed the rows the Wave API is expected to list.
- Frontend placeholders existed for `/waves` and `/events`; Task 3 replaced them with functional pages wired to the formal API and WebSocket hook.

## Verification

- `go test ./internal/server ./internal/store -count=1` passed.
- `npm --prefix "E:/04-Claude/Projects/多终端 AI 编排平台/web" run build` passed.

## Known Stubs

None. Placeholder Wave/Event page text was removed and replaced with data-backed UI. Input placeholders such as `TASK-001` and `dispatch-2026...` are field examples only and do not flow to rendering as mock data.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Wave management and event observability now have formal project-scoped browser entries.
- Future UI phases can link to `/waves`, `/events?dispatch_ref=...`, and `/events?task_id=...` without relying on legacy dispatch screens.

## Self-Check: PASSED

- Found all key files created or modified by this plan.
- Found task commits `c0bcf4e`, `8c4f703`, and `1f44cb1` in git history.

---
*Phase: 05-interface*
*Completed: 2026-05-19*
