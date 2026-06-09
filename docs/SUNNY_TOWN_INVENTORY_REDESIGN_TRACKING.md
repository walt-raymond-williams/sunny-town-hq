# Sunny Town Inventory Redesign Tracking

## Purpose

This document preserves the current GitHub/task workflow state for compaction recovery and fresh-agent handoff. GitHub issues remain the execution source of truth; this file is a local memory snapshot.

## Integration Branch

Current feature integration branch:

`codex/inventory-redesign-dev`

Workflow:

- Agents work issues against `codex/inventory-redesign-dev`.
- Issues may be closed once their work lands on the integration branch and verification is recorded.
- `main` remains stable until the integrated feature slice is ready for human review.
- A human reviews the integrated feature before merging `codex/inventory-redesign-dev` into `main`.

## Current GitHub Issues

Snapshot refreshed: 2026-06-09 after issue #18 creation.

### Closed

- #8 Add item metadata needed for grid inventory  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/8

- #9 Design slotted player inventory schema and compatibility plan  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/9

- #10 Implement slotted player inventory persistence  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/10

- #11 Add inventory stack move, swap, and merge API  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/11

- #12 Build reusable inventory slot component  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/12

- #13 Render Sunny Town inventory as a grid  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/13

- #15 Add inventory drag/drop state and API integration  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/15

- #16 Replace hotbar assignment buttons with drop targets  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/16

- #17 Replace equipment buttons with drop targets  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/17

### Open

- #18 Remove legacy button-based inventory assignment UI  
  Status: `status:ready`  
  Blocked by: #16 and #17, both closed  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/18

## Recommended Next Management Step

Start issue #18:

`Remove legacy button-based inventory assignment UI`

Why:

- Inventory slot movement, hotbar drops, and equipment drops are complete.
- The selected-item `Slot N` and `Wear` buttons are now duplicate assignment paths.
- Removing them makes the inventory menu feel like one coherent game UI before adding character preview or crafting polish.
- Handoff: `docs/SUNNY_TOWN_LEGACY_INVENTORY_BUTTON_REMOVAL_HANDOFF.md`.

## Useful Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 18 --comments
git checkout codex/inventory-redesign-dev
git pull
```

## Related Local Docs

- `AGENTS.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TICKETS.md`
- `docs/SUNNY_TOWN_INVENTORY_METADATA_HANDOFF.md`
- `docs/SUNNY_TOWN_INVENTORY_SLOT_COMPONENT_HANDOFF.md`
- `docs/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_HANDOFF.md`
- `docs/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_PLAN.md`
- `docs/SUNNY_TOWN_INVENTORY_DRAG_DROP_HANDOFF.md`
- `docs/SUNNY_TOWN_HOTBAR_DROP_TARGET_HANDOFF.md`
- `docs/SUNNY_TOWN_EQUIPMENT_DROP_TARGET_HANDOFF.md`
- `docs/SUNNY_TOWN_LEGACY_INVENTORY_BUTTON_REMOVAL_HANDOFF.md`
