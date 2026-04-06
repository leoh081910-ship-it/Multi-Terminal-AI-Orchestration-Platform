# Stack Research

**Domain:** Multi-AI Orchestration Platform (Go + React + SQLite)
**Researched:** 2026-04-06
**Confidence:** MEDIUM-HIGH

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

```bash
# Go 后端依赖
go get github.com/go-chi/chi/v5
go get github.com/enta/ent
go get github.com/enta/ent/dialect/sql
go get modernc.org/sqlite/v3
go get github.com/coder/websocket

# Ent代码生成
go get -d entgo.io/ent/cmd/ent

# 开发工具
go install github.com/air-verse/air@latest
go install entgo.io/ent/cmd/ent@latest

# 前端依赖
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install @tanstack/react-query
npx shadcn@latest init

# shadcn/ui 基础组件
npx shadcn@latest add button card dialog input tabs toast
npm install -D tailwindcss @tailwindcss/vite autoprefixer
```

## 技术选型详解

### 为什么 chi v5 而非 Gin/Echo

| 对比项 | chi v5 | Gin | Echo |
|--------|--------|-----|------|
| 性能 | 高 | 最高 | 高 |
| 依赖 | 0 | 9个 | 9个 |
| net/http兼容 | 100% | 中 | 中 |
| 中间件生态 | 兼容所有 | 自有 | 自有 |
| 维护状态 | 活跃 | 活跃 | 活跃 |

**选chi的理由**: 本项目需要与agent CLI进程通信，chi的零依赖和小体积更适合嵌入式场景。Gin的Logger中间件在生产环境极其实用，Echo功能最全但重量略高。

### 为什么 ent 而非 GORM/sqlc

| 对比项 | ent | GORM | sqlc |
|--------|-----|------|------|
| 类型安全 | 100% | 反射 | 100% |
| 代码生成 | 是 | 否 | 是 |
| 学习曲线 | 中 | 低 | 低 |
| SQL控制 | 中 | 高 | 高 |
| Graph遍历 | 是 | 否 | 否 |

**选ent的理由**: 静态类型是Go的核心优势，ent生成的代码在编译时发现90%的错误。sqlc只做SQL类型检查，缺少ORM的便捷性；GORM依赖反射，运行时才有错误。ent的图遍历能力对Wave/Task关联查询友好。

**备选方案**: 若团队GORM熟练度高，可换GORM。关键是别混用。

### 为什么 coder/websocket 而非 gorilla

| 对比项 | coder/websocket | gorilla/websocket |
|--------|-----------------|-------------------|
| API现代度 | context-first | 传统 |
| 依赖 | 0 | 0 |
| 维护状态 | 活跃(nhooyr演进) | 已归档后恢复 |
| 性能 | 零分配 | 成熟 |

**选coder/websocket的理由**: 项目使用标准context传递取消信号，coder/websocket原生支持。gorilla在2022年曾归档，社区已fork恢复但维护不如以前稳定。

### 为什么 shadcn/ui 而非 MUI/AntD

| 对比项 | shadcn/ui | MUI | AntD |
|--------|------------|-----|------|
| 包体积 | 按需 | 大 | 大 |
| 定制性 | 源码级 | 主题覆盖 | 样式覆盖 |
| React 19 | 支持 | 部分 | 部分 |
| Tailwind集成 | 原生 | 无 | 无 |

**选shadcn/ui的理由**: 不是"库"是"组件源码"，复制到项目中随意定制。Tailwind CSS v4是2025-2026趋势。与ent组合: type-safe后端 + type-safe前端。

### 为什么 TanStack Query 而非 Redux/Zustand

| 对比项 | TanStack Query | Redux | Zustand |
|--------|----------------|-------|---------|
| 定位 | 服务端状态 | 通用状态 | 通用状态 |
| 样板代码 | 极少 | 多 | 少 |
| 缓存机制 | 自动 | 手动 | 手动 |
| 学习曲线 | 低 | 高 | 低 |

**选TanStack Query的理由**: 多AI任务状态需要实时更新(轮询/WebSocket)，TanStack Query处理服务端状态(缓存/重取/竞态)极其擅长。本地UI状态用React Context足够。

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

**场景: 需要多实例部署**
- Redis用于WebSocket pub/sub，Centrifugo可选
- PostgreSQL替代SQLite (ent支持)

**场景: Electron桌面端**
- Electron Forge 替代 Vite
- Tauri 不推荐 (Rust学习曲线，Go生态已足够)

**场景: 复杂CRUD管理后台**
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

---

*Stack research for: Multi-AI Orchestration Platform*
*Researched: 2026-04-06*