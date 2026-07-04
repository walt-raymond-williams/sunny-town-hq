# Godot Sunny Town Mobile Controls Handoff

## Purpose

This handoff starts GitHub issue #68, "Add phone browser controls for Godot Sunny Town."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/68

The goal is to make the Godot web client playable in a phone/mobile browser.

## Current Status

- Blocked by #67.
- Desktop movement should already work before this starts.

## Product Direction

- This epic remains web-only.
- A phone browser is a required target.
- Do not plan native iOS or Android apps in this epic.

## Current Code Surface

- Godot input and HUD scripts from prior issues.
- `frontend/src/style.css`
- `frontend/src/App.vue`
- `docs/current/RUNTIME.md`
- `README.md` local-network workflow.

## Recommended Implementation Plan

1. Add virtual movement control:
   - joystick or compact D-pad.
2. Add touch action controls:
   - primary interact/use,
   - cancel/back or close overlay,
   - inventory/hotbar access as needed.
3. Add touch hotbar selection.
4. Handle safe-area insets and narrow viewport layout.
5. Handle orientation changes without overlapping critical controls.
6. Keep desktop keyboard/mouse behavior intact.
7. Document the phone browser smoke workflow.

## Out Of Scope

- Native mobile app export.
- App-store packaging.
- Full PWA/offline install behavior unless required by the web export decision.
- New gameplay.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual phone smoke:

```powershell
"HQ_PUBLIC_HOST=<YOUR_LAN_IP>" | Set-Content deploy/.env
docker compose -f deploy\docker-compose.yml up -d --build
```

Then on a phone browser:

- Open `http://<YOUR_LAN_IP>:18080`.
- Log in as a student.
- Enter Godot Sunny Town.
- Move, stop, select hotbar, use/interact, and exit.
- Record device, browser, host, and result in the issue close comment.

## Close Criteria

Close #68 when:

- Phone browser movement and action controls work.
- HUD/control layout is acceptable on narrow screens.
- Desktop controls still work.
- Manual phone smoke result is recorded.
