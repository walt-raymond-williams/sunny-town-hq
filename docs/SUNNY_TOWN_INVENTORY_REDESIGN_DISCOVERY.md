# Sunny Town Inventory Redesign Discovery

## Purpose

This document captures the design direction for redesigning the Sunny Town `E` menu from the current debug-functional inventory and crafting view into a game-style character management menu.

The immediate goal is not implementation. The goal is to run a focused discovery audit, then produce a second document containing a structured roadmap and GitHub-ready issue list with epics, dependencies, blockers, and related work.

## Current Direction

The existing inventory and crafting UI works well enough to prove that the APIs and gameplay logic function. It is not the target final experience.

The target experience should feel closer to inventory systems in games such as Minecraft, Stardew Valley, or similar life-sim/RPG interfaces:

- Inventory items appear in a visual grid, not a text-heavy list.
- Each item has an icon for quick recognition.
- Item stacks show quantities directly in the grid cell.
- Hovering an item shows a tooltip with the item name and useful details.
- Players can drag items around within the inventory grid.
- Players can drag items from inventory into hotbar slots.
- Players can drag compatible items from inventory into equipment slots.
- Players can drag items between their inventory and container inventories such as chests.
- The current "equip this to slot 1 / slot 2" style button flow should be replaced by direct drag-and-drop interaction.
- Crafting remains available in the same menu, but should feel like part of the game UI rather than a debug panel.

## Broader E Menu Vision

The `E` menu should eventually become a broader character/gameplay management menu, with inventory and crafting as the first major tab rather than the whole menu forever.

Likely future tabs:

- Inventory, equipment, hotbar, and crafting.
- Character stats.
- Skills and progression.
- Jobs or task capability.
- Chest or container inventory views.
- NPC character inspection or interaction views.
- Other gameplay systems as they emerge.

The first redesign should avoid hard-coding the overlay as only an inventory screen. It should leave room for a tabbed menu shell.

## Inventory Tab Target Shape

The inventory tab should eventually include:

- Character/avatar preview.
- Equipment slots near or around the avatar preview.
- Future stats-ready area near the character preview.
- Main inventory grid.
- Hotbar slot assignment.
- Crafting recipe area.
- Item tooltip support.
- Drag-and-drop movement between supported slot types.

A possible layout direction:

- Upper or left area: avatar preview, equipment slots, and future stats.
- Main area: inventory grid.
- Bottom area: hotbar.
- Side or lower area: crafting recipes, ingredients, and craft action.

This is not a locked layout. It is a product direction for the audit and later UI design work.

## Chest And Container Inventory Direction

Chest and container inventory should use the same grid interaction language as player inventory.

Existing Cookie Shop work is an important precedent. The Cookie Shop already treats a physical chest fixture as the in-world representation of HQ-owned durable shop storage. NPC production writes to durable shop output stock, and the shared HQ recipe execution foundation is already shaped around storage-agnostic inputs and outputs.

Target behavior:

- Opening a chest shows the chest inventory grid and the player's inventory grid together.
- Players can drag item stacks from player inventory into the chest.
- Players can drag item stacks from the chest into player inventory.
- Players can swap, merge, or move stacks using the same semantics as normal inventory movement where possible.
- Container slots should support item icons, quantities, hover tooltips, empty slot states, and invalid-drop feedback.
- The UI should make clear which grid belongs to the player and which grid belongs to the opened chest.
- Crafting should be designed around explicit storage sources and destinations, not hard-coded to only the player's personal inventory.
- Workstation, shop, and chest-backed crafting should be able to consume from selected or permitted storage endpoints once ownership and access rules are defined.

Design guardrails:

- Do not create a second local inventory source in Sunny Town fixture/client state.
- Durable quantities should live in HQ-owned storage tables or inventory tables.
- Physical chest fixtures should act as routing, interaction, and presentation metadata.
- Crafting access to chest contents should be explicit: the player must open, select, or be near a permitted storage/workstation according to server-validated rules.
- Avoid broad "all nearby chests are automatically available" behavior until the gameplay and ownership rules are intentionally designed.
- Multi-user chest conflict handling should be included enough to keep persistence safe, but broad shared-building permissions can remain later work.

