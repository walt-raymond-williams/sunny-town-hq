# Cookie Shop Storage Plan

This document tracks the Cookie Shop production/storage model that grows out of NPC job production. It is intentionally separate from raw NPC movement state: HQ owns durable shop/storage state, while Sunny Town owns live NPC simulation and validates when an NPC worked.

## Fresh-Agent Handoff

Read this section first after compaction.

Current implemented baseline:

- Cookie Keeper is an authored shop NPC in `sunny-town/maps/sunny-town-house-1.json`.
- `sunny-town-house-1` has an authored `Cookie Shop` area with stable ID `cookie-shop` and tags `shop`, `workplace`, `cookie_shop`, and `storage_owner`.
- Cookie Keeper has a reachable work anchor at `cookie-keeper-counter`.
- `cookie-keeper-counter` remains the owned Cookie Keeper work anchor; the broader `cookie-shop` area is not NPC-owned.
- `sunny-town-house-1` has an authored fixture `cookie-shop-output-chest` inside/associated with `cookie-shop`.
- Sunny Town initializes that fixture as a runtime `worldObject` with source `fixture`, kind `chest`, and metadata `shopId: cookie-keeper-shop`, `storageRole: output`, `itemKey: cookie`, and `locationId: cookie-shop`.
- The frontend can render chest world objects from the normal `worldObjects` snapshot stream.
- Sunny Town emits service-authenticated `shopkeeper_stock` / `shop_stock_progress` NPC job production events when Cookie Keeper is at the work anchor.
- HQ records those events in `sunny_town_npc_job_production_ledger`.
- HQ stores saleable Cookie Keeper cookies in durable `shop_stock_item` / `shop_stock_ledger` tables.
- Player purchases from `cookie-keeper-shop` consume durable stock before granting a cookie.
- `cookie-keeper-shop` cookie output storage has a logical capacity of `64`.
- NPC production stock increments are clamped at capacity in HQ, while idempotent production and stock ledger events are still recorded.
- `GET /api/student/shop/stock` exposes current stock and capacity to the frontend.
- The Sunny Town shop UI displays current stock as `current / 64` and disables Buy when stock is `0`.
- `0010_seed_cookie_keeper_shop_stock.sql` gives fresh databases a small starter stock of 5 cookies.

Current architecture decision:

- The stable gameplay owner is the shop/storage, not the NPC.
- Do not add NPC-held inventory for Cookie Keeper unless a future gameplay requirement truly needs character-carried items.
- Treat current `shop_stock_item` as the durable stock backing the physical Cookie Shop output chest. The chest fixture is presentation/routing/interaction metadata, not a second inventory source.
- Cookie production does not require ingredients yet. Cookie Keeper can produce as long as he is working in the shop.

## Slice: Logical Output Chest Capacity

Status: `Implemented`

Goal: give Cookie Shop output storage a maximum capacity so NPC production cannot grow stock forever.

Recommended initial rule:

- `cookie-keeper-shop` cookie output storage capacity: `64`.
- HQ clamps production-derived stock increments at that capacity.
- Purchases still decrement stock normally.
- The shop UI displays stock as `current / 64`.
- NPC production events should still be recorded idempotently even when storage is full, but stock should not increase above capacity.

Implementation notes:

- Implemented as an HQ-owned narrow rule in `internal/hq/inventory`.
- Storage capacity rules are not in Sunny Town movement or NPC controller code.
- Service-authenticated production writes still flow through HQ.
- The first schema stays simple and shop/item scoped. A future upgrade system can raise the capacity without changing Sunny Town movement state.

Suggested tests:

- Production stock increment clamps at `64`.
- Duplicate production event does not increase stock.
- Purchase after full stock decrements below capacity.
- UI shows `current / 64` and disables Buy at `0`.

## Immediate Next Slice: Output Chest Inspection/Interaction

Recommended next step:

- Decide and implement how players inspect/interact with `cookie-shop-output-chest`.
- Inspection should read the same existing HQ-owned `cookie-keeper-shop` cookie stock and capacity already shown in the shop UI.
- Keep the chest as an interaction surface over the existing stock; do not create separate fixture inventory.
- A lightweight first version can show current cookie count/capacity near the chest, without adding withdrawals/deposits yet.
- The alternate branch is still teacher lesson prep durable progress if broader NPC job gameplay becomes the priority, but the Cookie Shop storage path is now ready for the chest.

## Slice: Authored Cookie Shop Area

Status: `Implemented`

Goal: make "Cookie Shop" a visible/authored world concept instead of only a shop ID and counter location.

Recommended shape:

- Add an authored map location named `Cookie Shop`.
- Give it stable ID `cookie-shop`.
- Tag it with `shop`, `workplace`, `cookie_shop`, and `storage_owner`.
- Represent it using the best current map-location primitive. Today map locations are point/radius based; if rectangular areas are needed, extend map location schema intentionally rather than improvising one-off fields.
- Keep `cookie-keeper-counter` as a work point inside or associated with the broader Cookie Shop area.

Acceptance criteria:

- Designers and debug tools can identify the Cookie Shop area.
- Cookie Keeper work/storage behavior can refer to a shop-owned area rather than only an NPC counter.
- Existing pathing/anchor behavior remains stable.

Implemented notes:

- Added `cookie-shop` to `sunny-town-house-1` using the current point/radius map-location primitive.
- Tagged it with `shop`, `workplace`, `cookie_shop`, and `storage_owner`.
- Kept `cookie-keeper-counter` as the owned work point for Cookie Keeper so routine anchor resolution still prefers the counter.
- Checked-in map validation covers the new location metadata.

## Slice: Physical Output Chest Fixture

Status: `Implemented`

Goal: show the Cookie Shop output storage as a physical chest/fixture in the shop.

Recommended shape:

- Add a static authored fixture or map object for the output chest.
- Associate it with `cookie-keeper-shop` and output storage.
- The physical chest should point at the same HQ-owned stock state, not create a separate inventory model.
- Keep capacity owned by the shop/storage rule. The fixture can display or upgrade that rule later.

Acceptance criteria:

- The chest is visible/inspectable in the Cookie Shop.
- Cookie production stores into the chest-backed stock.
- Cookie Seller purchases consume from the same stock.
- No duplicate stock source exists.

Implemented notes:

- Added optional map `fixtures` with validation for IDs, bounds, location references, storage role, and output storage metadata.
- Added `cookie-shop-output-chest` to `sunny-town-house-1` with `locationId: cookie-shop`, `shopId: cookie-keeper-shop`, `storageRole: output`, and `itemKey: cookie`.
- Sunny Town initializes map fixtures into the existing runtime `worldObjects` stream using source `fixture`.
- Chest snapshots include name, location, shop, storage role, item key, and tags.
- Frontend Sunny Town types and world-object drawing support `chest` objects.
- This slice makes the chest visible and metadata-inspectable by the client/debug payloads. Player-facing chest interaction UI is the next slice.

## Later Slice: Input Chest And Ingredients

Goal: require ingredients before Cookie Keeper can produce cookies.

Deferred intentionally. Do not implement this before output storage capacity and physical ownership are stable.

Recommended future shape:

- Add a Cookie Shop input chest/storage.
- Define cookie recipe requirements in HQ-owned inventory/crafting/economy rules.
- Sunny Town can validate that Cookie Keeper is working, but HQ should decide whether required inputs exist and consume them atomically with production output.
- If ingredients are missing, record/debug "worked but no inputs" or "production blocked" without mutating output stock.

## Non-Goals For Current Work

- Do not persist raw NPC drive values, routes, goals, path indexes, or live position for this feature.
- Do not create general NPC inventory for Cookie Keeper.
- Do not require ingredients before the output storage loop is stable.
- Do not build a broad building/ownership system before the Cookie Shop use case proves the needed fields.
