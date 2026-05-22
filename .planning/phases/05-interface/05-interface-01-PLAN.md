---
phase: 05-interface
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/App.tsx
  - web/src/components/Layout.tsx
  - web/src/pages/OrchestratorHome.tsx
  - web/src/styles/global.css
autonomous: true
requirements:
  - UI-05
  - UI-01
must_haves:
  truths:
    - "正式 Phase 5 不重做前端壳层，而是在现有 Layout + page routes 基础上收敛成首页总览 + 核心控制页 + 次级扩展页。"
    - "`/` 必须回归总览首页职责，`/board` 专注任务推进，不再兼任系统总览首页。"
    - "timeline/goals/agents/swimlane/org/knowledge 必须从一层导航降级，但页面能力保留。"
  artifacts:
    - path: "web/src/App.tsx"
      provides: "正式核心路由结构"
      contains: "Route path=\"waves\"|Route path=\"events\""
    - path: "web/src/components/Layout.tsx"
      provides: "核心导航与次级导航分层"
      contains: "navItems"
    - path: "web/src/pages/OrchestratorHome.tsx"
      provides: "正式首页总览"
      contains: "dashboard|summary|health"
  key_links:
    - from: "web/src/App.tsx"
      to: "web/src/components/Layout.tsx"
      via: "核心入口与次级入口在同一个正式 shell 内承载"
      pattern: "Layout"
    - from: "web/src/pages/OrchestratorHome.tsx"
      to: "web/src/api/schedulerApi.ts"
      via: "首页总览复用现有统计与健康数据，而不是新建平行数据源"
      pattern: "getStats|getSystemHealth|getSystemWorkers"
---

<objective>
收敛 Phase 5 的正式页面骨架：重组路由、导航和首页职责，让控制台优先 IA 在现有 UI 基础上落地。

Purpose: 当前前端已经有大量页面，但正式产品入口过载。此计划负责把首页、主导航和页面层级整理成可持续的正式控制台结构。
Output: 正式核心路由、分层导航、收口后的首页总览，以及从主导航降级的扩展入口。
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
@web/src/App.tsx
@web/src/components/Layout.tsx
@web/src/pages/OrchestratorHome.tsx
@web/src/pages/SchedulerBoard.tsx
@web/src/api/schedulerApi.ts
@web/src/hooks/useProject.ts

<interfaces>
正式一级入口必须收敛到：
- `/`
- `/board`
- `/waves`
- `/events`
- `/tasks/:taskId`

扩展页可以保留原路由，但不能继续全部停留在一层主导航。
不要把首页总览逻辑塞回 `SchedulerBoard.tsx`。
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: 重组正式路由并引入核心入口</name>
  <files>web/src/App.tsx</files>
  <read_first>
    <file>web/src/App.tsx</file>
    <file>web/src/pages/OrchestratorHome.tsx</file>
    <file>.planning/phases/05-interface/05-CONTEXT.md</file>
  </read_first>
  <action>
    调整 `web/src/App.tsx` 的正式路由结构：
    - 保留 `/` 为首页总览
    - 保留 `/board` 与 `/tasks/:taskId`
    - 引入 `/waves` 与 `/events` 正式路由占位（由后续计划实现页面内容）
    - 保留扩展页原路由，但为后续在 Layout 中降级入口做准备

    不要删除现有扩展页，不要引入新的路由框架。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>正式核心入口出现在路由层，页面层级与 Phase 5 context 一致。</done>
</task>

<task type="auto">
  <name>Task 2: 收敛 Layout 导航为核心区 + 次级区</name>
  <files>web/src/components/Layout.tsx</files>
  <read_first>
    <file>web/src/components/Layout.tsx</file>
    <file>.planning/phases/05-interface/05-CONTEXT.md</file>
    <file>.planning/phases/05-interface/05-RESEARCH.md</file>
  </read_first>
  <action>
    改造 `Layout.tsx`：
    - 主导航只保留首页、调度看板、Wave 管理、事件日志
    - 把 timeline/goals/agents/swimlane/org/knowledge 收纳为“更多”或明确次级分组
    - 保留当前项目切换和新增项目弹窗

    不要删除项目切换，不要重做整体 sidebar 壳层。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>主导航不再过载，扩展页成功降级为次级入口。</done>
</task>

<task type="auto">
  <name>Task 3: 让首页只承担总览职责</name>
  <files>web/src/pages/OrchestratorHome.tsx, web/src/styles/global.css</files>
  <read_first>
    <file>web/src/pages/OrchestratorHome.tsx</file>
    <file>web/src/pages/SchedulerBoard.tsx</file>
    <file>web/src/api/schedulerApi.ts</file>
  </read_first>
  <action>
    把首页收敛为总览页：
    - 任务统计
    - 运行时 / worker 健康
    - 最近更新
    - 当前重点 / 待处理事项
    - 活跃 dispatch / Wave 快照（若当前 API 不足，可先用已有统计与任务数据组织）

    避免把看板列、批量导入、任务编辑等交互放回首页。
  </action>
  <verify>
    <automated>npm run build</automated>
  </verify>
  <done>首页成为正式总览落地页，`/board` 保持任务推进定位。</done>
</task>

</tasks>

<verification>
```bash
npm run build
```
</verification>

<success_criteria>
- Phase 5 的正式核心入口已经在路由和导航层可见。
- 首页与看板职责分离清晰。
- 扩展页仍可访问，但不再挤占一级导航。
</success_criteria>

<output>
After completion, create `.planning/phases/05-interface/05-interface-01-SUMMARY.md`
</output>