The first goal is direct player-to-chest transfer plus a storage-aware crafting design that can expand from the Cookie Shop precedent. Full workstation and shared-storage crafting can be phased in after basic container movement and access validation are stable.

## Storage-Aware Crafting Direction

Crafting should become a recipe execution problem over one or more storage endpoints.

Current precedent to account for:

- Student crafting already uses HQ APIs and durable student inventory.
- HQ has a shared recipe catalog and a storage-agnostic recipe executor.
- Cookie Shop NPC production writes to HQ-owned `shop_stock_item` through idempotent service-authenticated production events.
- The Cookie Shop output chest is a physical fixture that represents durable shop output stock, not its own separate local inventory.
- The Cookie Shop input chest exists as a physical fixture. Durable input storage is being implemented as HQ-owned `shop_input_storage_item`, scoped by shop and item quantity, with the fixture remaining interaction/routing metadata.
- This row-based shop input storage is a bridge for recipe execution and future container UI. It should not be treated as the final visual grid/slot model, but the grid/container redesign should reuse or wrap it instead of creating a second Sunny Town-local quantity source.

Target direction:

- Recipes remain HQ-owned gameplay definitions.
- Recipe execution can consume from a specific storage adapter: player inventory, shop input storage, workstation storage, or another explicitly permitted container.
- Recipe execution can produce into a specific output adapter: player inventory, shop output stock, workstation output, or another explicitly permitted container.
- The UI should show which storage source is being used for ingredients.
- The UI should show where crafted outputs will go.
- Missing ingredient states should account for the active storage context, not only personal inventory.
- NPC production and player crafting should share recipe semantics where practical, even if their triggers and validation differ.

Open product decision:

- Whether player crafting with an open chest should automatically count chest contents, require selecting a storage source, or require using a workstation tied to that chest/storage owner.

## Character Preview And Equipment

When the inventory menu opens, the player should be able to see their character/avatar. As equipment changes, the preview should reflect the equipped items where possible.

Future goals:

- Show currently equipped clothing, tools, accessories, or gear.
- Layer visual equipment over the base avatar when item art exists.
- Display character stats in the same area later.
- Show stat changes or derived effects as equipment changes.

The first implementation can use placeholder preview behavior if the data and art pipeline are not ready.

## Stats, Skills, And Progression Direction

Stats and skills should be shared concepts for player characters and NPCs.

The player should not manually distribute stat or skill points. Progression should come from activity:

- Crafting improves relevant crafting or production skills.
- Cooking or baking improves relevant food-making skills.
- Mining improves mining or tool-use skills.
- Gathering, combat, building, or other actions can improve their own skill families.
- NPCs should use the same underlying progression model as players where appropriate.

Actions, tools, jobs, and recipes can later require specific skills, stats, tools, or knowledge.

Example:

- Making cookies may require a recipe, ingredients, a workstation, and a sufficient cooking/baking capability.
- Higher-level recipes or tools may require higher stats or skills.
- NPCs may be better or worse at tasks based on the same stat and skill model.

This progression system should be treated as a related but separate epic. It should influence the menu design, but it should not block the inventory UI redesign unless a specific requirement needs it.

## Proposed Epics

### Epic 1: Inventory Redesign Discovery

Audit the current frontend, backend, data model, and runtime ownership boundaries. Produce a dependency-aware implementation plan and GitHub issue list.

### Epic 2: Inventory Slot And Item Metadata Foundation

Add or confirm the data model needed for a grid inventory:

- Inventory slot positions.
- Stack counts and max stack behavior.
- Item icon keys.
- Display names and descriptions.
- Item categories or types.
- Equip compatibility metadata.

### Epic 3: Hotbar And Equipment State

Make hotbar and equipment state first-class and durable:

- Persisted hotbar slot assignments.
- Equipment slots and slot compatibility.
- API support for assigning, swapping, and clearing slots.
- Server validation of equipped tool state for gameplay effects.

### Epic 4: Inventory Grid UI

Replace the current card/list inventory presentation with a visual grid:

- Reusable inventory slot component.
- Item icons.
- Stack counts.
- Empty slot states.
- Selected, hover, focus, and disabled states.
- Loading and error states.

### Epic 5: Drag-And-Drop Inventory Interactions

Add direct manipulation:

- Drag within inventory.
- Drag inventory items to hotbar.
- Drag inventory items to equipment slots.
- Swap and merge behavior.
- Invalid-drop feedback.
- Pending or optimistic update handling.

### Epic 6: Crafting UI Redesign

Redesign crafting as part of the same game-style menu:

- Recipe list or grid.
- Recipe output preview.
- Ingredient requirements.
- Missing ingredient states.
- Craft action flow.
- Result feedback.
- Later support for skill, stat, tool, or workstation requirements.
- Storage-aware ingredient availability and output destination.
- Compatibility with the existing shared recipe execution foundation used by student crafting and planned Cookie Shop production.

### Epic 7: Chest And Container Inventory

Add container inventory as a compatible grid-based inventory surface:

- Chest inventory storage model.
- Open chest interaction flow.
- Player inventory plus chest inventory view.
- Drag-and-drop transfers between player and chest grids.
- Backend validation and persistence for container transfers.
- Storage context hooks that let crafting later consume from explicitly allowed chests, shop input storage, or workstation storage.

### Epic 8: Character Preview And Equipment UI

Add the visual character area:

- Avatar preview panel.
- Equipment slots arranged near the preview.
- Current equipment rendering.
- Placeholder or layered gear visuals.
- Stats-ready area.

### Epic 9: Tabbed E Menu Framework

Turn the current `E` overlay into a tabbed menu shell:

- Inventory/crafting/equipment tab.
- Placeholder stats/skills tab if useful.
- Menu open/close behavior.
- Input handling so movement and gameplay controls do not conflict with the menu.

### Epic 10: Stats And Skills Progression

Define and implement the deeper simulation model:

- Character stats.
- Skills.
- Activity-driven XP or progression.
- Player and NPC shared progression primitives.
- Skill-gated tools, jobs, recipes, and actions.
- Stats/skills menu tab.

## Proposed Phase Breakdown

### Phase 0: Discovery And Design Lock

Goal: understand current implementation and convert this design direction into concrete system requirements.

Deliverables:

- Current frontend component/store/API map.
- Current backend API/data model map.
- List of missing data model pieces.
- Proposed HQ vs Sunny Town ownership boundaries.
- Proposed implementation dependency map.
- GitHub-ready issue list with epics, blockers, and related issues.
- Risk list and recommended implementation order.

Questions to answer:

- Is inventory currently a flat list or slot-based?
- Are item stack counts modeled cleanly?
- Are hotbar slots persisted?
- Are equipment slots persisted?
- Do items have stable IDs, icon keys, categories, stackability, and compatibility metadata?
- Does crafting consume inventory through HQ or Sunny Town?
- How does the current shared recipe executor work, and what storage adapters already exist?
- How should the redesign reuse the current student crafting adapter and planned shop input/output adapters?
- Does Sunny Town need immediate access to equipped tool or hotbar state?
- What belongs to HQ durable state vs Sunny Town realtime state?
- Can avatar appearance be derived from current equipment, or do we need a placeholder preview first?
- Is there any current chest, storage, building, container, or map object inventory concept?
- If chests exist or are planned, does chest inventory belong to HQ durable state, Sunny Town realtime state, or a split model?
- What container transfer operations are needed: move, swap, merge, split, deposit all, or withdraw all?
- How should the server validate that a player is allowed to open or modify a specific chest?
- What can be reused from the Cookie Shop output chest, `shop_stock_item`, `shop_stock_ledger`, and stock capacity model?
- What durable shop input storage is still missing for Cookie Shop ingredient-aware production?
- Should player crafting with an open chest use chest contents automatically, through an explicit source selector, or only through a workstation/storage-owner relationship?

