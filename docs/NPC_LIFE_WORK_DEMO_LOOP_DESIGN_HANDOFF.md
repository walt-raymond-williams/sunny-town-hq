# NPC Life Work Demo Loop Design Handoff

## Purpose

This handoff starts GitHub issue #48, "Design NPC home and work demo loop."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/48

The goal is to lock the first NPC life/work vertical slice before agents edit maps or routine code.

## Current Status

- Epic #38 is open: `Epic: NPC life and work simulation`.
- Discovery and roadmap docs exist:
  - `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
  - `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- Current integration branch for this epic:
  - `codex/npc-life-work-dev`
- The first implementation should center on Cookie Keeper moving between a home/bed and `cookie-keeper-counter`.

## Product Direction

- First demo NPC: Cookie Keeper.
- Add a new home interior map and portal from an unoccupied building in `sunny-town-v1`.
- Beds should be visible fixtures/world objects from the start.
- Bed metadata should be compatible with future player-placeable furniture.
- Cookie Keeper should have an explicit home assignment.
- Inside his assigned home, Cookie Keeper should use an available bed.
- Routine model starts drive-driven, not hard schedule-driven.
- Cadence target:
  - default/local-play: 24 real minutes per simulated day
  - demo/dev: 8 real minutes per simulated day

## Current Code And Doc Surface

Maps:

- `sunny-town/maps/sunny-town-v1.json`
  - Main town, building shapes, portals to the existing house/classroom.
- `sunny-town/maps/sunny-town-house-1.json`
  - Existing interior, Cookie Keeper work anchor, Cookie Shop, Mayor Sunny bed, fixtures.
- `sunny-town/maps/sunny-town-classroom.json`
  - Existing interior and Teacher work anchor.

NPC/runtime docs:

- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
  - Current pathing, drives, anchors, debug, and remaining notes.
- `docs/current/NPC_CHARACTER_MODEL_PLAN.md`
  - Shared character identity and NPC model direction.
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
  - Cookie Keeper work/storage/production model.

Epic docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`

## Questions To Close

1. Which unoccupied main-town building becomes Cookie Keeper's home entrance?
2. What are the new map ID, portal ID, and target coordinates?
3. What location IDs/tags represent Cookie Keeper's home area?
4. What fixture metadata marks a bed as visible and usable?
5. Should the authored bed fixture also have an explicit matching `location`, or should the fixture itself be enough for routine targeting?
6. For the first slice, is routine behavior pure drive-driven or drive-driven with time-of-day pressure?
7. Should demo cadence be a hard-coded server constant first, or a runtime config value?
8. What exact child issues should be created after this design issue closes?

## Recommended Plan

1. Inspect current map geometry and blocked rectangles in `sunny-town-v1`.
2. Pick the easiest unoccupied building entrance that can receive a portal without conflicting with current blockers.
3. Define a new home interior map using the existing house map as a layout reference.
4. Define the smallest bed fixture shape that can be visible now and placeable later.
5. Update the discovery/roadmap docs with accepted decisions.
6. Create or draft the next child issues:
   - `Author Cookie Keeper home and bed locations`
   - `Add Cookie Keeper home/work routine goal selection`
   - `Improve NPC routine debug output`

## Out Of Scope

- Implementing the new map.
- Implementing NPC routine code.
- Implementing player-placeable bed items.
- Adding durable home assignment schema unless the design proves it is required now.
- Broad NPC AI or day/night visuals.

## Verification

No code verification is required for documentation-only work.

If code or map JSON changes are made anyway, run:

```powershell
go test ./internal/sunnytown/maps
go test ./internal/sunnytown/server
task verify
```

## Close Criteria

Close #48 when:

- accepted design decisions are captured in `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md` and/or `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- next child issues are identified clearly enough to create
- work is committed and pushed to `codex/npc-life-work-dev`
- the issue close comment includes the commit hash and verification status
