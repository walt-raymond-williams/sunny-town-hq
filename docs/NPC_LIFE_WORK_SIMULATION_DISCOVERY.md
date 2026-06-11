# NPC Life And Work Simulation Discovery

## Purpose

This document starts discovery for GitHub epic #38, "Epic: NPC life and work simulation."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

The goal is to define a practical first version of Sunny Town NPC life/work behavior: NPCs should visibly move between meaningful places, such as home/bed and work, and their behavior should be inspectable enough that future routines can be debugged instead of guessed.

## Product Direction

Sunny Town should feel like a small living town. NPCs do not need deep AI at first. They need readable routines:

- go home and sleep/rest
- go to work
- use authored homes, beds, shops, kitchens, classrooms, and public locations
- produce or fail to produce for understandable reasons
- expose enough debug state that a human can answer "where is this NPC going and why?"

The first strong demo loop should be simple and visible: an NPC travels between a home/bed and a workplace. That home/work rhythm becomes the test bed for later hunger, energy, social, shop production, and progression behavior.

## Existing Baseline

Current implemented pieces already support much of this epic:

- NPCs have durable character identity through `sunny_town_character` and `sunny_town_npc_character`.
- Maps support authored `locations` with IDs, names, coordinates, radius, tags, owner hints, and capacity.
- Sunny Town builds server-side portal-aware navigation routes.
- NPCs can move through portals and appear in the correct room snapshots.
- NPCs have runtime drives: hunger, energy, social, and work.
- Drive values deplete over time and replenish at matching tagged locations.
- NPC controllers can select reachable locations using drive tags and owner hints.
- `/debug/npcs` exposes NPC movement/debug state behind service-secret auth.
- Cookie Keeper has a work anchor at `cookie-keeper-counter`.
- Cookie Shop production consumes HQ-owned input ingredients, respects output capacity, and records blocked attempts.

Current map facts:

- `sunny-town-v1` has Mayor Sunny in the town square and a portal into `sunny-town-house-1`.
- `sunny-town-v1` also has a classroom portal and multiple building-shaped blocked areas.
- `sunny-town-house-1` has:
  - `mayor-sunny-bed`, tagged `rest`, `home`, `bed`, `sleep`, `personal`, owned by `mayor-sunny`
  - `shared-house-kitchen`, tagged `food`, `kitchen`, `meal`
  - `cookie-shop`, tagged `shop`, `workplace`, `cookie_shop`, `storage_owner`
  - `cookie-keeper-counter`, tagged `work`, `shop`, `merchant`, owned by `cookie-keeper`
  - Cookie Shop input/output chest fixtures
- `sunny-town-classroom` has:
  - Teacher NPC
  - `teacher-desk-work`, tagged `work`, `school`, owned by `teacher`
  - `classroom-study-circle`, tagged `public`, `social`, `idle`, `school`

## New Product Input

For demo and development, we should author homes early:

- Create a house out of one unoccupied Sunny Town building.
- Prefer making two houses/home interiors if the map layout supports it cleanly.
- Put beds in those homes.
- Assign at least one NPC a home/bed and a workplace so we can watch a repeatable sleep/work loop.
- Use this loop to test pathing, portals, scheduling pressure, drive choice, and debug output.

Initial interpretation:

- Use Cookie Keeper as the first demo NPC because he already has a job, work anchor, shop storage, and production behavior.
- Keep the existing `sunny-town-house-1` home/bed as one home candidate.
- Add a second home as a new interior map with a new portal from an unoccupied Sunny Town building.
- Represent beds as visible fixtures/world objects, not only invisible locations. Longer term, beds should become inventory/placeable furniture that can create a usable bed fixture when placed.
- For the first slice, keep home assignment explicit: an NPC has a home location/area. Inside that home, the NPC can use an available bed rather than permanently owning one specific bed object.
- Use drive-driven behavior first. Do not require a hard global schedule before the drive loop proves useful.
- Use a demo-friendly time scale so sleep/work can be observed during manual testing.
- Do not invent a parallel "cell" or building abstraction. Use existing maps, portals, locations, fixtures, and owner hints.

## Cadence Recommendation

Reference games use compressed time because readable routine loops need to happen while a player is still paying attention:

- Minecraft has a 20-minute full day/night cycle.
- The Sims series commonly lands around a 24-to-36-minute day depending on title/settings, with time acceleration available.
- RimWorld uses a 24-hour in-game day with pawn schedules, sleep, work, and recreation blocks rather than a purely cosmetic day/night cycle.

