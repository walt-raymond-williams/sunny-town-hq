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

### #39 Epic: Demo Polish And Portfolio Readiness

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/39

Goal: make the project easy to understand, run, review, and demo from the repository front door.

Best first step: consolidate existing demo-polish issues into a concrete demo path and asset plan.

### #40 Epic: Player And NPC Progression

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/40

Goal: expand activity-driven progression beyond the first mining XP slice, including skills/stats that can influence recipes, tools, jobs, and actions.

Best first step: choose the next vertical slice after mining XP without broad balance work.

### #41 Epic: Teacher/Student Learning Loop

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/41

Goal: connect assignments, feedback, AI grading, rewards, and Sunny Town progression into a coherent product flow.

Best first step: discovery across assignment, grading, pet, inventory, reward, and progression systems.

## Prioritization Note

Recommended next epic: #38, NPC Life And Work Simulation.

Why:

- It builds directly on the inventory, storage, Cookie Shop, NPC movement, and production work already in flight.
- It creates visible behavior quickly.
- It gives the project a strong demo story: a town with agents whose routines respond to real state.
