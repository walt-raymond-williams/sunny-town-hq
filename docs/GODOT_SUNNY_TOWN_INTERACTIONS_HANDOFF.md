# Godot Sunny Town Interactions Handoff

## Purpose

This handoff starts GitHub issue #69, "Port Sunny Town interactions to Godot with Vue overlay bridge."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/69

The goal is to support current Sunny Town interactions while preserving Sunny Town/HQ authority.

## Current Status

- Blocked by #68.
- Rendering, desktop movement, and mobile controls should work before this starts.

## Product Direction

- Reuse the existing WebSocket protocol where practical.
- Complex HQ forms may remain Vue-owned during this epic.
- Godot should not become authoritative for rewards, inventory, wallet, ledgers, map objects, or container state.

## Current Code Surface

Frontend reference:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
- `frontend/src/composables/useSunnyTownNpcInteractions.ts`
- `frontend/src/composables/useSunnyTownChestInteractions.ts`
- `frontend/src/composables/useSunnyTownPlacement.ts`
- `frontend/src/composables/useSunnyTownMessageEffects.ts`
- `frontend/src/composables/useSunnyTownInventoryActions.ts`
- `frontend/src/features/sunny-town/SunnyTownShop.vue`
- `frontend/src/features/sunny-town/SunnyTownSchoolworkPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownChestPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownDialogue.vue`

Backend:

- `internal/sunnytown/server/client_gameplay.go`
- `internal/sunnytown/server/world_objects.go`
- `internal/sunnytown/server/server_workers.go`
- `internal/hq/sunnytownbridge/*`
- `internal/hq/inventory/*`

## Recommended Implementation Plan

1. Implement nearest NPC prompt logic in Godot.
2. Implement dialogue progression for dialogue-only NPCs.
3. Add a bridge event for shop and schoolwork flows if Vue retains those overlays.
4. Send `tool_use` for selected hotbar pickaxe.
5. Render resource hit/depleted feedback from snapshots.
6. Handle `resource_committed` and `resource_failed` feedback.
7. Implement stone-block placement mode:
   - grid preview,
   - validity feedback,
   - `place_object` message,
   - `map_object_placed`/`map_object_removed` handling.
8. Implement chest/container open and transfer path:
   - either Godot UI,
   - or Vue overlay bridge with Godot request trigger.
9. Preserve server validation for proximity, ownership, cooldown, storage roles, and durable mutations.

## Out Of Scope

- New interaction types.
- Client-authoritative inventory/wallet/resource updates.
- Full Godot replacement for every Vue overlay if a bridge is sufficient for first cutover.

## Verification

Run:

```powershell
go test ./...
cd frontend
npm run build
```

Manual smoke:

- Talk to Mayor Sunny.
- Use Cookie Keeper shop.
- Use Teacher schoolwork path.
- Mine a Forest Crossing rock node.
- Place a stone block.
- Mine/remove a placed stone block.
- Open Cookie Shop input and output chests.
- Confirm allowed and rejected container transfers match current behavior.
- Confirm wallet/inventory updates come from server/HQ commit messages.

## Close Criteria

Close #69 when:

- Current NPC, tool, resource, placement, and chest/container interactions work in Godot or through the approved Vue bridge.
- Authority boundaries remain unchanged.
- Verification is recorded.
