# NPC Life And Work Simulation Roadmap

## Purpose

This roadmap decomposes GitHub epic #38 into child issues that can be assigned to agents.

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

The first target is a visible, inspectable NPC home/work loop that uses authored homes, beds, work anchors, existing portal navigation, and server-authoritative routine state.

## Recommended First Slice

Create a short demo loop:

- Cookie Keeper has an explicit home
- Cookie Keeper has a visible bed fixture in that home
- Cookie Keeper keeps `cookie-keeper-counter` as his work anchor
- Cookie Keeper can route between home and work through portals
- debug state explains current goal and route
- the loop is observable in minutes, not hours

This should be a practical scaffolding slice for richer routines later.

Initial cadence decision:

- Default/local-play day length target: `24` real minutes.
- Demo/dev day length target: `8` real minutes.
- First routine model: drive-driven. Time of day may influence drive pressure later, but the first slice should not require a hard schedule.

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

- Confirm Cookie Keeper as the first demo NPC.
- Select the unoccupied main-town building that will receive the new home portal.
- Define the new interior map ID and portal IDs.
- Define home-area and bed-fixture metadata.
- Confirm drive-driven routine behavior and document whether time of day influences drive pressure in the first slice.
- Document the expected demo cadence: `8` minute demo day and `24` minute default day.
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

- A new home interior map is authored for Cookie Keeper.
- A portal from an unoccupied Sunny Town building leads to the new home interior.
- The home contains at least one visible bed fixture.
- Existing `mayor-sunny-bed` remains valid and reachable.
- Home areas use `locations` with `home` tags and owner/home assignment metadata where needed.
- Bed fixtures or associated bed locations use `bed` and `sleep` semantics.
- Bed fixture metadata is shaped so future player-placed beds can produce equivalent usable beds.
- Map validation tests cover the authored locations.
- Route tests prove Cookie Keeper can reach home and work.

Implementation notes:

- Existing home candidate: `sunny-town-house-1` location `mayor-sunny-bed`.
- Existing work candidates:
  - `cookie-keeper-counter` for Cookie Keeper
  - `teacher-desk-work` for Teacher
  - town square/public role for Mayor Sunny if a work marker is added
- Main map has building-shaped blocked areas that may become future houses. For this epic, add a new interior map and portal for Cookie Keeper's home.
- Beds should be visible fixtures from the start. They can also have associated map locations if that keeps existing drive/location routing simple.
- Do not implement player placement yet, but avoid metadata that would make player-placed beds incompatible later.

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

- Cookie Keeper alternates between home/rest and work goals on a demo-friendly cadence.
- The NPC routes to the selected home/bed and work locations.
- Home/rest goal chooses an available bed inside Cookie Keeper's assigned home.
- The routine avoids rapid goal flipping.
- Server snapshots keep players in sync as the NPC changes maps.
- The implementation does not make clients authoritative over NPC movement or routine choice.

Implementation notes:

- Reuse current drive state and anchor resolution where possible.
- Prefer drive-driven behavior first. A small time-of-day pressure layer can later modify drive rates, but should not replace drives with a hard schedule.
- Start with an `8` minute demo day target and a `24` minute default/local-play day target.
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

- Second home:
  - Which unoccupied main-town building should become Cookie Keeper's home entrance?
  - How much should the new interior reuse the layout language of `sunny-town-house-1`?
- Routine model:
  - Pure drive-based behavior first?
  - Hybrid later: time of day changes drive pressure while drives still choose goals?
- Bed representation:
  - What metadata should mark a bed fixture as usable?
  - Should an authored bed fixture automatically create a matching sleep location, or should both be authored explicitly?
  - What future item key should represent player-placeable beds?

## Recommended Next Management Step

Create the first child issue:

`Design NPC Home And Work Demo Loop`

Why:

- It locks the first NPC, home, work anchor, and cadence before agents edit maps or routine code.
- It can answer the remaining product questions cheaply.
- It reduces the risk of overbuilding a general AI system before the demo loop is clear.
