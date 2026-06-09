# Sunny Town Equipment Drop Target Handoff

## Purpose

This handoff starts GitHub issue #17, "Replace equipment buttons with drop targets."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/17

The goal is to let players drag a compatible item from the Sunny Town inventory grid onto an equipment slot in the inventory menu, using the existing durable equipment API.

## Current Status

The direct-manipulation foundation is now in place on `codex/inventory-redesign-dev`:

- #8 item metadata is complete.
- #9 slotted inventory schema design is complete.
- #10 slotted inventory persistence is complete.
- #11 stack move/swap/merge API is complete.
- #12 reusable slot component is complete.
- #13 inventory grid rendering is complete.
- #15 inventory grid slot-to-slot drag/drop is complete.
- #16 inventory-to-hotbar drop targets are complete.

This ticket should be the next direct-manipulation slice. Keep it scoped to inventory-to-equipment assignment.

## Product Direction

The current inventory menu still has a debug-style equipment flow: select an inventory item and click `Wear`, then use `Unequip` from the equipment list.

The desired behavior is game-style:

- gear, accessory, and tool slots are visible equipment drop targets
- a player drags compatible inventory gear onto a matching equipment slot
- incompatible drops get immediate invalid feedback where the client can detect the mismatch
- server validation remains authoritative
- other players still see equipment visual updates

Legacy `Wear` controls can remain during this ticket if removing them would expand scope. The removal ticket comes later.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - renders the inventory grid, equipment section, selected item actions, and hotbar editor
  - currently renders equipment as text rows with `Unequip` buttons
  - still shows the old selected-item `Wear` button
  - wires inventory slot drag/drop from #15
  - wires hotbar drop targets from #16
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
  - reusable slot component
  - supports `dragStart`, `dragOver`, `dragLeave`, `drop`, `pending`, and `invalidDrop`
  - supports `draggable-enabled="false"` for drop-only slots
  - can render compact slots through `variant="compact"`
- `frontend/src/stores/studentInventory.ts`
  - owns `equipmentSlots`
  - has `equipItem(slot: EquipmentSlot, itemKey: string)`
  - has `unequipItem(slot: EquipmentSlot)`
  - has #15/#16 drag state:
    - `draggedInventorySlotIndex`
    - `invalidInventoryDropSlotIndex`
    - `pendingHotbarDropSlot`
    - `invalidHotbarDropSlot`
    - `startInventorySlotDrag`
    - `dropInventorySlot`
    - `dropInventorySlotOnHotbar`
- `frontend/src/api/equipmentApi.ts`
  - `equipStudentItem(slot: EquipmentSlot, itemKey: string)` calls `POST /api/student/equipment/equip`
  - `unequipStudentItem(slot: EquipmentSlot)` calls `POST /api/student/equipment/unequip`
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - receives `equipItem` / `unequipSlot` events from the inventory panel
  - current equipment changes should still send `equipment_changed` so Sunny Town visuals update

Backend:

- Existing endpoint: `POST /api/student/equipment/equip`
- Request body:

```json
{
  "slot": "gear",
  "itemKey": "sunny_hoodie"
}
```

- Existing endpoint: `POST /api/student/equipment/unequip`
- Request body:

```json
{
  "slot": "gear"
}
```

Equipment slots are:

- `gear`
- `accessory`
- `tool`

Inventory item metadata includes `equipSlot`. Use it for client-side compatibility hints, but keep server validation authoritative.

## Recommended Implementation Plan

1. Add store-level equipment drop state:
   - pending equipment slot
   - invalid equipment drop slot
   - source inventory slot can reuse `draggedInventorySlotIndex`
2. Add a store action such as `dropInventorySlotOnEquipment(slot: EquipmentSlot): Promise<boolean>`:
   - require an active dragged inventory slot
   - require the source slot to still contain an item
   - reject obvious client-side mismatches where `item.equipSlot` is present and not equal to the target slot
   - call `equipItem(slot, item.key)`
   - clear drag/drop state afterward
3. Render equipment slots in `SunnyTownInventoryPanel.vue` as drop targets:
   - use `SunnyTownInventorySlot` if practical
   - show labels for `gear`, `accessory`, and `tool`
   - show existing equipped item icon/quantity where present
   - show pending/invalid states during drops
   - keep `Unequip` available
4. Wire drop handlers:
   - inventory slot drag starts should continue using #15 behavior
   - equipment slots should accept drops only from occupied inventory slots
   - equipment slots should not start stack movement themselves in this ticket
5. Preserve visual update behavior:
   - do not bypass existing `equipItem` event flow if `SunnyTownPage.vue` owns websocket notification
   - if the store action directly equips, make sure the caller still triggers the existing `equipment_changed` behavior
   - prefer a narrow panel event if that better preserves current architecture
6. Add or extend frontend tests:
   - successful compatible inventory-to-equipment drop calls equipment API path or emits the existing equip event
   - incompatible drop is rejected before API call when `equipSlot` mismatch is known
   - failed equipment API call clears pending state and preserves last confirmed equipment
   - hotbar drop-target tests and inventory slot-to-slot tests still pass

## Suggested UX Semantics

- Dragging an empty inventory slot should remain rejected by existing #15 behavior.
- Dropping onto the matching equipment slot should equip the item.
- Dropping onto the wrong equipment slot should show invalid feedback.
- Dropping a non-equippable item should show invalid feedback.
- Unequipping should remain possible through the current `Unequip` affordance.
- Pending equipment assignment should visually affect the target equipment slot.

## Out Of Scope

- Removing all legacy `Wear` controls.
- Removing hotbar legacy assignment controls.
- Character/avatar preview.
- Changing equipment persistence to stack identity.
- Consuming inventory quantity when equipping.
- Chest/container transfer.
- Stack splitting.

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
- drag a gear item onto the gear slot
- drag a tool item onto the tool slot
- attempt an incompatible drop such as a resource onto a gear/tool slot
- unequip an equipped item
- verify the visible player equipment updates locally
- verify `equipment_changed` behavior still updates Sunny Town visuals
- verify inventory slot-to-slot drag/drop still works
- verify hotbar drop targets still work

If full Sunny Town browser verification is blocked by login/session setup, record the blocker in the GitHub issue close comment and include deterministic checks that passed.

## Close Criteria

Close #17 when:

- the work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- equipment visual update behavior is preserved or the remaining manual smoke blocker is documented
- legacy controls either remain intentionally or are only removed if the follow-up scope is explicitly expanded

Use a close comment like:

```text
Completed in <commit>. Equipment slots now accept compatible inventory item drops and persist through POST /api/student/equipment/equip while preserving equipment_changed behavior. Verified with: cd frontend; npm run test -- --run; npm run build. Manual Sunny Town smoke: <result or blocker>.
```
