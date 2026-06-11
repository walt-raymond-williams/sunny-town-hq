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

Snapshot refreshed: 2026-06-11 after creating #48.

### Epic

- #38 Epic: NPC life and work simulation
  Status: `status:needs-design`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

### Open

- #48 Design NPC home and work demo loop
  Status: `status:ready`
  Handoff: `docs/NPC_LIFE_WORK_DEMO_LOOP_DESIGN_HANDOFF.md`
  https://github.com/walt-raymond-williams/sunny-town-hq/issues/48

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

Start #48:

`Design NPC home and work demo loop`

Why:

- It chooses the exact home entrance, map IDs, portal IDs, bed metadata, and cadence/config shape before implementation agents edit maps or routine code.
- It keeps the first implementation slice small and visible.
- It prevents the first coding agent from silently inventing product decisions.

## Useful Commands

```powershell
gh issue view 38 --comments
gh issue view 48 --comments
git checkout codex/npc-life-work-dev
git pull
task verify
```

## Related Local Docs

- `docs/EPIC_BACKLOG.md`
- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_DEMO_LOOP_DESIGN_HANDOFF.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/NPC_CHARACTER_MODEL_PLAN.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
