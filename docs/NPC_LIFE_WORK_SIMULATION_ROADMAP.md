# NPC Life And Work Simulation Roadmap

## Purpose

This roadmap decomposes GitHub epic #38 into child issues that can be assigned to agents.

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

Tracking: `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`

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
- First routine model: drive-driven with server-owned schedule pressure. Time of day can bias drive selection for strong anchors, but the first slice should not replace drives with a hard schedule.

Accepted design decisions from issue #48:

- Cookie Keeper is the first demo NPC.
- The northwest unoccupied building in `sunny-town-v1` becomes Cookie Keeper's home entrance.
- Main-town portal ID: `cookie-keeper-home-door`.
- New interior map ID: `sunny-town-cookie-keeper-home`.
- Interior exit portal ID: `cookie-keeper-home-exit-door`.
- Cookie Keeper home/rest location IDs: `cookie-keeper-home` and `cookie-keeper-bed`.
- The bed is both a routeable `location` and a visible `fixture`; the fixture references the location through `locationId`.
- Proposed bed fixture kind: `bed`; proposed future furniture item key: `simple_bed`.
- NPC day cadence should be a server-owned runtime config value, proposed as `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`, defaulting to `24`.

## Phase Order

1. Map/home authoring, bed fixture support, and route verification.
2. Configurable NPC day cadence.
3. Routine tuning for Cookie Keeper home/rest versus work.
4. Debug/inspectability improvements.
5. Optional frontend debug panel or richer visual cues.
6. Production and progression integration once the routine is understandable.

## GitHub-Ready Child Issues

### Issue: Design NPC Home And Work Demo Loop

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/48

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

Implemented design outcome:

- The selected building is the northwest blocked building in `sunny-town-v1`.
- The new portal IDs, map ID, location IDs, bed fixture metadata, routine model, and cadence config shape are captured in `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`.
- Follow-up issues are listed below and can be created directly from this roadmap.

### Issue: Author Cookie Keeper Home Map And Bed Fixture

Type: feature
Priority: P1

Blocked by:

- Design NPC Home And Work Demo Loop

Related:

- Epic #38

User story:

As a player, I want Cookie Keeper to have a visible home with a usable bed so he feels like a resident of Sunny Town.

Acceptance criteria:

- `sunny-town-v1` has a `cookie-keeper-home-door` portal in the northwest unoccupied building.
- The existing northwest building blocked rectangle is split to leave a reachable doorway gap.
- A new `sunny-town-cookie-keeper-home` map is authored.
- The new home map has a `cookie-keeper-home-exit-door` portal back to `sunny-town-v1`.
- The home contains `cookie-keeper-home` and `cookie-keeper-bed` locations.
- `cookie-keeper-bed` uses tags `rest`, `home`, `bed`, `sleep`, and `personal`, with `ownerNpcKey: "cookie-keeper"` and `capacity: 1`.
- The home contains a visible `cookie-keeper-bed-fixture` with `kind: "bed"` and `locationId: "cookie-keeper-bed"`.
- Fixture validation and world-object initialization support `kind: "bed"` without regressing existing chest fixtures.
- Existing `mayor-sunny-bed` remains valid and reachable.
- Map validation tests cover the authored locations.
- Route tests prove Cookie Keeper can route from `cookie-keeper-bed` to `cookie-keeper-counter` and back.

Implementation notes:

- Use the accepted design in `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`.
- Main-town portal: `cookie-keeper-home-door`, `x: 240`, `y: 264`, `width: 64`, `height: 24`, target `sunny-town-cookie-keeper-home` at `320,416`.
- Home exit portal: `cookie-keeper-home-exit-door`, `x: 304`, `y: 448`, `width: 64`, `height: 32`, target `sunny-town-v1` at `272,320`.
- Start with a compact interior mirroring `sunny-town-house-1` dimensions and wall/exit shape.
- Do not implement player placement yet, but keep fixture metadata compatible with future placed furniture by using `itemKey: "simple_bed"` and tags `furniture`, `bed`, `sleep`, `rest`, `placeable`.
- Do not add durable HQ home-assignment schema in this issue.

Verification:

- `go test ./internal/sunnytown/maps`
- `go test ./internal/sunnytown/server`

### Issue: Add Configurable NPC Day Cadence

Type: feature
Priority: P1

Blocked by:

- Author Cookie Keeper Home Map And Bed Fixture

Related:

- Epic #38

User story:

As a developer, I want a compressed server-owned NPC day cadence so home/work routines can be reviewed without waiting for real-world clock phases.

Acceptance criteria:

- Sunny Town supports `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`.
- The default/local-play value is `24` real minutes per simulated day when config is absent.
- Dev/demo can set `8` real minutes per simulated day.
- Existing schedule phases remain `morning`, `day`, `evening`, and `night`.
- Existing schedule pressure semantics are preserved: day biases work, night biases home/rest, evening biases social, morning biases food where strong anchors exist.
- `/debug/npcs` still exposes current phase and schedule pressure.
- Clients cannot control NPC simulation time.

Implementation notes:

- Current code uses UTC-hour phase bands in `internal/sunnytown/server/npc_schedule.go`.
- Adapt that helper to read from server/runtime configuration and derive phase from an elapsed compressed day.
- Keep the clock server-owned and deterministic enough for tests.
- Do not persist schedule phase or raw drive values.

Verification:

- `go test ./internal/sunnytown/server`
- Manual or debug smoke showing phase changes under an `8` minute day.

### Issue: Tune Cookie Keeper Home/Work Demo Loop

Type: feature
Priority: P1

Blocked by:

- Author Cookie Keeper Home Map And Bed Fixture
- Add Configurable NPC Day Cadence

Related:

- Epic #38

User story:

As a player, I want Cookie Keeper to go home to rest and then go to work so the town has a readable routine.

Acceptance criteria:

- Cookie Keeper alternates between `cookie-keeper-bed` and `cookie-keeper-counter` on a demo-friendly cadence.
- The NPC routes to the selected home/bed and work locations through portals.
- Home/rest goal chooses the owned bed inside Cookie Keeper's assigned home.
- Work goal chooses the existing `cookie-keeper-counter` anchor.
- The routine avoids rapid goal flipping.
- Server snapshots keep players in sync as the NPC changes maps.
- Cookie Keeper production still requires the NPC to be physically at `cookie-keeper-counter`.
- The implementation does not make clients authoritative over NPC movement or routine choice.

Implementation notes:

- Reuse current drive state, runtime anchors, schedule pressure, focus windows, and failed-target cooldowns.
- Treat the new owned bed location as Cookie Keeper's strong home anchor.
- Keep runtime routine state in Sunny Town memory for the first slice.
- Do not persist raw route/path indexes.

Verification:

- `go test ./internal/sunnytown/server`
- Manual smoke: observe the NPC route home and to work.

### Issue: Improve NPC Routine Debug Output

Type: feature
Priority: P1

Blocked by:

- Tune Cookie Keeper Home/Work Demo Loop

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

GitHub: https://github.com/walt-raymond-williams/sunny-town-hq/issues/55

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

- Should work/sleep state become visible in normal UI, debug-only UI, or just `/debug/npcs` after the first backend demo loop works?
- Should the second authored home go to Teacher, Mayor Sunny, or a future NPC after Cookie Keeper's loop is working?

## Recommended Next Management Step

Create the first child issue:

`Design NPC Home And Work Demo Loop`

Why:

- It locks the first NPC, home, work anchor, and cadence before agents edit maps or routine code.
- It can answer the remaining product questions cheaply.
- It reduces the risk of overbuilding a general AI system before the demo loop is clear.
