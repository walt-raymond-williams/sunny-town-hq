# Sunny Town Inventory Redesign Roadmap

## Purpose

This document turns the inventory redesign discovery into an implementation roadmap and GitHub-ready issue list.

The target is a game-style `E` menu for Sunny Town: slotted inventory grid, item icons, stack quantities, hover tooltips, drag/drop to hotbar and equipment slots, crafting, chest/container transfer, and a character preview area that can later show stats and skill progression.

## Current Implementation Findings

### Frontend

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue` renders the current inventory as text-heavy cards plus action buttons.
- Equipment is updated through `Wear` and `Unequip` buttons.
- Hotbar assignment is based on the currently selected hotbar slot plus a `Slot N` button on each inventory card.
- `frontend/src/stores/studentInventory.ts` owns inventory, equipment, hotbar, and crafting state together.
- `frontend/src/types/inventory.ts` models inventory as `InventoryItem[]`, where each item is a quantity row keyed by item type.
- `frontend/src/features/sunny-town/SunnyTownHud.vue` already renders a compact five-slot hotbar, but the inventory overlay does not use that same direct slot language.
- Item icons exist as CSS classes in `frontend/src/style.css`, not as durable item metadata or asset references.
- `frontend/src/composables/useSunnyTownChestInteractions.ts` only supports inspecting output chest fixtures backed by shop stock.
- `frontend/src/features/sunny-town/SunnyTownChestPanel.vue` is read-only and shows one output stock item.

### Backend And Persistence

- HQ owns durable inventory state.
- Sunny Town owns realtime world state and validates gameplay interactions before calling HQ.
- `student_inventory_item` stores one quantity row per `(app_user_id, item_type_id)`. It is not slot-based.
- `inventory_item_type` has stable key, name, description, nullable `equip_slot`, and nullable `visual_key`.
- `student_equipped_item` already persists equipment slots: `gear`, `accessory`, and `tool`.
- `student_hotbar_slot` already persists five hotbar slots by item type.
- Hotbar slots reference item type, not a specific stack or inventory slot.
- Crafting is HQ-owned through `/api/student/crafting/...`.
- `internal/hq/inventory/crafting.go` already has a storage-agnostic recipe execution seam through `recipeStorage`.
- Cookie Shop output stock is durable in `shop_stock_item` and `shop_stock_ledger`.
- Cookie Shop input chest exists as a fixture, but durable input storage does not exist.
- General player-owned or world-owned chest inventory does not exist.
- General chest transfer APIs do not exist.

### Boundary Recommendation

- Keep durable quantities in HQ.
- Keep chest fixtures in Sunny Town as interaction, routing, and presentation metadata.
- Keep recipe definitions and recipe execution in HQ.
- Let Sunny Town validate world access such as nearby chest/workstation/player position, then call HQ for durable mutations through public student APIs or service-authenticated internal APIs as appropriate.
- Do not store authoritative inventory quantities in Sunny Town map JSON, fixture metadata, or client state.

## Data Model Gaps

- Inventory slot identity for player inventory.
- Stack move, swap, merge, and split semantics.
- Max stack size per item type.
- Icon key or asset key per item type.
- Item category metadata beyond `equip_slot`.
- Equipment visual layering metadata beyond `visual_key`.
- General storage/container identity.
- Container slot model.
- Container ownership/access model.
- Shop input storage table and operations.
- Storage-context-aware recipe availability and execution responses.

## API Gaps

- Load slotted player inventory.
- Move/swap/merge/split inventory stacks.
- Assign hotbar by drag/drop semantics.
- Assign equipment by drag/drop semantics.
- Load container inventory for an accessible container.
- Transfer stacks between player inventory and container storage.
- Load crafting recipes for a selected storage context.
- Execute recipes against explicit input/output storage contexts.
- Notify or refresh Sunny Town equipment visuals after drag/drop equipment changes.

## UX Gaps

- Reusable inventory slot component.
- Tooltip behavior for item name/details.
- Drag/drop interaction model.
- Invalid drop feedback.
- Pending and rollback states for server-validated moves.
- Slotted player inventory grid.
- Slotted chest/container grid.
- Equipment slot UI near character preview.
- Character preview shell.
- Crafting panel integrated with inventory visual language.
- Explicit ingredient source and output destination display for storage-aware crafting.

## Recommended Phase Order

1. Preserve current behavior while adding item metadata needed by grid UI.
2. Add slotted inventory backend model and move operations.
3. Build reusable frontend slot/grid primitives against real or adapter-shaped data.
4. Convert hotbar and equipment assignment to drag/drop using existing durable state.
5. Redesign crafting UI while preserving player-inventory crafting.
6. Add general container/chest storage and transfer APIs.
7. Extend recipe execution to explicit storage contexts, including Cookie Shop input storage.
8. Add character preview and equipment layout.
9. Add tabbed `E` menu shell.
10. Design stats/skills progression separately, then integrate into the menu.

## Concurrency Groups

- Group A, backend inventory foundation: item metadata, slotted inventory schema, stack movement APIs.
- Group B, frontend visual foundation: slot component, tooltip, grid layout, icon rendering.
- Group C, current-state adapters: keep existing inventory, hotbar, equipment, and crafting flows working while new response shapes are introduced.
- Group D, container/storage design: chest storage schema, access validation contract, shop input storage.
- Group E, character/menu design: tab shell, avatar preview, stats-ready area.

Do not run two agents against the same files in `frontend/src/stores/studentInventory.ts`, `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`, or `internal/hq/inventory/*.go` without a narrow interface contract.

## Issue: Add Item Metadata Needed For Grid Inventory

Epic: Inventory Slot And Item Metadata Foundation
Type: backend
Priority: P0

Blocked by:
- None

Related to:
- Build Reusable Inventory Slot Component
- Render Player Inventory As A Grid

User story:
As a player, I want items to have stable icons, names, descriptions, stack rules, and categories, so that the inventory can be recognized visually instead of read as a debug list.

Acceptance criteria:
- `inventory_item_type` can represent `iconKey` or equivalent UI asset key.
- `inventory_item_type` can represent max stack size or the roadmap documents why max stack is deferred.
- API responses include the metadata needed by item slots without hard-coding every icon in CSS.
- Existing item responses remain backward compatible or have a documented migration path.

Implementation notes:
- Start from `deploy/postgres/migrations/0003_inventory_equipment.sql`.
- Update `internal/hq/inventory/inventory.go`, `hotbar.go`, `equipment.go`, and `crafting.go` response structs as needed.
- Keep `visual_key` for avatar/equipment rendering; do not overload it as the inventory icon key unless that is explicitly chosen.

Verification:
- `go test ./internal/hq/inventory`
- `go test ./internal/hq/schema`

## Issue: Design And Implement Slotted Player Inventory Persistence

Epic: Inventory Slot And Item Metadata Foundation
Type: backend
Priority: P0

Blocked by:
- Add Item Metadata Needed For Grid Inventory

Related to:
- Add Stack Move Swap Merge And Split Operations
- Render Player Inventory As A Grid

User story:
As a player, I want my inventory to have stable grid positions, so that items stay where I put them.

Acceptance criteria:
- Player inventory supports a fixed or configurable number of slots.
- Each occupied slot records item type and quantity.
- Existing `student_inventory_item` quantities are migrated or adapted into initial slots.
- Existing gameplay mutations still preserve total owned quantities.
- A compatibility path exists for APIs that still need total quantity by item key.

Implementation notes:
- Current `student_inventory_item` is aggregate quantity, not a slot table.
- Consider adding `student_inventory_slot` rather than mutating aggregate rows in place.
- Decide whether `student_inventory_item` remains an aggregate cache, becomes a view, or is replaced by slot totals.
- Sunny Town currently calls internal inventory quantity for placement validation; preserve that behavior.

Verification:
- Migration smoke test through `go test ./internal/hq/schema`
- Focused inventory tests for migration/adaptation and quantity totals.

## Issue: Add Stack Move Swap Merge And Split Operations

Epic: Inventory Slot And Item Metadata Foundation
Type: backend
Priority: P0

Blocked by:
- Design And Implement Slotted Player Inventory Persistence

Related to:
- Add Drag And Drop Inventory Store Actions
- Add Player To Container Transfer Operations

User story:
As a player, I want to rearrange item stacks directly, so that inventory organization feels like a real game inventory.

Acceptance criteria:
- Backend supports moving a stack between slots.
- Backend supports swapping occupied slots.
- Backend supports merging compatible stacks up to max stack size.
- Backend supports splitting stacks if selected for the first implementation slice, or records it as deferred.
- Invalid moves return clear client-safe errors.
- Operations are transactional.

Implementation notes:
- Add focused command functions in `internal/hq/inventory`.
- Prefer one explicit move endpoint with source/destination descriptors if that will later support containers.
- Keep optimistic UI optional until backend semantics are protected by tests.

Verification:
- Unit tests for move, swap, merge, overflow, invalid slot, and insufficient quantity.
- `go test ./internal/hq/inventory`

## Issue: Build Reusable Inventory Slot Component

Epic: Inventory Grid UI
Type: frontend
Priority: P0

Blocked by:
- Add Item Metadata Needed For Grid Inventory

Related to:
- Render Player Inventory As A Grid
- Render Hotbar As Drop Targets In Inventory Menu
- Render Equipment Slots As Drop Targets
- Render Chest Inventory Grid

User story:
As a player, I want each inventory item to appear as a compact slot with an icon and quantity, so that I can scan my belongings quickly.

Acceptance criteria:
- Slot component renders empty, occupied, selected, hover, disabled, and invalid-drop states.
- Occupied slots show item icon and stack quantity.
- Tooltip shows item name and description.
- Component is usable for player inventory, hotbar, equipment slots, and containers.
- Component has stable dimensions and does not resize based on content.

Implementation notes:
- Add a component under `frontend/src/features/sunny-town/` or a shared inventory component folder if that pattern is introduced.
- Existing icon CSS is in `frontend/src/style.css`; migrate carefully if item metadata introduces `iconKey`.
- Avoid baking slot behavior into only the Sunny Town inventory overlay.

Verification:
- `cd frontend; npm run build`
- Focused component tests if the project already supports the pattern.

## Issue: Render Player Inventory As A Grid

Epic: Inventory Grid UI
Type: frontend
Priority: P0

Blocked by:
- Build Reusable Inventory Slot Component
- Design And Implement Slotted Player Inventory Persistence

Related to:
- Add Drag And Drop Inventory Store Actions
- Redesign Crafting Panel For Inventory Menu

User story:
As a player, I want my inventory to appear as a grid, so that it feels like a final game interface instead of a debug list.

Acceptance criteria:
- `SunnyTownInventoryPanel.vue` renders a grid of inventory slots.
- Empty slots are visible.
- Item stacks show icon and quantity.
- Hovering a slot shows item details.
- Existing inventory loading and error states still work.
- Existing crafting, equipment, and hotbar capabilities are not removed during this slice.

Implementation notes:
- `frontend/src/stores/studentInventory.ts` will need slot-shaped state or adapter getters.
- Keep compatibility with existing flat item responses until backend slot APIs are ready if this is worked in parallel.

Verification:
- `cd frontend; npm run build`
- Manual Sunny Town check: open `E`, verify grid, quantities, and tooltips.

## Issue: Add Drag And Drop Inventory Store Actions

Epic: Drag-And-Drop Inventory Interactions
Type: frontend
Priority: P0

Blocked by:
- Add Stack Move Swap Merge And Split Operations
- Render Player Inventory As A Grid

Related to:
- Render Hotbar As Drop Targets In Inventory Menu
- Render Equipment Slots As Drop Targets

User story:
As a player, I want to drag items around my inventory, so that organizing items feels direct.

Acceptance criteria:
- Store exposes actions for drag start, drag cancel, and drop.
- Store calls backend move APIs for durable slot changes.
- UI handles pending moves and failed moves without losing state.
- Invalid drops are rejected visibly.
- Keyboard or accessible fallback is considered or documented.

Implementation notes:
- Current `useSunnyTownInventoryActions.ts` is action-button oriented.
- Drag/drop state can live near inventory menu state, but durable data should remain in the inventory store.

Verification:
- `cd frontend; npm run build`
- Frontend tests for successful drop, invalid drop, and failed backend response.

## Issue: Render Hotbar As Drop Targets In Inventory Menu

Epic: Hotbar And Equipment State
Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Add Drag And Drop Inventory Store Actions

Related to:
- Preserve Existing Hotbar Persistence API
- Remove Button-Based Hotbar Assignment From Inventory Menu

User story:
As a player, I want to drag inventory items into hotbar slots, so that assigning quick-use items is natural.

Acceptance criteria:
- Inventory menu shows all five hotbar slots.
- Dragging an item onto a hotbar slot assigns it.
- Dragging away or clearing a hotbar slot is supported.
- Existing number-key hotbar selection still works.
- Existing Sunny Town placement/tool-use behavior still reads selected hotbar item.

Implementation notes:
- Current API `PUT /api/student/hotbar` can assign item type by slot and may be enough for the first drag/drop slice.
- Current hotbar slots are item-type references, not stack references.

Verification:
- `cd frontend; npm run build`
- Manual: drag `pickaxe` or `stone_block` to a hotbar slot, select slot with number key, use tool/place block.

## Issue: Preserve Existing Hotbar Persistence API

Epic: Hotbar And Equipment State
Type: backend
Priority: P1

Blocked by:
- None

Related to:
- Render Hotbar As Drop Targets In Inventory Menu

User story:
As a developer, I want hotbar drag/drop to reuse existing durable hotbar behavior where possible, so that we do not create unnecessary backend churn.

Acceptance criteria:
- Existing `GET /api/student/hotbar` and `PUT /api/student/hotbar` continue working.
- Hotbar assignment validates item ownership.
- Hotbar slots return enough item metadata for the new slot component.
- If slotted inventory changes hotbar semantics, the decision is documented.

Implementation notes:
- Current implementation is in `internal/hq/inventory/hotbar.go`.
- Current frontend API is `frontend/src/api/hotbarApi.ts`.

Verification:
- `go test ./internal/hq/inventory`
- Existing Sunny Town hotbar manual smoke test.

## Issue: Render Equipment Slots As Drop Targets

Epic: Hotbar And Equipment State
Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Add Drag And Drop Inventory Store Actions

Related to:
- Preserve Existing Equipment Persistence API
- Add Character Preview Shell

User story:
As a player, I want to drag compatible gear onto equipment slots, so that equipping items feels visual and immediate.

Acceptance criteria:
- Equipment area shows gear, accessory, and tool slots as slot UI.
- Compatible inventory items can be dropped onto matching equipment slots.
- Incompatible drops show invalid feedback.
- Unequipping can be done through direct interaction without returning to debug-style buttons.
- Equipment changes still notify Sunny Town so other players see updated visuals.

Implementation notes:
- Current equipment API already validates slot compatibility.
- Current frontend sends `equipment_changed` after equip/unequip from `useSunnyTownInventoryActions.ts`.

Verification:
- `cd frontend; npm run build`
- Manual: equip/unequip hoodie, cap, and pickaxe; confirm avatar visuals update.

## Issue: Preserve Existing Equipment Persistence API

Epic: Hotbar And Equipment State
Type: backend
Priority: P1

Blocked by:
- None

Related to:
- Render Equipment Slots As Drop Targets
- Add Character Preview Shell

User story:
As a developer, I want equipment drag/drop to reuse existing server validation, so that incompatible gear cannot be equipped from the client.

Acceptance criteria:
- Existing `GET /api/student/equipment`, `POST /equip`, and `POST /unequip` continue working.
- Equipment responses include metadata needed by slot UI.
- Sunny Town internal equipment endpoint remains service-authenticated.
- Tool-use validation remains server-side.

Implementation notes:
- Current implementation is in `internal/hq/inventory/equipment.go`.
- Sunny Town loads equipment through `internal/hq/sunnytownbridge/http.go` and `internal/sunnytown/hqclient/client.go`.
- Tool ownership validation in `internal/sunnytown/server/client_gameplay.go` currently checks ownership by item key. Consider whether later slices should require equipped tool, selected hotbar tool, or both.

Verification:
- `go test ./internal/hq/inventory`
- `go test ./internal/sunnytown/server`

## Issue: Redesign Crafting Panel For Inventory Menu

Epic: Crafting UI Redesign
Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Render Player Inventory As A Grid

Related to:
- Add Storage Context To Crafting Recipe APIs
- Add Cookie Shop Input Storage

User story:
As a player, I want crafting to use the same visual item language as inventory, so that recipes are easy to understand.

Acceptance criteria:
- Crafting recipes show output icon, quantity, and recipe name.
- Ingredients show required/owned quantities with missing states.
- Craft button is disabled when requirements are missing.
- Craft result updates inventory and hotbar quantities.
- The UI has a reserved place for active ingredient source and output destination, even if only player inventory is supported initially.

Implementation notes:
- Current panel is inside `SunnyTownInventoryPanel.vue`.
- Current API client is `frontend/src/api/craftingApi.ts`.
- Current fallback recipe injection in `frontend/src/stores/craftingRecipes.ts` should be removed once server recipes are reliable.

Verification:
- `cd frontend; npm run build`
- Manual: craft `stone_block` from rocks and confirm inventory/hotbar updates.

## Issue: Add Storage Context To Crafting Recipe APIs

Epic: Storage-Aware Recipe Execution
Type: backend
Priority: P1

Blocked by:
- Add Stack Move Swap Merge And Split Operations

Related to:
- Add Cookie Shop Input Storage
- Add Player To Container Transfer Operations
- Redesign Crafting Panel For Inventory Menu

User story:
As a player or NPC system, I want recipes to consume from an explicit storage source and produce to an explicit destination, so that crafting works with personal inventory, shops, chests, and workstations without duplicate rules.

Acceptance criteria:
- Recipe availability can be calculated for a storage context.
- Recipe execution accepts explicit input/output storage descriptors or has a documented first-slice equivalent.
- Player inventory crafting still works.
- Missing ingredient errors are scoped to the selected storage context.
- The storage-agnostic executor remains shared.

Implementation notes:
- Build on `internal/hq/inventory/crafting.go`.
- Current `recipeStorage` interface has the right basic shape but no source/destination metadata.
- Avoid adding Cookie Keeper-specific recipe logic to `sunnytownbridge.Store.CommitNPCJobProduction`.

Verification:
- `go test ./internal/hq/inventory`
- Tests for player storage context and fake non-player storage context.

## Issue: Add Cookie Shop Input Storage

Epic: Storage-Aware Recipe Execution
Type: backend
Priority: P1

Blocked by:
- Add Storage Context To Crafting Recipe APIs

Related to:
- Add Player To Container Transfer Operations
- Add Ingredient-Aware Cookie Shop Production

User story:
As a developer, I want Cookie Shop ingredients to be stored in HQ-owned shop input storage, so that NPC production can later consume ingredients through the shared recipe executor.

Acceptance criteria:
- HQ persists input stock keyed by shop/storage identity.
- Input storage has capacity behavior or an explicit deferred capacity decision.
- Storage operations can consume input stock transactionally.
- Existing Cookie Shop output stock and purchases remain unchanged.
- No quantities are stored in fixture metadata or Sunny Town client state.

Implementation notes:
- Follow `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`.
- Keep `cookie-shop-input-chest` as the physical fixture anchor only.
- Add a migration after `0010_seed_cookie_keeper_shop_stock.sql`.

Verification:
- `go test ./internal/hq/inventory`
- `go test ./internal/hq/schema`

## Issue: Add Ingredient-Aware Cookie Shop Production

Epic: Storage-Aware Recipe Execution
Type: backend
Priority: P2

Blocked by:
- Add Cookie Shop Input Storage
- Add Storage Context To Crafting Recipe APIs

Related to:
- Redesign Crafting Panel For Inventory Menu

User story:
As a player, I want shop production to depend on ingredients eventually, so that the shop economy can become a real simulation instead of free stock generation.

Acceptance criteria:
- Cookie production can consume required inputs and produce output stock in one HQ transaction.
- Missing ingredients block output stock changes.
- Production ledger behavior stays idempotent.
- Blocked production has a debuggable result without double-producing on retries.

Implementation notes:
- Current NPC production path is `internal/sunnytown/server/npc_production.go` to `internal/hq/sunnytownbridge/store.go`.
- Do not make Sunny Town decide ingredient availability.
- Add recipe definitions to HQ catalog rather than Cookie Keeper-specific conditionals.

Verification:
- `go test ./internal/hq/sunnytownbridge`
- `go test ./internal/hq/inventory`
- `go test ./internal/sunnytown/server`

## Issue: Design General Container Storage Schema And Access Contract

Epic: Chest And Container Inventory
Type: design
Priority: P1

Blocked by:
- Design And Implement Slotted Player Inventory Persistence

Related to:
- Add Player To Container Transfer Operations
- Render Chest Inventory Grid
- Add Storage Context To Crafting Recipe APIs

User story:
As a developer, I want a clear container ownership and access model, so that chest transfers are durable and safe in multiplayer.

Acceptance criteria:
- Document whether containers are keyed by fixture ID, placed object ID, room/map/location, shop/storage owner, or another stable identity.
- Document who can open and modify a container.
- Document how Sunny Town validates distance/access before HQ mutation.
- Document how conflicts are handled when multiple players use the same container.
- Document how container storage can be used as a crafting source later.

Implementation notes:
- Current fixtures have ID, location ID, shop ID, storage role, and item key.
- General player chests may need different ownership fields than Cookie Shop chests.
- This should become an implementation note in `docs/current/ARCHITECTURE.md` after decisions land.

Verification:
- Documentation review against HQ/Sunny Town ownership boundaries.

## Issue: Add Player To Container Transfer Operations

Epic: Chest And Container Inventory
Type: backend
Priority: P1

Blocked by:
- Design General Container Storage Schema And Access Contract
- Add Stack Move Swap Merge And Split Operations

Related to:
- Render Chest Inventory Grid
- Add Storage Context To Crafting Recipe APIs

User story:
As a player, I want to move item stacks between my inventory and an opened chest, so that containers are useful for storage and future crafting workflows.

Acceptance criteria:
- Backend can load accessible container contents.
- Backend can transfer stack quantities from player inventory to container.
- Backend can transfer stack quantities from container to player inventory.
- Backend can swap or merge compatible stacks where applicable.
- All transfer operations are transactional.
- Access validation cannot be bypassed by client-supplied IDs alone.

Implementation notes:
- Decide whether public student APIs receive a Sunny Town access token/context or Sunny Town calls internal HQ after validating position.
- Reuse stack movement semantics from player inventory where possible.
- Include enough concurrency protection for simultaneous transfer attempts.

Verification:
- Backend tests for transfer, insufficient quantity, full target, invalid container, and conflict behavior.

## Issue: Render Chest Inventory Grid

Epic: Chest And Container Inventory
Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Add Player To Container Transfer Operations

Related to:
- Render Player Inventory As A Grid
- Redesign Crafting Panel For Inventory Menu

User story:
As a player, I want opening a chest to show my inventory and the chest inventory together, so that transfer by drag/drop is obvious.

Acceptance criteria:
- Opening a supported chest shows player grid and container grid together.
- Drag/drop transfers between grids.
- UI clearly labels player storage versus chest storage.
- Empty, occupied, hover, tooltip, and invalid-drop states match player inventory.
- Moving away or changing maps closes the container view.

Implementation notes:
- Current `SunnyTownChestPanel.vue` is read-only and shop-output-specific.
- Current `nearestSunnyTownChest` ignores input chests and general chests.
- This likely replaces or generalizes the current chest panel rather than extending it in place forever.

Verification:
- `cd frontend; npm run build`
- Manual: open a chest, transfer an item both directions, move away and confirm it closes.

## Issue: Add Character Preview Shell

Epic: Character Preview And Equipment UI
Type: frontend
Priority: P2

Blocked by:
- Render Equipment Slots As Drop Targets

Related to:
- Add Stats Ready Character Panel
- Preserve Existing Equipment Persistence API

User story:
As a player, I want to see my avatar near my equipment slots, so that gear changes feel visible and personal.

Acceptance criteria:
- Inventory menu includes a character preview area.
- Preview shows the current base avatar.
- Equipped visual keys can influence the preview where current art supports it.
- Layout leaves room for future stats.
- Equipment drag/drop remains usable near the preview.

Implementation notes:
- Sunny Town renderer already understands equipment visual keys for live players.
- The first slice can reuse or approximate existing avatar rendering rather than building a new art system.

Verification:
- `cd frontend; npm run build`
- Manual: equip/unequip items and confirm preview updates.

## Issue: Add Stats Ready Character Panel

Epic: Character Preview And Equipment UI
Type: frontend
Priority: P2

Blocked by:
- Add Character Preview Shell

Related to:
- Define Stats And Skills Progression Model

User story:
As a player, I want the character area to have room for stats later, so that future progression can be added without redesigning the whole menu again.

Acceptance criteria:
- Character preview layout reserves a stats-ready area.
- No fake stat progression is introduced.
- Placeholder state does not imply manual point allocation.
- Layout works on desktop and mobile sizes.

Implementation notes:
- The user direction is activity-driven progression, not manual stat distribution.

Verification:
- `cd frontend; npm run build`
- Responsive manual check.

## Issue: Add Tabbed E Menu Shell

Epic: Tabbed E Menu Framework
Type: frontend
Priority: P2

Blocked by:
- Render Player Inventory As A Grid

Related to:
- Add Character Preview Shell
- Define Stats And Skills Progression Model

User story:
As a player, I want the `E` menu to support multiple tabs eventually, so that inventory, crafting, stats, and skills can live in one coherent character menu.

Acceptance criteria:
- `E` opens a menu shell with an inventory/crafting tab.
- Existing close/Escape behavior still works.
- Gameplay inputs do not conflict while the menu is open.
- Placeholder tabs are added only if they clarify future structure without pretending features exist.

Implementation notes:
- Current menu state lives in `useSunnyTownInventoryActions.ts` and `SunnyTownPage.vue`.
- Keep first screen useful; do not build a landing page or purely explanatory menu.

Verification:
- `cd frontend; npm run build`
- Manual: open/close with `E` and Escape; verify movement/tool inputs are suppressed while open.

## Issue: Define Stats And Skills Progression Model

Epic: Stats And Skills Progression
Type: design
Priority: P2

Blocked by:
- None

Related to:
- Add Stats Ready Character Panel
- Add Ingredient-Aware Cookie Shop Production

User story:
As a designer, I want player and NPC stats/skills to progress through activity, so that character capability emerges from what they do rather than manual point allocation.

Acceptance criteria:
- Document shared player/NPC progression primitives.
- Document activity-to-skill progression examples.
- Document how recipes, tools, jobs, and actions can require skills/stats.
- Explicitly reject manual point distribution unless design direction changes.
- Identify first vertical slice for progression.

Implementation notes:
- This should remain separate from base inventory redesign.
- Future examples include mining, crafting, cooking/baking, and NPC work.

Verification:
- Documentation review.

## Open Product Questions

- Should player inventory slot count be fixed at first, upgradeable later, or derived from backpack/equipment?
- Should hotbar slots reference item type, inventory slot, or stack identity after slotted inventory exists?
- Should equipment consume the actual inventory slot identity or remain a separate view over owned item type?
- Should stack splitting ship in the first drag/drop slice or after move/swap/merge?
- Should open-chest crafting automatically count chest contents, require explicit source selection, or require a workstation-bound storage relationship?
- Should Cookie Shop input storage be general container storage with a shop owner, or a separate shop-specific table that later adapts into general storage?
- Should all chest transfer mutations go through Sunny Town internal validation, or can HQ public APIs validate a signed/short-lived interaction context?
- What is the first target inventory grid size?
- What is the first target chest grid size?
- What item art pipeline should replace CSS-only icon classes?

## Suggested First Implementation Tickets

Start with these in order:

1. Add Item Metadata Needed For Grid Inventory.
2. Design And Implement Slotted Player Inventory Persistence.
3. Add Stack Move Swap Merge And Split Operations.
4. Build Reusable Inventory Slot Component.
5. Render Player Inventory As A Grid.

Parallel after interface contracts:

- Preserve Existing Hotbar Persistence API can run early.
- Preserve Existing Equipment Persistence API can run early.
- Design General Container Storage Schema And Access Contract can run alongside frontend slot work.
- Define Stats And Skills Progression Model can run independently as design work.

Do not start chest transfer UI before the container schema/access contract exists. Do not start storage-aware crafting behavior before storage context and shop input storage decisions are made.

## Backlog Usage Notes

Treat this roadmap as a product and engineering backlog, not a sprint plan. The tickets can later be pulled into milestones, sprints, or GitHub Projects iterations once team capacity and sequencing are clear.

Recommended first backlog slice:

1. Add Item Metadata Needed For Grid Inventory.
2. Design Slotted Player Inventory Schema And Compatibility Plan.
3. Implement Slotted Player Inventory Persistence.
4. Add Inventory Stack Move, Swap, And Merge API.
5. Build Reusable Inventory Slot Component.
6. Render Sunny Town Inventory As A Grid.

Recommended second backlog slice:

1. Add Inventory Drag And Drop State And API Integration.
2. Replace Hotbar Assignment Buttons With Drop Targets.
3. Replace Equipment Buttons With Drop Targets.
4. Remove Legacy Button-Based Assignment UI.

Recommended later backlog slices:

- Crafting redesign.
- Container/chest storage and transfer.
- Storage-aware recipe execution.
- Cookie Shop input storage and ingredient-aware production.
- Character preview and stats-ready layout.
- Tabbed `E` menu shell.
- Stats and skills progression model.
