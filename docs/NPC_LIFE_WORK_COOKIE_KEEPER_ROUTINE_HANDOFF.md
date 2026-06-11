# NPC Life Work Cookie Keeper Routine Handoff

## Purpose

This handoff starts GitHub issue #51, "Tune Cookie Keeper home/work demo loop."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/51

The goal is to make Cookie Keeper visibly alternate between his home bed and Cookie Shop work counter on a demo-friendly, server-authoritative routine.

## Current Status

- Epic #38 is open for NPC life/work simulation.
- #48 design work is closed.
- #49 authored Cookie Keeper home map and bed fixture.
- #50 added configurable NPC day cadence.
- #51 is `status:ready`.
- #52 remains blocked until #51 lands.
- Work should happen on `codex/npc-life-work-dev`.

## Product Direction

- Cookie Keeper is the first demo NPC for life/work simulation.
- Home/rest anchor:
  - map: `sunny-town-cookie-keeper-home`
  - location: `cookie-keeper-bed`
- Work anchor:
  - map: `sunny-town-house-1`
  - location: `cookie-keeper-counter`
- The routine should be observable under the demo cadence.
- The behavior should remain drive-driven with server-owned schedule pressure, not a hard client-controlled schedule.
- Cookie production must still require Cookie Keeper to be physically at `cookie-keeper-counter`.

## Current Code Surface

NPC drives, schedule, and movement:

- `internal/sunnytown/server/npc_movement.go`
  - Route following, goal selection, drive behavior, room transitions.
- `internal/sunnytown/server/npc_schedule.go`
  - Schedule phase/pressure, now using configurable day cadence from #50.
- `internal/sunnytown/server/npc_anchors.go`
  - Home/work anchor resolution.
- `internal/sunnytown/server/npc_production.go`
  - Cookie Keeper production validation.
- `internal/sunnytown/server/server_workers.go`
  - Runtime ticks/workers if cadence affects update timing.

Tests:

- `internal/sunnytown/server/npc_movement_test.go`
- `internal/sunnytown/server/npc_schedule_test.go`
- `internal/sunnytown/server/npc_anchors_test.go`
- `internal/sunnytown/server/npc_production_test.go`

Docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`

## Expected Behavior

Under the demo/dev day cadence:

- Cookie Keeper should go to `cookie-keeper-counter` when work pressure is dominant.
- Cookie Keeper should go to `cookie-keeper-bed` when rest/home pressure is dominant.
- The route should use portals between `sunny-town-house-1`, `sunny-town-v1`, and `sunny-town-cookie-keeper-home`.
- Goal changes should respect existing focus windows/cooldowns enough to avoid rapid flipping.
- If the home or work route fails, debug/log state should make the failure discoverable.

## Recommended Implementation Plan

1. Inspect current drive and schedule pressure behavior after #50.
2. Verify Cookie Keeper anchors resolve to `cookie-keeper-bed` and `cookie-keeper-counter`.
3. Tune drive thresholds, schedule pressure, or owner-anchor preference so Cookie Keeper reliably chooses work and home/rest over time.
4. Add tests proving Cookie Keeper can select and route to both anchors.
5. Add tests proving route transitions keep server snapshots consistent across maps if existing coverage is not enough.
6. Verify Cookie Keeper production still only happens at the counter.
7. Update current docs if routine behavior or cadence semantics become more specific.

## Out Of Scope

- New maps or fixtures, unless #49 missed a small required correction.
- Debug endpoint polish beyond what is needed for tests/logging; #52 owns richer debug output.
- Frontend visual routine cues.
- Durable home assignment schema.
- NPC inventory or player-placeable beds.
- Changing Cookie Shop recipe/storage rules.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
task verify
```

Manual smoke if practical:

- Run Sunny Town with `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES=8`.
- Observe Cookie Keeper routing between home bed and work counter.
- Confirm Cookie Keeper only produces while physically at `cookie-keeper-counter`.

## Close Criteria

Close #51 when:

- Cookie Keeper visibly alternates between home/rest and work under demo cadence.
- Tests cover the routine selection/route behavior.
- Cookie production validation still depends on counter presence.
- Work is committed and pushed to `codex/npc-life-work-dev`.
- The issue close comment records commit hash and verification results.
