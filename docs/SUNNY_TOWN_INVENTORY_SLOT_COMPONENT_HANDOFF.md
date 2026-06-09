# Sunny Town Inventory Slot Component Handoff

## Purpose

Use this handoff to start a Codex agent on the next inventory redesign task:

GitHub issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/12

Issue title:

`Build reusable inventory slot component`

Work branch:

`codex/inventory-redesign-dev`

## Task Contract

Treat GitHub issue #12 as the active task contract.

Goal:

Build a reusable frontend inventory slot component that can become the visual foundation for player inventory, hotbar slots, equipment slots, crafting ingredients, and future container/chest grids.

This task should create the component foundation only. It should not rewrite the full Sunny Town inventory panel or implement drag/drop yet.

User story:

As a player, I want each item to appear in a compact slot with an icon and quantity, so that I can scan my belongings quickly.

## Read First

- `AGENTS.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TICKETS.md`
- `docs/INVENTORY_AND_EQUIPMENT.md`
- GitHub issue #12
- GitHub issue #8, for metadata context

## Current State Summary

- Issue #8 added item metadata foundation:
  - `iconKey`
  - `maxStack`
  - `category`
  - `visualKey` remains separate for avatar/equipment rendering.
- The current Sunny Town inventory overlay still uses item cards and buttons in `SunnyTownInventoryPanel.vue`.
- The current hotbar already renders compact slots in `SunnyTownHud.vue`.
- Existing item icon visuals are CSS classes in `frontend/src/style.css`, using classes such as `inventory-item__icon--cookie`.
- The frontend inventory model currently exposes item-level quantity data, not slotted inventory data.
- Slotted inventory persistence is not implemented yet. Do not block this component on that backend work.

## Files To Inspect

Frontend:

- `frontend/src/types/inventory.ts`
- `frontend/src/api/inventoryApi.ts`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownHud.vue`
- `frontend/src/features/sunny-town/SunnyTownChestPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownShop.vue`
- `frontend/src/style.css`

Tests:

- `frontend/src/stores/studentInventory.test.ts`
- `frontend/src/composables/useSunnyTownInventoryActions.test.ts`
- Any existing frontend component test pattern available in `frontend/src`.

## Acceptance Criteria

- Slot component renders empty, occupied, selected, hover, disabled, pending, and invalid-drop states.
- Occupied slots show item icon and stack quantity.
- Tooltip shows item name and description.
- Component has stable dimensions and does not resize based on content.
- Component can be reused for player inventory, hotbar, equipment slots, crafting ingredients, and containers.

## Recommended Component Shape

Prefer a display-focused component with a stable prop API. Example direction:

- Component name: `SunnyTownInventorySlot.vue` or `InventorySlot.vue`.
- Location: likely `frontend/src/features/sunny-town/`, unless the agent introduces a small shared inventory component folder.
- Props should support:
  - item data or null
  - quantity
  - slot label/index
  - selected
  - disabled
  - pending
  - invalidDrop
  - compact or variant mode, if needed
  - tooltip enable/disable if useful
- Emits may support click/clear/select if needed, but drag/drop behavior should remain out of scope.
- Use existing icon CSS through item `iconKey` where available, falling back to item `key`.
- Keep dimensions stable with CSS, not content-driven.

If adapting current hotbar markup is practical, use the existing hotbar visual language as a reference. Do not break number-key hotbar selection behavior.

## Out Of Scope

- No slotted inventory backend work.
- No inventory grid rewrite.
- No drag/drop behavior.
- No hotbar assignment redesign.
- No equipment assignment redesign.
- No chest/container transfer UI.
- No crafting panel redesign.
- No schema/API changes unless a small frontend type compatibility fix is required.

## Implementation Guidance

- Keep the component reusable and presentation-oriented.
- Avoid coupling it directly to Pinia store state.
- Avoid hard-coding it only for the Sunny Town player inventory list.
- Prefer item metadata fields from #8:
  - `iconKey` for inventory icon class/asset selection.
  - `name` and `description` for tooltip.
  - `quantity` for stack count.
- Do not use `visualKey` as the inventory icon unless no better fallback exists.
- If this task needs a small adapter type, define it clearly in `frontend/src/types/inventory.ts` or locally in the component file.
- Keep UI text compact. This is an app/game interface, not an explanatory page.

## Verification

Run:

```powershell
cd frontend
npm run build
```

If adding component tests or changing existing frontend test-covered behavior, also run:

```powershell
cd frontend
npm run test
```

If the agent unexpectedly changes backend or shared API behavior, also run the relevant backend tests.

## GitHub Workflow

Before implementation:

1. Open issue #12.
2. Change the issue label from `status:ready` to `status:in-progress`.
3. Work on branch `codex/inventory-redesign-dev` unless the user asks for a smaller task branch.

During implementation:

- Work from issue #12.
- Keep comments for durable discoveries, blockers, scope changes, and verification results.
- If the component requires broader grid or drag/drop work, create or reference a follow-up issue instead of expanding #12.

Closeout:

- Close #12 only after the component lands on `codex/inventory-redesign-dev`.
- Closing comment should include the commit hash and verification commands.
- If a PR is used, the PR description should include `Closes #12`.

## Suggested First Agent Prompt

```text
You are working in the Sunny Town HQ repo on branch codex/inventory-redesign-dev.

Start with GitHub issue #12:
https://github.com/walt-raymond-williams/sunny-town-hq/issues/12

Read AGENTS.md and docs/SUNNY_TOWN_INVENTORY_SLOT_COMPONENT_HANDOFF.md first. Treat issue #12 as the active task contract.

Goal: build a reusable frontend inventory slot component for Sunny Town inventory/hotbar/equipment/crafting/container surfaces. Keep this task display-focused. Do not implement slotted inventory persistence, inventory grid rewrite, hotbar/equipment drag/drop, or chest transfer.

Before editing, inspect frontend inventory types, current inventory panel, hotbar HUD, chest panel, shop item rendering, and existing icon CSS. Then implement the smallest reusable component that satisfies the issue acceptance criteria, wire it into a low-risk existing surface only if that is useful for verification, run frontend build/tests as appropriate, and report the result with issue closeout notes.
```
