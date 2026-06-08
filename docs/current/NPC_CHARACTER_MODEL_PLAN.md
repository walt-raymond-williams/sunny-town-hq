# NPC Character Model Plan

This document tracks the investigation, product intent, user stories, requirements, and phased plan for evolving Sunny Town NPCs into simulation-controlled town characters.

Status: `Phase 3A implemented; static NPCs can be backed by durable character rows`

## Fresh-Agent Handoff

Read this section first if picking up the NPC character feature cold.

The project is moving Sunny Town toward a small RimWorld-like living-town simulation. The immediate goal is not full AI. The immediate goal is to make NPCs real durable characters, using the same `sunny_town_character` identity foundation that now exists for student/player characters.

Current foundation:

- `deploy/postgres/migrations/0006_sunny_town_characters.sql` adds `sunny_town_character`.
- `deploy/postgres/migrations/0007_sunny_town_npc_characters.sql` adds `sunny_town_npc_character` for stable room-scoped NPC keys and seeds `mayor-sunny`.
- `internal/hq/characters` owns durable Sunny Town character identity helpers.
- Student sync creates/updates player character rows through `internal/hq/characters.EnsurePlayer`.
- Static NPC sync creates/updates NPC character rows through `internal/hq/characters.EnsureNPCs`.
- HQ Sunny Town session creation returns/signs `character_id`.
- Sunny Town WebSocket auth requires `character_id`.
- Realtime player IDs are now based on `character_id`, while current wallet/inventory/map-object APIs still use `app_user_id`.
- Sunny Town startup and WebSocket join paths call HQ's internal NPC ensure endpoint and merge returned `character_id` values into map NPCs without changing map-defined position, facing, dialogue, shop, or activity behavior.

Current branch/worktree notes:

- Feature foundation was committed as `30ea53a Add Sunny Town character identity foundation`.
- As of the last planning update, untracked local files may include `docs/PLAYWRIGHT_E2E_TEST_PLAN.md`, `hq-local.err.log`, and `hq-local.out.log`. Do not stage logs unless explicitly asked.

Latest completed task:

Implement **Phase 3A: NPC Character Records For Static NPCs**.

The target slice is deliberately modest: make current map-defined NPCs, starting with `mayor-sunny`, backed by durable `sunny_town_character` rows while preserving existing behavior. Mayor Sunny can keep standing still and talking. The win is that he becomes a real NPC character, not just map JSON.

Phase 3A produced:

- A durable way to create or ensure NPC character rows.
- A stable `sunny_town_npc_character` mapping from room-scoped map NPC keys to NPC character records.
- Sunny Town loading NPC character identity from HQ through `/api/internal/sunny-town/npc-characters/ensure`.
- Existing dialogue/shop/schoolwork interactions still working.
- Tests proving current map NPC behavior did not regress.

Recommended implementation shape:

1. Extend `sunny_town_character` or add a small companion table to store a stable NPC key.
2. Add `EnsureNPC`/`EnsureNPCs` in `internal/hq/characters`.
3. Seed or ensure `mayor-sunny` as an NPC character for `sunny-town-main`.
4. Add an HQ internal Sunny Town endpoint or startup path for Sunny Town to load NPC characters by `room_id`.
5. In Sunny Town, merge durable character identity with map JSON defaults for position, facing, dialogue, shop, and activity.
6. Keep NPCs static in this slice; do not add autonomous movement yet.

Important constraints:

- HQ owns durable character state.
- Sunny Town owns live realtime projections.
- Sunny Town must not write HQ tables directly.
- Do not migrate inventory/equipment in Phase 3A.
- Do not build motives, pathfinding, schedules, or economic loops yet.
- Preserve map JSON compatibility so current maps keep loading.

Recommended verification:

```powershell
go test ./...
cd frontend
npm run build
```

If runtime smoke testing is practical after implementation:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

## Feature Intent

Sunny Town should evolve from a multiplayer world with player characters and mostly static shopkeepers into a town simulation with durable citizens.

The central design rule is that players and NPCs should share the same underlying character model wherever practical. A player character is controlled by a human client and linked to a student/account. An NPC character is controlled by simulation logic and linked to town/world state. Both should be able to use shared character systems such as inventory, equipment, health, skills, XP, position, jobs, food, and future stats.

