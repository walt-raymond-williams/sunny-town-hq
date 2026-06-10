# Codex GitHub Harness Plan

## Purpose

This document captures the repeatable agent workflow for turning customer feature requests into GitHub issues, handoff docs, implementation work, verification, and human review.

The goal is to make the current manual two-agent process durable enough that project contributors can run it, inspect it, and demo it:

1. A customer describes the website or app change they want.
2. A planning agent turns that conversation into user stories and requirements.
3. The planning agent audits the current code and docs, then identifies the delta needed without breaking existing requirements.
4. The planning agent creates a roadmap, GitHub issues, and focused handoff docs.
5. Worker agents claim ready GitHub issues, implement them, verify them, and report results.
6. A human reviews completed feature slices or epics and creates follow-up issues for defects, polish, or requirement changes.

GitHub Issues remain the execution source of truth. Local Markdown docs remain the durable memory for roadmap context, architecture rationale, and fresh-agent handoff.

## Current Manual Workflow

The project already uses this pattern manually:

- Roadmap and discovery docs define the feature direction.
- A management agent creates the next issue and a focused handoff doc.
- A worker agent reads `AGENTS.md`, the GitHub issue, the handoff doc, and related current-state docs.
- The worker implements one issue, runs verification, comments or closes the issue, and reports completion.
- The management agent reads the latest issue state and updates the tracking doc before preparing the next issue.

The harness should preserve this workflow. It should automate the loop mechanics, not replace the planning judgment or human review gates.

## Harness Definition

In this project, a harness is the outer control system that starts agents, supplies task context, records results, and decides whether to continue.

The harness should be intentionally simple:

```text
planner:
  read customer input, roadmap, docs, and GitHub state
  produce or update requirements, issues, and handoff docs
  label ready work

worker:
  find the next ready issue related to the current feature branch
  claim it
  run Codex with the issue and handoff context
  verify the result
  update GitHub
  repeat until no ready issue remains
```

## Agent Roles

### Customer Intake Agent

Purpose:

- Convert customer conversation into clear product direction.
- Ask clarifying questions when requirements are ambiguous.
- Produce user stories and acceptance criteria before implementation planning starts.

Inputs:

- Customer request.
- Existing roadmap docs.
- `docs/current/` architecture and runtime docs.
- Existing user stories and requirements.

Outputs:

- Updated requirements or discovery doc.
- Open questions and assumptions.
- Draft user stories grouped into feature slices.

### Planning Agent

Purpose:

- Audit the current codebase and docs.
- Identify the implementation delta needed for the requested feature.
- Split the work into dependency-aware GitHub issues.
- Prepare one focused handoff doc per ready issue.

Inputs:

- Customer intake output.
- Current codebase.
- `AGENTS.md`.
- `docs/GITHUB_TRACKING_DOC_GUIDE.md`.
- Relevant current-state docs.
- Live GitHub issue state.

Outputs:

- Roadmap or tracking doc updates.
- GitHub issues with acceptance criteria and verification commands.
- Handoff docs linked from those issues.
- Ready labels for worker pickup.

### Worker Agent

Purpose:

- Implement exactly one ready issue at a time.
- Keep scope aligned to the issue and handoff doc.
- Verify deterministic checks before claiming completion.
- Record durable results in GitHub.

Inputs:

- Claimed GitHub issue.
- Linked handoff doc.
- `AGENTS.md`.
- Relevant current-state docs.
- Current branch state.

Outputs:

- Code and doc changes.
- Test/build/runtime verification results.
- Issue comments with blockers, discoveries, and completion notes.
- Optional pull request or commit reference, depending on the active branch workflow.

### Human Reviewer

Purpose:

- Review completed epics or coherent feature slices end to end.
- Validate behavior manually as a customer or product owner would.
- Create feedback tickets when defects, UX gaps, or changed requirements appear.

Inputs:

- Integrated feature branch.
- Closed issues and verification notes.
- Roadmap/tracking docs.
- Running app.

Outputs:

