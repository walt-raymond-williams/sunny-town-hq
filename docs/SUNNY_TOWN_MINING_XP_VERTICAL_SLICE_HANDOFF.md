# Sunny Town Mining XP Vertical Slice Handoff

## Purpose

This handoff starts GitHub issue #34, "Implement mining XP progression vertical slice."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/34

The goal is to turn the accepted stats/skills model into the first real player-facing progression slice by awarding mining XP from server-authoritative harvest events.

## Current Status

- Issue #28 defined the stats and skills progression model.
- `docs/current/STATS_SKILLS_PROGRESSION.md` recommends mining XP as the first implementation.
- The Sunny Town character panel has room for real stats, but should not show fake progression.
- Work should continue on `codex/inventory-redesign-dev`.

## Product Direction

- Progression comes from activity, not manual point allocation.
- Mining is the first vertical slice because successful harvests are already validated by Sunny Town.
- HQ owns durable progression state and idempotency.
- Sunny Town owns live validation before asking HQ to mutate durable XP.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town`
  - Character panel and inventory menu UI.
- `frontend/src/stores/studentInventory.ts`
  - Existing E-menu adjacent state; add a dedicated store/composable if current patterns support it.

Backend:

- `internal/sunnytown/server`
  - Resource harvest validation and event handling.
- `internal/hq/sunnytownbridge`
  - Service-authenticated Sunny Town to HQ calls.
- `internal/hq`
  - Add or extend a package for character skill persistence and student read APIs.
- `deploy/postgres/migrations`
  - Durable skill definition/state/ledger tables.

Docs:

- `docs/current/STATS_SKILLS_PROGRESSION.md`
  - Accepted model and first vertical slice.
- `docs/current/API.md`
  - Update for new service and student APIs.
- `docs/current/DATABASE.md`
  - Update for new progression tables.

## Recommended Implementation Plan

1. Add durable schema for skill definitions, character skill state, and idempotent XP ledger.
2. Seed or define the `mining` skill.
3. Add an HQ service-authenticated award endpoint that is idempotent by harvest/event ID.
4. Call that endpoint from successful Sunny Town resource harvest flow.
5. Add a student-facing read endpoint for current character skills.
6. Render real mining progress in the character panel.
7. Update current-state docs with the accepted API/schema/runtime behavior.

## Out Of Scope

- Manual stat allocation.
- Global character levels.
- NPC skill automation.
- Broad balancing formulas.
- Relationship or social progression.

## Verification

Run:

```powershell
go test ./internal/hq/...
go test ./internal/hq/schema
go test ./internal/sunnytown/server
cd frontend
npm run build
```

Manual smoke:

- Mine a resource and verify mining progress appears in the character panel.
- Repeat or retry the same event path where possible and verify XP is not double-awarded.

## Close Criteria

Close #34 when:

- work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- current-state docs are updated for API/schema/runtime changes
