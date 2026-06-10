# Sunny Town Container Storage Contract Handoff

## Issue

GitHub issue #26: Design general container storage schema and access contract

https://github.com/walt-raymond-williams/sunny-town-hq/issues/26

## Branch

Work against:

`codex/inventory-redesign-dev`

## Goal

Create the design contract for durable container/chest storage before implementing player-to-container transfer APIs.

The output should answer what identifies a container, who can access it, where durable contents live, how Sunny Town validates live access, and how conflicts are handled.

## Product Rationale

The player-facing chest grid depends on backend storage rules that are safe in multiplayer. This design step prevents a leaky implementation where the browser can mutate arbitrary container IDs or where container identity is tied to unstable map/client state.

## Dependencies and Related Work

Completed blocker:

- #10 Implement slotted player inventory persistence

Related follow-ups:

- #29 Add player to container transfer operations
- #30 Render chest inventory grid
- #25 Add storage context to crafting recipe APIs

## Likely Docs and Code Areas

- `docs/current/ARCHITECTURE.md`
- `docs/current/DATABASE.md`
- `docs/current/API.md`
- `docs/current/COOKIE_SHOP_STORAGE_PLAN.md`
- `sunny-town/maps`
- `internal/sunnytown/server`
- `internal/hq/inventory`
- `internal/hq/sunnytownbridge`

## Scope

In scope:

- Define stable container identity.
- Define ownership/access rules.
- Define Sunny Town distance/access validation responsibility.
- Define HQ durable storage responsibility.
- Define how concurrent edits should behave.
- Define how container storage can later participate in crafting.
- Capture accepted current-state decisions in `docs/current/`.

Out of scope:

- Implementing migrations.
- Implementing transfer APIs.
- Implementing chest UI.
- Implementing storage-aware crafting.

## Acceptance Criteria

- Document whether containers are keyed by fixture ID, placed object ID, room/map/location, shop/storage owner, or another stable identity.
- Document who can open and modify a container.
- Document how Sunny Town validates distance/access before HQ mutation.
- Document how conflicts are handled when multiple players use the same container.
- Document how container storage can be used as a crafting source later.
- Current-state docs are updated with the accepted design.

## Verification

Documentation review against:

- HQ owns durable inventory/container state.
- Sunny Town owns live position and access validation.
- Browser/client messages are requests, not authority.

## Closeout Notes

When closing #26, include:

- design doc path(s)
- key decisions
- rejected alternatives if important
- follow-up notes for #29, #30, and #25
