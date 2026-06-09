# Sunny Town Character Preview Handoff

## Purpose

This handoff starts GitHub issue #19, "Add character preview shell."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/19

The goal is to add a small character preview area to the Sunny Town inventory menu so players can see supported visual equipment changes while managing gear.

## Current Status

The inventory direct-manipulation flow is complete on `codex/inventory-redesign-dev`:

- #15 inventory grid slot-to-slot drag/drop is complete.
- #16 inventory-to-hotbar drop targets are complete.
- #17 inventory-to-equipment drop targets are complete.
- #18 legacy selected-item assignment buttons are removed.

The inventory menu now has stable equipment drop targets, so it is ready for a character preview shell.

## Product Direction

The preview should make equipment feel visible and personal:

- show the current player/avatar near equipment slots
- reflect supported gear visuals such as hoodie, cap, and selected/equipped tool where current art supports it
- leave room for a future stats panel
- avoid fake stats or manual point allocation
- keep equipment drag/drop and unequip easy to use

This is a shell, not a full character system.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - owns the inventory overlay layout
  - renders the equipment panel, inventory grid, selected item details, hotbar editor, and crafting panel
  - should receive the preview shell in or near the equipment area
- `frontend/src/features/sunny-town/rendering/characterDrawing.ts`
  - live Sunny Town player rendering
  - `drawPlayer` already uses:
    - `player.equipment?.gear`
    - `player.equipment?.accessory`
    - selected or equipped tool
  - supported visual keys include current hoodie/cap/pickaxe behavior
- `frontend/src/stores/studentInventory.ts`
  - getter `equippedVisuals` returns visual keys by equipment slot
  - equipment slots contain item metadata including `visualKey`
- `frontend/src/types/sunnyTown.ts`
  - `SunnyTownEquipment` shape is `{ gear?: string; accessory?: string; tool?: string }`
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - already sends `equipment_changed` and locally updates player equipment after equipment changes
  - this should not need backend changes

## Recommended Implementation Plan

1. Add a small frontend-only preview component, likely under:
   - `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
2. Feed it visual state from `inventoryStore.equippedVisuals`.
3. Render a compact player preview that matches the current Sunny Town visual language:
   - base body color
   - hoodie visual when `gear === 'sunny_hoodie'`
   - star cap visual when `accessory === 'star_cap'`
   - pickaxe/tool hint when `tool === 'pickaxe'`
4. Prefer reusing or extracting small drawing helpers from `characterDrawing.ts` if practical.
   - Do not make a broad renderer refactor unless it is genuinely smaller than duplicating a tiny preview.
   - If using a canvas, keep dimensions stable and verify nonblank rendering.
   - If using CSS/SVG-style markup, keep it local and consistent with existing colors.
5. Place the preview near equipment slots in `SunnyTownInventoryPanel.vue`.
6. Leave a reserved area or layout affordance for future stats, but do not invent stats.
7. Preserve:
   - equipment drop targets
   - unequip controls
   - inventory grid drag/drop
   - hotbar drop targets
   - crafting panel
   - selected item detail display
8. Add focused tests if the project has an obvious pattern for component/store tests.

## UX Guidance

- The preview should be visible at normal inventory menu sizes without taking over the whole panel.
- Keep it compact and operational, more like a game equipment panel than a marketing hero.
- Equipment slots should remain the active controls; the preview should be feedback, not a new interaction surface.
- Do not place a card inside another card.
- Avoid placeholder copy that explains future stats. Leave space visually if needed.

## Out Of Scope

- Stats implementation.
- Skills/progression model.
- Tabbed `E` menu shell.
- Crafting redesign.
- New backend state.
- New item art pipeline.
- Full avatar customization.
- Chest/container UI.

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
- verify the character preview is visible near equipment
- equip/unequip hoodie and confirm preview changes where supported
- equip/unequip cap and confirm preview changes where supported
- equip/unequip pickaxe and confirm preview shows a supported tool hint if implemented
- verify equipment drag/drop still works
- verify hotbar drop targets still work
- verify inventory slot-to-slot drag/drop still works
- check desktop and mobile-ish viewport sizes for overlap

If full Sunny Town browser verification is blocked by login/session setup, record the blocker in the GitHub issue close comment and include deterministic checks that passed.

## Close Criteria

Close #19 when:

- the work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- character preview is visible and reflects supported equipment visual keys
- no fake stats or unrelated menu redesign was added

Use a close comment like:

```text
Completed in <commit>. Inventory menu now includes a character preview shell near equipment slots and reflects supported equipped visual keys. Verified with: cd frontend; npm run test -- --run; npm run build. Manual Sunny Town smoke: <result or blocker>.
```
