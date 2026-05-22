---
phase: 4
slug: integration
status: passed_with_manual_followup
nyquist_compliant: true
wave_0_complete: true
verified: 2026-05-16T00:00:00Z
manual_followup:
  - "Windows 11 中文路径/长路径工作区执行一次真实 merge 流程，确认 artifact copy、git add、git commit 行为稳定"
created: 2026-05-16
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go package tests |
| **Quick run command** | `go test ./internal/connector ./internal/mergequeue -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/connector ./internal/mergequeue -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 4-01-01 | 01 | 1 | CONN-01, CONN-02, CONN-03 | unit/integration | `go test ./internal/connector -run 'Test(DiscoverTasks|HydrateContext|AckResult|WriteBackArtifacts)$' -count=1` | ✅ | ✅ green |
| 4-02-01 | 01 | 1 | MERG-01, MERG-02, MERG-03, MERG-04, MERG-05, MERG-06 | integration | `go test ./internal/mergequeue -count=1` | ✅ | ✅ green |
| 4-03-01 | 02 | 1 | AGNT-01, AGNT-02, AGNT-03 | integration | `go test ./internal/server -run 'Test(MergeQueueAdapterSyncsCompatPayloadOnDone|MergeQueueRepositoryAdapterFiltersVerifiedAndChecksDependencies)$' -count=1` | ✅ | ✅ green |
| 4-04-01 | phase | 1 | Phase 4 automated integration baseline | end-to-end integration | `go test ./... -count=1` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/connector/gsd_test.go` — connector contract coverage for discover/hydrate/ack/writeBack
- [x] connector -> queue -> server-side adapter automated integration baseline via focused regressions plus full `go test ./... -count=1`

---

## Manual-Only Verifications

| Behavior | Coverage Note | Why Manual | Test Instructions |
|----------|---------------|------------|-------------------|
| Windows 长路径/中文路径下的 artifact copy 与 git merge 行为 | Native Windows path-semantics confidence beyond current automated local fixtures | 当前自动化测试难以稳定覆盖真实 Windows 路径特性 | 在 Windows 11 本机使用中文路径工作区执行一次完整 merge 流程，确认 artifact copy、`git add`、`git commit` 全部成功 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or manual follow-up explicitly called out
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references from the original validation draft
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** automated scope passed; one manual Windows path follow-up remains
