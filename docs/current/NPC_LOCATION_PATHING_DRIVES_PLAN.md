# NPC Location, Pathing, And Drives Plan

This document captures the future Sunny Town design for location tags, NPC pathing, and a basic drive-based movement loop. It builds on `docs/current/NPC_CHARACTER_MODEL_PLAN.md`. The sliced implementation sequence is tracked in `docs/current/NPC_MOVEMENT_IMPLEMENTATION_PLAN.md`.

Status: `Concept plan for post-durable-NPC movement work`

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

Status: `Future`

Sunny Town maps should support optional location definitions.

Acceptance criteria:

- Location IDs are unique within a map.
- Location tags are non-empty strings when present.
- Invalid locations fail map validation.
- Maps without locations remain valid.

### REQ-PATH-001: Server-Authoritative NPC Navigation

Status: `Future`

NPC pathing and movement must be owned by Sunny Town.

Acceptance criteria:

- Clients do not authoritatively choose NPC paths.
- Sunny Town validates walkable positions.
- Pathing can account for map blocked rectangles.
- Route planning can include portal edges between maps.
- Pathing can later account for dynamic collision objects.

### REQ-DRIVE-001: Basic Drive State Belongs To NPC Simulation

Status: `Future`

NPC drives should be simulation state layered on top of shared character state.

Acceptance criteria:

- Drive values are not player-only state.
- Drive values are not stored in map JSON.
- Initial drive values can be simple runtime state with defaults.
- Drives decrease over time and replenish near matching tagged locations.
- NPC controllers maintain goal focus windows, reevaluation intervals, and failed-target cooldowns.
- Durable drive snapshots may live in HQ later, but transient decision state can stay in Sunny Town memory.
- The drive system can be paused or coarse-caught-up when no players are connected.

### REQ-MP-LOC-001: Multiplayer Consistency

Status: `Future`

All players in the same room should see the same public NPC movement and location-driven behavior.

Acceptance criteria:

- Public NPC position/state is room-shared.
- Private player dialogue/progress can remain player-specific.
- NPC movement snapshots are server broadcast.
- NPC interactions use server-validated position and capability state.

## Implementation Plan

### Phase A: Location Tags Only

- Add optional `locations` to map JSON types.
- Validate location IDs, coordinates, radius, and tags.
- Add tests for map loading and validation.
- Do not move NPCs yet.

Deliverable: maps can describe meaningful places.

### Phase B: Route Planning And Same-Map Pathing

- Build a map/portal route graph from loaded maps.
- Build a simple nav grid from map dimensions and blocked rectangles.
- Implement A* for same-map route segments.
- Define the route result shape so it can include portal transitions.
- Add pathing tests for blocked routes, reachable same-map routes, and at least one cross-map route plan.
- Keep dynamic objects out of the first pass if needed.

Deliverable: server can compute walkable same-map paths and represent cross-map routes.

### Phase C: Static Goal Movement

- Give one NPC a scripted goal to move between two tagged locations.
- Prefer a route that exercises portal-aware planning if practical; same-map movement is acceptable only as a smaller smoke test.
- Broadcast live NPC position through Sunny Town snapshots.
- Preserve current dialogue interactions.

Deliverable: first visible moving NPC.

### Phase D: Basic Drive-Based Goal Selection

- Add a small set of drive values to NPC runtime state, such as hunger, energy, social, and work/purpose.
- Decrease those values on a fixed interval.
- Map drives to location tags.
- Let one NPC choose the lowest satisfiable drive and move to a matching location.
- Replenish that drive quickly while the NPC remains near the destination.
- Add focus windows, reevaluation intervals, and failed-target cooldowns to prevent jitter.

Deliverable: first intentional-feeling NPC routine.

### Phase E: Richer Cross-Map Movement

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

This feature should come after durable NPC character identity exists.

Recommended order:

1. `NPC_CHARACTER_MODEL_PLAN.md` Phase 3A: make current static NPCs durable character-backed entities.
2. This plan Phase A: add map location tags.
3. This plan Phase B/C: add portal-aware route planning, pathing, and one simple moving NPC.
4. This plan Phase D: add basic drive-based destination choice with fixed depletion and fast replenishment.

The key dependency is identity: movement, drives, jobs, and home/work assignments should attach to durable NPC characters, not anonymous map fixtures.
