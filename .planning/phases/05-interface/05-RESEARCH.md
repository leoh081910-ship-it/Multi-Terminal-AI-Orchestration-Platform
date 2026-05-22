# Phase 5: Interface - Research

**Researched:** 2026-05-16
**Domain:** 正式控制台界面收敛（HTTP API + React Web UI）
**Confidence:** HIGH（已核对现有前后端代码、Phase 5 context、requirements 与项目化 API 契约）

## Summary

Phase 5 不是从零做一套新前台，而是在现有兼容调度 UI / API 之上，把平台正式入口收敛成“首页总览 → 核心控制页 → 次级扩展页”的结构。现有代码已经具备可复用的壳层、任务 CRUD、任务详情、运行时健康、项目作用域 API 和 WebSocket 推送基础，真正缺口集中在：信息架构收束、正式 Wave 页面、正式事件日志页、首页/看板职责拆分，以及把实时推送从“已有能力”提升成“页面级一致行为”。

**Primary recommendation:** 按 3 个执行面拆分：
1. 先重组 `App.tsx` / `Layout.tsx` / 首页，总结正式导航与一级入口；
2. 再补齐 Wave 管理页、事件日志页及其前端数据访问；
3. 最后把任务列表/详情/总览页面的实时刷新统一为“WebSocket 驱动 + 必要轮询兜底”，并补全表单与验收基线。

---

<user_constraints>

## Locked Decisions (from 05-CONTEXT.md)

- 正式 IA 采用“控制台优先”，不是任务中心优先，也不是纯仪表盘优先。
- `/` 是总览落地页，用于项目概览、关键统计、运行健康、待处理事项。
- 正式核心入口限定为 4 个：调度看板、Wave 管理、事件日志、任务详情。
- `timeline`、`goals`、`agents`、`swimlane`、`org`、`knowledge` 不再是一层主导航，降级为次级入口。

## Scope Boundaries

- 目标是 formalize 现有 API/UI，不是重做后端领域模型。
- 不新增 roadmap 外的新能力；优先重组现有能力和补正式缺口。
- 仍然是桌面优先、本地优先、单用户项目化控制台。

</user_constraints>

---

## Current State Inventory

### Frontend shell and routing

- `web/src/App.tsx` 已经采用页面路由结构，当前挂载：`/`、`/board`、`/timeline`、`/goals`、`/agents`、`/swimlane`、`/org`、`/knowledge`、`/tasks/:taskId`。
- `web/src/components/Layout.tsx` 已经承担正式壳层：侧边栏导航、当前项目切换、项目创建弹窗、`<Outlet />` 页面承载。
- 当前一层导航仍然把所有扩展页平铺在主侧栏，和已锁定的 D-03/D-04 冲突。

### Existing task surfaces

- `web/src/pages/SchedulerBoard.tsx` 已具备：
  - 项目作用域任务列表读取
  - 创建任务
  - 批量导入
  - 派发 / 重试
  - 运行时健康卡
  - 任务侧栏编辑
  - 执行时间线和实时输出面板
- `web/src/pages/TaskDetailPage.tsx` 已具备：
  - dedicated 详情页路由 `/tasks/:taskId`
  - tabs：details / lineage / logs / agent-calls
  - task retry / dispatch
  - task-level websocket invalidate

### Existing realtime and logs foundation

- `web/src/hooks/useWebSocket.ts` 已提供 project query 参数、重连逻辑、最后一条消息订阅。
- `internal/server/websocket.go` 已提供 `/api/v1/ws`，并支持按 `project` 过滤广播。
- `web/src/components/LogViewer.tsx` 已能把 websocket message 渲染为日志条目，但当前更接近“事件流查看器”，不是真正的正式事件日志页。

### Existing API surface

- `docs/project-api-contract.md` 明确前端主入口已统一到 `/api/v1/projects/{projectId}`。
- `web/src/api/schedulerApi.ts` 已覆盖：
  - tasks list/create/update/dispatch/retry
  - execution / lineage / runtimes / agents / board summary
  - system health / system workers
- `internal/server/compat_scheduler.go` 已有项目作用域兼容调度接口。

### Existing backend gaps relative to formal UI

