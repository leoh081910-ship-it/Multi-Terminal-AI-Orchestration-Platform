# PR: v3 — 通用多 AI Agent 编排平台

**分支**: `feature/v3-multi-agent`  
**基分支**: `main`  
**类型**: 功能发布  
**状态**: 待合并  

---

## 摘要

将平台从硬编码三个 AI 平台（Claude/Codex/Gemini）升级为支持任意 AI Agent 的通用编排平台。引入 Runner 接口、Agent Registry 注册中心，实现动态注册、能力自述和智能路由。

---

## 变更范围

### 新增文件

| 路径 | 说明 |
|------|------|
| `internal/runner/interface.go` | Runner 核心接口 |
| `internal/runner/cli_runner.go` | CLI Runner 实现 |
| `internal/runner/http_runner.go` | HTTP API Runner |
| `internal/runner/mcp_runner.go` | MCP 协议 Runner |
| `internal/runner/capability.go` | 能力自述模型 |
| `internal/runner/telemetry.go` | 遥测与追踪 |
| `internal/registry/registry.go` | Agent 注册中心 |
| `internal/registry/heartbeat.go` | 心跳机制 |
| `internal/server/agent_api.go` | Agent 管理 API |
| `web/src/pages/Agents.tsx` | Agent 管理页面 |
| `web/src/components/AgentCapabilities.tsx` | 能力展示组件 |
| `ent/schema/agent_call.go` | 调用历史存储 |
| `docs/v3-architecture.md` | 架构文档 |
| `docs/agent-onboarding.md` | 接入指南 |
| `docs/migration-v2-to-v3.md` | 迁移指南 |

### 修改文件

| 路径 | 变更 |
|------|------|
| `ent/schema/agent.go` | 增加 `runner_type`, `runner_config` 字段 |
| `internal/transport/cli.go` | 导出核心方法供 CLIRunner 使用 |
| `internal/server/compat_dispatch.go` | 执行路径改为 Runner 接口调用 |
| `internal/router/capability_matcher.go` | 增强为基于 Manifest 评分 |
| `cmd/server/main.go` | 初始化 Registry、加载 DB Agent |
| `go.mod` | 增加 `mcp-go` 依赖 |
| `config.yaml` | 新增 routing.capability 权重配置 |

### 删除/废弃文件

| 路径 | 替代 |
|------|------|
| `config.yaml` 中硬编码 `claude/gemini/codex` 块 | 迁移到数据库 Agent 表（自动迁移） |

---

## 为什么需要这个变更

### 当前问题
- 硬编码三个 AI 平台，无法扩展
- 无 Agent 自述能力，路由仅靠文本标签
- 不支持 HTTP API 或 MCP 协议接入
- 配置分散在 config.yaml 和数据库

### v3 解决
- 任意 Agent 可通过 CLI/HTTP/MCP 接入
- 动态注册/注销，无需重启
- 能力自述 + 智能路由
- 统一配置（全部迁移到数据库）

---

## 测试计划

### 单元测试
- [ ] Runner 接口测试
- [ ] CLIRunner 单元测试
- [ ] HTTPRunner 单元测试（mock HTTP server）
- [ ] MCPRunner 单元测试（mock MCP server）
- [ ] Registry 并发测试
- [ ] CapabilityMatcher 评分逻辑

### 集成测试
- [ ] 现有任务（Claude/Codex/Gemini）执行正常
- [ ] 动态注册 Agent 并执行任务
- [ ] HTTPRunner 调用 OpenAI API 完成任务
- [ ] MCPRunner 连接 MCP server 调用 tool
- [ ] 能力自述驱动路由决策
- [ ] 心跳超时后 Agent 标记 offline

### E2E 测试
- [ ] 完整任务生命周期（从 Task Card 到合并）
- [ ] 多个 Agent 竞争同一任务
- [ ] 注册 → 执行 → 注销 → 重新注册

### 性能测试
- [ ] Agent 调用 overhead < 50ms (P95)
- [ ] 注册 100 个 Agent 后路由延迟 < 200ms
- [ ] 并发 50 个任务同时调用不同 Agent

---

## 迁移指南

### 自动迁移（推荐）
```bash
./bin/migrate-v3 --config config.yaml
```
效果：
- 读取 config.yaml 中的 projects
- 为每个项目的 claude/gemini/codex 创建 Agent 记录
- 保留原文件，仅标记 deprecated

### 手动迁移
见 `docs/migration-v2-to-v3.md`

### 回滚
```bash
git revert <merge-commit>
docker-compose down && docker-compose up -d
```

---

## 破坏性变更

| 变更 | 影响 | 缓解 |
|------|------|------|
| `config.yaml` 不再支持 `projects.*.claude` 等块 | 旧配置会打印警告但继续工作 | 自动迁移脚本 |
| Agent 管理 API 新增 | 无影响 |
| 执行路径改为 Runner 接口 | 内部重构，外部无感知 |

**无** API 契约破坏性变更。所有现有 API 端点保持不变。

---

## 依赖

- Go 1.25+
- 新增依赖: `github.com/mcp-go/mcp` (MCP 支持)
- 无新增外部服务依赖（MCP 可选）

---

## 回滚计划

1. `git revert` 本 PR 的 merge commit
2. 重新部署 v2.1 版本（二进制或容器）
3. 数据库无需回滚，v2.1 会忽略新字段

验证：回滚后执行 10 个任务确保正常。

---

## 检查清单

### 代码质量
- [ ] 所有新增代码有单元测试，覆盖率 > 80%
- [ ] 无 lint 警告
- [ ] 无硬编码凭证
- [ ] 日志结构化（trace_id、agent_id、task_type）

### 文档
- [ ] v3 架构说明
- [ ] Agent 接入指南（CLI/HTTP/MCP）
- [ ] API 文档（Agent 管理端点）
- [ ] 迁移指南（v2 → v3）
- [ ] 示例配置（OpenAI、Anthropic、Ollama、MCP demo）

### 运维
- [ ] Prometheus 指标正常工作
- [ ] Web UI Agent 管理页可用
- [ ] 心跳超时后标记 offline
- [ ] 能力缓存 TTL 可配置

---

## 相关链接

- [v3 路线图](.planning/v3-multi-agent-roadmap.md)
- [详细实施计划](.planning/v3-implementation-plan.md)
- [设计文档](docs/v3-architecture.md)

---

**审核人**: @leoh081910-ship-it  
**预计合并时间**: 各 Phase 完成后逐步合并，最终 Phase 6 后合并此分支

🤖 生成工具: Claude Code / v3 规划助手