This avoids implementing future mechanics twice. When Sunny Town gains health, mining skill, farming skill, stamina, hunger, experience points, or equipment effects, those systems should attach to a shared character concept rather than separate player-only and NPC-only models.

The inspiration is closer to a small RimWorld colony than full Dwarf Fortress scale. The target is not hundreds of simulated citizens with deep historical simulation. The target is a small number of recognizable town residents whose visible routines, needs, relationships, work, rest, eating, and social behavior make the town feel alive.

Emergence should come from simple systems interacting, not from a huge first-pass AI. A few NPCs deciding between eating, resting, working, wandering, and talking can create a strong sense of life if those decisions are visible, consistent, and tied to shared town systems.

Future movement, location tags, pathing, and drive-based behavior are tracked in `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`. That work should come after static NPCs are backed by durable character rows.

## Current Assumptions To Verify

- HQ owns durable student/account/economy/inventory/equipment state.
- Sunny Town owns live realtime world state, including connected players, accepted positions, map membership, resource validation, and transitions.
- Existing shopkeepers are closer to static world fixtures than simulated citizens.
- Student inventory and equipment currently use student/account identifiers.
- Player movement and gameplay effects are server-validated by Sunny Town.
- Sunny Town must not write the HQ database directly; durable changes go through service-authenticated HQ internal endpoints.

## Proposed Concept Model

```text
Character
  Shared state and capabilities for person-like world actors.
  Examples: identity, display name, appearance, inventory, equipment, stats,
  skills, XP, health, needs, current map, position, and capabilities.

PlayerCharacter
  A Character controlled by a human client and linked to app user/student state.
  Examples: account link, permissions, session ownership, homework access.

NPCCharacter
  A Character controlled by simulation logic and linked to town state.
  Examples: AI controller, home, workplace, schedule, motives, job role.
```

The controller should differ, but common mechanics should not fork unless there is a real domain reason.

## Non-Goals For The First Pass

- Do not implement full colony-sim AI immediately.
- Do not move homework into the shared character model.
- Do not let Sunny Town write directly to HQ database tables.
- Do not create duplicate player and NPC versions of health, skills, XP, inventory, or equipment.
- Do not introduce a parallel map/cell abstraction unless the Sunny Town architecture changes intentionally.
- Do not aim for Dwarf Fortress population scale in the initial design.
- Do not persist every transient AI thought, pathing choice, or short-lived desire.

## User Stories

### STORY-NPC-001: NPCs Exist As Real Town Characters

As a player, I want shopkeepers and future residents to exist as real characters in the town so that the world feels populated by citizens, not only static objects.

Acceptance criteria:

- NPCs have durable identity and display information.
- NPCs can appear in Sunny Town alongside connected players.
- NPCs can be addressed by stable identifiers rather than only map fixture data.
- Existing static shopkeepers can be migrated or wrapped into the NPC model over time.

### STORY-NPC-002: Shared Inventory

As a designer, I want NPCs and player characters to use the same inventory concepts so that item ownership, trading, shops, eating, and work outputs do not require duplicate systems.

Acceptance criteria:

- The inventory model can represent items owned by a character, whether player-controlled or simulation-controlled.
- Existing student inventory behavior remains intact.
- NPC inventory can be persisted through HQ-owned durable state or another explicitly documented durable owner.
- Equipment rules can apply to NPCs and players where the item and slot rules are shared.

### STORY-NPC-003: Shared Stats

As a developer, I want future stats like health, stamina, hunger, and mood to attach to shared character state so that gameplay mechanics work consistently for players and NPCs.

Acceptance criteria:

- Shared character stats are modeled independently from login/session state.
- Player-only schoolwork state remains separate from shared character stats.
- NPC-only motive state can build on top of shared stats without redefining them.
- Sunny Town validates gameplay effects using server-authoritative character state.

### STORY-NPC-004: Shared Skills And XP

As a designer, I want skills and XP to be implemented once so that both students and NPC workers can improve at town activities.

Acceptance criteria:

- Skills and XP are keyed to the shared character model.
- A player and an NPC can both gain progress in the same skill, such as mining or farming.
- Skill effects are evaluated by shared gameplay rules.
- Student academic progress and homework grading remain separate from town skills.

### STORY-NPC-005: NPC Homes And Workplaces

As a town simulation designer, I want NPCs to have homes and workplaces so that residents can follow routines and participate in the town economy.

Acceptance criteria:

- NPCs can be assigned a home location.
- NPCs can be assigned a workplace or role.
- Assignments reference existing map/portal concepts rather than a separate cell system.
- NPCs can use these assignments for future routines and behavior planning.

### STORY-NPC-006: Motives And Goals

As a player, I want NPCs to have needs, motives, and goals so that the town feels alive and reacts to its own conditions.

Acceptance criteria:

- NPCs can track motive state such as hunger, energy, work need, or social desire.
- NPC behavior can choose simple goals from motive state.
- Motive updates are deterministic and cheap enough for multiplayer runtime.
- The simulation can be paused, bounded, or degraded safely if runtime load becomes high.

### STORY-NPC-007: Multiplayer Coexistence

As a player in multiplayer, I want NPCs and other players to appear and act in the same world without desync or client authority bugs.

Acceptance criteria:

- Sunny Town remains authoritative over live movement, positions, and gameplay validation.
- Browser messages remain requests, not authority.
- NPC state changes are broadcast to clients through Sunny Town realtime messages.
- Player and NPC interactions use server-validated positions and capabilities.

### STORY-NPC-008: Economy Participation

As a player, I want NPCs to gather, consume, trade, and produce goods so that the town economy becomes more than static shops.

Acceptance criteria:

- NPCs can hold goods in inventory.
- NPCs can consume items, such as food, through shared item rules.
- NPCs can produce or gather resources through server-validated gameplay rules.
- Shopkeeper behavior can eventually use NPC inventory or workplace stock.

## Requirements

### REQ-CHAR-001: Shared Character Identity

Status: `Partially implemented`

The system must define a shared character identity for player-controlled and simulation-controlled town actors.

Acceptance criteria:

- Every character has a stable character ID.
- Character identity is separate from app login/session identity.
- A player character can link to an app user/student. `Implemented for students through sunny_town_character`.
- An NPC character can link to simulation-owned town data. `Implemented for static NPC keys through sunny_town_npc_character`.

### REQ-CHAR-002: Shared Character-Owned Inventory

Status: `Planned`

Inventory-capable gameplay must be modeled around character ownership rather than hard-coded student-only ownership.

Acceptance criteria:

- Player inventory behavior remains compatible with existing student inventory APIs.
- NPC inventory does not require a parallel item catalog.
- Shared inventory code can enforce item quantities, equipment eligibility, and consumption rules.
- Durable inventory ownership is documented clearly before schema changes.

### REQ-CHAR-003: Shared Equipment

Status: `Planned`

Equipment slots and equipped-item validation should work for any character that can equip items.

Acceptance criteria:

- Equipment slot definitions are shared.
- Equipping still requires ownership of the item.
- Tool checks, such as mining requiring a pickaxe, can apply to players and NPCs.
- Player-facing equipment APIs remain stable unless intentionally migrated.

### REQ-CHAR-004: Shared Stats, Health, Skills, And XP

Status: `Planned`

Future stats and progression systems must be implemented once against shared character state.

Acceptance criteria:

- Health, skills, XP, stamina, hunger, or similar stats are not duplicated into player-only and NPC-only tables or structs.
- Player-only academic progress remains outside the shared character progression model.
- Gameplay systems read shared character stats where relevant.
- Sunny Town validates live stat-changing actions before durable writes occur.

### REQ-NPC-001: NPC Controller Layer

Status: `Planned`

NPC behavior must be implemented as a controller over shared character state.

Acceptance criteria:

- NPC AI/motives/schedules are separate from base character state.
- NPC controllers can issue movement and action intents through Sunny Town server logic.
- NPC controllers do not bypass gameplay validation.
- The first implementation can support static or minimal behavior before full routines.

### REQ-NPC-002: Homes, Workplaces, And Roles