- 任务事件的正式项目化查询入口没有暴露给当前项目化前端客户端；仓储层虽有 `ListEventsByTaskID`，但当前前端主 API client 没有对应 task events / dispatch events 正式读取能力。
- Wave 后端能力存在于非项目化 `/api/dispatches/{dispatchRef}/waves*` 路由（`internal/server/server.go`），但当前浏览器主客户端 `schedulerApi.ts` 没有正式 Wave 数据访问层。
- WebSocket 能力已存在，但页面层大多仍以轮询为主，导致“有推送但没收束成正式交互模型”。

---

## Requirements Coverage Analysis

### Already satisfied or mostly satisfied

#### API-01: RESTful API 暴露任务 CRUD、Wave 操作、状态查询、事件查询
- **Partially satisfied / effectively present**
- 证据：
  - task CRUD 与状态查询已存在于 `internal/server/server.go` 和 `internal/server/compat_scheduler.go`
  - 项目化 task CRUD 已被 `schedulerApi.ts` 使用
  - wave 操作只在非项目化 dispatch 路径存在
  - 事件查询在非项目化 `/api/tasks/{id}/events` 存在，但未进入项目化正式前端主流
- **结论：** API-01 的“任务 CRUD / 状态查询”已具备，但 Wave / 事件查询仍需正式化接入，才能支撑 Phase 5 的正式入口。

#### API-02: API 支持手动创建/编辑 Task Card（非 Connector 来源）
- **Mostly satisfied but schema surface still narrow**
- 证据：
  - `handleCompatCreateSchedulerTask` / `handleCompatUpdateSchedulerTask` 已支持浏览器项目化创建与编辑。
  - `SchedulerBoard.tsx` 已有创建表单和侧栏编辑。
- **差距：**
  - 当前表单字段偏“兼容看板任务”，不是“完整 Task Card 正式编辑体验”。
  - `CreateScheduledTaskInput` / `UpdateScheduledTaskInput` 缺少 `dispatch_ref`、`wave`、更完整的关系编辑语义，不足以支撑“正式 Task Card 手动编辑”定位。

#### API-03: WebSocket 端点推送实时任务状态变更
- **Satisfied at transport level, not fully consumed at page level**
- 证据：
  - `internal/server/websocket.go` 提供广播与 project filter。
  - `useWebSocket.ts` 已可连接并接收消息。
- **差距：**
  - 页面对 ws 的消费不统一；多数 query 仍靠 `refetchInterval`。
  - 正式事件日志 / Wave / 首页概览还没有共同的实时刷新策略。

### UI requirements gap map

#### UI-01: 任务列表页，支持按状态 / wave / dispatch_ref 筛选和排序
- **Partially satisfied**
- `SchedulerBoard.tsx` 有 agent/status 筛选，但没有 wave / dispatch_ref 正式筛选，也没有围绕列表语义的排序控制。
- 当前呈现是状态列式看板，不是正式“任务列表 + 看板兼容”的统一体验。

#### UI-02: 任务详情页，展示 Task Card 完整字段、状态历史、事件日志
- **Partially satisfied**
- `TaskDetailPage.tsx` 已有详情 / lineage / logs / agent-calls。
- 缺口：
  - query 通过 `getTaskExecution + getTasks().find(...)` 获取任务，不够正式。
  - “完整 Task Card 字段”仍依赖 compat 映射后的有限字段，不是完整业务模型呈现。
  - 缺正式事件历史读取接口与显式状态历史组件。

#### UI-03: Wave 管理页面，展示各 wave 状态（open / sealed）、支持 seal 操作
- **Missing formal page**
- 当前前端没有正式 Wave 页面；仅后端存在 dispatch-scoped waves 路由。

#### UI-04: 任务创建 / 编辑表单，校验字段格式约束
- **Partially satisfied**
- 当前 `SchedulerBoard.tsx` 中已有创建/编辑，但字段有限，校验也偏弱。
- 正式表单仍需扩展为“Task Card 编辑器”而不是简单任务录入框。

#### UI-05: 全局状态仪表盘，展示任务统计、活跃 dispatch_ref、合并队列状态
- **Partially satisfied**
- `/` 当前由 `OrchestratorHome` 承担，但本研究尚未核查到其是否符合新职责；从现有代码结构看，总览职责尚未明确收敛。
- `schedulerApi.getStats()` 能提供部分统计，system health/workers 也已存在。
- 缺口：活跃 dispatch_ref、合并队列状态、正式待处理事项模块需要在首页模型中明确组织。

