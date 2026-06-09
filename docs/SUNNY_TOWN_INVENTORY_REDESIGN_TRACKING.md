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

## Management Workflow Loop

Use this loop when advancing the inventory redesign backlog:

1. Check live GitHub issue state with `gh issue list` and inspect the last completed issue comments.
2. Confirm the previous ticket is closed with a commit hash and verification notes.
3. Pick the next roadmap item based on blockers, dependencies, and the current integrated branch state.
4. Create the GitHub issue if it does not already exist.
5. Create a focused handoff doc under `docs/` for the selected issue.
6. Update this tracking file with the closed/open issue snapshot, recommended next task, and handoff path.
7. Commit and push only the handoff/tracking docs to `codex/inventory-redesign-dev`.
8. Comment on the GitHub issue with the handoff path, branch, and commit hash.
9. The implementation agent works from the GitHub issue plus the linked handoff doc.
10. When implementation lands on `codex/inventory-redesign-dev`, close the issue with the commit hash and verification results.
11. Repeat from step 1.

Keep GitHub issues as the execution source of truth. Keep this file as the compact local recovery snapshot for compaction, coordination, and fresh-agent handoff.

## Current GitHub Issues

Snapshot refreshed: 2026-06-09 after creating issue #22.

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

- #18 Remove legacy button-based inventory assignment UI  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/18

- #19 Add character preview shell  
  Completed in `20bac63`.  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/19

- #21 Expand E menu to use the play area  
  Completed in `985ef34`.  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/21

### Open

- #20 Add stats-ready character panel  
  Status: `status:in-progress`  
  Blocked by: #19 and #21, both closed  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/20

- #22 Move equipment slots into character preview card  
  Status: open follow-up / ready after #20 coordination  
  Related: #19, #20, #17  
  Handoff: `docs/SUNNY_TOWN_CHARACTER_EQUIPMENT_RAIL_HANDOFF.md`  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/22

## Recommended Next Management Step

Start issue #20:

`Add stats-ready character panel`

Why:

- The character preview shell landed and is verified.
- The expanded play-area E menu landed and is verified.
- The stats-ready panel directly follows the preview work now that the menu has room.
- This preserves room for future activity-driven progression without adding fake stats or manual point allocation.
- It is a small frontend slice before larger crafting, container, and tabbed-menu work.
- Handoff: `docs/SUNNY_TOWN_STATS_READY_CHARACTER_PANEL_HANDOFF.md`.

Follow-up UI polish now tracked:

`Move equipment slots into character preview card` (#22)

Why:

- Manual review found the gear/accessory/tool slots listed below the character image, which wastes vertical space and reads poorly.
- The desired layout is a compact equipment rail inside the character preview card, preferably to the left of the avatar.
- This should generally wait for #20 or coordinate with it, because both touch the same character-panel surface.
- Handoff: `docs/SUNNY_TOWN_CHARACTER_EQUIPMENT_RAIL_HANDOFF.md`.

## Useful Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 20 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 22 --comments
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
- `docs/SUNNY_TOWN_CHARACTER_PREVIEW_HANDOFF.md`
- `docs/SUNNY_TOWN_E_MENU_PLAY_AREA_HANDOFF.md`
- `docs/SUNNY_TOWN_STATS_READY_CHARACTER_PANEL_HANDOFF.md`
- `docs/SUNNY_TOWN_CHARACTER_EQUIPMENT_RAIL_HANDOFF.md`
