# Sunny Town Hotbar Drop Target Handoff

## Purpose

This handoff starts GitHub issue #16, "Replace hotbar assignment buttons with drop targets."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/16

The goal is to let players drag an item from the Sunny Town inventory grid onto a hotbar slot in the inventory menu, using the existing durable hotbar persistence API.

## Current Status

The inventory redesign foundation is complete on `codex/inventory-redesign-dev`:

- #8 item metadata is complete.
- #9 slotted inventory schema design is complete.
- #10 slotted inventory persistence is complete.
- #11 stack move/swap/merge API is complete.
- #12 reusable slot component is complete.
- #13 inventory grid rendering is complete.
- #15 inventory grid slot-to-slot drag/drop is complete.

This ticket should be the next direct-manipulation slice. Keep it scoped to inventory-to-hotbar assignment.

## Product Direction

The current inventory menu still has a debug-style hotbar assignment flow: select a hotbar slot, select an inventory item, then click `Slot N`.

The desired behavior is game-style:

- all hotbar slots are visible in the inventory menu
- a player drags an inventory item onto a hotbar slot
- the hotbar slot updates immediately after the server accepts the assignment
- number-key selection and gameplay use still work exactly as before

Legacy assignment controls can remain during this ticket if removing them would expand scope. The removal ticket comes later.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - renders the inventory grid
  - currently renders a selected-hotbar-slot editor with a clear button
  - still shows the old `Slot {{ selectedHotbarIndex + 1 }}` assignment button for the selected inventory item
  - wires inventory slot drag/drop from #15
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
  - reusable slot component
  - supports `dragStart`, `dragOver`, `dragLeave`, `drop`, `pending`, and `invalidDrop`
  - can render compact slots through `variant="compact"`
- `frontend/src/stores/studentInventory.ts`
  - owns `hotbarSlots`
  - has `setHotbarSlot(slot: number, itemKey: string)`
  - has #15 drag state:
    - `draggedInventorySlotIndex`
    - `pendingInventoryMoveSourceIndex`
    - `pendingInventoryMoveDestinationIndex`
    - `invalidInventoryDropSlotIndex`
    - `startInventorySlotDrag`
    - `dropInventorySlot`
    - `cancelInventorySlotDrag`
- `frontend/src/api/hotbarApi.ts`
  - `setStudentHotbarSlot(slot: number, itemKey: string)` calls `PUT /api/student/hotbar`
- `frontend/src/features/sunny-town/SunnyTownHud.vue`
  - renders the runtime hotbar
  - keep existing selected-slot behavior intact

Backend:

- Existing endpoint: `PUT /api/student/hotbar`
- Request body:

```json
{
  "slot": 1,
  "itemKey": "pickaxe"
}
```

Hotbar slots are one-based: `1` through `5`.

## Recommended Implementation Plan

1. Add store-level hotbar drop state if needed:
   - pending hotbar slot
   - invalid hotbar drop slot
   - source inventory slot for hotbar assignment can reuse `draggedInventorySlotIndex`
2. Add a store action such as `dropInventorySlotOnHotbar(hotbarSlot: number): Promise<boolean>`:
   - require an active dragged inventory slot
   - require the source slot to still contain an item
   - call `setHotbarSlot(hotbarSlot, item.key)`
   - sync hotbar quantities through existing `setHotbarSlot` behavior
   - clear drag/drop state afterward
3. Render all five hotbar slots inside `SunnyTownInventoryPanel.vue` as drop targets:
   - use `SunnyTownInventorySlot` with `variant="compact"` if it fits
   - show slot labels `1` through `5`
   - show pending/invalid states during drops
   - keep the existing clear affordance available
4. Wire drop handlers:
   - inventory slot drag starts should continue using #15 behavior
   - hotbar slots should accept drops from occupied inventory slots
   - hotbar slots should not start stack movement themselves in this ticket
5. Keep runtime hotbar behavior unchanged:
   - `SunnyTownHud.vue` should still show selected hotbar state
   - number-key selection should still use the selected hotbar slot
   - tool/block behavior should continue reading the selected hotbar item
6. Add or extend frontend tests:
   - successful inventory-to-hotbar drop calls `setStudentHotbarSlot`
   - empty/no-source drop is rejected before API call
   - failed hotbar API call leaves last confirmed hotbar state intact and clears pending state
   - inventory slot-to-slot move tests still pass

## Suggested UX Semantics

- Dragging an empty inventory slot should still be rejected by existing #15 behavior.
- Dropping onto a hotbar slot should assign the item type, not move the stack.
- Assigning to an occupied hotbar slot should replace that hotbar assignment.
- Clearing a hotbar slot should remain possible through the existing clear control.
- A pending hotbar assignment should visually affect the target hotbar slot.
- Invalid/no-source drops should show a short invalid-drop state and should not call the API.

## Out Of Scope

- Equipment drop targets.
- Removing all legacy `Slot N` assignment controls.
- Changing hotbar persistence to stack identity.
- Moving item quantities into or out of the hotbar.
- Chest/container transfer.
- Stack splitting.
- Character preview.

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
- drag an inventory item onto hotbar slot 1
- drag a different inventory item onto another hotbar slot
- replace an occupied hotbar slot by dropping another item onto it
- clear a hotbar slot
- use number keys to select slots
- verify selected hotbar item still drives tool/block behavior
- verify inventory slot-to-slot drag/drop still works

If full Sunny Town browser verification is blocked by login/session setup, record the blocker in the GitHub issue close comment and include deterministic checks that passed.

## Close Criteria

Close #16 when:

- the work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- legacy controls either remain intentionally or are only removed if the follow-up scope is explicitly expanded

Use a close comment like:

```text
Completed in <commit>. Inventory menu hotbar slots now accept inventory item drops and persist through PUT /api/student/hotbar. Verified with: cd frontend; npm run test -- --run; npm run build. Manual Sunny Town smoke: <result or blocker>.
```
