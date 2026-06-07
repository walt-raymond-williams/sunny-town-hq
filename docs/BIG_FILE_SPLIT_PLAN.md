# Big File Split Plan

This plan tracks the next maintainability pass after the repository restructure. The broad package boundaries are now good enough; this work is about making the largest remaining files easier for humans and AI agents to edit safely.

## Resume Snapshot

Last updated: 2026-06-07.

Current state:

- The broad restructure is complete in `docs/RESTRUCTURE_IMPLEMENTATION_PLAN.md`.
- Do not move `cmd/hq` app construction or route registration unless new code pain appears. Those files now act as binary composition glue.
- Known untracked local files should remain uncommitted unless the user explicitly asks:
  - `hq-local.err.log`
  - `hq-local.out.log`
- Slice 1 is complete: `internal/hq/sunnytownbridge/bridge.go` was split into focused production files.
- Slice 2 is in progress: `frontend/src/features/sunny-town/SunnyTownPage.vue` is down from 1,554 lines to about 831 lines.
- Slice 2 completed sub-slices:
  - `frontend/src/features/sunny-town/worldObjects.ts` plus tests for world-object conversion helpers.
  - `frontend/src/composables/useSunnyTownRemotePlayers.ts` plus tests for remote snapshot history/interpolation.
  - `frontend/src/composables/useSunnyTownLocalPlayer.ts` plus tests for predicted/rendered local-player state.
  - `frontend/src/composables/useSunnyTownToolUseAnimation.ts` plus tests for tool-use animation progress.
  - `frontend/src/composables/useSunnyTownNpcInteractions.ts` plus tests for dialogue/shop/schoolwork overlay state and nearest-NPC selection.
  - `frontend/src/composables/useSunnyTownPlacement.ts` plus tests for placement hover state, pointer-to-grid conversion, and placement validation.
  - `frontend/src/features/sunny-town/rendering/` draw helpers plus tests for pure facing-vector logic.
  - `frontend/src/composables/useSunnyTownWorldState.ts` plus tests for map normalization, snapshots, and placed-object state updates.
- Next recommended Slice 2 sub-step: extract inventory/hotbar action handling from `SunnyTownPage.vue` into a focused composable, while leaving socket sends and UI panel wiring in the page until the boundaries are clearer.
- AI grading file cleanup should wait if another agent is actively refactoring the AI service.

Verification baseline:

```powershell
go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth
go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth
go test ./...
cd frontend
npm test
npm run build
```

Run the narrow command first for the area touched, then run `go test ./...` or `npm test`/`npm run build` before committing. For mixed backend/frontend slices, run both backend and frontend checks.

Before committing any Go production-code change, confirm that the touched package has meaningful unit tests. If it does not, add focused unit tests for the moved or changed behavior in the same slice, then run them before committing. Existing broad integration coverage is useful, but it is not a substitute for package-level unit coverage when a package has no direct tests.

## Goals

- Make the largest files easier to review, merge, and assign to different people or agents.
- Preserve the package boundaries created by the restructure.
- Split by ownership and behavior, not arbitrary line count.
- Consolidate duplicated helpers only when behavior is actually the same.
- Keep every slice behavior-preserving unless the user explicitly asks for product behavior changes.

## Non-Goals

- Do not chase tiny files just to satisfy a line-count target.
- Do not create shared helpers for code that only looks similar but has different domain error semantics.
- Do not move app construction out of `cmd/hq` as part of this plan.
- Do not mechanically split tests into many files unless the split makes ownership clearer.
- Do not touch ignored generated `web/` assets or local logs.

## Shared-Code Policy

Prefer one shared implementation when two call sites are truly doing the same thing. Examples worth checking during each slice:

- JSON response writers in HQ HTTP packages.
- HTTP method checks and request-body decode errors.
- Service-secret authentication helpers for internal service endpoints.
- Test fixtures for repeated DB setup or app construction.
- Sunny Town object conversion helpers used by both rendering and interaction logic.

Rules:

- If the status code, response body, logging behavior, or caller contract differs, keep the logic local or create a narrow helper with explicit options.
- Prefer package-local helpers before creating cross-domain shared packages.
- For shared HTTP helpers, prefer `internal/hq/httpapi` only when the helper is domain-neutral.
- For frontend shared logic, prefer composables for stateful behavior and pure utility modules for stateless transforms.

## Current Large Files

Snapshot generated on 2026-06-07:

| File | Lines | Priority | Notes |
| --- | ---: | --- | --- |
| `frontend/src/features/sunny-town/SunnyTownPage.vue` | 831 | High | Still holds orchestration, WebSocket side effects, inventory/hotbar actions, shop/schoolwork commands, and camera/input routing. |
| `cmd/hq/reward_test.go` | 1,358 | Medium | Mixed regression tests for Sunny Town bridge, pet, inventory/shop/crafting/equipment, hotbar, and AI grading. Useful but broad. |
| `internal/sunnytown/server/room_test.go` | 974 | Medium | Mixed room/world tests for movement, collisions, portals, mining, map validation, NPCs, and fixtures. |
| `internal/hq/sunnytownbridge/bridge.go` | 848 | High | Store operations, ledgers, positions, map objects, service HTTP handlers, auth, parsing, and JSON helpers in one file. |
| `internal/hq/assignments/http.go` | 517 | Medium | Teacher and student handlers plus helper parsing/JSON in one file. |
| `internal/hq/pet/store.go` | 488 | Medium | Profile actions, game rewards, sleep/wake, decay ticker, decay math, and profile loading in one file. |
| `internal/hq/ai/grading.go` | 460 | Low for now | Handler, context loading, grade recording, auto-apply policy, outbound request, async trigger, and JSON helper. Defer if AI refactor is active. |
| `internal/hq/auth/auth.go` | 370 | Low | JWT verification, context helpers, role checks. Large but cohesive enough for now. |
| `internal/sunnytown/hqclient/client.go` | 346 | Low/Medium | Multiple internal HQ client endpoints in one client file. Split only if endpoint groups grow. |
| `internal/hq/assignments/assignments.go` | 316 | Low/Medium | Assignment DTOs and typed list/load operations. Watch for future growth. |
| `internal/sunnytown/server/client_gameplay.go` | 304 | Low/Medium | Gameplay message handlers; split if placement/mining/interaction grows. |
| `internal/hq/inventory/http.go` | 297 | Low/Medium | Inventory, hotbar, crafting, equipment, shop handlers. Could split if handler edits collide. |

## Recommended Slice Order

- [x] Slice 1: Split `internal/hq/sunnytownbridge/bridge.go`.
- [ ] Slice 2: Split `frontend/src/features/sunny-town/SunnyTownPage.vue`.
- [ ] Slice 3: Split `cmd/hq/reward_test.go` by package ownership.
- [ ] Slice 4: Split `internal/sunnytown/server/room_test.go` by behavior area.
- [ ] Slice 5: Split `internal/hq/assignments/http.go`.
- [ ] Slice 6: Split `internal/hq/pet/store.go`.
- [ ] Slice 7: Reassess `internal/hq/ai/grading.go` after AI refactor status is clear.
- [ ] Slice 8: Reassess lower-priority files and duplicate helper opportunities.

## Slice 1: HQ Sunny Town Bridge

Target file: `internal/hq/sunnytownbridge/bridge.go`.

Why it matters:

- This is an important cross-service boundary between Sunny Town and HQ.
- It currently mixes persistence, ledger idempotency, map-object ownership, HTTP transport, service auth, and response formatting.
- Splitting it will let one agent work on map objects while another works on reward/resource ledger behavior with fewer merge conflicts.

Suggested target files:

```text
internal/hq/sunnytownbridge/
  types.go          request/response structs and exported errors/constants
  store.go          Store type plus high-level CommitReward, CommitResource, Load/SavePosition
  ledger.go         CommitStudentStarReward, CommitStudentInventoryLedgerDelta, LoadStudentInventoryQuantity
  map_objects.go    LoadMapObjects, PlaceMapObject, RemoveMapObject, map-object error mapping
  http.go           HTTPHandler, NewHTTPHandler, service endpoint handlers
  http_helpers.go   authorized, parsePositiveIntQuery, package-local JSON helpers if not shared
  bridge_test.go    keep initially; split tests only after production split is stable
```

Shared-code check:

- Consider replacing package-local `writeJSON` with `httpapi.WriteJSON` if response behavior stays identical.
- Keep `StatusForMapObjectError` and `MapObjectErrorMessage` in the bridge package because they encode domain semantics.
- Keep service-secret auth local unless another internal-service package needs the exact same header behavior.

Verification:

```powershell
go test ./internal/hq/sunnytownbridge
go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth
go test ./...
```

Completion checklist:

- [x] Production code split with no behavior changes.
- [x] Package tests still pass.
- [x] Full Go tests pass.
- [x] Tracking log updated.
- [x] Commit made with local logs left untracked.

## Slice 2: Sunny Town Page

Target file: `frontend/src/features/sunny-town/SunnyTownPage.vue`.

Why it matters:

- It is still the largest product file.
- The previous frontend split extracted major UI panels, but the page still owns many state machines.
- Frontend agents are likely to collide here when changing gameplay interaction, rendering, NPCs, inventory, shop, or schoolwork.

