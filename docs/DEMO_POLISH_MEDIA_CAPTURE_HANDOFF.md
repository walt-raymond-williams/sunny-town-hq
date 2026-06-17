# Demo Polish Media Capture Handoff

## Purpose

This handoff starts GitHub issue #60, "Capture Sunny Town demo screenshots and routine clip."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/60

The goal is to add polished README-facing Sunny Town media so reviewers can see the strongest demo story before running the app.

## Current Status

- Epic #39 is open and ready for execution.
- Planning issue #59 is closed.
- README/front-door issue #7 is closed.
- #60 is open and `status:ready`.
- Existing committed demo asset:
  - `docs/assets/demo/architecture-overview.svg`
- Work is currently on `codex/demo-polish-planning`; confirm branch strategy before implementation if this branch has not been merged.

## Product Direction

Use the merged Cookie Keeper NPC life/work loop as the visual anchor.

The media should support this README story:

```text
student login
  -> Sunny Town
  -> full-viewport realtime world
  -> hotbar-selected pickaxe, mining, inventory, and progression
  -> Cookie Keeper traveling between home/rest and Cookie Shop work
```

Prefer a few polished, small assets over a large raw recording. The README should stay fast to load and pleasant to review.

## Target Assets

Commit assets under:

```text
docs/assets/demo/
```

Recommended assets:

- `sunny-town-overview.png`
  - Full-viewport Sunny Town game surface.
  - HUD visible.
  - Prefer a useful scene near Cookie Shop or town center.
- `inventory-hotbar-character.png`
  - Inventory/character/progression UI visible.
  - Hotbar-selected active tool context visible if practical.
- `cookie-keeper-routine-poster.png`
  - Cookie Keeper visible with a routine cue.
  - Use this if GIF/video is too large.
- Optional `cookie-keeper-routine.gif`
  - Short, optimized clip only if file size is reasonable.

Do not commit:

- generated `web/` assets
- local logs
- raw screen recordings
- large binary exports

If a routine clip is too large, commit a poster screenshot and document the external video/link decision in #60.

## Current Code And Doc Surface

Primary:

- `README.md`
- `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`
- `docs/DEMO_POLISH_PORTFOLIO_ROADMAP.md`
- `docs/assets/demo/`

Runtime:

- `deploy/docker-compose.yml`
- `docs/current/RUNTIME.md`
- `docs/PROJECT_SETUP.md`

Useful app areas to capture:

- Sunny Town full-viewport canvas.
- Inventory and hotbar.
- Character/progression panel.
- Cookie Keeper routine cues:
  - `>` traveling
  - `W` working
  - `Z` resting
  - `!` blocked

## Recommended Capture Plan

1. Confirm the branch is current and local generated noise is not staged.
2. Start or verify the Docker Compose runtime:

   ```powershell
   docker compose -f deploy/docker-compose.yml config
   docker compose -f deploy/docker-compose.yml up -d --build
   Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
   Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
   ```

3. Open HQ through the same host used by Keycloak issuer.
   - Default local path: `http://localhost:18080`.
   - If using a LAN IP, set `HQ_PUBLIC_HOST` before Compose startup and open that same host.
4. Log in as:

   ```text
   playwright-student / playwright
   ```

5. Enter Sunny Town.
6. Capture a clean overview screenshot.
7. Open inventory/character/progression and capture the UI screenshot.
8. Observe Cookie Keeper's routine cue and capture a poster screenshot or short optimized clip.
9. Update `README.md` to reference the committed media.
10. Add a short issue comment describing exact host/account used and what was captured.

## Practical Notes

- Imported demo users exist only when the Keycloak realm import initializes a fresh Keycloak database volume. If the local volume predates those users, either create them manually or record the existing-volume caveat.
- Keep screenshots focused. Avoid capturing error toasts, browser chrome, devtools, local file paths, or personal account data.
- Use stable viewport sizes where possible so README assets feel intentional.
- Prefer PNG screenshots for committed assets. Optimize GIFs aggressively if used.

## Out Of Scope

- README architecture rewrite; #7 already handled that.
- Frontend bundle warning cleanup; #5 owns that.
- Playwright/demo automation.
- Runtime reset automation.
- New game/UI features just to stage a screenshot.

## Verification

Run:

```powershell
git diff --check
docker compose -f deploy/docker-compose.yml config
```

Manual verification:

- README image links render locally.
- Captured images show current implemented behavior.
- Assets are under `docs/assets/demo/`.
- No generated `web/` files, local logs, or raw recordings are staged.

If services are rebuilt:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected health response for both services: `ok`.

## Close Criteria

Close #60 when:

- At least one polished Sunny Town overview screenshot is committed.
- A second visual asset is committed, either inventory/hotbar/character UI or Cookie Keeper routine poster/clip.
- README references the committed media.
- Verification is recorded on #60.
- Any skipped media item has a clear follow-up or explicit no-asset decision.
