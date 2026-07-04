# Epic Backlog

## Purpose

This file lists active product epics and the workflow for turning them into focused GitHub issues. GitHub issues remain the execution source of truth; this document is a small map for humans and agents choosing the next large body of work.

## Epic Workflow

Use GitHub issues with the `type:epic` label for broad product outcomes.

Epic issues should stay high-level until discovery is complete. Do not assign a coding agent to implement an epic directly. Instead:

1. Pick one epic issue.
2. Create a discovery doc under `docs/`.
3. Audit current code and `docs/current/`.
4. Create a roadmap/ticket doc with child issues, dependencies, blockers, acceptance criteria, and verification.
5. Create child GitHub issues with `type:feature`, `type:bug`, `type:docs`, `type:research`, or `type:tech-debt`.
6. Create a feature integration branch if the epic needs multiple dependent issues.
7. Use a tracking doc and per-issue handoff docs while implementation is active.
8. Move durable architecture decisions into `docs/current/`.
9. Archive or delete completed planning/handoff docs when the epic closes.

## Active Epics

### #38 Epic: NPC Life And Work Simulation

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/38

Goal: make Sunny Town NPCs visibly follow understandable routines and production behavior that responds to world state, storage, needs, and blocked conditions.

Best first step: discovery against current NPC movement/pathing/drive/job-production docs and code.

Discovery docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`

### #39 Epic: Demo Polish And Portfolio Readiness

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/39

Goal: make the project easy to understand, run, review, and demo from the repository front door.

Discovery and roadmap:

- `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`
- `docs/DEMO_POLISH_PORTFOLIO_ROADMAP.md`

Best next step: issue #7, `Add README architecture highlights and demo assets`, should use the roadmap to add the front-door demo path, first media assets, architecture highlights, demo-user notes, and local-network caveats. Keep issue #5, `Reduce frontend production bundle warnings`, as a follow-up demo polish task unless build warnings block README/media verification.

### #40 Epic: Player And NPC Progression

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/40

Goal: expand activity-driven progression beyond the first mining XP slice, including skills/stats that can influence recipes, tools, jobs, and actions.

Best first step: choose the next vertical slice after mining XP without broad balance work.

### #41 Epic: Teacher/Student Learning Loop

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/41

Goal: connect assignments, feedback, AI grading, rewards, and Sunny Town progression into a coherent product flow.

Best first step: discovery across assignment, grading, pet, inventory, reward, and progression systems.

### #61 Epic: Godot Web Client For Sunny Town RPG Surface

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/61

Goal: replace the prototype Vue/canvas Sunny Town RPG surface with a Godot-powered web client embedded in the existing HQ web app, while preserving HQ durable state ownership, Sunny Town realtime authority, the existing session/WebSocket model where practical, and the documentation-first GitHub issue workflow.

Roadmap and tracking:

- `docs/GODOT_SUNNY_TOWN_ROADMAP.md`
- `docs/GODOT_SUNNY_TOWN_TRACKING.md`

Child issues:

- #62 Decide Godot web integration architecture for Sunny Town
- #63 Add minimal Godot web client skeleton for Sunny Town
- #64 Wire Godot web export into HQ build/runtime
- #65 Connect Godot client to Sunny Town session and WebSocket
- #66 Render current Sunny Town maps and world snapshots in Godot
- #67 Implement Godot movement, prediction, and remote interpolation
- #68 Add phone browser controls for Godot Sunny Town
- #69 Port Sunny Town interactions to Godot with Vue overlay bridge
- #70 Add Godot HUD, hotbar, inventory summary, and equipment visuals
- #71 Prepare Godot Sunny Town cutover and fallback validation

Best first step: start #62 using `docs/GODOT_SUNNY_TOWN_ARCHITECTURE_DECISION_HANDOFF.md`. Do not start implementation until #62 records the architecture decision for the Godot project location, export artifact handling, Vue wrapper, session handoff, fallback route, mobile constraints, and first-pass Godot/Vue UI ownership.

## Prioritization Note

Recommended next epic: #61, Godot Web Client For Sunny Town RPG Surface.

Why:

- The project owner approved this plan and asked to create the GitHub issue sequence, roadmap, tracking doc, and handoffs.
- The current canvas RPG surface is intentionally a prototype/debug renderer.
- The backend/live-game foundation is already strong enough to support a client replacement without a backend rewrite.
- The first child issue is planning-only and safely establishes the integration architecture before implementation starts.
