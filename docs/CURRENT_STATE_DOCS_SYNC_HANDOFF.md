# Current-State Docs Sync Handoff

## Purpose

This handoff starts GitHub issue #6, "Keep current-state docs synchronized with architecture changes."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/6

The goal is to do the final current-state documentation and integration-review pass for `codex/inventory-redesign-dev` before opening or merging the feature PR to `main`.

## Current Status

- The Sunny Town inventory redesign feature work is complete on `codex/inventory-redesign-dev`.
- The tracking doc marks inventory redesign issues #8 through #35 as closed.
- Deterministic verification passed after installing Go Task and fixing Taskfile Compose paths:
  - `task verify`
- GitHub CLI auth was expired during handoff preparation, so the agent should refresh auth before reading/commenting on #6:
  - `gh auth refresh -h github.com`

## Product Direction

- Treat this as review and documentation sync, not a new feature slice.
- The final PR should be understandable to a reviewer who starts from `docs/current/`.
- Historical planning docs can stay as history, but current-state docs must describe what the branch actually does now.
- Do not broaden #6 into new gameplay, schema, or UI work unless the review finds a real correctness bug.

## Current Code Surface

Docs to audit first:

- `docs/current/API.md`
  - Inventory move/split, crafting, shop stock, container transfer, progression APIs.
- `docs/current/DATABASE.md`
  - Inventory slots, containers, shop input/output storage, NPC production blocked ledger, character progression.
- `docs/current/ARCHITECTURE.md`
  - HQ/Sunny Town ownership boundaries and service-authenticated flows.
- `docs/current/CONTAINER_STORAGE.md`
  - Container identity, access validation, transfer behavior.
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
  - Historical slice notes and current implemented Cookie Shop storage behavior.
- `docs/current/STATS_SKILLS_PROGRESSION.md`
  - Accepted progression model and implemented mining XP slice.
- `docs/current/SCHEMA_OWNERSHIP.md`
  - Durable ownership boundaries for inventory, shop storage, progression, and Sunny Town realtime state.
- `docs/current/PACKAGE_BOUNDARIES.md`
  - Package-level ownership changes introduced by the feature branch.

Supporting docs:

- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TRACKING.md`
  - Closed/open issue snapshot and PR-readiness context.
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
  - Historical roadmap; do not treat stale future wording here as current state unless it leaks into `docs/current/`.
- `AGENTS.md`
  - Repository workflow and PR/verification expectations.

## Recommended Implementation Plan

1. Refresh GitHub auth and read issue #6 comments.
2. Compare `docs/current/` against the completed branch behavior.
3. Fix stale current-state language, especially anything that says implemented behavior is still future/deferred.
4. Confirm the tracking doc accurately names remaining work before PR.
5. Run `task verify`.
6. Optionally run Docker Compose runtime smoke if practical:
   - `task compose:rebuild-runtime`
   - `task health`
7. Commit and push documentation-only changes to `codex/inventory-redesign-dev`.
8. Comment on #6 with commit hash and verification results.

## Out Of Scope

- New inventory, crafting, Cookie Shop, container, or progression features.
- Broad historical roadmap cleanup unless it blocks reviewer understanding.
- Generated frontend `web/` assets.
- Local logs.

## Verification

Run:

```powershell
task verify
```

Recommended runtime smoke if Docker is available and time allows:

```powershell
task compose:rebuild-runtime
task health
```

Manual review:

- Open `docs/current/` and confirm the implemented inventory redesign can be understood without reading every handoff doc.
- Confirm the PR description can cite the verification commands run.

## Close Criteria

Close #6 when:

- current-state docs match the implemented architecture/API/schema/runtime behavior on `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- the final PR is ready for human review
