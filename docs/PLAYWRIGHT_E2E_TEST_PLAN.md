# Playwright E2E Test Implementation Plan

This document is a handoff prompt for a fresh Codex agent session. The goal is to add Playwright browser tests for the current HQ and Sunny Town product workflows, then iterate until the tests pass reliably against the local Docker Compose runtime.

The agent should treat this as an implementation task, not only a planning task.

## Objective

Build a Playwright end-to-end test suite that verifies the user-facing workflows described in:

- `docs/USER_STORIES_AND_REQUIREMENTS.md`
- `docs/TEST_CASE_COVERAGE_MATRIX.md`
- `docs/TESTING_GUIDELINES.md`
- `docs/SUNNY_TOWN_ARCHITECTURE.md`
- `docs/INVENTORY_AND_EQUIPMENT.md`

After each Playwright test is made to pass, compare the test behavior against the original requirement or test case it claims to cover. If the test is weaker than the requirement, either strengthen the test or explicitly document the remaining gap.

Do not use Playwright to duplicate low-level backend/unit coverage that is already better tested in Go or Vitest. The browser suite should focus on workflows that prove the deployed app, authentication, frontend UI, API calls, WebSocket connection, and cross-service behavior work together.

## Current Runtime Expectations

Prefer the container workflow from `docs/TESTING_GUIDELINES.md`.

If Go Task is available:

```powershell
task test
task frontend:build
task compose:rebuild-runtime
task health
```

If Go Task is not available:

```powershell
go test ./...
cd frontend
npm test
npm run build
cd ..
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected health response for both services is `ok`.

Useful URLs:

- HQ app: `http://127.0.0.1:18080`
- Keycloak: `http://127.0.0.1:18081`
- Sunny Town health: `http://127.0.0.1:18082/healthz`

Known local noise:

- `hq-local.err.log`
- `hq-local.out.log`

Do not stage those log files unless the user explicitly asks.

## Required Agent Workflow

For each Playwright test or small group of related tests:

1. Read the relevant requirement from `docs/USER_STORIES_AND_REQUIREMENTS.md`.
2. Check `docs/TEST_CASE_COVERAGE_MATRIX.md` for the corresponding test case ID and current coverage notes.
3. Implement the Playwright test using stable selectors where possible.
4. If selectors are missing or brittle, add small stable `data-testid` attributes to the app rather than relying on fragile CSS or text-only selectors.
5. Run the new Playwright test by itself until it passes.
6. Run the relevant local test group.
7. Compare the passing test back to the requirement:
   - What acceptance criteria are covered?
   - What acceptance criteria remain uncovered?
   - Is the uncovered behavior already covered by Go/Vitest?
   - If not covered anywhere, add a note to this document or the relevant coverage matrix.
8. Commit the completed slice only after tests pass.

Before each commit:

```powershell
git status --short
```

Stage files explicitly. Do not use `git add .`.

## Playwright Scope

Good Playwright targets:

- Real page loads and route refreshes.
- Login and role-based navigation.
- Student and teacher workflows.
- Visible assignment creation, submission, grading, reset, delete.
- Pet UI actions.
- Inventory UI actions.
- Sunny Town entry, canvas smoke checks, keyboard controls, overlays, NPC interactions, shop, schoolwork, and core gameplay happy paths.
- Cross-service flows where frontend, HQ, Sunny Town, and durable state must agree.

Poor Playwright targets:

- Map JSON validation.
- Exact movement collision math.
- JWT signature parsing internals.
- Service-secret endpoint internals.
- WebSocket malformed/oversized protocol edge cases.
- AI grading real API behavior.

Those are better covered by Go, Vitest, or future protocol-level integration tests.

## Setup Work

If Playwright is not already installed, add it in the least disruptive way.

Recommended location:

- `frontend/` if using the frontend package as the browser test home.
- Use a dedicated config such as `frontend/playwright.config.ts`.
- Use a test folder such as `frontend/e2e/`.

Recommended scripts in `frontend/package.json`:

```json
{
  "test:e2e": "playwright test",
  "test:e2e:headed": "playwright test --headed"
}
```

