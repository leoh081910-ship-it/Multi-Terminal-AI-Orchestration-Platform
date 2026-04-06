# Requirements: 多终端 AI 编排平台

**Defined:** 2026-04-06
**Core Value:** 任务从提交到合并全流程自动化——用户定义"做什么"，平台负责调度、隔离、冲突回避、工件合并和失败重试

## v1 Requirements

### Core Model（核心模型）

- [ ] **CORE-01**: 平台为每次 discoverTasks() 调用自动生成唯一 dispatch_ref，同次调用返回的所有任务共享该值
- [ ] **CORE-02**: task_id 仅允许 `[a-z0-9_-]`，长度 1-16；dispatch_ref 仅允许 `[a-z0-9_-]`，长度 1-32
- [ ] **CORE-03**: Task Card 包含最小字段集（id, dispatch_ref, source, source_ref, type, objective, context, files_to_read, files_to_modify, acceptance_criteria, relations, wave, priority）
- [ ] **CORE-04**: files_to_modify 作为写入白名单支持 glob 模式匹配
- [ ] **CORE-05**: source_ref 保留来源系统原始标识，与平台内部 dispatch_ref 语义分离
- [ ] **CORE-06**: relations[] 中每条边包含 task_id、type（depends_on/conflicts_with）、reason

### Persistence（持久化）

- [ ] **PERS-01**: tasks 表包含 id, dispatch_ref, state, retry_count, loop_iteration_count, transport, wave, topo_rank, workspace_path, artifact_path, last_error_reason, created_at, updated_at, terminal_at, card_json
- [ ] **PERS-02**: events 表包含 event_id, task_id, event_type, from_state, to_state, timestamp, reason, attempt, transport, runner_id, details
- [ ] **PERS-03**: waves 表包含 dispatch_ref, wave, sealed_at, created_at，(dispatch_ref, wave) 唯一约束
- [ ] **PERS-04**: card_json TEXT NOT NULL 保存完整 Task Card JSON，业务字段默认从 card_json 读取
- [ ] **PERS-05**: 事件写入和任务状态更新在同一 SQLite 事务内完成
- [ ] **PERS-06**: SQLite 使用 WAL 模式 + busy_timeout 配置，支持多 goroutine 并发访问

### Wave Management（Wave 管理）

- [ ] **WAVE-01**: 任务入队时平台自动 upert 对应的 (dispatch_ref, wave) 到 waves 表
- [ ] **WAVE-02**: sealed_at = null 表示 wave 未 seal，该 wave 下任务不得进入 routed
- [ ] **WAVE-03**: Connector 以 (dispatch_ref, wave) 为单位提交任务，或显式调用 sealWave()
- [ ] **WAVE-04**: 已 seal 的 wave 拒绝追加任务，reason = "wave_already_sealed"
- [ ] **WAVE-05**: tasks.wave 必须始终对应一条同 (dispatch_ref, wave) 的 waves 记录

### Dependency & Conflict（依赖与冲突）

- [ ] **DEPD-01**: topo_rank 只基于 depends_on 计算，无依赖任务默认 topo_rank = 0
- [ ] **DEPD-02**: depends_on 只能指向同 wave 或更早 wave，指向更晚 wave 的依赖入队拒绝，reason = "invalid_dependency"
- [ ] **DEPD-03**: conflicts_with 只在同 wave 内计算
- [ ] **DEPD-04**: 若两任务间已存在 depends_on，Router 不再生成 conflicts_with
- [ ] **DEPD-05**: conflicts_with 只影响路由与批次切分，不改变合并排序

### State Machine（状态机）

- [ ] **STAT-01**: 实现 13 态状态机（queued, routed, workspace_prepared, running, patch_ready, verified, merged, done, retry_waiting, verify_failed, apply_failed, failed）
- [ ] **STAT-02**: 状态转换严格遵守 PRD 定义的有向图，非法转换被拒绝
- [ ] **STAT-03**: done 和 failed 是唯一终态，终态任务拒绝所有迟到写入，只记录警告
- [ ] **STAT-04**: merged → done 是即时转换，无额外 finalize
- [ ] **STAT-05**: TTL 从 terminal_at 起算，不依赖 updated_at
- [ ] **STAT-06**: 前置任务 failed/apply_failed 时，后置非终态任务立即进 failed，reason = "dependency_failed"

