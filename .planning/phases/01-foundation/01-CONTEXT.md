# Phase 1: Foundation - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

SQLite 数据库持久化 + Go 项目基础设施。创建三表 schema（tasks, events, waves）、实现 Task Card CRUD、Wave 管理操作（create, query, seal）、原子事务写入（事件+状态同事务）、进程重启后状态恢复。不包括状态机逻辑、Transport 层、合并队列或 API 端点。
</domain>

<decisions>
## Implementation Decisions

### Project Structure
- **D-01:** Standard Go Layout — cmd/server/main.go + cmd/cli/main.go 双入口，internal/ 下按功能层分包
- **D-02:** internal 包结构：store（SQLite 持久化）、engine（状态机，后续阶段填充）、transport（CLI/API 传输层）、api（HTTP 端点）、connector（连接器接口）
- **D-03:** 双入口设计：cmd/server 提供 HTTP API 服务，cmd/cli 提供命令行操作（直接操作 SQLite）

### Schema Management
- **D-04:** Claude's Discretion — Migration 工具选择（推荐 ent auto-migrate 或 golang-migrate，权衡类型安全 vs 灵活控制）
- **D-05:** SQLite 必须启用 WAL 模式 + busy_timeout 配置，支持多 goroutine 并发访问

### Configuration
- **D-06:** Claude's Discretion — 配置来源（YAML/TOML 文件、环境变量、CLI flags 的组合），推荐至少支持配置文件 + 环境变量覆盖

### Logging
- **D-07:** Claude's Discretion — 日志框架和格式，推荐结构化日志（slog 或 zerolog）

### Claude's Discretion
- Schema migration 工具的具体选择
- 配置文件格式和加载方式
- 日志框架和格式
- SQLite 连接池参数
- 具体的 Go module 名称和路径
- 测试辅助工具（in-memory SQLite vs 临时文件）
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### PRD - Core Model
- `多终端 AI 编排平台 PRD.md` §1-3 — 标识系统（dispatch_ref, source_ref, task_id 格式约束）、Task Card 最小字段集、SQLite 三表 schema 定义、card_json 规则
- `多终端 AI 编排平台 PRD.md` §4 — Wave 一致性规则（入队 upsert、seal 语义、路由门控）
- `多终端 AI 编排平台 PRD.md` §Assumptions — 默认值表（存储后端 SQLite、task_id 最大长度 16、dispatch_ref 最大长度 32）

### Research
- `.planning/research/STACK.md` — Go 技术栈推荐（chi, ent, coder/websocket 等）
- `.planning/research/PITFALLS.md` — SQLite 并发、Windows 路径等已知陷阱
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 无现有代码 — 绿地项目，从零开始

### Established Patterns
- 无 — Phase 1 将建立项目的基础模式

### Integration Points
- Phase 1 是所有后续阶段的基础：
  - Phase 2 (Core Engine) 依赖 store 包的状态机查询
  - Phase 3 (Execution Layer) 依赖 store 包的任务更新
  - Phase 4 (Integration) 依赖 store 包的合并队列查询
  - Phase 5 (Interface) 依赖 api 包的端点定义
</code_context>

<specifics>
## Specific Ideas

- Standard Go Layout 是社区标准，确保新贡献者能快速理解项目结构
- 双入口设计让平台既可以通过 CLI 快速操作，也可以作为服务长期运行
- internal/ 按功能层分包保持了清晰的依赖方向：api → engine → store
</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope
</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-04-06*
