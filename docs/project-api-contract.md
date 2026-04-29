# Project-scoped orchestration contract

平台前端和兼容后端现在统一走项目化接口。

基础路径：

`/api/v1/projects/{projectId}`

## 当前主接口

- `GET /projects`
- `GET /projects/{projectId}/scheduler/tasks`
- `POST /projects/{projectId}/scheduler/tasks`
- `PATCH /projects/{projectId}/scheduler/tasks/{taskId}`
- `POST /projects/{projectId}/scheduler/tasks/{taskId}/dispatch`
- `POST /projects/{projectId}/scheduler/tasks/{taskId}/retry`
- `GET /projects/{projectId}/scheduler/tasks/{taskId}/execution`
- `GET /projects/{projectId}/scheduler/tasks/{taskId}/lineage`
- `POST /projects/{projectId}/scheduler/tasks/{taskId}/triage`
- `GET /projects/{projectId}/scheduler/executions`
- `GET /projects/{projectId}/scheduler/runtimes`
- `GET /projects/{projectId}/scheduler/failure-policies`
- `GET /projects/{projectId}/scheduler/agents`
- `GET /projects/{projectId}/board/summary`
- `GET /projects/{projectId}/agents/summary`

## 当前任务主字段

- `project_id`
- `task_id`
- `title`
- `owner_agent`
- `status`
- `type`
- `priority`
- `description`
- `depends_on`
- `input_artifacts`
- `output_artifacts`
- `acceptance_criteria`
- `blocked_reason`
- `result_summary`
- `next_action`
- `current_focus`
- `dispatch_mode`
- `auto_dispatch_enabled`
- `dispatch_status`
- `execution_runtime`
- `execution_session_id`
- `last_dispatch_at`
- `dispatch_attempts`
- `last_dispatch_error`
- `failure_code`
- `failure_signature`
- `coordination_stage`
- `review_decision`
- `parent_task_id`
- `root_task_id`
- `created_at`
- `updated_at`

## 当前 lineage 结构

`GET /scheduler/tasks/{taskId}/lineage` 返回：

- `root_task`
- `ancestors`
- `descendants`
- `siblings`

前端类型定义以 `web/src/types/scheduler.ts` 为准。
Go 兼容层返回字段以 `internal/server/compat_scheduler.go` 为准。

## 当前约束

- 浏览器只认项目化路径。
- 旧的不带 `projectId` 路由只保留给默认项目兼容，不再作为主入口。
- `output_artifacts` 对 CLI 任务是必需的。前端创建任务时如果没填，会自动生成默认报告路径。

## Runtime Reliability 扩展计划

已完成，按 `PRD-OPS-001 / PLAN-OPS-001` 执行。

### PR-1 单进程正式看板（已实现）

已实现内容：

- `GET /board` 直接返回前端页面
- 静态资源回退到 `index.html`
- API 与 board 共用一个 `8080` 服务进程
- `5173` 仅用于前端开发模式

### PR-2 启动恢复与僵尸任务回收（已实现）

已实现内容：

- `internal/server/startup_recovery.go`：启动时扫描 `running / triage / retry_waiting / review_pending / verified` 状态的任务
- `internal/server/execution_reaper.go`：每 15 秒扫描僵尸 `running` 任务并回收
- `internal/store/repository.go`：新增 `ListTasksByState` 方法
- `cmd/server/main.go`：启动时运行 recovery，随后启动 reaper 协程

行为：

- 启动时：`running` 任务如果没有活跃 session → 转 `triage`（带 `recovered_running_task` 事件）
- 运行时：`running` 任务超过 10 分钟未更新或无 session → reaper 自动回收（带 `execution_stalled` 事件）
- `triage / retry_waiting / review_pending / verified` 任务：启动后自动被现有 worker 重新接管

新增事件类型：

- `recovered_running_task`
- `execution_stalled`

### PR-3 Heartbeat / timeout / 健康提示（已实现）

已实现内容：

- `internal/server/system_health.go`：新增 `/api/v1/system/health` 和 `/api/v1/system/workers` 接口
- `internal/server/compat_scheduler.go`：`compatSchedulerTask` 新增 `started_at`、`last_heartbeat_at`、`timeout_at`、`stalled` 字段
- `internal/server/compat_dispatch.go`：dispatch 时设置 heartbeat 和 timeout 字段
- `web/src/types/scheduler.ts`：前端新增 SystemHealth、WorkerStatus 类型
- `web/src/api/schedulerApi.ts`：新增 getSystemHealth、getSystemWorkers API
- `web/src/pages/SchedulerBoard.tsx`：看板新增"后端状态"面板，显示在线/离线、运行时间、worker 状态

新增接口：

- `GET /api/v1/system/health`
- `GET /api/v1/system/workers`

新增字段（execution payload）：

- `started_at`
- `last_heartbeat_at`
- `timeout_at`
- `stalled`

前端行为：

- 每 5 秒轮询 system health
- 显示后端在线/离线状态
- 显示后台 workers 运行状态
- running 任务显示 stalled 标记（超过 5 分钟无更新）
- running 任务卡片显示最后心跳时间 ♥ HH:MM:SS
- 任务详情面板显示启动时间、最后心跳、超时时间、陈旧提示

执行行为：

- 每 30 秒更新 last_heartbeat_at（runCompatExecution 启动 heartbeat goroutine）
- heartbeat 使用 UpdateHeartbeatOnly 方法，只更新 last_heartbeat_at 字段，不覆盖其他字段
- execution_reaper 按 heartbeat 判活（5 分钟阈值）或 timeout_at（超时强制回收）
- startup_recovery 记录 heartbeat_age 用于恢复日志
- 超时事件：execution_timeout（区别于 heartbeat 过期的 execution_stalled）
