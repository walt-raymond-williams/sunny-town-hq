# Sunny Town Slotted Inventory Schema Handoff

## Purpose

Use this handoff to start a Codex agent on the slotted inventory design task:

GitHub issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/9

Issue title:

`Design slotted player inventory schema and compatibility plan`

Work branch:

`codex/inventory-redesign-dev`

## Task Contract

Treat GitHub issue #9 as the active task contract.

Goal:

Produce a clear design and compatibility plan for adding durable slotted player inventory without breaking existing aggregate inventory behavior used by crafting, hotbar, equipment, mining, placement, pet feeding, and Sunny Town validation.

This is a design task. Do not implement migrations or code changes unless the user explicitly expands the task.

User story:

As a developer, I want a clear slotted inventory schema plan, so that the grid inventory can be implemented without breaking existing crafting, hotbar, mining, placement, and inventory quantity flows.

## Read First

- `AGENTS.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TRACKING.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_DISCOVERY.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_TICKETS.md`
- `docs/INVENTORY_AND_EQUIPMENT.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/ARCHITECTURE.md`
- GitHub issue #9
- GitHub issue #10 and #11 for downstream implementation expectations

## Current State Summary

- HQ owns durable inventory state and database schema.
- Sunny Town owns realtime gameplay validation and calls HQ for durable effects.
- Current player inventory is aggregate quantity state in `student_inventory_item`.
- Current aggregate inventory is keyed by `(app_user_id, item_type_id)`.
- Current hotbar state is persisted in `student_hotbar_slot` and references item type, not stack identity.
- Current equipment state is persisted in `student_equipped_item` and references item type, not stack identity.
- Current crafting consumes and produces through HQ inventory functions.
- Current Sunny Town placement/tool behavior depends on aggregate quantity checks.
- #8 added item display metadata such as icon/stack/category metadata.
- #12 built or is expected to have built reusable slot UI component foundation.

## Files To Audit

Backend inventory and APIs:

- `deploy/postgres/migrations/0003_inventory_equipment.sql`
- Later inventory/shop migrations if present.
- `internal/hq/inventory/inventory.go`
- `internal/hq/inventory/hotbar.go`
- `internal/hq/inventory/equipment.go`
- `internal/hq/inventory/crafting.go`
- `internal/hq/inventory/shop.go`
- `internal/hq/inventory/http.go`
- `internal/hq/inventory/*_test.go`

Sunny Town/HQ bridge dependencies:

- `internal/hq/sunnytownbridge/http.go`
- `internal/hq/sunnytownbridge/store.go`
- `internal/sunnytown/hqclient/client.go`
- `internal/sunnytown/server/client_gameplay.go`

Docs:

