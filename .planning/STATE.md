---
gsd_state_version: 1.2
milestone: v2.1
milestone_name: runtime-reliability
status: completed
stopped_at: PR-OPS-003 / PR-3 fully complete
last_updated: "2026-04-12T01:00:00+08:00"
last_activity: 2026-04-12 -- Fixed heartbeat isolation + timeout enforcement
progress:
  total_phases: 3
  completed_phases: 3
  total_plans: 2
  completed_plans: 2
  percent: 100
---

# Project State

## Current Position

Phase: Runtime Reliability
Status: completed

所有 PR 已完成并修复 review findings：

- PR-1：Go 直接托管 web/dist，正式看板在 8080/board
- PR-2：启动恢复扫描 + 僵尸 running 回收 + execution reaper
- PR-3（完整收口）：
  - system health API + heartbeat loop + 前端心跳显示
  - **修复 P1**：heartbeat 使用 UpdateHeartbeatOnly，不覆盖其他字段
  - **修复 P2**：reaper 执行 timeout_at，超时强制回收

## PR-3 最终修复

### P1: Heartbeat isolation

问题：heartbeat goroutine 持有 payload 引用，整包 persistCompatPayload 会覆盖并发更新。

修复：
- `internal/store/repository.go`：新增 `UpdateHeartbeatOnly` 方法
- 只更新 `last_heartbeat_at` 字段，不覆盖其他字段
- `internal/server/compat_dispatch.go`：heartbeat goroutine 使用新方法

### P2: Timeout enforcement

问题：timeout_at 只是展示字段，reaper 不执行超时回收。

修复：
- `internal/server/execution_reaper.go`：`isZombie` 检查 timeout_at
- 超过 timeout_at 的任务视为 zombie，强制回收
- 新增事件类型 `execution_timeout`（区别于 `execution_stalled`）

## Current Validation Baseline

- `go build ./...` 通过
- `go test ./...` 通过
- `web`: `npm run build` 通过
- `web`: `npm run lint` 通过

## Milestone Summary

runtime-reliability milestone 完整收口：

1. 只启动 8080 就能打开看板 ✓
2. 服务重启后，旧任务会自动恢复推进 ✓
3. 假 running 会被自动回收（按 heartbeat + timeout 判活） ✓
4. 前端能提示后端离线和 worker 状态 ✓
5. running 任务显示心跳时间和陈旧提示 ✓
6. 执行中有周期性 heartbeat 更新（只更新心跳字段） ✓
7. timeout_at 被真正执行，超时任务强制回收 ✓
8. 不再依赖 5173 dev server ✓