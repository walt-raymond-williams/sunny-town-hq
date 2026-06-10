# Sunny Town Inventory Redesign Ticket Backlog

## Purpose

This document contains GitHub-ready tickets for the Sunny Town inventory redesign backlog. These are not assigned to a sprint yet. Future milestones or sprints can draw from this backlog.

## Epic: Inventory Grid Foundation

### Issue: Add Item Metadata Needed For Grid Inventory

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/8

Type: backend
Priority: P0

Blocked by:
- None

Related to:
- Build Reusable Inventory Slot Component
- Render Sunny Town Inventory As A Grid

User story:
As a player, I want items to have stable icons, display text, stack rules, and categories, so that I can recognize items visually in a grid inventory.

Acceptance criteria:
- Item catalog data can represent an inventory icon key or asset key.
- Item catalog data can represent max stack size, or the decision to defer max stack size is documented.
- Inventory, hotbar, equipment, and crafting responses expose the metadata needed by slot UI.
- `visual_key` remains available for avatar/equipment rendering and is not accidentally overloaded without an explicit decision.
- Existing item API consumers remain backward compatible or have a documented migration path.

Implementation notes:
- Start from `deploy/postgres/migrations/0003_inventory_equipment.sql`.
- Relevant backend files: `internal/hq/inventory/inventory.go`, `hotbar.go`, `equipment.go`, `crafting.go`.
- Relevant frontend normalizers: `frontend/src/api/inventoryApi.ts`, `hotbarApi.ts`, `equipmentApi.ts`, `craftingApi.ts`.

Verification:
- `go test ./internal/hq/inventory`
- `go test ./internal/hq/schema`
- `cd frontend; npm run build`

### Issue: Design Slotted Player Inventory Schema And Compatibility Plan

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/9

Type: design
Priority: P0

Blocked by:
- Add Item Metadata Needed For Grid Inventory

Related to:
- Implement Slotted Player Inventory Persistence
- Add Inventory Stack Move, Swap, And Merge API

User story:
As a developer, I want a clear slotted inventory schema plan, so that the grid inventory can be implemented without breaking existing crafting, hotbar, mining, placement, and inventory quantity flows.

Acceptance criteria:
- Decision is documented for whether `student_inventory_item` remains an aggregate table, becomes a cache/view, or is replaced.
- Proposed slot table shape is documented.
- Migration/backfill strategy is documented.
- Compatibility plan covers current aggregate quantity reads, especially Sunny Town internal inventory quantity checks.
- Initial player inventory slot count is selected or explicitly deferred.
- Stack splitting is either included or explicitly deferred from the first movement API.

Implementation notes:
- Current table `student_inventory_item` stores one quantity row per `(app_user_id, item_type_id)`.
- Sunny Town currently uses `/api/internal/sunny-town/inventory-quantity` for placement validation.
- Do this as a short design note before schema implementation.

Verification:
- Documentation review against `docs/INVENTORY_AND_EQUIPMENT.md`, `docs/current/DATABASE.md`, and `docs/current/ARCHITECTURE.md`.

### Issue: Implement Slotted Player Inventory Persistence

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/10

Type: backend
Priority: P0

Blocked by:
- Design Slotted Player Inventory Schema And Compatibility Plan

Related to:
- Add Inventory Stack Move, Swap, And Merge API
- Render Sunny Town Inventory As A Grid

User story:
As a player, I want my inventory items to have stable grid positions, so that items stay where I put them.

Acceptance criteria:
- Player inventory slots are persisted durably in HQ.
- Occupied slots store item type and quantity.
- Empty slots can be represented in load responses.
- Existing student inventory quantities are migrated or adapted into initial slots.
- Existing gameplay mutations preserve correct total owned quantities.
- Internal inventory quantity checks still return correct totals by item key.

Implementation notes:
- Add a migration under `deploy/postgres/migrations/`.
- Add focused persistence functions in `internal/hq/inventory`.
- Preserve HQ ownership of durable inventory state.
- Do not store player inventory quantities in Sunny Town runtime state or client state.

Verification:
- `go test ./internal/hq/inventory`
- `go test ./internal/hq/schema`
- Focused integration test for migration/adaptation and aggregate quantity compatibility.

### Issue: Add Inventory Stack Move, Swap, And Merge API

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/11

