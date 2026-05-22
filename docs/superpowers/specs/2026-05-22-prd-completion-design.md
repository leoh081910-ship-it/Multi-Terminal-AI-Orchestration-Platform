# PRD Completion Design

Date: 2026-05-22
Project: 多终端 AI 编排平台

## Goal

Move the project from "core functionality is mostly formed" to a state that is verifiable, deliverable, and explicit about remaining risks.

The work follows this sequence:

```text
Baseline audit -> dual-track completion -> final acceptance
```

This design does not restart the platform, expand into long-term v3/P3 ambitions, or treat documentation claims as verification evidence.

## Phase 0: Baseline Audit

Phase 0 establishes the current truth before implementation.

### Scope

1. Audit the working tree:
   - Identify modified tracked files.
   - Identify untracked planning files, databases, backups, and local Claude files.
   - Classify each as likely product work, validation artifact, local runtime artifact, or unknown.
2. Build a PRD gap matrix for core requirements:
   - `done_verified`: code, tests, and real validation evidence exist.
   - `implemented_unverified`: code exists but tests or real validation are missing.
   - `partial`: behavior exists only in part.
   - `missing`: no implementation found.
   - `manual_followup`: requires real external environment or human validation.
3. Establish validation baselines:
   - Frontend build.
   - Backend Go test availability.
   - Service startup availability.
   - Environment blockers such as Go not being on PATH.
4. Scan risks:
   - TODO or placeholder output.
   - manual follow-up items.
   - reverse artifacts that can masquerade as completion.
   - API/frontend contract drift.
   - documentation that claims completion without current validation.

### Output

- Current-state report.
- PRD gap matrix.
- Implementation task split for Phase 1A and Phase 1B.

Phase 0 should not change business code. The only acceptable changes are minimal validation-enabling updates, and only if they are clearly necessary.

## Phase 1A: Release Hardening

Release Hardening turns the ordinary orchestration platform into a stable local production build.

### Scope

1. Automated validation:
   - Run `go test ./... -count=1` where Go is available.
   - Run `npm --prefix web run build`.
   - Include existing lint or frontend test scripts only if already present.
2. Service and API smoke tests:
   - Server startup.
   - `/health`.
   - Task create, read, edit, cancel, and retry.
   - Wave list and seal.
   - Event query.
   - Scheduling and state progression.
   - Basic merge queue flow.
3. Browser smoke tests:
   - Home/control center.
   - Task list.
   - Task detail.
   - Task form.
   - Wave management.
   - Event log.
   - WebSocket or realtime refresh behavior.
4. Windows compatibility smoke tests:
   - Chinese paths.
   - Paths with spaces.
   - Long paths.
   - Artifact copy.
   - `git add` and `git commit` behavior.
   - Worktree and merge behavior.
5. Validation records:
   - Record automated evidence for automatically validated items.
   - Record real environment requirements as `manual_followup`.
   - Do not mark an item complete only because an older document says it is complete.

### Non-goals

- Redesign the UI style.
- Rewrite the scheduler model.
- Add unnecessary platform features.
- Add distributed deployment, Redis/PostgreSQL migration, plugin marketplace, or other long-term expansion.

## Phase 1B: Reverse Capability

Reverse Capability turns `reverse_static_c_rebuild` from an existing framework into a trustworthy closed loop.

### Scope

1. Task contract:
   - Define the minimal Task Card fields, including `target_so_path`, `target_function`, `frida_hook_spec`, `expected_behavior`, `max_loop_iterations`, and artifact output location.
   - Reject missing required fields or route them into a clear error state.
2. Execution loop:
   - IDA MCP static analysis.
   - Frida oracle execution.
   - C code generation.
   - Compile and run generated C.
   - Diff calculation.
   - Continue or finish based on `match_rate`.
3. Artifact contract:
   - `final.c`.
   - `RE-STATE.md`.
   - `static_output.json`.
   - `frida_oracle_output.json`.
   - `diff_report.json`.
   - Final artifacts must not contain TODO placeholders, unresolved offsets, or placeholder functions pretending to be complete.
4. Gates:
   - `match_rate < 100` cannot enter `patch_ready`.
   - `match_rate = 100` plus valid `final.c` is required before `patch_ready`.
   - `max_loop_iterations` exhaustion must produce a clear retry or failure reason.
   - IDA or Frida unavailability must produce `reverse_env_unavailable`, not a misleading business failure.
5. Testing:
   - Use fake IDA, fake Frida, and fake diff adapters for automated tests.
   - Treat real IDA/Frida validation as `manual_followup` unless the environment is present.
   - Keep at least one minimal sample proving the control loop can complete.

### Non-goals

- Build a generic anti-debugging framework.
- Support every architecture or every `.so` shape.
- Automatically solve arbitrary C semantic recovery.
- Bypass real device, IDA, or Frida environment requirements.

## Phase 2: Final Acceptance

Phase 2 combines both tracks into one delivery decision.

### Acceptance inputs

1. Automated validation results:
   - Go tests.
   - Frontend build.
   - Key smoke commands or scripts.
2. Browser validation results:
   - Pages load.
   - Actions take effect.
   - Realtime refresh works.
   - Error states are visible.
3. Core rule validation:
   - Wave seal.
   - Dependencies.
   - Retry.
   - Timeout and recovery.
   - Merge queue.
   - WebSocket events.
   - Reverse match gate.
4. Remaining risks:
   - Environment blockers.
   - Manual follow-up items.
   - Long-term features excluded from this round.
   - Limits of reverse validation against real samples.

### Final conclusion

The final output must explicitly classify the project as one of:

```text
Deliverable
Conditionally deliverable
Not deliverable
```

## Shared constraints

- Preserve the existing project API, state machine, runner, merge, and planning structure.
- Do not introduce a second scheduler, merge process, or runner path.
- Do not delete untracked files such as backups, local databases, or `.claude` files without explicit user confirmation.
- Inspect existing uncommitted work before editing related files.
- Keep changes surgical and tied to validation gaps.
- Record manual verification honestly instead of treating it as automated success.
