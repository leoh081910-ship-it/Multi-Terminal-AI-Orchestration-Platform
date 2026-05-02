---
phase: quick-260502-olx
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/components/GanttChart.tsx
  - web/src/hooks/useWebSocket.ts
  - web/src/pages/AgentWorkbenchPage.tsx
  - web/package.json
  - config.yaml
autonomous: true
requirements:
  - QUICK-260502-OLX
must_haves:
  truths:
    - "Frontend lint no longer reports Date.now() during render in GanttChart or AgentWorkbenchPage."
    - "Frontend lint no longer reports the reconnect callback as accessed before declaration/update in useWebSocket."
    - "web/package.json scripts use package-relative Vite/TypeScript paths and do not reference the old ai-orchestration-platform directory."
    - "config.yaml runtime commands point at this repository's scripts/runtime/*.ps1 instead of the old ai-orchestration-platform path."
    - "Validation reruns document frontend lint/build status, startup endpoint status if runnable, and Go test blockage if the Scoop Go shim is still broken."
  artifacts:
    - path: "web/src/components/GanttChart.tsx"
      provides: "Stable render-safe timestamp input for Gantt fallback times"
      contains: "useState"
    - path: "web/src/hooks/useWebSocket.ts"
      provides: "Reconnect scheduling that satisfies React hooks lint"
      contains: "scheduleReconnect"
    - path: "web/src/pages/AgentWorkbenchPage.tsx"
      provides: "Stable render-safe timestamp input for agent online status"
      contains: "nowMs"
    - path: "web/package.json"
      provides: "Portable frontend scripts"
      contains: "vite.config.ts"
    - path: "config.yaml"
      provides: "Runtime command paths for the current Chinese-path repository"
      contains: "多终端 AI 编排平台/scripts/runtime"
  key_links:
    - from: "web/src/components/GanttChart.tsx"
      to: "React render lifecycle"
      via: "stable state initializer or effect-updated clock value used inside useMemo"
      pattern: "useState.*Date\.now|nowMs"
    - from: "web/src/hooks/useWebSocket.ts"
      to: "ws.onclose reconnect timeout"
      via: "declared callback/function that does not violate no-use-before-define or exhaustive-deps"
      pattern: "setTimeout\(.*3000\)"
    - from: "web/package.json"
      to: "web/vite.config.ts and web/dist"
      via: "relative script arguments"
      pattern: "--config \\.\\\\vite\.config\.ts|--config ./vite\.config\.ts|--outDir \\.\\\\dist|--outDir ./dist"
    - from: "config.yaml"
      to: "scripts/runtime/*.ps1"
      via: "current repository absolute runtime commands"
      pattern: "多终端 AI 编排平台/scripts/runtime/(claude|gemini|codex)\.ps1"
---

<objective>
Fix the current validation failures in the multi-terminal AI orchestration platform without changing feature behavior.

Purpose: Restore the known-good validation baseline by addressing three frontend lint errors and two stale path configuration issues introduced by repository relocation.
Output: Updated React files, portable frontend package scripts, corrected runtime command paths, and fresh validation results.
</objective>

<execution_context>
@E:/04-Claude/Runtime/.claude/get-shit-done/workflows/execute-plan.md
@E:/04-Claude/Runtime/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@E:/04-Claude/Projects/多终端 AI 编排平台/.planning/STATE.md
@E:/04-Claude/Projects/多终端 AI 编排平台/CLAUDE.md
@E:/04-Claude/Projects/多终端 AI 编排平台/web/src/components/GanttChart.tsx
@E:/04-Claude/Projects/多终端 AI 编排平台/web/src/hooks/useWebSocket.ts
@E:/04-Claude/Projects/多终端 AI 编排平台/web/src/pages/AgentWorkbenchPage.tsx
@E:/04-Claude/Projects/多终端 AI 编排平台/web/package.json
@E:/04-Claude/Projects/多终端 AI 编排平台/config.yaml

<interfaces>
Current failure locations already observed and must be fixed exactly:
- `web/src/components/GanttChart.tsx:28` calls `Date.now()` during render inside `useMemo`.
- `web/src/hooks/useWebSocket.ts:41` schedules `setTimeout(connect, 3000)` inside the callback in a way ESLint reports as accessed before declaration/update.
- `web/src/pages/AgentWorkbenchPage.tsx:64` calls `Date.now()` during render in `AgentCard`.
- `web/package.json` scripts hardcode `E:\\04-Claude\\Projects\\ai-orchestration-platform\\web\\vite.config.ts` and `...\\web\\dist`.
- `config.yaml` runtime commands hardcode `E:/04-Claude/Projects/ai-orchestration-platform/scripts/runtime/*.ps1`.

