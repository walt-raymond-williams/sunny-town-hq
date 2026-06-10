# Cookie Shop Storage Plan

This document tracks the Cookie Shop production/storage model that grows out of NPC job production. It is intentionally separate from raw NPC movement state: HQ owns durable shop/storage state, while Sunny Town owns live NPC simulation and validates when an NPC worked.

## Fresh-Agent Handoff

Read this section first after compaction.

Current implemented baseline:

- Cookie Keeper is an authored shop NPC in `sunny-town/maps/sunny-town-house-1.json`.
- `sunny-town-house-1` has an authored `Cookie Shop` area with stable ID `cookie-shop` and tags `shop`, `workplace`, `cookie_shop`, and `storage_owner`.
- Cookie Keeper has a reachable work anchor at `cookie-keeper-counter`.
- `cookie-keeper-counter` remains the owned Cookie Keeper work anchor; the broader `cookie-shop` area is not NPC-owned.
- `sunny-town-house-1` has authored fixtures `cookie-shop-output-chest` and `cookie-shop-input-chest` inside/associated with `cookie-shop`.
- Sunny Town initializes that fixture as a runtime `worldObject` with source `fixture`, kind `chest`, and metadata `shopId: cookie-keeper-shop`, `storageRole: output`, `itemKey: cookie`, and `locationId: cookie-shop`.
- Sunny Town also initializes `cookie-shop-input-chest` as a runtime chest `worldObject` with `shopId: cookie-keeper-shop`, `storageRole: input`, and `locationId: cookie-shop`. It has no `itemKey` yet because ingredient storage is not durable yet.
- The frontend can render chest world objects from the normal `worldObjects` snapshot stream.
- Players can inspect the nearby output chest with `F`; the read-only panel loads existing `cookie-keeper-shop` stock/capacity from `GET /api/student/shop/stock`.
- Sunny Town emits service-authenticated `shopkeeper_stock` / `shop_stock_progress` NPC job production events when Cookie Keeper is at the work anchor.
- HQ records successful production events in `sunny_town_npc_job_production_ledger`.
- HQ records accepted but blocked production attempts in `sunny_town_npc_job_production_blocked_ledger`.
- HQ stores saleable Cookie Keeper cookies in durable `shop_stock_item` / `shop_stock_ledger` tables.
- HQ stores Cookie Shop input ingredients in durable `shop_input_storage_item`.
- Player purchases from `cookie-keeper-shop` consume durable stock before granting a cookie.
- `cookie-keeper-shop` cookie output storage has a logical capacity of `64`.
- `cookie-keeper-shop` input storage has a logical capacity of `64` total units.
- Cookie production uses the shared `cookie` recipe: 1 flour + 1 sugar -> 1 cookie.
- NPC production checks input ingredients and output capacity before mutating storage.
- If ingredients are missing, HQ records a blocked production attempt with reason `missing_inputs` and does not consume partial inputs or mutate output stock.
- If output storage is full, HQ records a blocked production attempt with reason `output_full` and does not consume inputs.
- Successful NPC production consumes Cookie Shop input storage and increments durable output stock atomically.
- NPC production stock increments remain idempotent by production event ID.
- `GET /api/student/shop/stock` exposes current stock and capacity to the frontend.
- The Sunny Town shop UI displays current stock as `current / 64` and disables Buy when stock is `0`.
- `0010_seed_cookie_keeper_shop_stock.sql` gives fresh databases a small starter stock of 5 cookies.
- `0014_seed_cookie_keeper_input_storage.sql` gives fresh databases a small starter input supply of 16 flour and 16 sugar.

Current architecture decision:

