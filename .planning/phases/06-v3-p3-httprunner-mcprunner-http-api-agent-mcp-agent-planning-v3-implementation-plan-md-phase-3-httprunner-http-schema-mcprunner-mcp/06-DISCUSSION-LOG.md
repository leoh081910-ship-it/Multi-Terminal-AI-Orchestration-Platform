# Phase 6: v3-P3 HTTPRunner + MCPRunner - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-13
**Phase:** 6-v3-P3 HTTPRunner + MCPRunner
**Areas discussed:** Registration and validation, HTTP template contract, MCP execution semantics, Verification and acceptance

---

## Gray Area Selection

| Option | Description | Selected |
|--------|-------------|----------|
| 全部讨论 | 推荐：这四项都会直接影响研究/计划阶段，尤其当前代码与文档已有几处契约不一致。 | ✓ |
| 注册校验 | API 创建 HTTP/MCP Agent 时，是严格校验并失败，还是先保存 DB、Runner 构建失败只记录日志。现有 org API 会保存后尝试注册。 | |
| 模板契约 | HTTP body_template 要支持哪些变量与函数：当前代码暴露 RunnerTask 字段，但文档提到 .Prompt/.TaskID 和 toJSON。 | |
| MCP 语义 | MCPRunner 是固定调用 execute_task 工具，还是支持配置 tool_name，或按当前实现直接 tools/call 透传任务参数。 | |

**User's choice:** 全部讨论
**Notes:** All identified gray areas were discussed.

---

## Registration and Validation

### Invalid runner_config behavior

| Option | Description | Selected |
|--------|-------------|----------|
| 严格失败 | 推荐：创建/更新直接返回 400，不保存无法执行的 Agent；适合本地编排平台，减少后续隐性失败。 | ✓ |
| 保存但离线 | 保存 Agent 记录，但标记 offline/error 并返回警告；适合先登记、后补配置的工作流。 | |
| 沿用现状 | 保存成功，Runner 构建失败只写日志；实现最少，但用户可能以为 Agent 可用。 | |

**User's choice:** 严格失败
**Notes:** API create/update should fail rather than saving an unusable agent.

### runner_config API shape

| Option | Description | Selected |
|--------|-------------|----------|
| JSON 字符串 | 推荐：沿用当前后端 AgentView 与前端表单，runner_config 是已序列化 JSON 字符串。 | ✓ |
| JSON 对象 | API 更自然，但需要调整 org service、前端类型、文档和兼容保存逻辑。 | |
| 两者兼容 | 接收字符串或对象，内部规范化为字符串；更友好但增加边界处理。 | |

**User's choice:** JSON 字符串
**Notes:** Keep the current serialized string contract.

### Registration health/capability checks

| Option | Description | Selected |
|--------|-------------|----------|
| 只构建校验 | 推荐：只验证 config 能构建 Runner；健康检查/能力探测由 capabilities endpoint 或 heartbeat 执行，避免创建接口被外部 API 卡住。 | ✓ |
| 同步健康检查 | 创建时立即调用 endpoint；反馈最准确，但会让注册依赖外部服务稳定性和密钥可用性。 | |
| 后台探测 | 创建成功后异步探测并更新状态；体验好，但需要额外状态流转和测试。 | |

**User's choice:** 只构建校验
**Notes:** Registration should not depend on external service reachability.

---

## HTTP Template Contract

### Template variable set

| Option | Description | Selected |
|--------|-------------|----------|
| RunnerTask 字段 | 推荐：以当前代码为准，使用 .ID、.Type、.Command、.FilesToModify、.Workspace、.Context.*。文档需统一。 | ✓ |
| 简化别名 | 增加 .Prompt、.TaskID 等别名，降低接入门槛，但需要新增模板数据结构和测试。 | |
| 两套都支持 | 兼容当前代码和旧文档示例；最友好，但更容易产生重复契约。 | |

**User's choice:** RunnerTask 字段
**Notes:** Documentation should align to actual RunnerTask fields rather than aliases.

### Template helper functions

| Option | Description | Selected |
|--------|-------------|----------|
| 支持 toJSON | 推荐：文档和示例已使用，能安全嵌入 JSON 字符串/对象；需要补 FuncMap 和测试。 | ✓ |
| 暂不支持 | 只支持原生 text/template；实现少，但现有文档示例必须改掉。 | |
| 仅文档禁用 | 代码不加函数，只在文档要求用户手写转义 JSON；接入体验较差。 | |

**User's choice:** 支持 toJSON
**Notes:** Add code support instead of removing the documented pattern.

### Default request body

