# Sunny Town Tabbed E Menu Handoff

## Issue

GitHub issue #27: Add tabbed E menu shell

https://github.com/walt-raymond-williams/sunny-town-hq/issues/27

## Branch

Work against:

`codex/inventory-redesign-dev`

## Goal

Add a tab-capable shell to the Sunny Town `E` menu while keeping the current inventory/crafting experience as the default useful screen.

The first slice should prepare the menu for future inventory, crafting, stats, skills, and container views without turning the menu into a placeholder page.

## Product Rationale

The E menu now contains inventory grid, crafting, hotbar, equipment, character preview, and stats-ready space. Tabs give the menu a growth path before future stats/skills and storage views overload a single surface.

## Dependencies and Related Work

Completed blocker:

- #13 Render Sunny Town inventory as a grid

Related:

- #19 Add character preview shell
- #28 Define stats and skills progression model
- #24 Redesign crafting panel for inventory menu

## Likely Code Areas

- `frontend/src/features/sunny-town/SunnyTownPage.vue`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue`
- `frontend/src/composables/useSunnyTownInventoryActions.ts`
- `frontend/src/style.css`
- frontend tests around Sunny Town inventory/menu behavior

## Scope

In scope:

- Add a compact tab shell for the E menu.
- Keep inventory/crafting as the default active tab.
- Preserve close/Escape behavior.
- Preserve gameplay input suppression while the menu is open.
- Add only useful or clearly future-facing tabs; avoid fake feature content.
- Keep desktop and narrow/mobile layouts usable.

Out of scope:

- Implementing stats/skills behavior.
- Implementing chest transfer UI.
- Rewriting the whole E menu layout.
- Adding a landing page or explanatory screen.

## Acceptance Criteria

- Pressing `E` opens a tab-capable menu shell.
- Inventory/crafting remains the default useful first tab.
- Existing close and Escape behavior still works.
- Gameplay inputs do not conflict while the menu is open.
- Placeholder tabs are absent or minimal and do not pretend unfinished features exist.
- Layout remains usable at desktop and narrow/mobile widths.

## Verification

Run:

```powershell
cd frontend
npm run build
```

Manual smoke:

1. Open Sunny Town.
2. Press `E`.
3. Confirm inventory/crafting is immediately usable.
4. Switch tabs if more than one tab is present.
5. Press Escape and confirm the menu closes.
6. Confirm movement/tool inputs are suppressed while the menu is open.

## Closeout Notes

When closing #27, include:

- implementation commit hash
- tabs added
- verification commands and results
- manual responsive/input smoke result if performed
