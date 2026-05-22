---
phase: 05-interface
plan: "02"
type: execute
wave: 1
depends_on:
  - 05-interface-01
files_modified:
  - web/src/api/schedulerApi.ts
  - web/src/pages/WaveManagementPage.tsx
  - web/src/pages/EventLogPage.tsx
  - internal/server/server.go
  - internal/server/compat_scheduler.go
  - internal/store/repository.go
  - internal/store/repository_test.go
  - internal/server/server_test.go
autonomous: true
requirements:
  - API-01
  - UI-03
  - UI-07
must_haves:
  truths:
    - "Phase 5 的正式 Wave 管理和事件日志必须复用现有项目化调度契约，而不是让正式 UI 横跨两套完全不同的 API 风格。"
    - "事件日志页必须同时具备历史查询与实时追加能力，不能只依赖 websocket 内存流。"
    - "Wave 页面是正式前台入口，不是把旧 dispatch 路由临时暴露给浏览器就算完成。"
  artifacts:
    - path: "web/src/api/schedulerApi.ts"
      provides: "Wave / event 的正式前端数据访问"
      contains: "getTaskEvents|getDispatchEvents|listWaves|sealWave"
    - path: "web/src/pages/WaveManagementPage.tsx"
      provides: "正式 Wave 管理页"
      contains: "Wave"
    - path: "web/src/pages/EventLogPage.tsx"
      provides: "正式事件日志页"
      contains: "event"
    - path: "internal/server/compat_scheduler.go"
      provides: "项目化 Wave / event 查询入口"
      contains: "wave|event"
  key_links:
    - from: "internal/store/repository.go"
      to: "internal/server/compat_scheduler.go"
      via: "持久事件和 wave 数据通过项目化接口供正式 UI 消费"
      pattern: "ListEvents|Wave"
    - from: "web/src/pages/EventLogPage.tsx"
      to: "web/src/hooks/useWebSocket.ts"
      via: "历史列表 + 实时追加组合成正式事件浏览体验"
      pattern: "useWebSocket"
---

<objective>
补齐 Phase 5 正式缺失的两个核心入口：Wave 管理与事件日志，并把它们接入项目化 API 和现有实时机制。

Purpose: 当前代码有 waves 路由和 websocket/event 基础，但浏览器主控制台没有正式 Wave 页面与事件浏览器。本计划把缺失入口补成正式产品面。
Output: Wave 管理页、事件日志页、对应的项目化 API client 和必要的后端项目化查询能力。
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
@docs/project-api-contract.md
@web/src/api/schedulerApi.ts
@web/src/components/LogViewer.tsx
@web/src/hooks/useWebSocket.ts
@internal/server/server.go
@internal/server/compat_scheduler.go
@internal/store/repository.go

<interfaces>
优先保持正式 UI 走 `/api/v1/projects/{projectId}` 体系。
如果缺少 Wave / 事件查询能力，应先补项目化 compat 接口，再由 `schedulerApi.ts` 接入。
不要为 Wave 页面直接在前端拼接另一套非项目化数据访问层，除非无法避免且有清楚说明。
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 为正式 UI 补齐项目化 Wave / 事件 API</name>
  <files>internal/server/server.go, internal/server/compat_scheduler.go, internal/store/repository.go, internal/store/repository_test.go, internal/server/server_test.go</files>
  <read_first>
    <file>internal/server/server.go</file>
    <file>internal/server/compat_scheduler.go</file>
    <file>internal/store/repository.go</file>
    <file>docs/project-api-contract.md</file>
  </read_first>
  <action>
    补齐正式浏览器所需的项目化接口：
    - Wave 列表 / 明细 / seal 所需查询
    - task events / dispatch events 等事件链查询

    规则：
    - 尽量走项目化 compat surface
    - 复用 repository 已有事件和任务数据，不建立第二套 event 存储
    - 返回结构要能直接被前端正式页面消费
  </action>
  <verify>
    <automated>go test ./internal/server ./internal/store -count=1</automated>
  </verify>
  <done>正式 UI 所需的 Wave / event 后端项目化能力齐备且有回归保障。</done>
</task>

<task type="auto">
  <name>Task 2: 扩展 schedulerApi 为正式 Wave / 事件访问层</name>
  <files>web/src/api/schedulerApi.ts</files>
  <read_first>
    <file>web/src/api/schedulerApi.ts</file>
    <file>docs/project-api-contract.md</file>
    <file>internal/server/compat_scheduler.go</file>
  </read_first>
  <action>
    在 `schedulerApi.ts` 中新增 Phase 5 正式页面使用的 Wave / 事件方法：
    - wave 列表 / 明细 / seal
    - task events / dispatch_ref 事件过滤

    保持它仍是正式 UI 的单一数据访问层。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>前端拥有正式 Wave / event 数据访问，不再依赖页面内临时 fetch。</done>
</task>

<task type="auto">
  <name>Task 3: 实现正式 Wave 管理页与事件日志页</name>
  <files>web/src/pages/WaveManagementPage.tsx, web/src/pages/EventLogPage.tsx, web/src/App.tsx</files>
  <read_first>
    <file>web/src/components/LogViewer.tsx</file>
    <file>web/src/pages/SchedulerBoard.tsx</file>
    <file>web/src/App.tsx</file>
  </read_first>
  <action>
    新增两个正式页面：
    - `WaveManagementPage.tsx`：展示 wave 状态、任务数、seal 操作、与 board/task 的跳转
    - `EventLogPage.tsx`：支持 task / dispatch_ref 过滤、历史事件列表、实时追加

    规则：
    - 复用现有样式和组件模式
    - 事件日志页可以复用 `LogViewer` 的 live stream 思路，但不能只有 live stream
    - 页面要从 `/waves` 和 `/events` 正式路由进入
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>Wave 管理和事件日志成为正式可访问的核心入口。</done>
</task>

</tasks>

<verification>
```bash
go test ./internal/server ./internal/store -count=1
npm run build
```
</verification>

<success_criteria>
- 正式前端存在 `/waves` 与 `/events` 页面。
- Wave 与事件数据通过正式项目化 API 消费。
- 事件日志同时具备历史查询与实时能力。
</success_criteria>

<output>
After completion, create `.planning/phases/05-interface/05-interface-02-SUMMARY.md`
</output>
