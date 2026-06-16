# NPC Life Work Day Cadence Handoff

## Purpose

This handoff starts GitHub issue #50, "Add configurable NPC day cadence."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/50

The goal is to replace real UTC-hour NPC schedule phase behavior with a server-owned configurable simulated day length so the Cookie Keeper home/work demo can be reviewed quickly and deterministically.

## Current Status

- Epic #38 is open for NPC life/work simulation.
- #48 design work is closed.
- #49 Cookie Keeper home map and bed fixture is closed in `01defbd`.
- #50 is `status:ready`.
- #51 remains blocked until #50 lands.
- Work should happen on `codex/npc-life-work-dev`.

## Product Direction

- The NPC day clock is server-owned.
- Clients must not control simulation time.
- Default/local-play cadence: `24` real minutes per simulated day.
- Demo/dev cadence: `8` real minutes per simulated day.
- Existing schedule phases remain:
  - `morning`
  - `day`
  - `evening`
  - `night`
- Existing schedule pressure semantics should remain recognizable:
  - day biases work
  - night biases home/rest
  - evening biases social
  - morning biases food where strong anchors exist
- The cadence should support review/debug without requiring a real-world wait.

## Current Code Surface

Sunny Town config/startup:

- `internal/sunnytown/config`
  - Add config parsing if this package owns Sunny Town environment settings.
- `cmd/sunny-town/main.go`
  - Wire config into server construction if needed.
- `internal/sunnytown/server/server.go`
  - Server/runtime struct may need to carry cadence config into rooms/NPC code.

NPC schedule/runtime:

- `internal/sunnytown/server/npc_schedule.go`
  - Current phase logic uses UTC-hour phase bands.
- `internal/sunnytown/server/npc_schedule_test.go`
  - Add deterministic tests for compressed day phase calculation.
- `internal/sunnytown/server/npc_debug.go`
  - Ensure current phase and schedule pressure remain visible in debug output.

Docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
  - Update if current-state cadence behavior changes.
- `docs/current/RUNTIME.md`
  - Update if a new environment variable is introduced.

## Recommended Implementation Plan

1. Inspect current NPC schedule phase calculation and tests.
2. Add `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES` parsing with default `24`.
3. Represent day length as a server-owned duration or minutes value.
4. Derive simulated phase from server time using the configured day length.
5. Keep tests deterministic by injecting or passing explicit times into the phase helper.
6. Preserve existing phase names and pressure behavior.
7. Ensure `/debug/npcs` still reports phase/schedule pressure.
8. Update current docs for the accepted environment variable/runtime behavior.
9. Run focused tests and `task verify`.

## Design Notes

Suggested phase split for one simulated day:

- `morning`: first 25 percent
- `day`: next 35 percent
- `evening`: next 20 percent
- `night`: final 20 percent

This does not need to be perfect life-sim math. It needs to make work/rest observable and keep the behavior easy to reason about.

If the current code already has a useful phase split, prefer preserving it proportionally inside the configured simulated day.

## Out Of Scope

- Tuning Cookie Keeper home/work behavior (#51).
- Adding frontend controls for cadence.
- Letting clients control simulation time.
- Persisting schedule phase, raw drive values, or route state.
- Changing Cookie Keeper production rules.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
task verify
```

Manual/debug smoke if practical:

- Run Sunny Town with `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES=8`.
- Confirm `/debug/npcs` phase changes over the compressed day.
- Confirm phase/schedule pressure remains server-owned.

## Close Criteria

Close #50 when:

- `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES` exists and defaults to `24`.
- Tests cover compressed phase behavior.
- Current docs describe the new config.
- Work is committed and pushed to `codex/npc-life-work-dev`.
- The issue close comment records commit hash and verification results.
