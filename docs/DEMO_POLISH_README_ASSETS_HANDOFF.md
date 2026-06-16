# Demo Polish README Assets Handoff

## Purpose

This handoff starts GitHub issue #7, "Add README architecture highlights and demo assets."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/7

The goal is to make the repository front door explain the product, strongest demo path, and architecture choices quickly.

## Current Status

- Epic #39 is open and ready for execution.
- Planning issue #59 is closed.
- Discovery: `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`
- Roadmap: `docs/DEMO_POLISH_PORTFOLIO_ROADMAP.md`
- #7 is open and `status:ready`.
- Work is currently on `codex/demo-polish-planning`; confirm branch strategy before implementation if this branch has not been merged.

## Product Direction

Use the merged Cookie Keeper NPC life/work loop as the primary visual demo story.

Recommended README story:

```text
HQ learning app context
  -> student login
  -> Sunny Town
  -> full-viewport realtime world
  -> hotbar-selected pickaxe/mining/inventory/progression
  -> Cookie Keeper home/work routine with visual cues and debug inspectability
```

The README should feel like a portfolio front door, not a full manual. Keep it concise and link to `docs/current/` for deep detail.

## Scope

Update `README.md` to include:

- A short "Demo Path" or "What To Demo" section.
- Imported local demo users:
  - `playwright-student` / `playwright`
  - `playwright-teacher` / `playwright`
- A compact architecture highlights section covering:
  - HQ durable state ownership
  - Sunny Town realtime state ownership
  - server-authoritative gameplay validation
  - service-authenticated internal APIs
  - idempotent ledgers for retries
  - Docker Compose reproducible runtime
  - disciplined AI-assisted engineering workflow
- A clearer local-network host consistency note near Quick Start.
- Links to committed demo media under `docs/assets/demo/`, if media is added in this pass.

Add first demo assets if practical:

- `docs/assets/demo/sunny-town-overview.png`
- `docs/assets/demo/inventory-hotbar-character.png`
- Optional small `cookie-keeper-routine.gif` or poster screenshot if a GIF would be too large.

If media capture is too much for this issue, add the folder/README references only when assets exist and create a follow-up issue for capture.

## Current Code And Doc Surface

Primary:

- `README.md`
- `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`
- `docs/DEMO_POLISH_PORTFOLIO_ROADMAP.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/RUNTIME.md`
- `docs/PROJECT_SETUP.md`

Useful supporting docs:

- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`

Runtime/media paths:

- `deploy/docker-compose.yml`
- `docs/assets/demo/`

## Recommended Implementation Plan

1. Read #7, the discovery doc, and the roadmap doc.
2. Draft README additions in small sections:
   - demo path
   - demo users
   - architecture highlights
   - local-network/runtime caveats
3. Keep Quick Start accurate and avoid duplicating all of `docs/PROJECT_SETUP.md`.
4. Add `docs/assets/demo/` only if committing actual media or a small README-facing asset index.
5. Capture or add demo media only if it is small and polished.
6. Verify README links and Docker Compose config.
7. Comment on #7 with summary, verification, and any follow-up issue suggestions.

## Out Of Scope

- Fixing frontend production bundle warnings; #5 owns that.
- Adding Playwright/demo automation.
- Large README rewrite that turns it into a manual.
- New product features or UI changes.
- Committing generated `web/`, local logs, raw recordings, or large binary media.

## Verification

Run:

```powershell
git diff --check
docker compose -f deploy/docker-compose.yml config
```

If runtime instructions change, also verify health checks:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected health response for both services: `ok`.

If demo media is captured through the app, manually confirm:

- HQ loads from the documented host.
- `playwright-student` can reach Sunny Town.
- Sunny Town canvas shows the intended screenshot state.
- Cookie Keeper routine cue is visible if routine media is captured.

## Close Criteria

Close #7 when:

- README includes a concise demo path.
- README surfaces demo users and host/Keycloak caveats.
- README includes architecture highlights for reviewer scanning.
- Demo assets are committed under `docs/assets/demo/` or a follow-up asset-capture issue is created.
- Verification is recorded on #7.
