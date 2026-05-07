[
# v3 — 通用多 AI Agent 编排平台

**版本目标**: v3.0  
**规划日期**: 2026-05-06  
**状态**: 草案

---

## 1. 背景与动机

### 1.1 v2 架构现状

当前平台已具备完整的任务编排能力：状态机、Wave 管理、依赖拓扑、Transport (CLI/API)、合并队列、逆向专项等。在 Agent 管理层面已埋入通用抽象：

- ✅ `ent/schema/agent.go` — Agent 实体定义
- ✅ `internal/org/service.go` — Agent 注册/查询/心跳/配置存储
- ✅ `internal/router/...` — 能力匹配、负载均衡、亲和度、加权路由策略
- ✅ `internal/connector/interface.go` — Connector 抽象 (GSD Connector 实现)
- ⚠️ `internal/transport` — 仅支持 CLI 和 API 两种执行方式，且三个 Agent (Claude/Codex/Gemini) 仍硬编码在 `config.yaml`

### 1.2 当前局限 (需要升级的原因)

| 问题 | 说明 |
|------|------|
| **平台锁定** | 硬编码三个 AI 平台 (Claude/Codex/Gemini)，无法动态添加新 Agent |
| **无 Runner 抽象** | AGNT-01 要求的 `Runner` 接口缺失，执行逻辑分散在 transport 中 |
| **Agent 类型单一** | 只支持 CLI (通过 git worktree)；不支持 HTTP API、MCP、WebSocket 等协议 |
| **能力描述静态** | `specialties` 只是文本标签，Agent 无法自述真实能力（模型、工具、输入输出约束） |
| **配置分离** | Agent 存储在数据库，但启动命令和凭证仍在 `config.yaml` 中 |
| **无动态发现** | 没有 Agent 注册中心或服务发现机制 |

### 1.3 v3 愿景

构建 **通用多 AI Agent 编排平台**：

- 任何 AI Agent（商业 API、开源模型、本地 CLI、MCP 服务器）皆可接入
- 通过统一 `Runner` 接口定义执行契约
- 支持多种通信协议（CLI、HTTP、MCP、gRPC、WebSocket）
- Agent 自述能力模型，平台动态路由匹配
- 与现有 Task Card、Wave、状态机、合并队列完全兼容 (向后兼容 v2)

---

## 2. 目标架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Orchestration Core (现有)                    │
│  Task Card │ Wave │ State Machine │ Merge Queue │ Dependency Graph  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Router (现有, 增强)                          │
│  CapabilityMatcher │ LoadBalancer │ AffinityScorer │ Weighted        │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Agent Registry (新增)                        │
│  - RegisterAgent() / DeregisterAgent()                              │
│  - Heartbeat monitoring                                             │
│  - Agent info caching                                               │
│  - Dynamic discovery (optional)                                     │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Runner Interface (新增)                      │
│  - Execute(ctx, TaskCard, workspace) → ExecutionResult              │
│  - HealthCheck() error                                              │
│  - Cancel(ctx) error                                                │
│  - GetCapabilities() Capabilities                                   │
└───────────────┬─────────────────┬─────────────────┬─────────────────┘
                │                 │                 │
                ▼                 ▼                 ▼
        ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
        │  CLIRunner   │ │  HTTPRunner  │ │  MCPRunner   │
        │ (现有 transport│ │ (新增)       │ │ (新增)       │
        │ 适配)        │ │              │ │              │
        └──────────────┘ └──────────────┘ └──────────────┘
                │                 │                 │
                ▼                 ▼                 ▼
        Claude/Codex/Gemini   OpenAI API /      任何 MCP
        (现有脚本)            Anthropic API     服务器
                            / 自定义 HTTP
```

---

## 3. 核心组件设计

### 3.1 Runner 接口 (补全 AGNT-01)

```go
// internal/runner/interface.go
package runner

type Runner interface {
    // Execute runs a task based on TaskCard and returns execution result.
    Execute(ctx context.Context, task *TaskCard, workspace Workspace) (*ExecutionResult, error)
    
    // HealthCheck returns nil if runner is healthy.
    HealthCheck(ctx context.Context) error
    
    // Cancel attempts to cancel a running task.
    Cancel(ctx context.Context, taskID string) error
    
    // GetCapabilities returns the runner's capability manifest.
    GetCapabilities(ctx context.Context) (*CapabilityManifest, error)
}

type CapabilityManifest struct {
    // Basic info
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Type        string   `json:"type"` // "cli", "http", "mcp", "grpc"
    
    // Task type support
    TaskTypes   []string `json:"task_types"`   // ["feature", "bugfix", "reverse_static_c_rebuild", ...]
    
    // Input constraints
    SupportsStdin      bool     `json:"supports_stdin"`
    MaxInputSizeBytes  int64    `json:"max_input_size_bytes,omitempty"`
    RequiredFiles      []string `json:"required_files,omitempty"`      // e.g., ["task.json", "context.yaml"]
    OutputGlobPatterns []string `json:"output_glob_patterns,omitempty"`
    
    // Runtime attributes
    MaxConcurrency    int           `json:"max_concurrency"`
    AvgExecutionTime  time.Duration `json:"avg_execution_time,omitempty"`
    Endpoint          string        `json:"endpoint,omitempty"`        // for HTTP/gRPC runners
    
    // Model information
    ModelFamily       string   `json:"model_family,omitempty"`         // "claude", "gpt", "gemini", "local"
    ModelName         string   `json:"model_name,omitempty"`
    SupportsThinking  bool     `json:"supports_thinking,omitempty"`
    ContextWindow     int      `json:"context_window,omitempty"`
}
```

### 3.2 Agent Registry (新增)

```go
// internal/registry/registry.go
type Registry struct {
    mu      sync.RWMutex
    agents  map[string]*AgentRecord   // agentID -> record
    runners map[string]runner.Runner  // agentID -> runner instance
    client  *ent.Client
}

type AgentRecord struct {
    Agent          *org.AgentView
    Capabilities   *runner.CapabilityManifest
    LastSeen       time.Time
    Status         string   // "online", "offline", "error"
}

func (r *Registry) Register(agentID string, runner runner.Runner) error
func (r *Registry) Deregister(agentID string) error
func (r *Registry) GetRunner(agentID string) (runner.Runner, error)
func (r *Registry) ListAvailable(taskType string) []*AgentRecord
func (r *Registry) RefreshCapabilities(ctx context.Context) error
```

### 3.3 具体 Runner 实现

#### 3.3.1 CLIRunner (现有 Transport 适配)

将现有的 `transport.CLITransport` 适配为 `runner.Runner` 接口。

**配置存储**：原 `config.yaml` 中的 claude/gemini/codex 命令迁移到 `agent.config` 字段。

```yaml
# agent.config 示例
command: '& ''./scripts/runtime/claude.ps1'''
shell: powershell
worktree_base: ./worktrees
timeout_seconds: 1800
```

**优势**：无需改动现有脚本，直接复用。

#### 3.3.2 HTTPRunner (新增)

支持调用任何 HTTP API 形式的 AI Agent。

```yaml
# agent.config 示例
endpoint: https://api.openai.com/v1/chat/completions
api_key: ${OPENAI_API_KEY}
model: gpt-4o
method: POST
headers:
  Content-Type: application/json
body_template: |
  {
    "model": "{{.Model}}",
    "messages": [{"role": "user", "content": {{.Prompt | toJSON}}}]
  }
output_extract_path: choices[0].message.content
```

#### 3.3.3 MCPRunner (新增)

支持 Model Context Protocol (MCP) 服务器。

```yaml
# agent.config 示例
type: mcp
endpoint: http://localhost:3000/mcp
capabilities:
  - tools
  - prompts
  - resources
transport: sse   # or websocket
```

### 3.4 能力匹配增强

现有 `CapabilityMatcher` 只做字符串匹配。v3 增强为基于 `CapabilityManifest` 的匹配：

- 任务类型匹配 (`taskTypes`)
- 输入大小约束
- 文件依赖检查
- 模型/上下文窗口匹配
- 支持同一 Agent 在不同任务上的能力差异

---

## 4. 实施阶段

### Phase 1: 接口抽象与 CLI Runner 适配 (Week 1-2)

**目标**：建立 `runner.Runner` 接口，将现有 CLI 执行逻辑迁移为 `CLIRunner`，实现向后兼容。

| 任务 | 描述 | 产出 |
|------|------|------|
| 1.1 | 定义 `runner` 包和 `Runner` 接口 | `internal/runner/interface.go` |
| 1.2 | 定义 `CapabilityManifest` 结构 | `internal/runner/capability.go` |
| 1.3 | 实现 `CLIRunner` (封装 `transport.CLITransport`) | `internal/runner/cli_runner.go` |
| 1.4 | 修改 `executor` 包使用 `Runner` 接口 | 解耦执行逻辑 |
| 1.5 | 数据库迁移：为 `agent` 表增加 `runner_config` JSON 字段 | ent schema 更新 |
| 1.6 | 将 `config.yaml` 中的 claude/gemini/codex 迁移到数据库 | bootstrap/迁移脚本 |
| 1.7 | 单元测试 + 集成测试 | 覆盖率 > 80% |

**验收**：现有任务（Claude/Codex/Gemini）正常运行，无行为变化。

---

### Phase 2: Agent Registry 与动态注册 (Week 3-4)

**目标**：实现 Agent 注册中心，支持运行时动态注册/注销。

| 任务 | 描述 | 产出 |
|------|------|------|
| 2.1 | 实现 `registry.Registry` | `internal/registry/registry.go` |
| 2.2 | 启动时从数据库加载 Agent 并初始化 Runner | `cmd/server` 集成 |
| 2.3 | 实现心跳机制 (goroutine 定期调用 `HealthCheck`) | 更新 `last_heartbeat_at` |
| 2.4 | HTTP API: 动态注册/注销/列表 Agent | `POST /api/v1/agents/register` 等 |
| 2.5 | Web UI: Agent 管理页面 | 展示 Agent 列表、状态、能力 |
| 2.6 | 集成测试 | 注册、心跳、注销流程 |

**验收**：可通过 API 动态添加一个新的 Agent（例如一个新的 CLI 脚本），平台自动识别并可用于任务路由。

---

### Phase 3: HTTPRunner 与 MCPRunner (Week 5-6)

**目标**：支持 HTTP API 类型的 Agent 和 MCP 协议 Agent。

| 任务 | 描述 | 产出 |
|------|------|------|
| 3.1 | 实现 `HTTPRunner` | `internal/runner/http_runner.go` |
| 3.2 | 定义 HTTP 配置 schema (模板化请求/响应提取) | 支持 Go template 或 JSONPath |
| 3.3 | 实现 `MCPRunner` (基于 mcp-go SDK) | `internal/runner/mcp_runner.go` |
| 3.4 | 集成 MCP 工具/提示词/资源能力发现 | 填充 `CapabilityManifest` |
| 3.5 | 添加示例 Agent 配置 (OpenAI, Anthropic, etc.) | `docs/examples/` |
| 3.6 | 端到端测试 | 调用真实 HTTP/MCP Agent |

**验收**：可成功调用 OpenAI API 或 Anthropic API 完成任务，结果正确合并。

---

### Phase 4: 能力自述与智能路由增强 (Week 7-8)

**目标**：Agent 自述能力，路由策略根据能力动态选择。

| 任务 | 描述 | 产出 |
|------|------|------|
| 4.1 | 修改 `Runner` 接口要求实现 `GetCapabilities` | 已有的 Runner 实现该方法 |
| 4.2 | Registry 定期刷新能力缓存 (TTL) | 处理能力变更 |
| 4.3 | 增强 `CapabilityMatcher` 使用 Manifest 信息 | 支持任务类型、上下文窗口等维度的打分 |
| 4.4 | 路由策略中加入能力匹配权重 | 可配置 |
| 4.5 | 为 `CLIRunner` 实现能力自述 (读取配置或自动检测) | 如检测 Claude Code 版本 |
| 4.6 | 展示 Agent 能力详情在 Web UI | Agent 详情页 |

**验收**：不同任务类型自动选择合适的 Agent；当 Agent 能力变更时，路由决策更新。

---

### Phase 5: 可观测性与运维 (Week 9)

**目标**：Agent 执行的可观测性、追踪和调试。

| 任务 | 描述 | 产出 |
|------|------|------|
| 5.1 | 为每个 Runner 执行添加 trace_id 和 span | 结构化日志 |
| 5.2 | 记录 Agent 调用耗时、成功/失败、输入/输出摘要 | metrics + events |
| 5.3 | Prometheus 指标：agent_requests_total, agent_duration_seconds | 监控 |
| 5.4 | Web UI 展示 Agent 调用历史 | 任务详情页增加 Agent 调用 tab |
| 5.5 | 错误分类与重试策略 (针对不同 Agent 类型) | 可配置 |

**验收**：可追踪每次 Agent 调用的完整链路；失败时可区分 Agent 问题 vs 平台问题。

---

### Phase 6: 文档与示例 (Week 10)

**目标**：完善用户文档、开发者指南、示例配置。

| 任务 | 描述 |
|------|------|
| 6.1 | 编写 v3 架构说明 |
| 6.2 | Agent 接入指南 (CLI/HTTP/MCP) |
| 6.3 | 示例 Agent 配置文件 (OpenAI, Anthropic, local LLM via Ollama, etc.) |
| 6.4 | API 文档更新 (Agent 管理端点) |
| 6.5 | 迁移指南 (从 v2 到 v3) |

---

## 5. 迁移策略 (向后兼容)

### 5.1 数据迁移

1. 创建 `agent` 表 (已存在)。确保字段足够存储 runner 配置 (新增 `runner_type`、`runner_config` 字段)。
2. 编写迁移工具：读取 `config.yaml` 中的 `projects.*.claude/gemini/codex`，为每个项目创建对应的 Agent 记录。
3. 保留旧的 `transport` 代码，但标记为 deprecated。

### 5.2 API 兼容

- 现有 API 端点不变（任务创建、查询等），只是内部执行改为通过 `Runner` 接口。
- 新增 Agent 管理端点不影响现有功能。

### 5.3 CLI 兼容

- `aiop` 命令行为不变。
- 启动参数扩展支持指定 Agent ID。

---

## 6. 成功标准

### 性能指标

- Agent 调用 overhead 增加 < 50ms (P95)
- 支持同时注册 > 100 个 Agent
- 能力刷新周期不影响路由延迟 (< 200ms)

### 功能指标

- ✅ 接入 OpenAI GPT-4o 作为 Agent 并完成任务
- ✅ 接入本地 Ollama 模型作为 Agent
- ✅ 接入 MCP 服务器并调用其工具
- ✅ 动态注册/注销 Agent 无需重启服务
- ✅ 任务根据能力自动路由到最适合的 Agent
- ✅ 所有现有 v2 任务类型（包括逆向专项）仍可正常工作

### 用户体验

- Web UI 可查看所有 Agent 及其实时状态、能力
- 文档覆盖三种 Runner 类型的接入示例

---

## 7. 技术债务与后续

- **Runner SDK**: 提供 Go/Node/Python 的轻量 SDK，便于第三方开发 Agent
- **Agent 市场**: 共享 Agent 配置模板
- **分布式 Agent**: 支持跨网络远程 Agent (通过 gRPC 或 Relay)
- **Agent 版本管理**: 多版本并存和灰度

---

## 8. 参考资料

- 现有代码: `internal/runner/` (待创建), `internal/org/service.go`, `internal/transport/cli.go`
- 需求: `REQUIREMENTS.md` AGNT-01, CONN-01~03
- 升级路线图: `IMPROVEMENT_ROADMAP.md` P3-11 (AI Agent 插件系统)

---

**规划版本**: v0.1 (草案)  
**下次审核**: 2026-05-10  
**负责人**: —  
**状态**: 📝 待审批
