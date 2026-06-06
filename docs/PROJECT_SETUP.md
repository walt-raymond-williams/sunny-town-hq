# HQ Project Setup

This file is a practical orientation guide for an AI agent or developer working on HQ.

HQ is a local-network homework app. A Go server serves the built Vue app, exposes JSON and Connect RPC APIs, stores app data in PostgreSQL, and validates Keycloak access tokens for teacher/student login.

## Services

### HQ Go Server

- Source: `cmd/hq/`
- Default port: `8080`
- Common local-network port: `18080`
- Serves:
  - built frontend from `web/`
  - JSON APIs under `/api/...`
  - Connect RPC pet APIs under `/hq.pet.v1.PetService/...`
  - health check at `/healthz`
- Requires:
  - `DATABASE_URL`
  - `KEYCLOAK_ISSUER`
  - `KEYCLOAK_AUDIENCE`
  - `KEYCLOAK_JWKS_URL` when running in Docker and the browser-facing issuer differs from the Docker-network URL

### Frontend

- Source: `frontend/`
- Framework: Vue 3 + Vuetify + Pinia
- Build output: `web/`
- Build command:

```powershell
cd frontend
npm run build
```

The frontend auth helper is `frontend/src/auth.ts`. It uses Keycloak authorization-code login, stores the returned refresh token in session storage, and sends refreshed access tokens as bearer tokens to the Go backend.

The helper intentionally avoids `keycloak-js` because browsers block some Web Crypto APIs on plain LAN HTTP URLs such as `http://<YOUR_LAN_IP>:18080`. PKCE uses S256 when available and falls back to the plain verifier method for trusted local-network development.

### App PostgreSQL

- Service name: `postgres`
- Container: `hq-postgres`
- Image: `postgres:16-alpine`
- Host port: `55432`
- Database/user/password: `hq` / `hq` / `hq`
- Init SQL: `deploy/postgres/init/001_create_assignment.sql`
- Persistent volume: `hq-postgres-data`

Main tables:

- `app_user`: local app user profile synced from Keycloak subject
- `app_user_role`: app copy of Keycloak `student` / `teacher` roles
- `assignment`: teacher-created questions
- `assignment_attempt`: per-student answers, grading, feedback, resets
- `pet_state`: per-student virtual pet state
- `student_wallet`: current per-student star balance
- `student_star_ledger`: idempotent star reward and spend records
- `inventory_item_type`: inventory catalog
- `student_inventory_item`: current per-student item quantities
- `student_inventory_ledger`: idempotent Sunny Town resource event records
- `student_equipped_item`: current gear/accessory/tool equipment
- `student_sunny_town_position`: last accepted Sunny Town map position

The Go app also runs schema-safety upgrades at startup in `cmd/hq/schema.go`.

### Keycloak

- Service name: `keycloak`
- Container: `hq-keycloak`
- Image: `quay.io/keycloak/keycloak:26.0`
- Host port: `18081`
- Realm: `hq`
- Client: `hq-web`
- Roles: `teacher`, `student`
- Realm import: `deploy/keycloak/hq-realm.json`
- Admin user: `admin`
- Admin password: `admin`

Keycloak data is stored in a separate Postgres service:

- Service name: `keycloak-postgres`
- Container: `hq-keycloak-postgres`
- Persistent volume: `hq-keycloak-postgres-data`

Important: Keycloak imports the realm only on first startup for a fresh Keycloak database. If the realm already exists, changing `deploy/keycloak/hq-realm.json` does not automatically update the running realm. Update the client in the Keycloak admin UI/API or reset the Keycloak volume.

### Sunny Town

- Service name: `sunny-town`
- Container: `hq-sunny-town`
- Source: `cmd/sunny-town/`
- Dockerfile: `deploy/sunny-town/Dockerfile`
- Host port: `18082`
- Serves:
  - WebSocket gameplay at `/sunny-town/ws`
  - health check at `/healthz`
- Requires:
  - `SUNNY_TOWN_JOIN_SECRET`

The HQ server must use the same `SUNNY_TOWN_JOIN_SECRET` when issuing join tokens, and should return the browser-reachable WebSocket URL through `SUNNY_TOWN_WS_URL`.
The Docker Compose service allows any WebSocket origin for local home-network development so the app works from `localhost`, the laptop LAN IP, and phone/tablet browsers.

## Start The Stack

Start the full stack:

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
```

HQ is available at `http://localhost:18080`, Keycloak at `http://localhost:18081`, Sunny Town at `http://localhost:18082`, and PostgreSQL at `localhost:55432`.

For another device on your local network, set the browser-facing host before starting Compose. This value must match the hostname or IP address users open in the browser, because Keycloak embeds it as the token issuer:

```powershell
"HQ_PUBLIC_HOST=<YOUR_LAN_IP>" | Set-Content deploy/.env
docker compose -f deploy/docker-compose.yml up -d --build
```

Find the laptop LAN IP:

