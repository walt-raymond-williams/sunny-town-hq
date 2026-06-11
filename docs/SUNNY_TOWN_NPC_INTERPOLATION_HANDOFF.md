# Sunny Town NPC Interpolation Handoff

## Purpose

This handoff starts GitHub issue #47, "Smooth Sunny Town NPC movement between snapshots."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/47

The goal is to make live NPC movement in Sunny Town render smoothly between authoritative server snapshots. Today NPCs visually hop from point to point because the frontend renders the latest NPC snapshot directly instead of interpolating between recent snapshots.

## Current Status

- The server already simulates NPC movement continuously enough for smooth rendering:
  - `internal/sunnytown/server/constants.go`
    - `simulationInterval = 50 * time.Millisecond`
    - `snapshotInterval = 100 * time.Millisecond`
    - `npcSpeed = 80.0`
  - At this speed, each 100ms snapshot can move a walking NPC about 8 pixels.
- Remote players already have client-side interpolation.
- NPCs do not currently have client-side interpolation.
- This is expected to be a frontend-only fix unless implementation uncovers a missing timestamp/protocol detail.

## Symptom

When an NPC walks a route, the player sees it move as a series of small jumps between intermediate positions. It looks like path-node or waypoint teleporting, but the visible stepping is mostly snapshot cadence without interpolation.

## Diagnosis

Remote players:

- `frontend/src/composables/useSunnyTownRemotePlayers.ts`
  - Records snapshot history with `recordSnapshots`.
  - Renders at `now() - 150ms`.
  - Interpolates `x`, `y`, `facing`, and `moving` in `interpolateRemotePlayer`.
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Calls `remotePlayerState.recordSnapshots(players.value, selfId.value, message.serverTimeMs || Date.now())` after applying snapshots.
  - Uses `remotePlayerState.smoothPlayers(remotePlayers)` in `renderedSunnyTownPlayers()`.

NPCs:

- `frontend/src/composables/useSunnyTownWorldState.ts`
  - `applySnapshot` assigns `npcs.value = message.npcs !== undefined ? message.npcs : npcs.value`.
  - `applyMapState` assigns live NPCs from `message.npcs` or static map NPCs.
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - `renderedSunnyTownNpcs()` currently returns `npcs.value` directly.
  - `drawScene` draws that direct array.
  - `nearestNpcToSelf()` also uses `renderedSunnyTownNpcs()`.
- `frontend/src/features/sunny-town/rendering/characterDrawing.ts`
  - `drawNpc` draws an NPC at its current `npc.x` / `npc.y`.

The result: each incoming authoritative NPC snapshot becomes the rendered position immediately, with no visual smoothing.

## Recommended Implementation Plan

1. Add a frontend NPC smoothing composable.
   - Suggested file: `frontend/src/composables/useSunnyTownRemoteNpcs.ts`.
   - Keep it parallel to `useSunnyTownRemotePlayers` unless a small shared interpolation helper is clearly cleaner.
   - Use the same default interpolation delay as remote players, currently 150ms.
2. Record NPC snapshots when `SunnyTownPage.vue` handles `snapshot`, `hello`, and `map_changed` messages.
   - For normal snapshots, use `message.serverTimeMs || Date.now()`.
   - For `hello` / `map_changed`, clear old history and seed from the incoming authoritative NPCs if useful.
3. Render smoothed NPCs from `renderedSunnyTownNpcs()`.
   - Preserve authoritative display fields from the later frame:
     - `characterId`
     - `name`
     - `spriteKey`
     - `dialogue`
     - `shop`
     - `activity`
   - Interpolate only positional/motion fields:
     - `x`
     - `y`
     - `facing`
     - `moving`
4. Clear stale NPC histories.
   - If an NPC is absent from an authoritative `npcs` array, remove its history.
   - If `message.npcs` is an authoritative empty array, clear histories and render no NPCs.
   - On map change, clear histories from the old map so NPCs do not interpolate or ghost across maps.
5. Keep interaction behavior intentional.
   - `nearestNpcToSelf()` currently uses `renderedSunnyTownNpcs()`.
   - That is likely fine for feel because the visible hint matches the visible NPC.
   - If strict authority becomes important later, add a separate authoritative NPC lookup for interaction requests. Do not add that complexity unless this ticket reveals a real issue.
6. Add focused frontend tests.
   - Mirror the style of `frontend/src/composables/useSunnyTownRemotePlayers.test.ts`.
   - Cover interpolation between two NPC frames.
   - Cover carrying display fields from the later frame.
   - Cover stale history removal when an NPC disappears.
   - Cover empty authoritative NPC arrays clearing histories.
   - Cover single-frame behavior.

## Out Of Scope

- Changing server NPC simulation, pathfinding, drive logic, or snapshot cadence.
- Changing Sunny Town protocol unless a frontend timestamp gap is discovered.
- Reworking NPC interactions, dialogue, shops, schoolwork, or map transfers.
- Browser Fullscreen API or layout work.

## Verification

Run:

```powershell
cd frontend
npm run test
npm run build
```

If practical, rebuild runtime:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual smoke:

- Open Sunny Town as a student.
- Watch at least one NPC walk a route; movement should be continuous rather than stepping every snapshot.
- Walk near a moving NPC and confirm the talk/shop/schoolwork hint appears and disappears sensibly.
- Open and close an NPC overlay if available.
- Transition maps if possible and confirm NPCs on old/new maps do not ghost or interpolate from stale positions.
- If multiple NPCs are visible, confirm each moves independently without sharing history.

## Close Criteria

Close #47 when:

- Moving NPCs render smoothly between server snapshots.
- Existing remote player interpolation still works.
- NPC map changes and empty NPC snapshots do not leave stale rendered NPCs behind.
- Frontend tests/build pass.
- Manual smoke results are recorded on the issue.
- Work is committed and pushed.

Use a close comment like:

```text
Completed in <commit>. Sunny Town NPCs now interpolate between authoritative snapshots instead of rendering each snapshot as a jump. Verified with: <commands>. Manual smoke: <result or blocker>.
```
