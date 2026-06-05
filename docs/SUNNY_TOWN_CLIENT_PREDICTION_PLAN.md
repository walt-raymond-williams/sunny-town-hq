# Sunny Town Client Prediction Plan

This plan improves Sunny Town movement feel by adding client-side prediction for the local player while keeping the server authoritative. Remote players should continue using interpolation.

Status: implemented. The server snapshots now include `lastProcessedSeq`, and the frontend predicts only the local player while reconciling against authoritative snapshots. The stop rubber-banding remedy is also implemented: server acknowledgements now mean an input was reflected in simulation, and the client uses release-aware correction easing.

## Goal

Make the local avatar start and stop immediately when the user presses movement keys, without waiting for the next server snapshot.

The implementation must avoid obvious rubber-banding. Small server corrections should be hidden with easing. Large corrections should snap only when necessary.

## Current State

- Sunny Town server runs an authoritative room simulation.
- Client sends movement input over WebSocket.
- Server broadcasts snapshots about 10 times per second.
- Client renders all players by smoothing visual positions toward the latest snapshot.
- This makes movement better than raw snapshots, but the local player still feels slightly delayed and steppy.

## Protocol Changes

Keep the existing client input shape:

```json
{
  "type": "input",
  "seq": 12,
  "up": false,
  "down": true,
  "left": false,
  "right": false
}
```

Add the latest processed input sequence to each server player snapshot:

```json
{
  "id": "123",
  "displayName": "Student",
  "x": 640,
  "y": 480,
  "facing": "down",
  "moving": true,
  "avatarId": "pet-default",
  "lastProcessedSeq": 12
}
```

Server requirements:

- Store both the latest input `seq` received for each player and the latest input `seq` reflected by a simulation step.
- Include the simulated sequence value in each snapshot as `lastProcessedSeq`.
- Do not acknowledge a newly received input in a snapshot until `room.step` has applied the current input state.
- Continue ignoring client-provided coordinates.
- Continue using the fixed server tick as authoritative time.

## Client Prediction Model

Use prediction only for the local player.

Client state:

```text
authoritativeSelf
  last server snapshot for self

predictedSelf
  authoritative self plus replayed unacknowledged inputs

renderedSelf
  what the canvas actually draws

pendingInputs
  inputs sent by the client but not yet acknowledged by the server
```

Input handling:

- On keydown/keyup, send input immediately.
- While moving, resend input at 20 messages per second.
- Store each sent input in `pendingInputs`.
- Record local send timestamps and enough timing data to replay predicted movement client-side.
- Keep zero-duration idle inputs in `pendingInputs` as stop sequence boundaries until the server confirms they were simulated.

Frame loop:

1. Read current movement state.
2. Advance the local predicted player immediately using the same movement rules as the server.
3. Render the local player from `renderedSelf`, eased toward `predictedSelf`.
4. Render remote players with the existing interpolation behavior.

## Reconciliation

When a server snapshot arrives:

1. Find the local player snapshot.
2. Drop all pending inputs with `seq <= lastProcessedSeq`.
3. Reset a temporary player state to the server position.
4. Replay remaining pending inputs.
5. The result becomes the corrected `predictedSelf`.
6. Compare corrected `predictedSelf` to current `renderedSelf`.
7. Apply the correction policy below.

Do not visibly rewind the player to the server position and then replay. The rewind/replay is internal only.

## Correction Policy

Use distance between `renderedSelf` and corrected `predictedSelf`.

```text
0px - 2px:
  Ignore the correction.

2px - 16px within 200ms after key release:
  Ease renderedSelf toward corrected predictedSelf slowly over about 200-300ms.

2px - 48px outside the release grace case:
  Ease renderedSelf toward corrected predictedSelf normally over several frames.

More than 48px:
  Snap renderedSelf to corrected predictedSelf.
```

This keeps normal LAN corrections invisible while still recovering from real desyncs, reconnect-like jumps, or collision mismatches.

The thresholds can be tuned after testing, but these are the initial defaults.

## Movement Rules To Mirror

The client prediction must match the server as closely as possible:

- `playerSpeed = 150`
- `playerSize = 28`
- four-direction input
- diagonal movement normalized
- axis-separated collision
- map bounds clamping
- blocked rectangle collision

The current client already hardcodes the map rectangles. For this slice, mirror the server logic against that same data. Later, load the shared `sunny-town/maps/sunny-town-v1.json` map data instead of hardcoding it.

## Failure Modes To Watch

- Rubber-banding near walls or corners means client/server collision differs.
- Slow visual drift means smoothing is too soft or errors are being ignored too aggressively.
- Small shaking while holding a key means the client and server movement speeds differ.
- Snapping while walking normally means replayed pending inputs are wrong or input acknowledgements are stale.
- Remote players should not use local prediction.

## Implementation Steps

Completed:

- Added received and simulated input sequence tracking to the Go `player` state.
- Updated `room.updateInput` to store the latest received input sequence without acknowledging it early.
- Updated `room.step` to mark the latest received sequence as simulated after applying the tick.
- Added `lastProcessedSeq` to `playerSnapshot`.
- Updated `SunnyTownPlayer` TypeScript type.
- Added local prediction state to `SunnyTownPage.vue`.
- Added pending input timestamps and release-grace correction easing.
- Added code-local prediction diagnostics for correction distance, pending count, acknowledged seq, latest sent seq, and release-grace state.
- Added frontend movement helpers:
   - movement vector
   - diagonal normalization
   - axis-separated collision
   - bounds clamping
- Use prediction for `selfId` only.
- Kept interpolation for non-self players.
- Applied correction policy on snapshots.
- Rebuilt frontend and Sunny Town service.

Follow-up:

- Load shared map/collision data from `sunny-town/maps/sunny-town-v1.json` in the frontend instead of duplicating map rectangles.
- Add developer-facing debug counters for ignored/eased/snapped corrections if movement tuning gets harder.

## Test Plan

Backend:

```powershell
go test ./...
```

Frontend:

```powershell
cd frontend
npm run build
```

Manual checks:

1. Open Sunny Town in one browser window.
2. Confirm local movement starts immediately on key press.
3. Confirm local movement stops immediately on key release.
4. Release repeatedly after short taps and long runs in open space.
5. Confirm no visible backward pull during normal release timing.
6. Walk into map boundaries and blocked rectangles.
7. Confirm large correction recovery still works near collisions.
8. Open a second browser window.
9. Confirm remote player movement remains smooth.
10. Confirm both clients still see authoritative final positions.

## Acceptance Criteria

- Local player feels immediate.
- Server remains authoritative.
- Normal LAN movement does not visibly rubber-band.
- Stop inputs are not acknowledged until they have been reflected in a server simulation tick.
- Wall and boundary collisions do not let the local player visibly pass through obstacles.
- Remote players remain smooth.
- Two-window multiplayer still works.
