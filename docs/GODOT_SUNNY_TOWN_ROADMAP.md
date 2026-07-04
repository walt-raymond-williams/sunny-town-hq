# Godot Sunny Town Roadmap

Epic: #61, "Godot Web Client for Sunny Town RPG Surface"

## Outcome

Replace the current Sunny Town Vue/canvas RPG surface with a Godot-powered web client embedded in the existing HQ web app.

The migration must preserve the current backend ownership model:

- HQ owns durable account, student, wallet, inventory, equipment, hotbar, ledgers, map objects, containers, NPC production storage, progression, and saved return-position state.
- Sunny Town owns live realtime world state, accepted movement, room/map membership, portals, collectibles, resource nodes, placed objects, fixtures, NPCs, container access validation, interactions, rewards validation, and gameplay validation.
- Browser clients, including Godot, send requests. They are not authority.

This epic is web-only. It targets normal desktop browsers and phone/mobile browsers. It does not include standalone desktop, native mobile, Steam, or other app-store distribution.

## Direction

Use a staged migration:

1. Make the architecture decision explicit.
2. Add a minimal embedded Godot web skeleton.
3. Integrate Godot export/build/runtime into HQ.
4. Connect Godot to the existing HQ session and Sunny Town WebSocket flow.
5. Render current Sunny Town world state.
6. Match current movement and remote interpolation behavior.
7. Add phone browser controls.
8. Port current interactions.
9. Add enough HUD/hotbar/inventory/equipment parity for cutover.
10. Cut over only after regression and mobile smoke checks pass.

The current canvas client remains valuable as a reference and debug client. Keep it available until the Godot client has passed the agreed parity checklist and the project owner approves a later removal issue.

## Target Architecture

Recommended architecture:

```text
Browser
  Vue HQ shell
    route guard and student auth
    POST /api/student/sunny-town/session
    Godot web export loader
    Back/fallback controls
    optional bridge to Vue-owned overlays during migration

  Godot web client
    in-game rendering
    desktop and mobile input
    WebSocket protocol client
    prediction/interpolation
    in-game HUD and prompts

        |
        | existing join token
        v

Sunny Town WebSocket service
  server-authoritative accepted movement
  live rooms/maps/world state
  gameplay validation
  service-authenticated HQ calls

        |
        v

HQ
  durable identity, wallet, inventory, hotbar, equipment, ledgers,
  map objects, containers, progression, saved position
```

Vue should obtain the Sunny Town session and pass the session payload into Godot. Godot should not own Keycloak/HQ authentication in the first epic.

## Architecture Decision (#62)

Decision date: 2026-07-04.

Godot Web export is the target for this epic. The scope remains web-only: desktop browsers and phone/mobile browsers served through the existing HQ web app. Standalone desktop, native mobile, Steam, app-store packaging, PWA/offline install behavior, and polished art production are not part of the first Godot Sunny Town slice.

### Godot Project Location

Place the committed Godot source project at:

```text
godot/sunny-town/
```

This keeps the Godot client separate from the existing Go `sunny-town/` realtime service and from the Vue `frontend/` app while still making ownership clear. The project should contain Godot source files, export presets, scripts, scenes, and committed first-party assets needed to reproduce the client. Do not place Godot source under `frontend/` or `sunny-town/`; those directories already have separate ownership boundaries.

### Web Export Artifacts

Treat Godot web export output as generated runtime assets, not source. The first implementation should export locally into an ignored generated path under the existing static host output:

```text
web/godot/sunny-town/
```

`web/` is already ignored and populated by `npm run build`, then copied into the HQ Docker image. A later build/runtime issue should decide the exact command wiring, but the artifact rule is fixed here: commit Godot source and export presets, do not commit generated `.wasm`, `.pck`, generated JavaScript glue, export HTML, or generated cache files unless a later release-packaging decision explicitly changes repo policy.

During development, implementation agents may use a local generated export directory for smoke testing, but commits should stay limited to source, scripts, docs, and non-generated assets. Docker/HQ serving should continue to use the Go static host rooted at `web/`.

### Vue Wrapper Route Shape

Keep the existing canvas client available until cutover:

```text
/student/pet/sunny-town
```

During Godot implementation, add a separate Vue wrapper route for the embedded Godot client:

```text
/student/pet/sunny-town/godot
```

Also add an explicit canvas fallback/debug route when the Godot route lands:

```text
/student/pet/sunny-town/canvas
```

Before cutover, `/student/pet/sunny-town` may continue to resolve to the current canvas page. At cutover, `/student/pet/sunny-town` can become the Godot default only after the parity checklist and mobile smoke checks pass, while `/student/pet/sunny-town/canvas` remains available for regression/debug access. A query flag such as `?client=canvas` may be added as a convenience, but the durable fallback should be a route so it is bookmarkable and easy to document.