#### UI-06: 实时状态更新，任务状态变更通过 WebSocket 推送到前端
- **Transport ready, page UX incomplete**
- 任务详情页已有 invalidate 模式；看板、首页、事件日志、Wave 页尚未形成统一实时策略。

#### UI-07: 事件日志浏览器，按任务或 dispatch_ref 筛选查看完整事件链
- **Missing formal page**
- 现有 `LogViewer` 更像 websocket live feed viewer；没有正式的事件查询页面、dispatch_ref 筛选器、完整事件链浏览结构。

---

## Key Reusable Assets

### Strong reuse candidates

1. `web/src/components/Layout.tsx`
   - 直接复用为正式壳层。
   - 需要改造的是导航分层，不是重写 shell。

2. `web/src/pages/SchedulerBoard.tsx`
   - 可继续作为 `/board` 主入口。
   - 需收敛成“任务推进与调度”角色，剥离系统总览职责。

3. `web/src/pages/TaskDetailPage.tsx`
   - 已经是正式任务详情页的最好基础。
   - 缺的是数据获取和字段覆盖深度，不是页面形态。

4. `web/src/components/LogViewer.tsx`
   - 可复用为事件日志页中的“实时流”区域或任务日志 tab 的 live pane。
   - 不适合直接等同于“正式事件日志浏览器”。

5. `web/src/hooks/useWebSocket.ts`
   - 可作为所有核心页统一的实时入口。

6. `web/src/api/schedulerApi.ts`
   - 应继续作为正式前端数据访问中心，补齐 wave / events / dashboard required fields。

### Reuse with caveats

1. `internal/server/server.go`
   - 同时存在 `/api/v1/projects/*` 与 `/api/tasks`、`/api/dispatches/*` 两套接口风格。
   - Phase 5 不应再扩散双轨风格，而应让正式 UI 优先走项目化路径。

2. `internal/server/compat_scheduler.go`
   - 当前正式前端其实强依赖 compat project-scoped API。
   - 这说明“compat”已经事实承担正式 UI contract，需要在计划中承认并稳住，而不是假设可立即切换到另一套新 contract。

---

## Architecture Decisions for Planning

### 1. Route / navigation target structure

推荐正式路由收束为：
- `/` — 总览首页
- `/board` — 调度看板
- `/waves` — Wave 管理
- `/events` — 事件日志
- `/tasks/:taskId` — 任务详情
- `/more/*` 或保留原扩展页原路由但从 Layout 降级显示

**Why:** 这样能符合已锁定 D-01~D-04，同时最大化复用现有页面和路由习惯。

### 2. Navigation grouping

推荐 `Layout.tsx` 改为两组：
- 核心控制：控制中心、调度看板、Wave 管理、事件日志
- 扩展能力：timeline / goals / agents / swimlane / org / knowledge

**Why:** 已锁定“降级为次级入口”，但又不需要删除现有页面。

### 3. Dashboard / homepage responsibility split

推荐把首页聚焦为：
- 任务总数 / 状态统计
- 活跃运行时 / worker 健康
- 最近更新任务
- 当前重点 / 待处理事项
- 活跃 dispatch_ref / waves 快照

同时把 `SchedulerBoard.tsx` 保持为任务推进主战场，不再承载系统总览叙事。

### 4. Wave page shape

正式 Wave 页面至少需要：
- dispatch_ref 选择或过滤
- wave 列表
- open / sealed 状态
- 每个 wave 下任务数 / 状态分布
- seal 操作
- 跳转至 board / task detail

### 5. Event page shape

正式事件日志页至少需要：
- task_id / dispatch_ref 过滤
- 时间倒序事件链
- event_type / from_state / to_state / reason 字段
- 实时流追加
- 跳转 task detail

---

## Risks and Constraints

### Risk 1: API style drift

- `internal/server/server.go` 的非项目化接口返回 `APIResponse { success, data, error }`
- `internal/server/compat_scheduler.go` 项目化接口多处直接返回 plain JSON
- 前端 `schedulerApi.ts` 已经绑定项目化 plain JSON 风格

**Planning impact:** 不要在 Phase 5 再混入第三种接口风格；正式 UI 继续以项目化 compat 接口为主，并在其上补齐所需能力。

### Risk 2: Query model drift

