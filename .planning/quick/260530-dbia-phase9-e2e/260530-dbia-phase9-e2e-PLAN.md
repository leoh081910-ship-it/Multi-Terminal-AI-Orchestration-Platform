---
phase: quick-260530-dbia-phase9-e2e
type: quick-plan
status: planned
mode: quick
scope:
  - "Package Phase 9 DBIA triage E2E automation into the Go and Python repositories."
  - "Keep commits limited to Phase 9 E2E scripts, runbook docs, and selector stability needed by the browser spec."
  - "Do not stage unrelated local runtime artifacts or pre-existing dirty files."
files_modified:
  - docs/dbia_runbook.md
  - web/e2e/triage-dashboard.spec.ts
  - web/src/pages/TriageDashboardPage.tsx
  - .planning/quick/260530-dbia-phase9-e2e/260530-dbia-phase9-e2e-SUMMARY.md
external_files:
  - ../workflow-library-system/scripts/dbia_triage_e2e.py
---

# Quick Plan: DBIA Phase 9 Triage E2E

## Objective

Close Phase 9 by committing the triage API/browser E2E automation and documenting how to run it.

## Tasks

1. Inspect the new API-driven Python E2E script and browser Playwright spec.
2. Add stable dashboard selectors if needed so browser tests do not depend on inline style text.
3. Run focused verification that is feasible in the current environment.
4. Commit Phase 9 files only and tag the Go baseline.

## Verification

- `go test ./internal/server -count=1 -timeout 120s -p 1 -ldflags="-w -s"`
- `npm run build`
- Python E2E script syntax check.
- Playwright spec syntax/install check when local dependencies allow it.
