# AGENTS.md

## Project Posture

This is an AI-assisted project. Treat this file as a living operating guide for Codex and other coding agents working in this repository.

Keep guidance practical and compact. Add to this file when a workflow repeatedly helps, when a mistake recurs, or when debugging teaches a rule future agents should inherit. Prefer concrete commands, ownership boundaries, and verification steps over broad philosophy.

## Repository Shape

- HQ binary startup and route composition live in `cmd/hq`; HQ domain code lives under `internal/hq/...`.
- Sunny Town binary startup lives in `cmd/sunny-town`; Sunny Town realtime service code lives under `internal/sunnytown/...`.
- Sunny Town map JSON lives in `sunny-town/maps`.
- Vue frontend code lives in `frontend/src`.
- Project docs, plans, setup notes, and testing guidance live in `docs/`.
- Current-state architecture, package boundaries, API, database, and runtime docs live in `docs/current/`.
- Historical planning docs live in `docs/archive/`.
- Production frontend assets are generated into local `web/` by `npm run build`; `web/` is ignored and should not be committed.
- Integration/runtime configuration lives under `deploy/`.
- Common verification and runtime commands live in `Taskfile.yml`.

## Ownership Boundaries

- HQ owns durable student/account/economy/inventory state and database schema.
- Sunny Town owns live realtime world state: connected players, current map membership, accepted player positions, collectibles, resource node state, transitions, and gameplay validation.
- Sunny Town should not write the HQ database directly. Use service-authenticated internal HQ HTTP endpoints with `X-HQ-Service-Secret`.
- Browser/client messages are requests, not authority. Validate gameplay effects server-side using accepted server position and equipped tools.

## Development Workflow

- Prefer the repo's existing patterns over new abstractions.
- Use `rg` / `rg --files` for search.
- Use `apply_patch` for manual file edits.
- Do not revert unrelated user changes.
- Keep generated or local noise out of commits unless explicitly requested.
- `hq-local.err.log` and `hq-local.out.log` are intentionally visible in `git status`; do not stage them unless explicitly asked.
- Historical planning docs may be removed after their decisions are captured in current-state architecture docs.

## Feature Planning Workflow

For broad new features or redesigns, do discovery before implementation:

- First capture the product direction, constraints, and open questions in a planning doc under `docs/`.
- Audit the current code and current-state docs before decomposing work.
- Convert the plan into user stories and GitHub-ready tickets with acceptance criteria, implementation notes, blockers, related work, and verification steps.
- Keep the repo docs as durable project memory and design rationale.
- Use GitHub issues for execution tracking, ownership, status, discussion, and PR linkage.
- Once a GitHub issue exists for the work, treat the issue as the active task contract.
- Agents assigned to a GitHub issue should work from the issue, read the linked repo docs for context, and update docs only when decisions or current-state architecture change.
- Prefer small tickets that one agent can complete and verify. Split design/schema decisions from implementation when the implementation depends on unresolved architecture.
- For multi-ticket features, use `docs/GITHUB_TRACKING_DOC_GUIDE.md` to create tracking docs and per-issue handoff docs.

## GitHub Issue Workflow

Use GitHub Issues as the execution backlog for meaningful project work.

Create or select an issue before starting:

- new product features or user stories
- bug fixes with user-visible behavior
- architecture, schema, service-boundary, or protocol changes
- tech debt cleanup that changes maintainability risk
- demo, resume, or documentation work meant to shape project presentation

Do not require a new issue for:

- typos, tiny formatting edits, or one-line cleanup
- local experiments that are not committed
- small follow-up edits already covered by the active issue

Issue contents should be useful to a future agent:

- user story or problem statement
- acceptance criteria
- relevant docs and code areas
- dependencies, blockers, and related issues
- implementation checklist when the work is non-trivial
- verification commands expected before close

During work:

- Read the issue and linked docs before editing.
- Add issue comments only for durable information: design decisions, scope changes, blockers, important discoveries, commit hashes, and verification results.
- If new work appears, prefer creating a linked follow-up issue instead of expanding the current ticket until it becomes vague.
- Keep Markdown docs for product vision, architecture rationale, and durable decisions; keep GitHub Issues for execution status, discussion, and history.
- If implementation changes architecture, APIs, schema, service boundaries, runtime behavior, or durable workflows, summarize the accepted current state in `docs/current/` before closing the issue.

Recommended `gh` commands:

```powershell
gh issue list --limit 30
gh issue view <number> --comments
gh issue comment <number> --body "..."
gh issue close <number> --comment "Completed in <commit>. Verified with <command>."

# If gh is not on PATH on this Windows machine:
& 'C:\Program Files\GitHub CLI\gh.exe' issue list --limit 30
& 'C:\Program Files\GitHub CLI\gh.exe' issue view <number> --comments
& 'C:\Program Files\GitHub CLI\gh.exe' issue comment <number> --body "..."
& 'C:\Program Files\GitHub CLI\gh.exe' issue close <number> --comment "Completed in <commit>. Verified with <command>."
```

## Feature Integration Branch Workflow

For broad multi-ticket features or redesigns, use a feature integration branch instead of merging partial work directly into `main`.

- Name feature integration branches with the `codex/` prefix, for example `codex/inventory-redesign-dev`.
- Agents may complete individual GitHub issues on the integration branch or on smaller task branches that merge into the integration branch.
- Close a GitHub issue when its work has landed in the feature integration branch and verification has been recorded.
- Keep `main` stable. Do not merge broad in-progress feature work into `main` until the feature slice is coherent and ready for human review.
- Before merging an integration branch into `main`, a human should review the feature end to end, run the agreed verification, and confirm the docs/current state is accurate.
- The final PR from the integration branch to `main` should reference the completed issues with `Refs #123` or `Closes #123` as appropriate.

## Verification

Favor deterministic checks before manual browser exploration.

Preferred task commands:

```powershell
task test
task frontend:build
task compose:rebuild-runtime
task health
task verify
```

If Go Task is unavailable, use the direct commands below.

For backend changes:

```powershell
go test ./...
```

For Sunny Town-only backend changes:

```powershell
go test ./cmd/sunny-town
```

For HQ-only backend changes:

```powershell
go test ./cmd/hq
```

For frontend changes:

```powershell
cd frontend
npm run build
```

For integration/runtime verification, prefer Docker Compose:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
```

Then check health:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected response for both is `ok`.

## Sunny Town Notes

- Use maps/portals for area transitions; do not introduce a parallel cell abstraction unless the architecture changes intentionally.
- Portal targets should not land the player inside the destination portal trigger. Keep the server-side portal re-entry guard in place so players must leave a portal before triggering another transfer.
- Movement is client-predicted for feel, but gameplay effects must use server-accepted positions.
- Mining requires an equipped pickaxe and should be validated in Sunny Town before HQ receives any resource event.
- Resource ownership persists in HQ inventory tables. Sunny Town resource node depletion is in-memory unless explicitly changed.

## Frontend Notes

- Do not build a landing page for app features; build the usable interface.
- Keep controls compact and operational. This project is closer to an app/tool than a marketing site.
- After frontend changes, run `npm run build` from `frontend/` for local production-bundle checks. This updates ignored local `web/` assets.
- After significant local frontend changes, smoke-test the container-served app at `http://127.0.0.1:18080` when practical.

## Commit Hygiene

- Check `git status --short` before staging and before final response.
- Stage files explicitly. Avoid `git add .` unless the user specifically wants everything.
- Reference relevant issues in commit messages when practical.
- PR descriptions should reference issues with `Closes #123` for completed work or `Refs #123` for partial/related work.
- PR descriptions should include the verification commands run.
- Do not stage generated `web/` assets; rebuild them locally or through the Docker HQ image.
- Leave local logs untracked unless the user explicitly asks to commit them.

## Learning Loop

When a bug takes real effort to understand, or a successful workflow would be useful next time, propose an `AGENTS.md` update. Keep each addition short enough that future agents will actually read it.
