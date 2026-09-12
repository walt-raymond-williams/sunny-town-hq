# Kubernetes Migration Discovery

Issue: #73
Epic: #72

## Decision Summary

Use `kind` as the first local Kubernetes target, with Docker Desktop Kubernetes documented later as an optional compatibility path.

Use one umbrella Helm chart at `deploy/helm/sunny-town-hq` for the first migration. Keep templates explicit and readable so the chart teaches the Kubernetes model before adding helper abstractions or subcharts.

Use port-forwarding for the first browser workflow:

- HQ: host `18080` to service port `18080`
- Keycloak: host `18081` to service port `8080`
- Sunny Town: host `18082` to service port `18082`
- AI: host `18083` to service port `18083`

Defer ingress until the port-forwarded workflow is healthy. This keeps the first runnable slice focused on Pods, Services, ConfigMaps, Secrets, PVCs, probes, local images, Helm releases, and operational debugging.

## Current Compose Runtime Audit

The source of truth is `deploy/docker-compose.yml`.

### hq

- Image: built locally from repository root with `deploy/hq/Dockerfile`.
- Build output: static Go binary plus production Vue bundle from `frontend/`, plus copied SQL migrations under `/app/deploy/postgres/migrations`.
- Container port: `18080`.
- Host port: `18080`.
- Health check: `wget -qO- http://127.0.0.1:18080/healthz`.
- Depends on: app Postgres healthy, Keycloak started, AI started.
- Persistent data: none directly; stores durable state in app Postgres.
- Important generated assets: the Dockerfile runs `npm ci` and `npm run build`; generated frontend assets are copied into the runtime image.
- Startup behavior: the HQ binary connects to Postgres and runs idempotent application migrations from `HQ_MIGRATIONS_DIR`, defaulting to `deploy/postgres/migrations`.
- Runtime configuration:
  - `HQ_HOST=0.0.0.0`
  - `HQ_PORT=18080`
  - `DATABASE_URL=postgres://hq:hq@postgres:5432/hq?sslmode=disable`
  - `KEYCLOAK_ISSUER=http://${HQ_PUBLIC_HOST:-localhost}:18081/realms/hq`
  - `KEYCLOAK_JWKS_URL=http://keycloak:8080/realms/hq/protocol/openid-connect/certs`
  - `KEYCLOAK_AUDIENCE=hq-web`
  - `SUNNY_TOWN_JOIN_SECRET=local-dev-secret`
  - `SUNNY_TOWN_SERVICE_SECRET=local-dev-service-secret`
  - `SUNNY_TOWN_WS_URL=ws://${HQ_PUBLIC_HOST:-localhost}:18082/sunny-town/ws`
  - `AI_SERVICE_URL=http://ai:18083`
  - `HQ_TO_AI_SERVICE_SECRET=local-dev-hq-to-ai-secret`
  - `AI_TO_HQ_SERVICE_SECRET=local-dev-ai-service-secret`
  - `AI_GRADING_ENABLED=false`
  - `AI_AUTO_APPLY_GRADES=false`
  - `AI_PROMPT_VERSION_GRADER=assignment-grader-v1`

### sunny-town

- Image: built locally from repository root with `deploy/sunny-town/Dockerfile`.
- Build output: static Go binary plus map JSON copied to `/app/sunny-town/maps`.
- Container port: `18082`.
- Host port: `18082`.
- Health check: `wget -qO- http://127.0.0.1:18082/healthz`.
- Depends on: no Compose dependency, but runtime gameplay calls require HQ.
- Persistent data: none. Live world state is in memory.
- Runtime configuration:
  - `SUNNY_TOWN_HOST=0.0.0.0`
  - `SUNNY_TOWN_PORT=18082`
  - `SUNNY_TOWN_MAPS_DIR=/app/sunny-town/maps`
  - `SUNNY_TOWN_JOIN_SECRET=local-dev-secret`
  - `SUNNY_TOWN_SERVICE_SECRET=local-dev-service-secret`
  - `HQ_INTERNAL_BASE_URL=http://hq:18080`
  - `SUNNY_TOWN_ALLOWED_ORIGINS=*`
  - Optional: `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES`, default `24`.

