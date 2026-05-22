# Phase 5: Interface - Context

**Gathered:** 2026-05-16
**Status:** Ready for planning

<domain>
## Phase Boundary

交付平台正式控制界面：HTTP API server、WebSocket 实时状态推送，以及 React Web UI 中围绕任务 CRUD、任务详情、Wave 管理、事件日志和实时状态更新的核心控制流。范围内是把现有 API/UI 基础收敛成正式产品入口，不新增超出 roadmap 的新能力。

</domain>

<decisions>
## Implementation Decisions

### 信息架构
- **D-01:** 正式产品采用“控制台优先”信息架构，而不是纯任务中心或纯运营面板。保留一个总览首页，再进入各核心控制页面。
- **D-02:** 首页 `/` 作为总览落地页，负责展示项目概览、关键统计、运行健康和待处理事项；`/board` 专注任务推进与调度流，不承担系统总览职责。
- **D-03:** Phase 5 的正式核心入口限定为 4 个：调度看板、Wave 管理、事件日志、任务详情。研究和规划必须围绕这 4 个入口组织页面与路由。
- **D-04:** 现有扩展页 `timeline`、`goals`、`agents`、`swimlane`、`org`、`knowledge` 不作为 Phase 5 的正式一层导航，降级为“更多”或其他次级入口。

### Claude's Discretion
- 正式导航的具体文案、图标与分组方式
- “更多”或二级导航的具体呈现形式
- 首页总览卡片的具体排布与视觉细节
- 任务详情从哪些主页面跳入的交互细节

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap and phase scope
- `.planning/ROADMAP.md` — Phase 5 goal, requirement mapping (`API-01`, `API-02`, `API-03`, `UI-01`..`UI-07`) and success criteria for the formal interface surface.
- `.planning/REQUIREMENTS.md` — Detailed API/UI requirement definitions, especially manual Task Card editing, WebSocket push, task list/detail, Wave management, dashboard, and event log coverage.
- `.planning/PROJECT.md` — Product vision, local-first single-user constraints, Windows-first constraints, and React + Go platform expectations.
- `.planning/STATE.md` — Current project status and existing runtime/UI baseline that Phase 5 should formalize rather than replace.

### Existing platform contracts
- `docs/project-api-contract.md` — Current project-scoped API contract and integration surface used by the frontend.
- `docs/prd/PRD-OPS-001-platform-runtime-reliability.md` — Existing runtime-health and operator-facing reliability requirements already surfaced in the current UI baseline.
- `docs/plans/PLAN-OPS-001-platform-runtime-reliability.md` — Prior implementation plan for runtime/health UX that should be preserved when formalizing Phase 5 navigation and overview.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/components/Layout.tsx` — Already provides the shell, sidebar navigation, current-project switcher, and modal pattern; can be reused as the Phase 5 primary frame.
- `web/src/pages/SchedulerBoard.tsx` — Existing task board with create/edit/dispatch/retry flows, runtime health, and per-task side panel; strongest base for the formal task list / board experience.
- `web/src/pages/TaskDetailPage.tsx` — Existing dedicated task detail page with tabs for details, lineage, logs, and agent calls; strong base for the formal task-detail experience.
- `web/src/hooks/useWebSocket.ts` — Existing frontend real-time hook with reconnect logic and project scoping.
- `web/src/api/schedulerApi.ts` — Existing typed client for scheduler, lineage, execution, agent calls, stats, and health endpoints.
- `internal/server/websocket.go` — Existing WebSocket hub, project-scoped broadcast filtering, auth handling, and `/api/v1/ws` endpoint registration.
- `internal/server/server.go` — Existing HTTP routes for task CRUD, task events, dispatch/wave operations, compat project scheduler APIs, and system health endpoints.

### Established Patterns
- Frontend uses functional React components plus TanStack Query for data fetching and cache invalidation.
- Existing UI is page-based and route-driven, not a single-screen app; formal Phase 5 IA should refine this rather than replace it with a wholly different interaction model.
- Existing backend route surface already mixes core `/api/tasks` and project-scoped `/api/v1/projects/{projectID}/scheduler/*`; Phase 5 planning must account for this API-shape drift instead of assuming a clean slate.
- Runtime/system-health visibility is already part of the board experience; the formal overview page should absorb and organize this pattern rather than discard it.

### Integration Points
- `web/src/App.tsx` is the route map that must be reorganized to reflect the locked Phase 5 information architecture.
- `web/src/components/Layout.tsx` is the navigation choke point where first-class vs secondary entry decisions will land.
- `web/src/pages/SchedulerBoard.tsx` and `web/src/pages/TaskDetailPage.tsx` are the primary anchors for task list/detail formalization.
- New or formalized Wave/event pages will need to bind to existing backend routes in `internal/server/server.go` and likely share client utilities with `web/src/api/schedulerApi.ts`.

</code_context>

<specifics>
## Specific Ideas

- 正式 IA 不是推倒重做，而是把现有零散页面收束成“首页总览 → 核心控制页 → 次级扩展页”的层级。
- `/board` 不应继续兼任“系统总览首页”，而应回归任务推进与调度主战场。
- Wave 管理和事件日志必须从“后端已存在能力 / 局部细节能力”提升为正式前台入口。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-interface*
*Context gathered: 2026-05-16*