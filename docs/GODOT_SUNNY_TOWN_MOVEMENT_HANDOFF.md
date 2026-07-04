# Godot Sunny Town Movement Handoff

## Purpose

This handoff starts GitHub issue #67, "Implement Godot movement, prediction, and remote interpolation."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/67

The goal is to preserve the current Sunny Town movement model in Godot.

## Current Status

- Blocked by #66.
- Rendering parity should exist before this starts.

## Product Direction

- Movement should feel immediate locally.
- Server snapshots should not rubber-band the local player during normal play.
- Sunny Town remains authoritative for accepted positions and gameplay effects.

## Current Code Surface

Docs:

- `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`

Frontend reference:

- `frontend/src/composables/useSunnyTownMovement.ts`
- `frontend/src/composables/useSunnyTownLocalPlayer.ts`
- `frontend/src/composables/useSunnyTownRemotePlayers.ts`

Backend:

- `internal/sunnytown/server/world_movement.go`
- `internal/sunnytown/server/constants.go`
- `internal/sunnytown/server/world_snapshots.go`

## Recommended Implementation Plan

1. Mirror current constants where needed:
   - player size: `28`
   - player speed: `150`
   - move send interval: about `50ms`
   - remote interpolation delay: about `150ms`
2. Add Godot input mappings for WASD and arrow keys.
3. Implement local movement vector/facing logic.
4. Use static blocked rectangles and active colliding world objects for local collision feel.
5. Send `move` messages with sequence numbers.
6. Stop movement on key release, blur, and visibility changes where Godot/web exposes them.
7. Keep local rendering from prediction and metadata from snapshots.
8. Implement remote player interpolation from snapshot history.
9. Reset movement state on `map_changed`.

## Out Of Scope

- Mobile touch controls.
- Server movement rewrite.
- New physics/collision model.

## Verification

Run:

```powershell
go test ./internal/sunnytown/server
cd frontend
npm run build
```

Manual smoke:

- Hold and release each movement key.
- Move diagonally.
- Walk into blocked rectangles and active placed objects.
- Use portals across current maps.
- Open two clients and confirm remote interpolation.
- Confirm stars/resources still trigger only from server-accepted position.

## Close Criteria

Close #67 when:

- Godot movement feels comparable to the canvas client.
- Valid sequence-numbered `move` messages are sent.
- Remote interpolation works in a two-client test.
- Portal transitions remain server-driven.
