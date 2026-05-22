# Phase 5: Interface - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-16
**Phase:** 05-interface
**Areas discussed:** 信息架构

---

## 信息架构

| Option | Description | Selected |
|--------|-------------|----------|
| 控制台优先 | 保留总览首页，核心入口收敛成正式控制流 | ✓ |
| 任务中心 | 围绕任务列表/详情组织全部入口 | |
| 运营面板 | 总览仪表盘作为第一主入口 | |

**User's choice:** 控制台优先
**Notes:** 现有代码已有首页壳层、看板和详情页，适合收敛成总览首页 + 核心控制页结构。

---

## 首页角色

| Option | Description | Selected |
|--------|-------------|----------|
| 总览落地页 | `/` 展示项目概览、关键统计、运行健康、待处理事项 | ✓ |
| 直接进看板 | `/` 直接作为调度看板 | |
| 个人工作台 | `/` 展示当前焦点和个人处理项 | |

**User's choice:** 总览落地页
**Notes:** 首页负责平台总览，`/board` 专注任务推进。

---

## 正式核心入口

| Option | Description | Selected |
|--------|-------------|----------|
| 调度看板 | 正式任务流主入口 | ✓ |
| Wave 管理 | 正式 Wave 页面与 dispatch/wave 状态入口 | ✓ |
| 事件日志 | 正式日志浏览入口 | ✓ |
| 任务详情 | 完整 Task Card / 状态 / 执行 / 谱系细节页 | ✓ |

**User's choice:** 调度看板、Wave 管理、事件日志、任务详情
**Notes:** 这四个入口共同覆盖了本阶段 roadmap 里的任务 CRUD、Wave、事件和实时状态核心能力。

---

## 扩展页处理

| Option | Description | Selected |
|--------|-------------|----------|
| 降级为次级入口 | timeline/goals/agents/swimlane/org/knowledge 改为“更多”或二级导航 | ✓ |
| 全部保留一层 | 继续全部挂在主导航 | |
| 先隐藏扩展页 | 只保留核心四页，其余暂时移除可见性 | |

**User's choice:** 降级为次级入口
**Notes:** 保留现有页面能力，但不让正式一层导航继续过载。

---

## Claude's Discretion

- 正式导航的具体分组与命名
- “更多”或二级导航的具体交互方式
- 首页总览模块的具体视觉布局

## Deferred Ideas

None
