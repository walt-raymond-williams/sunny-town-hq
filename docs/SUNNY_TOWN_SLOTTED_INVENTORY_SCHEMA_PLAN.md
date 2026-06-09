# Sunny Town Slotted Inventory Schema Plan

## Purpose

This document is the design output for GitHub issue #9, "Design slotted player inventory schema and compatibility plan."

The goal is to add durable player inventory slots for the Sunny Town grid inventory without breaking current aggregate inventory behavior used by crafting, hotbar assignment, equipment, pet feeding, mining rewards, map-object placement, map-object removal, and Sunny Town validation.

## Current State

HQ owns durable inventory state. Sunny Town owns realtime gameplay validation and calls HQ for durable effects.

Current player inventory is aggregate quantity state:

- `student_inventory_item` stores one row per `(app_user_id, item_type_id)`.
- `IncrementStudentItem` increments aggregate quantity.
- `ConsumeStudentItem` decrements aggregate quantity if enough quantity exists.
- `LoadStudent` returns positive aggregate rows for `GET /api/student/inventory`.
- Crafting uses `ConsumeStudentItem` and `IncrementStudentItem` through `studentRecipeStorage`.
- Pet feeding consumes cookies through `ConsumeStudentItem`.
- Shop purchases produce player items through `IncrementStudentItem`.
- Sunny Town mining rewards use `student_inventory_ledger`, then increment `student_inventory_item`.
- Sunny Town placement uses `ConsumeStudentItem` before inserting `sunny_town_map_object`.
- Sunny Town removal uses `IncrementStudentItem` after deleting `sunny_town_map_object`.
- Sunny Town internal quantity checks call `/api/internal/sunny-town/inventory-quantity`, backed by `LoadStudentInventoryQuantity`.
- Hotbar and equipment reference `inventory_item_type`, not a stack or inventory slot.

Item metadata needed for grid inventory already exists on `inventory_item_type`:

- `icon_key`
- `max_stack`
- `category`

## Decision Summary

Add `student_inventory_slot` as the durable slotted player inventory model.

For the first implementation slice, keep `student_inventory_item` as a transactional aggregate compatibility cache instead of replacing it immediately. Existing aggregate callers should continue reading it. Slot-aware APIs should read slots.

The target end state is for slots to be the source of truth and aggregate totals to be derived from slots by query or view. The transitional cache avoids a risky all-at-once rewrite while slot movement, crafting, placement, and Sunny Town bridge behavior are migrated and tested.

Initial player inventory slot count: `30` slots, indexed `0` through `29`. This maps cleanly to a 5-by-6 grid, leaves room beyond the current starter inventory, and stays small enough for a compact Sunny Town inventory panel.

Stack splitting is deferred from the first movement API. Issue #11 should implement move, swap, and merge first; splitting should be a follow-up operation unless implementation cost is trivial once the move command exists.

Hotbar and equipment should remain item-type references for the first slotted inventory slice. Moving them to stack identity is a later direct-manipulation decision, not required to render the inventory grid or preserve existing gameplay.

## Recommended Schema

Add a new migration after the current latest migration:

```sql
create table if not exists student_inventory_slot (
  app_user_id bigint not null references app_user(id) on delete cascade,
  slot_index integer not null,
  item_type_id bigint null references inventory_item_type(id) on delete restrict,
  quantity integer null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (app_user_id, slot_index),
  constraint student_inventory_slot_index_check check (slot_index >= 0 and slot_index < 30),
  constraint student_inventory_slot_quantity_check check (quantity is null or quantity > 0),
  constraint student_inventory_slot_empty_or_occupied_check check (
    (item_type_id is null and quantity is null)
    or
    (item_type_id is not null and quantity is not null)
  )
);

create index if not exists student_inventory_slot_app_user_id_idx
  on student_inventory_slot (app_user_id, slot_index);

create index if not exists student_inventory_slot_item_type_id_idx
  on student_inventory_slot (app_user_id, item_type_id)
  where item_type_id is not null;
```

Empty slots may be represented by explicit rows with `item_type_id = null` and `quantity = null`, or by missing rows. The load API should always return all 30 slots. Explicit empty rows are useful if future per-slot metadata appears, but the first implementation can generate empty response slots from `generate_series(0, 29)` and only store occupied rows.

