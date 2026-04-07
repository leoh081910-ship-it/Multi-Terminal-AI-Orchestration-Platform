---
type: quick-summary
id: 260407-w7l
title: Phase 01 PERS-04 task card_json source-of-truth fix
status: completed
project: 多终端 AI 编排平台
completed_at: 2026-04-07
plan: .planning/quick/260407-w7l-phase-01-pers-04-task-card-json-card-jso/260407-w7l-PLAN.md
---

# Quick Summary: Phase 01 PERS-04 task card_json source-of-truth fix

## Outcome
已补上 Phase 01 的 PERS-04 缺口：task 的业务字段现在以 `card_json` 为唯一业务真相；repository 写入时从 `card_json` 派生结构化列；server 读取时统一返回基于 `card_json` 重建的 `TaskView`，并在重建失败时回退到结构化列最小视图，避免接口中断。

## Files Changed
- `internal/store/repository.go`
- `internal/server/server.go`
- `internal/server/server_test.go`

## What Changed

### 1) Repository write semantics now derive from `card_json`
在 `internal/store/repository.go` 中收紧了 `deriveTaskCard()`：
- `card_json` 为空时报错 `invalid card_json: empty`
- `card_json` 非法 JSON 时报错 `invalid card_json: ...`
- `id` / `dispatch_ref` / `transport` 缺失时报错
- 业务字段不再从外层重复字段回填，而是只从 `card_json` 或现有持久化记录派生
- `state` 缺失时默认 `queued`

这保证了 create/update 的业务真相来自 `card_json`，结构化列只作为派生镜像和运行时查询字段。

### 2) Server read semantics now return mapped task views
在 `internal/server/server.go` 中，task 相关接口继续统一走 `mapTaskView()` / `mapTaskViews()`：
- `GET /api/tasks/{id}`
- `GET /api/tasks`
- `GET /api/dispatches/{dispatchRef}/tasks`
- `GET /api/tasks/stats` 的 `recent`

同时为 `mapTaskView()` 增加了 fallback：如果 `BuildTaskView()` 因损坏 `card_json` 失败，会记录 warning 并返回基于结构化列的最小 `TaskView`，保证接口可诊断且不中断。

### 3) Tests updated for strict source-of-truth behavior
在 `internal/server/server_test.go` 中：
- 修正测试 helper，生成包含 `dispatch_ref` / `state` / `transport` / `wave` / `topo_rank` 的合法 `card_json`
- 增加 PERS-04 HTTP 回归测试：
  - `TestServerTaskResponsesUseCardJSONBusinessFields`
  - `TestServerTaskListsUseCardJSONBusinessFields`
  - `TestServerTaskStatsRecentUsesMappedTaskView`
  - `TestServerRetryFlowPreservesPhase1TaskBehavior`

## Verification Run
已执行并通过：

```bash
go test ./internal/store -run 'TestRepository(TaskCardJSONSourceOfTruthOnCreate|TaskCardJSONSourceOfTruthOnUpdate|TaskCardJSONRejectsInvalidCardJSON|UpdateTaskStateStillUsesStructuredColumns)$' -count=1
```

```bash
go test ./internal/server -run 'TestServer(TaskResponsesUseCardJSONBusinessFields|TaskListsUseCardJSONBusinessFields|TaskStatsRecentUsesMappedTaskView|RetryFlowPreservesPhase1TaskBehavior)$' -count=1
```

```bash
go test ./internal/store ./internal/server ./cmd/server -count=1
```

## Result
- `internal/store` tests passed
- `internal/server` tests passed
- `cmd/server` has no test files

## Scope Check
本次 quick task 只覆盖 PERS-04 缺口和必要回归：
- 未新增 delete API
- 未修改 schema
- 未扩展到其它 phase
- 未改动 wave/event 设计，只修复 task card source-of-truth 与对应接口输出
