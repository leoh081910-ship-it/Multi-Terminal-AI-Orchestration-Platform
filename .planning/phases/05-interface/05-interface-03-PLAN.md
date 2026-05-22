---
phase: 05-interface
plan: "03"
type: execute
wave: 2
depends_on:
  - 05-interface-01
  - 05-interface-02
files_modified:
  - web/src/pages/SchedulerBoard.tsx
  - web/src/pages/TaskDetailPage.tsx
  - web/src/hooks/useWebSocket.ts
  - web/src/api/schedulerApi.ts
  - web/src/types/scheduler.ts
  - internal/server/compat_scheduler.go
  - internal/server/websocket.go
  - web/src/components/LogViewer.tsx
autonomous: true
requirements:
  - API-02
  - API-03
  - UI-01
  - UI-02
  - UI-04
  - UI-06
must_haves:
  truths:
    - "现有 SchedulerBoard 和 TaskDetailPage 是 Phase 5 正式 task list/detail 的基础，应该补强而不是重写。"
    - "实时更新要统一成 websocket 触发的页面级刷新策略，轮询只作为兜底，而不是继续让每个页面自行漂移。"
    - "任务创建 / 编辑必须朝正式 Task Card 表单迈进，至少覆盖 wave、depends_on、acceptance_criteria 等核心字段和格式校验。"
  artifacts:
    - path: "web/src/pages/SchedulerBoard.tsx"
      provides: "正式任务列表 / 调度页"
      contains: "dispatch_ref|wave|filter|sort"
    - path: "web/src/pages/TaskDetailPage.tsx"
      provides: "正式任务详情页"
      contains: "details|lineage|logs|agent-calls"
    - path: "web/src/hooks/useWebSocket.ts"
      provides: "统一实时订阅策略"
      contains: "lastMessage"
    - path: "web/src/types/scheduler.ts"
      provides: "正式 Task Card / event / wave 前端类型"
      contains: "ScheduledTask"
  key_links:
    - from: "web/src/pages/SchedulerBoard.tsx"
      to: "web/src/api/schedulerApi.ts"
      via: "任务列表、筛选、创建、编辑与 dispatch 行为统一走正式 client"
      pattern: "schedulerApi"
    - from: "web/src/pages/TaskDetailPage.tsx"
      to: "web/src/hooks/useWebSocket.ts"
      via: "详情页通过 ws invalidate 响应实时状态变化"
      pattern: "useWebSocket"
---

<objective>
把任务列表、任务详情、Task Card 表单和实时刷新统一收口，完成 Phase 5 剩余的正式交互闭环。

Purpose: 当前 board/detail 已可用，但字段覆盖、筛选维度和实时策略仍偏兼容实现。本计划负责把它们提升为正式产品体验。
Output: 正式任务列表筛选、完整详情页、增强 Task Card 表单，以及 websocket 驱动的统一实时更新行为。
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
@.planning/phases/05-interface/05-CONTEXT.md
@.planning/phases/05-interface/05-RESEARCH.md
@.planning/phases/05-interface/05-VALIDATION.md
@CLAUDE.md
@web/src/pages/SchedulerBoard.tsx
@web/src/pages/TaskDetailPage.tsx
@web/src/hooks/useWebSocket.ts
@web/src/api/schedulerApi.ts
@web/src/types/scheduler.ts
@web/src/components/LogViewer.tsx
@internal/server/compat_scheduler.go
@internal/server/websocket.go

<interfaces>
- 保持 `SchedulerBoard.tsx` 作为 `/board` 主页面。
- 保持 `TaskDetailPage.tsx` 作为 `/tasks/:taskId` 正式详情页。
- 正式前端仍以项目化 task contract 为主。
- websocket 作为实时信号，query 失效和兜底 polling 由统一规则控制。
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: 提升任务列表与 Task Card 表单覆盖面</name>
  <files>web/src/pages/SchedulerBoard.tsx, web/src/api/schedulerApi.ts, web/src/types/scheduler.ts</files>
  <read_first>
    <file>web/src/pages/SchedulerBoard.tsx</file>
    <file>web/src/types/scheduler.ts</file>
    <file>.planning/REQUIREMENTS.md</file>
  </read_first>
  <action>
    把 `SchedulerBoard.tsx` 提升为正式任务列表/调度页：
    - 增加 wave / dispatch_ref 维度筛选
    - 补充排序或至少明确排序视图
    - 让创建/编辑表单覆盖正式 Task Card 核心字段（如 depends_on、acceptance_criteria、wave 等）
    - 对字段做基础格式校验

    不要推翻现有看板列结构；是在其上补列表与筛选能力。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>任务列表和表单符合正式 task control surface 的最低要求。</done>
</task>

<task type="auto">
  <name>Task 2: 补强任务详情为正式详情页</name>
  <files>web/src/pages/TaskDetailPage.tsx, web/src/api/schedulerApi.ts, web/src/components/LogViewer.tsx</files>
  <read_first>
    <file>web/src/pages/TaskDetailPage.tsx</file>
    <file>web/src/components/LogViewer.tsx</file>
    <file>web/src/api/schedulerApi.ts</file>
  </read_first>
  <action>
    补强详情页：
    - 避免通过 `getTasks().find(...)` 间接查单任务，改为更正式的数据获取方式
    - 明确显示 Task Card 关键字段、状态历史、事件日志
    - 保留 lineage / agent calls / logs 等已有优势

    不要把详情页重新退回 board drawer-only 模式。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>任务详情页成为正式稳定入口，而不是兼容页面拼装结果。</done>
</task>

<task type="auto">
  <name>Task 3: 统一实时刷新策略</name>
  <files>web/src/hooks/useWebSocket.ts, web/src/pages/SchedulerBoard.tsx, web/src/pages/TaskDetailPage.tsx, web/src/pages/EventLogPage.tsx, internal/server/websocket.go</files>
  <read_first>
    <file>web/src/hooks/useWebSocket.ts</file>
    <file>web/src/pages/SchedulerBoard.tsx</file>
    <file>web/src/pages/TaskDetailPage.tsx</file>
    <file>internal/server/websocket.go</file>
  </read_first>
  <action>
    统一实时刷新策略：
    - websocket 负责推送状态变化信号
    - 页面通过 invalidate 或 append 进行更新
    - polling 仅保留为兜底，不再让每个页面无约束漂移

    需要覆盖 board、task detail、events，必要时首页也同步受益。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>Phase 5 的实时更新行为在核心页面上一致、可预期。</done>
</task>

</tasks>

<verification>
```bash
npm run build
```
</verification>

<success_criteria>
- 任务列表支持正式筛选与编辑需求。
- 任务详情展示完整度达到 Phase 5 要求。
- 核心页面统一采用 websocket 驱动的实时更新策略。
</success_criteria>

<output>
After completion, create `.planning/phases/05-interface/05-interface-03-SUMMARY.md`
</output>