| Option | Description | Selected |
|--------|-------------|----------|
| 任务上下文+元数据 | 推荐：保留当前 task_id/task_type + Context 展平，足够通用且简单。 | ✓ |
| 完整 RunnerTask | 发送 ID、Type、Workspace、Command、FilesToModify、Context 等完整结构；更强但可能暴露更多本地路径。 | |
| 必须模板 | 没有 body_template 就配置失败；最明确，但 Ollama/简单 webhook 接入更麻烦。 | |

**User's choice:** 任务上下文+元数据
**Notes:** Keep current default payload semantics.

---

## MCP Execution Semantics

### Tool invocation

| Option | Description | Selected |
|--------|-------------|----------|
| 配置 tool_name | 推荐：runner_config 指定 tool_name，默认 execute_task；符合文档且适配不同 MCP server。 | ✓ |
| 固定 tools/call | 沿用当前实现，只调用 tools/call 并透传任务参数；实现简单但不符合 MCP tools/call 的常见 name/arguments 结构。 | |
| 自动选工具 | 先 tools/list，再按任务类型选择工具；更智能，但属于路由增强，容易越过本阶段边界。 | |

**User's choice:** 配置 tool_name
**Notes:** Add `tool_name` to MCP runner config with default `execute_task`.

### tools/call params shape

| Option | Description | Selected |
|--------|-------------|----------|
| 标准 name/arguments | 推荐：按 MCP 常见格式发送 {name: tool_name, arguments:{task_id, task_type, context...}}。 | ✓ |
| 直接任务参数 | 沿用当前 params 直接包含 task_id/task_type/context；简单，但对标准 MCP server 兼容性较差。 | |
| 两种可配 | 通过 config 切换标准/legacy 参数格式；兼容强，但增加文档和测试复杂度。 | |

**User's choice:** 标准 name/arguments
**Notes:** Current direct-param behavior should be changed.

### MCP transport scope

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP only | 推荐：本阶段实现 Streamable HTTP/JSON-RPC；sse 字段可保留但明确暂不支持或验证失败。 | ✓ |
| HTTP+SSE | 同时支持 SSE；更完整，但会显著增加协议状态、测试和连接生命周期处理。 | |
| 配置占位 | 继续允许 transport=sse 但按 HTTP 调用；实现少，但会产生误导。 | |

**User's choice:** HTTP only
**Notes:** SSE support is out of scope and should not be silently accepted as implemented.

---

## Verification and Acceptance

### External API dependency

| Option | Description | Selected |
|--------|-------------|----------|
| 本地假服务 | 推荐：用 httptest/本地 mock HTTP 和 MCP JSON-RPC server 做可重复 e2e；真实 API 放手动示例。 | ✓ |
| 真实 API | 严格按旧计划调用 OpenAI/Anthropic；但需要密钥、网络和费用，CI/本地可重复性差。 | |
| 两层验收 | 自动测试用假服务，同时提供可选真实 API smoke test；质量最高但工作量更大。 | |

**User's choice:** 本地假服务
**Notes:** Automated acceptance should avoid external network/API keys/cost.

### Frontend registration form scope

| Option | Description | Selected |
|--------|-------------|----------|
| 基础可用 | 推荐：保留现有 AgentWorkbench 表单，修正字段/错误展示即可，不做复杂 UX。 | ✓ |
| 后端优先 | 只保证 API 和后端 Runner；前端表单留到后续 UI/文档阶段。 | |
| 完整体验 | 做较完整的配置编辑器、schema 提示和能力预览；体验好但明显扩大范围。 | |

**User's choice:** 基础可用
**Notes:** Form is in scope only at basic usability level.

### Example documentation standard

| Option | Description | Selected |
|--------|-------------|----------|
| 可复制运行 | 推荐：OpenAI/Anthropic/Ollama/MCP 示例与实际 API 契约一致，命令可复制，注明环境变量。 | ✓ |
| 参考配置即可 | 只保留 JSON 样例，作为后续文档阶段素材；实现快但容易继续漂移。 | |
| 暂不动文档 | 只改代码和测试，文档留给 v3 文档阶段；但当前文档已与代码不一致。 | |

**User's choice:** 可复制运行
**Notes:** Documentation must be reconciled with the actual implementation contract.

---

## Claude's Discretion

- Planner may decide whether to keep HTTP/MCP config code co-located or split it into roadmap-named files such as `http_config.go` and `mcp_capability.go`.
- Planner may choose the exact fake-service e2e harness shape as long as validation is local, repeatable, and covers registration plus execution semantics.

## Deferred Ideas

- Full schema-driven frontend config editor.
- MCP SSE transport implementation.
- Automatic MCP tool selection based on `tools/list` and task type.
- Required real external API smoke tests.
