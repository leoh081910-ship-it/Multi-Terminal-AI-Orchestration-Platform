---
phase: quick-260530-dbia-lineage-tests
type: quick-plan
status: planned
mode: quick
scope:
  - "Add focused backend regression coverage for the Phase 8 triage lineage endpoint."
  - "Do not change Phase 8 production behavior unless a test exposes a real contract gap."
  - "Leave unrelated working-tree changes untouched."
files_modified:
  - internal/server/triage_api_test.go
  - .planning/quick/260530-dbia-lineage-tests/260530-dbia-lineage-tests-SUMMARY.md
---

# Quick Plan: DBIA Lineage Regression Tests

## Objective

Lock the Phase 8 `/triage/tasks/{id}/lineage` contract with backend tests so future changes cannot silently break the rework/review timeline used by the Triage dashboard.

## Tasks

1. Add a focused lineage test covering original, review, rework, and triage entries under one `root_task_id`.
2. Add a 404 test for missing lineage task IDs.
3. Run focused server tests with low-memory settings.
4. Record the result in a quick summary.

## Verification

- `go test ./internal/server -run "TestCompatTriage(Lineage|Batch)" -count=1 -timeout 120s -p 1 -ldflags="-w -s"`
