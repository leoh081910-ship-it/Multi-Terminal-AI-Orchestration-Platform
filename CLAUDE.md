<!-- GSD:project-start source:PROJECT.md -->
## Project

**多终端 AI 编排平台**

一个单人、本地优先的多 AI 任务编排平台。用户通过 CLI 或 Web UI 提交 Task Card，平台按 Wave 分批、按依赖拓扑排序，调度多个 AI Agent（首个为 Claude Code CLI）在隔离 workspace 中并行执行任务，最后自动合并工件到主仓库。平台内置逆向工程专项任务类型，支持 IDA Pro + Frida 的可量化验证闭环。

**Core Value:** 任务从提交到合并全流程自动化——用户定义"做什么"，平台负责"怎么做"：隔离执行、依赖调度、冲突回避、工件合并、失败重试。

### Constraints

- **Tech Stack**: Go 后端 + React/Vite 前端 — PRD 未指定技术栈，通过讨论确定
- **Storage**: SQLite only — v1 不可配置，单文件数据库
- **OS**: Windows 11 为主要平台 — 必须处理长路径、中文路径、符号链接权限
- **Scale**: 10-20 并发任务 — 中等规模，合并队列串行但路由可并行
- **Agent**: Claude Code CLI 首个实现 — 但必须通过抽象接口支持扩展
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## 推荐技术栈
### Core 后端技术
| 技术 | 版本 | 用途 | 推荐理由 |
|------|------|------|----------|
| Go | 1.23+ | 运行时 | 模块系统稳定，WSL支持成熟，SQLite驱动优质 |
| chi | v5 | HTTP路由 | 轻量(~1000行)，100%兼容net/http，无第三方依赖，生产验证(Cloudflare/Heroku) |
| ent | v0.14.6 | SQLite ORM | 静态类型，代码生成，2026年3月仍活跃更新，SQLite支持完善 |
| coder/websocket | v1.8.14 | WebSocket | 现代Go API，first-class context支持，零依赖，2025年9月最新版本 |
| modernc.org/sqlite | latest | SQLite驱动 | 纯Go实现，无CGO依赖，跨平台编译简单 |
### Core 前端技术
| 技术 | 版本 | 用途 | 推荐理由 |
|------|------|------|----------|
| React | 19.x | UI框架 | 2025-2026主流，hook API成熟，生态系统完善 |
| Vite | 8.0.2 | 构建工具 | Rolldown引擎，tree-shaking优化，内置TypeScript/JSX支持 |
| TypeScript | 5.x | 语言 | 静态类型检查，ent生成的代码完美兼容 |
| TanStack Query | v5 | 服务端状态 | 自动缓存/重取/轮询，零依赖，devtools完善 |
| shadcn/ui | latest | UI组件 | 可定制，Tailwind CSS v4兼容，React 19支持 |
| Tailwind CSS | v4 | 原子化CSS | 2025-2026主流，shadcn/ui深度集成 |
### 开发工具
| 工具 | 用途 | 配置要点 |
|------|------|----------|
| Air | Go热重载 | `.air.toml`配置sql文件监听 |
| sqlc | SQL类型检查 | 与ent二选一，ent更推荐 |
| tsoa | OpenAPI生成 | 路由注释生成Swagger |
| concurrently | 多进程开发 | 并行启动Go后端+前端 |
| ESLint + Prettier | 前端代码规范 | React 19 hooks规则 |
| Vitest | 前端单元测试 | Vite原生集成 |
### 数据库
| 技术 | 版本 | 用途 | 推荐理由 |
|------|------|------|----------|
| SQLite | 3.x | 持久化 | 本地优先架构，single-file，Go驱动成熟 |
## 安装
# Go 后端依赖
# Ent代码生成
# 开发工具
# 前端依赖
# shadcn/ui 基础组件
## 技术选型详解
### 为什么 chi v5 而非 Gin/Echo
| 对比项 | chi v5 | Gin | Echo |
|--------|--------|-----|------|
| 性能 | 高 | 最高 | 高 |
| 依赖 | 0 | 9个 | 9个 |
| net/http兼容 | 100% | 中 | 中 |
| 中间件生态 | 兼容所有 | 自有 | 自有 |
| 维护状态 | 活跃 | 活跃 | 活跃 |
### 为什么 ent 而非 GORM/sqlc
| 对比项 | ent | GORM | sqlc |
|--------|-----|------|------|
| 类型安全 | 100% | 反射 | 100% |
| 代码生成 | 是 | 否 | 是 |
| 学习曲线 | 中 | 低 | 低 |
| SQL控制 | 中 | 高 | 高 |
| Graph遍历 | 是 | 否 | 否 |
### 为什么 coder/websocket 而非 gorilla
| 对比项 | coder/websocket | gorilla/websocket |
|--------|-----------------|-------------------|
| API现代度 | context-first | 传统 |
| 依赖 | 0 | 0 |
| 维护状态 | 活跃(nhooyr演进) | 已归档后恢复 |
| 性能 | 零分配 | 成熟 |
### 为什么 shadcn/ui 而非 MUI/AntD
| 对比项 | shadcn/ui | MUI | AntD |
|--------|------------|-----|------|
| 包体积 | 按需 | 大 | 大 |
| 定制性 | 源码级 | 主题覆盖 | 样式覆盖 |
| React 19 | 支持 | 部分 | 部分 |
| Tailwind集成 | 原生 | 无 | 无 |
### 为什么 TanStack Query 而非 Redux/Zustand
| 对比项 | TanStack Query | Redux | Zustand |
|--------|----------------|-------|---------|
| 定位 | 服务端状态 | 通用状态 | 通用状态 |
| 样板代码 | 极少 | 多 | 少 |
| 缓存机制 | 自动 | 手动 | 手动 |
| 学习曲线 | 低 | 高 | 低 |
## 不推荐使用
| 避免使用 | 原因 | 改用 |
|----------|------|------|
| GORM (如选ent) | 混用ORM导致事务边界混乱 | ent 或 sqlc |
| gorilla/websocket (除非需要) | 曾归档，维护不确定性 | coder/websocket |
| Redux (除非复杂表单) | 重，服务端状态非核心场景 | React Context |
| class组件 | React 19 hook优先 | Function组件+hook |
| styled-components | 与Tailwind冲突 | Tailwind CSS |
| MySQL/PostgreSQL | 本地优先架构 | SQLite |
## 栈变体
- Redis用于WebSocket pub/sub，Centrifugo可选
- PostgreSQL替代SQLite (ent支持)
- Electron Forge 替代 Vite
- Tauri 不推荐 (Rust学习曲线，Go生态已足够)
- React Table (TanStack Table v9) + React Hook Form
- Zod进行前端校验 (ent已生成schema)
## 版本兼容性
| 包 | 兼容版本 | 注意事项 |
|----|----------|----------|
| Go | 1.21+ | 1.23推荐(wasm支持) |
| chi v5 | Go 1.21+ | 使用context新API |
| ent v0.14 | Go 1.18+ | 生成代码依赖泛型 |
| React 19 | Vite 5+ | Vite 8已支持 |
| Tailwind v4 | Vite 6+ | 需@tailwindcss/vite |
| shadcn/ui | React 18+ | v0.14+支持React 19 |
## 信息来源
- [chi v5 GitHub](https://github.com/go-chi/chi) — 官方文档，2024-2025活跃
- [ent v0.14.6 GitHub](https://github.com/ent/ent) — 2026年3月最新发布
- [coder/websocket v1.8.14 GitHub](https://github.com/coder/websocket) — 2025年9月发布
- [Vite v8.0.2](https://vite.dev) — 官方网站，当前稳定版
- [TanStack Query v5](https://tanstack.com/query/latest) — 官方文档
- [shadcn/ui](https://ui.shadcn.com) — 官方文档，React 19兼容性说明
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
