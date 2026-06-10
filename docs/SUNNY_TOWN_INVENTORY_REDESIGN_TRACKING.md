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

Snapshot refreshed: 2026-06-10 after creating PR review follow-up issues #36 and #37.

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

- #20 Add stats-ready character panel
  Completed in `19e9957`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/20

- #21 Expand E menu to use the play area
  Completed in `985ef34`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/21

- #22 Move equipment slots into character preview card
  Completed in `c2c5247`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/22

- #23 Fix stone block crafting failure after slotted inventory migration
  Completed in `8da662b`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/23

- #24 Redesign crafting panel for inventory menu
  Completed in `1b082e7`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/24

- #25 Add storage context to crafting recipe APIs
  Completed in `99fe784`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/25

- #26 Design general container storage schema and access contract
  Completed in `52481cf`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/26

- #27 Add tabbed E menu shell
  Completed in `3786d8d`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/27

- #28 Define stats and skills progression model
  Completed in `cff87be`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/28

- #29 Add player to container transfer operations
  Completed in `f8a212b`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/29

- #30 Render chest inventory grid
  Completed in `bb8b30e`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/30

- #31 Add Cookie Shop input storage
  Completed in `a338f86`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/31

- #32 Add ingredient-aware Cookie Shop production
  Completed in `e210246`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/32

- #33 Add Cookie Shop input replenishment flow
  Completed in `ac8f7be`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/33

- #35 Add inventory stack splitting
  Completed in `999aa3e`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/35

- #34 Implement mining XP progression vertical slice
  Completed in `78f0a45`.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/34

- #6 Keep current-state docs synchronized with architecture changes
  Completed in `7d0f545`; stakeholder cleanup completed after that docs sync.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/6

### Open

- #36 Require equipped pickaxe for Sunny Town mining
  Status: `status:ready`
  Priority: `priority:p0`
  Handoff: `docs/SUNNY_TOWN_EQUIPPED_PICKAXE_MINING_HANDOFF.md`
  Blocking PR merge.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/36

- #37 Refresh mining XP in open character panel after mining
  Status: `status:ready`
  Priority: `priority:p2`
  Handoff: `docs/SUNNY_TOWN_MINING_XP_LIVE_REFRESH_HANDOFF.md`
  Non-blocking review polish; recommended after #36.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/37

## Recommended Next Management Step

PR review requested changes.

Recommended next step:

Start #36, `Require equipped pickaxe for Sunny Town mining`.

Why:

- It is a blocking PR review finding.
- Mining currently authorizes by pickaxe ownership instead of equipped tool state.
- It violates the project authority rule that gameplay effects must be validated server-side using accepted position and equipped tools.

Then address #37, `Refresh mining XP in open character panel after mining`, as non-blocking polish before re-review if practical.

## Useful Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' auth refresh -h github.com
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 6 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 25 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 26 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 27 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 28 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 33 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 34 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 35 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 36 --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 37 --comments
git checkout codex/inventory-redesign-dev
git pull
```

## Related Local Docs

- `AGENTS.md`
- `docs/current/API.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/CONTAINER_STORAGE.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
- `docs/current/DATABASE.md`
- `docs/current/PACKAGE_BOUNDARIES.md`
- `docs/current/SCHEMA_OWNERSHIP.md`
- `docs/current/STATS_SKILLS_PROGRESSION.md`
- `docs/archive/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md`
- `docs/archive/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
- `docs/archive/SUNNY_TOWN_INVENTORY_REDESIGN_TICKETS.md`
- `docs/archive/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_PLAN.md`
- `docs/archive/INVENTORY_AND_EQUIPMENT.md`
- `docs/SUNNY_TOWN_EQUIPPED_PICKAXE_MINING_HANDOFF.md`
- `docs/SUNNY_TOWN_MINING_XP_LIVE_REFRESH_HANDOFF.md`

Completed inventory handoff docs for issues #8 through #35 and #6 were deleted after their issue work landed. Durable rationale was preserved in `docs/current/` and `docs/archive/`.
