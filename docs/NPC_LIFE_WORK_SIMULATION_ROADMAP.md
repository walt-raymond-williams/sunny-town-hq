# NPC Life And Work Simulation Roadmap

## Purpose

This roadmap decomposes GitHub epic #38 into child issues that can be assigned to agents.

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

The first target is a visible, inspectable NPC home/work loop that uses authored homes, beds, work anchors, existing portal navigation, and server-authoritative routine state.

## Recommended First Slice

Create a short demo loop:

- an NPC has a home/bed
- the same NPC has a work anchor
- the NPC can route between those places, including portals if needed
- debug state explains current goal and route
- the loop is observable in minutes, not hours

This should be a practical scaffolding slice for richer routines later.

## Phase Order

1. Map/home authoring and route verification.
2. Routine goal selection for home/rest versus work.
3. Debug/inspectability improvements.
4. Optional frontend debug panel or richer visual cues.
5. Production and progression integration once the routine is understandable.

## GitHub-Ready Child Issues

### Issue: Design NPC Home And Work Demo Loop

Type: research
Priority: P1

Blocked by:

- None

Related:

- Epic #38

User story:

As a developer, I want a focused home/work demo loop design so that NPC life simulation starts with one observable behavior instead of a broad AI rewrite.

Acceptance criteria:

- Select the first demo NPC.
- Select home/bed and work locations.
- Decide whether the first loop is schedule-driven, drive-driven, or hybrid.
- Decide whether a second home requires a new interior map or can use an existing map.
- Document the expected demo cadence.
- Identify child issues and blockers.

Implementation notes:

- Start from `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`.
- Audit `sunny-town/maps/*.json`.
- Prefer existing map/location/portal primitives.
- Avoid adding durable schema until a stable gameplay concept needs it.

Verification:

- Documentation review against current maps and NPC movement docs.

### Issue: Author NPC Homes And Bed Locations

Type: feature
Priority: P1

Blocked by:

- Design NPC Home And Work Demo Loop

Related:

- Epic #38

User story:

As a player, I want NPCs to have visible homes with beds so they feel like residents of Sunny Town.

Acceptance criteria:

- At least one additional home/bed candidate is authored if the map layout supports it.
- Existing `mayor-sunny-bed` remains valid and reachable.
- Home/bed locations use `locations` with `home`, `bed`, and `sleep` tags.
- Owned beds use `ownerNpcKey`.
- Map validation tests cover the authored locations.
- Route tests prove the selected NPC can reach home and work.

Implementation notes:

- Existing home candidate: `sunny-town-house-1` location `mayor-sunny-bed`.
- Existing work candidates:
  - `cookie-keeper-counter` for Cookie Keeper
  - `teacher-desk-work` for Teacher
  - town square/public role for Mayor Sunny if a work marker is added
- Main map has building-shaped blocked areas that may become future houses, but adding a second house likely requires a new interior map and portal.
- A bed can start as a `location`; visible fixture art can be a later issue unless needed for the demo.

Verification:

- `go test ./internal/sunnytown/maps`
- `go test ./internal/sunnytown/server`

### Issue: Add Observable Home/Work Routine Goal Selection

Type: feature
Priority: P1

Blocked by:

- Author NPC Homes And Bed Locations

Related:

- Epic #38

User story:

As a player, I want an NPC to go home to rest and then go to work so the town has a readable routine.

Acceptance criteria:

- One selected NPC alternates between home/rest and work goals on a demo-friendly cadence.
- The NPC routes to the selected home/bed and work locations.
- The routine avoids rapid goal flipping.
- Server snapshots keep players in sync as the NPC changes maps.
- The implementation does not make clients authoritative over NPC movement or routine choice.

Implementation notes:

- Reuse current drive state and anchor resolution where possible.
- A small schedule pressure layer may be enough if pure drive behavior is hard to tune.
- Keep runtime routine state in Sunny Town memory for the first slice.
- Do not persist raw route/path indexes.

Verification:

- `go test ./internal/sunnytown/server`
- Manual smoke: observe the NPC route home and to work.

### Issue: Improve NPC Routine Debug Output

Type: feature
Priority: P1

Blocked by:

- Add Observable Home/Work Routine Goal Selection

Related:

- Epic #38
- Issue #4 production posture for NPC debug endpoint

User story:

As a developer or designer, I want to inspect an NPC's current routine state so I can understand why it is moving, idle, working, or blocked.

Acceptance criteria:

- `/debug/npcs` identifies current goal, anchor kind, target map/location, route progress, drive values, and current schedule/routine pressure where applicable.
- Failed route or blocked target state is visible.
- Cookie Keeper production blocked reasons remain visible through logs or debug output.
- Debug endpoint remains protected according to the accepted local/production posture.
- Tests protect the debug fields most useful for routine diagnosis.

Implementation notes:

- Existing debug endpoint: `internal/sunnytown/server/npc_debug.go`.
- Existing tech debt: issue #4 defines production posture for the debug endpoint.
- Keep debug output structured JSON; avoid relying only on logs.

Verification:

- `go test ./internal/sunnytown/server`
- Manual: call `/debug/npcs` while the demo NPC is traveling, resting, and working.

### Issue: Add Optional NPC Routine Visual Cues

Type: feature
Priority: P2

Blocked by:

- Improve NPC Routine Debug Output

Related:

- Epic #38

User story:

As a player, I want small visual cues for NPC routines so I can tell when an NPC is working, resting, or blocked without opening debug tools.

Acceptance criteria:

- NPC status cues are subtle and do not clutter normal play.
- The UI can distinguish at least traveling, resting, working, and blocked states if those states are available.
- No fake state is shown; cues reflect server-authoritative NPC state.
- The feature is optional for the first MVP if debug tools are enough.

Implementation notes:

- Consider a tiny status label in debug/dev mode first.
- Avoid broad UI redesign.
- Use existing snapshot/debug data rather than adding client-only inference.

Verification:

- `cd frontend; npm run build`
- Manual smoke in Sunny Town.

## Open Product Questions

- First demo NPC:
  - Mayor Sunny is already in town and has an owned bed.
  - Cookie Keeper already has a work anchor and production behavior.
  - Teacher already has a classroom work anchor.
- Second home:
  - Add a new interior map and portal for a second house?
  - Repurpose an existing unoccupied building shape first?
  - Wait until the first home/work loop proves useful?
- Routine model:
  - Pure drive-based behavior?
  - Explicit schedule pressure?
  - Hybrid: schedule pushes work/sleep, drives choose fallback?
- Bed representation:
  - Location-only first?
  - Visible bed fixture in the map?
  - New furniture art?

## Recommended Next Management Step

Create the first child issue:

`Design NPC Home And Work Demo Loop`

Why:

- It locks the first NPC, home, work anchor, and cadence before agents edit maps or routine code.
- It can answer the remaining product questions cheaply.
- It reduces the risk of overbuilding a general AI system before the demo loop is clear.
