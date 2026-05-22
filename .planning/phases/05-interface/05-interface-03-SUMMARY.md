---
phase: 05-interface
plan: "03"
subsystem: ui
tags: [react, vite, tanstack-query, websocket, scheduler, go, chi]
requires:
  - phase: 05-interface
    provides: Phase 05 Plan 01 shell/navigation and Phase 05 Plan 02 Wave/Event APIs
provides:
  - Scheduler board wave and dispatch_ref filtering with explicit sort modes
  - Task Card create/edit controls for dispatch_ref, wave, depends_on, and acceptance_criteria
  - Formal project-scoped single-task retrieval endpoint and detail page consumption
  - Task detail events/status history tab and execution summary above logs
  - WebSocket-driven board/detail/event refresh with polling fallback
  - Frontend support for paginated scheduler task list responses
affects: [interface, scheduler, realtime, task-detail]
tech-stack:
  added: []
  patterns:
    - WebSocket messages invalidate TanStack Query caches; polling remains fallback when disconnected
    - Project-scoped scheduler APIs are used for task list/detail/event reads
key-files:
  created:
    - .planning/phases/05-interface/05-interface-03-SUMMARY.md
  modified:
    - web/src/types/scheduler.ts
    - web/src/api/schedulerApi.ts
    - web/src/pages/SchedulerBoard.tsx
    - web/src/pages/TaskDetailPage.tsx
    - web/src/pages/EventLogPage.tsx
    - internal/server/server.go
    - internal/server/compat_scheduler.go
key-decisions:
  - "Use the smallest formal single-task compatibility endpoint instead of deriving detail data from getTasks().find(...)."
  - "Keep WebSocket as the primary refresh signal and retain polling only as disconnected fallback on core UI pages."
  - "Preserve the existing board/detail structure and add fields, filters, validation, and tabs surgically."
patterns-established:
  - "Scheduler task list responses tolerate both legacy arrays and paginated {items,total,limit,offset} envelopes."
  - "Task event WebSocket messages both append local event rows and invalidate canonical query caches."
requirements-completed: [API-02, API-03, UI-01, UI-02, UI-04, UI-06]
duration: 42min
completed: 2026-05-19
---

# Phase 05 Plan 03: Interface Task Control Surface Summary

**Scheduler board/detail task controls with Wave-aware filtering, formal single-task retrieval, and WebSocket-driven realtime refresh.**

## Performance

- **Duration:** 42 min
- **Started:** 2026-05-19T09:52:00Z
- **Completed:** 2026-05-19T10:34:39Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- Added board-level `wave` and `dispatch_ref` filtering, explicit sort modes, and visible task scheduling metadata.
- Expanded Task Card create/edit controls for `dispatch_ref`, `wave`, `depends_on`, and `acceptance_criteria` with basic boundary/format validation.
- Added a project-scoped single-task backend route and moved the detail page away from indirect `getTasks().find(...)` lookup.
- Added task detail event/status history display plus execution summary above logs while preserving lineage and agent-call tabs.
- Unified realtime behavior across board, task detail, and event log via WebSocket-triggered cache invalidation or event append, with polling retained as fallback.

## Task Commits

Each task was committed atomically:

1. **Task 1: 提升任务列表与 Task Card 表单覆盖面** - `0a8bfba` (feat)
2. **Task 2: 补强任务详情为正式详情页** - `07cf616` (feat)
3. **Task 3: 统一实时刷新策略** - `a83920a` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified

- `web/src/types/scheduler.ts` - Added scheduler grouping fields to task and mutation input types.
- `web/src/api/schedulerApi.ts` - Added paginated task-list support and formal `getTask` retrieval.
- `web/src/pages/SchedulerBoard.tsx` - Added filters, sort modes, form fields, validation, task metadata, and WebSocket/fallback refresh behavior.
- `web/src/pages/TaskDetailPage.tsx` - Added formal task retrieval, events tab, execution summary, and expanded Task Card fields.
- `web/src/pages/EventLogPage.tsx` - Added WebSocket invalidation alongside local event append and disconnected polling fallback.
- `internal/server/server.go` - Registered the single-task scheduler compatibility route.
- `internal/server/compat_scheduler.go` - Returned dispatch/wave/topo fields and implemented single-task compatibility lookup.

## Decisions Made

- Used the smallest backend extension, `GET /projects/{project}/scheduler/tasks/{id}`, instead of adding a new service layer.
- Preserved existing board/detail layouts and enhanced them in place to avoid regressing Phase 05 Plan 01/02 UI work.
- Treated WebSocket as the canonical realtime signal while retaining slower polling only when realtime is unavailable.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed scheduler task list response shape mismatch**
- **Found during:** Task 1 (提升任务列表与 Task Card 表单覆盖面)
- **Issue:** The backend scheduler list endpoint returns `{ items, total, limit, offset }`, but the frontend assumed a raw array and called `.map` directly.
- **Fix:** Added `SchedulerTaskListResponse` and made `getTasks` handle both paginated and legacy array responses.
- **Files modified:** `web/src/api/schedulerApi.ts`
- **Verification:** `npm --prefix "E:/04-Claude/Projects/多终端 AI 编排平台/web" run build`
- **Committed in:** `0a8bfba`

**2. [Rule 2 - Missing Critical] Added formal single-task retrieval support**
- **Found during:** Task 2 (补强任务详情为正式详情页)
- **Issue:** The detail page required direct task retrieval, but the compatibility API did not expose a single-task route.
- **Fix:** Added `GET /tasks/{id}` under project-scoped scheduler routes and wired `schedulerApi.getTask` to it.
- **Files modified:** `internal/server/server.go`, `internal/server/compat_scheduler.go`, `web/src/api/schedulerApi.ts`, `web/src/pages/TaskDetailPage.tsx`
- **Verification:** `go test ./internal/server ./internal/store -count=1` and frontend build
- **Committed in:** `07cf616`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical functionality)
**Impact on plan:** Both fixes were required for the planned formal board/detail behavior and did not add unrelated scope.

## Issues Encountered

- TypeScript and Go verification passed after implementation.
- Stub scan found placeholder/example text in existing UI copy and empty collection literals in non-stub control flow; no goal-blocking UI stubs were introduced.

## Known Stubs

None that block the plan goal.

## Verification

- `npm --prefix "E:/04-Claude/Projects/多终端 AI 编排平台/web" run build` passed.
- `go test ./internal/server ./internal/store -count=1` passed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 05 interface task control surface is ready for verification. Board/detail/event pages now share predictable WebSocket-driven refresh behavior and formal scheduler API usage.

## Self-Check: PASSED

- Verified modified source files exist.
- Verified task commits exist: `0a8bfba`, `07cf616`, `a83920a`.
- Verified frontend build and backend tests pass.

---
*Phase: 05-interface*
*Completed: 2026-05-19*