The Vue Godot wrapper is responsible for route access checks, session creation, loading the Godot web export, displaying load/error/fallback controls, and navigating back to the normal student UI. It should not duplicate Sunny Town gameplay authority in Vue.

### Vue-To-Godot Session Handoff

Vue remains the owner of browser authentication for the first epic. The wrapper should call the existing authenticated endpoint:

```text
POST /api/student/sunny-town/session
```

The wrapper then passes the normalized session payload into Godot before Godot opens the Sunny Town WebSocket. The first-pass handoff should use a small browser bridge exposed by the wrapper, such as a page global or post-load JavaScript callback, containing only the Sunny Town join data Godot needs:

- `roomId`
- `mapId`
- `characterId`
- `avatarId`
- `websocketUrl`
- `joinToken`
- `expiresAt`
- initial wallet/inventory/hotbar summary when needed for first HUD state

Godot should then connect to the existing Sunny Town WebSocket endpoint using the join token. Godot must not own Keycloak login, bearer-token refresh, student route guards, or direct HQ durable mutation calls in this epic. Browser and Godot messages remain requests; Sunny Town continues to validate realtime gameplay against accepted server position and owned hotbar-active tools, and HQ remains the durable owner for account, wallet, inventory, equipment, ledgers, map objects, containers, progression, and saved return position.

### Canvas Fallback And Debug Client

The current Vue/canvas implementation remains the reference client and debug fallback through this epic. Do not remove canvas rendering, current Vue Sunny Town composables, or current overlays until a later cutover issue proves Godot parity and the project owner approves removal.

The fallback is required for:

- comparing Godot behavior against the current movement, map, interaction, and inventory behavior;
- recovering from Godot web export failures on mobile browsers;
- debugging Sunny Town backend protocol regressions independently of Godot;
- preserving a working Sunny Town route while the Godot client is incomplete.

### Mobile Browser Constraints

First-pass Godot export should optimize for compatibility rather than maximum performance. The initial target is a non-threaded Godot Web export using the Compatibility renderer where possible. Defer threaded exports, cross-origin isolation header requirements, PWA/offline install behavior, and service-worker export behavior until a measured performance need exists.

Implementation and verification should assume:

- Godot web exports require browser support for WebAssembly and WebGL 2.0.
- Browser tab backgrounding may pause Godot processing, so reconnect/session behavior must tolerate visibility changes and mobile browser suspension.
- Phone browsers have tighter memory, GPU, audio unlock, viewport, and touch-input constraints than desktop browsers.
- The game surface must respect safe areas, orientation changes, and browser chrome resizing.
- The first mobile verification target is a LAN phone browser against the Docker-served HQ runtime at `http://<LAN-IP>:18080`, using the same host/IP consistently for HQ and Keycloak.

Reference: Godot's official web export documentation at `https://docs.godotengine.org/en/latest/tutorials/export/exporting_for_web.html`.

### First-Pass UI Ownership

Godot should own the in-game surface as soon as each feature lands:

- world rendering;
- local movement input and prediction;
- remote player interpolation;
- camera behavior;
- touch movement/action controls;
- proximity prompts;
- lightweight HUD/status/toasts;
- selected hotbar display;
- simple equipment visuals.

Vue may temporarily own complex overlays during migration through a narrow bridge:

- shop purchase panels;
- schoolwork/assignment panels;
- chest/container transfer panels;
- broader inventory/crafting/equipment management where the existing Vue implementation is still the source of parity.

The bridge should be treated as migration scaffolding, not a new authority layer. Godot can request an overlay or action; Sunny Town and HQ must still validate access, proximity, inventory, wallet, equipment, and durable mutations according to the existing service boundaries.

## Behavior Parity Requirements

The Godot client must account for these current behaviors before cutover:

- Sunny Town session creation from HQ.
- Join token flow.
- WebSocket connection to Sunny Town.
- `hello`, `snapshot`, `map_changed`, reward/resource/object/container result messages, and `error`.
- Local player movement with sequence-numbered samples.
- Remote player interpolation.
- Current JSON map loading/rendering.
- Camera following and map clamping.
- Static blocked rectangles and active colliding world objects.
- Portals and map transitions.
- Collectible stars and server-authoritative wallet rewards.
- NPC rendering, routine status cues, proximity prompts, dialogue, shops, and schoolwork triggers.
- Inventory display decisions.
- Hotbar selected item behavior.
- Equipment visuals for local and remote players.
- Pickaxe tool use for resource nodes and placed stone blocks.
- Resource nodes, hit/depleted state, respawn visibility, and committed resource feedback.
- Placed objects, placement preview, placement/removal messages, and persistence through HQ.
- Fixture objects, including chests and beds.
- Container/chest open and transfer flows where currently supported.
- HUD, toasts, connection status, and errors.
- Desktop keyboard controls.
- Phone/mobile browser touch controls.
- Full-viewport responsive game surface.
- Back/exit path to normal HQ/student UI.
- Canvas fallback/debug client.