- `docs/INVENTORY_AND_EQUIPMENT.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- `docs/current/ARCHITECTURE.md`

## Acceptance Criteria

- Decision is documented for whether `student_inventory_item` remains an aggregate table, becomes a cache/view, or is replaced.
- Proposed slot table shape is documented.
- Migration/backfill strategy is documented.
- Compatibility plan covers current aggregate quantity reads, especially Sunny Town internal inventory quantity checks.
- Initial player inventory slot count is selected or explicitly deferred.
- Stack splitting is either included or explicitly deferred from the first movement API.

## Design Questions To Answer

1. Should `student_inventory_item` remain the source of aggregate truth?
2. Should a new `student_inventory_slot` table become the source of truth, with aggregate quantities derived from slots?
3. If both aggregate and slot tables exist, how do mutations keep them consistent?
4. What is the initial player inventory slot count?
5. What is the slot index range and constraint shape?
6. Should slots support quantity per stack plus item type ID?
7. Should unique constraints prevent duplicate item rows per slot but allow the same item type in multiple stacks?
8. How should existing aggregate inventory be backfilled into slots?
9. How should zero-quantity rows and equipped/hotbar items be handled during backfill?
10. Should hotbar assignments continue to reference item type, or should future work move them to inventory slot/stack identity?
11. Should equipment continue to reference item type, or should future work move it to inventory slot/stack identity?
12. How do `ConsumeStudentItem` and `IncrementStudentItem` behave after slots exist?
13. How does `LoadStudentInventoryQuantity` continue to work for Sunny Town?
14. Should stack splitting be part of the first movement API, or deferred?
15. What API response shape should #10 and #11 implement for slotted inventory?

## Recommended Output

Create or update a design document under `docs/`, likely:

`docs/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_PLAN.md`

The design doc should include:

- Recommended schema.
- Rationale and rejected alternatives.
- Migration/backfill plan.
- Compatibility plan for existing APIs and gameplay paths.
- Mutation consistency strategy.
- API response shape recommendation.
- Open questions, if any.
- Downstream implementation notes for #10 and #11.

If the design is accepted and becomes current-state architecture, summarize the accepted decisions in:

- `docs/current/DATABASE.md`
- `docs/current/API.md`
- `docs/INVENTORY_AND_EQUIPMENT.md`

Do not over-update current-state docs before the user accepts the design.

## Suggested Design Bias

Prefer a conservative design that minimizes breakage:

- Keep aggregate quantity behavior available for existing systems.
- Introduce slot persistence in a way that can be migrated gradually.
- Preserve current hotbar/equipment behavior unless changing it is clearly necessary.
- Keep HQ as the durable inventory authority.
- Keep Sunny Town validation dependent on server-accepted player state and HQ-owned durable state.

The cleanest likely option may be:

- Add a slotted inventory table for display/organization.
- Continue to support aggregate quantity reads either by deriving from slots or maintaining aggregate rows transactionally.
- Delay hotbar/equipment stack-identity changes until direct manipulation semantics require them.
- Defer stack splitting unless needed for first usable drag/drop.

This is a starting hypothesis, not a requirement. The agent should verify it against the code.

## Out Of Scope

- No migration implementation.
- No backend mutation implementation.
- No frontend grid rendering.
- No stack movement API implementation.
- No drag/drop implementation.
- No chest/container schema, except notes about future compatibility.
- No storage-aware crafting behavior.

## Verification

This is a design task. Verification is review-based:

```powershell
git diff -- docs
```

If the agent touches code unexpectedly, run the relevant tests:

```powershell
go test ./internal/hq/inventory
go test ./internal/hq/schema
go test ./...
cd frontend
npm run build
```

## GitHub Workflow

Before implementation/design work:

1. Confirm issue #9 has label `status:in-progress`.
2. Work on branch `codex/inventory-redesign-dev`.

During work:

- Work from issue #9.
- Keep GitHub comments for durable design decisions, blockers, and discoveries.
- If implementation work is needed, leave it for #10 or #11 rather than expanding #9.

Closeout:

- Close #9 only after the design doc lands on `codex/inventory-redesign-dev`.
- Closing comment should include:
  - design doc path
  - key decision summary
  - any remaining open questions
  - verification/review performed
- After closing #9, update #10 from `status:blocked` to `status:ready` if the design fully unblocks implementation.

## Suggested First Agent Prompt

```text
You are working in the Sunny Town HQ repo on branch codex/inventory-redesign-dev.

Start with GitHub issue #9:
https://github.com/walt-raymond-williams/sunny-town-hq/issues/9

Read AGENTS.md and docs/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_HANDOFF.md first. Treat issue #9 as the active task contract.

Goal: produce a slotted player inventory schema and compatibility plan. This is a design task, not an implementation task. Do not add migrations or code unless explicitly asked.

Audit current inventory schema, inventory mutation functions, crafting, hotbar, equipment, Sunny Town internal inventory quantity checks, and current docs. Then create docs/SUNNY_TOWN_SLOTTED_INVENTORY_SCHEMA_PLAN.md with the recommended schema, migration/backfill plan, compatibility plan, API response recommendations, rejected alternatives, and downstream notes for issues #10 and #11.

Keep HQ as the durable inventory authority. Preserve existing aggregate quantity behavior for current gameplay flows unless you explicitly justify a safer replacement.
```
