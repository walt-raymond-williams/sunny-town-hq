# Godot Sunny Town Cutover Handoff

## Purpose

This handoff starts GitHub issue #71, "Prepare Godot Sunny Town cutover and fallback validation."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/71

The goal is to make Godot the default Sunny Town RPG surface only after explicit parity and mobile verification.

## Current Status

- Blocked by #70.
- Godot should have session, rendering, movement, mobile controls, interactions, HUD, hotbar, inventory feedback, and equipment visual parity before this starts.

## Product Direction

- Cut over incrementally, not as a big-bang replacement.
- Keep the canvas client as fallback/debug until a later explicit removal decision.
- Update durable current-state docs before closing the epic.

## Current Code Surface

Docs:

- `docs/GODOT_SUNNY_TOWN_ROADMAP.md`
- `docs/GODOT_SUNNY_TOWN_TRACKING.md`
- `docs/SUNNY_TOWN_ARCHITECTURE.md`
- `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/API.md`
- `docs/current/RUNTIME.md`
- `docs/current/CONTAINER_STORAGE.md`

Frontend/runtime:

- Godot wrapper and client files from previous issues.
- Current canvas fallback route/files.
- `frontend/e2e/sunny-town.spec.ts`
- `frontend/e2e/sunny-town-npc.spec.ts`
- `frontend/e2e/support/sunnyTown.ts`
- `deploy/docker-compose.yml`

## Recommended Implementation Plan

1. Define the cutover switch:
   - route default,
   - query flag,
   - env flag,
   - or config flag.
2. Keep and document canvas fallback/debug route.
3. Build the final regression checklist.
4. Add or update automated smoke checks where practical.
5. Run deterministic checks.
6. Run desktop manual parity smoke.
7. Run phone browser manual smoke.
8. Update `docs/current/` for accepted architecture/runtime changes.
9. Update tracking doc with any remaining follow-up issues.
10. Record project-owner cutover decision in the issue.

## Out Of Scope

- Removing the canvas client entirely.
- New gameplay or art polish.
- Native/mobile/desktop app distribution.

## Verification

Run:

```powershell
go test ./...
cd frontend
npm run build
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual desktop smoke:

- Login/session join.
- Godot export loads in HQ.
- Two clients see each other.
- Movement and remote interpolation.
- Map transitions.
- Inventory/hotbar/equipment sync.
- NPC dialogue, shop, and schoolwork.
- Resource/tool use.
- Rewards/ledger behavior.
- Placed object persistence.
- Chest/container interactions.
- Canvas fallback loads.

Manual phone smoke:

- Godot export loads on phone browser.
- Touch movement and actions work.
- Hotbar works.
- Inventory/interaction UI remains usable.
- Exit/back returns to HQ/student UI.

## Close Criteria

Close #71 when:

- Godot default/cutover behavior is implemented or explicitly staged.
- Canvas fallback remains available and documented.
- Regression checklist passes or blockers are recorded.
- Current-state docs are updated.
- Remaining follow-ups are tracked.