- Approval to merge the integration branch.
- Follow-up GitHub issues.
- Updated requirements or roadmap notes when the product direction changes.

## Queue Labels

Use consistent labels so the harness can treat GitHub Issues as a queue.

Recommended labels:

- `status:ready`: issue is ready for a worker.
- `status:in-progress`: issue has been claimed.
- `status:blocked`: issue cannot continue without human input or another dependency.
- `status:review`: implementation is complete and needs review.
- `status:done`: issue work has landed and verification is recorded.
- `agent:planner`: planning-agent issue or task.
- `agent:worker`: worker-agent issue or task.
- `type:feature`: user-visible feature work.
- `type:bug`: defect or regression.
- `type:docs`: documentation-only work.
- `type:harness`: harness workflow, prompts, scripts, or demo material.

The project already uses labels such as `status:ready` and `status:in-progress`; the harness should start with existing labels and add only the missing ones. For feature integration branches, `agent:worker` is optional queue metadata, not a hard requirement. A worker can claim any `status:ready` issue that is clearly related to the currently checked-out branch or feature slice.

## Harness Control Loop

### Planner Loop

1. Read the latest customer request, roadmap, tracking doc, and GitHub issue state.
2. Confirm completed worker issues include commit hashes and verification notes.
3. Update requirements or architecture docs if the accepted current state changed.
4. Select the next small issue based on blockers and feature value.
5. Create or update the GitHub issue.
6. Create or update the handoff doc.
7. Link the handoff doc from the issue.
8. Add `status:ready` when the worker can start.

### Worker Loop

1. Determine the currently checked-out branch and feature context from the branch name, tracking docs, linked handoff docs, issue titles, labels, and recent issue history.
2. Find the oldest or highest-priority issue labeled `status:ready` that is related to that branch or feature slice. Prefer `agent:worker` when present, but do not require it for branch-scoped feature work.
3. Claim it by replacing `status:ready` with `status:in-progress`.
4. Read `AGENTS.md`, the issue, linked handoff doc, and relevant docs.
5. Implement the smallest change that satisfies the issue.
6. Run the verification commands from the issue or handoff.
7. Comment results and either:
   - close the issue or move it to `status:review` when work is complete
   - move it to `status:blocked` with a clear blocker
8. Repeat until no ready issue remains for the current branch or feature slice.

## Safety Rules

- One worker should claim one issue at a time.
- The harness must avoid two workers claiming the same issue. Use label changes, assignees, or a short lock file.
- Workers should start each issue from a clean or understood branch state.
- Workers should not require `agent:worker` when the feature branch, tracking doc, handoff doc, or issue metadata already makes the issue's branch relationship clear.
- Workers should not infer the roadmap. The planner must convert roadmap state into an issue contract.
- Workers should not broaden scope when new work appears. They should create or request follow-up issues.
- Human review remains required before merging broad feature branches into `main`.
- Compaction is useful recovery context, but the durable state must live in GitHub issues and Markdown docs.

## Suggested Repository Shape

Initial documentation-only slice:

```text
docs/CODEX_GITHUB_HARNESS_PLAN.md
docs/CODEX_GITHUB_HARNESS_DEMO.md
docs/CODEX_GITHUB_HARNESS_PROMPTS.md
```

Future script slice:

```text
tools/codex-harness/
  README.md
  planner.ps1
  worker.ps1
  prompts/
    planner.md
    worker.md
  state/
    .gitkeep
```

The first implementation should be PowerShell-friendly because the project workflow and documented commands already target Windows.

## MVP Scope

The first usable harness should support:

- Listing ready worker issues.
- Claiming one issue by updating labels.
- Printing or launching the exact Codex prompt for that issue.
- Recording completion, blocker, or review status back to GitHub.
- Running until no ready issue remains.

The MVP does not need to solve:

- Fully unattended merging.
- Automatic conflict resolution.
- Automatic human review.
- Multiple concurrent workers.
- Cross-repository orchestration.
- A custom web UI.

## GitHub-Ready Implementation Tickets

