# NPC Life Work Open PR Handoff

## Purpose

This handoff starts GitHub issue #57, "Open NPC life/work integration PR."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/57

The goal is to open the PR from `codex/npc-life-work-dev` into `main` with enough context for human review.

## Current Status

- Epic #38 is open.
- First NPC life/work demo-loop implementation issues are closed:
  - #48 Design NPC home and work demo loop
  - #49 Author Cookie Keeper home map and bed fixture
  - #50 Add configurable NPC day cadence
  - #51 Tune Cookie Keeper home/work demo loop
  - #52 Improve NPC routine debug output for demo review
  - #55 Add optional NPC routine visual cues
- Related PR blocker #54 is closed:
  - Use selected hotbar item as active Sunny Town tool
- PR-readiness issue #56 is closed with recommendation: ready to open PR.
- No open PR currently exists from `codex/npc-life-work-dev` into `main` as of this handoff.

## Branches

- Head: `codex/npc-life-work-dev`
- Base: `main`

Before opening the PR:

```powershell
git checkout codex/npc-life-work-dev
git pull
gh pr list --state open --head codex/npc-life-work-dev --base main
```

## Suggested PR Title

```text
Add first NPC life/work demo loop
```

## Suggested PR Body

Use this as a starting point:

```markdown
## Summary

- Adds the first NPC life/work demo loop for Cookie Keeper, including a home interior, bed fixture, portal routing, and home/work schedule behavior.
- Adds configurable NPC day cadence plus richer `/debug/npcs` routine diagnostics.
- Adds server-authored NPC `routineStatus` snapshots and subtle canvas routine cues.
- Updates Sunny Town active tool behavior so the selected hotbar pickaxe is the active mining tool, with server-side ownership validation.
- Updates current docs for the runtime/config/API/database/NPC behavior touched by this slice.

## Issues

Closes #48
Closes #49
Closes #50
Closes #51
Closes #52
Closes #55
Closes #54
Closes #56
Refs #38

## Verification

From #56 PR-readiness review:

- `task verify`
- `docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town`
- HQ health `http://127.0.0.1:18080/healthz` -> `ok`
- Sunny Town health `http://127.0.0.1:18082/healthz` -> `ok`
- `/debug/npcs` without service secret -> `401`
- `/debug/npcs` with `X-HQ-Service-Secret: local-dev-service-secret` -> routine/debug JSON returned
- Static app loaded and reached Keycloak login
- Seeded student auth and `/api/student/sunny-town/session` succeeded when using the configured Compose public host
- Sunny Town websocket returned a valid `hello` for `sunny-town-house-1`
- `/debug/npcs` showed Cookie Keeper with work/home anchors, durable character identity, and eligible production state

Known non-blocking note: if Compose starts with `HQ_PUBLIC_HOST=<LAN IP>`, open HQ through that same host instead of `127.0.0.1` to avoid Keycloak issuer-host mismatch. This is documented in `docs/current/RUNTIME.md` and README.
```

## Commands

Open the PR:

```powershell
gh pr create --base main --head codex/npc-life-work-dev --title "Add first NPC life/work demo loop" --body-file <body-file>
```

After PR creation:

```powershell
gh pr view --web
gh pr view <number> --json number,title,url,baseRefName,headRefName,state,body
```

## Tracking Updates

After the PR is created:

- Add the PR URL to `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`.
- Comment on #38 with the PR URL and note that the first demo slice is ready for human review.
- Comment on #57 with the PR URL.
- Close #57 after tracking is updated and committed.

## Close Criteria

Close #57 when:

- The PR exists.
- The PR body references the completed issues and verification notes.
- Tracking doc includes the PR URL.
- #38 or the PR has a clear note that the first NPC life/work demo slice is ready for human review.
- Work is committed and pushed to `codex/npc-life-work-dev`.
