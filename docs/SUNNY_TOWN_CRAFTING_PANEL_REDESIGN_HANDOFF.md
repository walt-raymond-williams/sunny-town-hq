# Sunny Town Crafting Panel Redesign Handoff

## Issue

GitHub issue #24: Redesign crafting panel for inventory menu

https://github.com/walt-raymond-williams/sunny-town-hq/issues/24

## Branch

Work against the inventory redesign integration branch:

`codex/inventory-redesign-dev`

## Goal

Redesign the Sunny Town E-menu crafting panel so it uses the same visual item language as the inventory grid.

Recipes should be easy to scan: output icon, output quantity, recipe name, ingredient requirements, owned counts, missing states, and a clear craft action. The panel should feel like a natural part of the inventory screen, not a debug list.

## Product Rationale

The inventory redesign now has the main interaction foundation in place: item metadata, slotted persistence, grid rendering, drag/drop, hotbar/equipment drop targets, expanded E-menu space, character preview work, and a slotted-inventory crafting correctness fix.

Crafting is the next player-facing surface that still needs to catch up visually. This slice should make the current player-inventory crafting experience coherent before later storage-aware crafting, chests, workstations, and shop inputs expand the system.

## Dependencies and Related Work

Completed blockers:

- #12 Build reusable inventory slot component
- #13 Render Sunny Town inventory as a grid
- #23 Fix stone block crafting failure after slotted inventory migration

Related future work:

- Add Storage Context To Crafting Recipe APIs
- Add Cookie Shop Input Storage
- Container/chest inventory transfer

This ticket should not implement storage-aware crafting. It should leave room in the UI for source/destination context, but only player inventory needs to work in this slice.

## Likely Code Areas

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/stores/craftingRecipes.ts`
- `frontend/src/api/craftingApi.ts`
- `frontend/src/types/inventory.ts`
- `frontend/src/style.css`

## Scope

In scope:

- Redesign recipe rows/cards using output icons and quantities.
- Render ingredient icons/counts with required and owned quantities.
- Add a clear missing-ingredient visual state.
- Disable craft action when requirements are missing or crafting is already in progress.
- Preserve successful craft updates for inventory slots, aggregate items, hotbar quantities, and recipe availability.
- Reserve compact UI space for ingredient source and output destination, even if it only says/represents player inventory for now.
- Keep loading, empty, and error states readable inside the expanded E menu.
- Remove the old fallback recipe injection if server recipes are now reliable enough after #23, or document why it remains.

Out of scope:

- Storage-aware crafting API changes.
- Chest/container crafting.
- Cookie Shop input storage.
- New recipes.
- Recipe scaling/bulk crafting.
- Changing the `stone_block` recipe cost or backend behavior.
- Broad E-menu layout rewrites beyond what the crafting panel needs.

## Acceptance Criteria

- Crafting recipes show output icon, output quantity, and recipe name.
- Ingredients show required and owned quantities.
- Missing ingredients have a clear visual missing state.
- Craft button/action is disabled when requirements are missing.
- Craft button/action is disabled or visibly pending while crafting is in progress.
- Craft result updates inventory slots, item quantities, hotbar quantities, and recipe availability.
- The UI includes a compact reserved source/output destination area for future storage-aware crafting.
- The panel fits inside the expanded E menu without overlapping the inventory grid, hotbar, character preview, stats-ready area, or equipment rail.
- Desktop and narrow/mobile layouts remain usable.

## Suggested Implementation Approach

1. Inspect the current E-menu layout after #20, #22, and #23 before changing CSS. The character panel and equipment rail have moved since the older roadmap notes.
2. Start in `SunnyTownInventoryPanel.vue` and identify the current crafting markup.
3. Reuse `SunnyTownInventorySlot.vue` or the same item-icon CSS classes for recipe output and ingredients where practical.
4. Keep recipe controls compact and operational. Avoid explanatory copy inside the app UI.
5. Preserve store behavior from #23. The craft result should continue using authoritative slotted inventory state.
6. If touching `craftingRecipes.ts`, decide whether fallback recipe injection is still needed. Prefer removing it if server recipes are reliable, but do not break offline/test expectations without updating tests.
7. Add/update focused frontend tests if selectors, store behavior, or recipe fallback behavior changes.

## Verification

Run:

```powershell
cd frontend
npm run build
```

If tests are touched or behavior is changed in the store/API layer, run:

```powershell
cd frontend
npm run test -- --run
```

Manual smoke:

1. Open Sunny Town and press `E`.
2. Confirm crafting recipes show output icon, quantity, recipe name, and ingredient counts.
3. Confirm missing ingredients show a missing state and the craft action is disabled.
4. Give or obtain at least 4 rocks.
5. Craft `stone_block`.
6. Confirm rocks decrease by 4.
7. Confirm stone block quantity increases by 1.
8. Confirm the inventory grid updates without losing the intended slotted layout.
9. Confirm hotbar quantities sync if stone block or rock are visible in the hotbar.
10. Check desktop and narrow/mobile widths for overlap or clipped controls.

## Closeout Notes

When closing #24, include:

- implementation commit hash
- whether fallback recipe injection was removed or retained
- verification commands and results
- manual smoke result if performed
