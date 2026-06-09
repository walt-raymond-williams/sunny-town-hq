# Sunny Town Stats-Ready Character Panel Handoff

## Purpose

This handoff starts GitHub issue #20, "Add stats-ready character panel."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/20

The goal is to evolve the inventory menu's new character preview area so it has a compact, stable place for future stats and skills without implementing stat progression yet.

## Current Status

The inventory redesign is on `codex/inventory-redesign-dev`.

Completed foundation:

- #15 inventory grid slot-to-slot drag/drop is complete.
- #16 inventory-to-hotbar drop targets are complete.
- #17 inventory-to-equipment drop targets are complete.
- #18 legacy selected-item assignment buttons are removed.
- #19 character preview shell is complete and verified in commit `20bac63`.

The menu now has an avatar/equipment preview surface. This ticket should extend that surface so future character stats can fit into the layout cleanly.

## Product Direction

The character panel should feel like the beginning of a game-style character sheet:

- avatar preview remains visible near equipment
- equipment management remains the primary interaction
- a compact stats-ready area exists for future progression data
- no manual point allocation UI appears
- no fake stat values or invented mechanics appear

The player direction is activity-driven progression. Characters improve by doing work, crafting, gathering, or using skills. Do not add a "spend points" model.

## Current Code Surface

Frontend:

- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
  - owns the inventory overlay layout
  - renders the character preview, equipment panel, inventory grid, selected item details, hotbar editor, and crafting panel
- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
  - likely owns the preview shell added for #19
  - should be the first place to inspect for the stats-ready extension
- `frontend/src/stores/studentInventory.ts`
  - provides equipment/visual state used by the inventory menu
- `frontend/src/style.css`
  - may contain supporting inventory/preview layout styles

## Recommended Implementation Plan

1. Inspect the #19 implementation before editing:
   - `SunnyTownInventoryPanel.vue`
   - `SunnyTownCharacterPreview.vue`
   - related CSS
2. Add a stats-ready layout region near the character preview/equipment area.
3. Keep the stats region visually reserved but mechanically honest:
   - neutral labels are acceptable
   - empty rows, disabled-looking stat chips, or a small quiet placeholder are acceptable
   - avoid copy that promises a specific progression system
4. Preserve all existing inventory controls:
   - equipment drop targets
   - unequip controls
   - inventory grid drag/drop
   - hotbar drop targets
   - crafting panel
   - selected item detail display
5. Check the layout at narrow and desktop widths.
6. Add focused tests only if the repository has an obvious existing pattern for the touched component.

## UX Guidance

- Keep this compact. It should feel like an equipment/character panel inside an operational game UI.
- Do not make the stats area a large explanatory card.
- Do not place cards inside cards.
- Keep text short enough to fit on mobile.
- Do not add controls that look interactive unless they do something.
- Prefer labels that stay compatible with activity-driven progression, such as `Skills`, `Traits`, or `Stats`, over point-allocation language.

## Out Of Scope

- Implementing stats, skill values, XP, or level-up math.
- Manual stat point allocation.
- Backend schema for skills or stats.
- Tabbed `E` menu shell.
- Crafting redesign.
- Container/chest inventory work.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual checks:

- Open Sunny Town inventory menu.
- Confirm character preview still renders.
- Confirm the stats-ready area is visible without implying manual point allocation.
- Confirm equipment drop targets and unequip controls remain usable.
- Confirm inventory grid, hotbar drop targets, and crafting panel still fit at desktop width.
- Check a narrow/mobile viewport for text wrapping and overlap.

## Completion Notes

When done, close #20 only after the implementation lands on `codex/inventory-redesign-dev` and verification is recorded in the issue close comment.
