# DBIA Runbook

Dual-Brain Integrated Agent Orchestration — Operations Guide

## Quick Start

### 1. Start Services (order matters)

```powershell
# Terminal 1: Go orchestration server
cd "E:\vibe coding\Projects\多终端 AI 编排平台"
go run ./cmd/server

# Terminal 2: Python reviewer + planner
cd "E:\vibe coding\Projects\workflow-library-system\scripts"
python workflow_web.py
```

### 2. Verify Environment

```powershell
cd "E:\vibe coding\Projects\workflow-library-system\scripts"
python dbia_env_check.py
```

Expected output: `ALL CHECKS PASSED`

### 3. Run E2E Smoke

```powershell
python dbia_e2e_smoke.py
```

---

## Architecture Overview

```
dbia_llm_planner.py     LLM planning (goal → DAG draft)
    ↓ fallback
dbia_planner.py          Deterministic planning (goal → DAG)
    ↓
dbia_dispatch.py         Dispatch client (bulk API / wave API / polling)
    ↓
Go AutoDispatcher        Execution (Runner / ReviewWorker / MergeQueue)
    ↓
Python workflow_web.py   Reviewer endpoint (POST /api/runner/review)
```

## Service Endpoints

| Service | URL | Purpose |
|---------|-----|---------|
| Go API | http://127.0.0.1:8080 | Orchestration, task CRUD, dispatch |
| Python Reviewer | http://127.0.0.1:8765/api/runner/review | LLM-based code review |
| Python Health | http://127.0.0.1:8765/api/runner/review (GET) | Health check |

## Task Lifecycle

```
queued → routed → running → review_pending → verified → done
                                    ↓ (rejected)
                              retry_waiting → [rework] → review_pending → ...
                                    ↓ (anti-loop limit)
                                  blocked (manual triage)
```

## Review Reports

Reports are written to disk at:
```
.orchestrator/artifacts/{review-task-id}/.orchestrator/reports/{review-task-id}-code-review.md
```

Each report contains:
- **Decision**: approved or rejected
- **Summary**: LLM's analysis of the artifacts
- **Raw LLM Output**: full LLM response for debugging

## Common Issues

### Task stuck in `retry_waiting`

**Cause**: CLI executor (Claude/Gemini/Codex) not available or failed.

**Fix**: Check that the CLI runner is installed and configured in `config.yaml`. For smoke testing, the E2E script applies CLI fallback automatically.

### Review task stuck in `running`

**Cause**: Python reviewer not responding (down or network issue).

**Fix**:
1. Check Python server is running: `curl http://127.0.0.1:8765/api/runner/review`
2. If down, restart: `cd scripts && python workflow_web.py`
3. Go ReviewWorker auto-approves stuck reviews after 15 minutes (safety valve).

### Task in `blocked` state (manual triage)

**Cause**: Anti-loop policy triggered — either:
- `auto_repair_count >= 2` (too many rework attempts)
- Same rejection reason repeated (defect persists after fix)

**Fix**:
1. Read the review report at the artifact path
2. Manually fix the artifacts
3. Reset the task state: update DB directly or use API

### Reviewer not registered (falls back to Claude CLI)

**Cause**: Seed not run or Go server not restarted after seed.

**Fix**:
```powershell
cd "E:\vibe coding\Projects\多终端 AI 编排平台\scripts"
python seed_reviewer.py
# Then restart Go server
```

### LLM planner returns 401 Unauthorized

**Cause**: `.env` file missing or API key expired.

**Fix**: Check `E:\vibe coding\Projects\workflow-library-system\.env` has valid:
```
WORKFLOW_LLM_API_KEY=your_key
WORKFLOW_LLM_BASE_URL=https://your-endpoint/v1
WORKFLOW_LLM_MODEL=your_model
```

## DB Seed (Idempotent)

```powershell
cd "E:\vibe coding\Projects\多终端 AI 编排平台\scripts"
python seed_reviewer.py
```

Safe to run multiple times. Verifies registration after insert.

## Testing

```powershell
# Go regression tests
cd "E:\vibe coding\Projects\多终端 AI 编排平台"
go test ./internal/server/ -count=1

# Python planner + reviewer tests
cd "E:\vibe coding\Projects\workflow-library-system\scripts"
python -m pytest test_dbia_planner.py test_dbia_llm_planner.py test_review_endpoint.py -v

# Full E2E smoke
python dbia_e2e_smoke.py

# Triage dashboard E2E (API + browser)
python dbia_triage_e2e.py

# Playwright browser tests (requires Go server running with frontend built)
cd "E:\vibe coding\Projects\多终端 AI 编排平台\web"
npx playwright install chromium
npx playwright test e2e/triage-dashboard.spec.ts
```

## Key Files

| File | Location | Purpose |
|------|----------|---------|
| `seed_reviewer.sql` | `多终端 AI 编排平台/scripts/` | Idempotent SQL seed |
| `seed_reviewer.py` | `多终端 AI 编排平台/scripts/` | Seed + verification |
| `dbia_planner.py` | `workflow-library-system/scripts/` | Deterministic DAG planner |
| `dbia_llm_planner.py` | `workflow-library-system/scripts/` | LLM-based planner |
| `dbia_dispatch.py` | `workflow-library-system/scripts/` | Dispatch client |
| `dbia_e2e_smoke.py` | `workflow-library-system/scripts/` | E2E smoke test |
| `dbia_env_check.py` | `workflow-library-system/scripts/` | Environment checker |
| `review_executor.py` | `workflow-library-system/scripts/runtime/executors/` | Review endpoint logic |
| `defect_ticket.go` | `多终端 AI 编排平台/internal/server/` | Defect ticket + anti-loop |
| `review_worker.go` | `多终端 AI 编排平台/internal/server/` | Review orchestration |
