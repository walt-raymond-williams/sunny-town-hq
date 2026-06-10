# Sunny Town Player Container Transfer Handoff

## Issue

GitHub issue #29: Add player to container transfer operations

https://github.com/walt-raymond-williams/sunny-town-hq/issues/29

## Branch

Work against:

`codex/inventory-redesign-dev`

## Status

This issue is currently blocked.

Do not start implementation until #26 is closed and the accepted container identity/access contract is available.

## Goal

Implement backend operations for moving item stacks between player inventory and an accessible container.

Transfers must be durable, transactional, and protected by server-side access validation. The browser should never be able to mutate arbitrary container IDs just by sending them.

## Dependencies and Related Work

Blocked by:

- #26 Design general container storage schema and access contract
- #11 Add inventory stack move, swap, and merge API, closed

Related:

- #30 Render chest inventory grid
- #25 Add storage context to crafting recipe APIs

## Likely Code Areas

- `internal/hq/inventory`
- `internal/hq/sunnytownbridge`
- `internal/sunnytown/server`
- `internal/sunnytown/hqclient`
- `deploy/postgres/migrations`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/ARCHITECTURE.md`

## Scope

In scope:

- Load accessible container contents.
- Transfer stack quantities from player inventory to container.
- Transfer stack quantities from container to player inventory.
- Swap or merge compatible stacks where applicable.
- Make operations transactional.
- Enforce access validation through the accepted #26 contract.
- Add concurrency/conflict tests.

Out of scope:

- Chest frontend grid.
- Storage-aware crafting behavior.
- Cookie Shop production changes.
- Broad inventory UI redesign.

## Acceptance Criteria

- Backend can load accessible container contents.
- Backend can transfer stack quantities in both directions.
- Backend can swap or merge compatible stacks where applicable.
- All transfer operations are transactional.
- Access validation cannot be bypassed by client-supplied IDs alone.
- Tests cover insufficient quantity, full target, invalid container, and conflict behavior.

## Verification

Run the relevant backend tests after implementation. Expected minimum:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/sunnytownbridge
go test ./internal/sunnytown/server
```

## Closeout Notes

When closing #29, include:

- implementation commit hash
- access validation flow used
- API/internal endpoint shape
- verification commands and results
- notes for #30 chest UI
