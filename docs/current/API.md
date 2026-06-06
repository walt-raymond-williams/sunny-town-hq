# Current API Surface

This file summarizes the current API ownership and route groups. Handler details remain in code until the backend split creates package-level docs.

## Public App APIs

HQ exposes JSON APIs under `/api/...` and validates Keycloak bearer tokens for app users.

Current route groups:

- `/api/me`
- `/api/students`
- `/api/teacher/...`
- `/api/student/profile`
- `/api/student/inventory`
- `/api/student/hotbar`
- `/api/student/crafting/...`
- `/api/student/equipment/...`
- `/api/student/shop/purchase`
- `/api/student/pet/feed`
- `/api/student/sunny-town/session`
- `/api/student/assignments/...`
- `/api/assignments...`

Pet APIs use Connect RPC at:

```text
/hq.pet.v1.PetService/...
```

The protobuf source is `proto/hq/pet/v1/pet.proto`. Generated Go code lives under `proto/`; generated TypeScript lives under `frontend/src/gen/`.

## Internal Service APIs

Sunny Town calls HQ through service-authenticated internal endpoints using `X-HQ-Service-Secret`.

Current Sunny Town internal groups:

- `/api/internal/sunny-town/reward-events`
- `/api/internal/sunny-town/resource-events`
- `/api/internal/sunny-town/student-equipment`
- `/api/internal/sunny-town/inventory-quantity`
- `/api/internal/sunny-town/player-position`
- `/api/internal/sunny-town/map-objects`
- `/api/internal/sunny-town/map-objects/place`
- `/api/internal/sunny-town/map-objects/remove`

AI service integration uses internal HQ endpoints under:

```text
/api/internal/ai/assignment-attempts/...
```

Do not refactor the AI route shape while the AI service refactor is in progress.

## Sunny Town WebSocket

Sunny Town exposes realtime gameplay at:

```text
/sunny-town/ws
```

Browsers receive a short-lived join token from HQ before connecting. Client messages are requests; Sunny Town validates gameplay effects against server-accepted position and equipped/owned tools before committing durable effects to HQ.