- The stable gameplay owner is the shop/storage, not the NPC.
- Do not add NPC-held inventory for Cookie Keeper unless a future gameplay requirement truly needs character-carried items.
- Treat current `shop_stock_item` as the durable stock backing the physical Cookie Shop output chest. The chest fixture is presentation/routing/interaction metadata, not a second inventory source.
- Recipes must be shared HQ-owned gameplay definitions, not Cookie Keeper-specific production rules.
- Player crafting and NPC/shop production should use the same recipe catalog and execution semantics, with different storage endpoints.
- Workstations, such as a future Cookie Shop stove, should be the recipe interaction point. Chests remain storage anchors.
- Cookie production requires input ingredients in Cookie Shop input storage. Cookie Keeper can produce only while working in the shop and while ingredients/output capacity are available.
- HQ crafting now has a shared recipe catalog/execution foundation in `internal/hq/inventory/crafting.go`.
- The current `stone_block` player recipe still uses the existing `/api/student/crafting/...` behavior, but execution is routed through storage operations instead of being hard-coded to student inventory.
- Shared recipe execution is intentionally storage-agnostic: current student crafting adapts `ConsumeStudentItem` / `IncrementStudentItem` behind a `player_inventory` storage descriptor, while Cookie Shop production uses `shop_input_storage` as the recipe input descriptor and `shop_stock` as the output descriptor.
- Inventory redesign discovery in `docs/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md` agrees with this direction: chest fixtures are interaction metadata, durable quantities live in HQ, and future grid/container UI should wrap explicit storage endpoints rather than create Sunny Town-local inventory state.
- General container/chest identity, access validation, conflict handling, and future crafting integration are defined in `docs/current/CONTAINER_STORAGE.md`.

Next task for a fresh agent:

- Add a way for gameplay to replenish Cookie Shop input storage, probably starting with a service/internal or narrow debug/admin path before player transfer UI.
- Keep the storage source explicit: player-to-chest deposit and chest withdraw still need ownership/access rules before becoming player-facing.
- Do not create Cookie Keeper-held inventory.
- Do not create Sunny Town fixture-local ingredient quantities.
- If UI changes are made, show blocked/empty input state clearly instead of making the shop look broken.
- Verify backend changes with `go test ./...`; run frontend checks only when API response shapes or UI change.

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

## Immediate Next Slice: Shared Recipe Execution Foundation

Status: `Implemented`

Recommended next step:

- Refactor the existing HQ student crafting recipe code into a shared recipe catalog/execution foundation.
- Preserve current player crafting behavior for `stone_block`: consume from `student_inventory_item`, output to `student_inventory_item`, and keep existing APIs compatible.
- Model recipes as actor-agnostic definitions: inputs, output, quantity, and later station/workstation requirements.
- Add execution seams for different storage endpoints without implementing all endpoints at once:
  - student inventory input -> student inventory output for player crafting,
  - shop input storage -> shop output stock for NPC/store production later,
  - possible future player-at-shop mode after ownership transfer rules are designed.
- Do not bake cookie recipe consumption directly into `CommitNPCJobProduction`.
- Treat `cookie-shop-input-chest` as the physical interaction/ownership anchor for future ingredients, not as a local inventory source.
- Define a clear future "production blocked: missing inputs" result for the NPC production path before mutating output stock.
- Keep chest actions such as withdraw/deposit deferred until the sale path and ownership transfer rules are explicit.
- The alternate branch is teacher lesson prep durable progress if broader NPC job gameplay becomes the priority.
- Do not create a second fixture inventory source; keep durable quantities in HQ-owned stock/inventory tables.

Acceptance criteria for this next slice:

- Existing student crafting tests still pass without frontend API changes.
- Recipe definitions can be reused by non-student execution code without depending on a player `app_user_id`.
- The code shape makes the later Cookie Shop path explicit: consume ingredients from HQ-owned shop input storage and produce cookies into HQ-owned shop output stock.
- No NPC production behavior changes yet unless the shared executor and storage endpoints are ready.

Suggested implementation shape:

- Keep a recipe definition type that is independent of storage ownership: key, output item key, output quantity, and ingredient item keys/quantities.
- Add a small executor/helper that receives storage operations for consuming ingredients and producing output.
- Implement the first storage operations against existing student inventory functions (`ConsumeStudentItem`, `IncrementStudentItem`).
- Keep recipe metadata loading usable for the current player crafting UI.
- Add or update tests around `CraftStudentRecipe` and recipe lookup so the behavior is protected before later shop storage is introduced.

Implemented notes:

