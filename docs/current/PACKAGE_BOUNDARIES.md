# Current Package Boundaries

This document records the post-restructure source ownership model. Use it when deciding where new code belongs.

## Command Packages

Command packages are binary entrypoints and composition layers.

- `cmd/hq`: HQ startup, DB connection, startup migrations, route composition, static frontend serving, and remaining top-level session handlers.
- `cmd/sunny-town`: Sunny Town startup only.
- `cmd/ai`: AI grading service startup and service implementation.

Do not move `cmd/hq` app construction or route registration just to reduce line count. The current decision is to keep command startup and composition local unless a future change reveals real edit pain.

## HQ Internal Packages

- `internal/hq/app`: HQ runtime configuration loading.
- `internal/hq/auth`: Keycloak JWT verification, auth user types, context helpers, and role checks.
- `internal/hq/characters`: durable Sunny Town character identity shared by player-controlled characters and future NPCs.
- `internal/hq/users`: app user persistence, role sync, default student provisioning, and student listing.
- `internal/hq/assignments`: assignment DTOs, typed assignment queries, submission, grading commands, teacher/student HTTP handlers, and AI grading coordination points.
- `internal/hq/inventory`: student inventory, equipment, hotbar, crafting, wallet, shop, and related HTTP handlers.
- `internal/hq/pet`: pet rules, profile persistence, pet actions, decay, game-result rewards, REST handlers, and Connect RPC handler.
- `internal/hq/sunnytownbridge`: service-authenticated Sunny Town internal endpoints, durable Sunny Town position/map-object persistence, star and inventory ledgers for Sunny Town events.
- `internal/hq/ai`: HQ-side AI grading context, result persistence, auto-apply policy, internal AI endpoints, and outbound grade triggering.
- `internal/hq/schema`: startup migration runner for ordered SQL migrations.
- `internal/hq/httpapi`: domain-neutral HTTP helpers such as JSON responses, static SPA serving, request logging, and local IPv4 discovery.

HQ owns durable state. Sunny Town and AI services interact with HQ through service-authenticated APIs instead of writing HQ tables directly.

## Sunny Town Internal Packages

- `internal/sunnytown/config`: runtime config loading.
- `internal/sunnytown/hqclient`: service-authenticated internal HTTP client for HQ calls.
- `internal/sunnytown/maps`: map JSON types, loading, and validation.
- `internal/sunnytown/protocol`: WebSocket protocol message types.
- `internal/sunnytown/server`: realtime rooms, players, movement validation, portals, collectibles, resources, placement, snapshots, workers, and WebSocket handling.
- `internal/sunnytownauth`: signed join-token claims for HQ-to-Sunny-Town browser sessions.

Sunny Town owns live realtime state. Browser messages are requests; gameplay effects must be validated against server-accepted state before HQ receives durable reward/resource events.

## Shared Service Packages

- `internal/serviceauth`: shared helpers for service-authenticated internal calls where a generic helper is appropriate.
- `internal/aiapi`: shared AI grading request/response DTOs.

Prefer package-local helpers when behavior is domain-specific. Promote helpers to shared packages only when the contract is identical across call sites.

## Frontend Shape

- `frontend/src/api`: typed frontend API clients.
- `frontend/src/features/student`: student workspace screens.
- `frontend/src/features/teacher`: teacher workspace screens.
- `frontend/src/features/sunny-town`: Sunny Town page and extracted UI panels.
- `frontend/src/composables`: reusable Vue state/effect logic, including Sunny Town socket, movement, and renderer lifecycle.
- `frontend/src/stores`: Pinia stores for app state.
- `frontend/src/types`: shared frontend TypeScript types.
- `frontend/src/gen`: generated protobuf TypeScript.

The largest remaining frontend ownership hotspot is `frontend/src/features/sunny-town/SunnyTownPage.vue`. Future work is tracked in `docs/BIG_FILE_SPLIT_PLAN.md`.

## Documentation Pointers

- Current architecture: `docs/current/ARCHITECTURE.md`
- Current API surface: `docs/current/API.md`
- Database and migration workflow: `docs/current/DATABASE.md`
- Schema ownership: `docs/current/SCHEMA_OWNERSHIP.md`
- Runtime and health checks: `docs/current/RUNTIME.md`
- Completed restructure history: `docs/RESTRUCTURE_IMPLEMENTATION_PLAN.md`
- Next large-file cleanup plan: `docs/BIG_FILE_SPLIT_PLAN.md`
