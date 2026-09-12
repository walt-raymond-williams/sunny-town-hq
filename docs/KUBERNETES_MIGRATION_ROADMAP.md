# Kubernetes Migration Roadmap

Epic: #72, "Migrate Local Runtime To Kubernetes And Helm"

## Outcome

Migrate the project runtime from Docker Compose to Kubernetes so the repository can be used as a hands-on Kubernetes and Helm learning project.

The goal is not only to make the app run. The goal is to learn the Kubernetes model by converting the current container runtime into Kubernetes resources, organizing those resources into Helm charts, and documenting the operational workflow well enough that future agents can keep using and extending it.

## Direction

Docker Compose is the current runtime source of truth. Kubernetes is the target runtime for this epic.

Phase 1 selected the first target architecture:

- Local Kubernetes target: `kind`.
- Helm shape: one readable umbrella chart at `deploy/helm/sunny-town-hq`.
- Browser access: documented `kubectl port-forward` commands on the current Compose ports before adding ingress.
- Local images: build with Docker, tag as `sunny-town-hq/*:dev`, and load into kind with `kind load docker-image`.
- Persistence: PVCs for app Postgres and Keycloak Postgres.
- Secrets: a local-only Helm Secret for development credentials/shared service secrets, with clear warnings that defaults are not production-safe.
- App migrations: let HQ run its existing idempotent schema migrations at startup for the first Kubernetes path; revisit a dedicated migration Job after the basic runtime is healthy.
- Keycloak realm import: mount the realm JSON into the Keycloak container for the first pass and verify clean install/reset behavior before hardening.

Use a staged migration:

1. Audit the existing containers, images, configuration, ports, health checks, persistent data, secrets, startup order, and local developer workflows.
2. Design the Kubernetes and Helm structure for the repo.
3. Create the first local Kubernetes development path.
4. Convert stateless application services.
5. Convert PostgreSQL and Keycloak persistence/import behavior.
6. Wire service discovery, ingress or port-forwarding, browser-facing URLs, WebSocket routing, and auth issuer/JWKS behavior.
7. Add repeatable build, install, upgrade, health, reset, and teardown commands.
8. Update verification and current runtime docs after the Kubernetes workflow becomes usable.
9. Retire Docker Compose as the preferred runtime only after Kubernetes reaches feature parity for local development.

## Current Compose Runtime To Convert

The current runtime is defined in `deploy/docker-compose.yml`.

Services:

- `hq`: Go API/static host on `18080`, built from `deploy/hq/Dockerfile`.
- `sunny-town`: realtime WebSocket service on `18082`, built from `deploy/sunny-town/Dockerfile`.
- `ai`: AI grading service on `18083`, built from `deploy/ai/Dockerfile`.
- `postgres`: app PostgreSQL 16 with migration init scripts and durable volume.
- `keycloak-postgres`: Keycloak PostgreSQL 16 with durable volume.
- `keycloak`: Keycloak 26 dev server with realm import.

Runtime concerns to preserve:

- HQ depends on app Postgres, Keycloak, and AI.
- Keycloak depends on its own Postgres.
- HQ and Sunny Town share `SUNNY_TOWN_JOIN_SECRET` and `SUNNY_TOWN_SERVICE_SECRET`.
- HQ and AI share service secrets.
- HQ needs browser-facing Keycloak issuer settings and cluster-internal JWKS access.
- HQ returns a browser-reachable Sunny Town WebSocket URL.
- Sunny Town calls HQ through an internal service URL.
- AI calls HQ through an internal service URL.
- App Postgres runs migration SQL from `deploy/postgres`.
- Keycloak imports `deploy/keycloak/hq-realm.json`.
- Local development currently exposes stable host ports `18080`, `18081`, `18082`, and `18083`.

## Target Repository Shape

Phase 1 should confirm or revise this proposed shape:

```text
deploy/
  docker-compose.yml            existing runtime until Kubernetes is ready
  k8s/                          raw manifests only if useful for learning/debugging
  helm/
    sunny-town-hq/              umbrella chart for local/dev runtime
      Chart.yaml
      values.yaml
      templates/
    charts/                     optional subcharts if the design needs them
```

The first Helm chart should favor readability and learning value over premature generality. Prefer explicit templates and documented values before introducing complex chart abstraction.

## Phase 1: Discovery And Migration Design

Goal: produce the concrete Kubernetes migration plan and child tickets.

This phase must not merely research Kubernetes in the abstract. It must audit this repository and decide how this specific system maps to Kubernetes and Helm.

Deliverables:

- Document every current container/image, environment variable, port, volume, health check, dependency, and local workflow that must survive the migration.
- Decide the local Kubernetes target for initial practice, such as Docker Desktop Kubernetes, kind, or minikube.
- Decide the Helm chart structure, values strategy, naming conventions, and where manifests live.
- Decide how local images are built and made available to the cluster.
- Decide how local development exposes HQ, Keycloak, Sunny Town WebSocket, and AI to the browser.
- Decide how secrets are represented for local development, and clearly mark what is local-only.
- Decide how app Postgres migrations and Keycloak realm import work in Kubernetes.
- Identify the smallest first runnable Kubernetes slice.
- Update this roadmap with the detailed phase sequence.
- Create child GitHub issues for the migration implementation work, with acceptance criteria and verification steps.