Type: backend
Priority: P0

Blocked by:
- Implement Slotted Player Inventory Persistence

Related to:
- Add Inventory Drag And Drop State And API Integration
- Add Player To Container Transfer Operations

User story:
As a player, I want to rearrange item stacks directly, so that inventory organization feels like a real game inventory.

Acceptance criteria:
- Backend supports moving a stack from one inventory slot to another.
- Backend supports swapping occupied inventory slots.
- Backend supports merging compatible stacks up to max stack size when max stack size exists.
- Invalid source slots, destination slots, item types, and quantities return clear client-safe errors.
- Operations are transactional.
- Stack splitting is either implemented or explicitly deferred to a later ticket.

Implementation notes:
- Prefer a source/destination descriptor shape that can later extend to containers.
- Keep optimistic frontend behavior out of scope until backend semantics are tested.
- Add API handler coverage in `internal/hq/inventory/http.go` or a focused companion file if the package style supports it.

Verification:
- Unit tests for move, swap, merge, overflow, invalid slot, empty slot, and insufficient quantity.
- `go test ./internal/hq/inventory`

## Epic: Inventory Grid UI

### Issue: Build Reusable Inventory Slot Component

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/12

Status: Complete
Completed in: `78f9192`

Type: frontend
Priority: P0

Blocked by:
- Add Item Metadata Needed For Grid Inventory

Related to:
- Render Sunny Town Inventory As A Grid
- Replace Hotbar Assignment Buttons With Drop Targets
- Replace Equipment Buttons With Drop Targets
- Render Chest Inventory Grid

User story:
As a player, I want each item to appear in a compact slot with an icon and quantity, so that I can scan my belongings quickly.

Acceptance criteria:
- Slot component renders empty, occupied, selected, hover, disabled, pending, and invalid-drop states.
- Occupied slots show item icon and stack quantity.
- Tooltip shows item name and description.
- Component has stable dimensions and does not resize based on content.
- Component can be reused for player inventory, hotbar, equipment slots, crafting ingredients, and containers.

Implementation notes:
- Existing CSS icon classes live in `frontend/src/style.css`.
- Consider a component path under `frontend/src/features/sunny-town/` unless a shared inventory component directory is introduced.
- Keep this component display-focused; drag/drop behavior can be layered in a later ticket.

Verification:
- `cd frontend; npm run build`
- `cd frontend; npm run test`

### Issue: Render Sunny Town Inventory As A Grid

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/13

Type: frontend
Priority: P0

Blocked by:
- Build Reusable Inventory Slot Component
- Implement Slotted Player Inventory Persistence

Related to:
- Add Inventory Drag And Drop State And API Integration
- Redesign Crafting Panel For Inventory Menu

User story:
As a player, I want my Sunny Town inventory to appear as a grid, so that the `E` menu feels like a final game interface instead of a debug list.

Acceptance criteria:
- `SunnyTownInventoryPanel.vue` renders inventory as a grid of stable slots.
- Empty slots are visible.
- Occupied slots show icon, quantity, and tooltip.
- Existing inventory loading and error states still work.
- Existing crafting, equipment, and hotbar capabilities are not regressed.
- Layout works on desktop and mobile viewports.

Implementation notes:
- Current state lives in `frontend/src/stores/studentInventory.ts`.
- Current overlay lives in `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`.
- If frontend work starts before backend slot APIs land, use a narrow documented adapter contract.

Verification:
- `cd frontend; npm run build`
- Manual Sunny Town check: open `E`, verify grid, quantities, empty slots, and tooltips.

## Epic: Direct Inventory Manipulation

### Issue: Add Inventory Drag And Drop State And API Integration

Type: frontend
Priority: P1

Blocked by:
- Add Inventory Stack Move, Swap, And Merge API
- Render Sunny Town Inventory As A Grid

Related to:
- Replace Hotbar Assignment Buttons With Drop Targets
- Replace Equipment Buttons With Drop Targets

User story:
As a player, I want to drag items around my inventory, so that organizing items feels direct.

Acceptance criteria:
- Store or composable exposes drag start, drag cancel, and drop behavior.
- Inventory-to-inventory drops call durable backend move APIs.
- UI handles pending moves and failed moves without losing state.
- Invalid drops are visibly rejected.
- Keyboard or accessible fallback is considered or documented.