Prefer tests against the Docker-served app at `http://127.0.0.1:18080`, not the Vite dev server, because the current project preference is container runtime verification.

If authentication through Keycloak is difficult, first inspect existing setup docs and seed/demo users. Prefer real login if reliable. If login is too brittle for the first slice, document the blocker and add only unauthenticated smoke tests until a reliable auth helper is built.

## Requirement Mapping

Use these source requirement IDs when naming or annotating tests:

- `REQ-AUTH-001`: Keycloak authentication and role sync.
- `REQ-AUTH-002`: local-network host consistency.
- `REQ-RUNTIME-001`: health checks.
- `STORY-TCHR-001`: teacher creates assignments.
- `STORY-TCHR-002`: teacher reviews submitted work.
- `STORY-TCHR-003`: teacher grades work.
- `STORY-TCHR-004`: teacher resets an assignment attempt.
- `STORY-TCHR-005`: teacher deletes assignments.
- `STORY-STUD-001`: student answers next assignment.
- `STORY-STUD-002`: student sees reset attempt history.
- `STORY-STUD-003`: student reviews grades.
- `STORY-PET-001`: student views pet state.
- `STORY-PET-002`: student feeds pet.
- `STORY-PET-003`: student plays with pet.
- `STORY-PET-004`: pet sleep and recovery.
- `STORY-INV-001`: student views inventory.
- `STORY-INV-002`: student equips items.
- `STORY-INV-003`: starter equipment.
- `STORY-INV-004`: student buys shop item.
- `STORY-CRAFT-001`: student crafts stone block.
- `STORY-ST-001`: student starts Sunny Town session.
- `STORY-ST-002`: Sunny Town restores last position.
- `STORY-ST-MOVE-001`: local movement feel.
- `REQ-ST-MOVE-001`: server-accepted movement.
- `STORY-ST-MULTI-001`: nearby players.
- `REQ-ST-MAP-001`: load map JSON.
- `STORY-ST-PORTAL-001`: map portal transitions.
- `STORY-ST-STAR-001`: collect stars.
- `STORY-ST-NPC-001`: interact with NPCs.
- `STORY-ST-MINE-001`: mine resource nodes.
- `REQ-ST-MINE-001`: idempotent resource commit.
- `STORY-ST-BUILD-001`: place stone block.
- `STORY-ST-BUILD-002`: remove stone block.
- `REQ-WS-001`: supported client WebSocket messages.
- `REQ-WS-002`: supported server WebSocket messages.
- `REQ-FE-001`: app shape.
- `REQ-FE-002`: student inventory store.
- `REQ-FE-003`: equipment change broadcast.

## P0 Playwright Tests

These are the first tests to implement. They give broad coverage without immediately requiring difficult canvas automation or multiplayer setup.

| ID | Source requirement | Test case | Notes |
| --- | --- | --- | --- |
| `PW-SMOKE-001` | `REQ-FE-001`, `REQ-RUNTIME-001` | App loads at `/` with no console errors. | Verify title/app shell and no page errors. |
| `PW-SMOKE-002` | `REQ-FE-001` | Deep routes refresh correctly: `/student`, `/teacher`, `/teacher/login`, `/student/pet/sunny-town`. | Confirms SPA fallback through HQ static handler. |
| `PW-AUTH-001` | `REQ-AUTH-001` | Student can log in and lands on student dashboard. | Use real Keycloak login if possible. |
| `PW-AUTH-002` | `REQ-AUTH-001` | Teacher can log in and lands on teacher dashboard. | Use real Keycloak login if possible. |
| `PW-AUTH-003` | `REQ-AUTH-001` | Student cannot access teacher page. | Verify redirect or access-block behavior. |
| `PW-AUTH-004` | `REQ-AUTH-001`, `STORY-ST-001` | Teacher cannot access student-only Sunny Town/session UI. | Role boundary visible in browser. |
| `PW-SCHOOL-001` | `STORY-TCHR-001` | Teacher creates an assignment. | Use unique prompt text per run. |
| `PW-SCHOOL-002` | `STORY-STUD-001` | Student sees the next unanswered assignment. | Should find the assignment created by the teacher. |
| `PW-SCHOOL-003` | `STORY-STUD-001` | Student submits an answer. | Verify submitted state or next-assignment transition. |
| `PW-SCHOOL-004` | `STORY-TCHR-002` | Teacher sees submitted answer needing review. | Cross-role workflow. |
| `PW-SCHOOL-005` | `STORY-TCHR-003` | Teacher grades answer as passed with feedback. | Verify grade action succeeds. |
| `PW-SCHOOL-006` | `STORY-STUD-003` | Student sees passed grade and feedback. | Verify result is visible in student grades/history. |
| `PW-SCHOOL-007` | `STORY-TCHR-004` | Teacher resets an assignment attempt. | Use a submitted/graded or submitted active attempt as appropriate for UI. |
| `PW-SCHOOL-008` | `STORY-STUD-002` | Student sees reset assignment answerable again with prior history/feedback. | Compare closely to reset history acceptance criteria. |
| `PW-SCHOOL-009` | `STORY-TCHR-005` | Teacher deletes assignment and it disappears from teacher/student lists. | Use assignment created during test setup. |

