# NPC Life Work Routine Debug Handoff

## Purpose

This handoff starts GitHub issue #52, "Improve NPC routine debug output for demo review."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/52

The goal is to make Cookie Keeper's routine state easy to inspect while he travels between home and work, rests, works, or gets blocked.

## Current Status

- Epic #38 is open for NPC life/work simulation.
- #48, #49, #50, and #51 are closed.
- #52 is open and `status:ready`.
- Related issue #4 still owns the broader production posture for the NPC debug endpoint.
- Work should happen on `codex/npc-life-work-dev`.

## Product Direction

The first demo loop now depends on being inspectable. A reviewer should be able to call `/debug/npcs` and understand:

- what Cookie Keeper is trying to do
- whether the selected goal is home/rest, work, food, social, or idle
- which map/location the NPC is targeting
- whether route planning or route following is blocked
- how drive values and schedule pressure influenced the current choice
- whether Cookie Keeper is eligible to produce shop progress at the counter
- whether production is blocked because HQ rejected the durable production commit as blocked

Keep the output structured JSON. Logs are useful supporting evidence, but the debug endpoint should carry the routine diagnosis fields that a reviewer needs.

## Current Code Surface

Primary file:

- `internal/sunnytown/server/npc_debug.go`

Existing tests:

- `internal/sunnytown/server/npc_debug_test.go`
- `internal/sunnytown/server/npc_production_test.go`
- `internal/sunnytown/server/npc_schedule_test.go`

Supporting behavior:

- `internal/sunnytown/server/npc_movement.go`
  - Active goal, route, route failures, failed target cooldowns.
- `internal/sunnytown/server/npc_schedule.go`
  - Current phase and schedule pressure.
- `internal/sunnytown/server/npc_anchors.go`
  - Home/work/food/social anchor selection.
- `internal/sunnytown/server/npc_production.go`
  - Local work-anchor eligibility and production progress.
- `internal/sunnytown/server/server_workers.go`
  - HQ production commit worker and blocked production logging.
- `internal/sunnytown/hqclient/client.go`
  - `NPCJobProductionResponse.Blocked` and `BlockedReason`.

Docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`

## Current Debug Endpoint Baseline

`/debug/npcs` already exposes a strong baseline:

- room/map/tick grouping
- NPC identity, map, position, facing, and moving state
- drive values
- routine anchors
- schedule phase and pressure values
- active drive
- goal map/location/anchor kind/tags
- route step index, path index, step count, current map, current portal, and target map
- focus, reevaluation, goal start, and goal arrival timestamps
- failure count and failed target retry timestamps
- production eligibility, job key, output key, local progress, last production time, and last local event
- service-secret protection when `ServiceSecret` is configured

Do not remove or rename those fields without a compatibility reason.

## Recommended Implementation Plan

1. Inspect the current #51 implementation and the actual `/debug/npcs` JSON for Cookie Keeper under the demo cadence.
2. Identify the smallest missing fields needed to diagnose travel/rest/work behavior.
3. Add blocked or failed route detail if the existing `failedTargets` key is not readable enough for demo review.
4. Add production blocked visibility beyond logs if feasible without broad HQ state polling.
   - At minimum, make local debug state distinguish "not at work anchor", "no durable character identity", and "waiting on/last queued local event" when those are the blocker.
   - If adding HQ blocked responses to runtime state is small, capture the last blocked response reason from `RunNPCJobProductionWorker`.
5. Add or update tests around the debug fields most likely to regress:
   - schedule pressure in debug output
   - route progress in debug output
   - failed target or route-blocked state
   - production debug state for eligible and ineligible Cookie Keeper
   - endpoint protection
6. Update `docs/current/` only if the accepted debug contract changes in a durable way.

## Out Of Scope

- Building a frontend NPC debug panel.
- Changing the Cookie Keeper home/work routine.
- Changing schedule cadence.
- Solving issue #4's broader production posture unless a narrow protection bug blocks #52.
- Adding persistent NPC routine state.
- Changing HQ inventory or Cookie Shop production rules.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
task verify
```

Manual smoke if practical:

```powershell
$env:SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES = "8"
docker compose -f deploy/docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/debug/npcs -Headers @{"X-HQ-Service-Secret"="<service secret>"} | Select-Object -ExpandProperty Content
```

Inspect Cookie Keeper while he is:

- traveling between `sunny-town-house-1`, `sunny-town-v1`, and `sunny-town-cookie-keeper-home`
- resting at `cookie-keeper-bed`
- working at `cookie-keeper-counter`
- blocked from a target in a forced or test scenario

## Close Criteria

Close #52 when:

- `/debug/npcs` can explain Cookie Keeper's current routine state during travel, rest, and work.
- Blocked route or failed target state is readable.
- Cookie Keeper production blocked or ineligible state is visible through structured debug output or a clearly documented log path.
- Tests cover the useful debug fields.
- Work is committed and pushed to `codex/npc-life-work-dev`.
- The issue close comment records commit hash and verification results.
