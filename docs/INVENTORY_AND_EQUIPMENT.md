# Inventory and Equipment

This document explains how player inventory, wallet stars, shops, and equippable items work in HQ and Sunny Town.

## Concepts

Inventory is for student-owned items. Cookies, clothing, hats, and future usable items belong here.

Stars are not inventory. Stars stay in `student_wallet` and are tracked through `student_star_ledger`.

Equipment is a view over inventory. A student can only equip an item they own, and equipping does not consume the item.

## Database Tables

### `inventory_item_type`

This is the item catalog. It defines every kind of inventory item that can exist.

Important columns:

- `key`: stable item key, such as `cookie`, `sunny_hoodie`, `star_cap`, or `pickaxe`.
- `name`: display name.
- `description`: display text.
- `equip_slot`: nullable equipment slot. Current values are `gear`, `accessory`, and `tool`.
- `visual_key`: nullable Sunny Town render key.

If `equip_slot` is null, the item is not equippable.

Current seeded item types:

- `cookie`: not equippable.
- `sunny_hoodie`: equippable in `gear`.
- `star_cap`: equippable in `accessory`.
- `pickaxe`: equippable in `tool`.

### `student_inventory_item`

This table stores per-student item quantities.

Primary key:

- `(app_user_id, item_type_id)`

Rules:

- `quantity` must be nonnegative.
- API responses only show inventory rows with quantity greater than zero.
- Equippable items are not consumed when equipped.
- Cookies are consumed by pet feeding.

### `student_equipped_item`

This table stores what each student has equipped.

Primary key:

- `(app_user_id, slot)`

Current slots:

- `gear`
- `accessory`
- `tool`

Rules:

- One item per slot.
- The equipped item must exist in `inventory_item_type`.
- The item must have a matching `equip_slot`.
- The student must own at least one of the item in `student_inventory_item`.
- Equipment loading only returns equipped items that are still owned with quantity greater than zero.

### `student_wallet` and `student_star_ledger`

Stars remain wallet currency.

`student_wallet` stores the current star balance.

`student_star_ledger` records star changes from sources such as:

- Sunny Town star pickup rewards.
- Pet falling-stars game rewards.
- Shop purchases.

Inventory should not contain a `star` item type.

## Seeding and Existing Accounts

`ensureSchema` and `deploy/postgres/init/001_create_assignment.sql` seed the item catalog.

Existing student accounts receive:

- 1 `sunny_hoodie`
- 1 `star_cap`
- 1 `pickaxe`

New student accounts also receive those starter items during authenticated user sync.

Existing legacy cookie balances are migrated from `app_user.cookies` into `student_inventory_item`, then `app_user.cookies` is reset to zero. The column still exists for compatibility but is no longer authoritative.

## HQ APIs

### `GET /api/student/inventory`

Student-authenticated endpoint.

Returns positive-quantity inventory items:

```json
{
  "items": [
    {
      "key": "sunny_hoodie",
      "name": "Sunny Hoodie",
      "description": "A cozy hoodie for Sunny Town.",
      "quantity": 1,
      "equipSlot": "gear",
      "visualKey": "sunny_hoodie",
      "equipped": true
    }
  ]
}
```

Notes:

- Non-equippable items have no `equipSlot` or `visualKey`.
- `equipped` is true when the item is currently equipped in any slot.
- Items with quantity zero are omitted.

### `GET /api/student/equipment`

Student-authenticated endpoint.

Returns all equipment slots, even empty ones:

```json
{
  "slots": [
    {
      "slot": "gear",
      "item": {
        "key": "sunny_hoodie",
        "name": "Sunny Hoodie",
        "description": "A cozy hoodie for Sunny Town.",
        "equipSlot": "gear",
        "visualKey": "sunny_hoodie"
      }
    },
    {
      "slot": "accessory",
      "item": null
    },
    {
      "slot": "tool",
      "item": {
        "key": "pickaxe",
        "name": "Pickaxe",
        "description": "A sturdy starter tool.",
        "equipSlot": "tool",
        "visualKey": "pickaxe"
      }
    }
  ]
}
```

### `POST /api/student/equipment/equip`

Student-authenticated endpoint.

Request:

```json
{
  "slot": "gear",
  "itemKey": "sunny_hoodie"
}
```

Validation:

- Slot must be `gear`, `accessory`, or `tool`.
- Item must exist.
- Item must be equippable for that slot.
- Student must own quantity greater than zero.

Returns updated equipment.

### `POST /api/student/equipment/unequip`

Student-authenticated endpoint.

Request:

```json
{
  "slot": "gear"
}
```

Deletes the equipped row for that slot and returns updated equipment.

### `POST /api/student/shop/purchase`

Student-authenticated endpoint.

Currently supports buying cookies from the Cookie Keeper shop:

```json
{
  "shopId": "cookie-keeper-shop",
  "itemKey": "cookie",
  "quantity": 1
}
```

The purchase:

- Verifies the student has enough stars.
- Subtracts stars from `student_wallet`.
- Writes a negative `student_star_ledger` row with source `shop_purchase`.
- Increments cookie inventory.
- Returns the updated star balance and inventory.

## Internal Sunny Town API

### `GET /api/internal/sunny-town/student-equipment?app_user_id=...`

Service-authenticated endpoint used by the Sunny Town server.

Sunny Town calls this endpoint:

- When a player joins.
- When the client reports that equipment changed.

The endpoint is protected by `X-HQ-Service-Secret`.

Sunny Town treats HQ as the source of truth. The browser never directly tells other players what visual equipment to show.

## Frontend Flow

### Shared Store

`frontend/src/stores/studentInventory.ts` owns both inventory and equipment state.

It loads:

- `/api/student/inventory`
- `/api/student/equipment`

It exposes:

- `items`
- `equipmentSlots`
- `cookieQuantity`
- `equippedVisuals`
- `equipItem`
- `unequipItem`

After equipment changes, the store reloads inventory so `equipped` flags stay correct.

### Student Pet Page

The Student Pet page has an Inventory button.

The inventory dialog shows:

- Gear slot.
- Accessory slot.
- Tool slot.
- Inventory items.
- Equip/Unequip buttons for equippable items.
- Quantity badges for non-equippable items, such as cookies.

### Sunny Town

Pressing `E` opens the Sunny Town inventory overlay.

The overlay shows:

- Current equipment slots.
- Inventory items.
- Equip/Unequip controls.

After equip or unequip succeeds:

1. The frontend updates HQ through the equipment API.
2. The frontend sends a WebSocket message:

```json
{
  "type": "equipment_changed"
}
```

3. Sunny Town reloads that player's equipment from HQ.
4. Sunny Town broadcasts snapshots with the updated equipment.

## Sunny Town Multiplayer Snapshots

Player snapshots include equipment:

```json
{
  "id": "player-123",
  "displayName": "Student",
  "x": 320,
  "y": 416,
  "facing": "down",
  "moving": false,
  "avatarId": "pet-default",
  "equipment": {
    "gear": "sunny_hoodie",
    "accessory": "star_cap",
    "tool": "pickaxe"
  },
  "lastProcessedSeq": 42
}
```

The client renders equipment visually from `visual_key` values.

Current MVP visuals:

- `sunny_hoodie`: changes the character body/gear styling.
- `star_cap`: draws a cap/headpiece.
- `pickaxe`: draws a held tool and can play a local swing animation.

### Sunny Town Tool Use

Pressing `F` is the primary Sunny Town interaction key.

Current priority:

- If a nearby NPC or activity can be interacted with, `F` uses that interaction.
- Otherwise, if a tool is equipped, `F` uses the equipped tool.

For MVP, using `pickaxe` plays a local swing animation and sends this WebSocket message:

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

The Sunny Town server validates that the player currently has the requested tool equipped. World-object effects such as mining, combat, fishing, harvesting, or object state changes should be added behind this server handler later.

## Important Rules

- Stars are wallet currency, not inventory.
- Cookies are inventory, not equipment.
- Equipment must be owned before it can be equipped.
- Equipping does not decrement quantity.
- Unequipping does not remove the item from inventory.
- Inventory responses hide zero-quantity items.
- Sunny Town is not the source of truth for equipment; HQ is.

## Testing Checklist

Backend:

- `go test ./...`
- Existing students receive starter equipment during schema migration.
- New students receive starter equipment during authenticated user sync.
- `GET /api/student/inventory` returns only positive quantities and includes equipment metadata.
- `GET /api/student/equipment` returns `gear`, `accessory`, and `tool` slots.
- Equipping a valid owned item succeeds.
- Equipping cookies fails.
- Equipping an unowned item fails.
- Equipping `pickaxe` into the `tool` slot succeeds when owned.
- Equipping an item into the wrong slot fails.
- Unequipping clears the slot.
- Sunny Town internal equipment endpoint requires the service secret.

Frontend:

- `npm run typecheck`
- `npm run build`
- Student Pet inventory dialog shows equipment slots.
- Sunny Town `E` inventory overlay shows equipment slots.
- Equip/Unequip updates the UI.
- Sunny Town snapshots update other players after equipment changes.

Manual:

- Rebuild and restart containers.
- Log in as a student.
- Open the Student Pet inventory dialog and equip/unequip the hoodie and cap.
- Enter Sunny Town and press `E`.
- Equip/unequip from Sunny Town.
- Confirm the player avatar changes visually.
- Confirm another player sees the updated equipment.
