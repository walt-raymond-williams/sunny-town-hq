# Sunny Town Legacy Inventory Button Removal Handoff

## Purpose

This handoff starts GitHub issue #18, "Remove legacy button-based inventory assignment UI."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/18

The goal is to remove the old selected-item assignment buttons now that inventory-to-hotbar and inventory-to-equipment drag/drop paths exist.

## Current Status

The direct-manipulation inventory path is complete on `codex/inventory-redesign-dev`:

- #15 inventory grid slot-to-slot drag/drop is complete.
- #16 inventory-to-hotbar drop targets are complete.
- #17 inventory-to-equipment drop targets are complete.

This ticket should be a narrow cleanup slice. It should remove duplicate/debug-style assignment controls while preserving the new drop target flows and the useful clearing/unequipping controls.

## Product Direction

The inventory menu should now behave like a game UI rather than a debug panel:

- drag item stacks around the inventory grid
- drag items onto hotbar slots
- drag compatible items onto equipment slots
- keep clear and unequip affordances available
- stop showing redundant `Slot N` and `Wear` buttons in the selected item panel

The selected item panel can still show item name, description, icon, quantity, and useful state. It should no longer be the main assignment mechanism.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - selected item panel still includes:
    - `Wear` button emitted through `equipItem`
    - `Slot {{ selectedHotbarIndex + 1 }}` button emitted through `assignHotbar`
  - equipment slots are already rendered as drop targets
  - hotbar slots are already rendered as drop targets
  - `Clear` hotbar button should remain
  - `Unequip` equipment button should remain
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - may still wire `assignHotbar` and `equipItem` panel events for legacy controls
  - keep wiring needed for `equipInventorySlotDrop`, `clearHotbar`, `unequipSlot`, and equipment visual notifications
- `frontend/src/stores/studentInventory.ts`
  - keep store actions for hotbar/equipment if still used by drag/drop or parent action callbacks
  - do not remove backend APIs or store actions merely because the selected-item buttons are removed
- `frontend/src/stores/studentInventoryDragDrop.test.ts`
  - should continue passing
  - add or adjust frontend tests if existing tests assert legacy button presence

## Recommended Implementation Plan

1. Remove the selected-item `Wear` button from `SunnyTownInventoryPanel.vue`.
2. Remove the selected-item `Slot {{ selectedHotbarIndex + 1 }}` button from `SunnyTownInventoryPanel.vue`.
3. Keep selected item details readable:
   - icon
   - name
   - description
   - quantity
   - optional passive status such as equipped state if already present
4. Remove obsolete panel emits only if they are no longer used:
   - `assignHotbar`
   - `equipItem`
5. Remove obsolete parent handlers only if no remaining path uses them.
6. Keep these controls and flows:
   - `Clear` hotbar slot
   - equipment `Unequip`
   - inventory-to-inventory drag/drop
   - inventory-to-hotbar drop
   - inventory-to-equipment drop
   - crafting
   - loading and error alerts
7. Run tests and build.

## What Not To Remove

- Do not remove `setHotbarSlot` or hotbar API client functions; drop targets still use them.
- Do not remove `equipItem` store behavior if equipment drop or parent callbacks still depend on it.
- Do not remove `unequipItem`, `clearHotbar`, or their UI affordances.
- Do not remove equipment visual update behavior or `equipment_changed` notification paths.
- Do not redesign crafting in this ticket.
- Do not add character preview in this ticket.

## Suggested UX Semantics

- Clicking an inventory slot should still select it and show details.
- Assignment should happen through drag/drop surfaces, not selected-item buttons.
- The selected item details area should become calmer and informational.
- If there is no selected item, keep a concise empty state.

## Verification

Run:

```powershell
cd frontend
npm run test -- --run
npm run build
```

Manual smoke:

- open Sunny Town
- press `E`
- select an inventory item
- confirm there is no `Slot N` assignment button
- confirm there is no `Wear` assignment button
- drag an item to a hotbar slot
- drag a compatible item to an equipment slot
- clear a hotbar slot
- unequip an equipment item
- verify inventory slot-to-slot drag/drop still works
- verify crafting panel still opens and existing recipe actions still work

If full Sunny Town browser verification is blocked by login/session setup, record the blocker in the GitHub issue close comment and include deterministic checks that passed.

## Close Criteria

Close #18 when:

- the work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- selected item assignment buttons are gone
- direct manipulation paths remain intact

Use a close comment like:

```text
Completed in <commit>. Removed the legacy selected-item `Slot N` and `Wear` assignment buttons while preserving inventory, hotbar, and equipment drop target flows. Verified with: cd frontend; npm run test -- --run; npm run build. Manual Sunny Town smoke: <result or blocker>.
```
