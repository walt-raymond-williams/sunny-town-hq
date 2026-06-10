# Container Storage Contract

This document is the accepted current contract for Sunny Town container and chest storage. HQ durable container identity and Sunny Town-validated chest open/transfer flows are implemented for the inventory redesign branch.

## Ownership Boundary

- HQ owns durable container identity, contents, slot layout, item quantities, and mutation transactions.
- Sunny Town owns live access validation: current room, map, accepted player position, object activity, and interaction distance.
- Browsers may request container actions, but they are not authority for proximity, object identity, or access.
- Sunny Town must not write the HQ database directly. It should call service-authenticated HQ endpoints or mint an HQ-verifiable short-lived access grant after live validation.

## Stable Container Identity

Durable containers are identified by an HQ-owned `container_id`.

Implemented first schema:

```text
storage_container
  id text primary key
  kind text not null
  room_id text not null
  map_id text not null
  fixture_id text null
  placed_object_id text null
  shop_id text null
  storage_role text null
  location_id text null
  owner_app_user_id bigint null references app_user(id)
  slot_count integer not null
  access_policy text not null
  revision bigint not null default 0
  created_at timestamptz not null default now()
  updated_at timestamptz not null default now()

storage_container_slot
  container_id text not null references storage_container(id) on delete cascade
  slot_index integer not null
  item_type_id bigint null references inventory_item_type(id)
  quantity integer null
  primary key (container_id, slot_index)
```

Container IDs should be stable and deterministic where possible:

- Authored fixture chest: `fixture:<room_id>:<map_id>:<fixture_id>`
- Placed object chest: `placed:<placed_object_id>`
- Shop-owned logical storage: `shop:<shop_id>:<storage_role>`

The ID is the durable key. Fixture ID, placed object ID, room, map, shop, role, and location remain queryable metadata for validation, migration, and debugging.

## Current Fixture Mapping

Cookie Shop fixtures are the first concrete container anchors:

- `cookie-shop-output-chest` maps to `fixture:sunny-town-main:sunny-town-house-1:cookie-shop-output-chest` and represents `shop:cookie-keeper-shop:output`.
- `cookie-shop-input-chest` maps to `fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest` and represents `shop:cookie-keeper-shop:input`.

Current `shop_stock_item` and `shop_input_storage_item` tables remain the specialized storage backing for Cookie Shop stock/input until a migration intentionally moves them into general container slots. Do not duplicate quantities in fixture metadata or Sunny Town runtime state.

`cookie-shop-input-chest` is now projected through the general container WebSocket flow while retaining specialized storage: opening the chest returns slots built from `shop_input_storage_item`, and depositing `flour` or `sugar` consumes the player stack and increments `shop_input_storage_item` atomically in HQ. The generic `storage_container_slot` table is not used for Cookie Shop input quantities.

## Access Rules

Every container has an `access_policy`.

Recommended initial policies:

- `shop_public_read`: nearby students can inspect; mutation is disabled unless a narrower action grants it.
- `shop_input_deposit`: nearby students can deposit allowed ingredients into shop input storage.
- `owner_private`: only `owner_app_user_id` can inspect or mutate while nearby.
- `room_shared`: authenticated students in the same room can inspect or mutate while nearby.

For the first player-facing transfer slice, prefer the narrowest policy needed by the UI. Cookie Shop input deposit should not imply output withdrawal.

## Live Access Validation

Sunny Town validates live access before any HQ mutation:

1. Resolve the requested `worldObject` from server state, not from client payload alone.
2. Require `kind = chest`, `active = true`, and an allowed `storageRole` for the requested action.
3. Require the player to be in the same room and map as the chest.
4. Use the server-accepted player position, not client-predicted position.
5. Require distance to the chest center or bounds to be within `interactionRadius`.
6. Reject if the chest fixture or placed object no longer exists or is no longer active.

After validation, Sunny Town calls service-authenticated HQ endpoints with the validated student ID, container ID, action, and slot descriptors. Browsers send only chest object identity and slot refs through the Sunny Town WebSocket; they do not call HQ container routes directly and cannot supply an authoritative container ID.

Implemented Sunny Town WebSocket messages:

- `container_open`: validates `read` access and returns authoritative container slots.
- `container_transfer`: validates `deposit` or `withdraw` from the transfer direction, calls HQ, and returns updated player inventory slots plus updated container slots.

Current chest roles are narrow by design: `input` allows deposit, `output` is read-only for player transfers, and `general` allows both deposit and withdraw.

## Conflict Handling

HQ must serialize mutations that touch the same player inventory and container storage.

Implemented first behavior:

- Run each transfer in one database transaction.
- Lock the source and destination slot rows for update.
- Also lock the `storage_container` row or use a per-container advisory transaction lock.
- Increment `storage_container.revision` after a successful mutation.
- Return the updated player slots and container slots from the committed transaction.
- If a source slot changed before the transaction lock, return a normal client error such as `source slot is empty`, `destination slot is occupied`, or `stacks cannot be merged`.

The UI should treat the server response as authoritative and refresh both grids after every mutation. Optimistic local drag state is allowed only as a temporary visual state.

## Inventory Descriptor Shape

Container transfers use the existing inventory slot descriptor pattern.

Recommended shape:

```json
{
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": { "kind": "container", "containerId": "fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest", "slotIndex": 3 },
  "mode": "auto"
}
```

Supported storage kinds currently begin with:

- `player_inventory`
- `container`

Specialized shop storage can continue using `shop_input_storage` and `shop_stock` internally until it is folded into general containers.

## Crafting Integration

Container storage can participate in crafting through the same recipe storage descriptor model used by player and shop crafting.

Current recipe storage descriptors include `player_inventory`, `shop_input_storage`, and `shop_stock`. A future container-backed crafting route can use:

```json
{
  "kind": "container",
  "containerId": "fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest"
}
```

Recipe availability should read ingredient quantities from the selected input descriptor. Recipe execution should consume from the input descriptor and produce to the explicit output descriptor in one HQ transaction. Workstation validation remains a Sunny Town responsibility when crafting depends on live position.

## Rejected Alternatives

- Do not key durable contents only by fixture ID. Fixture IDs are stable only within a room/map and need surrounding metadata for safety.
- Do not store quantities in map fixture JSON, Sunny Town world object snapshots, or frontend state.
- Do not allow browser-supplied container IDs to mutate HQ storage without Sunny Town live validation.
- Do not make Cookie Keeper or another NPC the owner of shop storage; the stable owner is the shop/storage identity.
