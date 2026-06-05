# Sunny Town Architecture

Sunny Town is a planned multiplayer game mode launched from the student Pet page. It is a shared overhead 2D town, similar in feel to an old Zelda-style map, where students can move their pet/avatar around the same world and see nearby players in real time.

Sunny Town should be implemented as a separate realtime service from the main HQ app. The main HQ service remains the owner of account identity, student roles, pet state, cookies, assignments, rewards, and durable persistence. Sunny Town owns live world state, player movement, rooms, and realtime fan-out.

## Goals

- Launch Sunny Town from the existing student Pet page.
- Let multiple authenticated students join a shared 2D map.
- Show each student's avatar/pet moving around the map in near real time.
- Keep movement responsive while making the server authoritative.
- Keep durable student and pet data in the existing HQ service.
- Let Sunny Town grow into a larger game without forcing all realtime concerns into the main app server.

## Non-Goals for the First Version

- No public internet hosting requirement.
- No combat, trading, moderation tools, or complex inventory.
- No persistent open-world simulation while nobody is connected.
- No cross-server sharding at launch.
- No hard dependency on Redis, NATS, or a message queue for the first single-node version.
- No direct writes from Sunny Town to homework or account tables.

## Service Boundary

```text
Browser / Vue Frontend
  Student Pet page
  Sunny Town canvas/view
  Connect/HTTP client for entry
  WebSocket client for live gameplay

        |
        | 1. CreateSunnyTownSession
        v

HQ Main Service
  Keycloak token validation
  student role checks
  pet/avatar profile lookup
  short-lived Sunny Town join token issuer
  durable rewards and pet state

        |
        | 2. WebSocket connect with join token
        v

Sunny Town Service
  WebSocket endpoint
  join-token validation
  in-memory room state
  server tick loop
  movement samples and presence
  presence and snapshots
  gameplay event emission
```

The core ownership rule is:

```text
Sunny Town owns live world state.
HQ Main Service owns durable account and pet state.
```

## Runtime Shape

The first implementation should add a new Go command:

```text
cmd/sunny-town/
```

Suggested local ports:

```text
HQ Main Service:      http://localhost:18080
Sunny Town Service:  http://localhost:18082
Keycloak:            http://localhost:18081
PostgreSQL:          localhost:55432
```

The Sunny Town service should be runnable as a local Go process during development and later as a separate container. It should expose:

```text
GET /healthz
GET /sunny-town/ws
```

The main HQ service should expose a normal authenticated entry endpoint, using either Connect RPC or JSON HTTP:

```text
POST /api/student/sunny-town/session
```

or:

```text
POST /hq.sunnytown.v1.SunnyTownEntryService/CreateSession
```

Connect RPC is a good fit for the entry/session endpoint because it is request/response application logic. WebSocket is the better fit for the movement and presence channel because Sunny Town needs low-latency bidirectional updates.

## Entry Flow

```text
1. Student opens the Pet page.
2. Student clicks Sunny Town.
3. Frontend calls the HQ Main Service with the normal bearer token.
4. HQ validates the Keycloak token and requires the student role.
5. HQ loads the student profile and pet/avatar state.
6. HQ returns a short-lived Sunny Town session:
   - room_id
   - map_id
   - avatar appearance
   - websocket_url
   - join_token
   - expires_at
7. Frontend opens the Sunny Town WebSocket with the join token.
8. Sunny Town validates the join token.
9. Sunny Town places the player in the room and starts sending snapshots.
```

The join token should be short lived, signed by HQ, and scoped to Sunny Town. A simple HMAC-signed token is enough for the local-network first version. JWT is also acceptable if the project already wants a standard claims format.

Recommended join-token claims:

```json
{
  "sub": "keycloak-user-subject",
  "app_user_id": 123,
  "display_name": "Student",
  "roles": ["student"],
  "room_id": "sunny-town-main",
  "map_id": "sunny-town-v1",
  "avatar_id": "pet-default",
  "exp": 1780617600
}
```

The Sunny Town service should reject expired tokens, tokens without the `student` role, tokens for unknown rooms, and tokens signed with the wrong secret.

## Realtime Protocol

Use WebSocket for the live session.

Client-to-server movement messages should represent the client's current avatar position. They are accepted into server state after basic validation, but raw message coordinates never trigger rewards directly:

