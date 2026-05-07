# v3 实施计划 — 通用多 AI Agent 编排平台

**版本**: v3.0  
**规划日期**: 2026-05-06  
**状态**: 待审批  

---

## 概述

本计划将平台从硬编码三个 AI 平台升级为支持任意 Agent 的通用编排平台。核心是引入 `Runner` 接口、`AgentRegistry` 注册中心和三种具体 Runner 实现（CLI、HTTP、MCP）。

**总工期**: 10 周  
**团队**: 1-2 名开发者  

---

## Phase 1: Runner 接口与 CLIRunner (Week 1-2)

**目标**: 建立 Runner 抽象层，将现有 CLI 执行逻辑迁移到 `CLIRunner`，实现零破坏性变更。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 1.1 | 定义 Runner 接口和 CapabilityManifest | `internal/runner/interface.go` | 1d | 接口编译通过 |
| 1.2 | 实现 Executor 适配桥接 | `internal/runner/executor_bridge.go` | 1d | 旧执行逻辑可调用 |
| 1.3 | 实现 CLIRunner | `internal/runner/cli_runner.go` | 2d | 复用 `transport.CLITransport` |
| 1.4 | 修改 server/dispatch 使用 Runner | `internal/server/compat_dispatch.go` | 2d | 任务执行走 Runner 接口 |
| 1.5 | Agent Schema 扩展 | `ent/schema/agent.go` | 0.5d | 增加 `runner_type`, `runner_config` |
| 1.6 | 迁移工具: config → agent 表 | `cmd/migrate/main.go` | 1d | 自动迁移 claude/gemini/codex |
| 1.7 | 单元测试 + 集成测试 | `internal/runner/*_test.go` | 1.5d | 覆盖率 > 80% |

**关键代码结构**:

```go
// internal/runner/interface.go
type Runner interface {
    Execute(ctx context.Context, task RunnerTask) (*RunnerResult, error)
    HealthCheck(ctx context.Context) error
    Cancel(ctx context.Context, taskID string) error
    GetCapabilities(ctx context.Context) (*CapabilityManifest, error)
}
```

**验收**: 现有所有任务类型（包括逆向专项）运行结果与 v2 完全一致。

---

## Phase 2: Agent Registry 与动态注册 (Week 3-4)

**目标**: 实现运行时 Agent 注册中心，支持动态添加/移除 Agent。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 2.1 | 实现 Registry 核心 | `internal/registry/registry.go` | 2d | 注册/注销/查询功能 |
| 2.2 | 启动时加载 DB Agent | `cmd/server/main.go` | 1d | 从 agent 表初始化 Runner |
| 2.3 | 心跳机制 | `internal/registry/heartbeat.go` | 1d | 定期 HealthCheck 并更新状态 |
| 2.4 | HTTP API 端点 | `internal/server/agent_api.go` | 2d | POST/PUT/DELETE /api/v1/agents |
| 2.5 | Web UI 展示 | `web/src/pages/Agents.tsx` | 2d | 列表、状态、能力查看 |
| 2.6 | 集成测试 | `internal/registry/registry_test.go` | 0.5d | 端到端流程 |

**API 设计**:

```
POST   /api/v1/agents          # 注册新 Agent
GET    /api/v1/agents          # 列表（支持按状态筛选）
GET    /api/v1/agents/:id      # 详情（含能力）
PUT    /api/v1/agents/:id      # 更新配置
DELETE /api/v1/agents/:id      # 注销
POST   /api/v1/agents/:id/ping # 手动心跳
```

**验收**: 可通过 API 动态添加一个新的 CLI Agent，立即可用于任务路由。

---

## Phase 3: HTTPRunner 与 MCPRunner (Week 5-6)

**目标**: 支持 HTTP API 型 Agent 和 MCP 协议 Agent。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 3.1 | HTTPRunner 核心 | `internal/runner/http_runner.go` | 2d | 支持模板化请求/响应 |
| 3.2 | HTTP 配置 Schema | `internal/runner/http_config.go` | 1d | JSON schema 验证 |
| 3.3 | MCPRunner 基础 (mcp-go) | `internal/runner/mcp_runner.go` | 2d | 连接 MCP server |
| 3.4 | MCP 能力发现 | `internal/runner/mcp_capability.go` | 1d | 获取 tools/prompts |
| 3.5 | 示例配置 | `docs/examples/` | 1d | OpenAI、Anthropic、Ollama |
| 3.6 | 端到端测试 | `tests/e2e/runner_http_test.go` | 1d | 调用真实 API |

**HTTPRunner 配置示例**:

```json
{
  "endpoint": "https://api.openai.com/v1/chat/completions",
  "method": "POST",
  "headers": {"Authorization": "Bearer ${OPENAI_API_KEY}"},
  "body_template": "{\"model\":\"gpt-4o\",\"messages\":[{\"role\":\"user\",\"content\":{{.Prompt | toJSON}}}]}",
  "output_path": "choices[0].message.content"
}
```

**验收**: 可以调用 OpenAI API 执行任务并正确合并结果。

---

## Phase 4: 能力自述与智能路由增强 (Week 7-8)

