# Sunny Town Hotbar Active Tool Handoff

## Purpose

This handoff starts GitHub issue #54, "Use selected hotbar item as active Sunny Town tool."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/54

The goal is to fix the mining regression where a player can select the pickaxe in the 5-slot Sunny Town hotbar but still receive `tool_not_equipped`, because server validation still depends on the separate character-card `tool` equipment slot.

## Current Status

- #54 is `status:ready`.
- The issue was created after a live regression report: pickaxe no longer mines natural rocks or placed stone blocks when selected from the hotbar.
- Work should happen on the active feature branch unless the branch strategy changes before implementation.

## Product Direction

- The selected Sunny Town hotbar item is the active moment-to-moment usable item.
- The character card should not expose a separate `tool` equipment slot.
- Character equipment can remain for worn gear/accessory visuals or later stats, but tools used in the world come from hotbar selection.
- Client input remains a request, not authority. Server validation must still prove the player owns or has access to the requested tool.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Sends Sunny Town tool/placement requests and reads selected hotbar state.
- `frontend/src/composables/useSunnyTownInventoryActions.ts`
  - Hotbar selection and item action behavior.
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - Character card/equipment UI surface.
- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
  - Character preview currently renders equipped visuals/tools.
- `frontend/src/stores/studentInventory.ts`
  - Inventory, equipment, and hotbar state derivation.

Backend:

- `internal/sunnytown/server/client_gameplay.go`
  - `tool_use` validation and `tool_not_equipped` error path.
- `internal/sunnytown/server/room_mining_test.go`
  - Mining tests for rocks, placed stone blocks, tool validation, and resource events.
- `internal/sunnytown/server/world_snapshots.go`
  - Inventory/equipment snapshot compatibility for Sunny Town players.

Docs:

- `docs/current/API.md`
- `docs/current/NPC_CHARACTER_MODEL_PLAN.md`
- Any current Sunny Town inventory/runtime doc that describes active item, hotbar, equipment, or mining authority.

## Expected Behavior

- Selecting `pickaxe` in the 5-slot hotbar is enough to mine:
  - natural rock nodes
  - placed stone blocks
- Selecting `stone_block` in the hotbar continues to drive stone-block placement.
- A missing selected pickaxe should produce an error/state that matches the hotbar model, not the removed character-card `tool` slot.
- Mining still fails if the player does not own or otherwise have validated access to a pickaxe.
- Existing gear/accessory equipment behavior remains intact unless a narrow UI cleanup requires adjustment.

## Recommended Implementation Plan

1. Inspect the current hotbar selection flow and the `tool_use` message sender.
2. Inspect the server `tool_not_equipped` validation path in `client_gameplay.go`.
3. Replace mining authority from `equipment.tool` with validated hotbar/inventory ownership for the requested `ToolKey`.
4. Remove the visible `tool` equipment slot/section from the Sunny Town character card.
5. Keep gear/accessory equipment and character preview behavior stable.
6. Update frontend tests for hotbar-selected pickaxe behavior and character-card tool slot removal.
7. Update server tests so mining succeeds with an owned/selected pickaxe even when `equipment.tool` is empty.
8. Add/keep server tests proving spoofed `ToolKey: "pickaxe"` fails without owned/access-valid pickaxe.
9. Update current docs to state that hotbar selection is the active usable item model.

## Out Of Scope

- Full equipment schema redesign.
- Removing durable `student_equipped_item` support for tools from HQ APIs unless required by this slice.
- Character-stat or skill requirements for tools.
- New hotbar UI design beyond what is needed to make active selection clear.
- Changing stone block crafting or Cookie Shop storage behavior.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
cd frontend
npm run build
```

If practical, also run:

```powershell
go test ./...
```

Manual smoke if practical:

- Put pickaxe in the hotbar.
- Select the pickaxe slot.
- Mine a natural rock node.
- Mine a placed stone block.
- Confirm no separate character-card tool slot is visible or needed.
- Select stone block in the hotbar and confirm placement still works.

## Close Criteria

Close #54 when:

- The hotbar-selected pickaxe mines rocks and placed stone blocks without a character-card tool slot.
- The character-card `tool` section/slot is removed from the Sunny Town UI.
- Server validation still rejects unowned/spoofed pickaxe use.
- Tests cover the new hotbar-active-tool behavior.
- Current docs reflect the hotbar active item rule.
- Work is committed and pushed.
- The issue close comment records commit hash and verification results.