Status: `Planned`

NPCs must be able to reference home, work, and role assignments for future town simulation.

Acceptance criteria:

- NPC home/work assignments reference maps, portals, or world locations that already exist.
- Role data can describe jobs such as shopkeeper, miner, farmer, or cook.
- Assignments can be absent for simple/static NPCs.
- The data model can support future schedules without forcing immediate AI complexity.

### REQ-MP-001: Server-Authoritative Multiplayer Behavior

Status: `Planned`

NPCs must fit into Sunny Town's multiplayer authority model.

Acceptance criteria:

- Sunny Town owns live NPC presence and accepted live positions.
- Clients receive NPC updates but do not authoritatively decide NPC state.
- Player/NPC interactions are validated server-side.
- Durable NPC-related state changes use documented HQ service-authenticated endpoints when HQ owns the data.

### REQ-DOC-001: Architecture Documentation

Status: `Planned`

Before major implementation, the character/NPC model must be documented in current-state architecture docs.

Acceptance criteria:

- Shared character state, player-specific state, and NPC-specific state are explicitly separated.
- HQ and Sunny Town ownership boundaries are documented.
- Migration impact on existing student inventory/equipment is documented.
- The initial vertical slice is documented before code changes begin.

## Phased Plan

### Phase 1: Investigation And Architecture

- Inspect existing player, inventory, equipment, shopkeeper, and Sunny Town presence code.
- Identify current tables and structs that are student-specific but should eventually become character-capable.
- Decide whether `Character`, `TownCharacter`, or another name best fits the codebase.
- Document persistence ownership and runtime ownership.
- Define the intended simulation scale for the first version.

Deliverable: current-state architecture proposal and implementation slice.

Status: `Mostly complete for current scope`

### Phase 2: Minimal Character Foundation

- Add or adapt shared character identity/state with minimal behavior change.
- Keep existing player behavior working.
- Provide a path for player characters and NPC characters to reference the shared model.
- Current implementation creates durable `sunny_town_character` player rows for students and passes `character_id` through Sunny Town sessions and join tokens.

Deliverable: shared character foundation with low user-visible risk.

Status: `Complete enough to start Phase 3A`

Current implementation details:

- Migration `0006_sunny_town_characters.sql` creates `sunny_town_character`.
- `internal/hq/characters.EnsurePlayer` creates/updates student player-character rows.
- `internal/hq/users.SyncAuthenticated` ensures student characters during auth sync.
- `cmd/hq.handleSunnyTownSession` ensures a player character before signing the join token.
- `internal/sunnytownauth.Claims` includes `character_id`.
- `internal/sunnytown/server` uses `character_id` for realtime player ID.
- Frontend Sunny Town session/player types accept `characterId`.

### Phase 3: NPC Records And Static NPC Presence

- Represent one or more existing shopkeepers as NPC-backed characters.
- Give NPCs stable IDs and appearance/display data.
- Surface NPCs in Sunny Town runtime without complex autonomous behavior.

Deliverable: NPCs exist as real town characters.

Status: `Phase 3A implemented`

Phase 3A should focus on static NPC identity only:

- Start with `mayor-sunny` in `sunny-town/maps/sunny-town-v1.json`.
- Keep current map-defined position, facing, dialogue, shop, and activity fields as display/runtime defaults.
- Add durable NPC character identity in HQ.
- Expose NPC character identity to Sunny Town.
- Preserve current frontend interaction behavior.

Do not implement movement, motives, schedules, homes, workplaces, inventory migration, or shop stock ownership in Phase 3A.

Current Phase 3A implementation details:

- Migration `0007_sunny_town_npc_characters.sql` creates `sunny_town_npc_character` with unique `(room_id, npc_key)` mappings and seeds `mayor-sunny` for `sunny-town-main`.
- `internal/hq/characters.EnsureNPC` and `EnsureNPCs` create or refresh NPC rows in `sunny_town_character` and connect them to stable NPC keys.
- HQ exposes `/api/internal/sunny-town/npc-characters/ensure` as a service-authenticated internal Sunny Town endpoint.
- Sunny Town calls the endpoint during startup and before WebSocket joins through `internal/sunnytown/hqclient` and merges returned `characterId` values into `internal/sunnytown/maps.NPC`.
- The frontend `SunnyTownNpc` type accepts optional `characterId`, while existing interaction code continues to use map NPC data.
- NPCs are still static in this slice; no autonomous movement or durable position migration was added.

