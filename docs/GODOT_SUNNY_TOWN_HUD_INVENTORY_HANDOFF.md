# Godot Sunny Town HUD Inventory Handoff

## Purpose

This handoff starts GitHub issue #70, "Add Godot HUD, hotbar, inventory summary, and equipment visuals."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/70

The goal is to add enough in-game UI parity for Godot cutover.

## Current Status

- Blocked by #69.
- Core interactions should work before this starts.

## Product Direction

- Godot owns the in-game HUD, selected hotbar item, mobile controls, prompts, toasts, and basic game feedback.
- Vue continues to own broader HQ/student/teacher screens.
- Vue may still own complex overlays if the bridge is retained for this epic.

## Current Code Surface

Frontend reference:

- `frontend/src/features/sunny-town/SunnyTownHud.vue`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownInventorySlot.vue`
- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/api/hotbarApi.ts`
- `frontend/src/api/equipmentApi.ts`
- `frontend/src/api/inventoryApi.ts`
- `frontend/src/api/characterProgressionApi.ts`

Protocol:

- `internal/sunnytown/protocol/protocol.go`
- player `equipment` snapshots
- reward/resource/object/container result messages

## Recommended Implementation Plan

1. Add Godot HUD elements for:
   - connection/status,
   - player count,
   - star balance,
   - selected hotbar item,
   - interaction prompt,
   - toast,
   - error.
2. Feed initial inventory/hotbar/wallet state from the Vue-created session.
3. Update wallet/inventory quantities from existing commit messages and bridge responses.
4. Render equipment visual keys from player snapshots.
5. Decide what minimal inventory summary or overlay Godot needs before cutover.
6. Keep broader HQ screens and any intentionally retained Vue overlays outside Godot.
7. Document any deferred UI parity work.

## Out Of Scope

- Full polished RPG UI.
- New item/equipment systems.
- Replacing all Vue drag/drop inventory behavior unless this issue explicitly needs it.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual smoke:

- Select hotbar slots with keyboard.
- Select hotbar slots with touch.
- Mine, place, remove, collect stars, shop, and use chests.
- Confirm toasts/errors and quantities update.
- Open two clients and confirm remote equipment visuals.

## Close Criteria

Close #70 when:

- Godot HUD has sufficient parity for current play.
- Hotbar selection drives existing tool/place behavior.
- Reward/resource/object/container feedback is visible.
- Equipment visuals render from snapshots.
- Vue/Godot UI ownership is documented.
