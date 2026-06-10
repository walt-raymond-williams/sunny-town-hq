# Current API Surface

This file summarizes the current API ownership and route groups. Handler ownership now lives in focused packages under `internal/hq/...`, with `cmd/hq/routes.go` composing those handlers into the HQ binary.

## Public App APIs

HQ exposes JSON APIs under `/api/...` and validates Keycloak bearer tokens for app users.

Current route groups:

- `/api/me`
- `/api/students`
- `/api/teacher/...`
- `/api/student/profile`
- `/api/student/inventory`
- `/api/student/inventory/slots`
- `/api/student/inventory/move`
- `/api/student/hotbar`
- `/api/student/crafting/...`
- `/api/student/equipment/...`
- `/api/student/shop/purchase`
- `/api/student/shop/stock`
- `/api/student/pet/feed`
- `/api/student/sunny-town/session`
- `/api/student/assignments/...`
- `/api/assignments...`

Pet APIs use Connect RPC at:

```text
/hq.pet.v1.PetService/...
```

The protobuf source is `proto/hq/pet/v1/pet.proto`. Generated Go code lives under `proto/`; generated TypeScript lives under `frontend/src/gen/`.

Current public API handler ownership:

- Identity/session helper routes: `cmd/hq/handlers.go`, backed by `internal/hq/auth` and `internal/hq/users`
- Student profile and pet actions: `internal/hq/pet`
- Inventory, hotbar, crafting, equipment, wallet, stock-backed shop purchases, and shop stock persistence: `internal/hq/inventory`
- Assignments and grading commands: `internal/hq/assignments`
- Sunny Town session creation: `cmd/hq/handlers.go`, coordinated with `internal/hq/pet`, `internal/hq/inventory`, `internal/hq/sunnytownbridge`, and `internal/sunnytownauth`

`GET /api/student/shop/stock` returns each saleable stock item with its current `quantity` and shop-owned `capacity`; Cookie Keeper cookies currently use capacity `64`.

Inventory item payloads returned by student inventory, hotbar, equipment, and crafting endpoints include backward-compatible item identity/display fields plus grid-inventory metadata:

```json
{
  "key": "stone_block",
  "name": "Stone Block",
  "description": "A solid block crafted from stone.",
  "quantity": 1,
  "equipSlot": "",
  "visualKey": "",
  "iconKey": "stone_block",
  "maxStack": 64,
  "category": "building"
}
```

`visualKey` remains the Sunny Town avatar/equipment render key. Inventory icons should use `iconKey`.

`GET /api/student/inventory/slots` returns the durable player inventory grid while preserving an aggregate summary for compatibility:

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
        "category": "resource"
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

`POST /api/student/inventory/move` moves player inventory stacks transactionally. The request uses source and destination descriptors so later container work can extend the same shape:

```json
{
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": { "kind": "player_inventory", "slotIndex": 5 },
  "mode": "move"
}
```

Supported first-slice modes:

- `move`: move an occupied source stack into an empty destination slot.
- `swap`: swap two occupied slots.
- `merge`: merge compatible item stacks up to the destination item's `maxStack`.
- `auto`: choose move, merge, or swap from current slot state.

The response is the updated slotted inventory payload. Stack splitting is still deferred.

Future container transfer APIs should extend this descriptor pattern with `kind: "container"` and a stable `containerId`, following `docs/current/CONTAINER_STORAGE.md`. Browser requests must not be treated as live access authority; Sunny Town validates proximity/object access before HQ mutates durable container contents.

`GET /api/student/crafting/recipes` returns student-visible recipes with ingredient ownership derived from the selected recipe storage context. The current public student route uses a `player_inventory` context backed by `student_inventory_slot` totals. During the slotted-inventory transition, the aggregate `student_inventory_item` table is still maintained for compatibility, but crafting availability uses the same slot authority as crafting execution.

`POST /api/student/crafting/craft` consumes ingredients and produces output through the slotted inventory mutation helpers. Internally, shared recipe execution uses explicit input/output storage descriptors; the student route maps both input and output to `player_inventory`, while shop production maps input to `shop_input_storage` and output to `shop_stock`. Successful responses return the updated slotted inventory payload shape (`slotCount`, `slots`, and aggregate `items`) plus refreshed recipe availability, so the Sunny Town inventory grid can update without reconstructing or reordering slots from aggregate item totals. If there is no compatible stack or empty slot for the output, the endpoint returns `not enough room in inventory`.

## Internal Service APIs

Sunny Town calls HQ through service-authenticated internal endpoints using `X-HQ-Service-Secret`.

Current Sunny Town internal groups:

- `/api/internal/sunny-town/reward-events`
- `/api/internal/sunny-town/resource-events`
- `/api/internal/sunny-town/npc-job-production`
- `/api/internal/sunny-town/npc-job-production/progress`
- `/api/internal/sunny-town/student-equipment`
- `/api/internal/sunny-town/inventory-quantity`
- `/api/internal/sunny-town/container-slots`
- `/api/internal/sunny-town/container-transfer`
- `/api/internal/sunny-town/player-position`
- `/api/internal/sunny-town/map-objects`
- `/api/internal/sunny-town/map-objects/place`
- `/api/internal/sunny-town/map-objects/remove`

AI service integration uses internal HQ endpoints under:

```text
/api/internal/ai/assignment-attempts/...
```

Do not refactor the AI route shape without coordinating the AI service and `docs/AI_SERVICE_IMPLEMENTATION_PLAN.md`.

Current internal API handler ownership:

- Sunny Town internal endpoints: `internal/hq/sunnytownbridge`
- AI grading callback/context endpoints: `internal/hq/ai`
- Shared internal service authentication helpers: `internal/serviceauth` and package-local endpoint checks where needed

`GET /api/internal/sunny-town/container-slots?container_id=...` is service-authenticated and loads the authoritative slot grid for one durable container. Browser clients do not call this route directly; Sunny Town validates live chest access first, then calls HQ with the stable container ID.

`POST /api/internal/sunny-town/container-transfer` is service-authenticated and is the mutation path for player/container stack transfers. Sunny Town must validate live access before calling it. The request uses player and container slot descriptors:

```json
{
  "appUserId": 123,
  "source": { "kind": "player_inventory", "slotIndex": 0 },
  "destination": {
    "kind": "container",
    "containerId": "fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest",
    "slotIndex": 0
  },
  "mode": "auto"
}
```

The response includes updated `inventory` slots and updated `container` slots. Supported modes match inventory moves: `move`, `swap`, `merge`, and `auto`.

## Sunny Town WebSocket

Sunny Town exposes realtime gameplay at:

```text
/sunny-town/ws
```

Browsers receive a short-lived join token from HQ before connecting. The token includes the authenticated `app_user_id` and the linked Sunny Town `character_id`. Sunny Town uses the character ID for realtime player identity while current durable inventory, wallet, and student-owned actions continue to use the app user ID. Client messages are requests; Sunny Town validates gameplay effects against server-accepted position and equipped/owned tools before committing durable effects to HQ.

Container UI messages:

- `container_open`: client sends `objectSource`, `objectId`, and `clientTimeMs`. Sunny Town validates the active chest and accepted player position, loads the HQ container slots, and replies with `container_opened` plus `container`.
- `container_transfer`: client sends `objectSource`, `objectId`, `source`, `destination`, and `clientTimeMs`. Sunny Town validates the requested direction against the chest `storageRole`, calls HQ, and replies with `container_transfer_committed` plus updated `inventory` and `container` grids.

Current chest roles are enforced server-side: `input` chests allow deposit, `output` chests are read-only for player container transfers, and `general` chests allow both deposit and withdraw.
