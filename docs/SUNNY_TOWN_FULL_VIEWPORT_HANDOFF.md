# Sunny Town Full Viewport Handoff

## Purpose

This handoff starts GitHub issue #46, "Make Sunny Town a full-viewport play experience."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/46

The goal is to make Sunny Town fill the browser viewport while a student is playing, with HUD and panels layered over the game surface instead of presenting the game as a framed page section.

## Current Status

- The inventory redesign has landed on `main`.
- Recent Sunny Town bug fixes for inventory-backed actions and NPC portal ghosting have also landed on `main`.
- The current Sunny Town route works, but visually reads as a normal app page: toolbar above a framed stage.
- This is a frontend UX/layout ticket. It should not change gameplay authority, server protocol, inventory APIs, or map data.

## Product Direction

- Sunny Town should feel like a game surface when opened at `/student/pet/sunny-town`.
- The canvas should be the first visual priority and should fill the available browser viewport.
- HUD, status, stars, online count, help, and hotbar should overlay the play surface.
- Inventory, crafting, dialogue, shop, schoolwork, and chest storage should continue opening over the play surface.
- Keep an obvious Back/exit control visible.
- Avoid normal page scroll during play.
- Support narrow viewports without text overflow or incoherent overlap.
- Do not implement the browser Fullscreen API in this ticket. A future optional fullscreen button can be a follow-up after the full-viewport route is solid.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Owns the Sunny Town route, overlay orchestration, input handling, and canvas slot usage.
- `frontend/src/features/sunny-town/SunnyTownHud.vue`
  - Current wrapper for `.sunny-town-page`, toolbar, stage, help text, hotbar, toast, and slotted overlays.
- `frontend/src/features/sunny-town/SunnyTownCanvas.vue`
  - Emits the canvas element and pointer events. It probably does not need logic changes if CSS dimensions are stable.
- `frontend/src/composables/useSunnyTownRenderer.ts`
  - Reads the canvas CSS box with `getBoundingClientRect()` and resizes the backing canvas for device pixel ratio. This should keep working if CSS gives the canvas full-viewport dimensions.
- `frontend/src/style.css`
  - Existing relevant classes include `.sunny-town-page`, `.sunny-town-toolbar`, `.sunny-town-toolbar__actions`, `.sunny-town-stage`, `.sunny-town-stage canvas`, `.sunny-town-help`, and `.sunny-town-hotbar`.

Docs:

- `AGENTS.md`
  - Frontend guidance says build the usable app/tool surface, keep controls compact, avoid landing-page treatment, and verify significant frontend changes.

## Recommended Implementation Plan

1. Update the Sunny Town route layout so `.sunny-town-page` occupies the viewport and prevents normal page scrolling during play.
2. Convert `.sunny-town-stage` into the full-viewport positioning container and make its canvas fill that container.
3. Rework `.sunny-town-toolbar` into a compact overlay rather than a page header. Keep connection status, player count, stars, and Back visible.
4. Keep `.sunny-town-hotbar`, `.sunny-town-help`, and toast anchored over the stage with responsive positioning.
5. Review overlay panels from `SunnyTownPage.vue` in the full-viewport context:
   - inventory/E menu
   - crafting
   - dialogue
   - shop
   - schoolwork
   - chest storage
6. Add or adjust frontend tests only where layout behavior is represented in component/composable state. Most of this ticket will be CSS/manual smoke.
7. Run frontend tests/build and, if practical, rebuild local containers and smoke test the route.

## Out Of Scope

- Browser Fullscreen API.
- Touch controls.
- Reworking Sunny Town input semantics.
- Redesigning inventory, crafting, shops, dialogue, schoolwork, or storage panels beyond making sure they remain usable as overlays.
- Backend, schema, API, or map changes.

## Verification

Run:

```powershell
cd frontend
npm run test
npm run build
```

If practical, rebuild runtime:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual smoke:

- Open Sunny Town as a student.
- Confirm the game fills the browser viewport.
- Confirm normal page scroll does not appear during play.
- Move around with keyboard controls.
- Use hotbar selection.
- Open and close inventory with `E`.
- Open crafting, dialogue, shop, schoolwork, and chest panels if available.
- Mine/place/remove blocks if the test account has the required items.
- Check a narrow viewport for HUD wrapping, clipped text, and overlay collisions.

## Close Criteria

Close #46 when:

- Sunny Town route is full-viewport in normal browser mode.
- HUD and hotbar overlay the game surface and remain usable.
- Existing overlays remain usable.
- Verification results are recorded on the issue.
- Work is committed and pushed.

Use a close comment like:

```text
Completed in <commit>. Sunny Town now uses a full-viewport play layout with overlay HUD/hotbar and existing panels preserved. Verified with: <commands>. Manual smoke: <result or blocker>.
```
