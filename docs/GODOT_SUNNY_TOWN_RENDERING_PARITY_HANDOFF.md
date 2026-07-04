# Godot Sunny Town Rendering Parity Handoff

## Purpose

This handoff starts GitHub issue #66, "Render current Sunny Town maps and world snapshots in Godot."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/66

The goal is simple visual parity from existing server-delivered map and snapshot data.

## Current Status

- Blocked by #65.
- Godot should already connect and parse live protocol state before this starts.

## Product Direction

- Render current behavior first.
- Keep visuals intentionally basic.
- Existing Sunny Town JSON maps remain source of truth.

## Current Code Surface

Map data:

- `sunny-town/maps/*.json`

Frontend reference:

- `frontend/src/features/sunny-town/rendering/mapDrawing.ts`
- `frontend/src/features/sunny-town/rendering/objectDrawing.ts`
- `frontend/src/features/sunny-town/rendering/characterDrawing.ts`
- `frontend/src/features/sunny-town/worldObjects.ts`
- `frontend/src/composables/useSunnyTownWorldState.ts`

Backend protocol:

- `internal/sunnytown/protocol/protocol.go`
- `internal/sunnytown/server/world_snapshots.go`

## Recommended Implementation Plan

1. Build a Godot map-state reducer for `hello`, `snapshot`, and `map_changed`.
2. Render map bounds and a simple background/grid.
3. Render blocked rectangles.
4. Render portals.
5. Render collectibles and active/inactive state.
6. Render world objects:
   - natural resource nodes,
   - placed stone blocks,
   - fixture chests,
   - fixture beds.
7. Render NPCs with name/status/facing metadata.
8. Render players with local/remote distinction.
9. Clear/rebuild scene state on `map_changed`.

## Out Of Scope

- Polished sprites.
- Animation polish.
- Movement prediction.
- Interaction behavior.
- Godot map editor tooling.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual smoke:

- Visit `sunny-town-v1`.
- Visit `forest-crossing-v1`.
- Visit `sunny-town-house-1`.
- Visit `sunny-town-classroom`.
- Visit `sunny-town-cookie-keeper-home`.
- Confirm expected players, NPCs, collectibles, portals, blockers, resources, fixtures, and placed objects render.

## Close Criteria

Close #66 when:

- Every current map can render from server data.
- `map_changed` does not leave stale state.
- Placeholder visuals are sufficient for movement and interaction work.