## Sunny Town Playwright Tests

Implement after the initial auth and schoolwork suite is stable.

| ID | Source requirement | Test case | Notes |
| --- | --- | --- | --- |
| `PW-ST-001` | `STORY-ST-001` | Student enters Sunny Town from the student page. | Verify session creation and route/page load. |
| `PW-ST-002` | `STORY-ST-001`, `REQ-ST-MAP-001` | Sunny Town canvas/map renders after join. | Use canvas nonblank pixel check if practical. |
| `PW-ST-003` | `STORY-ST-MOVE-001` | WASD movement changes visible player/camera state. | Prefer user-level keyboard input. |
| `PW-ST-004` | `STORY-ST-MOVE-001` | Arrow-key movement works. | Alternate control path. |
| `PW-ST-005` | `STORY-INV-001` | Pressing `E` opens and closes inventory overlay. | Stable and high-value keyboard test. |
| `PW-ST-006` | `STORY-ST-MOVE-001`, `STORY-INV-001` | Pressing `E` inventory toggle does not permanently break movement. | Regression-prone UX. |
| `PW-ST-007` | `STORY-ST-PORTAL-001` | Student walks through house portal and receives new map view. | May require test helper positioning if keyboard travel is slow. |
| `PW-ST-008` | `STORY-ST-PORTAL-001` | Student walks back from house to outdoor map. | Return path. |
| `PW-ST-009` | `STORY-ST-PORTAL-001` | Student walks to forest crossing and map changes. | Forest path. |
| `PW-ST-010` | `STORY-ST-NPC-001` | Nearby NPC interaction opens dialogue with `F`. | Prefer a deterministic nearby NPC or helper setup. |
| `PW-ST-011` | `STORY-ST-NPC-001` | NPC dialogue advances and closes. | Overlay behavior. |
| `PW-ST-012` | `STORY-ST-NPC-001`, `STORY-STUD-001` | Schoolwork NPC opens assignment panel. | Sunny Town to schoolwork integration. |
| `PW-ST-013` | `STORY-ST-NPC-001`, `STORY-STUD-001` | Student submits schoolwork from Sunny Town panel. | In-world schoolwork flow. |
| `PW-ST-014` | `STORY-ST-NPC-001`, `STORY-INV-004` | Shop NPC opens shop panel. | Sunny Town to shop integration. |
| `PW-ST-015` | `STORY-INV-004` | Student buys cookie from shop and inventory/star balance updates. | Durable HQ change visible in UI. |
| `PW-ST-016` | `STORY-INV-002`, `REQ-FE-003` | Equip pickaxe from Sunny Town inventory. | Should commit through HQ before broadcast. |
| `PW-ST-017` | `STORY-ST-MINE-001`, `REQ-ST-MINE-001` | Mining resource node with pickaxe eventually updates inventory. | Full happy path. |
| `PW-ST-018` | `STORY-ST-MINE-001` | Mining without pickaxe shows/fails gracefully. | User-facing failure path. |
| `PW-ST-019` | `STORY-CRAFT-001` | Craft stone block from rocks. | May require resource setup. |
| `PW-ST-020` | `STORY-ST-BUILD-001` | Place stone block on valid tile. | Placement UI + server/HQ. |
| `PW-ST-021` | `STORY-ST-BUILD-001`, `REQ-ST-MOVE-001` | Placed stone block blocks movement. | Visible gameplay effect. |
| `PW-ST-022` | `STORY-ST-BUILD-002` | Mine placed stone block and inventory is refunded. | Removal flow. |
| `PW-ST-023` | `STORY-ST-002` | Re-enter Sunny Town and last position/map is restored. | Persistence integration. |

