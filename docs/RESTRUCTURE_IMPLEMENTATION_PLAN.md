# Restructure Implementation Plan

This plan tracks the repository restructure work identified in the project review. The work is now being handled by one refactor owner, so phases can touch any needed project area as long as each slice is behavior-preserving and thoroughly verified before commit.

## Resume Snapshot

Last updated: 2026-06-07.

Current state:

- Phase 8 Slice 8.1 is implemented and verified.
- Next work is Phase 8 Slice 8.2: move authenticated-user persistence/sync and student listing behind an internal boundary.
- Phase 7 adapter retirement is complete: inventory, pet, Sunny Town bridge, AI grading, and assignment adapters have been retired.
- Current `cmd/hq` shape after Phase 8 Slice 8.1:
  - `main.go`: 75 lines
  - `routes.go`: 92 lines
  - `handlers.go`: 163 lines
  - `schema.go`: 128 lines
  - `reward_test.go`: 1,358 lines
  - `internal/hq/auth/auth.go`: 287 lines
  - `internal/hq/auth/auth_test.go`: 148 lines
- After committing Phase 8 Slice 8.1, `git status --short` should show only known untracked local logs:
  - `hq-local.err.log`
  - `hq-local.out.log`
- Phases 1 through 6 are complete except for deferred future guardrails noted below.
- Remaining structural work should focus on moving authenticated user persistence/sync and command HTTP helpers behind internal boundaries before moving app construction.

Recommended next work:

1. Move authenticated-user sync, role persistence, default student provisioning, student listing, and `requireUser` from `cmd/hq/schema.go` into `internal/hq/users` or the auth package.
2. Replace remaining command-local role adapters with shared auth package adapters or narrow exported helpers.
3. After auth/user management moves, reassess moving route construction and shared HTTP helpers into `internal/hq/httpapi`, then reassess moving app construction into `internal/hq/app`.

