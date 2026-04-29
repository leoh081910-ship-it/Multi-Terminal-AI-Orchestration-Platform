# AI Orchestration Platform

多项目 AI 编排平台。

当前边界：

- `cmd/server` 是唯一后端入口。
- `web/` 是唯一平台前端。
- `web/_legacy_20260410/` 是归档的旧前端，不再继续开发。
- `E:\06 gemini\tiktok_shop_xuanping` 现在作为首个托管项目接入平台。

## 当前能力

- 多项目调度路由：`/api/v1/projects/{projectId}/...`
- 项目级 runtime / workspace / artifact / merge queue
- 自动派发、failure orchestration、retry、review、TTL cleanup
- 执行记录、任务谱系、兼容任务看板
- 项目注册与切换

## 当前运行现实

平台功能主线已经接通。

当前真正的稳定性问题不在任务谱系，也不在调度规则本身。
问题在进程生命周期：

- 正式看板还依赖单独的前端进程时，`5173` 一停，页面就不再刷新。
- `8080` 一停，auto dispatcher、failure orchestrator、retry worker、review worker、merge queue 会一起停。
- 服务重启后，旧的 `running / triage / retry_waiting / review_pending` 任务还缺少统一恢复入口。

这条治理线单独记录在：

- `docs/prd/PRD-OPS-001-platform-runtime-reliability.md`
- `docs/plans/PLAN-OPS-001-platform-runtime-reliability.md`

## 启动

### 正式运行（单进程）

```powershell
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
go run ./cmd/server -config config.yaml
```

默认监听：`http://127.0.0.1:8080`

正式看板入口：`http://127.0.0.1:8080/board`

### 前端开发模式（仅开发）

```powershell
cd "E:\04-Claude\Projects\多终端 AI 编排平台\web"
npm install
npm run dev
```

默认监听：`http://127.0.0.1:5173`

### 前端构建

```powershell
cd "E:\04-Claude\Projects\多终端 AI 编排平台\web"
npm run build
```

Windows 中文路径下的构建，当前通过 ASCII junction 路径执行：

- `E:\04-Claude\Projects\ai-orchestration-platform -> E:\04-Claude\Projects\多终端 AI 编排平台`

`web/package.json` 里的 `dev / build / preview` 都已经显式指向这个 ASCII 路径下的 `vite.config.ts`。

正式看板不再依赖 `npm run dev`，由 Go 服务直接托管 `web/dist` 并通过 `http://127.0.0.1:8080/board` 提供。

## 下一轮稳定性改进

本轮不做大重构，只做三件事：

- PR-1：Go 直接托管 `web/dist`，把正式看板收成单进程。
- PR-2：服务启动恢复和僵尸任务回收。
- PR-3：execution heartbeat、timeout、前端健康提示。

每个 PR 都必须同步更新文档。没有文档更新的 PR，不算完成。

## 关键文档

- `docs/project-api-contract.md`
- `docs/prd/PRD-DA-001-dynamic-coordinator-reviewer.md`
- `docs/plans/PLAN-DA-001-dynamic-coordinator-reviewer.md`
- `docs/prd/PRD-OPS-001-platform-runtime-reliability.md`
- `docs/plans/PLAN-OPS-001-platform-runtime-reliability.md`
- `config.yaml`
- `.planning/STATE.md`
- `.planning/codebase/STACK.md`

## 首个托管项目

- `project_id`: `tiktok_shop_xuanping`
- `repo_root`: `E:\06 gemini\tiktok_shop_xuanping`