### ai

- Image: built locally from repository root with `deploy/ai/Dockerfile`.
- Build output: static Go binary.
- Container port: `18083`.
- Host port: `18083`.
- Health check: `wget -qO- http://127.0.0.1:18083/healthz`.
- Depends on: no Compose dependency, but runtime grading calls require HQ.
- Persistent data: none.
- Runtime configuration:
  - `AI_HOST=0.0.0.0`
  - `AI_PORT=18083`
  - `AI_PROVIDER=fake`
  - `HQ_INTERNAL_BASE_URL=http://hq:18080`
  - `HQ_TO_AI_SERVICE_SECRET=local-dev-hq-to-ai-secret`
  - `AI_TO_HQ_SERVICE_SECRET=local-dev-ai-service-secret`
  - `AI_PROMPT_VERSION_GRADER=assignment-grader-v1`
  - Optional future provider settings: `OPENAI_MODEL`, and eventually `OPENAI_API_KEY` if a real provider is added.

### postgres

- Image: `postgres:16-alpine`.
- Container port: `5432`.
- Host port: `55432`.
- Database: `hq`.
- User/password: `hq` / `hq`.
- Volume: `hq-postgres-data:/var/lib/postgresql/data`.
- Init mount: `deploy/postgres` to `/docker-entrypoint-initdb.d:ro`.
- Init behavior: `001_run_migrations.sh` applies all `deploy/postgres/migrations/*.sql` during first database initialization.
- Health check: `pg_isready -U hq -d hq`.

### keycloak-postgres

- Image: `postgres:16-alpine`.
- Container port: `5432`.
- Host port: none.
- Database: `keycloak`.
- User/password: `keycloak` / `keycloak`.
- Volume: `hq-keycloak-postgres-data:/var/lib/postgresql/data`.
- Health check: `pg_isready -U keycloak -d keycloak`.

### keycloak

- Image: `quay.io/keycloak/keycloak:26.0`.
- Command: `start-dev --import-realm`.
- Container port: `8080`.
- Host port: `18081`.
- Depends on: Keycloak Postgres healthy.
- Import mount: `deploy/keycloak` to `/opt/keycloak/data/import:ro`.
- Realm import file: `deploy/keycloak/hq-realm.json`.
- Runtime configuration:
  - `KC_DB=postgres`
  - `KC_DB_URL_HOST=keycloak-postgres`
  - `KC_DB_URL_DATABASE=keycloak`
  - `KC_DB_USERNAME=keycloak`
  - `KC_DB_PASSWORD=keycloak`
  - `KC_BOOTSTRAP_ADMIN_USERNAME=admin`
  - `KC_BOOTSTRAP_ADMIN_PASSWORD=admin`
  - `KC_HTTP_ENABLED=true`
  - `KC_HOSTNAME_STRICT=false`

## Kubernetes Resource Mapping

### Naming

Use release name `sunny-town-hq` and namespace `sunny-town-hq` in local docs. Template resource names from the Helm release and component name:

- `sunny-town-hq-hq`
- `sunny-town-hq-sunny-town`
- `sunny-town-hq-ai`
- `sunny-town-hq-postgres`
- `sunny-town-hq-keycloak`
- `sunny-town-hq-keycloak-postgres`

### Images

For kind, build local images with stable tags and load them into the cluster:

```powershell
docker build -t sunny-town-hq/hq:dev -f deploy/hq/Dockerfile .
docker build -t sunny-town-hq/sunny-town:dev -f deploy/sunny-town/Dockerfile .
docker build -t sunny-town-hq/ai:dev -f deploy/ai/Dockerfile .
kind load docker-image sunny-town-hq/hq:dev --name sunny-town-hq
kind load docker-image sunny-town-hq/sunny-town:dev --name sunny-town-hq
kind load docker-image sunny-town-hq/ai:dev --name sunny-town-hq
```