### Phase 1: Inventory Data Model Foundation

Goal: support grid inventory as real state, not only a frontend rearrangement of a flat list.

Likely work:

- Add inventory slot model.
- Add stack move, swap, split, and merge semantics as needed.
- Add or confirm item metadata needed by the UI.
- Add or update inventory layout APIs.
- Add backend tests for move, swap, and stack behavior.

### Phase 2: Hotbar And Equipment State

Goal: make hotbar and equipment state first-class.

Likely work:

- Add persisted hotbar slot assignments.
- Add equipment slot compatibility validation.
- Expose hotbar and equipment state to the frontend.
- Ensure Sunny Town gameplay uses server-validated equipped tool state.
- Add backend tests for invalid assignment and missing-item behavior.

### Phase 3: Inventory Grid UI

Goal: replace the current list/card inventory with the visual grid.

Likely work:

- Build reusable slot and item icon components.
- Render item icons, stack counts, and empty slots.
- Add item tooltip behavior.
- Preserve existing inventory loading and error handling.
- Add frontend verification for the grid.

### Phase 4: Drag-And-Drop Interaction

Goal: make item movement direct and game-like.

Likely work:

- Drag/drop within inventory.
- Drag/drop from inventory to hotbar.
- Drag/drop from inventory to equipment.
- Swap/merge behavior in the UI.
- Invalid-drop feedback.
- Pending or optimistic update handling.

### Phase 5: Crafting Redesign

Goal: keep crafting in the menu while making it feel like part of the final game UI.

Likely work:

- Redesign crafting panel layout.
- Add recipe list/grid with icons.
- Add recipe detail view.
- Add ingredient requirement and missing requirement states.
- Add craft action and result feedback.
- Prepare for later skill/stat/tool requirements.
- Add visible active storage context for ingredient availability and output destination.
- Preserve compatibility with current player-inventory crafting behavior.

### Phase 6: Chest And Container Inventory

Goal: support direct movement between player inventory and a chest/container grid, while shaping the storage model so crafting can consume from explicit storage endpoints later.

Likely work:

- Define chest/container ownership and persistence.
- Add or confirm container slot model.
- Add backend transfer APIs for player-to-container and container-to-player movement.
- Add validation for container access based on world position, ownership, permissions, or interaction state.
- Build a chest inventory view that shows player grid and chest grid together.
- Reuse item slot, icon, tooltip, stack, and drag/drop components from player inventory.
- Model chest storage so it can be passed into storage-aware crafting rules later.
- Include Cookie Shop input/output storage as a concrete reference case: current shop storage is item-quantity based HQ state, while future general-purpose containers may add slot/grid layout on top or through a separate compatible table.

### Phase 6A: Storage-Aware Recipe Execution And Shop Storage

Goal: extend the existing HQ recipe execution foundation so player crafting, NPC production, shop input storage, and chest/workstation storage can use compatible semantics.

Likely work:

- Audit current `internal/hq/inventory/crafting.go` recipe catalog and storage adapter shape.
- Reuse the durable shop input storage for Cookie Shop ingredients; do not duplicate it in fixture/client state.
- Add recipe storage adapters for shop input/output storage when the schema is ready.
- Define how recipe availability is calculated for a selected storage context.
- Define blocked-production behavior when NPC/shop work has missing inputs.
- Decide whether open-chest player crafting uses automatic access, explicit source selection, or workstation-bound access.
- Keep recipe definitions HQ-owned and avoid Sunny Town fixture/client quantities.

### Phase 7: Character Preview And Equipment View

Goal: show the player avatar and equipment visually.

Likely work:

- Add character preview panel.
- Add equipment slot UI around or near the preview.
- Add placeholder avatar rendering based on current player appearance.
- Add equipment-driven visual layers when item art exists.
- Add stats-ready layout area.

