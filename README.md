# HQ

HQ is a local-network learning app with a Go backend, Vue frontend, PostgreSQL persistence, Keycloak login, and a small realtime multiplayer world called Sunny Town.

The app is designed to run from a laptop on a home network. Teachers can create and grade schoolwork, students can answer assignments, care for a virtual pet, earn stars, manage inventory, and enter Sunny Town for realtime play, shops, resource gathering, and map interactions.

## Features

- Teacher and student login through Keycloak roles.
- Teacher workspace for creating, reviewing, grading, resetting, and deleting assignments.
- Student workspace for answering questions, reviewing grades, and seeing subject progress.
- Virtual pet state with food, happiness, energy, sleep, and live updates.
- Inventory, equipment, crafting, hotbar, wallet stars, and shop purchases.
- Sunny Town realtime WebSocket service with map rooms, portals, collectibles, NPCs, mining, placed blocks, and persisted return position.
- PostgreSQL-backed durable state with idempotent reward and inventory ledgers.

## Stack

- Backend: Go 1.22
- Frontend: Vue 3, Vuetify, Pinia, Vite, TypeScript
- Realtime: Go WebSocket service
- Database: PostgreSQL 16
- Auth: Keycloak
- RPC: Connect RPC for pet service APIs
- Local runtime: Docker Compose

## Repository Layout

```text
cmd/hq/                 HQ API server and static frontend host
cmd/sunny-town/         Sunny Town realtime WebSocket service
frontend/               Vue 3 frontend source
sunny-town/maps/        Sunny Town map JSON
proto/                  Protobuf definitions and generated Go code
internal/               Shared internal Go packages
deploy/                 Docker Compose, Dockerfiles, Keycloak, database init
docs/                   Setup, testing, architecture, and feature notes
docs/current/           Current-state architecture, API, database, and runtime docs
docs/archive/           Historical planning docs
web/                    Ignored generated frontend build output
Taskfile.yml            Common verification and runtime commands
```

## Quick Start

Prerequisites:

- Docker Desktop
- Go 1.22+ for local backend tests and direct `go run`
- Node.js and npm for frontend development

Start the full local stack:

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
```

Open:

```text
HQ app:        http://localhost:18080
Keycloak:      http://localhost:18081
Sunny Town:    http://localhost:18082
PostgreSQL:    localhost:55432
```

Keycloak admin login:

```text
admin / admin
```

Create users in the `hq` realm and assign the `student` and/or `teacher` realm roles. The frontend uses Keycloak login, and the Go backend validates bearer tokens and enforces roles server-side.

## Local Network Use

For phone or tablet testing on the same Wi-Fi, set the public host to the laptop's LAN IP before starting Docker Compose:

```powershell
"HQ_PUBLIC_HOST=<YOUR_LAN_IP>" | Set-Content deploy/.env
docker compose -f deploy/docker-compose.yml up -d --build
```

Then open:

```text
http://<YOUR_LAN_IP>:18080
```

Use one consistent host/IP for HQ and Keycloak. A token issued for `localhost` will not match a browser session opened through the laptop LAN IP.

## Development

With Go Task installed, use the shared project commands:

```powershell
task test
task frontend:build
task compose:up
task health
task verify
```

Direct fallback commands are below.

Run backend tests:

```powershell
go test ./...
```

Run Sunny Town-only tests:

```powershell
go test ./cmd/sunny-town
```

Run HQ-only tests:

```powershell
go test ./cmd/hq
```

Install frontend dependencies:

```powershell
cd frontend
npm install
```

Run the frontend dev server:

```powershell
cd frontend
npm run dev
```

Build the production frontend bundle served by HQ:

```powershell
cd frontend
npm run build
```

The frontend production build writes ignored assets into `web/`.

## Health Checks

After starting or rebuilding services:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Both services should return:

```text
ok
```

## Service Boundaries

HQ owns durable account, student, assignment, pet, wallet, inventory, equipment, map object, and saved Sunny Town position state.

Sunny Town owns live realtime world state: connected players, map room membership, accepted movement, live collectibles, resource nodes, portals, and gameplay validation.

Sunny Town does not write the HQ database directly. It calls internal HQ HTTP endpoints with `X-HQ-Service-Secret`.

## More Documentation

- [Current architecture](docs/current/ARCHITECTURE.md)
- [Current API surface](docs/current/API.md)
- [Current database](docs/current/DATABASE.md)
- [Current runtime](docs/current/RUNTIME.md)
- [Project setup](docs/PROJECT_SETUP.md)
- [Testing guidelines](docs/TESTING_GUIDELINES.md)
- [Sunny Town architecture](docs/SUNNY_TOWN_ARCHITECTURE.md)
- [Sunny Town movement model](docs/SUNNY_TOWN_MOVEMENT_MODEL.md)
- [Inventory and equipment](docs/INVENTORY_AND_EQUIPMENT.md)
- [Keycloak setup](docs/KEYCLOAK_SETUP.md)
- [Restructure implementation plan](docs/RESTRUCTURE_IMPLEMENTATION_PLAN.md)

## Notes

- `web/` is generated by `npm run build` and should not be committed.
- Local log files may appear in `git status`; they are development artifacts.
- Prefer Docker Compose for integration verification because it exercises HQ, Sunny Town, PostgreSQL, and Keycloak together.
