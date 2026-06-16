# Demo Polish Portfolio Roadmap Handoff

## Purpose

This handoff starts GitHub issue #59, "Plan demo polish and portfolio readiness roadmap."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/59

The goal is to turn epic #39 into a concrete demo/portfolio roadmap with child issues that make the repository easy to understand, run, review, and demo.

## Current Status

- Epic #39 is open and `status:needs-design`.
- First NPC life/work demo PR #58 is merged into `main`.
- Existing related issues:
  - #7 Add README architecture highlights and demo assets
  - #5 Reduce frontend production bundle warnings
- Work should start from `main`.
- Suggested planning branch: `codex/demo-polish-planning`.

## Product Direction

The next demo-polish work should help a reviewer quickly answer:

- What is this product?
- What is the most impressive demo path?
- How do I run it locally?
- What architecture decisions are worth noticing?
- What evidence shows this was built with disciplined AI-assisted engineering?

The recently merged NPC life/work loop is now the strongest visual demo story:

- Sunny Town has a Cookie Keeper NPC with a home/work routine.
- The NPC has a home interior, bed fixture, work anchor, visual routine cues, and debug inspectability.
- Sunny Town gameplay remains server-authoritative, including hotbar-selected pickaxe mining with ownership validation.

Use that story as a likely anchor for the README/demo flow unless the audit finds a better path.

## Current Code And Doc Surface

Primary docs to audit:

- `README.md`
- `docs/EPIC_BACKLOG.md`
- `docs/current/ARCHITECTURE.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/RUNTIME.md`
- `docs/PROJECT_SETUP.md`
- `docs/PLAYWRIGHT_E2E_TEST_PLAN.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`

Related issues:

- #39 Epic: Demo polish and portfolio readiness
- #7 Add README architecture highlights and demo assets
- #5 Reduce frontend production bundle warnings

Useful recently merged context:

- PR #58 Add first NPC life/work demo loop
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/STATS_SKILLS_PROGRESSION.md`

## Expected Deliverables

Create one or more planning docs under `docs/`, likely:

- `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`
- `docs/DEMO_POLISH_PORTFOLIO_ROADMAP.md`

The docs should cover:

- current README/demo gaps
- recommended demo path
- screenshot/GIF needs and storage location
- README architecture-highlight outline
- runtime caveats that should be prominent
- whether #7 and #5 are still correctly scoped
- child issue drafts with acceptance criteria, implementation notes, and verification
- recommended order of execution

Update `docs/EPIC_BACKLOG.md` if the recommended next step for #39 changes after discovery.

## Recommended Implementation Plan

1. Read #39, #7, and #5.
2. Audit README and current-state docs for demo clarity and stale instructions.
3. Review the merged NPC life/work tracking doc to identify the best visual demo story.
4. Define one short happy-path demo script.
5. Decide which assets are worth creating:
   - screenshots
   - short GIF or video
   - architecture diagram
   - PR/review checklist
6. Draft child issues.
7. Comment on #59 with the plan summary, commit hash, and next recommended child issue.

## Out Of Scope

- Rewriting the README in full.
- Capturing final screenshots/GIFs.
- Fixing Vite bundle warnings.
- Building new product features.
- Reworking current architecture docs beyond tiny accuracy fixes.

Those should become child issues unless the changes are trivial.

## Verification

Planning-only verification:

```powershell
git diff --check
```

If runtime instructions are changed or challenged, verify:

```powershell
docker compose -f deploy/docker-compose.yml config
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected health response for both services: `ok`.

## Close Criteria

Close #59 when:

- Demo polish discovery/roadmap docs exist.
- Existing #7 and #5 are reused, revised, or explicitly superseded by the roadmap.
- Child issue drafts are ready to create or already created.
- The next recommended executable issue is clear.
- Work is committed and pushed.
- #39 has a comment linking the new docs and next step.
