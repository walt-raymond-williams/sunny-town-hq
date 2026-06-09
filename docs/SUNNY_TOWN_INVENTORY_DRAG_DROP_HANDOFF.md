# Sunny Town Inventory Drag And Drop Handoff

## Purpose

This handoff starts GitHub issue #15, "Add inventory drag/drop state and API integration."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/15

The goal is to make the existing Sunny Town inventory grid directly rearrange player inventory stacks by calling the durable HQ inventory move API.

## Current Status

The first inventory redesign slice is complete on `codex/inventory-redesign-dev`:

- #8 item metadata is complete.
- #9 slotted inventory schema design is complete.
- #10 slotted inventory persistence is complete.
- #11 stack move/swap/merge API is complete.
- #12 reusable slot component is complete.
- #13 inventory grid rendering is complete.

The next slice should stay focused on player inventory slot-to-slot drag/drop. Do not include hotbar drop targets, equipment drop targets, chest transfers, or legacy button removal in this ticket unless the issue scope is explicitly expanded.

## Relevant Product Direction

The final inventory experience should feel like Minecraft or Stardew Valley:

- inventory items appear in stable grid slots
- each item has an icon and stack quantity
- hover shows item details
- players drag items around the inventory
- later tickets let players drag items to hotbar and equipment slots
- later tickets add chest/container grids and storage-aware crafting

This ticket is the first direct-manipulation step.

## Current Code Surface

Frontend:

- `frontend/src/api/inventoryApi.ts`
  - already loads `GET /api/student/inventory/slots`
  - should add the move API client
- `frontend/src/stores/studentInventory.ts`
  - owns `inventorySlots`, aggregate `items`, hotbar, equipment, and crafting state
  - should expose drag/drop and move actions or a small composable-backed API
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - renders the inventory grid
  - currently selects slots by click
  - should wire drag/drop events to the store/composable
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
  - renders slot states including `pending` and `invalidDrop`
  - currently emits `click` and `select`
  - may need drag/drop props/events
- `frontend/src/types/inventory.ts`
  - contains `InventorySlot`, `InventoryItem`, and related response types

Backend contract already exists:

- `POST /api/student/inventory/move`
- request shape:

```json
{
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": { "kind": "player_inventory", "slotIndex": 5 },
  "mode": "auto"
}
```

Known supported modes from issue #11:

- `move`
- `swap`
- `merge`
- `auto`

The API returns an updated slotted inventory response. Keep frontend state aligned from that response instead of trying to permanently maintain optimistic state by hand.

## Recommended Implementation Plan

1. Add inventory move types and API client function in `frontend/src/api/inventoryApi.ts`.
2. Add store state for drag/drop:
   - source slot index
   - pending source/destination indexes
   - invalid drop target, if needed
   - a distinct update flag or reuse a clear inventory move flag
3. Add a store action for slot-to-slot moves:
   - reject empty-source drags client-side
   - reject same-slot drops client-side as a no-op
   - call `POST /api/student/inventory/move` with `mode: "auto"`
   - update `items`, `inventorySlotCount`, and `inventorySlots` from the returned response
   - run existing equipment/hotbar sync helpers afterward
   - on failure, set `error` and reload slots or leave the last confirmed state intact
4. Wire `SunnyTownInventorySlot.vue` for pointer/drag events without breaking click selection.
5. Wire `SunnyTownInventoryPanel.vue` so inventory slots can be dragged and dropped onto other inventory slots.
6. Use existing `pending` and `invalidDrop` visual states where practical.
7. Keep old hotbar/equipment buttons intact for now.

## Suggested UX Semantics

- Dragging an empty slot should not start a move.
- Dropping on the same slot should cancel/no-op.
- Dropping on an empty slot should move.
- Dropping on an occupied compatible stack should merge through server `auto` behavior.
- Dropping on an occupied incompatible stack should swap through server `auto` behavior.
- If the server rejects the operation, show the existing inventory error path and do not leave the UI in a half-moved state.
- During a pending move, prevent another slot move from starting.

## Out Of Scope

- Dragging from inventory to hotbar.
- Dragging from inventory to equipment.
- Removing legacy `Wear` or `Slot N` buttons.
- Stack splitting.
- Chest or container transfers.
- Storage-aware crafting.
- Character preview.

## Verification

Run:

```powershell
cd frontend
npm run build
npm run test -- --run
```

Manual smoke:

- open Sunny Town
- press `E`
- drag an occupied inventory slot to an empty slot
- drag two occupied slots to verify swap behavior
- drag compatible stacks to verify merge behavior when test data allows it
- verify failed/invalid drops do not corrupt inventory state
- verify click selection still works
- verify existing hotbar assignment, equipment buttons, and crafting still work

If full Sunny Town browser verification is blocked by login/session setup, record the blocker in the GitHub issue close comment and include the deterministic checks that passed.

## Close Criteria

Close #15 when:

- the work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- any durable architecture/API changes are reflected in `docs/current/` if needed

Use a close comment like:

```text
Completed in <commit>. Inventory grid slot-to-slot drag/drop now calls POST /api/student/inventory/move and refreshes slot state from the server response. Verified with: cd frontend; npm run build; npm run test -- --run. Manual Sunny Town smoke: <result or blocker>.
```
