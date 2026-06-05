# Sunny Town Implementation Plan

This plan builds Sunny Town as a new Go realtime service that launches from the existing student Pet page.

Current status: the first movement MVP is implemented. Sunny Town now runs as a separate Go WebSocket service, Docker Compose includes the `sunny-town` container, HQ issues short-lived join tokens, the Pet page opens a dedicated Sunny Town route, and the client uses client-owned local movement with interpolation for remote players.

## Phase 0: Decisions to Lock

- Service name: `sunny-town`
- Language: Go
- Runtime boundary: separate service, separate port
- Realtime transport: WebSocket
- Entry/session API: owned by HQ Main Service
- Initial persistence model: no direct Sunny Town database writes
- Initial room model: one shared room, `sunny-town-main`
- Initial map model: one static map, `sunny-town-v1`

## Phase 1: Service Skeleton

Add a new Go command:

```text
cmd/sunny-town/
```

Implement:

- config loading from environment
- `/healthz`
- `/sunny-town/ws`
- graceful shutdown
- structured startup logs

Suggested environment:

```text
SUNNY_TOWN_PORT=18082
SUNNY_TOWN_JOIN_SECRET=local-dev-secret
SUNNY_TOWN_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:18080
```

Expected verification:

```powershell
go run ./cmd/sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz
```

## Phase 2: Join Session Issuer in HQ Main Service

Add an authenticated student endpoint to the existing HQ service:

```text
POST /api/student/sunny-town/session
```

The endpoint should:

- require a valid Keycloak bearer token
- require the `student` role
- load or create the student's local app user row
- load pet/avatar state needed by the game
- issue a short-lived signed join token
- return room, map, avatar, token, and WebSocket URL

Initial response shape:

```json
{
  "roomId": "sunny-town-main",
  "mapId": "sunny-town-v1",
  "websocketUrl": "ws://127.0.0.1:18082/sunny-town/ws",
  "joinToken": "...",
  "expiresAt": "2026-06-04T20:30:00Z",
  "avatar": {
    "id": "pet-default",
    "displayName": "Student"
  }
}
```

Implementation note: an HMAC-signed compact token is enough for local development. Keep token creation and validation small and covered by tests because this becomes a cross-service contract.

## Phase 3: Join Token Validation in Sunny Town

Sunny Town should validate the join token before accepting a player into the room.

Validation rules:

- token signature is valid
- token has not expired
- token contains a student user id or subject
- token includes the `student` role
- room id and map id are known
- display name and avatar id are read from token claims, not from WebSocket messages

The WebSocket connection can pass the token as a query parameter for the first version:

```text
ws://127.0.0.1:18082/sunny-town/ws?token=...
```

If the app later needs stricter handling, move the token to the `Sec-WebSocket-Protocol` header or an initial authentication message.

## Phase 4: Room Runtime

Implement an in-memory room manager.

Core structs:

```text
Server
RoomManager
Room
Player
Client
InputState
Snapshot
```

Room responsibilities:

- hold connected players
- assign spawn positions
- accept player movement samples
- run a fixed tick loop
- accept/clamp movement samples
- broadcast snapshots
- remove disconnected players

Initial movement rules:

- grid-independent pixel coordinates
- constant movement speed
- four-direction movement
- no diagonal speed advantage
- client owns local movement feel
- server accepts move samples and clamps positions to map bounds
- server stores accepted positions directly instead of choosing a nearby collision fallback point
- rewards and interactions use only server-accepted positions

## Phase 5: Static Map Contract

Add a static map file:

```text
sunny-town/maps/sunny-town-v1.json
```

Minimum shape:

```json
{
  "id": "sunny-town-v1",
  "name": "Sunny Town",
  "tileSize": 32,
  "width": 40,
  "height": 30,
  "spawns": [{ "x": 640, "y": 480 }],
  "blockedRects": [
    { "x": 0, "y": 0, "width": 1280, "height": 32 }
  ]
}
```

The server should load collision and spawn data from this file. The frontend can start with a matching hard-coded render or load the same file later.

## Phase 6: Frontend Entry from Pet Page

Add a Sunny Town action to the student Pet page.

Frontend work:

