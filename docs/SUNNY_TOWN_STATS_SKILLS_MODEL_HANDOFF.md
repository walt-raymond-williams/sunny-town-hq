# Sunny Town Stats And Skills Model Handoff

## Issue

GitHub issue #28: Define stats and skills progression model

https://github.com/walt-raymond-williams/sunny-town-hq/issues/28

## Branch

Work against:

`codex/inventory-redesign-dev`

## Goal

Define the first durable design model for player and NPC stats/skills progression.

The direction is activity-driven progression: characters improve by doing work, crafting, gathering, using tools, and performing jobs. Do not introduce manual point allocation unless the product direction explicitly changes.

## Product Rationale

The character panel is stats-ready, but the game still needs a real progression model. This design should give future agents enough structure to add skill requirements to recipes, tools, jobs, and actions without inventing a one-off system each time.

## Dependencies and Related Work

No blockers.

Related:

- #20 Add stats-ready character panel
- #32 Add ingredient-aware Cookie Shop production
- future mining/crafting/cooking/job skill requirements

## Likely Docs

- `docs/current/ARCHITECTURE.md`
- `docs/current/API.md`
- `docs/current/DATABASE.md`
- a new focused `docs/current/...` design doc if the model is substantial
- `docs/SUNNY_TOWN_INVENTORY_REDESIGN_ROADMAP.md`

## Scope

In scope:

- Define shared player/NPC progression primitives.
- Define example activities and the skills/stats they improve.
- Define how actions, tools, jobs, and recipes can require stats/skills later.
- Explicitly reject manual point distribution for now.
- Identify the first vertical implementation slice.
- Capture decisions in current-state docs.

Out of scope:

- Implementing database schema.
- Implementing UI stats.
- Implementing skill XP changes.
- Adding fake placeholder stat numbers.
- Balancing every future skill.

## Acceptance Criteria

- Shared player/NPC progression primitives are documented.
- Activity-to-skill examples are documented.
- Recipe/tool/job/action requirement examples are documented.
- Manual point distribution is explicitly rejected unless direction changes.
- The first progression vertical slice is identified.
- Current-state docs contain the accepted model or link to the accepted model.

## Verification

Documentation review. Check that the model is usable by:

- a future backend schema ticket
- a future UI stats panel ticket
- a future recipe/job skill requirement ticket

## Closeout Notes

When closing #28, include:

- design doc path(s)
- key primitives chosen
- first recommended implementation slice
- open questions left for later