Implementation notes:
- Current `useSunnyTownInventoryActions.ts` is action-button oriented.
- Durable data should remain owned by `studentInventory.ts`.
- Keep hotbar/equipment drops as related but separate tickets if the first implementation is already large.

Verification:
- `cd frontend; npm run build`
- Frontend tests for successful drop, invalid drop, and failed backend response.

### Issue: Replace Hotbar Assignment Buttons With Drop Targets

Type: frontend
Priority: P1

Blocked by:
- Add Inventory Drag And Drop State And API Integration

Related to:
- Remove Legacy Button-Based Assignment UI

User story:
As a player, I want to drag items into hotbar slots, so that assigning quick-use items feels natural.

Acceptance criteria:
- Inventory menu shows all hotbar slots as drop targets.
- Dragging an inventory item onto a hotbar slot assigns it.
- Clearing a hotbar slot is supported.
- Existing number-key hotbar selection still works.
- Existing Sunny Town tool use and placement behavior still read selected hotbar item.

Implementation notes:
- Current API `PUT /api/student/hotbar` may be sufficient for this slice.
- Current hotbar slots reference item type, not stack identity.
- Do not change hotbar persistence semantics unless the slotted inventory design explicitly requires it.

Verification:
- `cd frontend; npm run build`
- Manual: drag `pickaxe` or `stone_block` to a hotbar slot, select with number key, use tool/place block.

### Issue: Replace Equipment Buttons With Drop Targets

Type: frontend
Priority: P1

Blocked by:
- Add Inventory Drag And Drop State And API Integration

Related to:
- Remove Legacy Button-Based Assignment UI
- Add Character Preview Shell

User story:
As a player, I want to drag compatible gear onto equipment slots, so that equipping items feels visual and immediate.

Acceptance criteria:
- Equipment area shows gear, accessory, and tool slots as drop targets.
- Compatible inventory items can be dropped onto matching equipment slots.
- Incompatible drops show invalid feedback.
- Unequipping is supported without relying on the old debug-style item card buttons.
- Equipment changes still notify Sunny Town so other players see updated visuals.

Implementation notes:
- Existing equipment API already validates slot compatibility.
- Current frontend sends `equipment_changed` after equipment changes.
- Keep server validation authoritative.

Verification:
- `cd frontend; npm run build`
- Manual: equip/unequip hoodie, cap, and pickaxe; confirm player visuals update.

### Issue: Remove Legacy Button-Based Assignment UI

Type: frontend
Priority: P1

Blocked by:
- Replace Hotbar Assignment Buttons With Drop Targets
- Replace Equipment Buttons With Drop Targets

Related to:
- Render Sunny Town Inventory As A Grid

User story:
As a player, I want the inventory menu to use direct manipulation instead of debug buttons, so that the final UI feels coherent.

Acceptance criteria:
- Item cards no longer show `Slot N` hotbar assignment buttons.
- Item cards no longer show old `Wear` buttons once equipment drop targets are working.
- Clear/unequip affordances remain available in the new slot UI.
- No existing hotbar/equipment behavior is lost.

Implementation notes:
- This is intentionally separate so legacy controls can remain temporarily while drag/drop ships.
- Keep any accessibility fallback if it was intentionally added in the drag/drop tickets.

Verification:
- `cd frontend; npm run build`
- Manual: assign hotbar, equip gear, clear hotbar, unequip gear.

## Epic: Crafting UI Redesign

### Issue: Redesign Crafting Panel For Inventory Menu

Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Render Sunny Town Inventory As A Grid

Related to:
- Add Storage Context To Crafting Recipe APIs
- Add Cookie Shop Input Storage

User story:
As a player, I want crafting to use the same visual item language as inventory, so that recipes are easy to understand.

Acceptance criteria:
- Recipes show output icon, output quantity, and recipe name.
- Ingredients show required and owned quantities with missing states.
- Craft button is disabled when requirements are missing.
- Craft result updates inventory and hotbar quantities.
- UI has a reserved place for ingredient source and output destination, even if only player inventory is supported initially.

Implementation notes:
- Current panel is inside `SunnyTownInventoryPanel.vue`.
- Current API client is `frontend/src/api/craftingApi.ts`.
- Current fallback recipe injection in `frontend/src/stores/craftingRecipes.ts` should be removed once server recipe loading is reliable.