Do not add a uniqueness constraint on `(app_user_id, item_type_id)`. The same item type must be allowed in multiple stacks.

Do not add a separate stack identity table in the first slice. A slot row is enough identity for move/swap/merge and keeps the migration smaller. If future requirements need stable stack IDs across moves, add `stack_id uuid` or a `student_inventory_stack` table later.

## Backfill Plan

Backfill from positive aggregate inventory rows into slots.

Recommended algorithm:

1. For each student, read `student_inventory_item` rows with `quantity > 0`, joined to `inventory_item_type`.
2. Order deterministically by `inventory_item_type.id`.
3. Split each aggregate quantity into stacks of at most `coalesce(max_stack, quantity)`.
4. Use `max_stack = 1` for equipment/tools and `max_stack = 64` for current stackable items because the catalog is already seeded that way.
5. Fill slots from `0` upward.
6. If a student's expanded stacks exceed 30 slots, do not silently drop or strand overflow. The implementation should fail the migration smoke test or choose a larger slot count before rollout. Current seeded inventories should fit in 30 slots.
7. Do not create slots for zero-quantity aggregate rows.

Equipped items and hotbar assignments do not need special backfill rows. They reference item types and remain valid if the student owns positive aggregate quantity. The slot API can mark equipment/hotbar state separately from the slot contents when needed.

## Compatibility Strategy

### First Implementation Slice

During #10, keep both representations consistent:

- New slot-aware load and movement code reads and writes `student_inventory_slot`.
- Existing aggregate reads continue using `student_inventory_item`.
- Existing aggregate mutations continue working through compatibility wrappers.
- Every mutation that changes slots must update `student_inventory_item` in the same database transaction.
- Every existing aggregate mutation should either:
  - also mutate slots in the same transaction, or
  - call a shared inventory service function that mutates both.

The second option is preferred. Avoid spreading dual-write SQL across crafting, pet, shop, map-object, and Sunny Town bridge code.

### Aggregate Quantity Reads

Keep `LoadStudentInventoryQuantity` and `/api/internal/sunny-town/inventory-quantity` stable for Sunny Town. In the first slice, they can continue reading `student_inventory_item`.

Once all player inventory mutations write slots reliably, change quantity loading to derive totals from slots:

```sql
select coalesce(sum(sis.quantity), 0)
from inventory_item_type iit
left join student_inventory_slot sis on sis.item_type_id = iit.id
  and sis.app_user_id = $1
where iit.key = $2
```

At that point, `student_inventory_item` can become a view or be removed in a later migration. Do not make that replacement in #10 unless all existing aggregate call sites are covered by focused tests.

### Existing Mutation Behavior

`IncrementStudentItem` should place new quantity into slots. Recommended behavior:

- Lock the student's inventory rows for the transaction.
- Resolve item metadata and max stack.
- Fill existing compatible stacks that are below max stack.
- Create new occupied slots in the first empty slots.
- If there is not enough empty capacity, return a clear inventory-full error and leave state unchanged.
- Update `student_inventory_item` only after the slot operation succeeds.

`ConsumeStudentItem` should consume from slots. Recommended behavior:

- Lock the student's inventory rows for the transaction.
- Sum matching slot quantities first.
- If the sum is insufficient, return `false` and leave state unchanged.
- Decrement from matching stacks deterministically. Prefer higher slot indexes first so front-packed inventory remains more stable for the player.
- Delete occupied rows that reach zero, or convert them to explicit empty rows.
- Update `student_inventory_item` only after slot consumption succeeds.

Crafting, pet feeding, shop purchase, mining rewards, placement, and removal can keep calling these functions if the functions become the shared dual-write boundary.

## API Response Recommendation

Keep `GET /api/student/inventory` backward-compatible during the first slot implementation. It should continue returning:

```json
{
  "items": [
    {
      "key": "rock",
      "name": "Rock",
      "description": "A sturdy rock from Forest Crossing.",
      "quantity": 12,
      "iconKey": "rock",
      "maxStack": 64,
      "category": "resource",
      "equipped": false
    }
  ]
}
```

Add a slotted response shape either as a new endpoint or an additive field. A new endpoint is cleaner for compatibility:

```text
GET /api/student/inventory/slots
```

Recommended response:

```json
{
  "slotCount": 30,
  "slots": [
    {
      "slotIndex": 0,
      "item": {
        "key": "rock",
        "name": "Rock",
        "description": "A sturdy rock from Forest Crossing.",
        "quantity": 12,
        "iconKey": "rock",
        "maxStack": 64,
        "category": "resource",
        "equipSlot": "",
        "visualKey": ""
      }
    },
    {
      "slotIndex": 1,
      "item": null
    }
  ],
  "items": [
    {
      "key": "rock",
      "quantity": 12
    }
  ]
}
```

The `items` aggregate summary is optional but useful while the frontend store still has aggregate consumers such as crafting availability, hotbar quantity display, and placement UI.

For #11 movement, prefer an endpoint that can later extend to containers:

```text
POST /api/student/inventory/move
```

Request:

```json
{
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": { "kind": "player_inventory", "slotIndex": 5 },
  "mode": "move"
}
```

First slice modes:

- `move`: move source stack into an empty destination.
- `swap`: swap two occupied slots.
- `merge`: merge compatible stacks up to `maxStack`.

The server may also accept `mode: "auto"` and choose move/swap/merge from current slot state, but tests should still cover each behavior explicitly.

Do not include split quantity in the first required API shape. A later split request can add `quantity`:

```json
{
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": { "kind": "player_inventory", "slotIndex": 5 },
  "quantity": 8,
  "mode": "split"
}
```

## Hotbar And Equipment Plan

Keep current persistence:

- `student_hotbar_slot(app_user_id, slot_index, item_type_id)`
- `student_equipped_item(app_user_id, slot, item_type_id)`

Rationale:

- Current gameplay cares whether the student owns an item type, not which stack contains it.
- Equipment is a view over ownership and does not consume inventory quantity.
- Hotbar selection is a quick-use assignment by item type.
- Moving hotbar/equipment to stack identity would force extra decisions about what happens when the backing stack is split, merged, consumed, or moved into a chest.

Future direct-manipulation work can still let players drag from an inventory slot onto a hotbar or equipment slot. The backend can translate the dragged slot into `item_type_id` and keep the current tables.

## Rejected Alternatives

### Replace `student_inventory_item` Immediately

Rejected for #10 because too many stable paths currently read or mutate aggregate rows. A direct replacement would touch crafting, pet feeding, shop purchase, mining rewards, placement, removal, hotbar ownership validation, equipment ownership validation, and Sunny Town bridge checks in one change.

### Keep `student_inventory_item` As Permanent Source Of Truth

Rejected as the target model because aggregate rows cannot represent stable grid positions or multiple stacks of the same item type. It can only remain a compatibility cache.

### Store Slot Layout Only In The Frontend

Rejected because inventory organization should be durable across sessions and devices, and the client is not authoritative for gameplay state.

### Add Stack IDs Before Slots

Deferred. Slot identity is sufficient for the first grid and movement API. Stable stack IDs can be added later if container transfers, audit logs, or item instances require them.

## Downstream Notes For #10

#10 should implement:

- Migration adding `student_inventory_slot`.
- Backfill from `student_inventory_item`.
- Load function for all 30 slots.
- Shared mutation helpers that update slots and aggregate totals transactionally.
- Compatibility tests proving aggregate totals remain correct after slot mutations.
- Tests for inventory-full behavior on increments.
- Tests for consumption across multiple stacks.
- Tests that `/api/internal/sunny-town/inventory-quantity` still returns correct totals.

Suggested focused verification:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/schema
go test ./internal/hq/sunnytownbridge
```

## Downstream Notes For #11

#11 should implement:

- Transactional move into empty slot.
- Transactional swap between occupied slots.
- Transactional merge for same item type up to `max_stack`.
- Clear client-safe errors for invalid slot, empty source, occupied destination for move, incompatible merge, and overflow.
- Response that returns the updated slotted inventory and aggregate summary.
- Stack splitting as explicitly deferred unless it is added with focused tests.

Use `source` and `destination` descriptors now so later container work can extend `kind` without redesigning the player-only API.

## Open Questions

- Should explicit empty slot rows be stored during migration, or should empty slots be generated at load time? Recommendation: generate empty slots for now.
- Should inventory capacity later become upgradeable through backpack/equipment state? Recommendation: keep fixed 30 slots for #10 and add capacity upgrades later.
- When general containers arrive, should slot indexes be zero-based for containers too? Recommendation: yes, use zero-based indexes consistently for all grid storage.
