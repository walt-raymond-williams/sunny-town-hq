# Keycloak Local Setup

HQ now uses Keycloak for login and app-owned database rows for grades, cookies, and pet state.

## Run Services

```powershell
docker compose -f deploy/docker-compose.yml up -d
```

Keycloak admin:

```text
http://localhost:18081
```

Default bootstrap admin:

```text
admin / admin
```

## Create Users

The `hq` realm, `hq-web` public client, and `teacher` / `student` roles are imported automatically from `deploy/keycloak/hq-realm.json`.

In the Keycloak admin console:

1. Switch to the `hq` realm.
2. Create a user for each teacher/student.
3. Set a password in the user's Credentials tab.
4. Assign realm role `teacher` or `student`.

## Run HQ

Use the same laptop IP/hostname for HQ and Keycloak when logging in from multiple devices. For example, if the laptop IP is `<YOUR_LAN_IP>`:

```powershell
$env:DATABASE_URL="postgres://hq:hq@localhost:55432/hq?sslmode=disable"
$env:HQ_PORT="18080"
$env:KEYCLOAK_ISSUER="http://<YOUR_LAN_IP>:18081/realms/hq"
$env:KEYCLOAK_AUDIENCE="hq-web"
go run ./cmd/hq
```

Then open:

```text
http://<YOUR_LAN_IP>:18080
```

The frontend derives Keycloak's URL from the current browser hostname and port `18081` by default. Override it with `VITE_KEYCLOAK_URL` if needed.
