# NPC Life Work Cookie Keeper Home Handoff

## Purpose

This handoff starts GitHub issue #49, "Author Cookie Keeper home map and bed fixture."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/49

The goal is to author Cookie Keeper's home as a real Sunny Town map destination with a visible usable bed, so later routine work can route him between home/rest and the Cookie Shop counter.

## Current Status

- Epic #38 is open for NPC life/work simulation.
- Design issue #48 is closed.
- Accepted design decisions are captured in:
  - `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
  - `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
  - `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- Work should happen on `codex/npc-life-work-dev`.
- This issue unblocks #50 and #51.

## Product Direction

- Cookie Keeper is the first demo NPC.
- The northwest unoccupied building in `sunny-town-v1` becomes Cookie Keeper's home entrance.
- Cookie Keeper's home is a new interior map, not a reused room.
- Beds should be visible fixtures/world objects from the start.
- Bed metadata should stay compatible with future player-placeable furniture.
- Do not implement player placement or durable HQ home-assignment schema in this issue.

## Accepted IDs And Coordinates

Main-town portal:

- ID: `cookie-keeper-home-door`
- Map: `sunny-town-v1`
- Position: `x: 240`, `y: 264`
- Size: `width: 64`, `height: 24`
- Target map: `sunny-town-cookie-keeper-home`
- Target position: `x: 320`, `y: 416`
- Target facing: likely `up`, unless map validation/pathing suggests another direction.

Home map:

- ID: `sunny-town-cookie-keeper-home`
- Start with compact interior dimensions similar to `sunny-town-house-1`.
- Exit portal ID: `cookie-keeper-home-exit-door`
- Exit portal position: `x: 304`, `y: 448`
- Exit portal size: `width: 64`, `height: 32`
- Exit target map: `sunny-town-v1`
- Exit target position: `x: 272`, `y: 320`

Locations:

- `cookie-keeper-home`
  - Home area for Cookie Keeper.
- `cookie-keeper-bed`
  - Tags: `rest`, `home`, `bed`, `sleep`, `personal`
  - `ownerNpcKey: "cookie-keeper"`
  - `capacity: 1`

Fixture:

- ID: `cookie-keeper-bed-fixture`
- Kind: `bed`
- `locationId: "cookie-keeper-bed"`
- `itemKey: "simple_bed"`
- Tags: `furniture`, `bed`, `sleep`, `rest`, `placeable`

## Current Code Surface

Maps:

- `sunny-town/maps/sunny-town-v1.json`
  - Add main-town portal and split northwest building blocked rectangle to leave a reachable doorway gap.
- `sunny-town/maps/sunny-town-house-1.json`
  - Use as interior layout reference.
- New file:
  - `sunny-town/maps/sunny-town-cookie-keeper-home.json`

Map loading and validation:

- `internal/sunnytown/maps/maps.go`
- `internal/sunnytown/maps/maps_test.go`

World fixtures and snapshots:

- `internal/sunnytown/server/world_objects.go`
- `internal/sunnytown/server/world_fixtures_test.go`
- `internal/sunnytown/server/world_types.go`
- `frontend/src/types/sunnyTown.ts` if fixture kind typing needs a frontend update.

NPC route/anchor tests:

- `internal/sunnytown/server/npc_anchors_test.go`
- `internal/sunnytown/server/npc_movement_test.go`

## Recommended Implementation Plan

1. Add the new `sunny-town-cookie-keeper-home.json` map.
2. Add the `cookie-keeper-home-door` portal to `sunny-town-v1`.
3. Split/adjust the northwest building blocked rectangle so the portal center is reachable.
4. Add `cookie-keeper-home` and `cookie-keeper-bed` locations.
5. Add `cookie-keeper-bed-fixture`.
6. Extend fixture validation/world-object initialization for `kind: "bed"` without weakening chest behavior.
7. Add or update map validation tests for the new portal, home locations, bed fixture, and existing Mayor bed.
8. Add route/anchor tests proving Cookie Keeper can route from bed to counter and counter to bed.
9. Run focused tests.

## Out Of Scope

- Configurable NPC day cadence (#50).
- Routine tuning or drive changes (#51).
- NPC debug output improvements (#52).
- Player-placeable beds.
- Durable HQ home assignment schema.
- Cookie Keeper inventory.

## Verification

Run:

```powershell
go test ./internal/sunnytown/maps
go test ./internal/sunnytown/server
```

If fixture type changes touch shared frontend types or snapshots, also run:

```powershell
cd frontend
npm run build
```

## Close Criteria

Close #49 when:

- Cookie Keeper home map, portal, home/bed locations, and bed fixture are implemented.
- Tests prove map validation and routeability.
- Work is committed and pushed to `codex/npc-life-work-dev`.
- The issue close comment records commit hash and verification results.
