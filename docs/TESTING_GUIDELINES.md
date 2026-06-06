# Testing Guidelines

Use this checklist when validating changes in HQ and Sunny Town. Favor the container workflow unless a task explicitly needs a local development server.

## Standard Checks

Run backend tests from the repo root:

```powershell
go test ./...
```

Run frontend type checks from `frontend/`:

```powershell
npm run typecheck
```

For production-bundle validation, rebuild through Docker. The HQ image runs `npm run build` internally, so this also catches Vue production build issues:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
```

Then confirm both services are healthy:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
docker ps --filter "name=hq-" --format "{{.Names}}\t{{.Status}}\t{{.Ports}}"
```

Expected health response is `ok`. Expected containers include healthy `hq-server` and `hq-sunny-town`.

## Sunny Town Multiplayer / Room Testing

For Sunny Town map, room, movement, collectible, resource, or inventory overlay changes, verify:

- The server starts with all map JSON files loaded from `SUNNY_TOWN_MAPS_DIR`.
- `sunny-town-v1` remains the default first join map when no saved position exists.
- A saved `student_sunny_town_position` can return the player to their last map and position.
- The frontend receives map data from the server instead of relying on hardcoded map geometry.
- Player movement still works with arrow keys and WASD.
- Pressing `E` toggles inventory and does not affect movement.
- Stars still appear and collect only on maps with `starSpawns`.
- A player walking into the outdoor southern building door transitions into `sunny-town-house-1`.
- A player walking into the indoor south door transitions back to `sunny-town-v1`.
- A player walking through the eastern path transitions to `forest-crossing-v1` and can return.
- Two students inside the same map can see each other.
- Players in different maps do not appear in each other's snapshots.
- Mining a Forest Crossing node requires an equipped pickaxe, advances node hit count, depletes after three hits, and sends a resource commit result.

Backend tests should cover at least:

- Map loading succeeds with multiple maps.
- Duplicate map IDs are rejected.
- Portals targeting missing maps are rejected.
- Portal entry sends a `map_changed` message with the target map.
- Snapshot lists are scoped to the player's current map.
- Indoor maps without `starSpawns` do not enqueue star reward events.
- Resource node definitions are validated, including duplicate id rejection.
- Mining rejects missing tools, wrong tools, inactive nodes, and out-of-range players.
- HQ resource event commits are idempotent through `student_inventory_ledger`.

## Inventory Testing

For inventory changes, verify:

- `go test ./...` still covers cookie award/feed flows.
- `GET /api/student/inventory` returns only items with quantity greater than zero.
- Student Pet inventory dialog shows the same cookie quantity as Sunny Town inventory.
- Feeding the pet consumes cookie inventory, not `app_user.cookies`.
- Stars remain in `student_wallet` and are not represented as inventory items.
- Mining resources appear as `rock` or `crystal` inventory items after HQ confirms the resource event.

## Manual Browser Notes

The in-app browser can inspect local pages, but login text entry may fail if the Browser plugin reports the virtual clipboard is unavailable. When that happens:

- Do not treat the failed browser login as a product failure by itself.
- Prefer deterministic checks first: Go tests, frontend typecheck/build, Docker rebuild, health checks, and server logs.
- If manual authenticated verification is needed, use the visible local app in a normal browser session.

Useful local URLs:

- HQ: `http://localhost:18080`
- Keycloak: `http://localhost:18081`
- Sunny Town health: `http://localhost:18082/healthz`

## Container Workflow

Current project preference is to run HQ and Sunny Town from containers while developing. After code changes that affect either service, rebuild and recreate the containers:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
```

Avoid leaving local `go run ./cmd/hq` processes running alongside the HQ container on the same port. If a local HQ process was started during debugging, stop it before returning to the container workflow.

## Git Hygiene

Before finalizing, check changed files:

```powershell
git status --short
```

Do not revert unrelated changes. Built `web/` assets may change after local frontend builds; the Docker HQ image builds frontend assets internally, so local `web/` changes are not always required for container verification.
