# Godot Sunny Town Architecture Decision Handoff

## Purpose

This handoff starts GitHub issue #62, "Decide Godot web integration architecture for Sunny Town."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/62

The goal is to capture the architecture decision that implementation agents should follow before adding Godot project files.

## Current Status

- Epic #61 is open and ready.
- Issue #62 is the first child issue and is not blocked.
- Child issues #63 through #71 are intentionally blocked on this decision chain.
- This planning pass created the roadmap and tracking docs but did not implement code.

## Product Direction

- Replace the current Vue/canvas Sunny Town RPG surface with a Godot web client.
- Preserve HQ durable ownership and Sunny Town realtime authority.
- Keep this epic web-only for desktop browser and phone/mobile browser.
- Keep the current canvas client as fallback/debug until Godot passes parity.
- Prefer behavior parity and compatibility before visual polish.

## Current Code Surface

Docs:

- `README.md`
  - Front-door runtime, demo, and ownership summary.
- `docs/SUNNY_TOWN_ARCHITECTURE.md`
  - Current Sunny Town session, WebSocket, maps, movement, inventory, interaction, and service-boundary model.
- `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`
  - Implemented movement authority model.
- `docs/current/API.md`
  - Public and internal HQ/Sunny Town API surface.
- `docs/current/RUNTIME.md`
  - Docker Compose runtime and LAN phone testing workflow.
- `docs/GODOT_SUNNY_TOWN_ROADMAP.md`
  - Approved staged migration plan.
- `docs/GODOT_SUNNY_TOWN_TRACKING.md`
  - Current issue order and next task.

Frontend:

- `frontend/src/router.ts`
  - Current Sunny Town route path.
- `frontend/src/App.vue`
  - Full-viewport Sunny Town container class.
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Current orchestration for session, socket, canvas, input, interactions, inventory, shop, schoolwork, and chests.
- `frontend/src/api/sunnyTownApi.ts`
  - Existing HQ session API client.
- `frontend/src/types/sunnyTown.ts`
  - Current protocol/client-side DTOs.

Backend:

- `cmd/hq/handlers.go`
  - `POST /api/student/sunny-town/session`.
- `internal/sunnytown/protocol/protocol.go`
  - WebSocket message shapes.
- `internal/sunnytown/server/*`
  - Live world state, movement, snapshots, interactions, workers.
- `internal/sunnytownauth/token.go`
  - Join token signing/validation.

## Recommended Implementation Plan

1. Re-read the roadmap, current Sunny Town docs, and the current Vue route/client surfaces.
2. Decide the Godot project source location, for example `godot/sunny-town/` or `sunny-town-godot/`.
3. Decide the Godot export output location and generated-file policy.
4. Decide the Vue wrapper shape:
   - existing route mode switch, or
   - new Godot route plus current canvas fallback route.
5. Decide the session handoff mechanism:
   - recommended first pass: Vue obtains session and exposes it to Godot through JavaScript bridge or page global before Godot starts.
6. Decide first-pass UI ownership:
   - Godot owns in-game surface, movement, prompts, touch controls, HUD/hotbar.
   - Vue may temporarily own complex shop, schoolwork, chest, or inventory overlays through a bridge.
7. Document mobile browser constraints:
   - single-threaded Godot web export first,
   - WebGL2/Compatibility renderer,
   - safe-area/orientation handling,
   - LAN phone smoke target.
8. Update `docs/GODOT_SUNNY_TOWN_ROADMAP.md` if decisions change from the current recommendation.
9. Update `docs/GODOT_SUNNY_TOWN_TRACKING.md` to mark #62 complete-ready and #63 next once implementation lands.

## Out Of Scope

- Adding Godot project files.
- Exporting a real web build.
- WebSocket implementation.
- Backend protocol changes.
- Removing the current canvas client.

## Verification

Run:

```powershell
git diff --check
```

If docs reference runtime commands, spot-check them against:

```powershell
docker compose -f deploy\docker-compose.yml config
```

## Close Criteria

Close #62 when:

- The architecture decision is documented locally.
- The decision covers project location, export artifact handling, Vue wrapper, session handoff, fallback route, mobile constraints, and UI ownership.
- The tracking doc identifies #63 as the next implementation issue.
- Work is committed and pushed to the chosen branch.
- The issue close comment records commit hash and verification results.
