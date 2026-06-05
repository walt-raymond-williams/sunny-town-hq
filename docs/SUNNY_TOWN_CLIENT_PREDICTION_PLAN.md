# Sunny Town Movement Model

Sunny Town now uses client-owned movement with server-accepted position samples.

Status: implemented. The local client moves and renders its own avatar immediately. The Sunny Town service accepts finite move samples, clamps map bounds, and broadcasts the accepted position to other clients. Remote players are interpolated from a short snapshot buffer.

## Goal

Movement should feel direct: when the user presses a key, the avatar moves; when the user releases the key, the avatar stops. Normal network lag must not rubber-band the local avatar backward.

This is acceptable because Sunny Town is not a competitive PvP game. The server still owns gameplay effects, rewards, room membership, and the accepted position other clients see.

## Protocol

Client-to-server movement messages use position samples:

```json
{
  "type": "move",
  "seq": 12,
  "x": 640,
  "y": 480,
  "facing": "down",
  "moving": true,
  "clientTimeMs": 123456
}
```

Server snapshots include the latest accepted movement sequence:

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

`lastProcessedSeq` means "accepted by the server." It is not used for local rewind/replay.

## Client Model

- The frontend calculates the local player's next position every animation frame.
- The frontend sends move samples on input changes and at a fixed interval while moving.
- Key release sends a final forced `moving: false` sample with the current local position.
- The local player renders from local state, not from server-correction easing.
- Server snapshots can initialize the local player and update metadata such as display name, avatar, and accepted sequence.
- Server snapshots do not pull the local player's rendered `x`/`y` around during normal play.
- Remote players render about 150ms behind real time so the client can interpolate between two known server snapshots instead of chasing the newest one.

## Server Model

- The server stores the last accepted position for each player.
- Out-of-order move samples are ignored by sequence number.
- Invalid numeric samples are rejected.
- Position samples are clamped to map bounds.
- The accepted position is stored directly after bounds clamping; the server does not choose a nearby collision fallback point.
- If no new samples arrive, the player remains at the last accepted position.

## Rewards And Gameplay Effects

Raw client coordinates must not trigger rewards.

The server's accepted player position is the only position used for stars, rewards, interactions, and future gameplay effects. This means a client can own its movement feel without being able to claim rewards from arbitrary raw coordinates.

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

1. Hold and release each movement key.
2. Alternate left/right quickly and confirm the avatar never continues sliding after release.
3. Move diagonally and confirm speed feels stable.
4. Walk into walls and bounds.
5. Simulate lag or packet delay and confirm the local avatar is not rubber-banded backward.
6. Open two clients and confirm remote players still move smoothly.
7. Pick up stars and confirm rewards save from the server-accepted position.

## Acceptance Criteria

- Local movement starts immediately.
- Local movement stops immediately.
- Normal lag does not cause local rubber-banding.
- Server snapshots still let other clients see the accepted player position.
- Rewards are only triggered from accepted server state.
