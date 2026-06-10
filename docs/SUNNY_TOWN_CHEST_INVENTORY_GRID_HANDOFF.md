# Sunny Town Chest Inventory Grid Handoff

## Issue

GitHub issue #30: Render chest inventory grid

https://github.com/walt-raymond-williams/sunny-town-hq/issues/30

## Branch

Work against:

`codex/inventory-redesign-dev`

## Status

This issue is currently blocked.

Do not start implementation until #29 is closed and the backend can load and transfer container contents.

## Goal

Render an opened chest/container as a grid beside the player inventory grid, with drag/drop transfers between player and container storage.

## Product Rationale

The final inventory direction includes container-aware storage. Players should be able to open a chest, see both inventories, and move items visually instead of using buttons or a read-only panel.

## Dependencies and Related Work

Blocked by:

- #12 Build reusable inventory slot component, closed
- #29 Add player to container transfer operations

Related:

- #13 Render Sunny Town inventory as a grid
- #24 Redesign crafting panel for inventory menu
- #26 Design general container storage schema and access contract

## Likely Code Areas

- `frontend/src/features/sunny-town/SunnyTownChestPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/api/*`
- `frontend/src/composables/*`
- `frontend/src/types/*`
- `frontend/src/style.css`

## Scope

In scope:

- Show player grid and container grid together when a supported chest is open.
- Support drag/drop transfer between grids.
- Clearly label player storage and chest storage.
- Match existing empty, occupied, hover, tooltip, pending, and invalid-drop states.
- Close the container view when moving away or changing maps.
- Keep desktop and narrow/mobile layouts usable.

Out of scope:

- Backend transfer implementation.
- Storage-aware crafting.
- Cookie Shop input/production behavior.
- New chest art or broad map changes unless required for smoke testing.

## Acceptance Criteria

- Opening a supported chest shows both player and container grids.
- Drag/drop transfers items both directions.
- Player and chest storage are clearly distinguished.
- Slot behavior matches player inventory slot behavior.
- Moving away or changing maps closes the container view.
- UI remains usable without overlap at desktop and narrow/mobile widths.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual smoke:

1. Open Sunny Town.
2. Open a supported chest.
3. Transfer an item from player inventory to chest.
4. Transfer an item from chest to player inventory.
5. Move away and confirm the container view closes.

## Closeout Notes

When closing #30, include:

- implementation commit hash
- backend API assumptions used
- verification commands and results
- manual transfer smoke result if performed
