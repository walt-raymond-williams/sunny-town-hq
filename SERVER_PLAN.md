# HQ Stage 1 Server Proof

This is the first small checkpoint for HQ: prove that the laptop can serve a real website, accept a form submission, and save an assignment record to PostgreSQL running in Docker Desktop.

## Goal

Serve the built Vue 3 + Vuetify website from a Go web server.

The current server has three pieces:

- `/` serves the static website in `web/`
- `/api/assignments` saves assignment records
- `/healthz` returns `ok` for a quick server check

## Why This Shape

For the first version, Go should serve both the API and the built frontend from one port. That keeps the home-network setup simple because phones and other devices only need one URL, such as:

```text
http://<YOUR_LAN_IP>:8080
```

The frontend source lives in `frontend/`. Vite builds production assets into `web/`, and Go serves those files.

In development, Vite can run separately for hot reload. For the laptop-hosted version, Vue builds static files and Go serves those files.

## Runtime Decision

For now, run the Go server directly on the laptop. It is lighter, easier to debug, and gives a faster edit/run loop.

Use Docker Compose for PostgreSQL when the database is added. That gives the project a portable database setup without forcing the Go app into a container immediately.

Chosen setup:

```text
Go server:  local process
PostgreSQL: Docker Compose with official postgres image on localhost:55432
```

HQ will not use the PostgreSQL install on the host machine. The project database should come from Docker Compose so the expected database version, port, database name, and credentials are documented in code.

The container uses port `5432` internally, but the laptop maps it to `55432` because the Windows host already has PostgreSQL listening on `5432`.

## Run It

Install Go if it is not already available on PATH.

Start PostgreSQL:

```powershell
docker compose -f deploy/docker-compose.yml up -d
```

Then run the Go server:

```powershell
$env:DATABASE_URL="postgres://hq:hq@localhost:55432/hq?sslmode=disable"
go run ./cmd/hq
```

After frontend changes, rebuild the Vue app:

```powershell
cd frontend
npm run build
```

The server listens on:

```text
http://localhost:8080
```

It also binds to `0.0.0.0`, which means other devices on the same Wi-Fi can use the laptop's local IP address.

## Network Checklist

1. Start the server with `go run ./cmd/hq`.
2. Open `http://localhost:8080` on the laptop.
3. Find the Wi-Fi address printed by the server, such as `http://<YOUR_LAN_IP>:8080`.
4. Open that address on a phone connected to the same Wi-Fi.
5. If the phone cannot connect, allow the app or port `8080` through Windows Firewall.

## Next Step

Once this proof works locally with Go installed, the next stage is to split the teacher and student workflows and later replace the static page with a Vue 3 frontend while keeping the Go server as the single home-network entry point.
