# Sunny Town Forest Crossing and Mining Plan

This document records the planned Sunny Town expansion slice: add a connected Forest Crossing map with mineable resource nodes and durable resource persistence.

Status: planning approved for documentation. Code implementation has not started in this document.

## Current Architecture Findings

Sunny Town already has a multi-map structure. The product language for this feature can say "cell", but the codebase currently calls these areas `maps`.

Current map files live in:

```text
sunny-town/maps/
```

Existing maps:

- `sunny-town-v1`
- `sunny-town-house-1`
- `sunny-town-classroom`

The Sunny Town service loads every map JSON file into a `gameMap`. Each loaded map gets a separate in-memory `room`, while the logical shared Sunny Town room id remains `sunny-town-main`.

Current transition support already exists through map `portals`. A portal is a rectangle with a target map id, target coordinates, and target facing. When the server-accepted player position overlaps a portal, Sunny Town transfers the player to the target map and sends a `map_changed` WebSocket message.

Presence is already scoped per map. Players in `sunny-town-v1`, `sunny-town-house-1`, and `sunny-town-classroom` do not see each other unless they are in the same current map.

Movement is client-owned for smooth local feel. The browser moves and renders the local player immediately, while Sunny Town stores server-accepted position samples. Gameplay effects such as star pickup, transitions, and future mining should use only the server-accepted position.

## Current Durable State Model

HQ Main Service owns durable student/account/economy/inventory state.

Sunny Town owns live world state:

- connected players
- current map membership
- accepted player positions
- live collectible/node state
- transition detection
- gameplay validation
- WebSocket fan-out

Sunny Town does not directly write the HQ database. It calls service-authenticated internal HQ endpoints using `X-HQ-Service-Secret`.

## Stars, Inventory, and Ledgers

Stars are wallet currency, not inventory.

Current star tables:

- `student_wallet`: current star balance
- `student_star_ledger`: audit and idempotency records for star changes

Inventory items are stored separately.

Current inventory tables:

- `inventory_item_type`: item catalog
- `student_inventory_item`: current per-student item quantity
- `student_equipped_item`: equipment slots backed by inventory ownership

Cookies currently live in `student_inventory_item`. Cookie changes are direct HQ operations:

- shop purchase increments cookie inventory
- pet feeding decrements cookie inventory

There is no current cookie ledger. That is acceptable because those flows are handled directly inside HQ.

Mining is different because Sunny Town is a separate realtime service reporting a reward event to HQ over HTTP. If Sunny Town sends a resource event and then retries after a timeout, HQ must not award the same resource twice.

For mining resources:

- Current quantity should live in `student_inventory_item`.
- `rock` and `crystal` should be seeded in `inventory_item_type`.
- A new inventory/resource ledger should record resource event ids and prevent duplicate awards.

The ledger is not the inventory store. It is the receipt trail and duplicate-protection layer for externally reported resource events.

## Feature Slice

Add the first new outdoor expansion map:

```text
map id: forest-crossing-v1
display name: Forest Crossing
```

The map should contain:

- trees
- rocks
- river
- bridge
- at least two mineable rock nodes

This slice does not include:

- combat
- crafting
- a full store
- cosmetic equipment beyond the existing equipment system
- new tool systems beyond the existing pickaxe support

## Cell and Transition Plan

Use existing map/portal conventions instead of adding a parallel `cell` abstraction.

Add a portal from `sunny-town-v1` to `forest-crossing-v1`.

Add a return portal from `forest-crossing-v1` to `sunny-town-v1`.

Transition behavior should continue to use the existing flow:

1. Client sends movement samples.
2. Sunny Town clamps and accepts the latest position.
3. Sunny Town checks portal overlap using the accepted player rectangle.
4. Sunny Town moves the player between map rooms.
5. Sunny Town sends `map_changed` with the new map, players, collectibles, and resource nodes.
6. Future snapshots come only from the new map room.

## Mining Node Model

Extend the map JSON contract with resource nodes.

Recommended field name:

```json
"resourceNodes": []
```

Recommended node shape:

```json
{
  "id": "rock-node-001",
  "kind": "rock",
  "x": 320,
  "y": 448,
  "radius": 24,
  "interactionRadius": 48,
  "respawnSeconds": 15
}
```

Sunny Town should create live node state from map definitions:

- stable node id
- kind
- map id
- position
- radius
- interaction radius
- active/depleted state
- respawn time
- harvest sequence number

The live node state should remain in Sunny Town memory. Durable resource ownership should remain in HQ.

## Mining Interaction Plan

Use the existing `tool_use` WebSocket message instead of creating a separate browser-side `give_resource` or reward-claim message.

Current client behavior already supports:

- `F` as primary interaction
- NPC interactions first
- equipped tool use second
- local pickaxe swing animation
- `tool_use` WebSocket message