### Transport（传输）

- [ ] **TRAN-01**: CLI Transport 在 git worktree 中执行任务，按 files_to_modify 白名单 glob 抽取工件到 artifacts/{task_id}/
- [ ] **TRAN-02**: CLI 白名单匹配为空时 running → retry_waiting，reason = "empty_artifact_match"
- [ ] **TRAN-03**: CLI 白名单外新增文件只记录警告，不抽取，不单独判失败
- [ ] **TRAN-04**: API Transport 返回完整文件工件，写入 artifacts/{task_id}/ 并同步到 API 隔离目录
- [ ] **TRAN-05**: API workspace 同步失败时 running → retry_waiting，reason = "workspace_write_failed"
- [ ] **TRAN-06**: 从 patch_ready 开始 CLI 和 API 走同一条后续流程
- [ ] **TRAN-07**: 所有 transport 检查根路径长度、空格、中文字符；CLI 额外检查符号链接权限和 worktree 路径长度

### Merge Queue（合并队列）

- [ ] **MERG-01**: 任务进入 verified 后立即加入全局合并队列
- [ ] **MERG-02**: 合并队列单消费者串行处理
- [ ] **MERG-03**: 只消费依赖已全部 done 的 verified 任务
- [ ] **MERG-04**: 排序为 topo_rank 升序 + created_at 升序
- [ ] **MERG-05**: 合并操作为工件复制到主 checkout + git add + git commit
- [ ] **MERG-06**: apply_failed 只能人工处理，不自动重试

### Retry & Recovery（重试与恢复）

- [ ] **RETR-01**: retry_count 跨所有主动失败累计，默认 max_retries = 2
- [ ] **RETR-02**: 退避为 30 秒、60 秒，从 retry_waiting 事件的 timestamp 起算
- [ ] **RETR-03**: 恢复时复用原始时间戳，不重置计时
- [ ] **RETR-04**: attempt 等于写事件时的当前 retry_count
- [ ] **RETR-05**: 消耗 retry_count 的原因码：execution_failure, workspace_write_failed, empty_artifact_match, deterministic_check_failed, test_command_failed, reverse_loop_exhausted, reverse_env_unavailable
- [ ] **RETR-06**: 不消耗 retry_count：process_resume, dependency_failed
- [ ] **RETR-07**: 恢复时重新触发依赖失败传播检查

### Reverse Engineering（逆向专项）

- [ ] **REVR-01**: reverse_static_c_rebuild 任务类型，context 必须包含 target_so_path, ida_mcp_endpoint, frida_hook_spec, oracle_input_spec, oracle_output_ref, analysis_state_md_path, final_artifact_path
- [ ] **REVR-02**: 缺少逆向必要字段时任务不得进入 routed
- [ ] **REVR-03**: 运行中固定循环：IDA 静态分析 → 生成/修正 .c → 编译 → 运行采集 static_output → Frida 黑盒采集 frida_oracle_output → Diff → match_rate
- [ ] **REVR-04**: 单步失败（compile_failed, static_run_failed, frida_oracle_failed, diff_failed, oracle_mismatch）为内部自重试，不触发外层状态迁移，不消耗 retry_count
- [ ] **REVR-05**: 内部失败通过 events 表记录，event_type = "loop_iteration"，from_state = "running"，to_state = "running"
- [ ] **REVR-06**: 每完成一轮完整循环 tasks.loop_iteration_count 加 1
- [ ] **REVR-07**: loop_iteration_count 超过 max_loop_iterations（默认 50）或不可恢复环境错误时触发 running → retry_waiting
- [ ] **REVR-08**: 外层重试重新执行时 loop_iteration_count 重置为 0
- [ ] **REVR-09**: match_rate = 100% 且最终工件生成后才允许 running → patch_ready
- [ ] **REVR-10**: 进程恢复时 loop_iteration_count 保留不重置，但从循环第 1 步重新开始
- [ ] **REVR-11**: 恢复后第一步为读取 analysis_state_md_path
- [ ] **REVR-12**: 逆向工件写入 artifacts/{task_id}/reverse/（final.c, static_output.json, frida_oracle_output.json, diff_report.json）
- [ ] **REVR-13**: diff_report.json 包含 match_rate, mismatch_cases, normalization_rules
- [ ] **REVR-14**: 验收额外检查：final.c 独立可编译、包含所有依赖结构体定义、不含未解析偏移量