Preserve stack and behavior from project instructions: Go backend + React/Vite frontend, SQLite, Windows 11 primary platform, local-first operation.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Fix the three frontend lint errors without changing UI behavior</name>
  <files>web/src/components/GanttChart.tsx, web/src/hooks/useWebSocket.ts, web/src/pages/AgentWorkbenchPage.tsx</files>
  <behavior>
    - GanttChart still uses current time as a fallback for missing `created_at`/`updated_at`, but the timestamp is captured outside render-sensitive expressions so lint passes.
    - AgentCard still marks agents online when `last_heartbeat_at` is within 60 seconds, but the comparison timestamp is captured via stable state/prop plumbing rather than direct `Date.now()` during render.
    - useWebSocket still reconnects 3 seconds after close and cleans up the timeout on unmount, but reconnect scheduling is declared in an order/dependency structure accepted by ESLint.
  </behavior>
  <action>
    In `GanttChart.tsx`, replace the render-time `Date.now()` inside `useMemo` with a stable timestamp value such as `const [nowMs] = useState(() => Date.now())`, then use `nowMs` inside the memo dependency list. Import `useState` alongside `useMemo`. Do not add timers or animations; this is only a lint-safe fallback timestamp.

    In `AgentWorkbenchPage.tsx`, compute a stable `nowMs` once in the parent component with `useState(() => Date.now())`, pass it to each `AgentCard`, update `AgentCard` props to `{ agent, nowMs }`, and use `nowMs - new Date(agent.last_heartbeat_at).getTime() < 60000`. Do not call `Date.now()` inside `AgentCard` render.

    In `useWebSocket.ts`, refactor reconnect scheduling so the timeout callback invokes a declared callback/function without violating hooks lint. Prefer introducing a `scheduleReconnect` callback declared before `connect`, and store the latest connect function in a ref if needed to avoid circular callback dependencies. Preserve `projectId` URL behavior, `connected` state updates, JSON message parsing, `ws.onerror` close behavior, and cleanup that clears `reconnectTimer.current`.
  </action>
  <verify>
    <automated>cd "E:/04-Claude/Projects/多终端 AI 编排平台/web" && npm run lint</automated>
  </verify>
  <done>`npm run lint` no longer reports the three observed errors in `GanttChart.tsx`, `useWebSocket.ts`, or `AgentWorkbenchPage.tsx`, and the WebSocket reconnect behavior remains present.</done>
</task>

<task type="auto">
  <name>Task 2: Replace stale hardcoded old-project paths in frontend and runtime config</name>
  <files>web/package.json, config.yaml</files>
  <action>
    In `web/package.json`, replace all script references to `E:\\04-Claude\\Projects\\ai-orchestration-platform\\web\\vite.config.ts` with package-relative config paths. Keep the existing Windows-friendly `cmd /d /c` wrapper if it is required for this project, but the scripts must not depend on the old absolute project directory. Use `--config .\\vite.config.ts` for `dev`, `build`, and `preview`; use `--outDir .\\dist` for `build`; keep `tsc -b` and `eslint .` unchanged. Do not change dependencies or devDependencies.

    In `config.yaml`, replace every runtime command path under each project agent (`claude`, `gemini`, `codex`) from `E:/04-Claude/Projects/ai-orchestration-platform/scripts/runtime/*.ps1` to `E:/04-Claude/Projects/多终端 AI 编排平台/scripts/runtime/*.ps1`. Preserve PowerShell invocation syntax, shell values, project IDs, repo roots, workspace paths, and YAML structure.
  </action>
  <verify>
    <automated>cd "E:/04-Claude/Projects/多终端 AI 编排平台/web" && npm run build</automated>
  </verify>
  <done>`web/package.json` contains no `ai-orchestration-platform` path, `config.yaml` runtime command paths target the current repository, and frontend build succeeds using package-relative script paths.</done>
</task>

<task type="auto">
  <name>Task 3: Rerun validation and record remaining environment blockers</name>
  <files>E:/04-Claude/Projects/多终端 AI 编排平台/.planning/quick/260502-olx-ai-lint-3-web-package-json-config-yaml-r/260502-olx-SUMMARY.md</files>
  <action>
    Run focused validation after the code/config fixes. First run frontend lint and build from `web/`. Then attempt backend validation from the repository root with `go test ./...`; if it fails because the Scoop Go shim is broken, do not fix Scoop in this quick task unless the fix is trivial and already obvious. Record the Go test result as blocked by the broken Scoop Go shim, including the exact command and summarized error.

    If build succeeds, perform endpoint startup checks if possible: start the server using the repository's normal run command or built binary, then check `http://localhost:8080/board` and a health/API endpoint already present in the project. If startup is blocked by the same Go shim or another environment issue, record it as an environment blocker rather than expanding scope. Create the required quick summary with commands run, pass/fail results, files changed, and any remaining blockers.
  </action>
  <verify>
    <automated>cd "E:/04-Claude/Projects/多终端 AI 编排平台/web" && npm run lint && npm run build</automated>
    <automated>cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go test ./...</automated>
  </verify>
  <done>Summary exists with frontend lint/build results, endpoint startup check result if runnable, and explicit note that Go tests are blocked by the broken Scoop Go shim if that environment problem persists.</done>
</task>

</tasks>

<verification>
Run these commands after implementation:

1. `cd "E:/04-Claude/Projects/多终端 AI 编排平台/web" && npm run lint`
2. `cd "E:/04-Claude/Projects/多终端 AI 编排平台/web" && npm run build`
3. `cd "E:/04-Claude/Projects/多终端 AI 编排平台" && go test ./...`
4. If Go/runtime is runnable, start the backend and check `http://localhost:8080/board` plus an available health/API endpoint. If Go fails due to the known broken Scoop Go shim, document this as blocked rather than treating it as an implementation failure.
</verification>

<success_criteria>
- The three specified frontend lint failures are fixed.
- `web/package.json` no longer hardcodes the old `E:\04-Claude\Projects\ai-orchestration-platform` path.
- `config.yaml` runtime commands no longer hardcode the old `E:/04-Claude/Projects/ai-orchestration-platform/scripts/runtime` path.
- `npm run lint` passes from `web/`.
- `npm run build` passes from `web/`.
- Endpoint startup checks are attempted when possible.
- Go test status is recorded, with the broken Scoop Go shim noted as a separate environment blocker if still present.
</success_criteria>

<output>
After completion, create `E:/04-Claude/Projects/多终端 AI 编排平台/.planning/quick/260502-olx-ai-lint-3-web-package-json-config-yaml-r/260502-olx-SUMMARY.md`.
</output>
