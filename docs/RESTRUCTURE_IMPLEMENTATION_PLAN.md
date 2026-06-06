# Restructure Implementation Plan

This plan tracks the repository restructure work identified in the project review. It is intentionally split so Sunny Town and general collaboration improvements can proceed while AI service work happens separately.

## Coordination Rules

- Do not touch AI service implementation while another agent is refactoring it.
- Avoid AI integration files until the AI work lands:
  - `cmd/ai/main.go`
  - `internal/aiapi/grading.go`
  - `cmd/hq/ai_grading.go`
  - AI-related sections of `cmd/hq/schema.go`
  - AI env vars in `deploy/docker-compose.yml`
  - `docs/AI_SERVICE_IMPLEMENTATION_PLAN.md`
  - `docs/AI_GRADING_HARDENING_PLAN.md`
- Avoid broad HQ backend movement until AI/HQ integration changes are known.
- Keep each phase behavior-preserving unless the task explicitly says otherwise.
- Run scoped verification after each phase and record results in this file.

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
- [ ] Confirm `git status --short` contains only intentional docs/task-runner changes plus known local logs and known concurrent AI edits.

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
internal/sunnytown/world/
```

Tasks:

- [x] Move config loading into `internal/sunnytown/config`.
- [x] Move map structs, loading, and validation into `internal/sunnytown/maps`.
- [x] Move WebSocket message structs into `internal/sunnytown/protocol`.
- [ ] Move room, player, world, movement, portal, collectible, resource, and placement logic into `internal/sunnytown/world`.
- [x] Split Sunny Town world data types into `cmd/sunny-town/world_types.go` as a preparatory step.
- [x] Split Sunny Town world lifecycle methods into `cmd/sunny-town/world_lifecycle.go` as a preparatory step.
- [x] Split Sunny Town movement, collision, clamping, and portal helpers into `cmd/sunny-town/world_movement.go` as a preparatory step.
- [x] Split Sunny Town snapshot and broadcast helpers into `cmd/sunny-town/world_snapshots.go` as a preparatory step.
- [x] Move internal HQ HTTP calls into `internal/sunnytown/hqclient`.
- [ ] Move WebSocket server setup and request handling into `internal/sunnytown/server`.
- [x] Split Sunny Town server/WebSocket wiring into `cmd/sunny-town/server.go` as a preparatory step.
- [ ] Update Sunny Town tests to import/use the new packages.

Verification:

- [x] `go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth`
- [ ] `go test ./...`

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

- [ ] Extract Sunny Town socket/session lifecycle.
- [ ] Extract movement input and client prediction.
- [ ] Extract canvas rendering.
- [ ] Extract HUD and status UI.
- [ ] Extract inventory and crafting panels.
- [ ] Extract dialogue and shop panels.
- [ ] Extract schoolwork panel.
- [ ] Update router import if the page moves from `pages/` to `features/sunny-town/`.

Verification:

- [ ] `cd frontend && npm run build`
- [ ] Smoke-test Sunny Town in browser or container-served app when practical.

## Phase 4: HQ Backend Split

Goal: reduce `cmd/hq/main.go` after AI service work lands.

Wait for the AI refactor before starting this phase.

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
- [ ] Move AI integration after reconciling with the other agent's refactor.
- [ ] Keep `cmd/hq/main.go` as a thin binary entrypoint.

Verification:

- [ ] `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`
- [ ] `go test ./...`

## Phase 5: Database Migration Story

Goal: replace mixed Docker init SQL plus runtime schema patching with explicit migrations.

Wait for AI schema changes to settle before starting this phase.

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
