# Godot Sunny Town Skeleton

Issue #63 adds the smallest source project and wrapper needed to prove that a Godot Web export can be embedded in the HQ web app.

## Source And Generated Paths

- Godot source project: `godot/sunny-town/`
- Generated Web export output: `web/godot/sunny-town/`
- Embedded Godot route: `/student/pet/sunny-town/godot`
- Canvas fallback/debug routes: `/student/pet/sunny-town` and `/student/pet/sunny-town/canvas`

`web/` is ignored and should not be committed. Commit Godot source files, scenes, scripts, export presets, and docs only.

## Local Export

Install Godot 4 with Web export templates, then run from the repository root:

```powershell
cd frontend
npm run build
cd ..
godot --headless --path godot\sunny-town --export-release "Sunny Town Web"
```

The frontend build clears and recreates `web/`, so export Godot after `npm run build` when manually smoke-testing the embedded route.

## Smoke Test

Start HQ after exporting, then open:

```text
http://127.0.0.1:18080/student/pet/sunny-town/godot
```

Confirm the iframe displays the Godot placeholder marker. Then open the canvas fallback/debug route:

```text
http://127.0.0.1:18080/student/pet/sunny-town/canvas
```

This skeleton intentionally does not create a Sunny Town session for Godot, open a WebSocket, render maps, handle movement, or show polished art.