```powershell
go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth
go test ./...
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
- [x] Slice 3.4: Extract canvas rendering lifecycle into `frontend/src/composables/useSunnyTownRenderer.ts` and canvas DOM/events into `SunnyTownCanvas.vue`.
- [x] Slice 3.5: Extract HUD/status and hotbar controls into `SunnyTownHud.vue`.
- [x] Slice 3.6: Extract inventory and crafting panels into `SunnyTownInventoryPanel.vue`.
- [x] Slice 3.7: Extract dialogue, shop, and schoolwork panels into `SunnyTownDialogue.vue`, `SunnyTownShop.vue`, and `SunnyTownSchoolworkPanel.vue`.
- [x] Slice 3.8: Final Phase 3 review pass: remove dead code, verify layout, update docs, and commit the final frontend split.

Original task coverage:

- [x] Extract Sunny Town socket/session lifecycle.
- [x] Extract movement input and client prediction.
- [x] Extract canvas rendering.
- [x] Extract HUD and status UI.
- [x] Extract inventory and crafting panels.
- [x] Extract dialogue and shop panels.
- [x] Extract schoolwork panel.
- [x] Update router import if the page moves from `pages/` to `features/sunny-town/`.

Verification:

- [x] `cd frontend && npm run build`
- [x] Smoke-test Sunny Town in browser or container-served app when practical.
- [x] Run `go test ./...` after Phase 3 completion to catch cross-project regressions.

## Phase 4: HQ Backend Split

Goal: reduce `cmd/hq/main.go` after the Sunny Town backend and frontend splits are complete, while making pet a clearly separated HQ domain module that can later become its own service if the product needs it.

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

Slice plan:

- [x] Slice 4.1: Move HQ config loading into `internal/hq/app` while keeping behavior unchanged.
- [x] Slice 4.2: Split HQ route registration out of startup into `cmd/hq/routes.go` as a no-behavior-change waypoint.
- [ ] Slice 4.3: Move app construction into `internal/hq/app` once the `app` type can move cleanly. Deferred because the current `app` type still intentionally owns command-package auth, adapters, and route wiring.
- [ ] Slice 4.4: Move HTTP route registration and shared HTTP helpers into `internal/hq/httpapi`. Deferred until route wiring can depend on narrow handler interfaces instead of the command-package `app`.
- [x] Slice 4.5: Move pet domain into `internal/hq/pet`.
  - [x] Slice 4.5a: Move deterministic pet rules, game result normalization, and proto mood mapping into `internal/hq/pet`.
  - [x] Slice 4.5b: Move the Connect RPC handler into `internal/hq/pet` behind a narrow backend interface.
  - [x] Slice 4.5c: Move pet persistence operations and decay logic behind a pet store/service boundary.
  - [x] Slice 4.5d: Move pet decay ticker orchestration behind the new pet boundary.
- [x] Slice 4.6: Move inventory, equipment, hotbar, crafting, wallet, and shop logic into `internal/hq/inventory`.
  - [x] Slice 4.6a: Move base inventory response types plus load/increment/consume operations into `internal/hq/inventory`.
  - [x] Slice 4.6b: Move equipment load/equip/unequip operations into `internal/hq/inventory`.
  - [x] Slice 4.6c: Move hotbar load/set/default-seed operations into `internal/hq/inventory`.
  - [x] Slice 4.6d: Move crafting recipes and craft operation into `internal/hq/inventory`.
  - [x] Slice 4.6e: Move wallet/shop purchase operations into `internal/hq/inventory` or a clearer economy subpackage if the split calls for it.
- [x] Slice 4.7: Move assignment handlers and assignment service logic into `internal/hq/assignments`.
  - [x] Slice 4.7a: Move assignment request/response types plus load/filter helpers into `internal/hq/assignments`.
  - [x] Slice 4.7b: Move assignment grading command logic into `internal/hq/assignments` while preserving AI integration.
  - [x] Slice 4.7c: Move assignment create, submit, reset, delete, and load-by-id service operations behind the assignments boundary.
  - [x] Slice 4.7d: Move assignment HTTP handlers or route-ready handler factories behind the assignments boundary.
  - [x] Slice 4.7e: Harden the assignments boundary with typed query operations, direct package tests, and removal of stale adapter scaffolding.
- [x] Slice 4.8: Move Sunny Town internal bridge endpoints into `internal/hq/sunnytownbridge`.
- [x] Slice 4.9: Move AI grading integration behind `internal/hq/ai`.
- [x] Slice 4.10: Prepare schema ownership boundaries for Phase 5 migrations without changing the migration story yet.
- [x] Slice 4.11: Final HQ cleanup: keep `cmd/hq/main.go` as a thin binary entrypoint, remove dead code, update docs, and run full verification.

Pet module direction:

- [x] Keep pet in the HQ binary for Phase 4, but separate it as `internal/hq/pet`.
- [x] Keep pet schema in the shared HQ database for now.
- [x] Make pet depend on narrow interfaces for wallet/inventory/profile operations instead of broad app internals where practical.
- [x] Do not merge pet into AI; leave room for AI to consume pet context or emit pet-affecting commands through explicit interfaces later.

Original task coverage:

- [x] Move app config into `internal/hq/app`.
- [ ] Move app construction into `internal/hq/app` after command-local auth/user management is internalized.
- [x] Split route registration out of HQ startup.
- [ ] Move route registration and HTTP helpers into `internal/hq/httpapi` after auth/user dependencies are narrow enough to avoid moving the whole command package.
- [x] Move assignment handlers and assignment service logic.
- [x] Replace temporary raw SQL suffix APIs with typed package operations where package boundaries now own the queries.
- [x] Add direct package tests for extracted HQ domain packages, starting with assignments and inventory.
- [x] Remove temporary adapter aliases/wrappers once routes and handlers no longer need them.
- [x] Move inventory, equipment, hotbar, and crafting logic.
- [x] Move pet service logic into `internal/hq/pet`.
- [x] Move Sunny Town bridge endpoints.
- [x] Move AI integration behind the chosen HQ package boundaries.
- [x] Prepare schema ownership boundaries for Phase 5 migration conversion.
- [x] Keep `cmd/hq/main.go` as a thin binary entrypoint.

Verification:

- [x] `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`
- [x] `go test ./...`

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

- [x] Convert current schema state into ordered migration files.
- [x] Choose migration runner: lightweight Go runner for HQ startup plus the Docker Postgres init script for fresh databases.
- [x] Update Docker fresh database startup to apply migrations.
- [x] Reduce `ensureSchema` to migration execution or remove it.
- [x] Document reset and migration workflows.

Verification:

- [x] Fresh Docker Compose startup creates a working database.
- [x] Existing local database can migrate without data loss.
- [x] `go test ./...`

## Phase 6: CI And Guardrails

Goal: make the established verification path automatic.

Tasks:

- [x] Add CI for `go test ./...`.
- [x] Add CI for `cd frontend && npm ci && npm run build`.
- [x] Add CI for `docker compose -f deploy/docker-compose.yml config`.
- [ ] Consider `buf lint` and `buf breaking` after generated-code workflow is documented.
- [ ] Consider frontend unit or browser smoke tests after Sunny Town frontend split.

Verification:

- [ ] CI passes on the restructure branch once pushed.

## Phase 7: HQ Adapter Retirement

Goal: remove temporary `cmd/hq` adapter wrappers now that domain logic lives under `internal/hq/...`.

Do this one domain at a time. The point is not to move files for neatness; the point is to make package boundaries real so future changes do not bounce through package-main compatibility names.

Current adapter clusters:

- [x] Inventory/equipment/hotbar/crafting/shop adapters in `cmd/hq/inventory.go`, `cmd/hq/equipment.go`, `cmd/hq/hotbar.go`, `cmd/hq/crafting.go`, and `cmd/hq/shop.go`.
- [x] Pet profile/service adapters in `cmd/hq/pet_adapter.go` and `cmd/hq/pet_service.go`.
- [x] Sunny Town bridge compatibility adapters in `cmd/hq/sunnytown_bridge.go`.
- [x] AI grading compatibility adapters in `cmd/hq/ai_grading.go`.
- [x] Assignment grading compatibility adapters in `cmd/hq/assignment_grading.go` and `cmd/hq/assignments.go`.

Recommended slice order:

- [x] Slice 7.1: Move inventory, equipment, hotbar, crafting, and shop HTTP handlers from `cmd/hq/handlers.go` into `internal/hq/inventory` behind a route-ready handler factory.
- [x] Slice 7.2: Update `cmd/hq/routes.go` to mount the new inventory handler methods directly.
- [x] Slice 7.3: Add focused `internal/hq/inventory` HTTP handler tests for method checks, auth role behavior, bad JSON, known validation errors, and happy paths where practical.
- [x] Slice 7.4: Delete inventory/equipment/hotbar/crafting/shop adapter aliases and wrappers once no production code or tests depend on them.
- [ ] Slice 7.5: Reassess `cmd/hq/handlers.go` line count and remaining adapter clusters before choosing the next domain.
- [x] Slice 7.6: Retire pet profile/service adapters by using `internal/hq/pet.Store` directly for REST and Connect handlers, keeping only command auth/store construction glue.
- [x] Slice 7.7: Add focused `internal/hq/pet` HTTP handler tests for role checks, method checks, profile JSON, and no-cookie error mapping.
- [x] Slice 7.8: Retire Sunny Town bridge adapters by wiring routes and tests directly to `internal/hq/sunnytownbridge` and deleting `cmd/hq/sunnytown_bridge.go`.
- [x] Slice 7.9: Retire AI grading adapters by wiring internal AI routes, tests, and async grade triggering directly to `internal/hq/ai`, then deleting `cmd/hq/ai_grading.go`.
- [x] Slice 7.10: Retire remaining assignment compatibility adapters in `cmd/hq/assignment_grading.go` and `cmd/hq/assignments.go`.
- [x] Slice 7.11: Reassess `cmd/hq` after adapter retirement and decide whether Phase 4 deferred app construction or HTTP helper internalization is now worth doing.

Phase 7 reassessment result:

- [x] Adapter retirement is complete.
- [x] Moving all HQ app construction into `internal/hq/app` is not the next best code move. The `app` type still depends on command-local auth/user-sync types and static/logging helpers.
- [x] Moving route registration into `internal/hq/httpapi` is now plausible, but should wait until auth/user management is internalized or expressed behind narrow interfaces.
- [x] Next code phase should internalize HQ auth/user management first.

Verification:

- [x] `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`
- [x] `go test ./...`

## Phase 8: HQ Command Cleanup

Goal: shrink `cmd/hq` to binary startup plus minimal route composition by moving command-local auth/user management and HTTP infrastructure behind internal package boundaries.

Start after Phase 7 adapter retirement.

Recommended slice order:

- [x] Slice 8.1: Move Keycloak JWT verification and auth user types from `cmd/hq/auth.go` into `internal/hq/auth`.
- [ ] Slice 8.2: Move authenticated-user sync, role persistence, default student provisioning, student listing, and `requireUser` from `cmd/hq/schema.go` into the auth/user package or a focused `internal/hq/users` package.
- [ ] Slice 8.3: Replace command-local role adapters (`inventory_auth.go`, `pet_auth.go`, inline assignment role bridge, `requireStudentID`) with shared auth package adapters or narrow exported helpers.
- [ ] Slice 8.4: Move shared JSON/static/logging HTTP helpers into `internal/hq/httpapi` if the resulting API is small and boring.
- [ ] Slice 8.5: Reassess route registration after auth/user cleanup; move route construction only if it reduces `cmd/hq` coupling without creating a large configuration object.
- [ ] Slice 8.6: Reassess app construction in `internal/hq/app`; move only if `cmd/hq/main.go` can remain a readable binary entrypoint and tests stay straightforward.

Non-goals:

- [ ] Do not move package-main tests mechanically just to chase line counts; split `cmd/hq/reward_test.go` only when it improves ownership or lets tests live beside the package under test.
- [ ] Do not move `cmd/hq/main.go` startup concerns that are specific to the binary, such as fatal startup logging and local address printing, unless they become reusable.

Verification:

- [x] `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`
- [x] `go test ./...`

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
- 2026-06-06: Phase 3 Slice 3.4 extracted Sunny Town canvas DOM/events into `SunnyTownCanvas.vue` and render-loop/canvas preparation into `frontend/src/composables/useSunnyTownRenderer.ts`.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.4. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.5 extracted Sunny Town toolbar/status, toast/help, and hotbar controls into `SunnyTownHud.vue`.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.5. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.6 extracted Sunny Town inventory, equipment, hotbar editor, and crafting tray into `SunnyTownInventoryPanel.vue`.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.6. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.7 extracted Sunny Town dialogue, shop, and schoolwork overlays into focused components.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 Slice 3.7. Vite still reported the large chunk warning.
- 2026-06-06: Phase 3 Slice 3.8 completed final review. `SunnyTownPage.vue` is reduced from 2,142 lines to 1,554 lines, with socket, movement, renderer lifecycle, canvas, HUD, inventory/crafting, dialogue, shop, and schoolwork concerns split into focused files.
- 2026-06-06: `cd frontend && npm run build` passed after Phase 3 completion. Vite still reported the large chunk warning.
- 2026-06-06: `go test ./...` passed after Phase 3 completion.
- 2026-06-06: Vite preview smoke test returned HTTP 200 for `/` and `/student/pet/sunny-town`; in-app browser tooling was unavailable in this session.
- 2026-06-07: Phase 4 Slice 4.1 moved HQ runtime config loading into `internal/hq/app`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.1.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.1.
- 2026-06-07: Phase 4 Slice 4.2 split HQ route registration into `cmd/hq/routes.go` without changing handlers or auth wiring.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.2.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.2.
- 2026-06-07: Phase 4 Slice 4.5a created `internal/hq/pet` for deterministic pet rules, game result normalization, and proto mood mapping.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.5a.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.5a.
- 2026-06-07: Phase 4 Slice 4.5b moved the pet Connect RPC handler into `internal/hq/pet` and added a `cmd/hq` adapter for existing app persistence methods.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.5b.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.5b.
- 2026-06-07: Phase 4 Slices 4.5c-4.5d moved pet profile loading, feed/play/sleep/wake/game-result persistence, decay, and ticker orchestration into `internal/hq/pet.Store`, with HQ inventory and star ledger integration supplied as callbacks.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slices 4.5c-4.5d.
- 2026-06-07: `go test ./...` passed after Phase 4 Slices 4.5c-4.5d.
- 2026-06-07: Phase 4 Slice 4.6a moved base inventory response types plus load/increment/consume operations into `internal/hq/inventory`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.6a.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.6a.
- 2026-06-07: Phase 4 Slice 4.6b moved equipment load/equip/unequip operations into `internal/hq/inventory`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.6b.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.6b.
- 2026-06-07: Phase 4 Slice 4.6c moved hotbar load/set/default-seed operations into `internal/hq/inventory`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.6c.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.6c.
- 2026-06-07: Phase 4 Slice 4.6d moved crafting recipes and craft operation into `internal/hq/inventory`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.6d.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.6d.
- 2026-06-07: Phase 4 Slice 4.6e moved student wallet ensure and cookie shop purchase operations into `internal/hq/inventory`, leaving shared star reward ledger helpers for the Sunny Town bridge slice.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.6e.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.6e.
- 2026-06-07: Phase 4 Slice 4.7a moved assignment request/response types plus assignment load/filter helpers into `internal/hq/assignments`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.7a.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.7a.
- 2026-06-07: Phase 4 Slice 4.7b moved assignment grading command logic into `internal/hq/assignments` while preserving manual and AI grading call sites.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.7b.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.7b.
- 2026-06-07: Phase 4 Slice 4.7c moved assignment create, submit, reset, delete, and load-by-id service operations into `internal/hq/assignments`, leaving HTTP auth/JSON flow in `cmd/hq` for the next handler-factory slice.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.7c.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.7c.
- 2026-06-07: Phase 4 Slice 4.7d moved assignment HTTP handlers and route-ready handler factories into `internal/hq/assignments`, leaving `cmd/hq` with a narrow role-check adapter and route wiring.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.7d.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.7d.
- 2026-06-07: Phase 4 Slice 4.7e replaced the exported raw SQL suffix assignment loader with typed list operations and added direct package tests for `internal/hq/assignments` and `internal/hq/inventory`.
- 2026-06-07: `go test ./internal/hq/assignments ./internal/hq/inventory` passed after Phase 4 Slice 4.7e.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.7e.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.7e.
- 2026-06-07: Phase 4 Slice 4.8 moved Sunny Town internal bridge handlers, reward/resource commits, position persistence, map-object operations, and bridge ledger helpers into `internal/hq/sunnytownbridge`, leaving `cmd/hq` with compatibility adapters and route wiring.
- 2026-06-07: `go test ./internal/hq/sunnytownbridge` passed after Phase 4 Slice 4.8.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.8.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.8.
- 2026-06-07: Phase 4 Slice 4.9 moved AI grading internal callbacks, result persistence, auto-apply skip policy, and outbound AI grade requests into `internal/hq/ai`, leaving `cmd/hq` with compatibility adapters and route wiring.
- 2026-06-07: `go test ./internal/hq/ai` passed after Phase 4 Slice 4.9.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.9.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.9.
- 2026-06-07: Phase 4 Slice 4.10 documented HQ schema ownership boundaries in `docs/current/SCHEMA_OWNERSHIP.md` and linked the migration map from `docs/current/DATABASE.md`.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.10.
- 2026-06-07: Phase 4 Slice 4.11 split remaining HQ HTTP handlers into `cmd/hq/handlers.go` and shared HTTP/server helpers into `cmd/hq/http_helpers.go`, reducing `cmd/hq/main.go` to a thin 73-line binary entrypoint.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 4 Slice 4.11.
- 2026-06-07: `go test ./...` passed after Phase 4 Slice 4.11.
- 2026-06-07: Phase 5 started by replacing the single fresh-database init SQL file with ordered SQL migrations under `deploy/postgres/migrations/` and a Postgres init script that applies them for fresh Docker databases.
- 2026-06-07: Disposable Postgres migration smoke test passed with all 15 expected public tables, all 7 inventory item types, and all 6 AI grading columns on `assignment_attempt`.
- 2026-06-07: `go test ./...` passed after Phase 5 migration file creation.
- 2026-06-07: `docker compose -f deploy\docker-compose.yml config` passed after Phase 5 migration mount update.
- 2026-06-07: Phase 5 added `internal/hq/schema` as a lightweight startup migration runner, removed the old runtime `ensureSchema` patch list, and copied migration SQL into the HQ Docker image.
- 2026-06-07: `HQ_SCHEMA_TEST_DATABASE_URL=postgres://hq:hq@127.0.0.1:55543/hq?sslmode=disable go test ./internal/hq/schema -run TestRunMigrationsIntegration -count=1` passed against disposable Postgres, including a second idempotent migration run.
- 2026-06-07: Existing local Docker database migrated through HQ startup; `schema_migration` recorded `0001_initial` through `0005_ai_grading`, and HQ health returned `ok`.
- 2026-06-07: `go test ./...` passed after replacing runtime schema patching with the migration runner.
- 2026-06-07: `docker compose -f deploy\docker-compose.yml config` passed after replacing runtime schema patching with the migration runner.
- 2026-06-07: Phase 5 documented migration creation, applied-version inspection, disposable migration smoke tests, and Docker volume reset workflows in `docs/current/DATABASE.md`.
- 2026-06-07: Phase 6 added `.github/workflows/verify.yml` with jobs for `go test ./...`, `npm ci && npm run build` from `frontend/`, and `docker compose -f deploy/docker-compose.yml config`.
- 2026-06-07: Added Phase 7 adapter-retirement plan. First target is moving inventory/equipment/hotbar/crafting/shop HTTP handlers into `internal/hq/inventory`, then deleting the matching `cmd/hq` adapters once unused.
- 2026-06-07: Phase 7 Slices 7.1-7.4 moved inventory/equipment/hotbar/crafting/shop HTTP handlers into `internal/hq/inventory`, mounted them directly from `cmd/hq/routes.go`, added direct inventory HTTP handler tests, updated command tests to call `internal/hq/inventory` directly, and deleted `cmd/hq/inventory.go`, `cmd/hq/equipment.go`, `cmd/hq/hotbar.go`, `cmd/hq/crafting.go`, and `cmd/hq/shop.go`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slices 7.1-7.4.
- 2026-06-07: `go test ./...` passed after Phase 7 Slices 7.1-7.4.
- 2026-06-07: Phase 7 Slices 7.6-7.7 retired the pet backend adapter by wiring REST and Connect handlers directly to `internal/hq/pet.Store`, deleted `cmd/hq/pet_adapter.go`, kept only command auth/store construction glue, and added direct `internal/hq/pet` HTTP handler tests.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slices 7.6-7.7.
- 2026-06-07: `go test ./...` passed after Phase 7 Slices 7.6-7.7.
- 2026-06-07: Phase 7 Slice 7.8 retired the Sunny Town bridge adapter by wiring internal routes directly to `internal/hq/sunnytownbridge.NewHTTPHandler`, updating HQ tests to call `internal/hq/sunnytownbridge.Store` and helper APIs directly, and deleting `cmd/hq/sunnytown_bridge.go`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slice 7.8.
- 2026-06-07: `go test ./...` passed after Phase 7 Slice 7.8.
- 2026-06-07: Phase 7 Slice 7.9 retired AI grading adapters by wiring `/api/internal/ai/assignment-attempts/` directly to `internal/hq/ai.NewHandler`, updating tests to call `internal/hq/ai.RecordGradeResult` directly, moving async grade triggering into `internal/hq/ai.TriggerGradeAsync`, and deleting `cmd/hq/ai_grading.go`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slice 7.9.
- 2026-06-07: `go test ./...` passed after Phase 7 Slice 7.9.
- 2026-06-07: Phase 7 Slice 7.10 retired assignment compatibility adapters by moving transactional grading into `internal/hq/assignments.GradeAttemptInTx`, inlining the route role bridge, updating HQ tests to call assignment package APIs directly, and deleting `cmd/hq/assignment_grading.go` plus `cmd/hq/assignments.go`.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/assignments` passed after Phase 7 Slice 7.10.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slice 7.10.
- 2026-06-07: `go test ./...` passed after Phase 7 Slice 7.10.
- 2026-06-07: Phase 7 Slice 7.11 reassessed `cmd/hq` after adapter retirement. Conclusion: adapter retirement is complete, but moving app construction is still premature until command-local auth/user management moves behind internal package boundaries. Added Phase 8 plan for HQ command cleanup.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 7 Slice 7.11 planning update.
- 2026-06-07: `go test ./...` passed after Phase 7 Slice 7.11 planning update.
- 2026-06-07: Phase 8 Slice 8.1 moved Keycloak JWT verification, auth user types, context helpers, and role checks from `cmd/hq/auth.go` into `internal/hq/auth`; added direct auth package tests for context helpers, role filtering, missing bearer rejection, and signed JWT verification against a local JWKS server.
- 2026-06-07: `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth` passed after Phase 8 Slice 8.1.
- 2026-06-07: `go test ./...` passed after Phase 8 Slice 8.1.
