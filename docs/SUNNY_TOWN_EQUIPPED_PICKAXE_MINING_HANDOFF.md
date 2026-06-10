# Sunny Town Equipped Pickaxe Mining Handoff

## Purpose

This handoff starts GitHub issue #36, "Require equipped pickaxe for Sunny Town mining."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/36

The goal is to fix a blocking PR review finding: mining must require a server-validated equipped pickaxe, not just pickaxe ownership somewhere in inventory.

## Current Status

- PR review requested changes before merging `codex/inventory-redesign-dev`.
- `internal/sunnytown/server/client_gameplay.go` currently authorizes mining through inventory ownership.
- `internal/sunnytown/server/room_mining_test.go` includes expectations that need to change with the authority rule.
- This issue blocks the inventory redesign PR merge.

## Product And Architecture Direction

- Browser/client messages are requests, not authority.
- Sunny Town owns live action validation: accepted player position, target node, and equipped tool requirement.
- HQ owns durable equipment state.
- Mining requires the `tool` equipment slot to be `pickaxe`.
- Owning a pickaxe in inventory is not enough.

## Current Code Surface

Backend:

- `internal/sunnytown/server/client_gameplay.go`
  - `handleToolUse` currently calls `ownsInventoryItem(...)` for mining authorization.
- `internal/sunnytown/server/room_mining_test.go`
  - Update tests to prove owning-but-not-equipped fails and equipped pickaxe succeeds.
- `internal/sunnytown/hqclient/client.go`
  - Existing Sunny Town to HQ client surface; inspect before adding new bridge calls.
- `internal/hq/sunnytownbridge/http.go`
  - Existing service-authenticated HQ bridge endpoints.
- `internal/hq/inventory/equipment.go`
  - Durable equipment behavior and validation.

Docs:

- `AGENTS.md`
  - Authority rule: gameplay effects use accepted server position and equipped tools.
- `docs/current/API.md`
  - Update only if endpoint behavior/contract changes.
- `docs/current/ARCHITECTURE.md` and `docs/current/SCHEMA_OWNERSHIP.md`
  - Update only if the accepted authority boundary needs clarification.

## Recommended Implementation Plan

1. Inspect how Sunny Town refreshes or loads player equipment today.
2. Add or reuse a server-side check that confirms the authenticated player's equipped `tool` slot is `pickaxe`.
3. Replace mining authorization based on `ownsInventoryItem(...)` with equipped-tool authorization.
4. Ensure invalid mining attempts do not deplete nodes, mutate inventory, or award XP.
5. Update room mining tests for equipped, owned-but-not-equipped, and missing-pickaxe cases.
6. Run focused Sunny Town tests and `task verify`.

## Out Of Scope

- Changing hotbar semantics.
- Rebalancing mining XP.
- Adding new equipment slots.
- Frontend UX changes beyond whatever is necessary to preserve current valid mining flow.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
go test ./internal/hq/inventory
task verify
```

Manual smoke if practical:

- Put a pickaxe in inventory but do not equip it; mining should fail.
- Equip the pickaxe; mining should succeed and award XP once.

## Close Criteria

Close #36 when:

- server-side mining requires equipped pickaxe
- tests cover the regression
- verification results are recorded on the issue
- work is committed and pushed to `codex/inventory-redesign-dev`