- `.planning/codebase/CONVENTIONS.md` 指出 `web/src/main.tsx` 与 `web/src/App.tsx` 存在 QueryClient setup drift。
- `main.tsx` 当前只渲染 `<App />`，实际 QueryClient 在 `App.tsx` 中；说明旧约定文档部分已过时，但也说明前端初始化模式曾有漂移。

**Planning impact:** 不需要为此单独开大重构，但如果 Phase 5 调整数据刷新策略，应保持 Query / invalidate 模式统一，不新增第二套 realtime state 管理。

### Risk 3: WebSocket consumption inconsistency

- `TaskDetailPage.tsx` 用 ws invalidate query。
- `SchedulerBoard.tsx` 和系统卡主要依靠 2s/4s/5s polling。
- `LogViewer.tsx` 直接把 ws 显示成 log stream。

**Planning impact:** Phase 5 应定义统一策略：
- 核心页面以 ws 触发 invalidate 为主；
- polling 只保留为兜底和离线恢复；
- 事件日志页允许 live append。

### Risk 4: Wave contract not yet project-scoped in frontend client

- 后端 Wave 操作在 `/api/dispatches/{dispatchRef}`。
- 前端主客户端是 `/api/v1/projects/{projectId}`。

**Planning impact:** Phase 5 需要决定是：
- 在 compat project API 下补新的 wave 查询/封装入口；或
- 前端单独调用旧 dispatch 路由。

**Recommendation:** 优先在项目化 compat surface 下补 Wave API，避免正式 UI 需要跨两套前缀取数。

### Risk 5: Event browser data source ambiguity

- 实时事件可以走 ws。
- 历史事件链要走 repository-backed HTTP query。

**Planning impact:** 事件日志页必须是“历史查询 + 实时追加”的组合，而不是只用 ws 替代持久事件查询。

---

## Concrete Planning Recommendations

### Recommended plan split

#### Plan 01 — IA / shell / overview / route consolidation
覆盖：
- `App.tsx` 路由重组
- `Layout.tsx` 主导航/次级导航收束
- 首页总览职责收束
- 从 board 中挪走不该属于总览首页的叙事负担

#### Plan 02 — Wave / events formal surfaces + API access
覆盖：
- `schedulerApi.ts` 补 wave / events 数据访问
- 后端补项目化 Wave / 事件接口（若缺）
- 新建正式 `WaveManagementPage` / `EventLogPage`

#### Plan 03 — task detail / form / realtime unification
覆盖：
- 任务详情数据模型补全
- Task Card 创建/编辑表单正式化
- websocket-driven invalidate 策略统一
- 关键页面 golden-path 验证

### Verification baseline

- 后端：`go test ./internal/server ./internal/store -count=1`
- 前端：`npm run build`
- UI 手动：
  - 首页 → board → task detail → waves → events 黄金路径
  - 创建任务 / 编辑任务 / seal wave / 查看事件 / 实时刷新

---

## File-Level Notes for Planner

- `web/src/App.tsx`
  - 是 Phase 5 正式路由结构的唯一主入口。
- `web/src/components/Layout.tsx`
  - 是一级/二级导航收口点。
- `web/src/pages/SchedulerBoard.tsx`
  - 任务列表、CRUD、执行侧栏已成熟，不应重写。
- `web/src/pages/TaskDetailPage.tsx`
  - 详情页应保留并补强，不应降级为 board 内嵌弹层替代。
- `web/src/components/LogViewer.tsx`
  - 适合做事件页中的 live stream panel，不适合作为唯一事件历史来源。
- `web/src/api/schedulerApi.ts`
  - 应扩成 Phase 5 正式 UI 的单一数据访问层。
- `internal/server/server.go`
  - 保留现有非项目化兼容接口，但正式 UI 不应继续依赖散落的老路由。
- `internal/server/compat_scheduler.go`
  - 当前事实上是正式浏览器 contract，应在其上补能力而非旁开新客户端协议。
- `internal/server/websocket.go`
  - 继续复用，不需要重做 ws hub。

---

## Conclusion

Phase 5 的核心不是“有没有页面”，而是“把现有页面和接口收敛成正式控制台产品面”。代码已经提供了足够强的基础：壳层、项目化 scheduler API、task board、task detail、运行时健康和 websocket hub 都在。真正要补的是四件事：正式导航层级、首页职责、Wave/事件日志正式页面、以及跨页面一致的实时刷新模式。只要规划围绕这四件事拆解，Phase 5 可以以较小增量把当前“兼容调度前台”提升为正式控制台界面。