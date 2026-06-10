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

- Keep the existing `sunny-town-house-1` home/bed as one home candidate.
- Add a second home only if the map and portal layout can support it without a broad map redesign.
- Do not invent a parallel "cell" or building abstraction. Use existing maps, portals, locations, fixtures, and owner hints.

## User Stories

### STORY-NPC-LIFE-001: NPCs Have Visible Homes

As a player, I want NPCs to have homes with beds so that town residents feel like they live somewhere instead of only standing near shops.

Acceptance criteria:

- At least one NPC has an authored home/bed location.
- Home/bed locations use existing map `locations`.
- Homes can be reached by server-side NPC pathing.
- Owned beds are preferred over generic rest locations for the owning NPC.
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
- Cross-map routines use existing portal route planning.

### REQ-NPC-LIFE-002: Home/Work Anchors Are Deterministic

NPC home and work anchors must resolve predictably from authored map locations and owner hints.

Acceptance criteria:

- Owned home/bed beats generic home/bed when reachable.
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

1. Author a second home candidate if practical, and add/verify bed locations.
2. Assign one NPC, likely Mayor Sunny or Teacher, a clear home bed and work location.
3. Make the NPC visibly route between home and work on a fast demo cadence.
4. Improve `/debug/npcs` or a small frontend/debug view so the current routine is obvious.
5. Keep Cookie Keeper production as the storage-aware work example, but do not require NPC-held inventory.

Recommended first implementation path:

- Start with map authoring and route tests for homes/beds.
- Then add routine scheduling or drive pressure that strongly alternates home/rest and work.
- Then add debug/inspectability polish.

## Open Questions

- Which NPC should be the first home/work demo resident: Mayor Sunny, Cookie Keeper, Teacher, or a new resident?
- Should the first second home be a new interior map or an existing building repurposed with a new portal?
- Should a bed be represented only as a `location` first, or also as a visible fixture/world object?
- Should the first routine be schedule-driven, drive-driven, or a small hybrid?
- Should work/sleep state be visible in normal UI, debug-only UI, or just `/debug/npcs` for the first slice?
- How fast should the demo cadence be?

## Suggested Next Document

Create `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md` with GitHub-ready child issues and recommended sequencing.
