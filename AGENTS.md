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
- Do not stage generated `web/` assets; rebuild them locally or through the Docker HQ image.
- Leave local logs untracked unless the user explicitly asks to commit them.

## Learning Loop

When a bug takes real effort to understand, or a successful workflow would be useful next time, propose an `AGENTS.md` update. Keep each addition short enough that future agents will actually read it.
