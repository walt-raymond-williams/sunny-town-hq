# Restructure Implementation Plan

This plan tracks the repository restructure work identified in the project review. The work is now being handled by one refactor owner, so phases can touch any needed project area as long as each slice is behavior-preserving and thoroughly verified before commit.

## Resume Snapshot

Last updated: 2026-06-06.

Current state:

- All changes through `a77d7a1 Update restructure plan resume snapshot` are committed.
- Active work has started on promoting Sunny Town backend code into `internal/sunnytown/server`.
- Before commit, `git status --short` should show only the current Sunny Town move, this plan update, and known untracked local logs:
  - `hq-local.err.log`
  - `hq-local.out.log`
- Phase 2 preparatory file splits are complete.
- Current Sunny Town backend package shape:
  - `cmd/sunny-town/main.go`
  - `internal/sunnytown/server/`

Recommended next work:

1. Review the `internal/sunnytown/server` move and keep the public API narrow.
2. Consider a later split from `internal/sunnytown/server` into `internal/sunnytown/world` only after explicit world/server interfaces are clear.
3. Keep each promotion behavior-preserving and run:

```powershell
go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth
go test ./...
cd frontend
npm run build
```

## Coordination Rules

- Treat the repository as a single-owner refactor during this work.
- Keep each phase behavior-preserving unless the task explicitly says otherwise.
- Run scoped verification after each meaningful slice, then full repository verification before commit.
- Record verification results in this file.

## Phase 1: Collaboration Surface

Goal: make repo operations and current-state documentation obvious for humans and agents.

- [x] Add a project task runner, preferably `Taskfile.yml`, with commands for backend tests, frontend build, Docker Compose rebuild, health checks, and full verification.
- [x] Create `docs/current/` for current-state docs.
- [x] Create `docs/archive/` for historical plans.
- [x] Update `README.md` to point to current docs and task commands.
- [x] Update `AGENTS.md` with task-runner commands once they exist.
- [x] Move or clearly mark stale historical docs without touching active AI plans.

Verification:

- [x] `go test ./...`
- [x] `cd frontend && npm run build`
- [x] Confirm `git status --short` contains only intentional docs/task-runner changes plus known local logs.

## Phase 2: Sunny Town Backend Split

Goal: reduce `cmd/sunny-town/main.go` into a thin entrypoint and focused internal packages.

Target structure:

```text
cmd/sunny-town/main.go
internal/sunnytown/config/
internal/sunnytown/hqclient/
internal/sunnytown/maps/
internal/sunnytown/protocol/
internal/sunnytown/server/
```

Tasks:

- [x] Move config loading into `internal/sunnytown/config`.
- [x] Move map structs, loading, and validation into `internal/sunnytown/maps`.
- [x] Move WebSocket message structs into `internal/sunnytown/protocol`.
- [x] Move room, player, world, movement, portal, collectible, resource, and placement logic into `internal/sunnytown/server`.
- [x] Split Sunny Town world data types into `cmd/sunny-town/world_types.go` as a preparatory step.
- [x] Split Sunny Town world lifecycle methods into `cmd/sunny-town/world_lifecycle.go` as a preparatory step.
- [x] Split Sunny Town movement, collision, clamping, and portal helpers into `cmd/sunny-town/world_movement.go` as a preparatory step.
- [x] Split Sunny Town snapshot and broadcast helpers into `cmd/sunny-town/world_snapshots.go` as a preparatory step.
- [x] Split Sunny Town collectibles, resources, placement geometry, and world object helpers into `cmd/sunny-town/world_objects.go` as a preparatory step.
- [x] Move internal HQ HTTP calls into `internal/sunnytown/hqclient`.
- [x] Move WebSocket server setup and request handling into `internal/sunnytown/server`.
- [x] Split Sunny Town server/WebSocket wiring into `cmd/sunny-town/server.go` as a preparatory step.
- [x] Split Sunny Town client WebSocket pumps and send/rate-limit helpers into `cmd/sunny-town/client_io.go` as a preparatory step.
- [x] Split Sunny Town client gameplay handlers into `cmd/sunny-town/client_gameplay.go` as a preparatory step.
- [x] Split Sunny Town reward and resource commit workers into `cmd/sunny-town/server_workers.go` as a preparatory step.
- [x] Move Sunny Town backend tests into `internal/sunnytown/server`.

Verification:

- [x] `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth`
- [x] `go test ./...`
- [x] `cd frontend && npm run build`

## Phase 3: Sunny Town Frontend Split

Goal: break `SunnyTownPage.vue` into composables and focused UI components.

Target structure:

```text
frontend/src/features/sunny-town/
  SunnyTownPage.vue
  SunnyTownCanvas.vue
  SunnyTownHud.vue
  SunnyTownInventoryPanel.vue
  SunnyTownCraftingPanel.vue
  SunnyTownDialogue.vue
  SunnyTownShop.vue
  SunnyTownSchoolworkPanel.vue

frontend/src/composables/
  useSunnyTownSocket.ts
  useSunnyTownMovement.ts
  useSunnyTownRenderer.ts
  useSunnyTownInteractions.ts
```

