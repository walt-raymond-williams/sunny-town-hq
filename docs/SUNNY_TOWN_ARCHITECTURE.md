# Sunny Town Architecture

Sunny Town is the realtime multiplayer game surface launched from the student Pet page. It is implemented as a separate Go WebSocket service with in-memory live world state. HQ remains the owner of identity, durable student state, wallet stars, inventory, equipment, and persisted Sunny Town return position.

## Service Boundary

```text
Browser / Vue Frontend
  Sunny Town canvas
  keyboard/pointer input
  inventory, shop, and schoolwork overlays
  WebSocket gameplay client

        |
        | POST /api/student/sunny-town/session
        v

HQ Service
  Keycloak bearer-token validation
  student role checks
  join-token issuer
  wallet, inventory, equipment, map-edit, and position persistence
  service-authenticated internal Sunny Town endpoints

        |
        | GET /sunny-town/ws?token=...
        v

Sunny Town Service
  WebSocket authentication
  map-backed in-memory rooms
  accepted movement state
  portals, collectibles, resource nodes, and gameplay validation
  snapshot fan-out
```

Core ownership rule:

```text
Sunny Town owns live realtime world state.
HQ owns durable account, student, wallet, inventory, equipment, map-edit, and persisted return-position state.
```

Sunny Town does not write the HQ database directly. It calls HQ internal HTTP endpoints with `X-HQ-Service-Secret`.

## Runtime

Current commands and default local ports:

```text
cmd/hq           http://localhost:18080 in Docker Compose, 8080 by Go default
cmd/sunny-town   http://localhost:18082
Keycloak         http://localhost:18081
PostgreSQL       localhost:55432
```

Sunny Town exposes:

```text
GET /healthz
GET /sunny-town/ws
```

HQ exposes the authenticated student entry endpoint:

```text
POST /api/student/sunny-town/session
```

The Docker Compose workflow builds `hq` and `sunny-town` as separate containers. The two services must share `SUNNY_TOWN_JOIN_SECRET` and `SUNNY_TOWN_SERVICE_SECRET`. HQ returns the browser-reachable WebSocket URL from `SUNNY_TOWN_WS_URL`.

## Entry Flow

1. The student opens Sunny Town from the Pet page.
2. The frontend calls `POST /api/student/sunny-town/session` with the normal Keycloak bearer token.
3. HQ validates the token and requires the `student` role.
4. HQ loads the student profile, star wallet, and last saved Sunny Town position.
5. HQ signs a short-lived Sunny Town join token scoped to the room and current map.
6. The frontend opens `/sunny-town/ws?token=...`.
7. Sunny Town validates the join token, loads equipment from HQ, loads saved position from HQ, and joins the player to the target map room.
8. Sunny Town sends `hello`, then periodic `snapshot` messages.

The session response includes:

```json
{
  "room_id": "sunny-town-main",
  "map_id": "sunny-town-v1",
  "avatar_id": "pet-default",
  "websocket_url": "ws://127.0.0.1:18082/sunny-town/ws",
  "join_token": "...",
  "expires_at": "2026-06-06T18:30:00Z",
  "wallet": {
    "star_balance": 12
  }
}
```

Join tokens are HMAC-signed by HQ and validated by Sunny Town through `internal/sunnytownauth`. They include the app user id, Keycloak subject, display name, roles, room id, map id, avatar id, and expiry.

## Maps and Rooms

Map JSON files live in:

```text
sunny-town/maps/
```

Current checked-in maps:

- `sunny-town-v1`
- `sunny-town-house-1`
- `sunny-town-classroom`
- `forest-crossing-v1`

Sunny Town loads every JSON map at startup. Each map becomes one in-memory room, while the logical shared room id remains `sunny-town-main`.

Map fields currently used by the server and client:

- `id`, `name`, `tileSize`, `width`, `height`
- `spawns`
- `blockedRects`
- `starSpawns`
- `portals`
- `npcs`
- `resourceNodes`

Portals transfer a player between map rooms after overlap is detected from the server-accepted player rectangle. Transfers send a `map_changed` message containing the new map, players, collectibles, and resource nodes. Presence is scoped per map, so players in different maps do not appear in each other's snapshots.

The portal target coordinates should not place the player inside the destination portal trigger. The server also keeps a portal re-entry guard so the player must leave a portal before triggering another transfer.

## Movement

Sunny Town uses client-owned movement feel with server-accepted position samples.

Client-to-server movement:

```json
{
  "type": "move",
  "seq": 42,
  "x": 640,
  "y": 480,
  "facing": "down",
  "moving": true,
  "clientTimeMs": 1760000000000
}
```

Server snapshots:

```json
{
  "type": "snapshot",
  "serverTimeMs": 1760000000000,
  "tick": 991,
  "mapId": "sunny-town-v1",
  "players": [
    {
      "id": "player-123",
      "displayName": "Student",
      "x": 640,
      "y": 480,
      "facing": "down",
      "moving": true,
      "avatarId": "pet-default",
      "equipment": {
        "gear": "sunny_hoodie",
        "accessory": "star_cap",
        "tool": "pickaxe"
      },
      "lastProcessedSeq": 42
    }
  ]
}
```

The client renders the local player from local prediction and renders remote players from interpolated snapshots. The server ignores out-of-order movement samples, rejects invalid numeric coordinates, clamps map bounds, and uses only accepted server state for gameplay effects.

See [SUNNY_TOWN_MOVEMENT_MODEL.md](./SUNNY_TOWN_MOVEMENT_MODEL.md) for the detailed movement model.

## Collision