### Phase 4: Shared Inventory And Equipment

- Generalize inventory/equipment ownership to character-capable ownership.
- Preserve current student inventory APIs and frontend behavior.
- Allow NPCs to own and equip items through the same catalog/rules.

Deliverable: players and NPCs can use common item/equipment systems.

### Phase 5: Shared Stats And Skills

- Add character-level health/stats/skills/XP primitives.
- Start with a small concrete use case, such as mining skill or health.
- Validate live stat effects in Sunny Town.

Deliverable: future gameplay mechanics attach to shared character state.

### Phase 6: Motives, Routines, And Economy Loops

- Add NPC motives and simple routines.
- Add home/workplace behavior.
- Connect NPC inventory to gathering, production, consumption, and shops.

Deliverable: Sunny Town begins behaving like a small simulated town.

## Open Questions

- Should shared character persistence live primarily in HQ, with Sunny Town maintaining live projections?
- Should NPC durable state be global, per town, per map, or per classroom/cohort?
- How should existing `student_inventory_item` and `student_equipped_item` migrate toward character-owned inventory?
- Do player characters need separate durable character rows, or can they be introduced through a compatibility layer first?
- How much NPC simulation should continue when no players are connected?
- What is the first vertical slice: shopkeeper inventory, static NPC identity, or a simple worker NPC?
- Should NPCs be visible to all players in the same shared town, or scoped by class/session in some cases?

## Proposed Answers

### Persistence Ownership

Proposed answer: shared durable character persistence should live in HQ. Sunny Town should maintain live projections while rooms are running.

Rationale:

- Current architecture already says HQ owns durable account, economy, inventory, equipment, map-object, and saved player position state.
- Sunny Town owns live realtime state and validates gameplay before durable writes.
- NPCs need durable identity, home/work assignments, inventory, stats, and long-term progression, which match HQ's durable ownership role.
- Sunny Town can load NPC character data from HQ at startup or room activation, simulate live behavior in memory, then commit important durable changes back through service-authenticated HQ endpoints.

Do not persist every live movement step. Persist durable milestones and state that should survive restart: identity, current home/work/map anchor, important inventory deltas, skills/XP, health, and maybe last known position at controlled intervals or shutdown.

### NPC Scope

Proposed answer: NPCs should be scoped to a Sunny Town `room_id` first, with map-specific current presence inside that room.

Rationale:

- The current runtime already uses `room_id` and `map_id`.
- Existing durable map objects are keyed by `room_id` and `map_id`.
- Scoping by room lets the project later support class/cohort/session-specific towns without redesigning NPC ownership.
- Map ID should describe where the NPC currently is, not who owns the NPC.

For the first implementation, use the existing default shared room unless product requirements demand classroom-specific towns.

### Inventory And Equipment Migration

Proposed answer: introduce character-owned inventory as the destination, but use a compatibility path rather than rewriting student inventory in the first slice.

Recommended path:

1. Add a durable `sunny_town_character` or `town_character` table with character type: `player` or `npc`.
2. Link player characters to `app_user_id`.
3. Keep current `student_inventory_item` and `student_equipped_item` APIs working.
4. Add character-owned tables only when NPC inventory needs to persist, or add a compatibility view/service layer that can read/write either student-owned or character-owned inventory.
5. Migrate player inventory behind stable student-facing APIs once the character model has proven useful.

The item catalog should remain shared. Avoid a separate NPC item catalog.

### Player Character Rows

Proposed answer: yes, player characters should eventually have durable character rows, but the first pass can create them lazily and keep student APIs stable.

Rationale:

- Future shared stats, skills, health, XP, and position should attach to `character_id`.
- Login/session/account concerns should stay on `app_user`.
- Lazy creation avoids a risky all-at-once migration.

Player character rows can start as a thin link:

```text
character_id -> app_user_id
character_type = player
display_name/avatar/default appearance
```