Acceptance criteria:

- A Kubernetes migration discovery doc exists under `docs/`.
- This roadmap is updated with the detailed implementation phases and ticket list.
- GitHub child issues exist for the next actionable migration steps.
- Each child issue names the expected files, verification commands, dependencies, and whether it is Kubernetes, Helm, docs, or runtime work.
- The plan includes a learning checklist that explains the Kubernetes concepts practiced by each phase.

Status: completed by #73 in `docs/KUBERNETES_MIGRATION_DISCOVERY.md`.

Key decisions:

- Start with kind instead of Docker Desktop Kubernetes or minikube because it makes local image loading, cluster lifecycle, and reset behavior explicit and repeatable.
- Start with port-forwarding instead of ingress because it preserves current ports and keeps the first implementation focused on core Kubernetes resources.
- Use an umbrella chart first. Split into subcharts only after the first chart becomes unwieldy.
- Keep Docker Compose as the preferred runtime until the Kubernetes path can run HQ, Keycloak, Sunny Town, AI, and both databases end to end.

## Phase 2: Kind Bootstrap And Helm Skeleton

Goal: add the smallest Kubernetes foundation without trying to run the full app yet.

Deliverables:

- Add `deploy/helm/sunny-town-hq/Chart.yaml`, `values.yaml`, and initial templates.
- Add values for image repositories/tags, public host, ports, service URLs, feature flags, and local-only secrets.
- Add helper labels and resource naming conventions.
- Add Taskfile commands or documented direct commands for creating/deleting the kind cluster.
- Add local image build/load commands for `hq`, `sunny-town`, and `ai`.

Verification:

```powershell
kind create cluster --name sunny-town-hq
docker build -t sunny-town-hq/hq:dev -f deploy/hq/Dockerfile .
docker build -t sunny-town-hq/sunny-town:dev -f deploy/sunny-town/Dockerfile .
docker build -t sunny-town-hq/ai:dev -f deploy/ai/Dockerfile .
kind load docker-image sunny-town-hq/hq:dev --name sunny-town-hq
kind load docker-image sunny-town-hq/sunny-town:dev --name sunny-town-hq
kind load docker-image sunny-town-hq/ai:dev --name sunny-town-hq
helm lint deploy/helm/sunny-town-hq
helm template sunny-town-hq deploy/helm/sunny-town-hq --namespace sunny-town-hq
```

Learning concepts:

- kind clusters
- local image loading
- Helm chart structure
- values and templates
- labels, names, and namespaces

## Phase 3: App Postgres And HQ

Goal: run the smallest useful application slice in Kubernetes.

Deliverables:

- Add app Postgres workload, PVC, Secret values, Service, and readiness probe.
- Add HQ Deployment and Service.
- Wire HQ `DATABASE_URL`, Keycloak placeholders, AI URL placeholder, public host, and migration directory.
- Add HQ readiness/liveness probes.
- Document how to port-forward HQ and inspect logs/events.

Verification:

```powershell
helm upgrade --install sunny-town-hq deploy/helm/sunny-town-hq --namespace sunny-town-hq --create-namespace
kubectl get pods,svc,pvc -n sunny-town-hq
kubectl logs -n sunny-town-hq deploy/sunny-town-hq-hq
kubectl port-forward -n sunny-town-hq svc/sunny-town-hq-hq 18080:18080
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
```

Learning concepts:

- Deployments
- Services
- PVCs
- Secrets
- probes
- application logs

## Phase 4: Sunny Town And AI

Goal: add the two stateless service workloads and service-auth wiring.

Deliverables:

- Add Sunny Town Deployment, Service, health probes, map directory assumptions, and service-auth Secret wiring.
- Add AI Deployment, Service, health probes, fake provider defaults, and service-auth Secret wiring.
- Wire HQ to AI through Kubernetes DNS.
- Wire Sunny Town and AI to HQ through Kubernetes DNS.

Verification:

```powershell
helm upgrade --install sunny-town-hq deploy/helm/sunny-town-hq --namespace sunny-town-hq --create-namespace
kubectl get pods,svc -n sunny-town-hq
kubectl port-forward -n sunny-town-hq svc/sunny-town-hq-sunny-town 18082:18082
kubectl port-forward -n sunny-town-hq svc/sunny-town-hq-ai 18083:18083
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18083/healthz | Select-Object -ExpandProperty Content
```

Learning concepts:

- stateless Deployments
- service discovery
- shared local Secrets
- WebSocket service exposure
- multi-service logs

## Phase 5: Keycloak And Auth Wiring

Goal: add Keycloak, Keycloak Postgres, realm import, and HQ auth URLs.

Deliverables:

- Add Keycloak Postgres workload, PVC, Secret values, Service, and readiness probe.
- Add Keycloak Deployment and Service.
- Mount or otherwise provide `deploy/keycloak/hq-realm.json` for local realm import.
- Wire Keycloak database env vars.
- Wire HQ `KEYCLOAK_ISSUER` to the browser-facing forwarded URL and `KEYCLOAK_JWKS_URL` to the internal Keycloak Service.
- Document clean install and reset behavior for realm import state.

Verification:

```powershell
helm upgrade --install sunny-town-hq deploy/helm/sunny-town-hq --namespace sunny-town-hq --create-namespace
kubectl get pods,svc,pvc -n sunny-town-hq
kubectl port-forward -n sunny-town-hq svc/sunny-town-hq-keycloak 18081:8080
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18081/realms/hq/.well-known/openid-configuration
```

Learning concepts:

- dependent workloads
- config file mounting
- browser-facing URLs versus cluster-internal DNS
- durable database reset behavior

## Phase 6: Full Local Workflow

Goal: make the Kubernetes runtime repeatable enough for normal local development.

Deliverables:

- Add Taskfile commands for kind bootstrap, image build/load, Helm install/upgrade, status, logs, port-forward, reset, and teardown.
- Add deterministic health checks for HQ, Sunny Town, and AI.
- Document multi-terminal or scripted port-forward workflow.
- Update `docs/current/RUNTIME.md` once Kubernetes is usable end to end, while keeping Docker Compose documented as the fallback until explicitly retired.

Verification:

```powershell
task k8s:kind:up
task k8s:images:load
task k8s:install
task k8s:status
task k8s:health
task k8s:reset
```

Learning concepts:

- Helm install and upgrade
- operational commands
- release status
- logs and events
- uninstall versus PVC deletion

## Phase 7: Optional Ingress And Runtime Parity

Goal: improve ergonomics after the port-forwarded workflow is proven.

Deliverables:

- Decide whether to add ingress, Docker Desktop Kubernetes compatibility, or both.
- Preserve the Keycloak issuer/JWKS split.
- Add runtime parity notes for frontend auth, Sunny Town WebSocket, AI, database state, and reset flows.
- Decide when Docker Compose is no longer the preferred local runtime.

Verification depends on the selected ingress or compatibility target.

Learning concepts:

- ingress or local load balancers
- hostnames
- TLS follow-up opportunities
- migration cutover criteria

## Initial Ticket Plan

Phase 1 created the detailed implementation ticket set after auditing the current runtime.

Initial tickets:

- #72: Epic: Migrate Local Runtime To Kubernetes And Helm
- #73: Audit runtime and design Kubernetes/Helm migration plan

Follow-up implementation tickets:

- #74: Add kind bootstrap, local image loading, and Helm chart skeleton.
- #75: Deploy app Postgres and HQ through Helm.
- #76: Deploy Sunny Town and AI through Helm.
- #77: Deploy Keycloak and Keycloak Postgres through Helm.
- #78: Add Kubernetes port-forward workflow and auth/WebSocket verification.
- #79: Add Kubernetes Taskfile operations and current runtime docs.
- #80: Evaluate ingress or Docker Desktop Kubernetes compatibility.

## Open Design Questions For Phase 1

- Resolved: first target is kind.
- Resolved: use first-party simple chart templates for learning instead of community subcharts.
- Resolved: use one umbrella chart first.
- Resolved: use documented port-forward commands before ingress.
- Resolved: represent `HQ_PUBLIC_HOST` as a Helm `publicHost` value, defaulting to `localhost`; derive browser-facing Keycloak issuer and Sunny Town WebSocket URL from it.
- Resolved: keep local-only development secrets in Helm values/Secret templates with clear warnings for the first pass.
- Resolved: Helm uninstall should not silently delete PVCs. Provide explicit reset commands that delete PVCs and namespace when the user wants a clean database/realm state.
- Still open: the exact Keycloak health endpoint/probe shape should be verified against `quay.io/keycloak/keycloak:26.0` during implementation.
- Still open: Docker Compose is superseded only after Kubernetes can run the full browser login, HQ API/static host, Sunny Town WebSocket, AI fake grading service, app Postgres, and Keycloak reset flow.

## Learning Checklist

The migration should intentionally exercise these Kubernetes concepts:

- Pods and Deployments for application workloads.
- Services for cluster-internal discovery.
- ConfigMaps and Secrets for runtime configuration.
- PersistentVolumeClaims for Postgres-backed state.
- Probes for container health.
- Jobs or init containers for setup work if needed.
- Ingress, NodePort, LoadBalancer, or port-forwarding for local browser access.
- Helm charts, templates, values, releases, upgrades, rollbacks, and uninstall/reset flows.
- Local image build and cluster image loading.
- Debugging with `kubectl get`, `describe`, `logs`, `exec`, and events.

## Notes

This epic is a deliberate learning migration. Do not re-litigate whether Kubernetes is necessary for production before doing the work. The accepted product direction is to convert the project runtime to Kubernetes because the project owner wants practical Kubernetes and Helm experience.
