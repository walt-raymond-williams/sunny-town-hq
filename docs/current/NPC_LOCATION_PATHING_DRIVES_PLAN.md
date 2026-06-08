# NPC Location, Pathing, And Drives Plan

This document tracks Sunny Town location tags, NPC pathing, and drive-based movement. It builds on `docs/current/NPC_CHARACTER_MODEL_PLAN.md`. The first implementation sequence is preserved in `docs/current/NPC_MOVEMENT_IMPLEMENTATION_PLAN.md`.

Status: `Initial movement/drives implementation complete; tuning and richer behavior remain`

## Fresh-Agent Handoff

Read this section first after context compaction.

The first NPC movement implementation is complete and committed. `docs/current/NPC_MOVEMENT_IMPLEMENTATION_PLAN.md` is now historical/current-state detail for the completed vertical slice, not the next-work tracker.

Current implemented baseline:

- Durable NPC identity exists for static map NPCs through `sunny_town_character` and `sunny_town_npc_character`.
- Maps support optional `locations` with IDs, coordinates, radius, tags, owner hints, and capacity.
- Checked-in maps include owned home/rest, kitchen/food, merchant work, school work, and public/social/idle locations.
- Sunny Town builds a server-side portal-aware navigation graph from loaded maps.
- Same-map route segments use A* over a coarse map grid with static blocked rectangles.
- NPC route planning can include active collision `worldObjects` from the NPC's current room as dynamic blocked rectangles.
- Live NPC runtime state is initialized per room and broadcast through server-authoritative NPC snapshots.
- NPCs have in-memory `hunger`, `energy`, `social`, and `work` drives.
- Drives deplete over time and replenish at matching tagged locations.
- NPCs choose the lowest below-threshold satisfiable drive, skip unrouteable drives, and route to matching locations.
- NPC goals have focus windows, periodic reevaluation, emergency interruption, arrival grace, failure counts, and failed-target cooldowns.
- NPCs can choose a low-priority `idle` fallback route to public/idle/wander/social locations when no urgent drive goal is available.
- NPCs resolve runtime-only routine anchors for home/rest, work, food, and social/public targets from authored locations.
- NPCs use deterministic UTC schedule phases (`morning`, `day`, `evening`, `night`) to apply selection-time drive pressure for strong routine anchors.
- Schedule pressure is visible in `/debug/npcs`, and urgent raw needs still override scheduled behavior.
- Rooms that have become empty pause exact NPC path-following and apply a bounded coarse drive catch-up when a player returns.
- ND-9 durability decision: do not persist raw NPC drive values, current map position, or movement-controller state yet; keep them Sunny Town runtime state until stable gameplay concepts require durability.
- NPCs at eligible work anchors can emit durable, idempotent HQ-owned job production events without persisting raw movement-controller state.
- HQ exposes service-authenticated aggregate NPC job progress over those ledger events for later gameplay consumption decisions.
- Cookie Keeper `shopkeeper_stock` production now increments durable HQ-owned Cookie Keeper shop cookie stock, and player purchases consume that stock.
- Cookie Keeper has a small durable starter stock seed, a logical output storage capacity of `64`, and the shop UI shows `current / 64` stock before purchase.
- `sunny-town-house-1` has an authored `Cookie Shop` area (`cookie-shop`) tagged as shop/workplace/storage owner; Cookie Keeper still uses `cookie-keeper-counter` as the owned work anchor.
- `sunny-town-house-1` has authored `cookie-shop-output-chest` and `cookie-shop-input-chest` fixtures. The output chest snapshots as a source `fixture`, kind `chest` world object tied to `cookie-shop`, `cookie-keeper-shop`, output storage, and `cookie`; the input chest snapshots with `storageRole: input`, `cookie-shop`, and `cookie-keeper-shop`, but no durable ingredient quantity yet.
- The frontend renders chest world objects from the normal `worldObjects` snapshot stream.
- Players can inspect the nearby output chest with `F`; the read-only panel loads existing `cookie-keeper-shop` stock/capacity from `GET /api/student/shop/stock`.
- Existing player crafting now uses the shared HQ recipe catalog/execution foundation in `internal/hq/inventory/crafting.go`; student crafting remains API-compatible.
- Cookie Shop storage planning now lives in `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`; it tracks logical output chest capacity, authored shop area, physical chest fixtures, shared recipe execution, and later input ingredients.
- Fresh continuation should start in `docs/current/COOKIE_SHOP_STORAGE_PLAN.md` under "Immediate Next Slice: HQ-Owned Shop Input Storage"; that section is the authoritative tracker for the next coding slice.
- NPCs can follow cross-map portal routes, move room membership, appear only in their current map snapshot, and avoid portal bounce.

Important current code touchpoints:

- Map loading and validation: `internal/sunnytown/maps/maps.go`
- Map JSON locations and portals: `sunny-town/maps/*.json`
- Navigation graph and A*: `internal/sunnytown/navigation/navigation.go`
- Sunny Town room tick/runtime: `internal/sunnytown/server/constants.go`, `internal/sunnytown/server/world_lifecycle.go`
- Live NPC state: `internal/sunnytown/server/world_types.go`
- NPC movement/drives/controller: `internal/sunnytown/server/npc_movement.go`
- NPC runtime initialization: `internal/sunnytown/server/npc_characters.go`
- NPC snapshots: `internal/sunnytown/server/world_snapshots.go`
- NPC drive debug endpoint: `internal/sunnytown/server/npc_debug.go`, `GET /debug/npcs` with `X-HQ-Service-Secret`
- NPC production loop: `internal/sunnytown/server/npc_production.go`, `internal/sunnytown/server/server_workers.go`
- HQ production ledger/progress API: `internal/hq/sunnytownbridge`, `deploy/postgres/migrations/0008_sunny_town_npc_job_production.sql`
- HQ shop stock persistence and purchase consumption: `internal/hq/inventory/shop.go`, `deploy/postgres/migrations/0009_shop_stock.sql`
- HQ shared recipe catalog/execution foundation and player crafting adapter: `internal/hq/inventory/crafting.go`
- Cookie Keeper starter stock seed: `deploy/postgres/migrations/0010_seed_cookie_keeper_shop_stock.sql`
- Cookie Shop storage continuation tracker: `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
- Cookie Shop storage fixtures: `sunny-town/maps/sunny-town-house-1.json` fixtures `cookie-shop-output-chest` and `cookie-shop-input-chest`
- Cookie Shop chest inspection UI: `frontend/src/composables/useSunnyTownChestInteractions.ts`, `frontend/src/features/sunny-town/SunnyTownChestPanel.vue`
- Movement tests: `internal/sunnytown/server/npc_movement_test.go`
- Frontend live NPC consumption: `frontend/src/features/sunny-town/SunnyTownPage.vue`, `frontend/src/composables/useSunnyTownNpcInteractions.ts`, `frontend/src/features/sunny-town/rendering/characterDrawing.ts`

Verification for movement-drive work:

```powershell
go test ./...
cd frontend
npm run build
```

Known recurring frontend build warnings are not caused by this work:

- `studentAssignmentsApi.ts` is both dynamically and statically imported.
- Some built chunks are larger than 500 kB.

Commit hygiene for this repository:

- Check `git status --short` before staging and before final response.
- Stage files explicitly.
- Do not stage `hq-local.err.log`, `hq-local.out.log`, or generated `web/` assets.

## Next Work Tracker

Use this section to continue the movement/drives work. Implement one slice at a time, update this tracker, verify, and commit before starting the next slice.

### Slice ND-1: Plan Status Cleanup

Status: `Implemented`

Goal: make this broader plan the post-compaction tracker after the first movement implementation finished.

Implemented notes:

- This plan now records the implemented baseline from the completed movement implementation slices.
- Requirements and phases below distinguish completed work from remaining tuning/richer behavior.
- Next slices are tracked here instead of creating another document.

### Slice ND-2: Location Scoring And Deterministic Target Choice

Status: `Implemented`

Goal: choose the best matching destination when multiple locations can satisfy the same drive.

Implementation notes:

- Replace first-match location selection in `routeToDriveLocationLocked` with candidate collection and scoring.
- Keep current-map preference, but make it a score factor rather than a hard-coded accident of iteration order.
- Suggested initial score inputs:
  - drive urgency
  - route path length or estimated route cost
  - current map bonus
  - owner match bonus using `ownerNpcKey`
  - role/tag match bonus, such as merchant/shop/work
  - failed target cooldown exclusion, preserving existing behavior
- Keep selection deterministic with stable tie-breakers: score, map ID, location ID.
- Do not add durable assignments yet unless needed by tests.

Implemented notes:

- `routeToDriveLocationLocked` now collects all reachable matching locations before selecting a destination.
- Candidate scores combine drive urgency, route path cost, current-map preference, owned-location bonus, and simple role hints for shop/school-style NPCs.
- Failed target cooldown exclusion still happens before scoring.
- Ties are deterministic by score, map ID, then location ID.
- Tests cover closer matching destinations, owned rest destinations, deterministic tie-breaking, and skipping a failed high-score destination.

Acceptance criteria:

- If two locations match the same drive, the NPC picks the higher-scored one deterministically.
- A closer route can beat a farther route when no ownership/role bonus applies.
- An owned bed/home can beat a generic bed/home.
- Failed target cooldowns still prevent immediate retry.
- Current tests still pass.

Suggested tests:

- Two food locations choose the closer route.
- Owned home/bed beats an otherwise comparable unowned location for the owning NPC.
- Tie-breaker is deterministic by map/location ID.
- Failed target is skipped even if it would otherwise score highest.

### Slice ND-3: Debug/Inspectability For NPC Drives And Goals

Status: `Implemented`

Goal: make current NPC behavior visible enough to tune without attaching a debugger.

Implementation notes:

- Add a server-side debug representation for each live NPC's drives, active drive, goal map/location, route step, focus/reevaluation timestamps, and failure count.
- Prefer an internal/admin HTTP endpoint or structured log helper that does not change public gameplay snapshots unless there is an existing debug protocol pattern.
- Keep private/transient state out of normal client snapshots by default.
- Include enough data to answer: "why is this NPC going there?"

Implemented notes:

- Sunny Town exposes `GET /debug/npcs` for NPC movement/debug state.
- The endpoint requires `X-HQ-Service-Secret` when `SUNNY_TOWN_SERVICE_SECRET` is configured.
- Debug output is deterministic: maps, NPCs, and failed targets are sorted.
- Each NPC debug entry includes public identity/position, drives, active drive, goal location/tags, route step/path indexes, focus and reevaluation timestamps, arrival/failure state, and failed target retry times.
- Normal websocket `hello`, `snapshot`, and `map_changed` payloads remain unchanged.

Acceptance criteria:

- A developer can inspect each live NPC's current drive values and active goal.
- Failed target cooldown state is visible or loggable.
- Debug output is deterministic and testable.
- Normal player-facing snapshots remain stable unless a deliberate debug flag is added.

Suggested tests:

- Debug snapshot includes drive values and active goal.
- Idle NPC debug state is clear when no goal is active.
- Failed target state is represented without exposing unrelated private data.

### Slice ND-4: Idle/Wander/Public Fallback

Status: `Implemented`

Goal: avoid lifeless idle behavior when no urgent drive is satisfiable.

Implementation notes:

- Add a lightweight fallback behavior that routes idle NPCs to `idle`, `wander`, `public`, or `social` locations when no below-threshold drive can be satisfied.
- Keep fallback lower priority than urgent drives.
- Use a cooldown/focus window so fallback movement does not thrash.
- Do not build schedules or a full behavior tree.

Implemented notes:

- Added `idle` as a non-depleting pseudo-drive used only for fallback routing.
- Fallback targets use `idle`, `wander`, `public`, and `social` tags.
- NPCs choose fallback only after normal urgent drive selection finds no satisfiable goal.
- Idle fallback goals use the normal focus and reevaluation timestamps, but do not replenish, complete, or stale-fail like normal drives.
- Below-threshold drive goals can interrupt idle fallback after the focus and reevaluation window.
- Tests cover no-urgent fallback, fallback persistence after arrival, unavailable urgent-drive fallback, and drive interruption after reevaluation.

Acceptance criteria:

- If all urgent drives are unavailable or no drive is urgent, an NPC can pick a public/idle fallback.
- An urgent satisfiable drive interrupts fallback after the normal focus/reevaluation rules.
- Fallback targets do not immediately repeat forever if multiple options exist.

Suggested tests:

- NPC with no urgent drives chooses an idle/public location.
- Unavailable urgent drive falls back to idle/public when no other urgent drive is satisfiable.
- New emergency drive can interrupt fallback after reevaluation.

### Slice ND-5: Dynamic Collision Awareness For Pathing

Status: `Implemented`

Goal: account for blocking world objects when planning NPC routes.

Implementation notes:

- Current A* pathing uses static map `blockedRects`.
- Extend route planning or grid construction to include active collision `worldObjects` where practical.
- Keep dynamic replanning bounded; do not replan every tick for every NPC.
- Start with placed/natural collision objects that are already represented server-side.

Implemented notes:

- Navigation now supports per-plan extra blocked rectangles through `PlanRouteToLocationWithBlockedRects`, `PlanRouteWithBlockedRects`, and `PlanPathWithBlockedRects`.
- Sunny Town NPC goal planning passes active collision `worldObjects` from the NPC's current room into route planning.
- Inactive or non-collision world objects are not included, so broken/removed objects can open routes on later planning attempts.
- Route-planning failures mark the target failed and use the existing failed-target cooldown, preventing repeated per-tick route attempts against the same blocked target.
- Tests cover extra blocked rect path detours/failures, active dynamic blockers preventing NPC routes, cooldown suppression of repeated blocked retries, and inactive blockers allowing later routes.
- Scope note: this slice handles current-room dynamic blockers. Cross-map route steps still only include dynamic blockers from the room where the NPC is currently planning.

Acceptance criteria:

- NPC route planning avoids active collision world objects.
- Removing/breaking an object can make a previously blocked route available on a later planning attempt.
- Pathing failures mark targets failed or defer retry without trapping the NPC.

Suggested tests:

- Active placed collision blocks a route.
- Inactive/broken object no longer blocks a route.
- Failed route does not cause per-tick expensive replanning.

### Slice ND-6: Richer Authored Locations And Ownership

Status: `Implemented`

Goal: give NPCs more meaningful destinations for home/work/food/social routines.

Implementation notes:

- Add or refine checked-in map locations for homes, beds, kitchens, shops, school/work places, and public gathering spots.
- Use `ownerNpcKey` for first-pass owned home/bed behavior.
- Add role-oriented tags such as `merchant`, `school`, `farm`, or `shop` where they match existing NPCs.
- Keep authored map changes small and validated.

Implemented notes:

- `sunny-town-house-1` now has `mayor-sunny-bed`, an owned `home`/`bed`/`sleep` rest target for `mayor-sunny`.
- `sunny-town-house-1` keeps `shared-house-kitchen` as a `food`/`kitchen`/`meal` target.
- `sunny-town-house-1` now has `cookie-keeper-counter`, an owned `work`/`shop`/`merchant` target for `cookie-keeper`.
- `sunny-town-classroom` now has `teacher-desk-work`, an owned `work`/`school` target for `teacher`.
- `sunny-town-classroom` now has `classroom-study-circle`, a `public`/`social`/`idle` school fallback location.
- `sunny-town-v1` doorway blockers were trimmed slightly so the house and classroom portal centers are reachable by server-side A*.
- Tests cover checked-in map location metadata, Mayor Sunny selecting the authored owned bed, and the teacher selecting the authored school work location.

Acceptance criteria:

- At least one NPC has a reachable owned rest/home target.
- At least one NPC has a reachable work target.
- Existing maps validate and load.
- New tags improve scoring/fallback behavior without requiring durable assignments.

Suggested tests:

- Map validation covers new locations.
- Owned location is selected by scoring for the matching NPC.
- Work-tagged location can satisfy the work drive.

### Later Work: Durability, Coarse Catch-Up, And Production

Status: `Planned`

The first movement/drives implementation, tuning slices, runtime routine anchors, and simple schedule pressure are complete. The next work should decide what routine state must persist and how far NPC simulation should advance when no players are connected.

Recommended next order:

1. ND-10: Resource/job production loops, expressed as durable gameplay outcomes rather than raw movement-controller persistence.
2. ND-11: Social relationship effects.
3. Later durable routine state only when there are stable gameplay concepts to store, such as job/home assignments, relationships, or production outputs.

Rationale:

- Runtime anchors and simple schedule pressure clarify the boundary between runtime controller state and durable gameplay state.
- Bounded no-player catch-up is in place, so production loops can be designed without relying on full per-tick simulation while nobody is connected.
- Resource/job production and social systems should wait until the basic routine loop is readable and debuggable.

### Slice ND-7: Runtime Routine Anchors

Status: `Implemented`

Goal: make home/work/social/food anchors explicit runtime state derived from authored locations, without adding HQ persistence yet.

Implementation notes:

- Build a per-NPC runtime anchor summary during room/world initialization or first NPC behavior configuration.
- Start with authored map locations:
  - home/rest anchor: owned `home`, `bed`, `sleep`, or `rest` location.
  - work anchor: owned or role-matching `work`, `school`, `shop`, `merchant`, `farm`, or similar location.
  - food anchor: nearest or scored `food`, `kitchen`, or `meal` location.
  - public/social anchor: nearest or scored `public`, `social`, `idle`, or `gathering` location.
- Prefer `ownerNpcKey` matches over role matches, and role matches over generic tags.
- Keep anchors runtime-only for this slice; do not add HQ schema or durable assignment tables yet.
- Use anchors as scoring bonuses or direct preferred candidates, rather than replacing generic drive fallback entirely.
- Extend `GET /debug/npcs` so each NPC shows its resolved anchors and whether the current goal came from an anchor-preferred target.

Implemented notes:

- Added runtime-only `npcRoutineAnchors` on live NPC state.
- Anchors are resolved after all rooms/maps are initialized, so cross-map authored locations can be used.
- Home, work, food, and social anchors are derived from authored map locations.
- Resolution prefers `ownerNpcKey`, then role matches for work, then generic tagged locations with deterministic tie-breakers.
- Anchor matches add a strong scoring bonus but do not replace generic candidate routing.
- If an anchor target is missing or unreachable, route selection can still fall back to another scored matching location.
- Goals selected from an anchor carry `anchorKind`, and `/debug/npcs` now includes each NPC's resolved anchors plus the active goal's anchor kind.
- Tests cover checked-in map anchors, owner preference, role work anchors, anchor-preferred goal marking, blocked-anchor fallback, and debug anchor output.

Acceptance criteria:

- Mayor Sunny resolves an owned home/rest anchor from `mayor-sunny-bed`.
- Teacher resolves a work anchor from `teacher-desk-work`.
- Cookie Keeper resolves a merchant/shop work anchor from `cookie-keeper-counter`.
- If an anchor target is missing or unreachable, the NPC can still use generic scored locations.
- Debug output includes resolved anchors in a deterministic form.
- Current movement, scoring, fallback, cross-map, and dynamic blocker tests still pass.

Suggested tests:

- Runtime anchor resolution prefers `ownerNpcKey` over generic matching locations.
- Role anchor resolution finds teacher school work and shopkeeper merchant work.
- Drive selection prefers an anchor target when routeable.
- Missing/unrouteable anchor falls back to generic scored target.
- NPC debug snapshot includes anchors.

### Slice ND-8: Simple Schedule And Time-Band Drive Pressure

Status: `Implemented`

Goal: make routines visibly change over time without building a full calendar or GOAP planner.

Implementation notes:

- Add a small deterministic Sunny Town clock or phase helper, such as `morning`, `day`, `evening`, and `night`.
- Start runtime-only; do not persist schedule state yet.
- Use time bands to bias drive depletion or selection:
  - day: work becomes more important for NPCs with work anchors.
  - evening: social/public becomes more likely.
  - night: energy/home/rest becomes more important.
- Keep urgent biological drives, especially hunger/energy, able to override schedules.
- Surface current phase and schedule pressure in `GET /debug/npcs` or adjacent debug output.
- Avoid client-facing protocol changes unless there is a clear UI/debug need.

Implemented notes:

- Added a deterministic runtime schedule phase helper with UTC-based bands:
  - `morning`: 06:00-09:59
  - `day`: 10:00-16:59
  - `evening`: 17:00-20:59
  - `night`: 21:00-05:59
- Schedule pressure is applied only during drive selection; raw drive values still deplete/replenish normally.
- Day applies work pressure for NPCs with a strong work anchor.
- Night applies energy/home pressure for NPCs with a strong home/rest anchor.
- Evening applies social pressure for NPCs with a strong social anchor.
- Morning applies lighter hunger pressure for NPCs with a strong food anchor.
- "Strong" schedule anchors are owner- or role-derived anchors, not generic fallback anchors, so generic public/idle locations do not make unrelated tests or NPCs time-sensitive.
- Emergency raw drive values still override schedule pressure. For example, urgent hunger can beat a daytime work schedule.
- `GET /debug/npcs` now includes the current schedule phase and active schedule pressure entries with raw and selection-adjusted drive values.
- Tests cover phase boundaries, daytime work pressure, nighttime home/rest pressure, urgent hunger override, and debug schedule output.

Acceptance criteria:

- NPC with a work anchor tends toward work during the day when no stronger urgent need exists.
- NPC with a home/rest anchor tends toward home/rest at night.
- Hunger or another urgent drive can interrupt scheduled behavior using existing focus/reevaluation rules.
- Schedule behavior is deterministic in tests by injecting or passing explicit time.
- Debug output explains the current schedule phase or pressure.

Suggested tests:

- Day phase chooses or biases work for Teacher/Cookie Keeper.
- Night phase chooses or biases Mayor Sunny toward owned rest/home.
- Emergency hunger interrupts a scheduled work/rest goal after reevaluation.
- Debug output includes current phase/schedule state.

### Slice ND-9: Persistence And No-Player Coarse Catch-Up

Status: `Implemented`

Goal: decide which NPC routine state should survive process restarts and how much simulation should advance when no players are connected.

Implementation notes:

- Do this after ND-7 and ND-8 clarify what runtime state is worth saving.
- Decision for current architecture:
  - Do not persist raw drive snapshots yet.
  - Do not persist current NPC map/position/facing yet.
  - Do not persist exact movement-controller state, including current goal, route path, path index, focus windows, reevaluation windows, arrival grace, or failed-target cooldowns.
  - Continue deriving current anchors from authored map locations at runtime.
  - Add HQ-owned persistence only when the state represents a stable gameplay concept rather than a controller implementation detail.
- Future durable candidates, once the related gameplay exists:
  - job/work/home assignments if players or systems can change them.
  - social relationship state and relationship events.
  - production outputs, resource reservations, inventory/economy effects, and other authoritative results.
  - schedule templates or durable routine preferences if they become authored/gameplay data rather than inferred runtime bias.
- Keep transient controller state, such as exact route path, focus windows, and path indexes, in memory unless a strong reason appears.
- Coarse catch-up should be bounded and approximate, not full per-tick simulation while nobody is connected.

Implemented notes:

- Room-level exact NPC path-following now pauses after the last player leaves a map.
- On player re-entry or transfer into a previously empty map, Sunny Town applies bounded coarse catch-up to live NPC drives before sending the player snapshot.
- Catch-up is capped by `npcNoPlayerCatchUpMax` and currently updates drive depletion/replenishment at the NPC's current location only.
- Catch-up clears transient route/goal/focus state so normal goal selection can resume from fresh drive values.
- Initial test worlds and rooms that have never had players still step NPCs normally, preserving deterministic unit-test ergonomics.
- No HQ persistence was added in this sub-slice.
- Architecture decision: raw drives, positions, anchors, and controller state stay runtime-only for now because they are still movement implementation details.
- Future persistence should attach to durable NPC characters through HQ-owned APIs and should store assignments, relationships, production outcomes, or other stable gameplay records.
- This avoids cementing the current drive tuning model into schema that would likely need replacement once jobs, housing, production, and relationships are real systems.

Acceptance criteria:

- A documented decision exists for which NPC drive/routine state is durable versus transient. (`Implemented`)
- If durable state is added, it belongs to HQ-owned storage and Sunny Town writes through service-authenticated internal APIs. (`Implemented as architecture decision; no new persistence needed now`)
- No-player catch-up is bounded and does not run full path simulation while nobody is connected. (`Implemented`)
- Exact path/controller state is not persisted unless there is a clear reason. (`Implemented for current runtime behavior`)

Suggested tests:

- Coarse catch-up advances drive values by a capped amount after an offline interval. (`Implemented`)
- Catch-up does not attempt exact portal/path movement for long offline intervals. (`Implemented`)
- Catch-up can replenish a drive when the NPC is already standing at a matching location. (`Implemented`)
- Durable save/load is intentionally deferred until durable gameplay fields exist; do not add a raw drive snapshot table in the current architecture.

### Slice ND-10: Resource And Job Production Loops

Status: `Partially implemented`

Goal: let NPC routines produce durable gameplay outcomes without persisting raw movement-controller internals.

Architecture direction:

- HQ owns durable economy, inventory, assignment, and production-result state.
- Sunny Town owns live simulation, pathing, NPC controller state, and server-side validation of what happened in the realtime world.
- Do not persist NPC drive values, current route, current goal, path index, focus windows, or exact live position as part of production.
- Production should be expressed as durable gameplay facts or events, such as "NPC worked at assigned station and produced X" or "job progress advanced by Y", not as "NPC work drive was 37.2".
- Use service-authenticated internal HQ APIs with `X-HQ-Service-Secret` for any durable production writes.

Recommended first sub-slice:

- Identify one small job loop tied to existing authored work anchors, such as Cookie Keeper at a shop/work counter or Teacher at the classroom work location.
- Add runtime production eligibility in Sunny Town: NPC is at a matching work anchor/location, has a work goal or work-compatible routine state, and enough time has elapsed.
- Emit a bounded production/progress event to HQ only if there is already a clear HQ domain owner for the resulting state. If not, first add the HQ domain/API for that stable gameplay outcome.
- Keep no-player behavior coarse: production can use bounded elapsed time, but should not require exact per-tick path simulation while nobody is connected.

Acceptance criteria:

- A first NPC job loop has a clear durable gameplay output or progress record.
- Durable writes go through HQ-owned service-authenticated APIs.
- The implementation does not persist raw movement-controller state.
- The loop remains bounded during no-player catch-up/offline intervals.
- Debug output or logs make it possible to understand why production did or did not occur.

Suggested tests:

- NPC at a valid work anchor can produce or advance job progress after enough elapsed time. (`Implemented`)
- NPC away from a valid work location does not produce. (`Implemented`)
- No-player/coarse elapsed time is capped or otherwise bounded for production. (`Implemented`)
- HQ internal API/service client tests cover durable production writes, if a new endpoint is added. (`Implemented for HQ store/API/client path`)
- HQ aggregate read tests cover durable production progress grouped by NPC/workplace/output. (`Implemented`)

Implemented notes:

- Added HQ-owned `sunny_town_npc_job_production_ledger` in migration `0008_sunny_town_npc_job_production.sql`.
- Added service-authenticated `POST /api/internal/sunny-town/npc-job-production`.
- Added service-authenticated `GET /api/internal/sunny-town/npc-job-production/progress` with aggregate progress grouped by room, map, NPC, job, location, output, and character.
- Added Sunny Town `hqclient.CommitNPCJobProduction` and an NPC job production worker.
- Added runtime production eligibility in Sunny Town:
  - NPC must have a supported job definition from existing map-authored role data (`shop` -> `shopkeeper_stock`, `schoolwork` activity -> `teacher_lesson_prep`).
  - NPC must have a resolved work anchor and physically be at that work anchor in the live room.
  - NPC must have durable character identity from map data or HQ `npc-characters/ensure`.
  - Enough eligible elapsed time must accumulate before one idempotent event is queued.
- Added bounded no-player catch-up production using the existing pause/catch-up path, without exact offline path simulation.
- Added `/debug/npcs` production fields: eligibility, job key, output key, progress seconds, last event, and last production time.
- Added HQ-owned shop stock tables in migration `0009_shop_stock.sql`.
- Cookie Keeper `shopkeeper_stock` events now atomically record the NPC production ledger and increment Cookie Keeper cookie stock through `internal/hq/inventory`.
- Player purchases from `cookie-keeper-shop` now require durable shop stock and consume it in the purchase transaction before granting the cookie.
- Duplicate NPC production events do not double-increment shop stock.
- Added `GET /api/student/shop/stock` so the shop UI can display and disable buys based on current durable stock.
- Added a small idempotent Cookie Keeper starter stock seed so fresh databases are not blocked before NPC production warms up.
- Ongoing stock replenishment remains NPC production-driven; the seed is only a cold-start floor.
- Decided that Cookie Shop output stock should be modeled as shop-owned storage, not NPC-held inventory.
- Decided that the current `shop_stock_item` table is the logical Cookie Shop output chest until a physical chest fixture is added.
- Added a Cookie Keeper cookie stock capacity of `64` in HQ-owned inventory rules.
- NPC-produced Cookie Keeper cookie stock now clamps at capacity while production and stock ledger events remain idempotent.
- `GET /api/student/shop/stock` now includes item capacity, and the shop UI displays `current / 64`.
- Added an authored `cookie-shop` map location named `Cookie Shop` in `sunny-town-house-1`, tagged `shop`, `workplace`, `cookie_shop`, and `storage_owner`.
- Kept `cookie-keeper-counter` as the owned work anchor so routine/pathing behavior remains stable while storage ownership becomes visible/inspectable.
- Added optional map `fixtures` and authored `cookie-shop-output-chest` as a physical output chest fixture inside/associated with `cookie-shop`.
- Sunny Town initializes fixtures into runtime `worldObjects`; the chest snapshots with `shopId: cookie-keeper-shop`, `storageRole: output`, `itemKey: cookie`, and `locationId: cookie-shop`.
- Frontend world-object rendering draws chest fixtures.
- Chest snapshots now expose authored `interactionRadius`.
- Added read-only chest inspection: pressing `F` near `cookie-shop-output-chest` opens a storage panel backed by existing `GET /api/student/shop/stock` stock/capacity.
- Added authored `cookie-shop-input-chest` fixture as the physical anchor for future Cookie Shop ingredient storage.
- Map validation requires input storage fixtures to identify their owning shop and location.
- Revised the next production architecture decision: recipes should be shared HQ-owned definitions used by player crafting, NPC/shop production, and future workstations. Do not implement a Cookie Keeper-only recipe path.
- Implemented the shared recipe execution foundation in HQ inventory: recipe definitions are actor-agnostic, student crafting adapts the existing student inventory functions, and missing ingredients stop output production before mutation.
- Deferred input ingredients: Cookie Keeper can make cookies while working in the shop for now.
- This slice intentionally does not add NPC-held inventory, assignments, or raw drive/position persistence.

Next ND-10 sub-slice:

- Implement HQ-owned Cookie Shop input storage before recipe/input consumption:
  - Define durable input storage under HQ ownership, attached to `cookie-keeper-shop`/`cookie-shop-input-chest` identity.
  - Add load and mutate operations in `internal/hq/inventory` that can later satisfy the shared recipe executor's consume side.
  - Do not add NPC-held inventory or store ingredient quantities in Sunny Town fixture/client state.
  - Do not wire Cookie Keeper production directly to a one-off cookie recipe rule.
  - Verify with `go test ./...`; frontend build is only needed if API response shapes change.
- After durable input storage exists, implement recipe/input consumption:
  - Add a Cookie Shop workstation fixture, likely a stove/oven, as the future player/NPC recipe interaction point.
  - Define cookie recipe requirements in the shared recipe catalog.
  - Change production commit handling so HQ atomically consumes inputs and increments output stock, or records/returns a missing-input/blocked result without mutating output.
- Defer explicit chest actions such as withdraw/deposit until the shop sale path and ownership transfer rules are designed.
- Alternate branch: make teacher lesson prep consume or expose durable progress if broader NPC job gameplay is the priority.
- Do not generalize inventory ownership to character-capable inventory unless the next gameplay requirement clearly needs NPC-held items.
- Do not create a second inventory/stock source for the output chest; keep durable quantities HQ-owned.

## Feature Intent

Sunny Town NPCs should eventually move through the world for understandable reasons. The goal is not to make NPCs wander randomly. The goal is to give them simple needs and goals that cause visible, readable routines: go to work, go home, eat, rest, socialize, idle in public, and later gather or produce resources.

The intended feel is a small RimWorld-like colony. A small number of recognizable NPCs should create a living-town effect through visible routines and simple interacting systems. The first versions should favor clarity and reliability over deep simulation.

The first movement version should be intentionally simple: drives deplete on their own over time, and visiting an appropriate tagged location replenishes the matching drive quickly. Later systems can make drives more realistic by having work, time of day, food availability, relationships, injuries, or events affect them. That complexity is not part of the first movement slice.

## Design Summary

The design has four separable layers:

```text
World locations
  Named/tagged points or regions on maps.
  Examples: store counter, farm plot, bed, kitchen, town square.

