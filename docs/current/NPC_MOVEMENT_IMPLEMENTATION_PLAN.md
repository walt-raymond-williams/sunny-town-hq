# NPC Movement Implementation Plan

This document is the handoff-ready implementation plan for adding Sunny Town NPC movement using location tags, portal-aware pathing, and a basic drive system.

Status: `Slice 6 implemented; Slice 7 is next`

Related docs:

- `docs/current/NPC_CHARACTER_MODEL_PLAN.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`

## Fresh-Agent Handoff

Read this first when starting the movement work.

Durable NPC identity is already implemented. Static map NPCs can be backed by `sunny_town_character` rows through `sunny_town_npc_character`. Sunny Town loads/ensures those NPC identities from HQ and merges `characterId` into map NPC definitions while preserving current map-defined dialogue, position, facing, shop, and activity behavior.

The movement work should now be implemented in slices. Do not jump straight to a large AI system. The first goal is a tiny but correct vertical path from "maps have tagged destinations" to "one NPC can move to a destination for a simple reason."

After each slice is implemented and verified, commit the slice before starting the next one. Keep slice commits scoped to the files changed for that slice and any tracking-document updates.

Important current code touchpoints:

- Map loading and validation: `internal/sunnytown/maps/maps.go`
- Map JSON files: `sunny-town/maps/*.json`
- Sunny Town runtime rooms/ticks: `internal/sunnytown/server/world_lifecycle.go`
- Sunny Town world state structs: `internal/sunnytown/server/world_types.go`
- Snapshot broadcast: `internal/sunnytown/server/world_snapshots.go`
- WebSocket protocol DTOs: `internal/sunnytown/protocol/protocol.go`
- Frontend Sunny Town types: `frontend/src/types/sunnyTown.ts`
- Frontend drawing and interaction: `frontend/src/features/sunny-town/rendering/characterDrawing.ts`, `frontend/src/features/sunny-town/SunnyTownPage.vue`, `frontend/src/composables/useSunnyTownNpcInteractions.ts`

Core constraints:

- Sunny Town owns live NPC movement and simulation.
- HQ owns durable character identity and future durable state.
- Clients must not authoritatively move NPCs.
- Existing NPC dialogue/shop/schoolwork interactions must keep working.
- Existing maps without `locations` must keep loading.
- Do not migrate inventory/equipment in this feature.
- Do not implement day-night, complex schedules, GOAP, combat, mood, or economy loops in the first movement slices.

Verification for every implementation slice:

```powershell
go test ./...
cd frontend
npm run build
```

Use Docker health checks when runtime smoke testing is practical:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

## Target Behavior

The first full feature should let a small number of NPCs move through Sunny Town for simple, readable reasons.

Initial behavior model:

- NPCs have simple drives from `0` to `100`.
- Drives deplete slowly on their own.
- A low drive value means the drive is urgent.
- Tagged locations replenish matching drives quickly.
- NPCs choose the lowest satisfiable drive, not just the lowest raw drive.
- If the most urgent drive cannot be routed to or replenished, the NPC tries the next urgent drive.
- NPCs commit to a goal for a short focus window and periodically reevaluate.
- Cross-map route planning through portals is designed from the start.

Example:

```text
Mayor Sunny's energy drive drops below the threshold.
The controller finds a location tagged bed/sleep/rest/home.
The route planner builds a route, possibly through a door/portal.
Mayor Sunny walks there.
While near the destination, energy replenishes quickly.
After a minimum focus window, he reevaluates other drives.
```

## Slice 1: Map Location Tags

Goal: maps can describe meaningful places without moving NPCs yet.

Status: `Implemented`

Implemented notes:

- `GameMap` now accepts optional per-map `locations`.
- Map loading validates location IDs, coordinates, radius, tags, and capacity.
- Checked-in maps include initial town square, rest/home/bed, and kitchen/food locations.
- Frontend map types include optional `locations`.

Implementation notes:

- Add optional `Locations []Location` to `internal/sunnytown/maps.GameMap`.
- Define `Location` with fields:
  - `ID string`
  - `Name string`
  - `X float64`
  - `Y float64`
  - `Radius float64`
  - `Tags []string`
  - optional `OwnerNPCKey string`
  - optional `Capacity int`
- If locations are stored inside each map file, `mapId` is implicit. Do not require `mapId` in per-map JSON yet.
- Validate:
  - IDs are non-empty and unique per map.
  - Coordinates are inside map bounds.
  - Radius is positive.
  - Tags are non-empty after trimming.
  - Maps with no locations remain valid.