Tasks:

Slice plan:

- [x] Slice 3.1: Move `SunnyTownPage.vue` from `pages/` to `features/sunny-town/` and update router imports without changing behavior.
- [x] Slice 3.2: Extract socket/session lifecycle into `frontend/src/composables/useSunnyTownSocket.ts`.
- [x] Slice 3.3: Extract movement input, client prediction, collision helpers, and placement validation into `frontend/src/composables/useSunnyTownMovement.ts`.
- [ ] Slice 3.4: Extract canvas rendering into `frontend/src/composables/useSunnyTownRenderer.ts` and `SunnyTownCanvas.vue`.
- [ ] Slice 3.5: Extract HUD/status and hotbar controls into `SunnyTownHud.vue`.
- [ ] Slice 3.6: Extract inventory and crafting panels into `SunnyTownInventoryPanel.vue` and `SunnyTownCraftingPanel.vue`.
- [ ] Slice 3.7: Extract dialogue, shop, and schoolwork panels into `SunnyTownDialogue.vue`, `SunnyTownShop.vue`, and `SunnyTownSchoolworkPanel.vue`.
- [ ] Slice 3.8: Final Phase 3 review pass: remove dead code, verify layout, update docs, and commit the final frontend split.

Original task coverage:

- [x] Extract Sunny Town socket/session lifecycle.
- [x] Extract movement input and client prediction.
- [ ] Extract canvas rendering.
- [ ] Extract HUD and status UI.
- [ ] Extract inventory and crafting panels.
- [ ] Extract dialogue and shop panels.
- [ ] Extract schoolwork panel.
- [x] Update router import if the page moves from `pages/` to `features/sunny-town/`.

Verification:

- [ ] `cd frontend && npm run build`
- [ ] Smoke-test Sunny Town in browser or container-served app when practical.
- [ ] Run `go test ./...` after Phase 3 completion to catch cross-project regressions.

## Phase 4: HQ Backend Split

Goal: reduce `cmd/hq/main.go` after the Sunny Town backend and frontend splits are complete.

Target structure:

```text
cmd/hq/main.go
internal/hq/app/
internal/hq/httpapi/
internal/hq/auth/
internal/hq/assignments/
internal/hq/inventory/
internal/hq/pet/
internal/hq/sunnytownbridge/
internal/hq/ai/
internal/hq/db/
```

Tasks:

- [ ] Move app config and construction into `internal/hq/app`.
- [ ] Move route registration and HTTP helpers into `internal/hq/httpapi`.
- [ ] Move assignment handlers and assignment service logic.
- [ ] Move inventory, equipment, hotbar, and crafting logic.
- [ ] Move pet service logic.
- [ ] Move Sunny Town bridge endpoints.
- [ ] Move AI integration behind the chosen HQ package boundaries.
- [ ] Keep `cmd/hq/main.go` as a thin binary entrypoint.

Verification:

- [ ] `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`
- [ ] `go test ./...`

## Phase 5: Database Migration Story

Goal: replace mixed Docker init SQL plus runtime schema patching with explicit migrations.

Start after the HQ backend split is stable enough that schema ownership is clear.

Target structure:

```text
deploy/postgres/migrations/
  0001_initial.sql
  0002_keycloak_users.sql
  0003_inventory_equipment.sql
  0004_sunny_town_state.sql
  0005_ai_grading.sql
```

Tasks:

- [ ] Convert current schema state into ordered migration files.
- [ ] Choose migration runner: lightweight Go runner or an external migration tool.
- [ ] Update Docker fresh database startup to apply migrations.
- [ ] Reduce `ensureSchema` to migration execution or remove it.
- [ ] Document reset and migration workflows.

Verification:

- [ ] Fresh Docker Compose startup creates a working database.
- [ ] Existing local database can migrate without data loss.
- [ ] `go test ./...`

## Phase 6: CI And Guardrails

Goal: make the established verification path automatic.

Tasks:

- [ ] Add CI for `go test ./...`.
- [ ] Add CI for `cd frontend && npm ci && npm run build`.
- [ ] Add CI for `docker compose -f deploy/docker-compose.yml config`.
- [ ] Consider `buf lint` and `buf breaking` after generated-code workflow is documented.
- [ ] Consider frontend unit or browser smoke tests after Sunny Town frontend split.

Verification:

- [ ] CI passes on the restructure branch.

## Current Verification Log

