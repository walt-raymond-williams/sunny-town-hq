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
- inventory, equipment, hotbar, crafting, shops, durable shop input storage, and durable shop stock
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

The AI service owns assignment grading recommendations and posts results back through service-authenticated HQ endpoints. AI implementation details and future AI changes are tracked separately from the completed repository restructure.

## Repository Shape

```text
cmd/hq/                 HQ binary startup, route composition, and static frontend host
cmd/sunny-town/         Sunny Town binary startup
cmd/ai/                 AI grading service
internal/hq/            HQ domain, auth, schema, and HTTP helper packages
internal/sunnytown/     Sunny Town config, protocol, map, HQ client, and server packages
internal/aiapi/         Shared AI API DTOs
proto/                  Protobuf definitions and generated Go/TS code
frontend/               Vue 3 frontend source
sunny-town/maps/        Sunny Town map JSON
deploy/                 Docker Compose, Dockerfiles, Keycloak, database init
docs/current/           Current-state architecture and operations docs
docs/archive/           Historical planning docs
web/                    Ignored generated frontend build output
```

## Current Package Boundaries

The broad restructure is complete. Current package ownership is documented in:

```text
docs/current/PACKAGE_BOUNDARIES.md
docs/current/SCHEMA_OWNERSHIP.md
```

`cmd/hq` intentionally remains the HQ binary composition layer. It owns startup, DB connection, migrations, route composition, and server logging. HQ business behavior lives under `internal/hq/...`.

Further maintainability work should focus on splitting large files inside the package boundaries that now exist, tracked in:

```text
docs/BIG_FILE_SPLIT_PLAN.md
```
