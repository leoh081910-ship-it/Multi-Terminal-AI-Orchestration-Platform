---
phase: 1
slug: foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-06
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Wave 0 installs |
| **Quick run command** | `go test ./internal/store/... -count=1 -short` |
| **Full suite command** | `go test ./... -count=1 -v` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/store/... -count=1 -short`
- **After every plan wave:** Run `go test ./... -count=1 -v`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | SETUP-01 | unit | `go test ./cmd/server/... -run TestServerStartup` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 1 | SETUP-02 | unit | `go test ./cmd/cli/... -run TestCLIStartup` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 1 | PERSISTENCE-01 | unit | `go test ./internal/store/... -run TestTasksTable` | ❌ W0 | ⬜ pending |
| 1-01-04 | 01 | 1 | PERSISTENCE-02 | unit | `go test ./internal/store/... -run TestEventsTable` | ❌ W0 | ⬜ pending |
| 1-01-05 | 01 | 1 | PERSISTENCE-03 | unit | `go test ./internal/store/... -run TestWavesTable` | ❌ W0 | ⬜ pending |
| 1-01-06 | 01 | 1 | PERSISTENCE-04 | unit | `go test ./internal/store/... -run TestCardJSON` | ❌ W0 | ⬜ pending |
| 1-01-07 | 01 | 1 | PERSISTENCE-05 | unit | `go test ./internal/store/... -run TestAtomicity` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/store/store_test.go` — test stubs for all CRUD operations
- [ ] `internal/store/testutil_test.go` — shared test helpers (in-memory SQLite setup)
- [ ] `go test` framework — built into Go, no install needed

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Server starts and listens on port | SETUP-01 | Requires process startup check | Run `go run cmd/server/main.go` and verify no errors |
| CLI command executes | SETUP-02 | Requires process startup check | Run `go run cmd/cli/main.go` and verify output |
| Process restart recovers state | PERSISTENCE-04 | Requires two sequential runs | Insert data, stop server, restart, verify data persists |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
