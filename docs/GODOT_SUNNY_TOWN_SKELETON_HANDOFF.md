# Godot Sunny Town Skeleton Handoff

## Purpose

This handoff starts GitHub issue #63, "Add minimal Godot web client skeleton for Sunny Town."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/63

The goal is to add the smallest Godot web project and embedded placeholder route without changing gameplay behavior.

## Current Status

- Blocked by #62.
- Start only after the architecture decision names the Godot project location, export output path, wrapper shape, and fallback strategy.
- Current canvas Sunny Town client must remain available.

## Product Direction

- Prove Godot can load inside HQ before connecting it to Sunny Town.
- Keep the first scene intentionally plain: a visible placeholder and load marker are enough.
- Verify desktop and phone browser load early.

## Current Code Surface

Frontend:

- `frontend/src/router.ts`
- `frontend/src/App.vue`
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
- `frontend/src/style.css`

Runtime/docs:

- `deploy/hq/Dockerfile`
- `deploy/docker-compose.yml`
- `docs/current/RUNTIME.md`
- `docs/GODOT_SUNNY_TOWN_ROADMAP.md`

## Recommended Implementation Plan

1. Create the Godot project in the location approved by #62.
2. Add a minimal main scene with visible placeholder text/state.
3. Add a documented web export preset or export instructions.
4. Add a Vue wrapper component or route mode to load the exported placeholder.
5. Preserve a route or feature flag for the current canvas client.
6. Add CSS/layout so the Godot surface fills the Sunny Town play viewport.
7. Document local export and smoke-test steps.

## Out Of Scope

- WebSocket connection.
- Session handoff.
- Map rendering.
- Input/gameplay.
- Polished visual assets.

## Verification

Run:

```powershell
cd frontend
npm run build
```

If practical:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual smoke:

- Open the Godot Sunny Town route on desktop browser.
- Open it on a phone browser using the LAN host workflow.
- Confirm the existing canvas route/fallback still loads.

## Close Criteria

Close #63 when:

- Minimal Godot project and placeholder load path exist.
- Canvas fallback remains available.
- Export/run docs exist.
- Verification results are recorded.
