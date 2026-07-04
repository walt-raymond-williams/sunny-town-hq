# Godot Sunny Town Tracking

## Purpose

This document preserves the current GitHub/task workflow state for compaction recovery and fresh-agent handoff. GitHub issues remain the execution source of truth; this file is a local memory snapshot.

## Integration Branch

Recommended feature integration branch:

```text
codex/godot-sunny-town-dev
```

Current status: branch exists and tracks `origin/codex/godot-sunny-town-dev`.

Workflow:

- Create or switch to `codex/godot-sunny-town-dev` before implementation begins.
- Agents work child issues against the integration branch.
- Issues may be closed once their work lands on the integration branch and verification is recorded.
- Docker Desktop is not required for #64-#70 while the Godot web client is still on the integration branch, as long as equivalent non-Docker verification is recorded and each issue comment explicitly says Docker runtime smoke was deferred.
- #71 is the hard Docker Desktop cutover gate. Do not close #71 until Docker Desktop is running, the Compose runtime has been rebuilt, both health endpoints return `ok`, and the Docker-served Godot route plus canvas fallback have been smoke-tested.
- `main` remains stable until the integrated Godot client slice is ready for project-owner review.
- A human should review the integrated feature before merging the integration branch into `main`.

## Management Workflow Loop

Use this loop when advancing the Godot Sunny Town backlog:

1. Check live GitHub issue state with `gh issue list` and inspect the last completed issue comments.
2. Confirm the previous ticket is closed with a commit hash and verification notes.
3. Pick the next roadmap item based on blockers, dependencies, and the current integrated branch state.
4. Create the GitHub issue if it does not already exist.
5. Create or update a focused handoff doc under `docs/` for the selected issue.
6. Update this tracking file with the closed/open issue snapshot, recommended next task, and handoff path.
7. Commit and push only the relevant docs/code for the current issue.
8. Comment on the GitHub issue with durable decisions, blockers, verification, and commit hash.
9. Close the issue only after implementation lands and verification is recorded.
10. Repeat from step 1.

## Current GitHub Issues

Snapshot refreshed: 2026-07-04 after implementing the #63 Godot skeleton on the integration branch.

### Open

- #61 Epic: Godot Web Client for Sunny Town RPG Surface
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/61
  Status: `status:ready`

- #64 Wire Godot web export into HQ build/runtime
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/64
  Status: `status:ready`; next recommended task
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_BUILD_RUNTIME_HANDOFF.md`

- #65 Connect Godot client to Sunny Town session and WebSocket
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/65
  Status: `status:blocked`
  Blocked by: #64
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_SESSION_WS_HANDOFF.md`

- #66 Render current Sunny Town maps and world snapshots in Godot
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/66
  Status: `status:blocked`
  Blocked by: #65
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_RENDERING_PARITY_HANDOFF.md`

- #67 Implement Godot movement, prediction, and remote interpolation
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/67
  Status: `status:blocked`
  Blocked by: #66
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_MOVEMENT_HANDOFF.md`

- #68 Add phone browser controls for Godot Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/68
  Status: `status:blocked`
  Blocked by: #67
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_MOBILE_CONTROLS_HANDOFF.md`

- #69 Port Sunny Town interactions to Godot with Vue overlay bridge
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/69
  Status: `status:blocked`
  Blocked by: #68
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_INTERACTIONS_HANDOFF.md`

- #70 Add Godot HUD, hotbar, inventory summary, and equipment visuals
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/70
  Status: `status:blocked`
  Blocked by: #69
  Docker note: Docker runtime smoke may be deferred to #71 if Docker Desktop is unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_HUD_INVENTORY_HANDOFF.md`

- #71 Prepare Godot Sunny Town cutover and fallback validation
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/71
  Status: `status:blocked`
  Blocked by: #70
  Docker note: Docker Desktop is required before this issue can close.
  Handoff: `docs/GODOT_SUNNY_TOWN_CUTOVER_HANDOFF.md`

### Closed / Complete On Integration Branch

- #62 Decide Godot web integration architecture for Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/62
  Status: closed and complete on `codex/godot-sunny-town-dev`
  Decision: `docs/GODOT_SUNNY_TOWN_ROADMAP.md` section "Architecture Decision (#62)"
  Handoff: `docs/GODOT_SUNNY_TOWN_ARCHITECTURE_DECISION_HANDOFF.md`

- #63 Add minimal Godot web client skeleton for Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/63
  Status: complete on `codex/godot-sunny-town-dev`
  Completed:
  - Added minimal Godot source project at `godot/sunny-town/`.
  - Added placeholder scene, script, icon, README, and Web export preset targeting ignored `web/godot/sunny-town/index.html`.
  - Added Vue Godot wrapper route at `/student/pet/sunny-town/godot`.
  - Added bookmarkable canvas fallback/debug route at `/student/pet/sunny-town/canvas`; `/student/pet/sunny-town` remains the current canvas client.
  - Added export and smoke documentation in `docs/GODOT_SUNNY_TOWN_SKELETON.md`.
  Verification:
  - `cd frontend; npm run build` passed.
  - `git diff --check` passed.
  - Godot 4.7 stable Web export passed with `godot --headless --path godot\sunny-town --export-release "Sunny Town Web" ..\..\web\godot\sunny-town\index.html`.
  - Vite production-preview/Playwright smoke with a temporary student token confirmed the Godot route embeds the exported canvas and the canvas fallback route mounts the existing canvas client.
  - Docker runtime smoke was not run because Docker Desktop's Linux engine pipe was unavailable.
  Handoff: `docs/GODOT_SUNNY_TOWN_SKELETON_HANDOFF.md`

## Recommended Next Management Step

After #63 closes, start issue #64:

```text
Wire Godot web export into HQ build/runtime
```

Why:

- #63 added the source skeleton, wrapper route, and canvas fallback route.
- #64 should make the Godot Web export repeatable in local and Docker-served HQ runtime.
- WebSocket, rendering, movement, mobile controls, interaction, HUD, and cutover work remains blocked until the build/runtime path is wired.

Handoff:

```text
docs/GODOT_SUNNY_TOWN_BUILD_RUNTIME_HANDOFF.md
```

## Useful Commands

```powershell
gh issue list --limit 30
gh issue view 61 --comments
gh issue view 62 --comments
git status --short
```

If `gh` is not on PATH:

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view 62 --comments
```

## Related Local Docs

- `AGENTS.md`
- `README.md`
- `docs/EPIC_BACKLOG.md`
- `docs/GODOT_SUNNY_TOWN_ROADMAP.md`
- `docs/SUNNY_TOWN_ARCHITECTURE.md`
- `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/API.md`
- `docs/current/CONTAINER_STORAGE.md`
- `docs/current/RUNTIME.md`
- `docs/current/STATS_SKILLS_PROGRESSION.md`
