# Godot Sunny Town Tracking

## Purpose

This document preserves the current GitHub/task workflow state for compaction recovery and fresh-agent handoff. GitHub issues remain the execution source of truth; this file is a local memory snapshot.

## Integration Branch

Recommended feature integration branch:

```text
codex/godot-sunny-town-dev
```

Current status: branch not created by this planning pass.

Workflow:

- Create or switch to `codex/godot-sunny-town-dev` before implementation begins.
- Agents work child issues against the integration branch.
- Issues may be closed once their work lands on the integration branch and verification is recorded.
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

Snapshot refreshed: 2026-07-04 after creating epic #61 and child issues #62 through #71.

### Open

- #61 Epic: Godot Web Client for Sunny Town RPG Surface
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/61
  Status: `status:ready`

- #62 Decide Godot web integration architecture for Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/62
  Status: `status:ready`
  Handoff: `docs/GODOT_SUNNY_TOWN_ARCHITECTURE_DECISION_HANDOFF.md`

- #63 Add minimal Godot web client skeleton for Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/63
  Status: `status:blocked`
  Blocked by: #62
  Handoff: `docs/GODOT_SUNNY_TOWN_SKELETON_HANDOFF.md`

- #64 Wire Godot web export into HQ build/runtime
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/64
  Status: `status:blocked`
  Blocked by: #63
  Handoff: `docs/GODOT_SUNNY_TOWN_BUILD_RUNTIME_HANDOFF.md`

- #65 Connect Godot client to Sunny Town session and WebSocket
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/65
  Status: `status:blocked`
  Blocked by: #64
  Handoff: `docs/GODOT_SUNNY_TOWN_SESSION_WS_HANDOFF.md`

- #66 Render current Sunny Town maps and world snapshots in Godot
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/66
  Status: `status:blocked`
  Blocked by: #65
  Handoff: `docs/GODOT_SUNNY_TOWN_RENDERING_PARITY_HANDOFF.md`

- #67 Implement Godot movement, prediction, and remote interpolation
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/67
  Status: `status:blocked`
  Blocked by: #66
  Handoff: `docs/GODOT_SUNNY_TOWN_MOVEMENT_HANDOFF.md`

- #68 Add phone browser controls for Godot Sunny Town
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/68
  Status: `status:blocked`
  Blocked by: #67
  Handoff: `docs/GODOT_SUNNY_TOWN_MOBILE_CONTROLS_HANDOFF.md`

- #69 Port Sunny Town interactions to Godot with Vue overlay bridge
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/69
  Status: `status:blocked`
  Blocked by: #68
  Handoff: `docs/GODOT_SUNNY_TOWN_INTERACTIONS_HANDOFF.md`

- #70 Add Godot HUD, hotbar, inventory summary, and equipment visuals
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/70
  Status: `status:blocked`
  Blocked by: #69
  Handoff: `docs/GODOT_SUNNY_TOWN_HUD_INVENTORY_HANDOFF.md`

- #71 Prepare Godot Sunny Town cutover and fallback validation
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/71
  Status: `status:blocked`
  Blocked by: #70
  Handoff: `docs/GODOT_SUNNY_TOWN_CUTOVER_HANDOFF.md`

### Closed

No Godot Sunny Town child issues are closed yet.

## Recommended Next Management Step

Start issue #62:

```text
Decide Godot web integration architecture for Sunny Town
```

Why:

- It is the only unblocked child issue.
- It turns this planning pass into an explicit architecture decision future implementation agents can follow.
- It decides project location, export artifact ownership, Vue wrapper shape, session handoff, fallback route, and first mobile/browser constraints before code is added.

Handoff:

```text
docs/GODOT_SUNNY_TOWN_ARCHITECTURE_DECISION_HANDOFF.md
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