- 2026-06-06: `go test ./...` passed during project review.
- 2026-06-06: `cd frontend && npm run build` passed during project review. Vite reported a large chunk warning.
- 2026-06-06: Phase 1 added `Taskfile.yml`, `docs/current/`, `docs/archive/`, and updated `README.md`, `AGENTS.md`, and `docs/TESTING_GUIDELINES.md`.
- 2026-06-06: `go test ./...` passed after Phase 1 changes.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 1 changes. Vite still reported the large chunk warning.
- 2026-06-06: `docker compose -f deploy\docker-compose.yml config` passed after Phase 1 changes.
- 2026-06-06: Phase 2 started by moving Sunny Town config loading into `internal/sunnytown/config`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town config extraction.
- 2026-06-06: `go test ./...` passed after Sunny Town config extraction.
- 2026-06-06: Phase 2 moved Sunny Town map structs, loading, and validation into `internal/sunnytown/maps`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town map extraction.
- 2026-06-06: `go test ./...` passed after Sunny Town map extraction.
- 2026-06-06: Phase 2 moved Sunny Town WebSocket protocol structs into `internal/sunnytown/protocol`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town protocol extraction.
- 2026-06-06: `go test ./...` passed after Sunny Town protocol extraction.
- 2026-06-06: Phase 2 moved Sunny Town internal HQ HTTP calls into `internal/sunnytown/hqclient`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town HQ client extraction.
- 2026-06-06: `go test ./...` passed after Sunny Town HQ client extraction.
- 2026-06-06: Phase 2 split Sunny Town server/WebSocket wiring into `cmd/sunny-town/server.go` as a preparatory step before moving it to `internal/sunnytown/server`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town server file split.
- 2026-06-06: `go test ./...` passed after Sunny Town server file split.
- 2026-06-06: Phase 2 split Sunny Town world data types into `cmd/sunny-town/world_types.go` as a preparatory step before moving world logic to `internal/sunnytown/world`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town world type split.
- 2026-06-06: `go test ./...` passed after Sunny Town world type split.
- 2026-06-06: Phase 2 split Sunny Town world lifecycle methods into `cmd/sunny-town/world_lifecycle.go` as a preparatory step before moving world logic to `internal/sunnytown/world`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town world lifecycle split.
- 2026-06-06: `go test ./...` passed after Sunny Town world lifecycle split.
- 2026-06-06: Phase 2 split Sunny Town movement, collision, clamping, and portal helpers into `cmd/sunny-town/world_movement.go` as a preparatory step before moving world logic to `internal/sunnytown/world`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town movement split.
- 2026-06-06: `go test ./...` passed after Sunny Town movement split.
- 2026-06-06: Phase 2 split Sunny Town snapshot and broadcast helpers into `cmd/sunny-town/world_snapshots.go` as a preparatory step before moving world logic to `internal/sunnytown/world`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town snapshot split.
- 2026-06-06: `go test ./...` passed after Sunny Town snapshot split.
- 2026-06-06: Phase 2 split Sunny Town collectibles, resources, placement geometry, and world object helpers into `cmd/sunny-town/world_objects.go` as a preparatory step before moving world logic to `internal/sunnytown/world`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town world object split.
- 2026-06-06: `go test ./...` passed after Sunny Town world object split.
- 2026-06-06: Phase 2 split Sunny Town client WebSocket pumps and send/rate-limit helpers into `cmd/sunny-town/client_io.go` as a preparatory step before moving server logic to `internal/sunnytown/server`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town client IO split.
- 2026-06-06: `go test ./...` passed after Sunny Town client IO split.
- 2026-06-06: Phase 2 split Sunny Town reward and resource commit workers into `cmd/sunny-town/server_workers.go` as a preparatory step before moving server logic to `internal/sunnytown/server`.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town worker split.
- 2026-06-06: `go test ./...` passed after Sunny Town worker split.
- 2026-06-06: Phase 2 split Sunny Town client gameplay handlers into `cmd/sunny-town/client_gameplay.go` as a preparatory step before moving server/world logic to internal packages.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after Sunny Town client gameplay split.
- 2026-06-06: `go test ./...` passed after Sunny Town client gameplay split.
- 2026-06-06: Phase 2 moved Sunny Town server, client, room, world, movement, placement, resource, snapshot, and backend tests into `internal/sunnytown/server`; `cmd/sunny-town/main.go` is now a thin entrypoint.
- 2026-06-06: `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth` passed after the Sunny Town server package move.
- 2026-06-06: `go test ./...` passed after the Sunny Town server package move.
- 2026-06-06: `cd frontend && npm run build` passed after the Sunny Town server package move. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.1 moved `SunnyTownPage.vue` into `frontend/src/features/sunny-town/` and updated the router import.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.1. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.2 extracted Sunny Town WebSocket/session lifecycle into `frontend/src/composables/useSunnyTownSocket.ts`.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.2. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.3 extracted Sunny Town movement input, prediction, collision, and placement validation into `frontend/src/composables/useSunnyTownMovement.ts`.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.3. Vite still reported the large chunk warning.
