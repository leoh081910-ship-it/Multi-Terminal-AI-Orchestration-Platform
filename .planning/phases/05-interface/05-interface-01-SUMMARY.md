---
phase: 05-interface
plan: "01"
subsystem: ui
tags: [react, vite, tanstack-query, routing, navigation]

requires:
  - phase: runtime-reliability
    provides: existing React shell, scheduler board, stats API, runtime health data
provides:
  - Formal Phase 5 core routes for overview, board, task detail, Wave management, and event log
  - Sidebar navigation split around the four core control entries
  - Overview homepage with task stats, runtime health, pending items, recent updates, and active dispatch/Wave snapshot
affects: [05-interface, web-ui, scheduler-board, wave-management, event-log]

tech-stack:
  added: []
  patterns:
    - React Router route map remains inside App.tsx
    - Homepage reuses schedulerApi.getStats rather than introducing a parallel data source

key-files:
  created:
    - .planning/phases/05-interface/05-interface-01-SUMMARY.md
  modified:
    - web/src/App.tsx
    - web/src/components/Layout.tsx
    - web/src/pages/OrchestratorHome.tsx

key-decisions:
  - "Keep the formal route map in App.tsx while preserving real Wave, Event, and Task Detail page components already landed by follow-up interface work."
  - "Keep homepage data on schedulerApi.getStats to avoid creating a second dashboard API path."
  - "Leave SchedulerBoard task creation, bulk import, and editing flows out of the homepage so /board remains the task progression surface."

patterns-established:
  - "Core IA routes are exposed by App.tsx and mirrored in Layout.tsx primary navigation."
  - "Overview-only homepage cards derive current focus, pending items, and active dispatch snapshots from existing scheduler stats."

requirements-completed: [UI-05, UI-01]

completed: 2026-05-19
---

# Phase 05 Plan 01: Interface IA and Overview Shell Summary

React control-console IA with formal core routes, primary navigation for board/Wave/events, and a dashboard-only homepage backed by existing scheduler stats.

## Accomplishments

- Added formal Phase 5 route entries for `/`, `/board`, `/tasks/:taskId`, `/waves`, and `/events` without changing routing frameworks.
- Promoted Wave management and event log into the sidebar's core navigation while preserving project switching, project creation, and extension routes.
- Refocused the homepage on overview concerns: task totals, active sessions, queued work, pending items, runtime health, recent updates, recent completions, current focus, and active dispatch/Wave snapshot.

## Task Commits

Plan 01 was integrated from an isolated worktree by selectively cherry-picking safe commits because the worktree branch was based on stale history.

1. **Task 1: 重组正式路由并引入核心入口** - skipped during integration because main already had the formal route map with real `/waves`, `/events`, and `/tasks/:taskId` pages from Plan 02.
2. **Task 2: 收敛 Layout 导航为核心区 + 次级区** - `6891fd5` (feat)
3. **Task 3: 让首页只承担总览职责** - `881a954` (feat)

## Files Created/Modified

- `web/src/App.tsx` - Already contains the formal core route map and real page components for board, Wave management, event log, and task detail.
- `web/src/components/Layout.tsx` - Updates sidebar navigation to core entries plus extension views.
- `web/src/pages/OrchestratorHome.tsx` - Keeps homepage as an overview page using existing stats data for pending items and active dispatch/Wave snapshot.
- `.planning/phases/05-interface/05-interface-01-SUMMARY.md` - Records Plan 01 completion after main-branch integration.

## Decisions Made

- Preserved main's real `WaveManagementPage`, `EventLogPage`, and `TaskDetailPage` routes instead of reintroducing the isolated worktree's placeholders.
- Kept the homepage on `schedulerApi.getStats` so the overview reuses the existing board summary, runtime, and execution data rather than adding a new data path.
- Did not edit `SchedulerBoard.tsx`; board-specific task creation, bulk import, editing, dispatch, retry, and execution timeline flows remain on `/board`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Skipped stale route placeholder commit during main integration**
- **Found during:** selective cherry-pick into main
- **Issue:** The isolated Plan 01 route commit added placeholders for `/waves`, `/events`, and `/tasks/:taskId`, but main already had real pages from Plan 02.
- **Fix:** Kept main's real route map and skipped the empty cherry-pick.
- **Files modified:** none from that commit

**2. [Rule 3 - Blocking] Resolved Layout navigation conflict against newer route set**
- **Found during:** cherry-pick of navigation commit
- **Issue:** Plan 01 grouped the sidebar, while main already included additional extension routes and real Wave/Event entries.
- **Fix:** Applied the core/extension IA split while preserving every current route entry.
- **Files modified:** `web/src/components/Layout.tsx`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Integration preserved the intended IA and overview behavior while avoiding regression from real pages back to placeholders.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None in the integrated main state for this plan. Wave management, event log, and task detail route entries point to real page components.

## Verification

- Pending final main-branch frontend build after integration.

## Next Phase Readiness

- Phase 5 now has formal core navigation and overview IA integrated with the functional Wave/Event pages from Plan 02.
- `/board` remains available for task progression workflows without homepage overload.

## Self-Check: PASSED

- Found `web/src/App.tsx`
- Found `web/src/components/Layout.tsx`
- Found `web/src/pages/OrchestratorHome.tsx`
- Found `.planning/phases/05-interface/05-interface-01-SUMMARY.md`

---
*Phase: 05-interface*
*Completed: 2026-05-19*
