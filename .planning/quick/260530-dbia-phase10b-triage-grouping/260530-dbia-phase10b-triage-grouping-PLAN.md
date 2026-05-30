# DBIA Phase 10b Triage Safety And Grouping Plan

## Scope

- Add a confirmation dialog before single and batch Won't Fix actions.
- Add a grouped triage view on top of the existing filtered/sorted task list.
- Keep the implementation frontend-only and avoid backend pagination or virtual scrolling.

## Checks

- TypeScript build must pass.
- Existing triage Playwright spec must still pass.
- Add or update browser coverage for grouping controls and Won't Fix confirmation.

## Non-Goals

- No backend API changes.
- No virtual scrolling or pagination.
- No unrelated worktree cleanup.