### Phase 8: Tabbed E Menu Shell

Goal: evolve the inventory overlay into a broader character menu.

Likely work:

- Add tabbed menu shell.
- Move inventory/crafting/equipment into first tab.
- Add placeholder stats/skills tab if appropriate.
- Preserve current open/close input behavior.
- Prevent input conflicts with movement and gameplay controls.

### Phase 9: Skills, Stats, And Use-Based Progression

Goal: define the deeper character simulation layer.

Likely work:

- Define character stats and skills schema.
- Add use-based skill progression.
- Share progression primitives between player and NPCs.
- Add requirements for tools, recipes, jobs, and actions.
- Add frontend stats/skills tab.

## Concurrency Notes

Good candidates for parallel work after discovery:

- Backend inventory slot model and APIs.
- Frontend inventory grid components using mocked or adapter-based data.
- Crafting UI redesign after shared item display components are defined.
- Chest/container storage discovery and ownership design.
- Storage-aware recipe execution and Cookie Shop input storage design.
- Stats and skills design document.

Work that should not be parallelized without a clear interface contract:

- Multiple agents modifying the same frontend inventory store or API client.
- Drag/drop UI before backend movement semantics are defined.
- Chest transfer UI before player inventory movement semantics are defined.
- Equipment visual preview before equipment metadata and visual keys are stable.
- Skill-gated crafting before base crafting and recipe requirement UI are stable.
- Broad automatic chest-aware crafting before storage ownership, access validation, and active storage context rules are stable.

## Immediate Next Step

Create and assign a discovery/audit ticket.

### Ticket Draft: Audit Current Sunny Town Inventory/Crafting/Equipment Implementation For Redesign

Goal:

Produce a concrete implementation plan for converting the current debug-style inventory/crafting UI into a game-style `E` menu with grid slots, drag/drop hotbar assignment, equipment slots, crafting, character preview, and future stats support.

Deliverables:

- Current frontend component map.
- Current frontend state/store/composable map.
- Current API client map.
- Current backend route/handler/service map.
- Current database/schema map.
- Current item metadata map.
- Current equipment and hotbar behavior summary.
- Current crafting behavior summary.
- Current chest/container/storage behavior summary, if any exists.
- Current Cookie Shop storage and NPC production behavior summary.
- Current shared recipe execution behavior summary.
- Recommendation for chest/container inventory ownership and transfer APIs.
- Recommendation for storage-aware crafting, including open chest, shop input storage, and workstation-adjacent crafting.
- HQ vs Sunny Town ownership recommendation.
- Data model gap list.
- UI/component gap list.
- API gap list.
- Risk list.
- Dependency map.
- GitHub-ready issue list grouped by epic.

Out of scope:

- No UI rewrite.
- No backend schema changes.
- No API behavior changes.
- No gameplay behavior changes.

Suggested verification:

- Use `rg` to find inventory, equipment, hotbar, crafting, and item metadata code.
- Compare findings against `docs/INVENTORY_AND_EQUIPMENT.md`.
- Run existing focused tests only if needed to confirm current behavior.

## Required Output From Discovery

The discovery audit should produce a follow-up document, tentatively named:

`docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`

That document should contain:

- Final epics.
- GitHub-ready issue titles.
- User story for each issue.
- Acceptance criteria for each issue.
- Implementation notes for each issue.
- Blocked-by relationships.
- Related-to relationships.
- Recommended assignment/concurrency groups.
- Suggested verification commands.
- Open product/design questions.

Each proposed issue should use this shape:

```md
## Issue: <title>

Epic: <epic name>
Type: <backend | frontend | full-stack | design | documentation>
Priority: <P0 | P1 | P2>
Blocked by:
- <issue title or "None">

Related to:
- <issue title or "None">

User story:
As a <player/developer/designer>, I want <capability>, so that <outcome>.

Acceptance criteria:
- <observable result>
- <observable result>

Implementation notes:
- <specific files, APIs, schemas, or components if known>

Verification:
- <command or manual check>
```
