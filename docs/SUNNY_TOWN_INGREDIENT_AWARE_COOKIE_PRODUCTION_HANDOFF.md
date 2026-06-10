# Sunny Town Ingredient-Aware Cookie Production Handoff

## Issue

GitHub issue #32: Add ingredient-aware Cookie Shop production

https://github.com/walt-raymond-williams/sunny-town-hq/issues/32

## Branch

Work against:

`codex/inventory-redesign-dev`

## Status

This issue is currently blocked.

Do not start implementation until #25 and #31 are closed.

## Goal

Make Cookie Shop production consume required ingredients and produce output stock in one HQ-owned transaction.

Missing ingredients should block output stock changes without breaking production idempotency or causing duplicate output on retries.

## Product Rationale

Cookie Shop production currently behaves like free stock generation. Ingredient-aware production moves the shop economy toward a real simulation where NPC work depends on durable input storage and shared recipe rules.

## Dependencies and Related Work

Blocked by:

- #25 Add storage context to crafting recipe APIs
- #31 Add Cookie Shop input storage

Related:

- #24 Redesign crafting panel for inventory menu
- #28 Define stats and skills progression model
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`

## Likely Code Areas

- `internal/sunnytown/server/npc_production.go`
- `internal/hq/sunnytownbridge/store.go`
- `internal/hq/inventory/crafting.go`
- `internal/hq/inventory`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
- `docs/current/ARCHITECTURE.md`

## Scope

In scope:

- Consume Cookie Shop recipe inputs through HQ-owned storage operations.
- Produce output stock in the same HQ transaction.
- Preserve production ledger idempotency.
- Return/debug blocked production when ingredients are missing.
- Keep Sunny Town from deciding ingredient availability.
- Add backend tests across HQ inventory/sunnytownbridge and Sunny Town production paths.

Out of scope:

- Adding UI for stocking ingredients.
- Player-to-container transfer implementation.
- Stats/skills requirements.
- New production recipes beyond what this ticket needs.

## Acceptance Criteria

- Cookie production consumes required inputs and produces output stock in one HQ transaction.
- Missing ingredients block output stock changes.
- Production ledger behavior stays idempotent.
- Blocked production has a debuggable result without double-producing on retries.
- Recipe definitions live in HQ/shared recipe catalog instead of Cookie Keeper-specific conditionals.
- Sunny Town does not decide ingredient availability.

## Verification

Run:

```powershell
go test ./internal/hq/sunnytownbridge
go test ./internal/hq/inventory
go test ./internal/sunnytown/server
```

Run broader backend tests if shared inventory/production interfaces change:

```powershell
go test ./...
```

## Closeout Notes

When closing #32, include:

- implementation commit hash
- recipe/storage path used
- idempotency behavior
- missing-ingredient behavior
- verification commands and results
