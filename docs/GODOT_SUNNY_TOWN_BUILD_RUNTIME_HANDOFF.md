# Godot Sunny Town Build Runtime Handoff

## Purpose

This handoff starts GitHub issue #64, "Wire Godot web export into HQ build/runtime."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/64

The goal is to make the Godot web export repeatable in local development and Docker-served HQ runtime.

## Current Status

- Blocked by #63.
- The skeleton should already load manually before this issue starts.

## Product Direction

- Avoid hidden manual steps for future agents.
- Keep generated or bulky local export artifacts out of commits unless the architecture decision says otherwise.
- HQ remains the web host for the embedded client.

## Current Code Surface

- Godot project and export paths chosen by #62/#63.
- `frontend/package.json`
- `frontend/vite.config.ts`
- `deploy/hq/Dockerfile`
- `deploy/docker-compose.yml`
- `.gitignore`
- `.dockerignore`
- `docs/current/RUNTIME.md`

## Recommended Implementation Plan

1. Inspect the skeleton export path and current frontend build output.
2. Decide whether Godot export artifacts are:
   - generated into an ignored local folder,
   - copied into `frontend/public/`,
   - produced during Docker image build,
   - or committed as a minimal exported placeholder.
3. Add scripts or documented commands for the selected workflow.
4. Ensure HQ serves `.wasm`, `.pck`, `.js`, and `.html`/loader assets correctly.
5. Confirm Docker build copies or generates the expected static files.
6. Update runtime docs.

## Out Of Scope

- Gameplay WebSocket integration.
- Map rendering.
- Protocol changes.

## Verification

Run:

```powershell
cd frontend
npm run build
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual smoke:

- Open the Docker-served Godot route.
- Check browser devtools for missing export files, MIME/static serving issues, or load errors.

## Close Criteria

Close #64 when:

- Build/runtime workflow is reproducible.
- Static output rules are documented.
- Docker-served HQ can load the Godot placeholder.
- Generated artifacts follow repo hygiene rules.
