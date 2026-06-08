# Technical Debt Tracker

This file tracks known rough edges that are worth addressing after the current feature work settles. Keep entries concrete: what hurts, why it matters, the likely next move, and how to verify the cleanup.

Last updated: 2026-06-08.

## Review Posture

The project does not currently read like AI-generated slop. The core architecture has a clear spine: HQ owns durable state, Sunny Town owns live realtime state, durable effects cross through service-authenticated internal APIs, and important mutation paths are covered by tests.

The items below are normal fast-iteration debt from an AI-assisted build. Address them when they create edit pain, review risk, runtime risk, or resume/demo friction.

## Active Debt

### AI grading remains prototype-only

Current state:

- The AI service defaults to the fake provider.
- Non-fake provider handling is explicitly not implemented.
- Existing docs already mark AI grading as deferred until the real integration direction is clearer.

Why it matters:

- It is fine for local development, but it should not be presented as a production-ready AI grading integration.
- Future changes may need different boundaries once the real provider, prompt contracts, retry policy, and failure model are known.

Likely next action:

- Keep fake mode for deterministic local tests.
- Add a real provider behind the existing service boundary only after defining request/response contracts, error handling, retry behavior, and observability.
- Split `internal/hq/ai/grading.go` only after real-provider behavior is known.

Verification:

```powershell
go test ./cmd/ai ./internal/hq/ai ./internal/aiapi
go test ./...
```

### Sunny Town page is still a large orchestrator

Current state:

- `frontend/src/features/sunny-town/SunnyTownPage.vue` has already been reduced substantially.
- It still owns a lot of orchestration for socket ordering, movement send, camera/input routing, and rendering composition.

Why it matters:

- Large Vue pages are prone to accidental coupling between input, networking, rendering, inventory, and NPC interactions.
- Future feature work may create merge conflicts or make AI-assisted edits harder to review.

Likely next action:

- Do not split just for line count.
- Extract only when a responsibility starts changing independently, such as camera/input routing, socket message sequencing, or render orchestration.
- Keep the page as an explicit composition layer instead of moving everything into one giant composable.

Verification:

```powershell
cd frontend
npm test
npm run build
```

### NPC movement logic is growing dense

Current state:

- `internal/sunnytown/server/npc_movement.go` and `npc_movement_test.go` are among the largest hand-written files.
- The logic now covers drives, routing, portals, focus windows, reevaluation, failed-target cooldowns, schedules, and production-adjacent behavior.

Why it matters:

- The behavior is meaningful, but the number of interacting state machines increases review risk.
- Dense AI-assisted systems can become hard to tune if concepts are not separated cleanly.

Likely next action:

- Split by behavior only when there is real edit pain.
- Candidate seams: drive selection, route following, schedule pressure, failure/cooldown handling, and debug snapshot formatting.
- Keep tests close to the behavior they describe.

Verification:

```powershell
go test ./internal/sunnytown/server
go test ./cmd/sunny-town ./internal/sunnytown/... ./internal/sunnytownauth
go test ./...
```

### NPC debug endpoint needs a production posture

Current state:

- `internal/sunnytown/server/npc_debug.go` exposes detailed NPC runtime state.
- The endpoint is guarded by `X-HQ-Service-Secret` when `ServiceSecret` is configured.

Why it matters:

- The endpoint is useful for local tuning, but it exposes detailed internal state.
- If runtime configuration ever allows an empty service secret outside local development, this becomes a footgun.

Likely next action:

- Document whether the endpoint is local/dev-only or supported operational tooling.
- If it stays, require a configured service secret before serving debug output.
- Consider an explicit `SUNNY_TOWN_DEBUG_ENDPOINTS_ENABLED` flag.

Verification:

```powershell
go test ./internal/sunnytown/server
docker compose -f deploy\docker-compose.yml config
```

### Frontend production bundle has large chunks

Current state:

- `npm run build` succeeds.
- Vite warns that some chunks exceed the default 500 kB warning threshold.
- Vite also warns that `studentAssignmentsApi.ts` is both dynamically and statically imported.

Why it matters:

- This is not blocking for a local-network app, but it can hurt first-load performance and makes the build look less polished.
- Mixed static/dynamic imports can make intended code splitting ineffective.

Likely next action:

- Decide whether route-level code splitting matters for this app.
- Normalize imports for `studentAssignmentsApi.ts`.
- Consider manual chunks only if the app grows or demo load time becomes noticeable.

Verification:

```powershell
cd frontend
npm run build
```

### Documentation can drift during fast iteration

Current state:

- The project has unusually good docs for its age.
- Current-state docs, implementation plans, and active feature work can diverge quickly.

Why it matters:

- Architecture docs are a major strength of this project.
- Stale docs are worse than missing docs when they describe service boundaries, schema ownership, or API contracts.

Likely next action:

- After each feature slice that changes routes, migrations, package ownership, or runtime behavior, update the matching `docs/current/` file.
- Keep historical planning docs in `docs/archive/` once their decisions are captured in current-state docs.

Verification:

```powershell
git diff -- docs
go test ./...
cd frontend
npm run build
```

## Candidate Resume/Demo Polish

- Add screenshots or a short demo GIF to the README.
- Add a concise "Architecture Highlights" section focused on service boundaries, trusted/untrusted clients, idempotent ledgers, and reproducible runtime.
- Keep the AI-assisted framing explicit: AI accelerated implementation, while architecture, review, integration, and final tradeoffs were human-owned.
- Before sharing the repo, make sure local logs and generated `web/` assets are not staged.

## Closure Rule

When an item is fixed, move it to a short "Resolved" section with:

- date
- files changed
- verification command
- commit hash, if committed
