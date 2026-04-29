# PLAN-DA-001｜动态协调与审核 Agent 系统实施计划

- 对应 PRD：`docs/prd/PRD-DA-001-dynamic-coordinator-reviewer.md`
- 目标：把失败任务处理从人工干预改成平台自动闭环
- 执行方式：按阶段完成，每阶段都有可回归的最小验收
- 实施状态：已实现（2026-04-10）

## 0. 现状问题清单

当前已确认问题：

1. 失败任务停在 `failed`，没有后续动作。
2. 失败原因只有字符串，没有结构化分类。
3. 补救任务不会自动生成。
4. 修复完成后原任务不会自动 retry。
5. 成功执行后没有独立审核层。
6. 当前真实任务定义中存在错误产物路径和 no-op 误判。

## 1. 开发范围

### 1.1 后端

- 失败分类
- failure orchestrator
- retry worker
- review worker
- agent registry 扩展
- 新状态与新任务类型
- lineage API

### 1.2 前端

- 失败结构化展示
- 任务链路展示
- 系统角色展示
- 自动协作阶段展示

### 1.3 当前任务配置修补

- 修正已有失败任务的产物定义
- 让第一批回归样本可复现

## 2. 阶段计划

### 阶段 1：失败结构化

工期：2 天

目标：

- 把失败从一段字符串，改成结构化失败模型。

改动模块：

- `internal/server/compat_dispatch.go`
- `internal/server/compat_scheduler.go`
- `internal/store/repository.go`

输出：

- `failure_code`
- `failure_signature`
- `error_stage`
- `retryable`
- `suggested_action`

验收：

- `empty_artifact_match`
- `nothing to commit`
- `exit status 1`
- `dependency_failed`

这四类错误能稳定映射成结构化结果。

---

### 阶段 2：协调层

工期：3 天

目标：

- 失败后自动进入 triage，并自动生成补救任务。

新增模块：

- `internal/server/failure_orchestrator.go`
- `internal/server/failure_policy.go`

输出：

- triage worker
- 自动分单
- 补救任务去重

验收：

- 原任务失败后自动生成一条补救任务
- 同一错误签名不会重复派生

---

### 阶段 3：自动重试层

工期：2 天

目标：

- 补救任务完成后，原任务自动 retry。

新增模块：

- `internal/server/retry_worker.go`

输出：

- `retry_waiting` 真正跑起来
- backoff 生效
- 未修复完成前不提前重试

验收：

- 补救任务 done 后，原任务自动 retry
- 不需要人工点 Retry

---

### 阶段 4：审核层

工期：3 天

目标：

- 执行成功后进入 review，而不是直接 done。

新增模块：

- `internal/server/review_worker.go`

输出：

- `review_pending`
- `code-review`
- `rework`

验收：

- 代码任务执行成功先进入 review
- 审核通过才 done
- 审核拒绝会产生返工任务

---

### 阶段 5：前端联调

工期：2 天

目标：

- 看板能完整表达自动协作链路。

改动模块：

- `web/src/pages/SchedulerBoard.tsx`
- `web/src/types/scheduler.ts`
- `web/src/api/*`

输出：

- 失败结构化展示
- 任务 lineage 展示
- 系统角色展示
- 自动阶段展示

验收：

- 看板能看出谁失败、谁接手、当前卡在哪

## 3. 首批专项修补

这 4 条要优先修，不然后面机制接上也会继续误判。

### 3.1 `TS-BF5C2847`

问题：

- 产物路径写错

修正：

- `app/repositories/product_repository.py`
- 改成
- `app/repositories/postgres_product_repository.py`

### 3.2 `TS-A5A59E27`

问题：

- 要求报告产物，但执行里没生成

修正：

- 明确输出
- `.orchestrator/reports/OPS-005-workspace-cleanup.md`

### 3.3 `TS-77F7A1C4`

问题：

- `nothing to commit, working tree clean`
- 不应按普通失败处理

修正：

- 进入 `noop-review`

### 3.4 `TS-27A52716`

问题：

- `exit status 1`
- 需要从日志定位真因

修正：

- 用执行日志定位失败点

## 4. 任务拆分建议

### Epic A：失败结构化

- A1 定义失败模型
- A2 后端落字段
- A3 API 返回增强
- A4 前端显示失败结构

### Epic B：协调层

- B1 定义 triage 状态
- B2 failure orchestrator
- B3 failure policy
- B4 自动派生任务
- B5 去重与签名

### Epic C：重试层

- C1 retry worker
- C2 backoff 判定
- C3 修复完成后自动 retry

### Epic D：审核层

- D1 review_pending
- D2 review worker
- D3 审核通过流转
- D4 审核拒绝返工

### Epic E：前端联调

- E1 lineage 展示
- E2 系统角色展示
- E3 自动阶段展示
- E4 当前失败样本回归

## 5. 风险与控制

### 风险 1：无限派生

控制：

- 同一 root task 自动补救次数上限 2

### 风险 2：无限重试

控制：

- 同一原任务自动 retry 上限 3

### 风险 3：同一错误重复分单

控制：

- 用 `failure_signature` 去重

### 风险 4：前端状态表达不清

控制：

- 先补 lineage API
- 再补看板展示

## 6. 阶段验收清单

### 阶段 1 验收

- 错误可分类
- API 返回结构化错误

### 阶段 2 验收

- 失败后自动 triage
- 自动生成补救任务

### 阶段 3 验收

- 补救完成后自动 retry

### 阶段 4 验收

- 执行成功后先 review

### 阶段 5 验收

- 看板可展示完整链路

## 7. 最终回归用例

必须完成这 4 类回归：

1. artifact 路径错误
2. nothing to commit
3. command exit status 1
4. 依赖任务恢复后继续推进

## 8. 最终验收

最终验收按这 7 条收：

1. 失败任务 30 秒内进入 triage。
2. 系统自动生成正确补救任务。
3. 补救任务完成后原任务自动 retry。
4. 代码任务成功后进入 review。
5. 系统不会无限派生。
6. 系统不会无限重试。
7. 看板能展示完整链路和当前阶段。
