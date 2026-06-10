# Sunny Town Stack Splitting Handoff

## Purpose

This handoff starts GitHub issue #35, "Add inventory stack splitting."

Issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/35

The goal is to let players split partial quantities from an existing stack into another slot while preserving slotted inventory authority and aggregate totals.

## Current Status

- Issue #11 added slotted inventory move, swap, merge, and auto operations.
- `docs/current/API.md` explicitly says stack splitting is still deferred.
- Container transfer work now exists, so split semantics should stay compatible with future storage descriptors.
- Work should continue on `codex/inventory-redesign-dev`.

## Product Direction

- Stack splitting is an inventory utility, not a full inventory redesign.
- Keep the interaction compact and operational.
- Backend correctness matters most; frontend UI can stay small if included.

## Current Code Surface

Frontend:

- `frontend/src/stores/studentInventory.ts`
  - Inventory move calls and state refresh.
- `frontend/src/features/sunny-town`
  - Inventory slot UI and drag/drop surfaces.

Backend:

- `internal/hq/inventory`
  - Slotted inventory mutation helpers and HTTP handlers.
- `deploy/postgres/migrations`
  - No schema change is expected unless implementation discovers a real gap.

Docs:

- `docs/current/API.md`
  - Update the inventory move contract for `split`.

## Recommended Implementation Plan

1. Inspect the existing inventory move operation enum and request/response shape.
2. Add `split` semantics to the backend mutation layer.
3. Validate source slot, destination slot, ownership, item type, and split quantity.
4. Ensure split is transactional and aggregate totals are preserved.
5. Add focused backend tests for success and invalid split cases.
6. Add a compact frontend interaction if feasible in this slice.
7. Update `docs/current/API.md`.

## Out Of Scope

- Redesigning the whole inventory panel.
- Adding broad container UI changes.
- Changing hotbar or equipment semantics.
- New item stack-size rules unless needed to preserve existing behavior.

## Verification

Run:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/schema
cd frontend
npm run build
```

Manual smoke if UI is added:

- Split a stack into an empty slot.
- Reject zero, negative, too-large, and same-slot split attempts.
- Refresh and verify item totals remain correct.

## Close Criteria

Close #35 when:

- work is committed and pushed to `codex/inventory-redesign-dev`
- verification results are recorded on the issue
- `docs/current/API.md` describes the accepted split contract
