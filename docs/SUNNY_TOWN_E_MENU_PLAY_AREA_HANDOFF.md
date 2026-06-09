# Sunny Town E Menu Play Area Handoff

## Purpose

This handoff starts GitHub issue #21, "Expand E menu to use the play area."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/21

The goal is to give the `E` inventory/crafting menu enough space for the current redesign before adding more character, stats, crafting, or tabbed-menu work.

## Current Status

The inventory redesign is on `codex/inventory-redesign-dev`.

Completed foundation:

- #15 inventory grid slot-to-slot drag/drop is complete.
- #16 inventory-to-hotbar drop targets are complete.
- #17 inventory-to-equipment drop targets are complete.
- #18 legacy selected-item assignment buttons are removed.
- #19 character preview shell is complete and verified in commit `20bac63`.

Pending work:

- #20 stats-ready character panel is open, but should wait until this layout/capacity fix is done.

## Problem

The current `E` menu is still shaped like a small debugging tray. It is capped around a narrow desktop width and anchored near the top-right of the game surface.

That is too cramped for the intended final direction:

- inventory grid
- selected item detail
- hotbar drop targets
- equipment drop targets
- character preview
- crafting panel
- future stats and tabs

For now, it is acceptable for the `E` menu to use most or all of the Sunny Town play area.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - owns the inventory overlay contents
  - renders inventory, equipment, preview, hotbar editor, selected item details, and crafting
- `frontend/src/features/sunny-town/SunnyTownPage.vue`
  - mounts `SunnyTownInventoryPanel`
  - likely defines the game surface context around the overlay
- `frontend/src/style.css`
  - contains the current overlay layout styles

Important selectors:

- `.game-overlay`
- `.sunny-town-inventory-tray`
- `.sunny-town-inventory`
- `.sunny-town-crafting`
- `.sunny-town-hotbar-editor`
- `.sunny-town-hotbar-editor__slots`
- `.sunny-town-inventory-grid`
- `.sunny-town-selected-item`
- `.sunny-town-character-preview`

Current `.sunny-town-inventory-tray` constraints to inspect:

- `grid-template-columns: minmax(260px, 340px) minmax(300px, 360px)`
- `height: min(640px, calc(100vh - 72px))`
- `max-width: min(756px, calc(100% - 24px))`
- `position: absolute`
- `right: 12px`
- `top: 54px`
- `width: max-content`

## Recommended Implementation Plan

1. Inspect the current layout in `SunnyTownInventoryPanel.vue` and `style.css`.
2. Change the `E` menu tray from a small top-right tray into a larger play-area overlay.
3. Let the tray use most or all available game surface space:
   - prefer stable offsets from the game surface edges
   - preserve room for any always-visible close/control affordance
   - keep the overlay scoped to the play area rather than full browser/page chrome
4. Rework desktop columns so crafting and inventory/equipment areas have enough room.
5. Ensure each major region can scroll internally when content exceeds available height.
6. Keep narrow/mobile behavior usable:
   - single-column stacking is fine
   - full-height scrolling is fine
   - avoid text overlap and clipped controls
7. Preserve existing behavior:
   - press `E` opens/closes as before
   - close button works
   - inventory drag/drop works
   - hotbar drops work
   - equipment drops and unequip controls work
   - crafting remains available

## UX Guidance

- This is a practical capacity fix. Do not redesign the entire inventory system.
- A full play-area panel is acceptable for now.
- Avoid adding tabs in this ticket; tabs are a later shell task.
- Avoid adding stats in this ticket; #20 handles stats-ready layout after this.
- Do not make the panel look like a marketing modal. It should feel like an in-game management screen.
- Keep cards shallow and operational. Do not nest cards inside cards.

## Out Of Scope

- Stats-ready character panel implementation.
- Actual stats, skills, XP, or progression math.
- Crafting UI redesign.
- Container/chest transfer work.
- Tabbed `E` menu shell.
- Backend/API changes.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual checks:

- Open Sunny Town.
- Press `E`.
- Confirm the menu uses most or all of the play area and is no longer cramped.
- Confirm the menu closes normally.
- Confirm inventory grid, equipment area, character preview, hotbar editor, selected-item details, and crafting panel all remain visible or reachable through clear scrolling.
- Confirm no major text/control overlap at desktop width.
- Check a narrow/mobile viewport for stacking, scrolling, and clipped controls.

## Completion Notes

When done, close #21 only after the implementation lands on `codex/inventory-redesign-dev` and verification is recorded in the issue close comment.