### Ticket 1: Document the Codex GitHub harness workflow

User story:

As a project maintainer, I want the current customer-to-planner-to-worker workflow documented so that contributors and demo viewers understand how the harness works.

Acceptance criteria:

- A harness plan exists under `docs/`.
- The plan defines customer intake, planning, worker, and human review roles.
- The plan defines GitHub label conventions and the issue queue loop.
- The plan explains how this builds on the existing tracking and handoff docs.

Verification:

```powershell
git diff -- docs/CODEX_GITHUB_HARNESS_PLAN.md
```

### Ticket 2: Add reusable planner and worker prompt templates

User story:

As a maintainer, I want reusable prompt templates so planner and worker agents can be started consistently from GitHub issue context.

Acceptance criteria:

- Add planner and worker prompt templates under `docs/` or `tools/codex-harness/prompts/`.
- Planner prompt tells the agent to read the customer request, roadmap, current docs, and GitHub state before creating issues.
- Worker prompt tells the agent to read `AGENTS.md`, the GitHub issue, linked handoff doc, and verification expectations before editing.
- Prompts explicitly preserve existing repo rules around issues, handoffs, verification, and human review.

Verification:

```powershell
git diff -- docs tools
```

### Ticket 3: Create a PowerShell worker queue script

User story:

As a maintainer, I want a worker script that finds the next ready GitHub issue, claims it, and prints the exact Codex startup prompt so worker sessions can be started with less manual coordination.

Acceptance criteria:

- Add `tools/codex-harness/worker.ps1`.
- Script finds issues labeled `status:ready` and `agent:worker`.
- Script claims one issue by applying `status:in-progress` and removing `status:ready`.
- Script prints the issue URL, title, and worker startup prompt.
- Script has a dry-run mode.
- Script handles missing `gh` with a clear error message.

Verification:

```powershell
powershell -ExecutionPolicy Bypass -File tools/codex-harness/worker.ps1 -DryRun
```

### Ticket 4: Create a PowerShell planner helper script

User story:

As a maintainer, I want a planner helper that gathers GitHub issue state and prints the planner startup prompt so planning sessions can reliably create the next issue and handoff.

Acceptance criteria:

- Add `tools/codex-harness/planner.ps1`.
- Script lists open and recently closed issues relevant to a feature label or tracking doc.
- Script prints a planner startup prompt with required docs and workflow expectations.
- Script does not create issues automatically in the MVP.
- Script has parameters for feature name, tracking doc, and integration branch.

Verification:

```powershell
powershell -ExecutionPolicy Bypass -File tools/codex-harness/planner.ps1 -FeatureName "inventory redesign" -TrackingDoc docs/SUNNY_TOWN_INVENTORY_REDESIGN_TRACKING.md -DryRun
```

### Ticket 5: Add a demo walkthrough

User story:

As someone evaluating the project, I want a short demo walkthrough that shows how a customer request becomes user stories, issues, worker tasks, verification, and human review.

Acceptance criteria:

- Add `docs/CODEX_GITHUB_HARNESS_DEMO.md`.
- Demo uses one realistic website/app change request.
- Demo shows the documents and GitHub labels involved at each step.
- Demo includes where human review happens and how follow-up tickets are created.

Verification:

```powershell
git diff -- docs/CODEX_GITHUB_HARNESS_DEMO.md
```

## Open Questions

- Should the first worker script launch Codex automatically, or should it print the exact prompt for a human to paste/start?
- Should completed worker issues move to `status:review` first, or close immediately after deterministic verification?
- Should planner-created issues be just-in-time only, or should some stable roadmap batches be created in advance?
- Should the harness use GitHub assignees for locks, or only labels?
- Should the harness eventually run multiple workers, or stay single-worker until the branch and locking behavior is proven?

## Recommended First Slice

Start with documentation and prompt templates before scripts.

Reason:

- The manual workflow is already working.
- The main risk is ambiguity in role boundaries, issue labels, and completion states.
- Scripts should automate a settled protocol instead of freezing the first draft too early.
