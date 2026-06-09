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

### Closed

- #8 Add item metadata needed for grid inventory  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/8

- #12 Build reusable inventory slot component  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/12

### Open

- #9 Design slotted player inventory schema and compatibility plan  
  Status: `status:needs-design`  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/9

- #10 Implement slotted player inventory persistence  
  Status: `status:blocked`  
  Blocked by: #9  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/10

- #11 Add inventory stack move, swap, and merge API  
  Status: `status:blocked`  
  Blocked by: #10  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/11

- #13 Render Sunny Town inventory as a grid  
  Status: `status:blocked`  
  Blocked by: #10 and #12  
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/13

## Recommended Next Management Step

Start or hand off issue #9:

`Design slotted player inventory schema and compatibility plan`

Why:

- It unblocks the backend implementation path.
- #13 remains blocked until slotted inventory persistence exists.
- #12 is closed, so the frontend component foundation is available or expected to be available on the integration branch.

## Useful Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 9 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue edit 9 --remove-label "status:needs-design" --add-label "status:in-progress"
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
