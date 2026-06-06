# AGENTS.md

## Project Posture

This is an AI-assisted project. Treat this file as a living operating guide for Codex and other coding agents working in this repository.

Keep guidance practical and compact. Add to this file when a workflow repeatedly helps, when a mistake recurs, or when debugging teaches a rule future agents should inherit. Prefer concrete commands, ownership boundaries, and verification steps over broad philosophy.

## Repository Shape

- HQ service code lives in `cmd/hq`.
- Sunny Town realtime service code lives in `cmd/sunny-town`.
- Sunny Town map JSON lives in `sunny-town/maps`.
- Vue frontend code lives in `frontend/src`.
- Production frontend assets are emitted into `web/` by `npm run build`.
- Integration/runtime configuration lives under `deploy/`.

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
- The planning doc `docs/SUNNY_TOWN_FOREST_CROSSING_MINING_PLAN.md` may be untracked during active work; do not stage it unless the user asks.

## Verification

Favor deterministic checks before manual browser exploration.

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
- After frontend changes, run `npm run build` from `frontend/`; this updates the checked-in `web/` bundle when the production build changes.
- After significant local frontend changes, smoke-test the container-served app at `http://127.0.0.1:18080` when practical.

## Commit Hygiene

- Check `git status --short` before staging and before final response.
- Stage files explicitly. Avoid `git add .` unless the user specifically wants everything.
- Include generated `web/` asset changes when they result from a deliberate production build.
- Leave local logs untracked unless the user explicitly asks to commit them.

## Learning Loop

When a bug takes real effort to understand, or a successful workflow would be useful next time, propose an `AGENTS.md` update. Keep each addition short enough that future agents will actually read it.
