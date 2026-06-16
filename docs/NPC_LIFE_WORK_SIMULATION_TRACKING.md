# NPC Life Work Simulation Tracking

## Purpose

This document preserves the GitHub/task workflow state for epic #38, "Epic: NPC life and work simulation." GitHub issues remain the execution source of truth; this file is a local memory snapshot for fresh-agent handoff.

## Integration Branch

Current feature integration branch:

`codex/npc-life-work-dev`

Workflow:

- Agents work child issues against `codex/npc-life-work-dev`.
- Issues may be closed once their work lands on the integration branch and verification is recorded.
- `main` remains stable until the first NPC life/work slice is coherent and ready for review.
- A human reviews the integrated feature before merging `codex/npc-life-work-dev` into `main`.

## Current GitHub Issues

Snapshot refreshed: 2026-06-16 after PR #58 merged.

Current integration PR:

- #58 Add first NPC life/work demo loop
  Status: merged into `main` on 2026-06-16
  https://github.com/walt-raymond-williams/sunny-town-hq/pull/58

### Epic

- #38 Epic: NPC life and work simulation
  Status: `status:needs-design`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

### Closed

- #48 Design NPC home and work demo loop
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_DEMO_LOOP_DESIGN_HANDOFF.md`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/48

- #49 Author Cookie Keeper home map and bed fixture
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_COOKIE_KEEPER_HOME_HANDOFF.md`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/49

- #50 Add configurable NPC day cadence
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_DAY_CADENCE_HANDOFF.md`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/50

- #51 Tune Cookie Keeper home/work demo loop
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_COOKIE_KEEPER_ROUTINE_HANDOFF.md`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/51

- #52 Improve NPC routine debug output for demo review
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_ROUTINE_DEBUG_HANDOFF.md`
  Related: #4
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/52

- #55 Add optional NPC routine visual cues
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_VISUAL_CUES_HANDOFF.md`
  Related: #38
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/55

- #54 Use selected hotbar item as active Sunny Town tool
  Status: closed
  Handoff: `docs/SUNNY_TOWN_HOTBAR_ACTIVE_TOOL_HANDOFF.md`
  Related blocker fixed before PR readiness review
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/54

- #56 Review NPC life/work integration branch for PR readiness
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_PR_READINESS_HANDOFF.md`
  Recommendation: ready to open PR from `codex/npc-life-work-dev` into `main`; no blocking follow-up issues found.
  Related: #38, #48, #49, #50, #51, #52, #54, #55
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/56

- #57 Open NPC life/work integration PR
  Status: closed
  Handoff: `docs/NPC_LIFE_WORK_OPEN_PR_HANDOFF.md`
  Result: opened PR #58 from `codex/npc-life-work-dev` into `main`.
  Related: #38, #48, #49, #50, #51, #52, #54, #55, #56
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/57

### Open

No open first-demo-loop child issues remain. PR #58 is merged.

### Related Existing Issues

- #4 Define production posture for NPC debug endpoint
  Status: `status:needs-design`
  Related to debug endpoint posture for routine inspection.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/4

- #3 Split dense NPC movement logic by behavior when needed
  Status: `status:ready`
  May become relevant if routine logic expands movement code too much.
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/3

## Recommended Next Management Step

Decide the next epic or child slice. Good candidates:

- #39 Demo polish and portfolio readiness, to turn the merged NPC loop into a clean demo path.
- #40 Player and NPC progression, to build on mining XP and NPC routine/progression hooks.
- A second #38 NPC life/work slice, such as adding another resident/home or expanding routine needs beyond work/rest.

## Useful Commands

```powershell
gh issue view 38 --comments
gh issue view 51 --comments
gh issue view 52 --comments
gh issue view 55 --comments
gh issue view 56 --comments
gh issue view 57 --comments
git checkout main
git pull
task verify
```

## Related Local Docs

- `docs/EPIC_BACKLOG.md`
- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_DEMO_LOOP_DESIGN_HANDOFF.md`
- `docs/NPC_LIFE_WORK_COOKIE_KEEPER_HOME_HANDOFF.md`
- `docs/NPC_LIFE_WORK_DAY_CADENCE_HANDOFF.md`
- `docs/NPC_LIFE_WORK_COOKIE_KEEPER_ROUTINE_HANDOFF.md`
- `docs/NPC_LIFE_WORK_ROUTINE_DEBUG_HANDOFF.md`
- `docs/NPC_LIFE_WORK_VISUAL_CUES_HANDOFF.md`
- `docs/NPC_LIFE_WORK_PR_READINESS_HANDOFF.md`
- `docs/NPC_LIFE_WORK_OPEN_PR_HANDOFF.md`
- `docs/SUNNY_TOWN_HOTBAR_ACTIVE_TOOL_HANDOFF.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/NPC_CHARACTER_MODEL_PLAN.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