### Agent Interface（代理接口）

- [ ] **AGNT-01**: 抽象 Runner 接口定义任务执行契约（输入 Task Card + workspace → 输出执行结果）
- [ ] **AGNT-02**: Claude Code CLI 作为首个 Runner 实现，通过 git worktree 隔离执行
- [ ] **AGNT-03**: Runner 接口支持健康检查、取消信号、进度上报

### Connector（连接器）

- [ ] **CONN-01**: Connector 接口定义 discoverTasks()、hydrateContext()、ackResult()、writeBackArtifacts()
- [ ] **CONN-02**: GSD Connector 从 PLAN 生成 Task Card，填充 wave、depends_on，补全新文件路径到 files_to_modify
- [ ] **CONN-03**: GSD Connector 在结果合并后回写 SUMMARY/STATE/ROADMAP/VERIFICATION

### HTTP API（HTTP 接口）

- [ ] **API-01**: RESTful API 暴露任务 CRUD、Wave 操作、状态查询、事件查询
- [ ] **API-02**: API 支持手动创建/编辑 Task Card（非 Connector 来源）
- [ ] **API-03**: WebSocket 端点推送实时任务状态变更

### Web UI（Web 界面）

- [ ] **UI-01**: 任务列表页，支持按状态/wave/dispatch_ref 筛选和排序
- [ ] **UI-02**: 任务详情页，展示 Task Card 完整字段、状态历史、事件日志
- [ ] **UI-03**: Wave 管理页面，展示各 wave 状态（open/sealed）、支持 seal 操作
- [ ] **UI-04**: 任务创建/编辑表单，校验字段格式约束
- [ ] **UI-05**: 全局状态仪表盘，展示任务统计（各状态计数）、活跃 dispatch_ref、合并队列状态
- [ ] **UI-06**: 实时状态更新，任务状态变更通过 WebSocket 推送到前端
- [ ] **UI-07**: 事件日志浏览器，按任务或 dispatch_ref 筛选查看完整事件链

## v2 Requirements

### Multi-User & Auth

- **AUTH-01**: 多用户支持和基本认证
- **AUTH-02**: 任务隔离和权限控制
- **AUTH-03**: API 密钥管理

### Advanced Features

- **ADVN-01**: apply_failed 自动重试策略
- **ADVN-02**: 分布式部署支持
- **ADVN-03**: 任务优先级调度算法
- **ADVN-04**: 更多 Runner 实现（OpenAI Codex 等）
- **ADVN-05**: 更多 Connector 实现（GitHub Issues, Linear 等）
- **ADVN-06**: 逆向专项更多任务类型（reverse_disassembly, reverse_protocol_rebuild）

## Out of Scope

| Feature | Reason |
|---------|--------|
| 多用户/权限系统 | v1 单用户本地运行 |
| 分布式部署 | 纯本地，无远程集群 |
| apply_failed 自动重试 | PRD 明确 v1 人工处理 |
| 非 SQLite 存储后端 | v1 固定 SQLite |
| 移动端适配 | Web UI 只做桌面端 |
| OAuth/外部认证 | 本地工具无需认证 |
| 实时协作 | 单用户场景不需要 |
| 插件市场 | 过度设计，v1 不需要 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CORE-01 ~ CORE-06 | Phase 1 | Pending |
| PERS-01 ~ PERS-06 | Phase 1 | Pending |
| WAVE-01 ~ WAVE-05 | Phase 1 | Pending |
| DEPD-01 ~ DEPD-05 | Phase 1 | Pending |
| STAT-01 ~ STAT-06 | Phase 2 | Pending |
| RETR-01 ~ RETR-07 | Phase 2 | Pending |
| TRAN-01 ~ TRAN-07 | Phase 3 | Pending |
| MERG-01 ~ MERG-06 | Phase 3 | Pending |
| REVR-01 ~ REVR-14 | Phase 4 | Pending |
| AGNT-01 ~ AGNT-03 | Phase 5 | Pending |
| CONN-01 ~ CONN-03 | Phase 5 | Pending |
| API-01 ~ API-03 | Phase 6 | Pending |
| UI-01 ~ UI-07 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 56 total
- Mapped to phases: 56
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-06*
*Last updated: 2026-04-06 after initial definition*