Sunny Town should start with two explicit cadence modes:

- Default/local-play cadence: `24` real minutes per simulated day.
- Demo/dev cadence: `8` real minutes per simulated day.

Why:

- `24` minutes is close enough to familiar life-sim pacing that work/rest can matter without feeling frantic.
- `8` minutes lets developers and reviewers observe a full home/work/rest loop in one short session.
- The first implementation can expose this as a server-side constant or config value; clients must not control simulation time.

Initial phase split for an `8` minute demo day:

- Work pressure rises for roughly `3` to `4` minutes.
- Energy/rest pressure rises enough to send the NPC home after a visible work period.
- Resting at a bed should recover quickly, roughly `60` to `90` seconds, so the NPC can return to work.
- Food/social can remain secondary fallback drives until the home/work loop is stable.

## User Stories

### STORY-NPC-LIFE-001: NPCs Have Visible Homes

As a player, I want NPCs to have homes with beds so that town residents feel like they live somewhere instead of only standing near shops.

Acceptance criteria:

- At least one NPC has an authored home/bed location.
- Home/bed locations use existing map `locations`.
- Beds are represented as visible fixtures/world objects in the home.
- Homes can be reached by server-side NPC pathing.
- NPC home assignment can identify the home area, and bed choice can use an available bed in that home.
- Map validation and NPC pathing tests cover the authored home location.

### STORY-NPC-LIFE-002: NPCs Move Between Home And Work

As a player, I want an NPC to move between home and work so that I can see a daily routine instead of random wandering.

Acceptance criteria:

- At least one NPC has both a home/rest anchor and a work anchor.
- The NPC can route from home to work and from work to home.
- Cross-map routes use existing portal concepts.
- Players see the NPC leave one map and appear in the target map through server snapshots.
- The routine avoids rapid flipping between goals.

### STORY-NPC-LIFE-003: NPCs Sleep Or Rest At Beds

As a player, I want NPCs to sleep/rest at a bed when tired so their behavior has an understandable cause.

Acceptance criteria:

- NPC energy or rest drive can choose a home/bed location.
- Staying near the bed replenishes the related drive.
- The selected bed is inside the NPC's assigned home for the first implementation.
- Debug state identifies the active rest/sleep goal.
- The first implementation may use a visibly accelerated debug/demo cadence.

### STORY-NPC-LIFE-004: NPCs Go To Work For A Reason

As a player, I want shopkeepers and workers to go to work so their production and services feel grounded in the world.

Acceptance criteria:

- NPC work drive or schedule pressure can select an owned work anchor.
- Cookie Keeper work remains tied to `cookie-keeper-counter`.
- Cookie production still happens only when Sunny Town validates Cookie Keeper is working at the authored work anchor.
- Blocked production remains HQ-owned and idempotent.

### STORY-NPC-LIFE-005: NPC Routine State Is Inspectable

As a developer or designer, I want to inspect an NPC's current goal, drive state, target, and blocked reason so I can tune routines and debug pathing.

Acceptance criteria:

- Debug output includes current map, position, active goal, anchor kind, target location, route state, drive values, and failure/blocking reason where available.
- Debug output remains protected by service-secret auth or an explicitly accepted local-only posture.
- Tests cover the debug shape enough to protect useful fields.

### STORY-NPC-LIFE-006: NPC Behavior Remains Server-Authoritative

As a multiplayer player, I want every player to see the same NPC movement and routine state so the town does not desync.

Acceptance criteria:

- Clients do not authoritatively move NPCs.
- Sunny Town owns live NPC position, routes, drive choice, and snapshots.
- HQ owns durable identity and durable economy/storage mutations.
- Browser messages cannot force NPC work, movement, or production outcomes directly.

## Requirements

### REQ-NPC-LIFE-001: Use Existing Map And Portal Concepts

NPC homes, beds, workplaces, kitchens, public areas, and shops must be authored with existing map `locations`, `fixtures`, and `portals`.

Acceptance criteria:

- No parallel cell/building abstraction is introduced for the first life/work slice.
- Home/work assignments reference map IDs and location IDs.
- Bed fixtures are authored in maps first and should be shaped so future player placement can create equivalent fixtures.
- Cross-map routines use existing portal route planning.