Mining should be added behind this server-authoritative tool-use handler.

Sunny Town should validate:

- player is connected and in a room
- requested tool is equipped
- tool is `pickaxe`
- node exists in the player's current map
- node is active
- player is close enough to the node
- interaction is not being spammed

Because the codebase already seeds a starter `pickaxe` and has a `tool` equipment slot, this slice should require the equipped pickaxe rather than inventing tool-less mining.

## Drop Rules

Initial mining drop table:

```text
85% -> 1 rock
10% -> 2 rock
5%  -> 1 crystal
```

The drop roll happens in Sunny Town after server validation.

After a successful mine:

1. Sunny Town increments the node harvest sequence.
2. Sunny Town marks the node inactive.
3. Sunny Town schedules node respawn.
4. Sunny Town creates an idempotent resource event id.
5. Sunny Town calls HQ to commit the resource event.
6. HQ persists the resource if the event id is new.
7. Sunny Town notifies the client after HQ confirms.

Recommended event id format:

```text
forest-crossing-v1:rock-node-001:7:123
```

Where:

- `forest-crossing-v1` is the map id
- `rock-node-001` is the node id
- `7` is the node harvest sequence
- `123` is the app user id

## HQ Resource Persistence Plan

Use existing inventory tables for current resource quantities.

Seed item types:

- `rock`
- `crystal`

Add a ledger table for idempotency and audit. Recommended name:

```text
student_inventory_ledger
```

Recommended shape:

```sql
create table if not exists student_inventory_ledger (
    id bigserial primary key,
    app_user_id bigint not null references app_user(id) on delete cascade,
    event_id text not null unique,
    source text not null,
    item_type_id bigint not null references inventory_item_type(id) on delete restrict,
    delta integer not null,
    room_id text null,
    map_id text null,
    node_id text null,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    constraint student_inventory_ledger_delta_nonzero check (delta <> 0)
);
```

For a one-resource event, HQ should:

1. Insert the ledger row with `on conflict (event_id) do nothing`.
2. Increment `student_inventory_item` only if the ledger insert happened.
3. Return whether the event was a duplicate.
4. Return the updated item quantity.

If later events can contain multiple resources, either:

- create one ledger row per item using a unique event id suffix per item, or
- add an `event_group_id` plus uniqueness on `(event_group_id, item_type_id)`.

For this MVP, one drop result per mining action is enough.

## Internal HQ API Plan

Add a service-authenticated endpoint:

```text
POST /api/internal/sunny-town/resource-events
```

The browser must not call this endpoint.

Request:

```json
{
  "event_id": "forest-crossing-v1:rock-node-001:7:123",
  "app_user_id": 123,
  "source": "sunny_town_mining",
  "room_id": "sunny-town-main",
  "map_id": "forest-crossing-v1",
  "node_id": "rock-node-001",
  "resource_key": "rock",
  "amount": 1
}
```

Response:

```json
{
  "accepted": true,
  "duplicate": false,
  "resource_key": "rock",
  "quantity": 12
}
```

## WebSocket Message Plan

Extend snapshots with resource nodes:

```json
{
  "type": "snapshot",
  "mapId": "forest-crossing-v1",
  "players": [],
  "collectibles": [],
  "resourceNodes": [
    {
      "id": "rock-node-001",
      "kind": "rock",
      "x": 320,
      "y": 448,
      "radius": 24,
      "active": true
    }
  ]
}
```

Use existing client tool-use message:

```json
{
  "type": "tool_use",
  "toolKey": "pickaxe",
  "x": 320,
  "y": 416,
  "facing": "down",
  "clientTimeMs": 1760000000000
}
```

Sunny Town can either select the nearest active mineable node in range or extend this message with `targetId`. For the first patch, nearest-in-range is simpler and avoids adding client targeting state.

After HQ confirms persistence, send:

```json
{
  "type": "resource_committed",
  "eventId": "forest-crossing-v1:rock-node-001:7:123",
  "nodeId": "rock-node-001",
  "resourceKey": "rock",
  "amount": 1,
  "quantity": 12
}
```

On failure:

```json
{
  "type": "resource_failed",
  "nodeId": "rock-node-001",
  "reason": "temporary_error"
}
```

## Frontend Plan

Extend Sunny Town frontend types:

- map `resourceNodes`
- snapshot `resourceNodes`
- resource node state
- `resource_committed`
- `resource_failed`

Extend canvas rendering:

- Forest Crossing background
- river
- bridge
- trees/rocks from existing primitive drawing
- active/depleted mining nodes

Interaction flow:

