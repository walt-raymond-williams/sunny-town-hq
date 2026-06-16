# NPC Life Work Visual Cues Handoff

## Purpose

This handoff starts GitHub issue #55, "Add optional NPC routine visual cues."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/55

The goal is to make the Cookie Keeper home/work demo readable without requiring a reviewer to keep `/debug/npcs` open.

## Current Status

- Epic #38 is open for NPC life/work simulation.
- #48 through #52 are closed.
- #55 is open and `status:ready`.
- The first backend demo loop is in place: Cookie Keeper has a home, bed, day cadence, home/work routing, and debug output.
- Work should happen on `codex/npc-life-work-dev`.

## Product Direction

Add subtle routine cues for NPCs, starting with Cookie Keeper.

The first slice should help a demo reviewer distinguish:

- traveling
- resting
- working
- blocked

Do not overbuild this into a full NPC inspector. A small dev/demo cue is enough if it reflects authoritative server state. If the implementation cannot get a trustworthy routine state cheaply, prefer a smaller truthful cue over client-side guessing.

## Current Code Surface

Frontend rendering:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Calls `drawNpc` for each rendered NPC.
- `frontend/src/features/sunny-town/rendering/characterDrawing.ts`
  - Draws NPC body, nearby ring, and name label.
- `frontend/src/features/sunny-town/rendering/characterDrawing.test.ts`
  - Existing rendering helper test surface.
- `frontend/src/composables/useSunnyTownRemoteNpcs.ts`
  - Smooths authoritative NPC snapshots.
- `frontend/src/types/sunnyTown.ts`
  - `SunnyTownNpc` and server message types.

Server/debug state:

- `internal/sunnytown/server/npc_debug.go`
  - `/debug/npcs` exposes routine, goal, route, failure, schedule, and production state.
- `internal/sunnytown/server/npc_movement.go`
  - Owns current NPC goal/route state.
- `internal/sunnytown/server/npc_production.go`
  - Owns work-anchor eligibility.

Docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/NPC_LIFE_WORK_ROUTINE_DEBUG_HANDOFF.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`

## Design Guidance

Keep cues compact and operational:

- Prefer a tiny icon, badge, or short status marker near the NPC name.
- Do not obscure the map, NPC body, player, interaction hint, or shop/dialog panels.
- Avoid large labels or permanent explanatory text.
- If the cue is dev/demo-only, make the condition explicit in code and issue notes.
- Keep the normal player experience quiet; this is not a landing-page moment.

Possible implementation paths:

- Add server-authoritative routine state to the normal NPC snapshot, then draw cues from `SunnyTownNpc.routine`.
- Add a small frontend debug poller for `/debug/npcs`, gated to dev/demo mode, and merge debug state by NPC ID.
- Add a narrow backend endpoint/message only if the normal snapshot should not carry debug-only fields.

Recommendation: start by deciding whether the cue belongs in normal snapshots or a dev/debug path. If it will be visible to normal players, use normal authoritative snapshots. If it is only for review tooling, prefer a debug-gated path and avoid broad snapshot protocol churn.

## Recommended Implementation Plan

1. Inspect the #52 implementation and the live `/debug/npcs` JSON shape.
2. Decide whether this first cue is normal UI or dev/demo-only.
3. Define a small frontend routine status type, for example `traveling`, `resting`, `working`, `blocked`.
4. Wire authoritative routine state to rendered NPCs without breaking NPC smoothing.
5. Extend `drawNpc` with an optional status cue input.
6. Add focused tests for status derivation and rendering helper behavior where practical.
7. Run frontend build and server tests if server fields change.
8. Update current docs if the NPC snapshot/debug contract changes.

## Out Of Scope

- Full NPC debug panel.
- Player-facing schedule UI.
- New routine logic or drive tuning.
- Persistent NPC state.
- Solving the broader `/debug/npcs` production posture from #4.
- Large Sunny Town page refactors.

## Verification

Run:

```powershell
cd frontend
npm run build
```

If server snapshot or debug fields change, also run:

```powershell
go test ./internal/sunnytown/server
task verify
```

Manual smoke if practical:

- Run with `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES=8`.
- Observe Cookie Keeper while traveling, resting, and working.
- Confirm cues stay readable and do not cover existing interaction hints or overlays.
- Confirm no cue appears when no trustworthy routine state is available.

## Close Criteria

Close #55 when:

- Cookie Keeper has a subtle, truthful routine cue for the supported states.
- The cue uses authoritative state or an explicitly debug-gated source.
- Existing Sunny Town rendering and NPC interactions still work.
- Verification results are recorded on the issue.
- Work is committed and pushed to `codex/npc-life-work-dev`.
