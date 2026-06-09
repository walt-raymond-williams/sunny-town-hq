# Sunny Town Character Equipment Rail Handoff

## Issue

GitHub issue #22: Move equipment slots into character preview card

https://github.com/walt-raymond-williams/sunny-town-hq/issues/22

## Branch

Work against the inventory redesign integration branch:

`codex/inventory-redesign-dev`

## Goal

Move the gear, accessory, and tool equipment slots out of the below-preview list and into the character preview card itself.

The preferred layout is a compact vertical rail of equipment slots beside the avatar preview, likely on the left side. The character panel should feel like one integrated character sheet: avatar image, equipment controls, and future stats-ready space should read as one surface instead of separate stacked blocks.

## Product Rationale

The current equipment list below the character image is functional but ugly and space-expensive. The E menu needs to preserve room for the inventory grid, crafting, hotbar, and future stats/skills. Placing the equipment slots inside the preview card makes the UI more like a game inventory screen and reduces wasted vertical space.

## Dependencies and Related Work

- Related: #19 Add character preview shell
- Related: #20 Add stats-ready character panel
- Related: #17 Replace equipment buttons with drop targets

This ticket should usually land after #20, because #20 may still be reshaping the same panel. If #20 is actively in progress, coordinate carefully and do not overwrite unmerged character-preview edits.

## Likely Code Areas

- `frontend/src/features/sunny-town/SunnyTownCharacterPreview.vue`
- `frontend/src/style.css`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue` if the equipment slot props or markup are still owned there
- Existing frontend tests for inventory/equipment drag/drop if layout changes affect selectors or component structure

## Scope

In scope:

- Move gear, accessory, and tool drop targets into the character preview card.
- Arrange those slots as a compact rail beside the avatar preview.
- Preserve equipment drag/drop behavior.
- Preserve unequip behavior.
- Keep empty, occupied, hover, drag-over, invalid-drop, and loading states readable.
- Make the layout responsive without overlapping the avatar, stats-ready region, inventory grid, or hotbar.

Out of scope:

- Adding new equipment slot types.
- Changing equipment compatibility rules.
- Adding stat calculations or manual point allocation.
- Adding new avatar art or equipment visual layers.
- Redesigning the full E menu beyond the character card layout.

## Acceptance Criteria

- Gear, accessory, and tool slots are visibly integrated into the character preview card.
- The old below-preview equipment list is removed or collapsed so it no longer consumes vertical space.
- Compatible equipment can still be dragged from inventory into the matching slot.
- Incompatible drops still fail or show the existing invalid state.
- Unequip remains clear and usable.
- The card remains compact on desktop and usable at narrow/mobile widths.
- No text, controls, avatar art, inventory grid, stats-ready area, or hotbar overlap.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual smoke:

1. Open Sunny Town and press `E`.
2. Confirm the character preview card shows the avatar and the gear/accessory/tool slots together.
3. Confirm the slots are beside the avatar preview, not listed below it.
4. Drag compatible equipment into each slot.
5. Unequip each slot.
6. Check a narrow/mobile viewport for wrapping, overlap, or clipped controls.

## Notes for Implementation Agent

Prefer a small layout pass over a broad component rewrite. The drag/drop behavior is already working; this ticket is about presentation and space efficiency.

If #20 is not closed yet, inspect the current branch and GitHub comments before editing. There may be local or recently pushed changes in the same Vue/CSS files.
