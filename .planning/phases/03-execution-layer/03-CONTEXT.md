# Phase 3: Execution Layer - Context

**Gathered:** 2026-05-17
**Status:** Ready for planning

<domain>
## Phase Boundary

交付任务执行层本身：CLI/API transport 的正式执行语义，以及 reverse engineering 专项任务在平台中的执行闭环。范围内是把现有 transport、reverse executor、dispatch wiring、事件记录、恢复语义和工件语义接通并收口，不新增超出 roadmap 的新能力或新状态模型。

</domain>

<decisions>
## Implementation Decisions

### Execution ingress
- **D-01:** Phase 3 不新建第二套正式执行入口；正式入口保持在现有 `internal/server` 调度 / dispatch 链路。
- **D-02:** server 在同一入口内按任务类型和 transport 分流到具体执行器；`reverse_static_c_rebuild` 任务在该链路内分支到 reverse executor。
- **D-03:** `internal/executor/coordinator.go` 若继续存在，只作为内部编排组件，不作为新的系统边界或对外 contract。

### Reverse execution model
- **D-04:** reverse 任务外层继续使用平台标准状态机；专项循环全部收敛在单个 `running` 窗口内完成，不新增 reverse 专属外层状态。
- **D-05:** reverse 进度通过 `loop_iteration` 事件和 `loop_iteration_count` 暴露，而不是通过新的状态名暴露。
- **D-06:** 只有达到 `match_rate=100%` 且最终工件校验通过时，才允许 `running -> patch_ready`。

### Reverse failure layering
- **D-07:** 编译失败、静态运行失败、Frida oracle 失败、diff 失败、oracle mismatch 等属于 reverse 内部循环失败；这些失败只记录事件，维持内部 `running -> running` 语义，不直接触发外层状态迁移，也不消耗外层 retry。
- **D-08:** 只有循环次数耗尽或出现不可恢复环境错误时，才触发外层 `running -> retry_waiting`。
- **D-09:** 外层主动重试时 `loop_iteration_count` 归零；进程恢复时保留计数，但从循环第 1 步重新开始。

### Environment and platform boundaries
- **D-10:** Windows 11 本地优先是硬边界，不是“尽量兼容”；Phase 3 必须显式处理路径长度、中文路径、空格路径、worktree 可用性与符号链接权限等平台约束。
- **D-11:** reverse 所需外部能力（如 IDA MCP、Frida、目标文件、oracle 引用）视为必需依赖，不做静默降级。
- **D-12:** 缺必填任务字段在入队 / 路由前阻断；缺外部工具或运行环境在执行前明确失败，并进入可追踪失败路径。

### API artifact semantics
- **D-13:** API transport 的权威结果是平台接收到的文件工件集合，而不是远端 agent 环境中的某个路径。
- **D-14:** 平台先把 API 返回工件写入 `artifacts/{task_id}/`，再把这些工件同步到任务隔离目录，供统一的后续验证与合并流程使用。
- **D-15:** 自 `patch_ready` 起，CLI/API 共享同一后续流程和同一工件语义。
- **D-16:** API 工件同步失败视为 transport failure，进入 `retry_waiting`，不保留半完成中间态。

### Scope control
- **D-17:** Phase 3 是执行层收口，不扩展新 transport 模式、新外部协调入口、或新的任务状态模型。

### Claude's Discretion
- Windows 路径检查在代码中采用 helper、validator 还是 transport 内部局部实现
- reverse executor 注入事件写入能力的具体接口形状
- API transport 工件同步的目录组织细节，只要满足平台托管语义
- enqueue 前置校验落在 engine、server 还是 repository 入口附近的具体代码位置

</decisions>

<specifics>
## Specific Ideas