- Add a few initial locations to map JSON for testing:
  - `town-square-center` tagged `public`, `social`, `idle`.
  - A rest/home/bed location in a house map.
  - A food/kitchen/meal location if a reasonable map spot exists.

Acceptance criteria:

- Existing maps load without `locations`.
- New `locations` data loads and validates.
- Invalid duplicate location IDs fail validation.
- Frontend types can accept `locations` on `SunnyTownMap`.
- No NPC movement behavior changes yet.

Suggested tests:

- `internal/sunnytown/maps` tests for valid locations.
- Tests for duplicate IDs, blank tags, invalid radius, and out-of-bounds coordinates.

## Slice 2: Portal-Aware Route Model And Nav Graph

Goal: define the route planner shape so cross-map travel is supported from the start.

Status: `Implemented`

Implemented notes:

- Added `internal/sunnytown/navigation` with a directed portal graph built from loaded maps.
- Added route DTOs for same-map destination steps and portal-transition steps.
- Route planning can target raw map points or authored map locations.
- Cross-map planning currently resolves map-to-map portal hops; walkable in-map A* segments are Slice 3.
- Tests cover same-map location routing, the existing Sunny Town to Forest Crossing portal, unknown target maps, unknown portal targets, and unconnected maps.

Implementation notes:

- Add a Sunny Town server-side route/navigation package or files under `internal/sunnytown/server`; a focused `internal/sunnytown/navigation` package is also reasonable if the boundary is cleaner.
- Build a world navigation graph from loaded maps:
  - maps are graph nodes
  - portals are directed edges
  - each edge knows source map, portal rectangle/center, target map, target coordinates, target facing
- Define route DTOs internally, for example:

```go
type Route struct {
	Steps []RouteStep
}

type RouteStep struct {
	MapID    string
	From     point
	To       point
	PortalID string // non-empty when this step ends in a portal transition
}
```

- The route shape should support:
  - same-map destination
  - one portal hop
  - multiple portal hops later
- Do not broadcast routes to clients unless needed for debugging.

Acceptance criteria:

- The server can identify whether a target location is same-map or cross-map.
- The route model can represent portal transitions.
- Portal routing uses existing map portal definitions.
- No NPC movement behavior changes yet.

Suggested tests:

- Route from `sunny-town-v1` to `forest-crossing-v1` uses the existing portal.
- Unknown target map fails clearly.
- Unconnected target map fails clearly.

## Slice 3: Same-Map A* Segment Pathing

Goal: compute walkable local path segments.

Status: `Implemented`

Implemented notes:

- Added static same-map A* path planning to `internal/sunnytown/navigation`.
- Built coarse navigation cells from each map's tile size and dimensions.
- Marked blocked cells from map `BlockedRects` using the server-compatible 28px agent footprint.
- `RouteStep` now carries path waypoints, and route planning fills same-map, portal approach, and final target-map segment paths.
- Tests cover straight paths, obstacle detours, unreachable targets, and pathing to a portal entry point.

Implementation notes:

- Build a coarse navigation grid from map dimensions and tile size.
- Mark blocked cells using `BlockedRects`.
- Start with static map collision only. Dynamic collision objects can come later.
- Implement A* for grid cells.
- Convert path cells back to map coordinates for NPC steering.
- Use the same route segment mechanism for current position -> destination and current position -> portal.

Acceptance criteria:

- A* finds a route around blocked rectangles.
- A* fails when no route exists.
- Same-map route segments can be used by later NPC movement.
- Cross-map route planning can combine portal graph steps with local segment pathing, even if full multi-hop movement is not yet animated.

Suggested tests:

- Straight reachable path.
- Path around an obstacle.
- Blocked/unreachable target.
- Path to a portal entry point.

## Slice 4: Live NPC Runtime State And Snapshots

Goal: stop treating NPCs as only static map payload when movement begins.

Status: `Implemented`

Implemented notes:

- Rooms now initialize live NPC runtime state from static map NPC definitions.
- Sunny Town protocol messages include live NPC snapshots on `hello`, `snapshot`, and `map_changed`.
- Live NPC snapshots preserve dialogue, shop, schoolwork activity, facing, sprite, and durable character identity overlays.
- Static map NPC payloads still exist for compatibility and are not mutated by durable identity overlays.
- Frontend world state tracks live NPC snapshots and renders/interacts with live NPC positions when present, falling back to static map NPCs otherwise.
- Tests cover live NPC snapshots, durable identity overlays, snapshot broadcasts, frontend fallback behavior, and live NPC snapshot application.

Implementation notes:

- Add live NPC state to `room`, likely keyed by durable `characterId` or stable NPC key:
  - `characterID`
  - `npcKey`
  - `displayName`
  - `spriteKey`
  - `mapID`
  - `x`, `y`
  - `facing`
  - current goal/path fields later
  - static interaction data copied or referenced from map NPC defaults
- Add NPC snapshots to the Sunny Town protocol.
- Frontend should render dynamic NPC snapshots if present, while preserving static map NPC behavior during transition.
- Keep current interaction lookup working. The nearby NPC list should use live NPC positions once dynamic NPCs exist.
- Preserve dialogue/shop/schoolwork payloads.

Acceptance criteria:

- Existing static NPCs still render and talk.
- Live NPC snapshots can represent an NPC at a changed position.
- Clients see the same public NPC state.
- No client can authoritatively move an NPC.

Suggested tests:

- Hello/snapshot includes live NPC state when initialized.
- Existing map NPC dialogue survives the transition.
- Frontend build catches type drift.

## Slice 5: Scripted Movement Smoke Test

Goal: one NPC visibly moves between tagged locations without drives yet.

Status: `Implemented`

Implemented notes:

- Sunny Town worlds now build a server-side navigation graph for loaded maps.
- `mayor-sunny` receives a scoped scripted route on maps with `town-square-center`.
- Room ticks advance live NPCs along route-planned same-map path waypoints at a modest server speed.
- Scripted NPCs pause briefly at endpoints, then route back to their starting point.
- NPC movement stays server-owned and continues broadcasting through existing live NPC snapshots.
- Route planning failures or unsupported portal steps leave the NPC idle instead of teleporting or crossing maps.
- Tests cover movement advancement, arrival/stop behavior, blocked routes, and moved NPC snapshot output.

Implementation notes:

- Pick one NPC, likely `mayor-sunny`.
- Give it a simple scripted target:
  - move between town square and another reachable tagged location
  - prefer using portal-aware route planning if a safe two-map target exists
- Use the route/path machinery from Slices 2 and 3.
- Move at a modest server-controlled speed during room ticks.
- Broadcast changing NPC position through snapshots.
- Keep player collision/simple interaction rules conservative:
  - NPC should not pass through blocked map geometry.
  - Do not add complex NPC/player collision yet unless needed.

Acceptance criteria:

- One NPC moves server-side.
- All clients see the same movement.
- NPC dialogue still works when near the NPC's current live position.
- Existing map transitions and player movement still work.

Suggested tests:

- Runtime step advances NPC along path.
- NPC reaches target and stops.
- NPC does not move when no path exists.
- Snapshot includes updated NPC position.

## Slice 6: Basic Drive Runtime

Goal: NPC movement is selected by simple depleting drives.

Status: `Implemented`

Implemented notes:

- Live NPCs now have in-memory hunger, energy, social, and work drives.
- Drives default near full and deplete every room tick.
- Drive tags map to authored locations: hunger to food/meal/kitchen, energy to bed/sleep/rest/home, social to social/public/gathering, and work to work.
- Matching tagged locations replenish the matching drive faster than depletion while an NPC is inside the location radius.
- Idle NPCs choose the lowest below-threshold satisfiable drive and route to a same-map matching location.
- Unrouteable or unavailable urgent drives are skipped so another satisfiable drive can be selected.
- Mayor Sunny is seeded with a low social drive on the town map so the visible movement smoke test remains active through the drive controller.
- Tests cover drive depletion, matching and non-matching replenishment, lowest-drive selection, movement, blocked paths, arrival, and moved snapshots.

Implementation notes:

- Add runtime drive state to live NPC state:
  - `hunger`
  - `energy`
  - `social`
  - `work` or `purpose`
- Drives are `0..100`, defaulting near full.
- Decrease each drive on a fixed interval.
- Drive-to-location tags:
  - hunger -> `food`, `meal`, `kitchen`, `food_source`
  - energy -> `bed`, `sleep`, `rest`, `home`
  - social -> `social`, `public`, `gathering`
  - work/purpose -> assigned workplace first, then `work` plus role tags
- Replenish matching drive quickly while NPC is within location radius.
- Keep drive state in Sunny Town memory for the first pass. Do not add durable drive tables yet.

Acceptance criteria:

- Drives decrease over time.
- NPC chooses a destination for the lowest satisfiable drive below threshold.
- Drive replenishes quickly at a matching location.
- NPC can satisfy several drives over time because replenishment is faster than depletion.

Suggested tests:

- Depletion tick lowers values.
- Matching location replenishes the correct drive.
- Non-matching location does not replenish.
- Lowest satisfiable drive is selected.

## Slice 7: Goal Commitment, Failure, And Re-evaluation

Goal: prevent jitter and handle unsatisfied drives gracefully.

Implementation notes:

- Add goal state:
  - `currentGoal`
  - `focusUntil`
  - `reevaluateAt`
  - `failureCount`
  - temporary failed-target cooldowns
- Rules:
  - Choose the lowest satisfiable drive when idle or current goal is complete.
  - A drive is satisfiable only if at least one matching location exists and a route can be planned.
  - If the most urgent drive is not satisfiable, try the next most urgent drive.
  - Once a goal starts, keep it through a minimum focus window.
  - Reevaluate periodically while focused.
  - Interrupt only for emergency-level drives or failed goal.
  - If a target is reached but the drive does not replenish after a grace period, mark target failed and choose again.
  - If all urgent drives fail, go to idle/wander/public fallback and retry later.

Acceptance criteria:

- NPCs do not rapidly flip between goals.
- Unreachable urgent drive does not trap the NPC forever.
- A failed target is avoided temporarily.
- A more urgent drive can interrupt after reevaluation/emergency threshold.

Suggested tests:

- Two close drives do not cause constant switching.
- Unreachable hunger target falls back to energy/social/work.
- Failed target cooldown prevents immediate retry.
- Focus window prevents premature switching.

## Slice 8: Cross-Map Movement Hardening

Goal: NPCs can use portal routes for real needs across multiple cells.

Implementation notes:

- Animate portal transitions for NPCs.
- Track NPC map membership as it moves across maps.
- Ensure NPC appears only in the relevant map's live state.
- Use portal target coordinates/facing.
- Preserve portal re-entry guard behavior so NPCs do not bounce.
- Add tests for at least one cross-map route. Add multi-hop tests when enough maps exist.

Acceptance criteria:

- NPC can leave one map and appear in another.
- NPC can path to a location on a different map.
- Players in each map see consistent NPC presence.
- No portal bounce loop.

Suggested tests:

- One-hop route and transition.
- Re-entry guard behavior.
- NPC snapshot scoped to current map.

## Open Design Choices

- Where to put navigation code: `internal/sunnytown/server` is simplest at first; `internal/sunnytown/navigation` may be cleaner once it grows.
- Whether live NPC snapshots replace static map NPC payloads fully or coexist for a transition.
- Whether map locations should be authored per map JSON only or also support future HQ-authored/durable locations.
- Whether `ownerNpcKey` is enough for authored locations or whether some locations should reference durable character IDs after load.
- How much debug state to expose in snapshots versus logs only.

## Do Not Do Yet

- Do not persist drives.
- Do not make drives affected by day-night yet.
- Do not make hunger require actual food inventory yet.
- Do not migrate inventory/equipment.
- Do not build schedules.
- Do not build job production/economy.
- Do not implement social relationship scoring.
- Do not run NPC simulation at Dwarf Fortress scale.

## Handoff Prompt

Use this when handing the feature to a fresh implementation agent:

```text
We are working in C:\Users\waltr\Documents\New project.

Implement the next Sunny Town NPC movement slice.

Read these docs first:

- docs/current/NPC_CHARACTER_MODEL_PLAN.md
- docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md
- docs/current/NPC_MOVEMENT_IMPLEMENTATION_PLAN.md

Durable NPC identity is already implemented. Static map NPCs can be backed by durable character rows, and Sunny Town can merge `characterId` into map NPC definitions. Do not redo that work.

Start with the next incomplete slice in docs/current/NPC_MOVEMENT_IMPLEMENTATION_PLAN.md. Implement only that slice, keep the change small, and update the plan with what changed.

Current target sequence:
1. Map location tags.
2. Portal-aware route model and nav graph.
3. Same-map A* segment pathing.
4. Live NPC runtime state and snapshots.
5. Scripted movement smoke test.
6. Basic drive runtime.
7. Goal commitment/failure/reevaluation.
8. Cross-map movement hardening.

Constraints:
- Sunny Town owns live movement.
- HQ owns durable identity.
- Clients must not authoritatively move NPCs.
- Existing NPC dialogue/shop/schoolwork interactions must keep working.
- Existing maps without locations must keep loading.
- Do not migrate inventory/equipment.
- Do not implement day-night, schedules, economy, complex mood, or GOAP.

Use `rg` for search and `apply_patch` for manual edits. Do not stage local logs.

Verify with:

go test ./...

Then:

cd frontend
npm run build
```
