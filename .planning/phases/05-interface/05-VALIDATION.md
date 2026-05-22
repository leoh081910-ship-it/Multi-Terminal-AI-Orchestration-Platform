---
phase: 5
slug: interface
status: executed_green
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-16
updated: 2026-05-19
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for Interface execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Backend framework** | go test |
| **Frontend framework** | npm run build |
| **Quick backend command** | `go test ./internal/server ./internal/store -count=1` |
| **Quick frontend command** | `npm run build` |
| **Estimated runtime** | backend ~30-60s, frontend ~15-30s |

---

## Sampling Rate

- **After every backend-affecting task commit:** `go test ./internal/server ./internal/store -count=1`
- **After every frontend-affecting task commit:** `npm run build`
- **After every plan wave:** run both backend + frontend validation commands
- **Before `/gsd:verify-work`:** backend + frontend validation must be green
- **Max feedback latency:** < 5 minutes per plan slice

---

## Per-Plan Verification Map

| Plan | Wave | Requirements | Test Type | Automated Command | Manual Check | Result |
|------|------|--------------|-----------|-------------------|--------------|--------|
| 05-interface-01 | 1 | UI-05, UI-01 | frontend build + shell smoke | `npm run build` | 打开首页和导航，确认一级/二级入口分层正确 | GREEN — dashboard route served, build passed, merge queue status added after verification gap closure |
| 05-interface-02 | 1 | API-01, UI-03, UI-07 | backend + frontend integration | `go test ./internal/server ./internal/store -count=1` + `npm run build` | 打开 `/waves`、`/events`，确认查询/过滤/跳转可用 | GREEN — backend/frontend validation passed; `/waves` and `/events` production routes returned 200 |
| 05-interface-03 | 2 | API-02, API-03, UI-01, UI-02, UI-04, UI-06 | frontend build + realtime/manual flow | `npm run build` | 创建/编辑任务、打开详情页、确认 websocket 驱动刷新 | GREEN with manual follow-up — reversible create/edit/get/delete API smoke passed; WebSocket wiring verified by code/endpoint alignment, browser observation remains manual |

---

## Manual-Only Verifications

| Behavior | Why Manual | Test Instructions | Execution Result |
|----------|------------|-------------------|------------------|
| 首页总览与看板职责分离是否清晰 | 需要真实浏览信息层级和视觉优先级 | 启动服务，访问 `/` 和 `/board`，确认首页是总览、看板是任务推进，不再互相混淆 | PARTIAL — production static routes returned 200; visual browser pass not recorded |
| 主导航 / 次级导航是否符合 locked IA | 自动化无法判断 IA 是否过载 | 检查 `Layout` 实际交互，确认核心四入口在主导航，扩展页降级为次级入口 | PARTIAL — frontend build passed; visual browser pass not recorded |
| Wave 管理页 seal 和任务跳转黄金路径 | 涉及多页面真实交互 | 在 `/waves` 中浏览 wave，执行 seal，跳转到相关任务或看板 | PARTIAL — `/waves` route and API smoke passed; seal mutation deferred to disposable sample data |
| 事件日志页历史查询 + 实时追加 | 需要同时观察 HTTP 历史数据与 WS 流 | 打开 `/events`，筛选 task 或 dispatch，再触发任务状态变化，确认列表实时追加 | PARTIAL — `/scheduler/events` returned 200 and WS hook is wired; browser realtime observation not recorded |
| 任务创建 / 编辑正式表单可用性 | 需要检查字段交互、校验与提交流 | 在 `/board` 创建任务并编辑关键字段，确认校验和保存都合理 | GREEN for API path — reversible create/update/get/delete smoke passed; visual form interaction not recorded |

---

## Executed Validation Results

| Check | Result | Evidence |
|-------|--------|----------|
| Focused backend regression | PASS | `go test ./internal/server ./internal/store -count=1` passed after gap closure and static route patch. |
| Frontend production build | PASS | `npm --prefix "E:/04-Claude/Projects/多终端 AI 编排平台/web" run build` passed after gap closure and static route patch. |
| Health and project scheduler API smoke | PASS | `/health`, `/api/v1/system/health`, project board summary, tasks, waves, and events returned 200 JSON. |
| Production SPA route smoke | PASS | `/`, `/board`, `/waves`, `/events`, and `/tasks/example-task` returned 200 HTML from Go static hosting. |
| Dashboard merge queue JSON contract | PASS | Board summary included `merge_queue_count` and `merge_queue_tasks` as expected. |
| Reversible scheduler CRUD smoke | PASS | Create returned 201, update/get/delete returned 200, follow-up get returned 404; smoke task was cleaned up. |
| WebSocket endpoint alignment | PASS by implementation | Frontend default points to `ws://localhost:8080/api/v1/ws`, matching backend route. |

---

## Wave 0 Requirements

- [x] 所有计划都绑定到明确 requirement IDs
- [x] 后端验证命令存在
- [x] 前端验证命令存在
- [x] 手动 UI 黄金路径明确列出
- [x] 执行后补充每计划实际结果与 green/red 状态

---

## Validation Sign-Off

- [x] All tasks have automated verify or explicit manual follow-up
- [x] Frontend and backend validation commands are both defined
- [x] No watch-mode flags
- [x] `nyquist_compliant: true` set in frontmatter
- [x] Execution results recorded after implementation

**Approval:** executed validation is green for code/API/build evidence, with non-blocking manual follow-up for visual browser and live WebSocket observation.