- Replaced the student-specific internal recipe structs with `RecipeDefinition` and `RecipeIngredient`.
- Renamed the recipe list to `recipeCatalog` to make the shared catalog role explicit.
- Added a storage-agnostic recipe executor that consumes required ingredients before producing output and carries explicit input/output storage descriptors through availability and execution.
- Added `studentRecipeStorage` as the first adapter over existing student inventory functions.
- Preserved the current `stone_block` API behavior and response shape.
- Added unit tests proving shared execution can run against fake storage without a player `app_user_id`, and that missing ingredients stop output production while retaining the selected storage context.

## Immediate Next Slice: HQ-Owned Shop Input Storage

Status: `Implemented`

Recommended next step:

- Add durable Cookie Shop input storage in HQ for ingredients that will eventually feed recipes.
- Scope the storage by shop/storage identity, such as `cookie-keeper-shop` plus input role/item, not by Cookie Keeper NPC identity.
- Keep the physical `cookie-shop-input-chest` as routing/interaction metadata only; quantities must live in HQ.
- Do not wire NPC production to consume ingredients until input storage operations and missing-input semantics exist.
- Do not implement player deposit/withdraw UI yet unless ownership transfer rules are designed in the same slice.

Acceptance criteria for this next slice:

- HQ can persist and load input chest ingredient quantities for a shop.
- Input storage has a clear capacity rule or an explicit documented reason capacity is deferred.
- New storage operations are shaped so shared recipe execution can later consume from shop input storage in a transaction.
- Existing shop output stock and student crafting behavior remain unchanged.

Suggested implementation shape:

- Add an HQ migration for shop input storage and, if useful for idempotent service writes later, a narrow ledger table.
- Add `internal/hq/inventory` methods for loading and incrementing/decrementing shop input storage.
- Keep any seed data small and explicit if needed for tests; avoid pretending ingredients are required for Cookie Keeper production in this slice.
- Add backend tests for persistence, capacity or validation, and insufficient input handling at the storage operation level.

Implemented notes:

- Added `deploy/postgres/migrations/0011_shop_input_storage.sql` with `shop_input_storage_item`.
- Added `CookieKeeperInputStorageCapacity = 64` as the first input chest capacity rule.
- Added `LoadShopInputStorage`, `IncrementShopInputStorageItem`, and `ConsumeShopInputStorageItem` in `internal/hq/inventory/shop.go`.
- Input storage is scoped by shop ID and item type, not by NPC identity or fixture-local state.
- Incrementing input storage rejects writes that would exceed the total input capacity; it does not partially fill.
- Input storage increment uses a per-shop transaction advisory lock so concurrent capacity checks serialize.
- This slice intentionally does not expose player deposit/withdraw UI, add a cookie recipe, or change NPC production behavior.

## Immediate Next Slice: Recipe-Aware Shop Production Design

Status: `Implemented`

Recommended next step:

- Design the blocked-production semantics before consuming ingredients from Cookie Keeper production.
- Decide what HQ records or returns when Cookie Keeper works but the shop input storage lacks required ingredients.
- Add a cookie recipe to the shared catalog only when it can be executed through storage adapters, not as a Cookie Keeper-only rule.
- Create a shop recipe storage adapter that consumes from `shop_input_storage_item` and produces to `shop_stock_item` in one transaction.
- Keep the future stove/workstation fixture in mind, but do not require it unless the slice explicitly includes player/NPC workstation interaction.

Acceptance criteria for this next slice:

- Missing ingredients do not mutate output stock.
- Successful recipe execution consumes input storage and increments output stock atomically.
- NPC production ledger behavior remains idempotent and understandable when production is blocked.
- Existing no-ingredient Cookie Keeper behavior changes only when the blocked/success path is fully covered by tests.

Implemented notes:

- Added `flour` and `sugar` item types for cookie ingredients.
- Added the shared `cookie` recipe as 1 flour + 1 sugar -> 1 cookie.
- Kept the cookie recipe out of the current student crafting UI until workstation/player storage context exists.
- Added `CommitShopRecipeProduction` and a shop recipe storage adapter that checks output capacity and input availability before consuming ingredients.
- Added `sunny_town_npc_job_production_blocked_ledger` for durable blocked attempts.
- `CommitNPCJobProduction` now returns accepted blocked outcomes for Cookie Keeper production when inputs are missing or output storage is full.
- Duplicate event IDs are idempotent whether the original outcome was successful production or blocked production.
- Sunny Town logs blocked production responses without treating them as commit failures.
- Added starter Cookie Shop input storage seed: 16 flour and 16 sugar.
- This slice intentionally does not add player deposit/withdraw UI or workstation interaction.