Pathing
  Server-side route planning from current NPC position to target location.
  Use a map/portal graph for cross-map routes from the start.
  Use A* over a navigation grid for each in-map segment.

Drives
  Slow-changing internal needs that deplete over time and are replenished
  by visiting matching locations.
  Examples: hunger, energy, social, work/idle.

NPC controller
  Chooses the lowest/most urgent satisfiable drive, commits to it for a short
  focus window, moves the NPC, and replenishes the drive while the NPC is at the destination.
```

Keeping these layers separate should make the system easier to debug. Locations do not decide behavior. Drives do not know geometry. Pathing does not know why the NPC wants to move. The controller glues them together.

## Location Tags

Location tags are the vocabulary NPCs use to understand the world.

Maps should eventually declare meaningful locations such as:

```json
{
  "locations": [
    {
      "id": "town-square-center",
      "name": "Town Square",
      "x": 736,
      "y": 512,
      "radius": 32,
      "tags": ["public", "social", "idle"]
    },
    {
      "id": "market-counter",
      "name": "Market Counter",
      "x": 900,
      "y": 320,
      "radius": 24,
      "tags": ["shop", "work", "merchant"]
    },
    {
      "id": "farm-plot-1",
      "name": "Farm Plot",
      "x": 240,
      "y": 700,
      "radius": 32,
      "tags": ["farm", "work", "food_source"]
    },
    {
      "id": "mayor-bed",
      "name": "Mayor Sunny's Bed",
      "x": 320,
      "y": 240,
      "radius": 24,
      "tags": ["home", "bed", "sleep"],
      "ownerNpcKey": "mayor-sunny"
    }
  ]
}
```

Recommended fields:

- `id`: stable map-local location ID.
- `name`: human-readable editor/debug name.
- `mapId`: optional if locations live inside a map file; required if using a room-level location index.
- `x`, `y`: center point in map coordinates.
- `radius`: acceptable arrival/interact radius.
- `tags`: searchable behavior tags.
- `ownerNpcKey`: optional temporary owner link before NPC character IDs are map-authoring-friendly.
- `ownerCharacterId`: possible future durable owner link, but avoid requiring this in hand-authored map JSON too early.
- `capacity`: optional future limit for social spots, work stations, beds, etc.

Recommended tag families:

- Work: `work`, `shop`, `merchant`, `farm`, `mine`, `kitchen`, `school`.
- Needs: `food`, `food_source`, `meal`, `bed`, `sleep`, `rest`.
- Social: `public`, `social`, `gathering`, `town_square`.
- Ownership: `home`, `personal`, `shared`.
- Behavior hints: `idle`, `wander`, `wait`, `patrol`.

## Pathing Model

NPC movement should be server-authoritative in Sunny Town.

Cross-map pathing should be part of the navigation design from the start. Sunny Town is likely to gain many map cells connected by doors, paths, and portals. Even if the first moving NPC demo stays on one map, the pathing API and data model should assume that an NPC may need to travel through several maps to reach a tagged destination.

Recommended pathing layers:

1. World navigation graph
   - Treat each map as a node or a group of local navigation nodes.
   - Treat portals/doors/exits as directed edges between maps.
   - A target location may be on the current map or many portal hops away.
   - The route planner should produce a sequence of map segments and portal transitions.

2. In-map pathing
   - Use A* over a coarse navigation grid.
   - Start with map tile size or a simple fixed grid derived from map geometry.
   - Block tiles using map `blockedRects` and active collision world objects.
   - Treat target locations as arrival circles or nearest reachable grid points.
   - Use A* for each segment: current position to portal, portal arrival to next portal, final portal arrival to destination.

3. Portal transitions
   - Use existing map portal concepts rather than adding a parallel cell abstraction.
   - Preserve the server-side portal re-entry guard idea so an NPC does not bounce between portals.
   - Example: current position -> house door portal -> target map -> bed location.

4. Replanning
   - Replan when a path is blocked, a target disappears, or the NPC changes goal.
   - Keep replanning bounded. Avoid doing expensive pathfinding every tick.

Initial visible movement may stay on one map if that makes testing easier, but the route planner shape should still support cross-map paths. Otherwise the first pathing API will calcify around same-map assumptions and become painful once NPCs need homes, beds, shops, and farms spread across many cells.

## Basic Drive Model

Use a simple depletion/replenishment drive model.

Each NPC has internal drives represented as values from `0` to `100`.

- `100` means the drive is fully satisfied.
- `0` means the drive is empty and urgent.
- Drives slowly decrease on their own over time.
- When a drive drops below a threshold, the NPC should seek a tagged location that can replenish it.
- When the NPC is at a matching location, that drive replenishes quickly.

This is deliberately inverted from some "need pressure" models. The simple rule is: low drive value means "go fix this."

Example drives:

- `hunger`: decreases over time; replenished at food/kitchen/meal locations.
- `energy`: decreases over time; replenished at bed/rest/home locations.
- `social`: decreases over time; replenished at public/social/gathering locations.
- `work` or `purpose`: decreases over time; replenished at an assigned workplace or role-specific work location.
- `idle` or `comfort`: fallback drive for wandering/being in public when no urgent need exists.

Example decision loop:

```text
Every few seconds:
  decrease drives
  if current goal is still in its focus window and not failing:
    continue current goal
  otherwise choose lowest actionable drive
  find candidate locations matching that drive
  score candidates
  path toward best candidate
  wait at destination while replenishing the drive quickly