## Child Issues

### #62 Decide Godot web integration architecture for Sunny Town

Type: research/docs

Goal: capture the architecture decision before implementation starts.

Acceptance summary:

- Godot Web export target is confirmed.
- Vue wrapper route and session handoff are specified.
- Canvas fallback strategy is specified.
- Godot versus Vue UI ownership is decided for the first epic.
- Mobile browser constraints are documented.

### #63 Add minimal Godot web client skeleton for Sunny Town

Type: feature

Goal: add the smallest embedded Godot project and placeholder route.

Acceptance summary:

- Minimal Godot project exists.
- Placeholder scene loads in HQ.
- Current canvas client remains reachable.
- Export/run docs exist.
- Desktop and phone browser smoke checks pass.

### #64 Wire Godot web export into HQ build/runtime

Type: feature

Goal: make Godot web export repeatable in local and Docker-served HQ runtime.

Acceptance summary:

- Static output path is clear.
- Build/runtime commands are documented.
- HQ can serve the Godot placeholder from Docker.
- Generated assets follow repo hygiene rules.

### #65 Connect Godot client to Sunny Town session and WebSocket

Type: feature

Goal: connect Godot to the existing HQ session and Sunny Town WebSocket protocol.

Acceptance summary:

- Vue passes session data to Godot.
- Godot opens the WebSocket with the join token.
- Godot parses `hello`, `snapshot`, and `map_changed`.
- Debug state proves live data is flowing.

### #66 Render current Sunny Town maps and world snapshots in Godot

Type: feature

Goal: render current server-delivered world state with simple placeholder visuals.

Acceptance summary:

- All checked-in maps render.
- Players, NPCs, collectibles, portals, blocked rectangles, resource nodes, placed objects, chests, and beds render.
- `map_changed` clears stale state.

### #67 Implement Godot movement, prediction, and remote interpolation

Type: feature

Goal: preserve the current movement model.

Acceptance summary:

- WASD and arrow-key movement work.
- Godot sends `move` messages with sequence numbers.
- Local prediction does not rubber-band during normal play.
- Remote players interpolate smoothly.
- Portals remain server-driven.

### #68 Add phone browser controls for Godot Sunny Town

Type: feature

Goal: make the Godot client playable in a phone/mobile browser.

Acceptance summary:

- Touch movement works.
- Touch action/hotbar controls work.
- HUD respects small screens and safe areas.
- LAN phone browser smoke check is recorded.

### #69 Port Sunny Town interactions to Godot with Vue overlay bridge

Type: feature

Goal: support current interaction behavior without moving authority out of Sunny Town/HQ.

Acceptance summary:

- NPC dialogue works.
- Shop and schoolwork paths work through Godot and/or Vue bridge.
- Pickaxe mining and placed-block removal work.
- Stone-block placement works.
- Chests/containers work for current fixture roles.

### #70 Add Godot HUD, hotbar, inventory summary, and equipment visuals

Type: feature

Goal: add enough in-game UI parity for cutover.

Acceptance summary:

- Connection/status, star balance, player count, toasts, errors, prompts, selected hotbar, and inventory feedback display.
- Equipment visuals render from snapshots.
- Vue ownership for broader HQ screens remains clear.

### #71 Prepare Godot Sunny Town cutover and fallback validation

Type: feature/docs

Goal: make Godot the default only after a complete parity and mobile checklist passes.

Acceptance summary:

- Cutover switch/route is documented.
- Canvas fallback remains available.
- Regression checklist passes.
- `docs/current/` is updated for accepted architecture/runtime changes.

## Suggested Integration Branch

Use a feature integration branch:

```text
codex/godot-sunny-town-dev
```

Agents should work child issues against that branch once it exists. `main` should remain stable until the integrated Godot slice is ready for project-owner review.

## Verification Baseline

Deterministic checks:

```powershell
go test ./...
cd frontend
npm run build
```

Runtime checks:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual parity checks should cover:

- Student login and Sunny Town session creation.
- Godot export load from HQ.
- Godot export load from phone browser.
- Two clients seeing each other.
- Movement and remote interpolation.
- Portal transitions across every current map.
- Inventory/hotbar/equipment sync.
- NPC dialogue, shop, and schoolwork interactions.
- Pickaxe resource use.
- Reward and resource ledger behavior.
- Stone block placement/removal and persistence.
- Chest/container interactions.
- Canvas fallback/debug route.

## Deferred Follow-ups

Create later issues only after the first Godot web client is proven:

- Polished art and animation.
- Godot-native map tooling or editor workflow.
- Full Godot inventory drag/drop replacement if Vue bridge is retained.
- PWA/offline install behavior.
- Threaded web export and cross-origin-isolation headers, if performance requires it.
- Native mobile, standalone desktop, or other distribution targets.