Current responsibility groups:

- WebSocket server-message handling and map-state application.
- Keyboard/pointer input routing.
- Inventory, equipment, crafting, hotbar, and placement actions.
- NPC interaction, dialogue, shop, and schoolwork commands.
- Local prediction and remote-player interpolation.
- Rendering helpers still inside the page despite `useSunnyTownRenderer`.
- Small utilities such as camera, grid conversion, error messages, and clamp.

Suggested target shape:

```text
frontend/src/features/sunny-town/SunnyTownPage.vue
frontend/src/composables/useSunnyTownWorldState.ts
frontend/src/composables/useSunnyTownInteractions.ts
frontend/src/composables/useSunnyTownNpcInteractions.ts
frontend/src/composables/useSunnyTownRemotePlayers.ts
frontend/src/composables/useSunnyTownPlacement.ts
frontend/src/features/sunny-town/rendering/
  mapDrawing.ts
  playerDrawing.ts
  objectDrawing.ts
```

Slice guidance:

- Start with pure/stateless helpers first, such as remote-player interpolation or object conversion.
- Move one concern at a time; keep `SunnyTownPage.vue` as orchestrator.
- Do not create one giant composable that simply moves the problem elsewhere.
- After each extraction, check that props/events passed to child components remain understandable.

Shared-code check:

- Reuse existing `useSunnyTownMovement`, `useSunnyTownRenderer`, and `types/sunnyTown.ts`.
- If renderer helpers are duplicated between page and composable, prefer moving pure draw helpers under a `rendering/` folder.
- If NPC dialogue/shop/schoolwork share overlay-close behavior, centralize it in an NPC interaction composable only if it makes state ownership clearer.

Verification:

```powershell
cd frontend
npm run build
```

When practical after a meaningful frontend slice, smoke-test the app served through Docker/HQ at `http://127.0.0.1:18080`.

## Slice 3: HQ Regression Tests

Target file: `cmd/hq/reward_test.go`.

Why it matters:

- It is valuable regression coverage, but it now spans several packages whose production code moved to `internal/hq/...`.
- Leaving all of it in `cmd/hq` makes test ownership less obvious.

Suggested split:

```text
cmd/hq/reward_test.go                         keep command-level route/session integration tests
internal/hq/sunnytownbridge/bridge_test.go    Sunny Town reward/resource/position/map-object tests
internal/hq/pet/store_test.go                 pet feed/game-result/sleep/decay store tests
internal/hq/inventory/store_test.go           shop/crafting/equipment/hotbar integration-ish tests
internal/hq/ai/grading_test.go                AI grade result edge cases already partly covered here
```

Guidance:

- Do not move tests mechanically. Move a test only when it can use the package under test directly without needing command-only setup.
- Keep true end-to-end command route tests in `cmd/hq`.
- Extract shared DB fixtures only if multiple packages need the exact same setup.

Verification:

```powershell
go test ./cmd/hq ./internal/hq/...
go test ./...
```

## Slice 4: Sunny Town Server Tests

Target file: `internal/sunnytown/server/room_test.go`.

Why it matters:

- It is good coverage, but it combines several behavior domains.
- Splitting test files makes it easier to find and extend the relevant tests.

Suggested target files:

```text
internal/sunnytown/server/room_lifecycle_test.go
internal/sunnytown/server/movement_test.go
internal/sunnytown/server/portal_test.go
internal/sunnytown/server/collectibles_test.go
internal/sunnytown/server/mining_test.go
internal/sunnytown/server/map_validation_test.go
internal/sunnytown/server/npc_test.go
internal/sunnytown/server/test_fixtures_test.go
```

Guidance:

- Start with test-only file moves. Avoid production changes in the same commit.
- Put common fixtures in `test_fixtures_test.go`.
- Keep package name unchanged.

Verification:

```powershell
go test ./internal/sunnytown/server
go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth
go test ./...
```

## Slice 5: Assignment HTTP Handlers

Target file: `internal/hq/assignments/http.go`.

Why it matters:

- Teacher and student HTTP flows live together.
- Assignment route behavior is important and likely to change as grading UX evolves.

Suggested target files:

```text
internal/hq/assignments/http.go              Handler type and constructor
internal/hq/assignments/http_student.go      next assignment, graded assignments, submit
internal/hq/assignments/http_teacher.go      list, create, answered, grade, reset, delete
internal/hq/assignments/http_helpers.go      parseAssignmentID, category validation, package JSON helper
```

Shared-code check:

- Evaluate whether package-local `writeJSON` can use `internal/hq/httpapi.WriteJSON`.
- Keep assignment-specific parsing and validation local.