- 执行入口统一、专项循环内聚、环境失败前置、工件语义平台托管。
- reverse 专项复杂度应留在执行器内部，通过事件流解释循环进度，不把专项细节扩散成全局状态机变体。
- API transport 应收敛到与 CLI transport 一致的治理、审计、恢复模型，而不是让远端 agent 直接影响主仓库。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap and phase scope
- `.planning/ROADMAP.md` — Phase 3 goal, requirement mapping and success criteria for execution layer delivery.
- `.planning/REQUIREMENTS.md` — `TRAN-01`~`TRAN-07` and `REVR-01`~`REVR-14` are the authoritative requirements for this phase.
- `.planning/PROJECT.md` — Windows-first, local-first, SQLite-only and single-user constraints that shape execution-layer decisions.
- `.planning/STATE.md` — Current project baseline and already completed integration/interface work that Phase 3 should not redesign.

### Phase-local analysis
- `.planning/phases/03-execution-layer/03-RESEARCH.md` — Existing implementation inventory, identified gaps, and concrete integration targets for Phase 3.

### Codebase architecture context
- `.planning/codebase/ARCHITECTURE.md` — Backend layering and duplicate API-surface cautions relevant to where execution wiring should land.
- `.planning/codebase/STRUCTURE.md` — Concrete package map for `internal/server`, `internal/transport`, `internal/executor`, `internal/reverse`, and related integration points.
- `.planning/codebase/INTEGRATIONS.md` — External integration surfaces and risks, especially GSD connector and reverse tooling boundaries.

### Existing code contracts
- `internal/server/server.go` — Current HTTP / dispatch surface and the most likely execution ingress that must remain canonical.
- `internal/transport/cli.go` — Existing CLI transport behavior and artifact extraction semantics.
- `internal/transport/api.go` — Existing API transport behavior and the starting point for platform-managed artifact sync semantics.
- `internal/executor/coordinator.go` — Current internal coordination scaffold that must remain internal, not become a second ingress.
- `internal/reverse/executor.go` — Reverse loop implementation, artifact writing, and loop-state behavior that Phase 3 must wire into the platform lifecycle.
- `internal/reverse/config.go` — Reverse task field contract and validation rules used to reject malformed reverse tasks before routing.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/transport/cli.go` — 已具备 worktree 执行、白名单工件提取和 artifact 输出，是 CLI transport 正式语义的主要基础。
- `internal/transport/api.go` — 已具备隔离目录执行和 artifact 输出，是 API transport 收口到平台托管语义的主要基础。
- `internal/reverse/executor.go` — 已实现 reverse 循环主体、diff 计算、artifact 输出和分析状态持久化，适合作为 reverse 执行核心继续使用。
- `internal/reverse/config.go` — 已具备 reverse 专项字段校验，可直接用于 enqueue / routing 前置校验。
- `internal/executor/coordinator.go` — 已有 transport / reverse 分发骨架，可保留为内部编排辅助层。

### Established Patterns
- 当前系统已经以 `internal/server` 路径承载任务执行、事件记录、状态推进与恢复治理；Phase 3 应在该链路内接线，而不是平行扩张系统边界。
- transport 已经把 CLI 和 API 结果标准化为平台可消费的执行结果；Phase 3 应进一步统一它们在 `patch_ready` 之后的后处理语义。
- reverse 现有实现已经采用“外层生命周期 + 内层循环”的方向；Phase 3 需要把事件与失败分层补齐，而不是改造成独立状态机。

### Integration Points
- `internal/server/server.go` — reverse 分支、dispatch wiring、以及外层失败 / retry 接入点。
- `internal/engine/` — reverse 任务前置校验与状态迁移规则的核心接入点。
- `internal/store/repository.go` — `loop_iteration` 事件、外层状态推进和可恢复失败原因的持久化接入点。
- `internal/transport/api.go` and `internal/transport/cli.go` — 平台托管工件语义、隔离目录同步和统一后处理的落点。
- `internal/reverse/executor.go` — reverse 内部循环、事件回调、最终工件校验、错误分层和恢复语义的实现落点。

</code_context>

<deferred>
## Deferred Ideas

- 新 transport 模式或新的 agent execution ingress — 超出 Phase 3 收口范围。
- reverse 专属外层状态模型或 UI 可视化增强 — 若需要，属于后续 phase 的可视化 / observability 工作。
- 对外部工具链（IDA / Frida）的高级自动安装、自动修复或降级运行 — 超出本阶段边界。

</deferred>

---

*Phase: 03-execution-layer*
*Context gathered: 2026-05-17*
