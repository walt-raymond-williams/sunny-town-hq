# Codex GitHub Harness Demo

## Purpose

This walkthrough shows how the project can use GitHub Issues as a queue for Codex agents while keeping humans in the product review loop.

The demo is intentionally process-focused. It can be run manually today and later automated with scripts under `tools/codex-harness/`.

## Demo Scenario

Customer request:

```text
I want the Sunny Town inventory menu to feel more like a game inventory. Crafting should be easier to scan, equipped gear should be obvious, and I should not lose track of what changed after I craft something.
```

## Step 1: Customer Intake

The customer intake agent turns the request into user stories.

Example output:

- As a player, I want the crafting panel to show available recipes clearly so I can decide what to craft quickly.
- As a player, I want equipped gear to be visually distinct so I understand my current loadout.
- As a player, I want crafting results to update inventory feedback immediately so I trust that the action worked.

The intake agent records:

- acceptance criteria
- assumptions
- open questions
- requirements that must not regress

Durable output:

- requirements or discovery doc under `docs/`
- handoff summary for the planning agent

## Step 2: Planning

The planning agent reads:

- `AGENTS.md`
- `docs/CODEX_GITHUB_HARNESS_PLAN.md`
- `docs/GITHUB_TRACKING_DOC_GUIDE.md`
- the intake output
- relevant `docs/current/` files
- live GitHub issues
- current frontend and backend code

The planning agent identifies the implementation delta:

- crafting panel layout and state feedback are frontend work
- equipped gear visual treatment touches existing inventory/equipment UI
- backend crafting correctness must remain stable
- broad changes should stay on the feature integration branch until human review

Then it creates small issues, such as:

- Redesign crafting panel for inventory menu
- Improve equipped gear visual states
- Add crafting result feedback

Each ready issue gets:

- acceptance criteria
- implementation notes
- verification commands
- a linked handoff doc
- `status:ready`
- optional `agent:worker`

## Step 3: Worker Execution

The worker loop finds the next ready issue for the current feature branch:

```powershell
git branch --show-current
gh issue list --label "status:ready" --limit 30
```

The worker selects an issue related to the checked-out branch, tracking doc, handoff docs, or feature labels. `agent:worker` can be used as helpful metadata, but branch-scoped worker loops do not require it.

The worker claims it:

```powershell
gh issue edit <number> --remove-label "status:ready" --add-label "status:in-progress"
```

The worker agent reads:

- `AGENTS.md`
- the GitHub issue and comments
- the linked handoff doc
- relevant current-state docs
- nearby code

The worker implements the issue, runs verification, and records results.

Example closeout comment:

```text
Completed in <commit>. Redesigned the crafting panel layout and preserved existing crafting behavior.

Verified with:
- cd frontend; npm run build

Manual smoke:
- Opened Sunny Town inventory menu.
- Confirmed recipe list, ingredient states, and craft action feedback render correctly.
```

## Step 4: Repeat Until No Ready Work Remains

The harness keeps running the worker loop:

```text
find ready issue
claim issue
start worker agent
wait for completion or blocker
update GitHub
repeat
```

When no `status:ready` issue remains for the current feature branch or slice, the worker loop stops or sleeps.

The planning agent can then inspect the completed work and prepare the next issue if the roadmap still has unblocked work.

## Step 5: Human Review

After a coherent feature slice or epic is complete, a human reviews it end to end.

The human reviewer:

- runs deterministic checks
- opens the app
- tests the customer-facing workflows
- checks for regressions against existing requirements
- decides whether the slice is ready to merge

If the review finds problems, the reviewer creates follow-up issues:

- `type:bug` for defects
- `type:feature` for changed requirements
- `type:docs` for missing current-state documentation
- `status:ready` when a worker can start; `agent:worker` may be added when useful

## What This Demonstrates

This workflow demonstrates:

- customer request intake
- user story generation
- codebase-aware implementation planning
- GitHub Issues as an agent work queue
- handoff docs as fresh-agent memory
- worker agents with narrow task contracts
- deterministic AI verification
- human product review before merging broad work
- feedback loops that create more issues when needed

## Demo Commands

Inspect current issues:

```powershell
gh issue list --limit 30
```

Inspect a specific issue:

```powershell
gh issue view <number> --comments
```

Find ready worker issues:

```powershell
git branch --show-current
gh issue list --label "status:ready"
```

Claim a worker issue:

```powershell
gh issue edit <number> --remove-label "status:ready" --add-label "status:in-progress"
```

Mark blocked:

```powershell
gh issue edit <number> --remove-label "status:in-progress" --add-label "status:blocked"
gh issue comment <number> --body "Blocked: <specific blocker and needed decision>."
```

Move to review:

```powershell
gh issue edit <number> --remove-label "status:in-progress" --add-label "status:review"
gh issue comment <number> --body "Ready for review. Verified with: <commands>."
```