Verification:

```powershell
go test ./internal/hq/assignments
go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth
go test ./...
```

## Slice 6: Pet Store

Target file: `internal/hq/pet/store.go`.

Why it matters:

- Pet is intentionally its own HQ domain boundary.
- The store file currently mixes actions, game rewards, decay, ticker orchestration, and profile loading.

Suggested target files:

```text
internal/hq/pet/store.go             Store type and constructor-adjacent helpers
internal/hq/pet/profile_store.go     LoadProfile and loadProfileWithoutDecay
internal/hq/pet/actions.go           Feed, Play, PutToSleep, Wake
internal/hq/pet/game_rewards.go      ApplyGameResult and star reward integration
internal/hq/pet/decay.go             StartDecayTicker, ApplyDecayForAll, ApplyDecay
```

Shared-code check:

- Keep inventory and reward callbacks explicit. Do not hide pet's dependencies behind a broad app object.
- If transaction patterns duplicate in multiple pet methods, consider a package-local transaction helper.

Verification:

```powershell
go test ./internal/hq/pet
go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth
go test ./...
```

## Slice 7: AI Grading

Target file: `internal/hq/ai/grading.go`.

Status:

- Defer until active AI-service refactor status is clear.
- This file is large, but it may be moving under another agent's work.

Possible future split:

```text
internal/hq/ai/http.go
internal/hq/ai/context.go
internal/hq/ai/results.go
internal/hq/ai/auto_apply.go
internal/hq/ai/client.go
internal/hq/ai/async.go
```

Shared-code check:

- `writeJSON` could use `internal/hq/httpapi.WriteJSON` if the package keeps the same response behavior.
- Keep AI skip-policy logic separately testable.

## Slice 8: Lower-Priority Reassessment

Files to revisit after high-priority slices:

- `internal/hq/auth/auth.go`
- `internal/sunnytown/hqclient/client.go`
- `internal/hq/assignments/assignments.go`
- `internal/sunnytown/server/client_gameplay.go`
- `internal/hq/inventory/http.go`

Do not split these unless there is clear edit pain, repeated conflicts, or an obvious ownership boundary.

## Working Rules For Each Slice

1. Start with `git status --short`; only known local logs should be untracked.
2. Read the target file and nearby tests before editing.
3. Move code in one behavior-preserving slice.
4. Prefer file moves/splits over API redesign.
5. Consolidate duplicate helpers only when behavior is identical and tests cover it.
6. For Go production changes, verify package-level unit coverage exists for the touched behavior; add focused unit tests if none exist.
7. Run narrow tests first, then broad tests, before committing.
8. Update this document with completed work, verification, and next slice.
9. Stage files explicitly. Do not stage `hq-local.err.log`, `hq-local.out.log`, ignored `web/`, or unrelated changes.
10. Commit every completed slice only after the relevant tests pass.

## Current Verification Log

- 2026-06-07: Created this plan after closing broad restructure. No code changed.
- 2026-06-07: Completed Slice 1 by splitting `internal/hq/sunnytownbridge/bridge.go` into types, store, ledger, map-object, HTTP handler, and HTTP helper files. Verified with `go test ./internal/hq/sunnytownbridge`, `go test ./cmd/hq ./internal/hq/... ./internal/serviceauth ./internal/sunnytownauth`, and `go test ./...`.
- 2026-06-07: Started Slice 2 by extracting Sunny Town world-object conversion helpers from `SunnyTownPage.vue` into `frontend/src/features/sunny-town/worldObjects.ts`, adding Vitest frontend unit tests in `worldObjects.test.ts`, and adding `npm test`. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting remote-player snapshot history and interpolation into `frontend/src/composables/useSunnyTownRemotePlayers.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting predicted/rendered local-player state into `frontend/src/composables/useSunnyTownLocalPlayer.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting Sunny Town tool-use animation state into `frontend/src/composables/useSunnyTownToolUseAnimation.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting NPC dialogue/shop/schoolwork overlay state and nearest-NPC selection into `frontend/src/composables/useSunnyTownNpcInteractions.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting Sunny Town placement hover state, pointer-to-grid conversion, and placement validation into `frontend/src/composables/useSunnyTownPlacement.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting Sunny Town canvas rendering helpers into `frontend/src/features/sunny-town/rendering/` with a focused unit test for facing-vector logic. Verified with `npm test` and `npm run build` from `frontend/`.
- 2026-06-07: Continued Slice 2 by extracting Sunny Town world-state refs and message-state reducers into `frontend/src/composables/useSunnyTownWorldState.ts` with focused unit tests. Verified with `npm test` and `npm run build` from `frontend/`.