Set `imagePullPolicy: IfNotPresent` for local app images. Do not push local development images to a registry for the first phase.

### ConfigMaps And Secrets

Use one application ConfigMap for non-secret runtime settings and one local-only Secret for credentials/shared secrets.

ConfigMap examples:

- host/port values
- service URLs
- Keycloak issuer/JWKS/audience
- feature flags
- AI provider mode
- prompt versions
- Sunny Town allowed origins and NPC day length

Secret examples:

- app Postgres password
- Keycloak Postgres password
- Keycloak bootstrap admin password
- `DATABASE_URL`
- `SUNNY_TOWN_JOIN_SECRET`
- `SUNNY_TOWN_SERVICE_SECRET`
- `HQ_TO_AI_SERVICE_SECRET`
- `AI_TO_HQ_SERVICE_SECRET`
- future `OPENAI_API_KEY`

The first chart may keep local defaults in `values.yaml` because they are already local-only development credentials in Compose. Clearly label them as unsafe outside local development. Add `values.local.example.yaml` only if the first values file becomes crowded.

### Workloads

Use Deployments for:

- `hq`
- `sunny-town`
- `ai`
- `keycloak`

Use either Deployments or StatefulSets for the two Postgres services. Start with StatefulSets if the chart is ready to teach stable identity and PVC ownership; otherwise Deployments with PVCs are acceptable for the local-only first pass. Either choice must use PVCs and explicit reset documentation.

### Services

Use ClusterIP Services for every component:

- `hq`: port `18080`
- `sunny-town`: port `18082`
- `ai`: port `18083`
- `postgres`: port `5432`
- `keycloak`: port `8080`
- `keycloak-postgres`: port `5432`

Port-forwarding exposes the browser-facing services to the same local ports Compose uses.

### Probes

Add HTTP readiness and liveness probes for:

- `hq` on `/healthz`
- `sunny-town` on `/healthz`
- `ai` on `/healthz`

Add Postgres readiness probes with `pg_isready`.

For Keycloak, use a conservative startup probe plus readiness check after testing the exact endpoint in the Keycloak 26 image. Do not block the first chart skeleton on perfect Keycloak probes.

### Volumes And Import Data

Use PVCs for:

- app Postgres data
- Keycloak Postgres data

Use ConfigMaps for small static local import/config files only if they fit comfortably:

- Keycloak realm import from `deploy/keycloak/hq-realm.json`
- Postgres init script and SQL migrations if using Postgres entrypoint initialization

Prefer an explicit migration strategy over large ConfigMaps. Since HQ already runs idempotent migrations from files copied into the HQ image, the Kubernetes path should initially let HQ own app schema migrations at startup and avoid duplicating all app migrations through the Postgres entrypoint. Keep the Postgres init script path for Compose until Compose is retired.

### Jobs And Init Containers

Initial chart:

- Use init containers where useful to wait for Postgres before starting Keycloak or HQ.
- Avoid a separate app migration Job in the first pass because HQ already runs migrations and has the migration files in its image.

Later hardening:

- Consider moving application migrations into a Helm hook Job after the basic Kubernetes runtime works.
- Consider a Keycloak import Job or documented reset/import flow if `start-dev --import-realm` becomes unreliable with an existing Keycloak database.

## Browser Reachability And URL Model

Port-forwarding preserves the existing host ports, which keeps browser behavior close to Compose:

- Browser opens HQ at `http://localhost:18080`.
- Frontend derives Keycloak at `http://localhost:18081` unless `VITE_KEYCLOAK_URL` is baked differently.
- HQ validates tokens against issuer `http://localhost:18081/realms/hq`.
- HQ fetches JWKS from cluster-internal `http://sunny-town-hq-keycloak:8080/realms/hq/protocol/openid-connect/certs`.
- HQ returns Sunny Town WebSocket URL `ws://localhost:18082/sunny-town/ws`.
- Sunny Town calls HQ through cluster-internal `http://sunny-town-hq-hq:18080`.
- AI calls HQ through cluster-internal `http://sunny-town-hq-hq:18080`.
- HQ calls AI through cluster-internal `http://sunny-town-hq-ai:18083`.

This split is important: browser-facing issuer and WebSocket URLs use forwarded localhost ports, while service-to-service traffic uses Kubernetes DNS.

For LAN/mobile testing, keep the same rule as Compose: use one public host consistently. Helm values should expose `publicHost`, defaulting to `localhost`, to derive Keycloak issuer and Sunny Town WebSocket URL.

## Reset And Teardown

Local reset must say exactly what happens to durable state.

Expected commands after implementation:

```powershell
helm uninstall sunny-town-hq -n sunny-town-hq
kubectl delete pvc -n sunny-town-hq -l app.kubernetes.io/instance=sunny-town-hq
kubectl delete namespace sunny-town-hq
kind delete cluster --name sunny-town-hq
```

Keep PVC deletion separate from normal Helm uninstall so users learn that Helm release deletion and durable data deletion are different operations.

## First Runnable Slice

The first implementation ticket should be small and verifiable:

1. Add kind bootstrap documentation and Taskfile commands.
2. Add local image build/load commands for the three app images.
3. Add the Helm chart skeleton with namespace, values, naming helpers, ConfigMap, Secret, and app image values.
4. Verify `helm lint` and `helm template`.

This does not need to run the full app. The first full runtime slice should follow with app Postgres plus HQ.

## Implementation Ticket Sequence

1. Add kind bootstrap, image build/load workflow, and Helm chart skeleton.
2. Add app Postgres PVC/Service and deploy HQ against it.
3. Add Sunny Town and AI Deployments/Services with service-auth secrets and health checks.
4. Add Keycloak Postgres, Keycloak realm import, and HQ auth URL wiring.
5. Add port-forward workflow for HQ, Keycloak, Sunny Town WebSocket, and AI.
6. Add Taskfile commands for install, upgrade, status, logs, reset, and teardown.
7. Update current runtime/testing docs after Kubernetes is usable end to end.
8. Add optional ingress or Docker Desktop Kubernetes compatibility after the port-forwarded kind workflow is stable.

## Learning Checklist By Phase

- Bootstrap and image loading: kind clusters, container images, local registries versus cluster image stores.
- Chart skeleton: Helm chart structure, values, templates, labels, release names, namespaces.
- HQ and app Postgres: Deployments, Services, PVCs, readiness probes, app-to-database configuration.
- Sunny Town and AI: service discovery, shared Secrets, WebSocket server exposure, stateless workloads.
- Keycloak: dependent services, realm import, admin credentials, browser-facing issuer versus internal JWKS URL.
- Port-forwarding: local browser access, multiple forwarded services, operational tradeoffs versus ingress.
- Reset and docs: Helm uninstall semantics, PVC lifecycle, namespace cleanup, repeatable developer workflows.

## Risks And Follow-Ups

- Keycloak realm import may behave differently after the first database initialization. The implementation must verify clean install and reset flows.
- HQ currently runs application migrations at startup. This is acceptable for the first Kubernetes pass, but a migration Job may become cleaner later.
- Port-forwarding several services is explicit and teachable, but less ergonomic than ingress. Add ingress only after the basic runtime is healthy.
- `KEYCLOAK_ISSUER` must match the browser-visible URL exactly. This remains the easiest way to break auth when switching between `localhost`, `127.0.0.1`, and LAN IPs.
- App image builds include generated frontend assets, so Kubernetes image rebuilds must happen after frontend changes just like Compose rebuilds do today.
