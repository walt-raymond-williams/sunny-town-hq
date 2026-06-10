# Sunny Town Cookie Shop Input Replenishment Handoff

## Purpose

This handoff starts GitHub issue #33, "Add Cookie Shop input replenishment flow."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/33

The goal is to give gameplay a controlled way to add ingredients to HQ-owned Cookie Shop input storage without direct database edits.

## Current Status

- Issue #31 added durable Cookie Shop input storage with capacity.
- Issue #32 added ingredient-aware Cookie Keeper production.
- `cookie-keeper-shop` input storage is seeded with starter flour and sugar, but there is no replenishment flow yet.
- Work should continue on `codex/inventory-redesign-dev`.

## Product Direction

- The first useful player-facing path is deposit into `cookie-shop-input-chest`.
- Sunny Town should validate the live access condition, such as nearby/opened chest.
- HQ should own the durable mutation: consume from player inventory and increment shop input storage atomically.
- Keep Cookie Shop output chest withdraw/deposit separate from this slice.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownChestPanel.vue`
  - Current chest UI surface.
- `frontend/src/stores/studentInventory.ts`
  - Player inventory state and refresh behavior.
- `frontend/src/features/sunny-town`
  - Sunny Town interaction UI and menu surfaces.

Backend:

- `internal/hq/inventory`
  - Player inventory and shop input storage mutation logic.
- `internal/hq/sunnytownbridge`
  - Existing service-authenticated Sunny Town to HQ bridge patterns.
- `internal/sunnytown/server`
  - Live world validation for object access and interactions.
- `deploy/postgres/migrations`
  - Existing Cookie Shop storage schema and seed data.

Docs:

- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
  - Source of truth for Cookie Shop storage decisions.
- `docs/current/API.md`
  - Update if an API contract is added or changed.

## Recommended Implementation Plan

1. Confirm whether the narrow first slice is player deposit or an internal/debug replenishment route.
2. If player deposit is chosen, add Sunny Town validation for `cookie-shop-input-chest` access.
3. Add an HQ-owned transaction that decrements player inventory and increments shop input storage within capacity.
4. Expose the smallest frontend affordance needed to deposit eligible ingredients.
5. Refresh relevant player inventory and shop input/output state after deposit.
6. Update `docs/current/COOKIE_SHOP_STORAGE_PLAN.md` and `docs/current/API.md` if behavior/API changes.

## Out Of Scope

- Withdrawing from Cookie Shop output stock.
- General shop storage upgrade mechanics.
- Storing ingredient quantities in Sunny Town fixture metadata.
- Broad container permission redesign beyond what the deposit flow needs.

## Verification

Run:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/sunnytownbridge
go test ./internal/sunnytown/server
cd frontend
npm run build
```

Manual smoke:

- Deposit valid ingredients near `cookie-shop-input-chest`.
- Confirm over-capacity deposits fail without partial/incorrect mutation.
- Confirm Cookie Keeper production can consume replenished inputs.

## Close Criteria

Close #33 when:

- work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- current-state docs are updated for any API/schema/runtime changes