1. Player presses `F`.
2. Existing NPC interaction gets priority.
3. If no NPC interaction applies, use equipped tool.
4. If equipped tool is pickaxe, send `tool_use`.
5. Sunny Town validates and commits mining.
6. Client shows a resource toast after `resource_committed`.
7. Client reloads or patches inventory with the confirmed resource quantity.

The existing inventory overlay should naturally show `rock` and `crystal` after they have positive quantities because it already renders `student_inventory_item` rows.

## Files Likely to Change

Sunny Town service:

- `cmd/sunny-town/main.go`
- `cmd/sunny-town/room_test.go`

HQ service:

- `cmd/hq/main.go`
- `cmd/hq/schema.go`
- `cmd/hq/inventory.go`
- `cmd/hq/reward_test.go`

Maps:

- `sunny-town/maps/sunny-town-v1.json`
- `sunny-town/maps/forest-crossing-v1.json`

Frontend:

- `frontend/src/types/sunnyTown.ts`
- `frontend/src/pages/SunnyTownPage.vue`
- `frontend/src/stores/studentInventory.ts`, if patching inventory locally after resource commits

Docs:

- `docs/INVENTORY_AND_EQUIPMENT.md`, to mention resource ledger behavior after implementation
- this document, if implementation details change during coding

## Tests to Add or Update

Sunny Town tests:

- checked-in maps load successfully with `resourceNodes`
- duplicate resource node ids are rejected
- invalid resource node definitions are rejected
- portal transition to Forest Crossing works
- mining fails when too far from node
- mining fails when node is inactive
- mining fails when pickaxe is not equipped
- mining succeeds when player is in range with pickaxe
- node becomes inactive after mining
- node respawns after cooldown

HQ tests:

- resource event inserts ledger row and increments inventory
- duplicate resource event does not double-award
- invalid resource key is rejected
- invalid amount is rejected
- endpoint requires service secret

Frontend verification:

- enter Sunny Town
- walk through Forest Crossing portal
- return through Forest Crossing portal
- see mineable nodes
- press `F` near a node with pickaxe equipped
- see resource toast
- see depleted node
- see resource quantity in inventory after commit

## Minimal First Patch Plan

1. Add `forest-crossing-v1` map JSON and portal wiring.
2. Add `resourceNodes` to Sunny Town map loading and snapshots.
3. Add in-memory resource node state, mining validation, drop rolling, and respawn.
4. Add HQ inventory ledger and internal resource event endpoint.
5. Seed `rock` and `crystal` in the inventory catalog.
6. Extend frontend rendering and resource commit handling.
7. Add focused Go tests.
8. Run relevant backend and frontend checks.

## Risks and Unknowns

- Current HQ tests use a custom test schema setup. New ledger tables and seed items must be added there too, not only in runtime schema setup.
- The frontend currently hard-codes map rendering by map id. Forest Crossing can follow that pattern for MVP, but future maps may need a more data-driven scenery layer.
- Current `tool_use` does not include a target id. Nearest-in-range mining is fine for MVP, but dense future interactions may need explicit targeting.
- Sunny Town currently has one generic reward worker for stars. Resource commits can reuse the retry pattern, but the event structs should stay clear so star and resource commits do not become ambiguous.
- Resource node depletion is in-memory. If Sunny Town restarts, nodes reset. That is acceptable for this slice.

## Follow-Up Work

These are intentionally not part of the first Forest Crossing mining patch.

### Data-Driven Map Scenery

Add optional map JSON layers for decorative scenery so new maps do not require frontend `if map.id` drawing branches.

Possible fields:

- terrain rectangles
- water rectangles
- bridge rectangles
- decorative sprites
- collision categories

### Explicit Interaction Message

Introduce a generic interaction message after there are multiple non-tool interactions:

```json
{
  "type": "interact",
  "seq": 81,
  "interaction": "mine",
  "targetId": "rock-node-001"
}
```

For now, `tool_use` is sufficient because pickaxe use already exists.

### Multi-Drop Resource Events

If future nodes can drop multiple resources at once, extend the HQ resource API to support a `drops` array and use either:

- one ledger row per item with unique item-specific event ids, or
- one event group id with uniqueness per resource key.

### Inventory History UI

Expose ledger history to a student or parent-facing UI only if useful. The ledger is primarily a backend integrity mechanism, not a user-facing feature.

### Resource Uses

Later slices can spend or use resources through HQ-owned inventory operations:

- crafting
- shop exchanges
- classroom rewards
- decorations

Those should consume `student_inventory_item` through HQ APIs and should use ledger/idempotency if an external service reports the event.

### Richer Tool Rules

Future mining can require:

- equipped pickaxe tier
- durability
- stamina or cooldown
- per-node tool requirements

Do not add these in the first patch.

### Persistence of World Node State

If node depletion needs to survive Sunny Town restarts, add a world-state persistence layer later. For the first slice, temporary in-memory respawn state is enough.
