# GitHub Tracking Document Guide

## Purpose

Use this guide when a feature is large enough to need more than one GitHub issue, more than one agent turn, or a dedicated integration branch.

The pattern is:

- GitHub Issues are the execution source of truth.
- Local Markdown docs preserve product direction, architecture rationale, and fresh-agent memory.
- A tracking doc preserves the current workflow state after compaction.
- A handoff doc gives the next implementation agent a narrow, actionable starting point.

## When To Create A Tracking Doc

Create a feature tracking doc when work has any of these traits:

- multiple dependent GitHub issues
- a feature integration branch
- sequential agent handoffs
- important product/design context that should survive compaction
- work that should not merge to `main` until a human reviews an integrated slice

Recommended filename:

```text
docs/<FEATURE_NAME>_TRACKING.md
```

Example:

```text
docs/SUNNY_TOWN_INVENTORY_REDESIGN_TRACKING.md
```

## Tracking Doc Template

Copy this template for new multi-ticket features.

```markdown
# <Feature Name> Tracking

## Purpose

This document preserves the current GitHub/task workflow state for compaction recovery and fresh-agent handoff. GitHub issues remain the execution source of truth; this file is a local memory snapshot.

## Integration Branch

Current feature integration branch:

`codex/<feature-name>-dev`

Workflow:

- Agents work issues against `codex/<feature-name>-dev`.
- Issues may be closed once their work lands on the integration branch and verification is recorded.
- `main` remains stable until the integrated feature slice is ready for human review.
- A human reviews the integrated feature before merging `codex/<feature-name>-dev` into `main`.

## Management Workflow Loop

Use this loop when advancing the feature backlog:

1. Check live GitHub issue state with `gh issue list` and inspect the last completed issue comments.
2. Confirm the previous ticket is closed with a commit hash and verification notes.
3. Pick the next roadmap item based on blockers, dependencies, and the current integrated branch state.
4. Create the GitHub issue if it does not already exist.
5. Create a focused handoff doc under `docs/` for the selected issue.
6. Update this tracking file with the closed/open issue snapshot, recommended next task, and handoff path.
7. Commit and push only the handoff/tracking docs to the feature integration branch.
8. Comment on the GitHub issue with the handoff path, branch, and commit hash.
9. The implementation agent works from the GitHub issue plus the linked handoff doc.
10. When implementation lands on the feature integration branch, close the issue with the commit hash and verification results.
11. Repeat from step 1.

Keep GitHub issues as the execution source of truth. Keep this file as the compact local recovery snapshot for compaction, coordination, and fresh-agent handoff.

## Current GitHub Issues

Snapshot refreshed: <YYYY-MM-DD after issue #N creation/closure>.

### Closed

- #<number> <title>  
  <url>

### Open

- #<number> <title>  
  Status: `status:ready`  
  Blocked by: #<number>, closed  
  <url>

## Recommended Next Management Step

Start issue #<number>:

`<Issue title>`

Why:

- <reason tied to blockers/dependencies>
- <reason tied to product value>
- Handoff: `docs/<FEATURE_NAME>_<TASK_NAME>_HANDOFF.md`.

## Useful Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view <number> --comments
git checkout codex/<feature-name>-dev
git pull
```

## Related Local Docs

- `AGENTS.md`
- `docs/<FEATURE_NAME>_DISCOVERY.md`
- `docs/<FEATURE_NAME>_ROADMAP.md`
- `docs/<FEATURE_NAME>_TICKETS.md`
- `docs/<FEATURE_NAME>_<TASK_NAME>_HANDOFF.md`
```

## Handoff Doc Template

Create one handoff doc per GitHub issue when an implementation agent needs focused startup context.

Recommended filename:

```text
docs/<FEATURE_NAME>_<TASK_NAME>_HANDOFF.md
```

Template:

```markdown
# <Feature/Task Name> Handoff

## Purpose

This handoff starts GitHub issue #<number>, "<issue title>."

Issue: <issue url>

The goal is to <one-sentence outcome>.

## Current Status

- <completed prerequisite issue/status>
- <current branch/state>
- <why this task is next>

## Product Direction

- <user-facing behavior>
- <important UX or design constraints>
- <what this should feel like>

## Current Code Surface

Frontend:

- `<path>`
  - <why it matters>

Backend:

- `<path or endpoint>`
  - <why it matters>

Docs:

- `<path>`
  - <why it matters>

## Recommended Implementation Plan

1. <step>
2. <step>
3. <step>

## Out Of Scope

- <explicitly deferred work>
- <related future tickets>

## Verification

Run:

```powershell
<command>
<command>
```

Manual smoke:

- <manual check>
- <manual check>

If full browser/runtime verification is blocked, record the blocker in the GitHub issue close comment and include deterministic checks that passed.

## Close Criteria

Close #<number> when:

- work is committed and pushed to `<branch>`
- verification results are recorded on the issue
- docs/current is updated if architecture/API/schema/runtime behavior changed

Use a close comment like:

```text
Completed in <commit>. <summary>. Verified with: <commands>. Manual smoke: <result or blocker>.
```
```

## Issue Creation Checklist

Before creating an issue:

- Confirm the issue does not already exist.
- Read the latest tracking doc and previous issue close comments.
- Confirm blockers are closed or label the new issue as blocked.
- Keep the issue small enough for one focused agent pass.

Issue body should include:

- user story or problem statement
- acceptance criteria
- implementation notes
- blockers
- related/future issues
- verification commands
- context docs

## Recommended Commands

```powershell
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 50 --state all
& 'C:\Program Files\GitHub CLI\gh.exe' issue view <number> --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue create --title "<title>" --body "<body>" --label "type:feature" --label "status:ready"
& 'C:\Program Files\GitHub CLI\gh.exe' issue comment <number> --body "Handoff prepared on <branch> in commit <commit>. Start from docs/<handoff>.md."
```

## Practical Rules

- Do not bulk-create every roadmap issue when early implementation details are still unstable.
- Prefer just-in-time issue creation when preceding tickets may change APIs, component boundaries, or acceptance criteria.
- Bulk-create a coherent batch only after dependencies and interfaces are stable.
- Keep GitHub for execution status and durable discussion.
- Keep Markdown docs for roadmap, architecture rationale, and handoff memory.
- Stage docs explicitly; do not stage unrelated active implementation files.