Verification:
- `cd frontend; npm run build`
- Manual: craft `stone_block` from rocks and confirm inventory/hotbar updates.

## Epic: Chest And Container Inventory

### Issue: Design General Container Storage Schema And Access Contract

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/26

Type: design
Priority: P1

Blocked by:
- Design Slotted Player Inventory Schema And Compatibility Plan

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
- Decisions should eventually move into `docs/current/ARCHITECTURE.md`.

Verification:
- Documentation review against HQ/Sunny Town ownership boundaries.

### Issue: Add Player To Container Transfer Operations

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/29

Type: backend
Priority: P1

Blocked by:
- Design General Container Storage Schema And Access Contract (#26)
- Add Inventory Stack Move, Swap, And Merge API

Related to:
- Render Chest Inventory Grid
- Add Storage Context To Crafting Recipe APIs

User story:
As a player, I want to move item stacks between my inventory and an opened chest, so that containers are useful for storage and future crafting.

Acceptance criteria:
- Backend can load accessible container contents.
- Backend can transfer stack quantities from player inventory to container.
- Backend can transfer stack quantities from container to player inventory.
- Backend can swap or merge compatible stacks where applicable.
- Transfer operations are transactional.
- Access validation cannot be bypassed by client-supplied IDs alone.

Implementation notes:
- Decide whether Sunny Town calls an internal HQ endpoint after validating position or whether HQ validates a short-lived interaction context.
- Reuse stack movement semantics from player inventory.
- Include enough concurrency protection for simultaneous transfer attempts.

Verification:
- Backend tests for transfer, insufficient quantity, full target, invalid container, and conflict behavior.

### Issue: Render Chest Inventory Grid

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/30

Type: frontend
Priority: P1

Blocked by:
- Build Reusable Inventory Slot Component
- Add Player To Container Transfer Operations

Related to:
- Render Sunny Town Inventory As A Grid
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
- This likely replaces or generalizes the current chest panel.

Verification:
- `cd frontend; npm run build`
- Manual: open a chest, transfer an item both directions, move away and confirm it closes.

## Epic: Storage-Aware Recipe Execution

### Issue: Add Storage Context To Crafting Recipe APIs

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/25

Type: backend
Priority: P1

Blocked by:
- Add Inventory Stack Move, Swap, And Merge API (#11, closed)

Related to:
- Add Cookie Shop Input Storage
- Add Player To Container Transfer Operations
- Redesign Crafting Panel For Inventory Menu

User story:
As a player or NPC system, I want recipes to consume from an explicit storage source and produce to an explicit destination, so that crafting works with personal inventory, shops, chests, and workstations without duplicate rules.

Acceptance criteria:
- Recipe availability can be calculated for a selected storage context.
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

### Issue: Add Cookie Shop Input Storage

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/31

Type: backend
Priority: P1

Blocked by:
- Add Storage Context To Crafting Recipe APIs

Related to:
- Add Ingredient-Aware Cookie Shop Production
- Add Player To Container Transfer Operations

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

### Issue: Add Ingredient-Aware Cookie Shop Production

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/32

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

## Epic: Character Preview And Menu Expansion

### Issue: Add Character Preview Shell

Type: frontend
Priority: P2

Blocked by:
- Replace Equipment Buttons With Drop Targets

Related to:
- Add Stats Ready Character Panel
- Add Tabbed E Menu Shell

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
- First slice can reuse or approximate existing avatar rendering rather than building a new art system.

Verification:
- `cd frontend; npm run build`
- Manual: equip/unequip items and confirm preview updates.

### Issue: Add Stats Ready Character Panel

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
- Player direction is activity-driven progression, not manual stat distribution.

Verification:
- `cd frontend; npm run build`
- Responsive manual check.

### Issue: Add Tabbed E Menu Shell

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/27

Type: frontend
Priority: P2

Blocked by:
- Render Sunny Town Inventory As A Grid

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
- Keep first screen useful.

Verification:
- `cd frontend; npm run build`
- Manual: open/close with `E` and Escape; verify movement/tool inputs are suppressed while open.

## Epic: Stats And Skills Progression

### Issue: Define Stats And Skills Progression Model

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/28

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