The frontend uses `blockedRects` and placed map objects for local movement feel. The server also enforces both static `blockedRects` and live placed objects against accepted player positions. If a proposed movement sample would overlap collision geometry, the player remains at the previous accepted position.

## Placed Map Objects

Players can place crafted `stone_block` items into the map grid. The browser sends a grid coordinate, Sunny Town validates the request against map bounds, static blocked rectangles, portals, NPCs, resource nodes, players, and existing placed objects, then asks HQ to persist the object and consume one `stone_block`.

HQ stores placed blocks in `sunny_town_map_object` with `(room_id, map_id, grid_x, grid_y)` uniqueness. The base JSON map remains unchanged; player edits are separate durable records. Sunny Town loads persisted objects at startup and refreshes the target map's object list when a player joins, so a restarted service can reconstruct the edited map state from HQ.

Placed objects are included in `hello`, `snapshot`, and `map_changed` messages:

```json
{
  "placedObjects": [
    {
      "id": "12",
      "itemKey": "stone_block",
      "gridX": 20,
      "gridY": 14,
      "x": 640,
      "y": 448,
      "width": 32,
      "height": 32
    }
  ]
}
```

Sunny Town also broadcasts immediate `map_object_placed` and `map_object_removed` events so other connected players see edits without waiting for the next snapshot. Inventory quantities from those mutations are sent only to the acting player.

## Collectibles and Stars

Maps can define `starSpawns`. Sunny Town creates in-memory star collectibles from those spawn points.

When the server-accepted player position is within pickup radius:

1. Sunny Town marks the collectible inactive.
2. Sunny Town increments that collectible's spawn sequence and schedules a respawn.
3. Sunny Town sends an idempotent reward event to HQ.
4. HQ inserts into `student_star_ledger` with unique `event_id` protection.
5. HQ updates `student_wallet`.
6. Sunny Town sends `reward_committed` or `reward_failed` to the client.

Stars are wallet currency, not inventory.

## Equipment

HQ is the source of truth for equipment. Sunny Town loads equipment through:

```text
GET /api/internal/sunny-town/student-equipment?app_user_id=...
```

The endpoint is protected by `X-HQ-Service-Secret`.

When the student changes equipment in Sunny Town, the frontend first updates HQ through the student equipment API, then sends:

```json
{ "type": "equipment_changed" }
```

Sunny Town reloads equipment from HQ and includes visual equipment keys in future snapshots.

## NPCs, Shops, and Schoolwork

Maps can include `npcs`. NPCs can be dialogue-only, shop-backed, or activity-backed.

Current activity type:

```json
{ "type": "schoolwork" }
```

Schoolwork NPCs open the existing student assignment flow from inside Sunny Town. Shop NPCs use the HQ shop API and currently support buying cookies with wallet stars.

## Mining and Resources

`forest-crossing-v1` contains mineable rock resource nodes. Resource node definitions are loaded from map JSON:

```json
{
  "id": "rock-node-001",
  "kind": "rock",
  "x": 704,
  "y": 352,
  "radius": 24,
  "interactionRadius": 58,
  "respawnSeconds": 15
}
```

The client sends the existing tool-use message:

```json
{
  "type": "tool_use",
  "toolKey": "pickaxe",
  "x": 704,
  "y": 352,
  "facing": "down",
  "clientTimeMs": 1760000000000
}
```

Sunny Town validates that the player is in a room, has the requested tool equipped, is using `pickaxe`, and is not inside the tool cooldown. Pickaxe use first checks for a nearby placed `stone_block`; if found, Sunny Town asks HQ to delete that map object and refund one `stone_block` to the player's inventory, then broadcasts the removal.

If no placed block is in range, Sunny Town checks for the nearest active resource node. Resource mining currently requires three accepted pickaxe hits. On the third hit, Sunny Town depletes the node, schedules respawn, rolls a drop, and sends an idempotent resource event to HQ.

Current drop table:

```text
85% -> 1 rock
10% -> 2 rock
5%  -> 1 crystal
```

HQ commits mining resources through:

```text
POST /api/internal/sunny-town/resource-events
```

HQ records the event in `student_inventory_ledger`, then increments `student_inventory_item` only if the event id was new. Retries with the same event id do not double-award resources. Sunny Town sends `resource_committed` or `resource_failed` after HQ responds.

Resource node active/depleted state is in Sunny Town memory. If Sunny Town restarts, nodes reset.

## Persisted Return Position

When a Sunny Town WebSocket leaves a room, Sunny Town asks HQ to save the player's last accepted map, coordinates, and facing:

```text
POST /api/internal/sunny-town/player-position
GET  /api/internal/sunny-town/player-position?app_user_id=...
```

HQ stores this in `student_sunny_town_position`. The next Sunny Town session starts on that saved map when possible.

## WebSocket Messages

Client-to-server:

```text
move
equipment_changed
tool_use
place_object
ping
```

Server-to-client:

```text
hello
snapshot
map_changed
reward_committed
reward_failed
resource_committed
resource_failed
map_object_placed
map_object_removed
error
```

Move messages are capped by a simple per-connection input rate window while moving. WebSocket read size is capped. Unknown messages receive a small error response.

## Scaling Path

The current design is intentionally single-process with in-memory map rooms.

Next scaling steps, if needed:

1. Multiple logical rooms in one Sunny Town process.
2. Sticky routing by room id or map id.
3. Multiple Sunny Town processes with each room assigned to one process.
4. Shared event fan-out through Redis or NATS.
5. Durable world-state persistence for node depletion or other map state that must survive restarts.

Do not add distributed state until the single-process model has a measured need for it.