```

Drive-to-location examples:

```text
hunger  -> tags: food, meal, kitchen, food_source
energy  -> tags: bed, sleep, rest, home
work    -> assigned workplace first, then tags for role
social  -> tags: social, public, gathering
idle    -> tags: idle, wander, public
```

Suggested first tuning:

- Drives deplete slowly enough that an NPC has time to satisfy several needs.
- Replenishment should be much faster than depletion.
- Use a threshold such as `40` to trigger goal selection.
- Replenish to a comfortable value such as `85` or `100` before choosing another urgent drive.
- Add a goal focus window so NPCs do not flicker between destinations when two drives are close.
- Add a short reevaluation interval, like a Sims-style pause: while working/resting/eating/socializing, the NPC periodically checks whether another drive has become urgent enough to interrupt.

For the first implementation, all drives can deplete at fixed rates. Do not yet model hunger increasing because of work, social decreasing because of isolation, or fatigue changing based on time of day. Those are later refinements.

## Goal Commitment And Failure

NPCs should not constantly switch goals every tick. Once an NPC chooses a goal, it should usually commit to that goal for a minimum focus window.

Recommended concepts:

- `currentGoal`: the drive being satisfied, target location, route, and start time.
- `focusUntil`: earliest time the NPC should consider normal goal switching.
- `reevaluateAt`: periodic check time while the NPC is working/resting/eating/socializing.
- `failureCount`: number of failed attempts for the current drive or target.
- `blacklistedTargets`: temporary cooldown list for locations that could not be reached or did not replenish the drive.

Basic rules:

- Pick the lowest satisfiable drive when idle or when the current goal is complete.
- A drive is satisfiable only if there is a matching location and a route can be planned.
- If the most urgent drive cannot be satisfied, try the next most urgent drive.
- If the NPC reaches a matching location but the drive does not replenish after a grace period, mark that target as failed and try another location or another drive.
- If pathing fails, temporarily blacklist that target and try another candidate.
- If all urgent drives fail, fall back to idle/wander/public locations and retry later.
- While focused on a goal, continue it unless another drive crosses a stronger emergency threshold or the goal fails.

This gives NPCs a simple "I am doing a thing" behavior instead of making them twitch between hunger, sleep, work, and social every time values move by one point.

## Location Scoring

When multiple locations match a depleted drive, the controller should score them instead of picking randomly.

Useful scoring factors:

- Distance/path cost.
- Ownership match, such as the NPC's own bed over any bed.
- Role match, such as shopkeeper to merchant counter.
- Capacity, if the location is full.
- Recent-use cooldown to avoid jitter.
- Current map preference, if two options are otherwise similar.

First implementation can use a simple score:

```text
score = urgency - pathCost + ownershipBonus + roleBonus
```

## User Stories

### STORY-LOC-001: Tagged Town Locations

As a designer, I want maps to define tagged locations so that NPCs can understand where to work, rest, eat, socialize, or idle.

Acceptance criteria:

- A map can define named locations with stable IDs.
- Locations can have one or more tags.
- Locations include map coordinates and an arrival radius.
- Existing maps continue loading if they have no locations.

### STORY-PATH-001: NPCs Can Path To A Location

As a player, I want NPCs to walk to meaningful places so that the town feels alive.

Acceptance criteria:

- Sunny Town can compute a server-side route from an NPC to a target location.
- Pathing respects blocked map geometry.
- Route planning can include portal transitions between maps.
- NPC movement is broadcast to clients through server-authoritative state.
- Pathing work is bounded and does not run every frame for every NPC.

### STORY-PATH-002: NPCs Can Use Portals

As a designer, I want NPCs to move between maps through portals so that homes, shops, farms, and work locations can exist across the town.

Acceptance criteria:

- Locations include a map identity.
- Portal connections can be used as cross-map route edges.
- NPC portal movement uses the same portal concepts as players where practical.
- Cross-map routing is included in the pathing design from the start, even if the first movement test case is same-map.

### STORY-DRIVE-001: NPC Drives Choose Destinations

As a player, I want NPCs to move to places that satisfy their needs so that their behavior feels intentional.

Acceptance criteria:

- NPCs can track slow-changing drive values from `0` to `100`.
- Drive values decrease over time.
- Drives map to candidate location tags.
- The lowest satisfiable drive can produce a destination.
- Staying at a matching location quickly replenishes the related drive.

### STORY-DRIVE-002: NPC Behavior Is Understandable

As a designer, I want simple drive conflicts to be visible and inspectable so that we can tune NPC movement behavior.

Acceptance criteria:

- The server can identify the NPC's current drive/goal in logs or debug state.
- NPCs avoid rapid goal flipping.
- If multiple drives are depleted, the controller chooses one deterministically using urgency and location score.
- If the most urgent drive cannot be satisfied, the NPC can try the next most urgent drive.
- If a target fails to replenish a drive, the NPC can abandon that target after a grace period.
- Replenishment is fast enough that NPCs can recover several drives over time.
- Behavior remains deterministic enough for multiplayer debugging.

## Requirements

### REQ-LOC-001: Map Location Definitions

Status: `Implemented`

Sunny Town maps should support optional location definitions.

Implemented notes:

- `GameMap` supports optional `locations`.
- Location validation covers IDs, coordinates, radius, tags, owner hints, and capacity.
- Existing maps without locations remain valid.
- Checked-in maps include initial social/rest/home/food locations.

Acceptance criteria:

- Location IDs are unique within a map.
- Location tags are non-empty strings when present.
- Invalid locations fail map validation.
- Maps without locations remain valid.

### REQ-PATH-001: Server-Authoritative NPC Navigation

Status: `Partially implemented`

NPC pathing and movement must be owned by Sunny Town.

Implemented notes:

- Clients do not authoritatively move NPCs.
- Sunny Town owns live NPC position, route following, drive selection, and snapshots.
- Route planning supports static map blocked rectangles and portal edges between maps.
- NPCs can transfer between rooms through portal route steps.
- NPC route planning can include active current-room collision world objects as dynamic blockers.

Remaining notes:

- Cross-map route planning does not yet include dynamic blockers from destination/intermediate rooms during the initial plan.
- Replanning is still intentionally simple and bounded by goal selection/failure behavior.

Acceptance criteria:

- Clients do not authoritatively choose NPC paths.
- Sunny Town validates walkable positions.
- Pathing can account for map blocked rectangles.
- Route planning can include portal edges between maps.
- Pathing can later account for dynamic collision objects.

### REQ-DRIVE-001: Basic Drive State Belongs To NPC Simulation

Status: `Implemented for runtime state`

NPC drives should be simulation state layered on top of shared character state.

Implemented notes:

- Live NPCs have in-memory hunger, energy, social, and work drives.
- Drive values are not stored in map JSON and are not player-only state.
- Drives decrease over time and replenish near matching tagged locations.
- NPC controllers maintain focus windows, reevaluation intervals, and failed-target cooldowns.

Remaining notes:

- Durable drive snapshots, home/work assignments, and no-player coarse catch-up are future work.

Acceptance criteria:

- Drive values are not player-only state.
- Drive values are not stored in map JSON.
- Initial drive values can be simple runtime state with defaults.
- Drives decrease over time and replenish near matching tagged locations.
- NPC controllers maintain goal focus windows, reevaluation intervals, and failed-target cooldowns.
- Durable drive snapshots may live in HQ later, but transient decision state can stay in Sunny Town memory.
- The drive system can be paused or coarse-caught-up when no players are connected.

### REQ-MP-LOC-001: Multiplayer Consistency

Status: `Implemented for public movement snapshots`

All players in the same room should see the same public NPC movement and location-driven behavior.

Implemented notes:

- NPC position/state is room-shared and server broadcast through live NPC snapshots.
- NPC snapshots are scoped to the room/map where the live NPC currently resides.
- Existing private player dialogue/progress can remain player-specific.

Remaining notes:

- Debug/inspectability state for current drives/goals is not yet exposed.

Acceptance criteria:

- Public NPC position/state is room-shared.
- Private player dialogue/progress can remain player-specific.
- NPC movement snapshots are server broadcast.
- NPC interactions use server-validated position and capability state.

## Implementation Plan

### Phase A: Location Tags Only

Status: `Implemented`

- Add optional `locations` to map JSON types.
- Validate location IDs, coordinates, radius, and tags.
- Add tests for map loading and validation.
- Do not move NPCs yet.

Deliverable: maps can describe meaningful places.

### Phase B: Route Planning And Same-Map Pathing

Status: `Implemented`

- Build a map/portal route graph from loaded maps.
- Build a simple nav grid from map dimensions and blocked rectangles.
- Implement A* for same-map route segments.
- Define the route result shape so it can include portal transitions.
- Add pathing tests for blocked routes, reachable same-map routes, and at least one cross-map route plan.
- Keep dynamic objects out of the first pass if needed.

Deliverable: server can compute walkable same-map paths and represent cross-map routes.

### Phase C: Static Goal Movement

Status: `Implemented`

- Give one NPC a scripted goal to move between two tagged locations.
- Prefer a route that exercises portal-aware planning if practical; same-map movement is acceptable only as a smaller smoke test.
- Broadcast live NPC position through Sunny Town snapshots.
- Preserve current dialogue interactions.

Deliverable: first visible moving NPC.

### Phase D: Basic Drive-Based Goal Selection

Status: `Implemented`

- Add a small set of drive values to NPC runtime state, such as hunger, energy, social, and work/purpose.
- Decrease those values on a fixed interval.
- Map drives to location tags.
- Let one NPC choose the lowest satisfiable drive and move to a matching location.
- Replenish that drive quickly while the NPC remains near the destination.
- Add focus windows, reevaluation intervals, and failed-target cooldowns to prevent jitter.

Deliverable: first intentional-feeling NPC routine.

### Phase E: Richer Cross-Map Movement

Status: `Implemented for one-hop/multi-step route runtime; richer authored routines remain`

- Let NPCs actively use multi-hop cross-map routes for real needs like home, bed, shop, farm, or kitchen.
- Preserve portal re-entry guard ideas from player movement where relevant.
- Add tests for multiple portal hops once the map set contains enough connected cells.

Deliverable: NPCs can go home/work across map boundaries.

## Non-Goals For The First Movement Work

- Do not build a general GOAP planner.
- Do not implement combat, injury, or complex mood systems.
- Do not simulate large populations.
- Do not persist every path step or transient goal.
- Do not make clients authoritative over NPC movement.
- Do not migrate inventory/equipment as part of location/pathing work.
- Do not model realistic drive causes in the first pass. Fixed depletion and tagged-location replenishment are enough.
- Do not require a day-night cycle for the first drive implementation.

## Relationship To Current NPC Character Plan

This feature came after durable NPC character identity exists.

Completed order:

1. `NPC_CHARACTER_MODEL_PLAN.md` Phase 3A: make current static NPCs durable character-backed entities.
2. This plan Phase A: add map location tags.
3. This plan Phase B/C: add portal-aware route planning, pathing, and one simple moving NPC.
4. This plan Phase D: add basic drive-based destination choice with fixed depletion and fast replenishment.
5. This plan Phase E: add cross-map NPC movement through portals.

Current continuation order:

1. Continue Slice ND-10: expand production outputs/consumption from the first durable NPC job production ledger.
2. Slice ND-11: social relationship effects.
3. Later: durable routine assignments/preferences only when they are stable gameplay concepts, not raw movement-controller internals.

The key dependency is identity: movement, drives, jobs, and home/work assignments should attach to durable NPC characters, not anonymous map fixtures.
