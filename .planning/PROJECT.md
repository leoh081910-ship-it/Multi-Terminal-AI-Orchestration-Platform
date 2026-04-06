# 多终端 AI 编排平台

## What This Is

一个单人、本地优先的多 AI 任务编排平台。用户通过 CLI 或 Web UI 提交 Task Card，平台按 Wave 分批、按依赖拓扑排序，调度多个 AI Agent（首个为 Claude Code CLI）在隔离 workspace 中并行执行任务，最后自动合并工件到主仓库。平台内置逆向工程专项任务类型，支持 IDA Pro + Frida 的可量化验证闭环。

## Core Value

任务从提交到合并全流程自动化——用户定义"做什么"，平台负责"怎么做"：隔离执行、依赖调度、冲突回避、工件合并、失败重试。

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] SQLite 三表持久化（tasks / events / waves），card_json 为业务字段权威来源
- [ ] Task Card 标识系统（task_id + dispatch_ref + source_ref 三层标识）
- [ ] Wave 一致性管理（入队自动 upsert、seal 后锁定、seal 前不得路由）
- [ ] 依赖与冲突计算（depends_on 跨 wave 校验、conflicts_with 同 wave 内计算、topo_rank 排序）
- [ ] 13 态状态机（8 主态 + 5 异常态）+ 严格转换规则
- [ ] CLI Transport（git worktree 隔离 + files_to_modify 白名单抽取）
- [ ] API Transport（完整文件工件写入 artifacts/ + 同步到隔离目录）
- [ ] 合并队列（单消费者串行，topo_rank + created_at 排序，依赖全部 done 后才消费）
- [ ] 重试与恢复（max_retries=2，30s/60s 退避，终态保护，依赖失败传播）
- [ ] 逆向专项任务 reverse_static_c_rebuild（IDA + Frida 可量化循环，match_rate=100% 才完成）
- [ ] Connector 接口（discoverTasks / hydrateContext / ackResult / writeBackArtifacts）
- [ ] GSD Connector（首个 Connector，从 PLAN 生成 Task Card，回写 SUMMARY/STATE/ROADMAP/VERIFICATION）
- [ ] Go HTTP API 服务（RESTful 接口，SQLite 存储，WebSocket 状态推送）
- [ ] React + Vite Web UI（完整管理面板：任务 CRUD、Wave 管理、状态监控、事件日志、实时更新）
- [ ] 抽象 Agent 接口（定义 Runner 契约，Claude Code CLI 为首个实现）
- [ ] Windows 兼容（路径长度/空格/中文检查，worktree 权限处理）

### Out of Scope

- 多用户 / 权限系统 — v1 单用户本地运行
- 分布式部署 — 纯本地，不涉及远程服务器集群
- apply_failed 自动重试 — v1 必须人工处理
- 非 Go 语言的存储后端 — v1 固定 SQLite
- 移动端适配 — Web UI 只做桌面端
- OAuth / 外部认证 — 本地工具无需认证

## Context

- PRD v24 已完整定义数据模型、状态机、Transport 流程、逆向专项规则和测试计划
- GSD（Get Shit Done）是平台的首个 Connector，不是平台内核
- 逆向分析循环协议（RE Analysis Loop Protocol）是项目内置的逆向工程工作规范
- 平台目标是成为 AI Agent 编排的通用基础设施，GSD 是第一个消费者
- 本地运行意味着可以假设文件系统访问、Git CLI 可用、SQLite 无并发写冲突

## Constraints

- **Tech Stack**: Go 后端 + React/Vite 前端 — PRD 未指定技术栈，通过讨论确定
- **Storage**: SQLite only — v1 不可配置，单文件数据库
- **OS**: Windows 11 为主要平台 — 必须处理长路径、中文路径、符号链接权限
- **Scale**: 10-20 并发任务 — 中等规模，合并队列串行但路由可并行
- **Agent**: Claude Code CLI 首个实现 — 但必须通过抽象接口支持扩展

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go 后端 | 强类型、并发模型、CLI 生态成熟、SQLite 驱动稳定 | — Pending |
| React + Vite 前端 | 完整管理 UI 需要复杂表单/表格/实时更新，React 生态最成熟 | — Pending |
| 抽象 Agent 接口 | v1 用 Claude Code CLI，但接口设计支持未来扩展其他 Runner | — Pending |
| SQLite 存储 | v1 单用户本地运行，SQLite 零运维、事务一致性、单文件部署 | — Pending |
| Git worktree 隔离 | CLI Transport 的标准隔离方案，与开发工作流天然集成 | — Pending |
| WebSocket 状态推送 | Web UI 需要实时任务状态更新，SSE 也可但 WebSocket 更灵活 | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-06 after initialization*