```json
{ "type": "move", "seq": 42, "x": 640, "y": 480, "facing": "down", "moving": true }
{ "type": "emote", "emote": "wave" }
{ "type": "ping", "client_time_ms": 123456 }
```

Server-to-client messages should represent authoritative state:

```json
{
  "type": "snapshot",
  "serverTimeMs": 123456,
  "tick": 991,
  "selfId": "123",
  "players": [
    {
      "id": "123",
      "displayName": "Student",
      "x": 128,
      "y": 96,
      "facing": "down",
      "moving": true,
      "avatarId": "pet-default",
      "lastProcessedSeq": 42
    }
  ]
}
```

The `lastProcessedSeq` value tells the client which movement sample the server has accepted. The first version uses JSON messages for simplicity. If message volume becomes a problem later, the protocol can move to protobuf binary messages without changing the service boundary.

## Simulation Model

The server should be authoritative over:

- room membership
- spawn points
- map boundaries
- interaction eligibility
- accepted player position used by gameplay effects

The client owns local avatar movement feel. It simulates its current player immediately, sends position samples to Sunny Town, and renders the local player from that local state. The server accepts finite samples, clamps map bounds, and stores the accepted position. Rewards and other gameplay effects run only from the server-accepted position, never from raw client coordinates.

Server snapshots do not correct the local player's rendered `x`/`y` during normal play. They are used to update server-accepted metadata and to show the player to other clients. Remote players are interpolated between server snapshots.

Initial tick settings:

```text
Simulation tick:       20 ticks/second
Snapshot broadcast:    10-20 snapshots/second
Move sample send rate: on movement change plus repeat at 10-20/second while moving
Idle timeout:          60 seconds without pong or input
```

For the first version, one in-memory room is enough:

```text
room_id: sunny-town-main
map_id:  sunny-town-v1
```

## Map and Collision

The first map should be static and checked into the repo. The map can start as a JSON file that the Sunny Town service and frontend both understand.

Suggested location:

```text
sunny-town/maps/sunny-town-v1.json
```

The map should define:

- dimensions in tiles
- tile size in pixels
- spawn points
- blocked rectangles or blocked tile ids for client-side movement feel
- decorative layers for the client
- interactive zones for later features

The server only needs bounds, spawn, and reward data for the current movement model. The client owns rendering detail and local collision feel.

## Persistence and Events

Sunny Town should not write directly to the core HQ tables in the first version. Instead, it should emit or report durable events back to the main HQ service when needed.

First-version events may be handled with direct HTTP/Connect calls from Sunny Town to HQ:

```text
player_entered_town
player_left_town
emote_used
interaction_completed
reward_earned
```

Later, if the game grows, these events can move to a queue such as NATS or Redis Streams. That should wait until there is a real need for buffering, replay, or multi-process fan-out.

## Scaling Path

Version 1 should run as one Sunny Town process with in-memory room state.

The next scaling steps are:

1. Multiple rooms in one process.
2. Sticky routing by room id.
3. Multiple Sunny Town processes with each room assigned to one process.
4. Presence and event fan-out through Redis or NATS.
5. Region or shard assignment if the game becomes public internet hosted.

Do not add distributed state before the single-process model is working and measured.

## Security

- Students must authenticate through the existing HQ frontend flow.
- HQ Main Service validates Keycloak access tokens.
- Sunny Town should accept only short-lived HQ-issued join tokens.
- Join tokens should expire quickly, for example after 60 seconds.
- WebSocket connections should have connection limits and message size limits.
- The server should rate-limit input messages per connection.
- The server must ignore client-provided coordinates.
- Display names should come from trusted token/session data, not client messages.

## Observability

Sunny Town should log:

- service startup config
- room creation
- player join and leave
- WebSocket authentication failures
- unexpected disconnects
- message decode errors with safe metadata only

Useful counters:

- active connections
- active rooms
- players per room
- move messages per second
- snapshots sent per second
- invalid move samples
- disconnect reasons
- tick duration

## Initial Decision Summary

- Build Sunny Town as a separate Go service.
- Use WebSocket for realtime gameplay.
- Use Connect RPC or JSON HTTP on the main HQ service for session creation.
- Keep the first world single-node and in-memory.
- Keep durable account, pet, and reward state in the main HQ service.
- Add queues, Redis, or sharding only after the single-process design has proven it needs them.
