# PLAN-OPS-001｜平台运行稳定性与恢复实施计划

- 对应 PRD：`docs/prd/PRD-OPS-001-platform-runtime-reliability.md`
- 目标：把平台从“服务一停就不更新”的开发态，改成“单进程正式看板 + 启动恢复 + 心跳超时回收 + 前端健康提示”。
- 执行方式：拆成 3 个 PR，顺序执行。每个 PR 都有单独验收。
- 文档约束：每个 PR 必须同步更新相关文档。没有文档更新的 PR，不算完成。

## 0. 当前问题

当前已经确认的运行问题：

1. 正式看板还依赖 `5173` 的前端进程。
2. `8080` 停止后，worker 全停，任务状态不再推进。
3. 服务重启后，旧的 `running / triage / retry_waiting / review_pending / verified` 任务恢复不完整。
4. `running` 缺少 heartbeat / timeout，容易形成假运行态。
5. 前端没有明确的后端离线、worker 停止、数据陈旧提示。

## 1. PR-1｜单进程正式看板

工期：1 天

目标：

- Go 服务直接托管 `web/dist`
- 正式看板统一到 `8080`
- `5173` 只保留给开发

改动范围：

- `cmd/server/main.go`
- `internal/server/server.go`
- `internal/server/static_web.go`（新增）
- `web/package.json`
- `web/vite.config.ts`
- `config.yaml`
- `README.md`
- `.planning/STATE.md`
- `.planning/codebase/STACK.md`

要求：

- `GET /board` 直接返回前端页面
- SPA 路由回退到 `index.html`
- 杀掉 `5173` 不影响正式看板
- 开发模式仍可保留 `npm run dev`

验收：

1. 只起 Go 服务。
2. `http://127.0.0.1:8080/board` 可打开。
3. 看板能正常请求 `/api/v1/...`。
4. 文档已更新正式启动方式。

## 2. PR-2｜启动恢复与僵尸任务回收

工期：1.5 天

目标：

- 服务重启后自动接管旧任务
- 清理假 `running`

改动范围：

- `internal/server/startup_recovery.go`（新增）
- `internal/server/execution_reaper.go`（新增）
- `cmd/server/main.go`
- `internal/server/compat_dispatch.go`
- `internal/store/repository.go`
- `docs/project-api-contract.md`
- `.planning/STATE.md`

要求：

- 启动时扫描：
  - `running`
  - `triage`
  - `retry_waiting`
  - `review_pending`
  - `verified`
- 如果任务是 `running`，但没有活跃 session / 心跳 / 进程痕迹：
  - 转 `triage` 或 `retry_waiting`
  - 写恢复事件
- 启动后 worker 要重新接管 `triage / retry_waiting / review_pending / verified`

验收：

1. 人工把任务推进到 `running`。
2. 直接杀掉服务。
3. 重启后 30 秒内状态自动修正。
4. worker 继续接管，不需要手点。

## 3. PR-3｜Heartbeat、timeout 与前端健康提示

工期：1.5 天

目标：

- 运行中任务有活性信号
- 卡死任务自动回收
- 前端明确提示后端健康状态

改动范围：

- `internal/transport/process.go`
- `internal/transport/cli.go`
- `internal/server/compat_scheduler.go`
- `internal/server/system_health.go`（新增）
- `web/src/api/client.ts`
- `web/src/api/schedulerApi.ts`
- `web/src/pages/SchedulerBoard.tsx`
- `web/src/types/scheduler.ts`
- `docs/project-api-contract.md`
- `docs/prd/PRD-OPS-001-platform-runtime-reliability.md`

要求：

- execution 增加：
  - `started_at`
  - `last_heartbeat_at`
  - `timeout_at`
  - `stalled`
- 运行中每隔 N 秒更新 heartbeat
- 超时没 heartbeat：
  - 自动写 `execution_stalled`
  - 任务转 `triage`
- 前端显示：
  - 后端在线 / 离线
  - worker 状态
  - 数据陈旧提示
  - running 任务最后心跳时间

验收：

1. 模拟一个卡死执行。
2. 超时后自动退出 `running`。
3. 看板显示“后端离线”或“数据陈旧”。
4. 不再假装任务还在正常执行。

## 4. 并行分工建议

### Agent A

负责 PR-1。
写入范围只限：

- `cmd/server/main.go`
- `internal/server/server.go`
- `internal/server/static_web.go`
- `web/package.json`
- `web/vite.config.ts`
- `config.yaml`
- `README.md`
- `.planning/STATE.md`
- `.planning/codebase/STACK.md`

### Agent B

负责 PR-2。
写入范围只限：

- `internal/server/startup_recovery.go`
- `internal/server/execution_reaper.go`
- `internal/server/compat_dispatch.go`
- `internal/store/repository.go`
- `cmd/server/main.go`
- `docs/project-api-contract.md`
- `.planning/STATE.md`

### Agent C

负责 PR-3。
写入范围只限：

- `internal/transport/process.go`
- `internal/transport/cli.go`
- `internal/server/system_health.go`
- `internal/server/compat_scheduler.go`
- `web/src/api/*`
- `web/src/pages/SchedulerBoard.tsx`
- `web/src/types/scheduler.ts`
- `docs/project-api-contract.md`
- `docs/prd/PRD-OPS-001-platform-runtime-reliability.md`

## 5. 不要做的事

- 不要改业务项目仓库。
- 不要把正式看板继续建立在 `npm run dev` 上。
- 不要只加前端轮询，不补后端恢复。
- 不要把“看板不更新”误判成 lineage 问题。
- 不要顺手扩成新的大重构。

## 6. 最终总验收

以下 5 条全部满足才算收口：

1. 只起 `8080` 就能打开看板。
2. 服务重启后，旧任务会自动恢复推进。
3. 假 `running` 会被自动回收。
4. 前端能提示后端离线和数据陈旧。
5. 不再依赖 `5173 dev server` 才能看进度。
