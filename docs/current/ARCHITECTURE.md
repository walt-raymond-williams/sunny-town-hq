# Current Architecture

HQ is a local-network learning app with three Go services, a Vue frontend, PostgreSQL persistence, and Keycloak authentication.

## Runtime Shape

```text
Browser
  Vue 3 frontend
  JSON API clients
  Connect RPC pet client
  Sunny Town WebSocket client

        |
        v

HQ Go service
  Static frontend host
  JSON HTTP API
  Connect RPC PetService
  Keycloak JWT validation
  Durable app state in PostgreSQL
  Internal endpoints for Sunny Town and AI

        |
        +--> PostgreSQL
        +--> AI service

Sunny Town Go service
  Realtime WebSocket world
  Server-authoritative gameplay validation
  Service-authenticated HQ HTTP calls
```

## Service Ownership

HQ owns durable state:

- account and role records synced from Keycloak
- assignments, attempts, grading, and feedback
- pet state
- wallet stars and star ledger
- inventory, equipment, hotbar, crafting, and shops
- persisted Sunny Town map objects
- last accepted Sunny Town player position

Sunny Town owns live realtime state:

- connected players
- room and map membership
- accepted player movement
- active collectibles
- active resource nodes
- portal transitions
- gameplay validation before reward/resource events reach HQ

The AI service owns assignment grading recommendations and posts results back through service-authenticated HQ endpoints. Active AI implementation details live outside this restructure plan while the AI refactor is in progress.

## Repository Shape

```text
cmd/hq/                 HQ API server and static frontend host
cmd/sunny-town/         Sunny Town realtime WebSocket service
cmd/ai/                 AI grading service
internal/               Shared internal Go packages
proto/                  Protobuf definitions and generated Go/TS code
frontend/               Vue 3 frontend source
sunny-town/maps/        Sunny Town map JSON
deploy/                 Docker Compose, Dockerfiles, Keycloak, database init
docs/current/           Current-state architecture and operations docs
docs/archive/           Historical planning docs
web/                    Ignored generated frontend build output
```

## Known Restructure Targets

- `cmd/hq/main.go` currently contains too much app logic and should eventually become a thin entrypoint.
- `cmd/sunny-town/main.go` currently contains config, protocol, map loading, world simulation, WebSocket handling, and HQ client code.
- `frontend/src/pages/SunnyTownPage.vue` currently contains too much Sunny Town frontend behavior.
- Database evolution is split between Docker init SQL and app-side schema upgrades; migrate to explicit ordered migrations after AI schema work settles.