### REQ-NPC-LIFE-002: Home/Work Anchors Are Deterministic

NPC home and work anchors must resolve predictably from authored map locations and owner hints.

Acceptance criteria:

- Explicit home assignment beats generic home/bed when reachable.
- Available beds inside the assigned home beat generic rest locations elsewhere.
- Owned work anchor beats generic work locations when reachable.
- Unreachable owned anchors degrade to a safe fallback instead of trapping the NPC.
- Tests cover owned and fallback behavior.

### REQ-NPC-LIFE-003: First Routine Cadence Is Demo-Friendly

The first routine loop should be observable in a short manual smoke test.

Acceptance criteria:

- The first home/work loop can be observed without waiting for a real-world day.
- Cadence is controlled by simulation constants or a debug/demo-friendly schedule, not client authority.
- Production defaults can remain conservative if behavior is inspectable.

### REQ-NPC-LIFE-004: Production Stays Storage-Aware

Work simulation that produces goods must use the existing HQ-owned storage and recipe model.

Acceptance criteria:

- Sunny Town validates that an NPC is at the work anchor.
- HQ decides whether inputs and output capacity allow production.
- Missing inputs or full output storage record blocked attempts without partial mutation.
- Do not create NPC-held Cookie Keeper inventory unless a future story requires it.

### REQ-NPC-LIFE-005: Debuggability Comes Before Complexity

Every new routine behavior should be inspectable enough to support agent handoff and human review.

Acceptance criteria:

- Debug endpoint or logs show why an NPC chose its current goal.
- Failed routes or blocked work attempts are visible to a developer.
- Child issues include verification steps that inspect behavior, not only compile tests.

## MVP Recommendation

Build the epic around a first visible home/work loop:

1. Use Cookie Keeper as the first demo NPC.
2. Add a second home as a new interior map and portal, with at least one visible bed fixture.
3. Assign Cookie Keeper an explicit home and keep `cookie-keeper-counter` as the work anchor.
4. Make Cookie Keeper visibly route between home and work on a drive-driven demo cadence.
5. Improve `/debug/npcs` or a small frontend/debug view so the current routine is obvious.
6. Keep Cookie Keeper production as the storage-aware work example, but do not require NPC-held inventory.

Recommended first implementation path:

- Start with map authoring and route tests for homes/beds.
- Then add routine scheduling or drive pressure that strongly alternates home/rest and work.
- Then add debug/inspectability polish.

## Accepted Demo Loop Design

This section closes GitHub issue #48 and fixes the concrete first-slice decisions before implementation agents edit maps or routine code.

### First NPC And Anchors

- First demo NPC: `cookie-keeper`.
- Existing work anchor remains `sunny-town-house-1` location `cookie-keeper-counter`.
- Cookie Keeper gets a new explicit home/rest anchor in a new home interior map.
- The first durable assignment remains authored through map metadata, not a new HQ schema table. The bed/rest location uses `ownerNpcKey: "cookie-keeper"` so existing runtime anchor resolution can treat the usable bed as the strong home anchor.

### Main-Town Home Entrance

Use the unoccupied northwest building in `sunny-town-v1`.

Current blocked building rectangle:

```json
{ "x": 160, "y": 128, "width": 224, "height": 160 }
```

Recommended doorway shape for the map-authoring issue:

- Portal ID: `cookie-keeper-home-door`
- Portal rectangle: `x: 240`, `y: 264`, `width: 64`, `height: 24`
- Target map ID: `sunny-town-cookie-keeper-home`
- Target coordinates: `targetX: 320`, `targetY: 416`, `targetFacing: "up"`
- Split the existing blocked building rectangle into a body plus doorway shoulders:
  - `{ "x": 160, "y": 128, "width": 224, "height": 136 }`
  - `{ "x": 160, "y": 264, "width": 80, "height": 24 }`
  - `{ "x": 304, "y": 264, "width": 80, "height": 24 }`

This mirrors the existing south-house doorway pattern: the portal occupies a small open doorway gap while the building shoulders remain blocked.

### New Interior Map

Recommended new map:

- Map ID: `sunny-town-cookie-keeper-home`
- Display name: `Cookie Keeper Home`
- Layout baseline: mirror the compact `sunny-town-house-1` dimensions and exit-door pattern first (`20x15`, `tileSize: 32`) so pathing behavior stays familiar.
- Exit portal ID: `cookie-keeper-home-exit-door`
- Exit portal rectangle: `x: 304`, `y: 448`, `width: 64`, `height: 32`
- Exit target: `targetMapId: "sunny-town-v1"`, `targetX: 272`, `targetY: 320`, `targetFacing: "down"`
- Spawn: `{ "x": 320, "y": 416 }`

Recommended first locations:

```json
{
  "id": "cookie-keeper-home",
  "name": "Cookie Keeper Home",
  "x": 224,
  "y": 224,
  "radius": 128,
  "tags": ["home", "personal"]
}
```

```json
{
  "id": "cookie-keeper-bed",
  "name": "Cookie Keeper Bed",
  "x": 160,
  "y": 224,
  "radius": 64,
  "tags": ["rest", "home", "bed", "sleep", "personal"],
  "ownerNpcKey": "cookie-keeper",
  "capacity": 1
}
```

Recommended first fixture:

```json
{
  "id": "cookie-keeper-bed-fixture",
  "name": "Cookie Keeper Bed",
  "kind": "bed",
  "x": 128,
  "y": 96,
  "width": 96,
  "height": 64,
  "interactionRadius": 56,
  "collision": true,
  "reservesPlacement": true,
  "locationId": "cookie-keeper-bed",
  "itemKey": "simple_bed",
  "tags": ["furniture", "bed", "sleep", "rest", "placeable"]
}
```

The current fixture validator only accepts `kind: "chest"`, so the map-authoring implementation must extend fixture validation and world-object initialization for `kind: "bed"` before adding this fixture to checked-in maps.

### Bed Location Versus Fixture

For the first slice, author both:

- A `location` is the gameplay target used by NPC routing, drive replenishment, anchors, and debug output.
- A `fixture` is the visible/interactable object that can later be created by player-placeable furniture.

The authored bed fixture should reference the matching bed location through `locationId`. Future player placement can create an equivalent bed fixture and either create/register a matching runtime location or use a later route-target abstraction, but the first implementation should not force route selection to understand fixture-only targets.

### Routine Model And Cadence

Use drive-driven behavior with server-owned schedule pressure.

- Existing drives remain the source of decisions.
- Time of day may adjust drive selection pressure for strong anchors, as current code already does for work/rest/social/food phases.
- Do not replace drives with a hard-coded schedule in the first demo loop.
- Avoid rapid flipping through existing focus windows, reevaluation intervals, and failed-target cooldowns.

Cadence should be server-owned runtime configuration:

- Default/local-play target: `24` real minutes per simulated day.
- Demo/dev target: `8` real minutes per simulated day.
- Proposed config key: `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`.
- Clients must not control simulation time.
- If config is absent, use the default `24` minute day.
- Docker/dev compose may set the demo value after the first loop is stable enough for review.

The implementation issue should adapt the current UTC-hour phase helper into a compressed Sunny Town simulation clock while preserving the existing phase semantics (`morning`, `day`, `evening`, `night`) and debug visibility.

### Follow-Up Issues

Create these child issues under epic #38:

1. `Author Cookie Keeper home map and bed fixture`
   - Add the new home map, portal pair, bed location, and visible bed fixture.
   - Extend fixture validation/rendering for `kind: "bed"`.
   - Add map validation and route tests proving Cookie Keeper can route between `cookie-keeper-bed` and `cookie-keeper-counter`.

2. `Add configurable NPC day cadence`
   - Add a server-owned compressed NPC day clock using `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`.
   - Keep the default at `24` minutes and document the `8` minute demo/dev setting.
   - Preserve debug output for current phase and schedule pressure.

3. `Tune Cookie Keeper home/work demo loop`
   - Use existing runtime anchors and drive selection to make Cookie Keeper visibly travel between home/rest and work.
   - Keep Cookie Keeper production tied to `cookie-keeper-counter`.
   - Verify the loop avoids rapid flipping and remains server-authoritative.

4. `Improve NPC routine debug output for demo review`
   - Ensure `/debug/npcs` is enough to inspect Cookie Keeper's active goal, anchor kind, target map/location, phase, pressure, route state, and blocked/failure reason while he is traveling, resting, and working.

## Open Questions

- Should work/sleep state become visible in normal UI, debug-only UI, or just `/debug/npcs` after the first backend demo loop works?

## Suggested Next Document

Create `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md` with GitHub-ready child issues and recommended sequencing.
