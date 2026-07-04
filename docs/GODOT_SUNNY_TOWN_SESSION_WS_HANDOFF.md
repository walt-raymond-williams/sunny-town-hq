# Godot Sunny Town Session WebSocket Handoff

## Purpose

This handoff starts GitHub issue #65, "Connect Godot client to Sunny Town session and WebSocket."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/65

The goal is to connect Godot to the existing HQ-created Sunny Town session and current Sunny Town WebSocket protocol.

## Current Status

- Blocked by #64.
- The Godot placeholder should load from Docker-served HQ before this starts.

## Product Direction

- Vue keeps authenticated HQ session creation.
- Godot receives session data and opens the Sunny Town WebSocket.
- Avoid backend changes unless a small additive compatibility field is clearly needed.

## Current Code Surface

Frontend:

- `frontend/src/api/sunnyTownApi.ts`
- `frontend/src/composables/useSunnyTownSocket.ts`
- `frontend/src/types/sunnyTown.ts`
- Godot wrapper created by #63.

Backend:

- `cmd/hq/handlers.go`
- `internal/sunnytown/protocol/protocol.go`
- `internal/sunnytown/server/server.go`
- `internal/sunnytown/server/client_io.go`
- `internal/sunnytown/server/world_snapshots.go`
- `internal/sunnytownauth/token.go`

## Recommended Implementation Plan

1. Reuse the existing Vue session API call.
2. Pass session payload into Godot using the mechanism chosen in #62.
3. Implement Godot protocol models for current server messages.
4. Open WebSocket with `websocket_url` and `join_token`.
5. Parse and store `hello`, `snapshot`, and `map_changed`.
6. Display debug state: self id, map id/name, tick, player count, NPC count, object count, connection status.
7. Handle errors and disconnects.
8. Define reconnect behavior that obtains a fresh HQ session instead of reusing expired join tokens.

## Out Of Scope

- Full rendering.
- Movement.
- Interactions.
- Inventory UI.

## Verification

Run:

```powershell
go test ./...
cd frontend
npm run build
```

Manual smoke:

- Log in as a student.
- Open Godot Sunny Town.
- Confirm Godot receives `hello` and periodic `snapshot`.
- Open a second client and confirm debug player counts change.
- Confirm canvas fallback still connects.

## Close Criteria

Close #65 when:

- Godot connects through the existing session/token flow.
- Current protocol messages parse without backend rewrites.
- Debug state proves live data is flowing.
- Reconnect/token behavior is documented or handled.