## Inventory And Pet Playwright Tests

| ID | Source requirement | Test case | Notes |
| --- | --- | --- | --- |
| `PW-INV-001` | `STORY-INV-001` | Student opens Pet page inventory dialog. | Visible inventory path. |
| `PW-INV-002` | `STORY-INV-001`, `STORY-PET-001` | Student Pet inventory cookie count matches Sunny Town inventory cookie count. | Cross-view consistency. |
| `PW-INV-003` | `STORY-INV-002` | Equipment appears with equip status. | UI presentation. |
| `PW-INV-004` | `STORY-INV-002` | Equip item and refresh; equipped state persists. | Browser persistence. |
| `PW-INV-005` | `STORY-INV-002` | Unequip item and refresh; state persists. | Browser persistence. |
| `PW-PET-001` | `STORY-PET-001` | Student pet profile loads hunger, happiness, energy, sleep state, and cookies. | Pet UI smoke. |
| `PW-PET-002` | `STORY-PET-002` | Feed pet consumes one cookie and updates hunger/cookie count. | User-visible pet action. |
| `PW-PET-003` | `STORY-PET-003` | Play pet updates happiness/energy. | User-visible pet action. |
| `PW-PET-004` | `STORY-PET-004` | Put pet to sleep updates sleeping UI. | User-visible pet action. |
| `PW-PET-005` | `STORY-PET-004` | Wake pet updates sleeping UI/state. | User-visible pet action. |
| `PW-PET-006` | `STORY-PET-001` | Falling Stars game can start, complete, and apply reward. | Game UI integration. |

## Multiplayer Playwright Tests

These are valuable but should wait until single-user Playwright auth/session helpers are stable. They likely require two authenticated browser contexts and two seeded student users.

| ID | Source requirement | Test case | Notes |
| --- | --- | --- | --- |
| `PW-MULTI-001` | `STORY-ST-MULTI-001` | Two students in the same Sunny Town map see each other. | Requires two student accounts. |
| `PW-MULTI-002` | `STORY-ST-MULTI-001`, `REQ-FE-003` | Student A changes equipment; Student B sees updated equipment. | Tests equipment broadcast. |
| `PW-MULTI-003` | `STORY-ST-BUILD-001` | Student A places a stone block; Student B sees it appear. | Same-map broadcast. |
| `PW-MULTI-004` | `STORY-ST-BUILD-002` | Student A removes a stone block; Student B sees it disappear. | Same-map broadcast. |
| `PW-MULTI-005` | `STORY-ST-MULTI-001` | Students in different maps do not see each other. | Map scoping. |

## Lower Priority Or Non-Playwright Cases

Keep these out of the initial Playwright suite unless there is a specific reason to cover the user-visible behavior in browser.

| Case family | Better test type | Reason |
| --- | --- | --- |
| `TC-ST-MAP-*` map JSON validation | Go tests | Startup validation is deterministic and non-UI. |
| `TC-ST-MOVE-*` exact server movement validation | Go tests | Collision and sequence math are easier and less flaky in server tests. |
| `TC-ST-MINE-*` cooldown/range/inactive-node details | Go tests | Browser should cover one happy path and one failure path. |
| `TC-ST-WS-*` unknown/oversized WebSocket messages | Go or WebSocket integration tests | Protocol edge cases do not need browser UI. |
| `TC-AUTH-*` bearer token parsing/signature details | Go tests | Backend auth internals. |
| `TC-SVC-*` service-secret internal endpoint behavior | Go tests | Internal service boundary. |
| AI grading real API behavior | Deferred | User explicitly deferred AI grading until real API path is validated. |