```powershell
$route = Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1
Get-NetIPAddress -InterfaceIndex $route.InterfaceIndex -AddressFamily IPv4
```

Run HQ with the same LAN host that users will open in their browsers. Example:

```powershell
$env:DATABASE_URL="postgres://hq:hq@localhost:55432/hq?sslmode=disable"
$env:HQ_PORT="18080"
$env:KEYCLOAK_ISSUER="http://<YOUR_LAN_IP>:18081/realms/hq"
$env:KEYCLOAK_AUDIENCE="hq-web"
$env:SUNNY_TOWN_JOIN_SECRET="local-dev-secret"
$env:SUNNY_TOWN_WS_URL="ws://<YOUR_LAN_IP>:18082/sunny-town/ws"
go run ./cmd/hq
```

Open the app:

```text
http://<YOUR_LAN_IP>:18080
```

Open Keycloak admin:

```text
http://<YOUR_LAN_IP>:18081
```

Use:

```text
admin / admin
```

## Create Users

In Keycloak:

1. Open the `hq` realm.
2. Go to `Users`.
3. Create one user per person.
4. Set a non-temporary password in the Credentials tab.
5. Assign realm role `student`, `teacher`, or both in Role mapping.

HQ determines authorization from Keycloak roles in the access token. The backend validates the token and enforces role checks server-side. On first authenticated request, HQ creates or updates the local `app_user` row and role rows.

## Useful Commands

Health check:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz
```

Verify Keycloak discovery:

```powershell
Invoke-WebRequest -UseBasicParsing http://<YOUR_LAN_IP>:18081/realms/hq/.well-known/openid-configuration
```

Run backend checks:

```powershell
go test ./...
```

Build frontend:

```powershell
cd frontend
npm run build
```

Inspect containers:

```powershell
docker ps --filter name=hq --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

View Keycloak logs:

```powershell
docker logs hq-keycloak --tail 120
```

Check Sunny Town:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz
docker logs hq-sunny-town --tail 120
```

Rebuild Sunny Town after Go service or WebSocket protocol changes:

```powershell
docker compose -f deploy/docker-compose.yml up -d --build sunny-town
```

Rebuild the frontend after Sunny Town client changes:

```powershell
cd frontend
npm run build
```

## Local-Network Auth Notes

- Use one consistent host/IP for HQ and Keycloak. If users open HQ at `http://<YOUR_LAN_IP>:18080`, then `KEYCLOAK_ISSUER` should be `http://<YOUR_LAN_IP>:18081/realms/hq`.
- Do not rely on `localhost` from student devices. On a phone/tablet, `localhost` means that device, not the laptop.
- Plain LAN HTTP is not a secure browser context. Browser Web Crypto `subtle` APIs may be unavailable, so the frontend auth helper falls back from S256 PKCE to plain PKCE there.
- The `hq-web` Keycloak client should keep standard flow enabled. Implicit flow is not required.
- If Keycloak rejects login with `Invalid parameter: redirect_uri`, add the exact HQ URL to the `hq-web` client's Valid redirect URIs.

## Common Pitfalls

- Rebuilding Vue changes hashed asset names in `web/assets/`. Hard-refresh the browser if it still loads an older bundle.
- If `127.0.0.1:18081` serves the HQ app instead of Keycloak, a stale Go process is probably listening on that address. Check listeners:

```powershell
Get-NetTCPConnection -State Listen | Where-Object { $_.LocalPort -in 18080,18081 }
```

- Keycloak realm import is not a live migration. For existing Keycloak data, update the realm/client through the admin UI/API.
- Existing uncommitted work may include generated frontend assets in `web/`; avoid deleting unrelated user changes.

## Important Files

- `cmd/hq/main.go`: JSON API routing and assignment workflows
- `cmd/hq/auth.go`: Keycloak JWT/JWKS validation
- `cmd/hq/schema.go`: runtime schema upgrades and app-user sync
- `cmd/hq/inventory.go`: inventory query and mutation helpers
- `cmd/hq/equipment.go`: equipment slot validation and updates
- `cmd/hq/pet_service.go`: Connect RPC pet service
- `cmd/sunny-town/main.go`: Sunny Town WebSocket service, rooms, portals, rewards, and mining
- `deploy/docker-compose.yml`: PostgreSQL and Keycloak services
- `deploy/sunny-town/Dockerfile`: Sunny Town container build
- `deploy/keycloak/hq-realm.json`: initial Keycloak realm/client/roles
- `deploy/postgres/init/001_create_assignment.sql`: fresh app database schema
- `sunny-town/maps/`: checked-in Sunny Town map JSON files
- `frontend/src/auth.ts`: local-network OIDC login helper
- `frontend/src/App.vue`: app shell, route outlet, game overlay, and floating pet
- `frontend/src/features/student/`: student workflow pages and tabs
- `frontend/src/features/teacher/`: teacher workflow pages and grading panels
- `frontend/src/stores/studentPet.ts`: pet Pinia store and Connect client
