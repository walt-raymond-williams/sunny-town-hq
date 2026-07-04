# Sunny Town Godot Client

This is the source project for the Godot web client that will replace the Vue/canvas Sunny Town surface over the Godot epic.

The first scene is intentionally only a load marker. It does not connect to Sunny Town, render maps, process movement, or implement gameplay.

## Export

Use Godot 4 with the Compatibility renderer and a non-threaded Web export:

```powershell
godot --headless --path godot\sunny-town --export-release "Sunny Town Web"
```

The preset writes generated files to:

```text
web/godot/sunny-town/
```

That output is generated runtime content and is ignored by git. Run the frontend build before exporting if `web/` was cleared:

```powershell
cd frontend
npm run build
cd ..
godot --headless --path godot\sunny-town --export-release "Sunny Town Web"
```

Then serve HQ and open:

```text
/student/pet/sunny-town/godot
```

The current Vue/canvas client remains available at:

```text
/student/pet/sunny-town
/student/pet/sunny-town/canvas
```