- add a button on `StudentPetTab.vue`
- add a route for Sunny Town, likely `/student/pet/sunny-town`
- create a Sunny Town view component
- call the session endpoint when the view opens
- connect to the WebSocket
- render the map and avatars
- send movement samples from keyboard or touch controls
- close the socket when leaving the route

Rendering can start with HTML canvas. Use a simple top-down map first:

- grass/base background
- paths or plaza areas
- blocking edges/buildings
- local player avatar
- remote player avatars with names

Do not overbuild art assets in the first implementation. The first goal is to validate movement, presence, and the service boundary.

## Phase 7: Protocol Messages

Start with JSON messages.

Client-to-server:

```json
{ "type": "move", "seq": 1, "x": 640, "y": 480, "facing": "down", "moving": true }
{ "type": "emote", "emote": "wave" }
{ "type": "pong", "serverTimeMs": 123456 }
```

Server-to-client:

```json
{ "type": "hello", "selfId": "123", "roomId": "sunny-town-main", "mapId": "sunny-town-v1" }
{ "type": "snapshot", "tick": 10, "players": [{ "id": "123", "lastProcessedSeq": 12 }] }
{ "type": "playerJoined", "playerId": "456" }
{ "type": "playerLeft", "playerId": "456" }
{ "type": "ping", "serverTimeMs": 123456 }
{ "type": "error", "code": "invalid_message" }
```

Rules:

- clients send movement samples for their own avatar
- server snapshots contain server-accepted positions
- server snapshots include `lastProcessedSeq` so the local client can see which sample was accepted
- local rendering is not corrected back to snapshots during normal play
- rewards and gameplay effects use only server-accepted positions, never raw client coordinates
- unknown message types are ignored or rejected with a small error
- message size is capped

## Phase 8: Local Development Wiring

Update docs and scripts after the first service is runnable.

Expected local startup:

```powershell
docker compose -f deploy/docker-compose.yml up -d

$env:DATABASE_URL="postgres://hq:hq@localhost:55432/hq?sslmode=disable"
$env:HQ_PORT="18080"
$env:KEYCLOAK_ISSUER="http://127.0.0.1:18081/realms/hq"
$env:KEYCLOAK_AUDIENCE="hq-web"
$env:SUNNY_TOWN_WS_URL="ws://127.0.0.1:18082/sunny-town/ws"
$env:SUNNY_TOWN_JOIN_SECRET="local-dev-secret"
go run ./cmd/hq

$env:SUNNY_TOWN_PORT="18082"
$env:SUNNY_TOWN_JOIN_SECRET="local-dev-secret"
go run ./cmd/sunny-town
```

Later, add a `sunny-town` service to `deploy/docker-compose.yml` once the command is stable.

Current Docker status: `deploy/docker-compose.yml` includes `sunny-town`, built from `deploy/sunny-town/Dockerfile`, and exposes host port `18082`.

## Phase 9: Tests

Backend tests:

- join token signing and validation
- expired token rejection
- wrong signature rejection
- student role requirement
- room join and leave
- input validation
- movement bounds
- bounds clamping

Frontend checks:

- Sunny Town button appears on Pet page
- session endpoint errors are shown cleanly
- WebSocket disconnect shows a reconnect or exit state
- route cleanup closes the socket

Manual multiplayer verification:

1. Start HQ, Keycloak, Postgres, and Sunny Town.
2. Open two browser windows with student login.
3. Enter Sunny Town in both.
4. Move one avatar and verify the other window sees it.
5. Close one window and verify the other sees the player leave.

## Phase 10: Later Enhancements

After the first version works:

- reconnect tokens
- map transitions
- NPCs
- simple quests
- emotes
- chat with moderation controls
- reward events back to HQ
- multiple rooms
- room assignment service
- Redis or NATS for cross-process events
- protobuf binary protocol if JSON becomes too chatty
- mobile touch joystick
- richer avatar cosmetics

## First Implementation Milestone

The first milestone is complete when:

- `cmd/sunny-town` runs separately from `cmd/hq`
- HQ issues Sunny Town join sessions for authenticated students
- Sunny Town accepts WebSocket joins with short-lived tokens
- two browser windows can see each other move on one shared map
- leaving the Pet/Sunny Town route closes the connection cleanly
- no permanent student or pet state is written by Sunny Town directly

This milestone is complete. The next implementation slice should load shared map data into the frontend instead of duplicating map rectangles in the Vue component.
