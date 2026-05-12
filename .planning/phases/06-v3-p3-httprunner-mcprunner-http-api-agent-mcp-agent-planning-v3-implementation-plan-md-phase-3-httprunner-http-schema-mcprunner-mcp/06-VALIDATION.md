---
phase: 6
slug: v3-p3-httprunner-mcprunner-http-api-agent-mcp-agent-planning-v3-implementation-plan-md-phase-3-httprunner-http-schema-mcprunner-mcp
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-13
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `httptest`; frontend ESLint/TypeScript/Vite build |
| **Config file** | `go.mod`, `web/package.json`, `web/vite.config.ts` |
| **Quick run command** | `go test ./internal/runner ./internal/registry ./internal/server` |
| **Full suite command** | `go test ./... && npm --prefix web run lint && npm --prefix web run build` |
| **Estimated runtime** | ~60-120 seconds |

---

## Sampling Rate

- **After every backend task commit:** Run `go test ./internal/runner ./internal/registry ./internal/server`
- **After every frontend task commit:** Run `npm --prefix web run lint && npm --prefix web run build`
- **After every plan wave:** Run `go test ./... && npm --prefix web run lint && npm --prefix web run build`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds for quick backend checks; 240 seconds for full suite

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | HTTP-01 | unit | `go test ./internal/runner -run 'TestHTTPRunnerConfig_Validate|TestHTTPRunner_Execute_TemplateBody|TestHTTPRunner_Execute_DefaultBody' -count=1` | existing + new tests | ⬜ pending |
| 06-01-02 | 01 | 1 | HTTP-01 | unit | `go test ./internal/runner -run 'TestHTTPRunner_TemplateToJSON|TestHTTPRunnerConfig_Validate' -count=1` | existing + new tests | ⬜ pending |
| 06-02-01 | 02 | 1 | MCP-01 | unit | `go test ./internal/runner -run 'TestMCPRunnerConfig_Validate|TestMCPRunner_Execute|TestMCPRunner_GetCapabilities_DiscoverTools' -count=1` | existing + new tests | ⬜ pending |
| 06-03-01 | 03 | 2 | HTTP-01/MCP-01 | integration | `go test ./internal/server -run 'TestHandle(Create|Update)Agent_Invalid.*RunnerConfig' -count=1` | new tests | ⬜ pending |
| 06-04-01 | 04 | 2 | HTTP-01/MCP-01 | frontend | `npm --prefix web run lint && npm --prefix web run build` | existing page | ⬜ pending |
| 06-05-01 | 05 | 3 | HTTP-01/MCP-01 | docs + full suite | `go test ./... && npm --prefix web run lint && npm --prefix web run build` | existing docs | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing test infrastructure covers this phase. No new framework installation required.

Required test additions during execution:
- [ ] `internal/runner/http_runner_test.go` — `toJSON`, RunnerTask field access, default body regression
- [ ] `internal/runner/mcp_runner_test.go` — `tool_name`, `{name, arguments}` params, SSE rejection, tools/list discovery
- [ ] `internal/server/*_test.go` — invalid HTTP/MCP create/update returns 400 and does not persist or corrupt existing Agent config

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Docs/examples are copy-runnable and match serialized `runner_config` contract | HTTP-01/MCP-01 | Shell quoting and env-var setup vary by platform | Review `docs/examples/README.md`, `docs/examples/*.json`, and `docs/agent-onboarding.md`; confirm examples use RunnerTask fields and serialized `runner_config` string |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or existing infrastructure
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 framework setup not required
- [x] No watch-mode flags
- [x] Feedback latency < 240s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-05-13
