# Demo Polish And Portfolio Discovery

Issue: #59
Epic: #39
Date: 2026-06-16

## Goal

Make the repository front door answer four reviewer questions quickly:

- What is HQ?
- What is the best thing to demo?
- How do I run it from a fresh checkout?
- What architecture choices are worth noticing?

This is a planning pass only. README rewriting, media capture, bundle cleanup, and new automation should happen in child issues.

## Audit Summary

`README.md` is accurate as a setup and architecture orientation document, but it is not yet a portfolio demo guide. It describes the product, stack, repository shape, Docker Compose startup, local-network caveats, health checks, and service boundaries. It does not yet provide a short demo script, screenshots/GIFs, a reviewer-focused architecture highlight section, or a clear "what to notice" flow.

`docs/current/` captures the implemented architecture well. The strongest current-state material for a reviewer is:

- HQ owns durable account, assignment, pet, wallet, inventory, equipment, map-object, NPC production, and progression state.
- Sunny Town owns realtime connected players, movement, map membership, portals, resource-node state, and server-side gameplay validation.
- Browser and WebSocket messages are requests, not authority.
- Durable effects cross service boundaries through service-authenticated internal HQ endpoints.
- Important gameplay mutations are idempotent through ledgers.

`docs/PROJECT_SETUP.md` and `docs/current/RUNTIME.md` have practical runtime instructions, including the local-network host caveat. They are useful support docs, but the README needs a shorter demo path that links to them instead of forcing a reviewer to infer the flow.

`deploy/keycloak/hq-realm.json` already imports two useful local users:

- `playwright-student` / `playwright`
- `playwright-teacher` / `playwright`

The README currently tells users how to create accounts manually, but does not mention these imported users. For demo polish, the README should distinguish "quick local demo users" from "create your own real local users."

`docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md` confirms the first NPC life/work demo loop is merged into `main`. That is currently the highest-signal visual demo: Cookie Keeper has a home/work routine, home interior, bed fixture, work anchor, routine cues, debug inspectability, and durable Cookie Shop production/storage context.

Issue #7 already covers README architecture highlights and demo assets. It is still correctly scoped and should become the first executable child issue after this planning ticket.

Issue #5 already covers frontend production bundle warnings. It is still valid, but it should follow README/media polish unless build warnings become embarrassing during asset capture or local demo startup.

## Current Demo Gaps

- The README starts with product and setup, not a guided demo path.
- No screenshots, GIFs, or video clips are committed or linked.
- No committed media storage location is defined.
- The imported Keycloak demo users are not surfaced in the README.
- The strongest visual story, Cookie Keeper's routine, is not described from the repo front door.
- The README does not explain what architecture details a reviewer should notice.
- The local-network host caveat is present, but it is easy to miss until login fails.
- Vite build warnings are known and tracked, but still make production build output look less clean.
- AI-assisted engineering process is documented in scattered planning/harness docs, but not framed concisely in the README.

## Recommended Demo Path

Use a 6-8 minute reviewer path anchored on the merged Sunny Town/NPC work:

1. Start the full stack:

   ```powershell
   docker compose -f deploy/docker-compose.yml up -d --build
   ```

2. Health-check HQ and Sunny Town:

   ```powershell
   Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
   Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
   ```

3. Open `http://127.0.0.1:18080`.
4. Log in as `playwright-student` with password `playwright`.
5. Enter Sunny Town from the student UI.
6. Show the full-viewport game surface, hotbar, inventory panel, character/equipment/progression panel, and mining with the hotbar-selected pickaxe.
7. Visit the Cookie Shop/home area and watch Cookie Keeper's routine cue:
   - `>` traveling
   - `W` working
   - `Z` resting
   - `!` blocked
8. Open or describe the debug endpoint for routine inspectability:

   ```powershell
   Invoke-WebRequest `
     -UseBasicParsing `
     -Headers @{ "X-HQ-Service-Secret" = "local-dev-service-secret" } `
     http://127.0.0.1:18082/debug/npcs
   ```

9. Briefly switch to `playwright-teacher` if showing the learning app context matters: create or review an assignment, then return to the student/Sunny Town story.

The demo should keep the schoolwork/teacher flow secondary. It establishes product context, but the town simulation is the most visually distinctive portfolio story right now.

## Media Plan

Preferred committed location:

```text
docs/assets/demo/
```

Recommended first assets:

- `sunny-town-overview.png`: full-viewport Sunny Town with HUD visible.
- `cookie-keeper-routine.gif` or `cookie-keeper-routine.mp4`: Cookie Keeper moving/working/resting with routine cue visible.
- `inventory-hotbar-character.png`: slotted inventory, active hotbar tool, character panel, and mining/progression context.
- `architecture-overview.png`: small diagram for README showing Browser, HQ, Sunny Town, AI, PostgreSQL, and Keycloak.

Commit small PNGs and a short compressed GIF only if file size stays reasonable. If the routine clip is large, store a poster screenshot in the repo and link to an external video from the README.

Do not commit generated `web/` assets, local logs, raw screen recordings, or huge media exports.

## README Architecture Highlight Outline

Add a compact section after the feature list or before Quick Start:

- `Local-network product`: laptop-hosted app for teacher/student workflows and realtime play.
- `Service boundary`: HQ owns durable state; Sunny Town owns realtime world state.
- `Server-authoritative gameplay`: clients request movement/mining/containers; servers validate position, ownership, tools, and access.
- `Idempotent durable effects`: rewards, resources, stock production, and skill XP use ledgers to tolerate retries.
- `Reproducible runtime`: Docker Compose starts HQ, Sunny Town, AI fake provider, PostgreSQL, and Keycloak.
- `AI-assisted engineering`: docs, issues, handoffs, review notes, and verification commands show how agents were used with explicit human/product control.

Keep it short. The README should invite deeper reading by linking to `docs/current/ARCHITECTURE.md`, `docs/current/API.md`, `docs/current/DATABASE.md`, and `docs/current/RUNTIME.md`.

## Runtime Caveats To Make Prominent

- Use one consistent host for HQ and Keycloak. If the app is opened through a LAN IP, set `HQ_PUBLIC_HOST` before Compose startup.
- `localhost` on a phone/tablet means the device, not the laptop.
- The Keycloak realm import only runs automatically for a fresh Keycloak database volume.
- The imported demo users are useful for local review, but a persistent local Keycloak database may not pick up changes unless the volume is reset or users are created manually.
- Frontend production assets are generated into ignored `web/`; do not commit them.
- AI grading defaults to the fake provider and should not be presented as a production-ready AI integration.

## Issue Reuse Decision

- Reuse #7 for README architecture highlights and demo assets. It should absorb the demo script, media storage, demo users, architecture highlights, and AI-assisted framing.
- Reuse #5 for frontend production bundle warnings. Keep it as demo polish, but run it after #7 unless #7's frontend build verification makes the warnings a presentation blocker.
- Add one or two new child issues only where #7 and #5 do not fit cleanly: demo smoke verification and optional Playwright/demo-readiness coverage.

## Next Recommendation

Start with #7. It is the shortest path from "good project" to "reviewer understands it quickly."
