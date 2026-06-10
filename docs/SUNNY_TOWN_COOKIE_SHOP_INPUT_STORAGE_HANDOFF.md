# Sunny Town Cookie Shop Input Storage Handoff

## Issue

GitHub issue #31: Add Cookie Shop input storage

https://github.com/walt-raymond-williams/sunny-town-hq/issues/31

## Branch

Work against:

`codex/inventory-redesign-dev`

## Status

This issue is currently blocked.

Do not start implementation until #25 is closed and the accepted storage-context crafting shape is known.

## Goal

Add HQ-owned durable input storage for Cookie Shop ingredients such as flour and sugar.

This creates the input side needed before NPC/shop production can consume ingredients through the shared recipe executor.

## Dependencies and Related Work

Blocked by:

- #25 Add storage context to crafting recipe APIs

Related:

- #32 Add ingredient-aware Cookie Shop production
- #29 Add player to container transfer operations
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`

## Likely Code Areas

- `internal/hq/inventory`
- `internal/hq/schema`
- `internal/hq/sunnytownbridge`
- `deploy/postgres/migrations`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
- `docs/current/DATABASE.md`
- `docs/current/API.md`

## Scope

In scope:

- Persist input stock keyed by shop/storage identity.
- Add storage operations that can consume input stock transactionally.
- Define or defer capacity behavior explicitly.
- Keep existing Cookie Shop output stock and purchases unchanged.
- Keep quantities out of fixture metadata and Sunny Town client state.
- Add migration/schema tests.
- Update current-state docs.

Out of scope:

- Ingredient-aware production consumption.
- Player-to-container transfers.
- Chest UI.
- New shop economy balancing.

## Acceptance Criteria

- HQ persists Cookie Shop input stock by durable storage identity.
- Input stock can be consumed transactionally by storage operations.
- Capacity behavior is implemented or explicitly deferred in docs.
- Existing output stock and purchases keep working.
- No input quantities are stored in fixture metadata or client state.
- Tests cover persistence and schema behavior.

## Verification

Run:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/schema
```

Run broader tests if sunnytownbridge paths change:

```powershell
go test ./internal/hq/sunnytownbridge
```

## Closeout Notes

When closing #31, include:

- implementation commit hash
- migration name
- input storage identity shape
- verification commands and results
- notes for #32 production consumption
