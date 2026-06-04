# HQ Stage 1 Server Proof

This document describes the current local server setup for HQ: the laptop serves the built Vue website, accepts JSON and Connect RPC API requests, and saves assignment and pet records to PostgreSQL running in Docker Desktop.

## Goal

Serve the built Vue 3 + Vuetify website from a Go web server.

The current server has four pieces:

- `/` serves the static website in `web/`
- `/api/...` exposes student and teacher JSON endpoints
- `/hq.pet.v1.PetService/...` exposes the generated Connect RPC pet service
- `/healthz` returns `ok` for a quick server check

## Why This Shape

For the first version, Go should serve both APIs and the built frontend from one port. That keeps the home-network setup simple because phones and other devices only need one URL, such as:

```text
http://<YOUR_LAN_IP>:8080
```

The frontend source lives in `frontend/`. Vite builds production assets into `web/`, and Go serves those files. The pet UI uses generated protobuf TypeScript in `frontend/src/gen/` through the Pinia store in `frontend/src/stores/studentPet.js`.

In development, Vite can run separately for hot reload. For the laptop-hosted version, Vue builds static files and Go serves those files.

## Runtime Decision

For now, run the Go server directly on the laptop. It is lighter, easier to debug, and gives a faster edit/run loop.

Use Docker Compose for PostgreSQL. That gives the project a portable database setup without forcing the Go app into a container immediately.

Chosen setup:

```text
Go server:  local process
PostgreSQL: Docker Compose with official postgres image on localhost:55432
```

HQ will not use the PostgreSQL install on the host machine. The project database should come from Docker Compose so the expected database version, port, database name, and credentials are documented in code.

The container uses port `5432` internally, but the laptop maps it to `55432` because the Windows host already has PostgreSQL listening on `5432`.

## Run It

Start PostgreSQL:

```powershell
docker compose -f deploy/docker-compose.yml up -d
```

Then run the Go server locally if Go is installed:

```powershell
$env:DATABASE_URL="postgres://hq:hq@localhost:55432/hq?sslmode=disable"
go run ./cmd/hq
```

On the current laptop, Go is not on PATH, so the server has been run with a disposable Go container instead:

```powershell
docker run --rm --name hq-demo-server `
  -p 18080:8080 `
  -e DATABASE_URL="postgres://hq:hq@host.docker.internal:55432/hq?sslmode=disable" `
  -v "${PWD}:/app" `
  -w /app `
  golang:1.22-alpine `
  go run ./cmd/hq
```

After frontend changes, rebuild the Vue app:

```powershell
cd frontend
npm run build
```

After protobuf changes, regenerate the Go and TypeScript clients:

```powershell
npx buf generate
```

The default server port inside the app is:

```text
http://localhost:8080
```

The currently running Docker helper maps that to:

```text
http://127.0.0.1:18080
```

It also binds to `0.0.0.0`, which means other devices on the same Wi-Fi can use the laptop's local IP address and the mapped port.

## Network Checklist

1. Start the server with `go run ./cmd/hq` or the Docker helper command above.
2. Open `http://localhost:8080` on the laptop, or `http://127.0.0.1:18080` when using the Docker helper.
3. Find the Wi-Fi address printed by the server, such as `http://<YOUR_LAN_IP>:8080`.
4. Open that address on a phone connected to the same Wi-Fi.
5. If using the Docker helper, use the mapped host port, such as `http://<YOUR_LAN_IP>:18080`.
6. If the phone cannot connect, allow the app or port through Windows Firewall.

## Current Website Workflows

The built Vue app currently includes:

- Splash page with Student and Teacher choices.
- Student answer tab for one unanswered question at a time, with a subject filter for All, Math, Science, and Reading.
- Student answer tab shows prior attempts below the answer form as expandable items.
- Student grade tab with category pass percentages, top-level pass/fail indicators, and graded attempts grouped by question.
- Student Pet tab shows the current cookie count, hunger, happiness, and energy from the student pet store.
- Student Pet tab can feed the pet by spending one cookie and increasing hunger by 10.
- Student page uses generated Connect/Protobuf clients for pet state, feed, play, sleep, wake, and live state streaming. Playing increases happiness and spends 10 energy.
- Student page opens a pet-state stream while visible so the floating avatar and Pet tab stats update without polling.
- Waking the pet early stops sleep recovery and subtracts 30 happiness; letting sleep finish naturally adds 10 happiness.
- Student virtual pet avatar remains visible on Student pages without the cookie counter badge.
- Teacher login with the hard-coded demo password `local-demo-password`.
- Teacher logout that clears the teacher cookie and returns to the splash page.
- Unified Teacher Desk questions workspace.
- Collapsible New Question form for category, prompt, and expected answer.
- Question filters for Needs Review, All, Unanswered, Reset, and Graded.
- Expanded question actions for pass/fail, feedback, reset that saves feedback, attempt history, and delete.
- Passing a submitted attempt awards one cookie to the demo student, once per attempt.