## Recommended First Implementation Slice

Build this minimal suite first:

1. `PW-SMOKE-001`
2. `PW-SMOKE-002`
3. `PW-AUTH-001`
4. `PW-AUTH-002`
5. `PW-SCHOOL-001`
6. `PW-SCHOOL-002`
7. `PW-SCHOOL-003`
8. `PW-SCHOOL-004`
9. `PW-SCHOOL-005`
10. `PW-SCHOOL-006`
11. `PW-ST-001`
12. `PW-ST-002`
13. `PW-ST-005`
14. `PW-ST-010`
15. `PW-INV-001`
16. `PW-PET-001`

This proves route/auth/schoolwork/Sunny Town/pet coverage without immediately depending on precise canvas travel, multiplayer, or long resource setup.

After that first slice is stable, add:

1. `PW-SCHOOL-007`
2. `PW-SCHOOL-008`
3. `PW-SCHOOL-009`
4. `PW-ST-012`
5. `PW-ST-013`
6. `PW-ST-014`
7. `PW-ST-015`
8. `PW-INV-004`
9. `PW-INV-005`
10. `PW-PET-002`
11. `PW-PET-003`
12. `PW-PET-004`
13. `PW-PET-005`

Then add Sunny Town advanced gameplay and multiplayer tests.

## Test Data Strategy

Use deterministic, unique data per test run.

Suggested assignment prompt pattern:

```text
Playwright ${testRunId} ${testCaseId}
```

Use cleanup through the UI when testing delete behavior. For tests that create durable state but do not naturally clean it up, prefer one of:

- a test-only backend setup/cleanup helper if the project already has one,
- unique names and category filters to avoid collisions,
- a documented cleanup step in the test fixture.

Do not add broad production-only reset endpoints just for tests without user approval. If a test helper endpoint is needed, keep it test-only and document the tradeoff.

## Selector Strategy

Prefer stable selectors in this order:

1. `data-testid` for important workflow controls.
2. Accessible role and name when the label is stable and user-facing.
3. Text locators only for durable visible text.
4. CSS structure selectors only as a last resort.

If adding `data-testid` attributes, keep them small and specific:

- `student-dashboard`
- `teacher-dashboard`
- `assignment-create-form`
- `assignment-prompt-input`
- `assignment-submit-button`
- `sunny-town-canvas`
- `sunny-town-inventory-panel`
- `sunny-town-dialogue`
- `pet-profile`

## Requirement Comparison Template

After each completed test, add or update a short note near the test file or in this document:

```text
Test: PW-SCHOOL-005
Requirement: STORY-TCHR-003
Covered:
- Teacher can mark latest active attempt as passed.
- Teacher can save feedback.
Also covered elsewhere:
- Cookie idempotency is covered by Go tests.
Remaining gap:
- This browser test does not re-save the same passing result to prove no duplicate cookie award.
Decision:
- Leave duplicate-cookie idempotency to Go coverage unless a UI regression appears.
```

The goal is to avoid false confidence. A passing browser test should clearly state which parts of the requirement it proves.

## Verification Before Final Handoff

At the end of each implementation slice, run:

```powershell
git status --short
go test ./...
cd frontend
npm test
npm run build
npm run test:e2e
cd ..
docker compose -f deploy\docker-compose.yml ps
```

If Playwright depends on the container runtime, make sure the containers are rebuilt before the final full e2e run:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
```

Then check:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected output for both is:

```text
ok
```

## Completion Criteria

The task is complete when:

- Playwright is installed/configured or an existing Playwright setup is extended.
- The first implementation slice passes reliably.
- Each implemented Playwright test maps to a requirement/test case ID.
- Each implemented Playwright test has a short requirement comparison note.
- Any remaining gaps are documented.
- Backend tests, frontend tests/build, and Playwright tests pass.
- Docker Compose runtime can rebuild and health checks pass.
- Changes are committed in focused slices with local logs left untracked.