**目标**: Agent 主动上报能力，路由策略基于能力 Manifest 决策。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 4.1 | 实现 GetCapabilities | 各 Runner | 1d | Runner 返回 Manifest |
| 4.2 | Registry 刷新缓存 | `internal/registry/capability.go` | 2d | 定期调用 GetCapabilities |
| 4.3 | 增强 CapabilityMatcher | `internal/router/capability_matcher.go` | 2d | 基于 Manifest 多维度评分 |
| 4.4 | 路由策略配置扩展 | `config.yaml` | 0.5d | 增加能力匹配权重 |
| 4.5 | CLIRunner 能力检测 | `internal/runner/cli_runner.go` | 1d | 解析 `--version` 等 |
| 4.6 | Web UI 能力展示 | `web/src/components/AgentCapabilities.tsx` | 1.5d | 可视化 Manifest |

**Manifest 结构**:

```go
type CapabilityManifest struct {
    Name            string
    TaskTypes       []string          // ["feature", "bugfix", "reverse"]
    MaxInputSize    int64
    OutputGlobs     []string
    ModelFamily     string             // "claude", "gpt", "local"
    ContextWindow   int
    SupportsThinking bool
}
```

**验收**: 不同类型任务自动路由到能力最匹配的 Agent；手动指定 Agent 时覆盖自动选择。

---

## Phase 5: 可观测性与运维 (Week 9)

**目标**: 追踪每次 Agent 调用，提供监控指标。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 5.1 | 结构化日志增强 | `internal/runner/telemetry.go` | 1d | 每个调用带 trace_id |
| 5.2 | Prometheus 指标 | `internal/runner/metrics.go` | 1.5d | counter, histogram, gauge |
| 5.3 | 调用历史存储 | `ent/schema/agent_call.go` | 1.5d | 记录请求/响应摘要 |
| 5.4 | Web UI 调用历史 | `web/src/pages/TaskDetail.tsx` | 1d | 展示 Agent 调用 tab |
| 5.5 | 错误分类与重试 | `internal/runner/retry.go` | 1d | 区分临时/永久失败 |

**指标**:

- `agent_requests_total{agent_id, task_type, status}`
- `agent_duration_seconds{agent_id, task_type}`
- `agent_heartbeat_timestamp{agent_id}`
- `registry_agent_count{status}`

**验收**: Grafana 可查询到 Agent 调用成功率、P95 延迟。

---

## Phase 6: 文档与示例 (Week 10)

**目标**: 完善用户文档、开发者指南、示例配置。

### 任务清单

| ID | 任务 | 文件 | 预计 | 验收 |
|----|------|------|------|------|
| 6.1 | v3 架构说明 | `docs/v3-architecture.md` | 1d | 覆盖架构图、组件说明 |
| 6.2 | Agent 接入指南 | `docs/agent-onboarding.md` | 1.5d | CLI/HTTP/MCP 三种方式 |
| 6.3 | 示例 Agent 配置 | `docs/examples/` | 1d | OpenAI、Anthropic、Ollama、MCP demo |
| 6.4 | API 文档更新 | `docs/api.md` | 0.5d | 新增 Agent 管理端点 |
| 6.5 | 迁移指南 | `docs/migration-v2-to-v3.md` | 1d | 自动迁移 + 手动步骤 |

**交付物**:

- 完整的 Markdown 文档集
- 至少 3 个可运行的示例配置
- 迁移脚本用户手册

---

## 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| Runner 接口破坏现有执行路径 | 中 | 高 | Phase 1 保持 transport 不变，只做适配层包装 |
| HTTPRunner 模板引擎复杂度高 | 中 | 中 | 先支持 Go template，后续迭代支持 JSONPath |
| MCP 协议版本兼容性 | 低 | 中 | 锁定 mcp-go 稳定版本，文档注明版本要求 |
| 动态注册并发安全问题 | 低 | 中 | Registry 使用 RWMutex，单元测试覆盖 |
| 能力自述导致路由延迟增加 | 中 | 低 | 缓存 Manifest，TTL 可配置 |

---

## 回滚计划

每个 Phase 完成后打 tag（`v3-phase1`, `v3-phase2`…）。若 Phase 发现问题：

1. 停止后续 Phase 开发
2. 回退到上一个稳定 tag
3. 记录问题并修复
4. 重新进入该 Phase

全流程回滚: `git revert` 整个 v3 开发分支，重新部署 v2.1 版本。数据库迁移工具支持向下兼容（保留原 config 读取路径）。

---

## 里程碑

| 里程碑 | 日期 | 交付物 |
|--------|------|--------|
| M1: Runner 抽象完成 | Week 2 结束 | CLIRunner 部署，v2 功能全通过 |
| M2: 动态注册可用 | Week 4 结束 | Agent API + Web UI 管理页 |
| M3: HTTP/MCP 支持 | Week 6 结束 | 可调用 OpenAI API |
| M4: 智能路由增强 | Week 8 结束 | 能力自述 + 自动路由 |
| M5: 可观测性上线 | Week 9 结束 | Prometheus + 调用历史 |
| M6: 文档完成 | Week 10 结束 | 完整文档集 + 示例 |

---

**计划版本**: 1.0  
**审批状态**: 待审批  
**下次更新**: Phase 1 启动前
