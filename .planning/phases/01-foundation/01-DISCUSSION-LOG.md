# Phase 1: Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in 01-CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 01-foundation
**Areas discussed:** Project Structure

---

## Project Structure

### Go Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Standard Go Layout | cmd/ + internal/, 业界标准，社区推荐 | ✓ |
| Domain-Driven | 按领域分模块 task/, wave/, event/ | |
| Flat Layout | 简单直接，所有代码同层级 | |

**User's choice:** Standard Go Layout
**Notes:** 用户明确选择社区标准布局，确保可维护性

### Entry Points

| Option | Description | Selected |
|--------|-------------|----------|
| Server + CLI 双入口 | cmd/server + cmd/cli，HTTP API 和命令行并存 | ✓ |
| Server only | 先只做 server | |
| CLI only first | 先做 CLI，server 后续 | |

**User's choice:** Server + CLI 双入口
**Notes:** 双入口设计让平台灵活使用

### Internal Package Organization

| Option | Description | Selected |
|--------|-------------|----------|
| 按功能层分包 | store, engine, transport, api, connector — 层间接口清晰 | ✓ |
| 按领域分包 | task/, wave/, event/ — Phase 1 可能过度设计 | |

**User's choice:** 按功能层分包
**Notes:** 推荐选项，Phase 1 只做持久化层，功能层分包更适合渐进式构建

## Claude's Discretion

- Schema migration 工具选择
- 配置文件格式和加载方式
- 日志框架和格式
- SQLite 连接池参数
- Go module 名称
- 测试辅助工具选择

## Deferred Ideas

None — discussion stayed within phase scope
