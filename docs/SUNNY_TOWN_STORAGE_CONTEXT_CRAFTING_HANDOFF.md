# Sunny Town Storage Context Crafting Handoff

## Issue

GitHub issue #25: Add storage context to crafting recipe APIs

https://github.com/walt-raymond-williams/sunny-town-hq/issues/25

## Branch

Work against:

`codex/inventory-redesign-dev`

## Goal

Make HQ crafting recipe availability and execution aware of explicit storage context.

The current player crafting path should keep working, but the recipe system needs a path toward consuming from player inventory, shop input storage, chests, and future workstation storage without duplicating recipe rules.

## Product Rationale

Crafting is becoming a shared economy primitive. Player stone-block crafting, Cookie Shop production, future chest-based crafting, and NPC work should use the same recipe definitions and execution semantics.

This ticket creates the API/domain foundation that later tickets use for Cookie Shop input storage, ingredient-aware production, and container-aware crafting.

## Dependencies and Related Work

Completed blocker:

- #11 Add inventory stack move, swap, and merge API

Related follow-ups:

- #31 Add Cookie Shop input storage
- #32 Add ingredient-aware Cookie Shop production
- #29 Add player to container transfer operations
- #24 Redesign crafting panel for inventory menu

## Likely Code Areas

- `internal/hq/inventory/crafting.go`
- `internal/hq/inventory/http.go`
- `internal/hq/inventory/store_integration_test.go`
- `docs/current/API.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`

## Scope

In scope:

- Add or design storage-context descriptors for recipe availability and execution.
- Keep player inventory crafting behavior compatible.
- Keep the storage-agnostic recipe executor shared.
- Scope missing ingredient errors to the selected storage context.
- Add tests for player storage context and at least one fake or minimal non-player storage context.
- Update current-state docs if API/domain behavior changes.

Out of scope:

- Cookie Shop input storage implementation.
- Ingredient-aware Cookie Shop production.
- Chest transfer APIs.
- Frontend crafting UI redesign.
- New recipes beyond what is needed for tests.

## Acceptance Criteria

- Recipe availability can be calculated for a storage context.
- Recipe execution accepts explicit input/output storage descriptors or has a documented first-slice equivalent.
- Existing player inventory crafting still works.
- Missing ingredient errors identify the selected storage context behavior.
- The recipe catalog/executor remains shared instead of branching into Cookie Shop-specific logic.
- Tests cover player inventory crafting through the new path.
- Tests cover a non-player or fake storage context enough to prove the abstraction.

## Verification

Run:

```powershell
go test ./internal/hq/inventory
```

If API docs change, review:

```powershell
git diff -- docs/current/API.md docs/current/ARCHITECTURE.md docs/current/COOKIE_SHOP_STORAGE_PLAN.md
```

## Closeout Notes

When closing #25, include:

- implementation commit hash
- storage descriptor shape chosen
- whether public API behavior changed
- verification commands and results
- follow-up notes for #31, #32, and #29