## Immediate Next Slice: Cookie Shop Input Replenishment

Recommended next step:

- Give gameplay a controlled way to add ingredients to `shop_input_storage_item`.
- Start narrow: an internal/service endpoint, debug/admin route, or explicitly designed player deposit flow.
- If player deposit is chosen, validate the player is near/opening `cookie-shop-input-chest`, consume from player inventory, and increment shop input storage in one HQ-owned transaction.
- Keep output chest withdraw/deposit separate from input ingredient deposit; selling cookies still consumes from shop output stock.

Acceptance criteria for this next slice:

- Input storage can be replenished without direct database edits.
- Replenishment cannot exceed the input storage capacity.
- Player-facing flows, if added, validate ownership/access and do not create fixture-local quantities.
- Existing recipe-aware production remains idempotent and atomic.

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

## Slice: Output Chest Inspection/Interaction

Status: `Implemented`

Goal: let players inspect the physical output chest and see the existing durable stock it represents.

Implemented notes:

- Chest snapshots expose `interactionRadius` so the client can use authored interaction range.
- Added `useSunnyTownChestInteractions` to find nearby active output chest fixtures, open/close the panel, and load stock through the existing shop stock API.
- Added `SunnyTownChestPanel`, a read-only storage panel showing the chest item quantity and capacity.
- Sunny Town page now shows an `F` prompt for nearby chests when no NPC prompt is active.
- Pressing `F` near `cookie-shop-output-chest` loads `cookie-keeper-shop` stock and shows `cookie current / 64`.
- Walking away, changing maps, or pressing Escape closes the chest panel.
- This does not add withdraw/deposit behavior and does not create a separate inventory source.

## Slice: Physical Input Chest Fixture

Status: `Implemented`

Goal: give future ingredient storage a physical Cookie Shop anchor without creating premature inventory logic.

Implemented notes:

- Added `cookie-shop-input-chest` to `sunny-town-house-1` with `locationId: cookie-shop`, `shopId: cookie-keeper-shop`, and `storageRole: input`.
- The input chest is a blocking/reserved `chest` fixture tagged `storage`, `input`, and `cookie_shop`.
- Map validation now requires input storage fixtures to include both shop and location metadata.
- Sunny Town initializes the input chest into the normal runtime `worldObjects` stream and snapshots its storage metadata.
- The input chest intentionally has no `itemKey` and no durable quantity yet. Future ingredient storage must be HQ-owned and attached to the shop/storage identity.

## Later Slice: Input Chest And Ingredients

Goal: require ingredients before Cookie Keeper can produce cookies.

Deferred intentionally. Do not implement this before the shared recipe execution foundation is stable.

Recommended future shape:

- Add HQ-owned Cookie Shop input storage.
- Add a Cookie Shop workstation fixture, likely a stove/oven, as the player/NPC recipe interaction point.
- Define cookie recipe requirements in the shared HQ recipe catalog.
- Sunny Town can validate that Cookie Keeper is working at the right authored location/workstation, but HQ should decide whether required inputs exist and consume them atomically with production output.
- If ingredients are missing, record/debug "worked but no inputs" or "production blocked" without mutating output stock.
- Player crafting at a stove should call the same recipe definition and execution semantics, using player inventory or explicitly designed shop-storage transfer rules.

## Non-Goals For Current Work

- Do not persist raw NPC drive values, routes, goals, path indexes, or live position for this feature.
- Do not create general NPC inventory for Cookie Keeper.
- Do not require ingredients before the output storage loop is stable.
- Do not create a separate Cookie Keeper-only recipe system.
- Do not put recipe state or ingredient quantities in Sunny Town fixture/client state.
- Do not build a broad building/ownership system before the Cookie Shop use case proves the needed fields.