NPC rows use the same base identity without an `app_user_id`.

### Offline Simulation

Proposed answer: keep offline NPC simulation minimal at first.

Recommended behavior:

- When players are connected, Sunny Town runs live NPC simulation and broadcasts visible changes.
- When no players are connected, either pause simulation or advance coarse needs on next load using elapsed time.
- Persist only coarse timestamps and durable resources needed to catch up safely.

This keeps the town alive without introducing surprising offline outcomes, runaway resource generation, or expensive background work.

### First Vertical Slice

Proposed answer: start with static NPC identity as real character records, then add one lightweight routine. Do not start with full shop inventory or a worker economy loop.

Recommended first slice:

1. Represent existing map NPCs as live NPC character projections in Sunny Town snapshots.
2. Add stable durable NPC records in HQ or a documented seed/migration path.
3. Keep dialogue/shop/schoolwork interactions working.
4. Let one NPC perform a simple visible routine, such as walking between a home marker and town square or changing activity text by time/state.

This delivers the first "the town is alive" feeling while creating the technical foundation for shared stats and inventory.

### Visibility Scope

Proposed answer: NPCs should be visible to all players in the same room/town by default.

Rationale:

- Sunny Town is currently a shared realtime room.
- Shared NPCs make multiplayer feel coherent: if an NPC walks to the shop, everyone in that town should see it.
- Per-player private NPC state can be added later only for explicitly personal interactions.

Default rule: public town NPC state is shared; private dialogue/progress can remain player-specific when needed.

### Initial Population Scale

Proposed answer: design for a small colony scale first: roughly 3 to 12 active NPCs per town.

Rationale:

- This better matches the current Sunny Town map and renderer.
- It is enough for recognizable social/work/eat/rest loops.
- It reduces pathing, persistence, and multiplayer synchronization risk.
- It encourages giving each NPC personality and routine instead of simulating a crowd.

The architecture should not hard-code a tiny limit, but the first performance and UX target should be small.

## Investigation Notes

- Current map NPCs are loaded from map JSON through `internal/sunnytown/maps.GameMap.NPCs`. They currently include ID, name, position, facing, sprite key, dialogue, optional shop, and optional activity.
- Students now have durable Sunny Town character identity through `sunny_town_character` and `internal/hq/characters.EnsurePlayer`.
- Sunny Town join tokens now carry both `app_user_id` and `character_id`; realtime player IDs are based on character ID.
- Map NPC validation currently requires non-empty ID, name, and dialogue. Shop inventory is static map data, not NPC-owned inventory.
- Sunny Town rooms already run a server-side simulation ticker in `internal/sunnytown/server/room.run`.
- Room snapshots currently broadcast players, collectibles, resource nodes, placed objects, and world objects. NPCs are currently part of the static map payload, not dynamic snapshot state.
- Player live state is in `internal/sunnytown/server.player` and includes app user ID, display name, avatar, equipment, inventory snapshot, position, facing, movement, and client.
- Player movement is server-accepted in Sunny Town. Portals and collision are handled in the room/world layer.
- HQ internal Sunny Town endpoints currently expose student equipment, inventory quantity, player position, rewards, resource events, and map objects.
- Durable inventory/equipment tables are currently student-owned: `student_inventory_item`, `student_inventory_ledger`, `student_equipped_item`, and `student_hotbar_slot`.
- Durable Sunny Town position is currently student-owned through `student_sunny_town_position`.
- Durable map objects already use `room_id` and `map_id`, which is a useful precedent for NPC scoping.
- Phase 3A keeps NPCs in the static map payload rather than adding dynamic NPC snapshots. This preserves current frontend behavior while making map NPCs addressable by durable `character_id`.
- The NPC mapping currently uses `room_id + npc_key`, where `npc_key` is the map NPC `id` such as `mayor-sunny`. If future maps need two distinct NPCs with the same map ID in the same room, the keying rule should be revisited before adding those maps.
- Sunny Town tolerates startup NPC identity load failure by logging the error and continuing with map-defined NPCs, then retries before WebSocket joins so compose startup order does not permanently leave map NPCs without durable IDs.
