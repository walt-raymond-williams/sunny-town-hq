# NPC Life Work PR Readiness Handoff

## Purpose

This handoff starts GitHub issue #56, "Review NPC life/work integration branch for PR readiness."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/56

The goal is to review `codex/npc-life-work-dev` end-to-end and decide whether the branch is ready for a PR into `main`.

## Current Status

- Epic #38 is still open.
- The first NPC life/work demo loop child issues are closed:
  - #48 design
  - #49 Cookie Keeper home map and bed fixture
  - #50 configurable NPC day cadence
  - #51 Cookie Keeper home/work routine
  - #52 NPC routine debug output
  - #55 NPC routine visual cues
- Related blocker #54, hotbar-selected active tool behavior, is closed.
- Work/review should happen on `codex/npc-life-work-dev`.

## Review Goal

This is a review/readiness ticket, not another feature slice.

The reviewer should answer:

- Is the integrated NPC life/work demo slice correct enough to open a PR?
- Are there blocking bugs that need follow-up issues before merge?
- Are the current-state docs accurate?
- Are any completed handoff/planning docs misleading compared with implemented behavior?
- Does #54's hotbar-active-tool fix stay compatible with the NPC branch and mining gameplay?

## Branch Scope

Key behavior added on this branch:

- Cookie Keeper home interior map and portal.
- Bed fixture support.
- Compressed NPC day cadence via `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`.
- Cookie Keeper drive/schedule tuning for home/work routine.
- `/debug/npcs` routine, route, failure, schedule, and production diagnosis.
- Authoritative NPC `routineStatus` snapshots.
- Canvas routine cues for NPCs.
- Hotbar-selected owned pickaxe as the active Sunny Town mining tool.
- Current docs updates for runtime/API/database/NPC behavior.

Useful diff commands:

```powershell
git log --oneline origin/main..HEAD
git diff --stat origin/main...HEAD
git diff origin/main...HEAD -- docs/current
```

## Code And Doc Surfaces

Sunny Town backend:

- `internal/sunnytown/server/client_gameplay.go`
- `internal/sunnytown/server/npc_movement.go`
- `internal/sunnytown/server/npc_schedule.go`
- `internal/sunnytown/server/npc_debug.go`
- `internal/sunnytown/server/world_snapshots.go`
- `internal/sunnytown/server/server_workers.go`
- `internal/sunnytown/config/config.go`
- `internal/sunnytown/protocol/protocol.go`

Sunny Town maps:

- `sunny-town/maps/sunny-town-v1.json`
- `sunny-town/maps/sunny-town-cookie-keeper-home.json`

Frontend:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
- `frontend/src/features/sunny-town/rendering/characterDrawing.ts`
- `frontend/src/features/sunny-town/rendering/objectDrawing.ts`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/types/sunnyTown.ts`

Current docs:

- `AGENTS.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/RUNTIME.md`
- `docs/current/NPC_LOCATION_PATHING_DRIVES_PLAN.md`
- `docs/current/STATS_SKILLS_PROGRESSION.md`

Planning/tracking docs:

- `docs/NPC_LIFE_WORK_SIMULATION_DISCOVERY.md`
- `docs/NPC_LIFE_WORK_SIMULATION_ROADMAP.md`
- `docs/NPC_LIFE_WORK_SIMULATION_TRACKING.md`
- `docs/SUNNY_TOWN_HOTBAR_ACTIVE_TOOL_HANDOFF.md`

## Review Checklist

- Cookie Keeper can route between `sunny-town-house-1`, `sunny-town-v1`, and `sunny-town-cookie-keeper-home`.
- Cookie Keeper uses `cookie-keeper-bed` for rest and `cookie-keeper-counter` for work.
- Cookie Keeper production still requires physical presence at the work anchor.
- NPC day cadence defaults to 24 minutes and can be configured to 8 minutes for demo review.
- `/debug/npcs` remains protected by service secret when configured.
- `/debug/npcs` clearly explains goal, route, schedule pressure, failures, routine status, and production state.
- `routineStatus` is server-authoritative and does not make the client infer fake routine state.
- Routine visual cues are subtle and do not obscure NPC/player interaction UI.
- Mining uses an owned pickaxe selected in the Sunny Town hotbar.
- Spoofed pickaxe tool use without ownership still fails server-side.
- Stone block placement still uses the selected hotbar item.
- The character card no longer suggests a separate active tool slot is required.
- `docs/current/` matches implemented behavior.
- Local generated files and logs are not staged.

## Verification

Prefer:

```powershell
$taskDir = 'C:\Users\waltr\AppData\Local\Microsoft\WinGet\Packages\Task.Task_Microsoft.Winget.Source_8wekyb3d8bbwe'
$env:Path = "$taskDir;$env:Path"
task verify
```

If Task is unavailable:

```powershell
go test ./...
cd frontend
npm run build
cd ..
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual smoke if practical:

- Start with `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES=8`.
- Open Sunny Town.
- Confirm the player can select pickaxe in the hotbar and mine a natural rock.
- Confirm placed stone block placement and mining still work.
- Enter Cookie Shop and Cookie Keeper home.
- Observe Cookie Keeper traveling, working, and resting.
- Confirm routine cues appear for supported states.
- Call `/debug/npcs` and compare debug state to what is visible.

## Expected Output

The reviewer should leave an issue comment with:

- verification commands run
- manual smoke coverage
- blocking findings, if any
- non-blocking follow-ups, if any
- recommendation: ready to open PR, needs fixes, or split/follow-up first

Create linked GitHub issues for any blocking bug that should not be buried in review prose.

## Close Criteria

Close #56 when:

- The branch has been reviewed against this checklist.
- Verification results are recorded.
- Blocking follow-ups, if any, are created and linked.
- The issue comment clearly states whether `codex/npc-life-work-dev` is ready for PR.
