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
