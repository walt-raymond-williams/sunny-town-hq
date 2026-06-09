# Sunny Town Inventory Metadata Handoff

## Purpose

Use this handoff to start a Codex agent on the first inventory redesign task:

GitHub issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/8

Issue title:

`Add item metadata needed for grid inventory`

## Task Contract

Treat GitHub issue #8 as the active task contract.

Goal:

Add the item metadata foundation needed for the future Sunny Town grid inventory UI, while keeping existing inventory, hotbar, equipment, crafting, mining, and placement behavior backward compatible.

User story:

As a player, I want items to have stable icons, display text, stack rules, and categories, so that I can recognize items visually in a grid inventory.

## Read First

- `AGENTS.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TICKETS.md`
- `docs/INVENTORY_AND_EQUIPMENT.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/ARCHITECTURE.md`

## Current Architecture Summary

- HQ owns durable inventory state and database schema.
- Sunny Town owns realtime gameplay validation and live world state.
- Inventory item catalog data currently lives in `inventory_item_type`.
- `inventory_item_type` currently has:
  - `key`
  - `name`
  - `description`
  - `equip_slot`
  - `visual_key`
- `visual_key` is used for Sunny Town avatar/equipment rendering. Do not casually reuse it as the inventory icon key unless the issue discussion explicitly decides that.
- Current frontend item icons are mostly CSS classes in `frontend/src/style.css`, such as `inventory-item__icon--cookie`.
- Current inventory is aggregate quantity rows, not slotted inventory yet.
- This issue should not implement slotted inventory. It should prepare item metadata so later grid/slot tickets have durable item display data.

## Files To Inspect

Backend:

- `deploy/postgres/migrations/0003_inventory_equipment.sql`
- `internal/hq/inventory/inventory.go`
- `internal/hq/inventory/hotbar.go`
- `internal/hq/inventory/equipment.go`
- `internal/hq/inventory/crafting.go`
- `internal/hq/inventory/http.go`
- `internal/hq/inventory/*_test.go`
- `internal/hq/schema`

Frontend:

- `frontend/src/types/inventory.ts`
- `frontend/src/api/inventoryApi.ts`
- `frontend/src/api/hotbarApi.ts`
- `frontend/src/api/equipmentApi.ts`
- `frontend/src/api/craftingApi.ts`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/features/sunny-town/SunnyTownHud.vue`
- `frontend/src/style.css`

Docs likely needing updates if behavior/schema changes:

- `docs/INVENTORY_AND_EQUIPMENT.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`

## Acceptance Criteria

- Item catalog data can represent an inventory icon key or asset key.
- Item catalog data can represent max stack size, or the decision to defer max stack size is documented.
- Inventory, hotbar, equipment, and crafting responses expose the metadata needed by slot UI.
- `visual_key` remains available for avatar/equipment rendering and is not accidentally overloaded without an explicit decision.
- Existing item API consumers remain backward compatible or have a documented migration path.

## Recommended Implementation Shape

Conservative likely path:

1. Add explicit item metadata columns to `inventory_item_type`, such as:
   - `icon_key`
   - `max_stack`
   - possibly `category`
2. Seed current item types with icon keys that match the existing frontend icon class suffixes:
   - `cookie`
   - `sunny_hoodie`
   - `star_cap`
   - `pickaxe`
   - `rock`
   - `crystal`
   - `stone_block`
3. Keep `visual_key` separate for avatar/equipment rendering.
4. Return the new metadata from:
   - `GET /api/student/inventory`
   - `GET /api/student/hotbar`
   - `GET /api/student/equipment`
   - `GET /api/student/crafting/recipes`
   - `POST /api/student/crafting/craft`
5. Update frontend TypeScript types and API normalizers to accept the new fields.
6. Keep existing UI behavior working. Do not redesign the inventory UI in this issue.
7. Update current docs if schema/API response shapes change.

If the agent decides max stack size or category should be deferred, that decision must be written in the GitHub issue and docs with a clear reason.

## Out Of Scope

- No slotted inventory table.
- No stack move/swap/merge API.
- No inventory grid UI rewrite.
- No drag/drop behavior.
- No hotbar or equipment UI redesign.
- No chest/container inventory.
- No storage-aware crafting behavior.
- No character preview work.

## Verification

Run the narrowest checks that cover the changes:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/schema
cd frontend
npm run build
```

If broader schema or API behavior changes touch other packages, run:

```powershell
go test ./...
```

## GitHub Workflow

Before implementation:

1. Open issue #8.
2. Add or update a comment if the chosen metadata shape differs from this handoff.
3. Change the issue label from `status:ready` to `status:in-progress`.

During implementation:

- Work from issue #8.
- Keep issue comments for durable discoveries, scope changes, blockers, and verification results.
- If the work reveals a separate task, create a linked follow-up issue instead of expanding #8 too far.

Closeout:

- PR description should include `Closes #8`.
- PR description should list verification commands run.
- If no PR is used, commit message should reference `#8`.
- If API/schema/current architecture changed, update `docs/current/` before closing.

## Suggested First Agent Prompt

```text
You are working in the Sunny Town HQ repo. Start with GitHub issue #8:
https://github.com/walt-raymond-williams/sunny-town-hq/issues/8

Read AGENTS.md and docs/SUNNY_TOWN_INVENTORY_METADATA_HANDOFF.md first.
Treat issue #8 as the active task contract.

Goal: add backward-compatible item metadata needed by the future grid inventory UI. Keep visual_key separate from inventory icon metadata unless you explicitly justify otherwise. Do not implement slotted inventory or drag/drop in this task.

Before editing, audit the current item catalog migration, inventory/hotbar/equipment/crafting response structs, frontend inventory types, and API normalizers. Then implement the smallest durable metadata change that satisfies the issue acceptance criteria, update current docs if API/schema changes, run verification, and report the result with issue/PR closeout notes.
```
