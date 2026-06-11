# Current Runtime

Docker Compose is the preferred integration runtime.

## Services

```text
hq                  Go API/static host, http://localhost:18080
sunny-town          Realtime WebSocket service, http://localhost:18082
ai                  AI grading service, http://localhost:18083
postgres            App PostgreSQL, localhost:55432
keycloak            Auth server, http://localhost:18081
keycloak-postgres   Keycloak PostgreSQL
```

## Common Commands

With Go Task installed:

```powershell
task test
task frontend:build
task compose:up
task compose:rebuild-runtime
task health
task verify
```

Equivalent direct commands:

```powershell
go test ./...
cd frontend
npm run build
cd ..
docker compose -f deploy\docker-compose.yml up -d --build
```

Health checks:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected response for both is:

```text
ok
```

## Local Network Use

Set the public host before starting Compose when testing from phones or tablets on the same Wi-Fi:

```powershell
"HQ_PUBLIC_HOST=<YOUR_LAN_IP>" | Set-Content deploy/.env
docker compose -f deploy\docker-compose.yml up -d --build
```

Use one consistent host/IP for HQ and Keycloak. A token issued for `localhost` will not match a browser session opened through the laptop LAN IP.

## Sunny Town Runtime Settings

Sunny Town supports `SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES` to control the server-owned simulated NPC day cadence. If unset or invalid, it defaults to `24` real minutes per simulated day. Demo/debug runs can set it to `8` to make NPC morning/day/evening/night behavior cycle quickly without giving clients control over simulation time.
