# Sunny Town Crafting Slotted Inventory Bug Handoff

## Issue

GitHub issue #23: Fix stone block crafting failure after slotted inventory migration

https://github.com/walt-raymond-williams/sunny-town-hq/issues/23

## Branch

Work against the inventory redesign integration branch:

`codex/inventory-redesign-dev`

## Goal

Fix the bug where a player who appears to have enough rocks to craft a `stone_block` can receive:

`crafting could not be completed`

The result should be boring in the best way: if the player has at least 4 rocks and output space, crafting succeeds, the grid updates, and the player sees the new stone block.

## Product Rationale

Crafting is part of the core Sunny Town inventory loop. Players mine rocks, craft stone blocks, place them, and later recover them. If the recipe panel says the player can craft but the action fails with a generic error, the redesigned inventory becomes hard to trust.

This should be fixed before the larger crafting panel redesign, because the UI redesign should not build on ambiguous backend behavior.

## Current Investigation Notes

- The visible error comes from HQ's crafting endpoint, not just frontend validation.
- `internal/hq/inventory/http.go` calls `CraftStudentRecipe` and maps unexpected backend errors through `CraftingErrorMessage`.
- `internal/hq/inventory/crafting.go` loads recipe ingredient ownership from `student_inventory_item` aggregate rows.
- The craft mutation now consumes and produces through the slotted inventory mutation layer in `internal/hq/inventory/inventory.go`.
- `ConsumeStudentItem` checks `student_inventory_slot` availability before updating the aggregate row.
- If aggregate and slot state diverge, the recipe can look craftable while the mutation sees insufficient slotted quantity.
- If output creation hits `ErrInventoryFull`, the player currently receives the same generic crafting failure message.
- A spot check of the local Docker DB after the report showed visible rock and stone block aggregate/slot rows in sync, so do not assume one persistent bad data state without proving it.

## Dependencies and Related Work

- Related: #10 Implement slotted player inventory persistence
- Related: #13 Render Sunny Town inventory as a grid
- Related: #15 Add inventory drag/drop state and API integration
- Related: future crafting panel redesign

This bug is not blocked by #20 or #22. It can be worked independently because it mostly touches HQ inventory/crafting behavior and the frontend inventory store refresh path.

## Likely Code Areas

- `internal/hq/inventory/crafting.go`
- `internal/hq/inventory/inventory.go`
- `internal/hq/inventory/http.go`
- `internal/hq/inventory/store_integration_test.go`
- `frontend/src/api/craftingApi.ts`
- `frontend/src/stores/studentInventory.ts`
- `frontend/src/features/sunny-town/SunnyTownInventoryPanel.vue` if UI error handling needs a small adjustment

## Scope

In scope:

- Make recipe craftability and craft execution use consistent authoritative inventory state.
- Make successful crafting refresh the frontend grid from authoritative slotted inventory state, or reload slots immediately after craft.
- Preserve existing `not enough ingredients` behavior for true missing ingredients.
- Return a clearer player-facing error when output cannot be produced because inventory is full.
- Add regression coverage around stone block crafting after slotted inventory persistence.

Out of scope:

- Redesigning the crafting panel UI.
- Adding container-aware crafting.
- Adding new recipes.
- Changing the stone block recipe cost.
- Reworking inventory capacity rules beyond what is needed to fix this bug.

## Acceptance Criteria

- Crafting `stone_block` succeeds when the player has at least 4 rocks in authoritative inventory storage and output space.
- Rocks decrease by 4 and stone blocks increase by 1 after a successful craft.
- The inventory grid reflects the updated authoritative slot state after craft.
- Recipe availability updates after craft.
- True missing ingredients still return/display `not enough ingredients`.
- Full inventory output failure returns a clear error instead of generic `crafting could not be completed`.
- Regression tests cover successful slotted stone block crafting.
- Regression tests cover the aggregate/slot mismatch case, or the implementation documents and tests why mismatch is impossible.
- Regression tests cover inventory-full output handling if that path remains possible.

## Suggested Implementation Approach

1. Reproduce or isolate the backend failure with a focused test before editing behavior.
2. Decide which state is authoritative for crafting recipe availability during the slotted-inventory transition.
3. Prefer one durable boundary over scattered special cases:
   - either load crafting ingredient ownership from slots, or
   - guarantee aggregate and slot rows cannot diverge before recipe availability is computed.
4. Make the craft response/store update use authoritative slot data. The current frontend reconstructs slots from aggregate inventory after craft, which can undo user-arranged grid layout.
5. Map `ErrInventoryFull` to a clear crafting error message if output production can still fail that way.
6. Keep the fix narrow. This is a correctness ticket, not the crafting redesign.

## Verification

Run:

```powershell
go test ./internal/hq/inventory
```

If frontend store/API code changes, also run:

```powershell
cd frontend
npm run build
```

Manual smoke:

1. Start the local runtime.
2. Log in as a student.
3. Ensure the student has at least 4 rocks and output space.
4. Open Sunny Town and press `E`.
5. Craft `stone_block`.
6. Confirm rocks decrease by 4.
7. Confirm stone block quantity increases by 1.
8. Confirm the inventory grid updates without losing the intended slotted layout.
9. Confirm no generic crafting error appears.
10. If practical, fill inventory output space and confirm the full-inventory message is clear.

## Closeout Notes

When closing #23, include:

- implementation commit hash
- whether crafting availability now reads slot state or aggregate state
- whether the frontend reloads authoritative slots after craft
- verification commands and results
- manual smoke result if performed
