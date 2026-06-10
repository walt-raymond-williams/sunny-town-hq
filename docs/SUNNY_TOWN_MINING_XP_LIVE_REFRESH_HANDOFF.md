# Sunny Town Mining XP Live Refresh Handoff

## Purpose

This handoff starts GitHub issue #37, "Refresh mining XP in open character panel after mining."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/37

The goal is to fix a PR review UX finding: mining XP should refresh in the open character panel after a successful mining commit, without requiring the player to close and reopen the E menu.

## Current Status

- PR review marked this as non-blocking, but it is a good polish fix before merge if time allows.
- `SunnyTownInventoryPanel.vue` loads progression on mount.
- `resource_committed` currently refreshes inventory/crafting state but not character progression.
- This issue should ideally land after #36 because mining commit semantics are being tightened there.

## Product Direction

- Mining XP is real progression now, so the character panel should feel live.
- Avoid noisy polling or unconditional progression refreshes while the character panel is closed.
- Keep the existing mount-time progression load as a fallback.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - Handles Sunny Town websocket messages and resource commits.
- `frontend/src/composables/useSunnyTownMessageEffects.ts`
  - Applies resource commit effects and refreshes inventory/crafting.
- `frontend/src/composables/useSunnyTownMessageEffects.test.ts`
  - Add or update tests for progression refresh behavior.
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - Loads and passes progression state into character preview.
- `frontend/src/stores/characterProgression.ts`
  - Store action for loading current progression.
- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
  - Displays mining progress and error/loading states.

## Recommended Implementation Plan

1. Inspect how inventory panel open/closed state is represented in Sunny Town page/composables.
2. Add a progression refresh callback to the resource-commit message effect path.
3. Gate the refresh so it runs only when the E menu/character panel is open or when the store/UI naturally needs it.
4. Preserve existing inventory and crafting refresh behavior.
5. Add focused frontend tests for the refresh trigger.
6. Run frontend build/tests and `task verify`.

## Out Of Scope

- Polling progression.
- Redesigning the character panel.
- Changing XP award amounts or level curves.
- Backend mining XP idempotency changes unless #36 requires frontend contract adjustments.

## Verification

Run:

```powershell
cd frontend
npm run build
npm run test
task verify
```

Manual smoke if practical:

- Open the E menu to the character/inventory panel.
- Mine a resource.
- Verify mining progress updates without closing and reopening the panel.

## Close Criteria

Close #37 when:

- open character panel refreshes mining XP after successful mining commits
- tests cover the refresh behavior
- verification results are recorded on the issue
- work is committed and pushed to `codex/inventory-redesign-dev`
