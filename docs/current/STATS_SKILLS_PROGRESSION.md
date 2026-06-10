# Sunny Town Stats And Skills Progression

This is the accepted first design model for Sunny Town character progression. It is intentionally a design contract, not an implemented schema.

## Direction

Sunny Town progression is activity-driven. Characters improve by doing town activities: gathering, crafting, cooking, using tools, completing jobs, tending production, trading, and future care/social actions.

Manual point allocation is rejected for now. Do not add spendable stat points, skill points, perk points, or level-up allocation UI unless the product direction explicitly changes.

Progression attaches to shared Sunny Town character identity:

- Player-controlled characters use `sunny_town_character` rows linked to an app user.
- NPC characters use `sunny_town_character` rows linked through `sunny_town_npc_character`.
- Student academic progress, assignment grades, and schoolwork mastery stay outside this model.
- NPC drive/controller state stays outside this model unless it becomes durable gameplay state.

## Primitives

Use a small set of stable primitives so future implementation tickets can add schema and APIs without redesigning the model.

### Stats

Stats are broad character capabilities. They change slowly and can affect multiple skills or action checks.

Initial stat vocabulary:

- `vitality`: endurance, health buffer, recovery, and physically demanding work.
- `focus`: crafting care, learning speed, recipe reliability, and precise tool use.
- `coordination`: tool handling, movement-sensitive actions, gathering speed, and future combat/avoidance.
- `social`: trading, cooperation, teaching, and town relationship actions.

Stats should be represented as durable values keyed by `character_id` and `stat_key`. The first implementation can store integer levels and optional XP/progress; exact formulas are deferred.

### Skills

Skills are activity-specific proficiency tracks. They improve directly from successful or meaningful attempts.

Initial skill vocabulary:

- `mining`: breaking rocks, harvesting stone/crystal, using pickaxes.
- `crafting`: player crafting, assembling building/material recipes.
- `cooking`: Cookie Shop ingredient work, food recipes, kitchen production.
- `shopkeeping`: stocking, selling, fulfilling shop/job production.
- `building`: placing and maintaining world objects.
- `learning`: optional bridge for future school-to-town bonuses, without replacing assignment grades.

Skills should be durable values keyed by `character_id` and `skill_key`. XP should be awarded by server-authoritative gameplay events, not by client-reported counters.

### XP Events

Progress should be recorded from validated actions. A future ledger should include:

- `event_id` for idempotency
- `character_id`
- `source` such as `mining`, `crafting`, `npc_job`, `shop_production`
- `activity_key`
- awarded `skill_key` and XP amount
- optional stat XP, room/map, item/recipe/job keys, and timestamp

The ledger is the audit/debug source; current skill/stat totals are the fast-read projection.

## Activity Examples

Mining:

- Validated pickaxe hit on a rock node can grant small `mining` XP.
- Successful harvest can grant larger `mining` XP and possibly `coordination` XP.
- Tool requirements can check `mining` or `coordination` later.

Crafting:

- Successful `stone_block` crafting can grant `crafting` XP.
- Higher-tier recipes can require `crafting >= 2` or `focus >= 2`.
- Failed validation should not grant XP; future partial/workstation attempts may grant capped attempt XP only if server-authoritative.

Cooking and Cookie Shop production:

- Cookie Keeper production can grant NPC `cooking` or `shopkeeping` XP when ingredients are consumed and output is produced.
- Player ingredient deposit should not by itself grant cooking XP unless the action becomes meaningful gameplay.
- Future player cooking recipes can require `cooking` and consume from explicit recipe storage contexts.

Building:

- Successful stone block placement can grant `building` XP.
- Placement removal/mining can remain `mining` unless future design splits demolition.

Shopkeeping:

- NPC job production, restocking, and future sales can grant `shopkeeping` XP.
- Player trade interactions should award only when the player performs a meaningful shopkeeping action, not passive buying.

## Requirements Model

Recipes, tools, jobs, and actions should be able to declare requirements without hard-coding checks in UI components.

Recommended requirement descriptor:

```json
{
  "all": [
    { "kind": "skill", "key": "crafting", "minLevel": 2 },
    { "kind": "stat", "key": "focus", "minLevel": 1 },
    { "kind": "item", "key": "pickaxe", "equipped": true }
  ]
}
```

Initial requirement consumers:

- Recipes: require skill/stat levels before crafting or production.
- Tools: require skill/stat levels before use or before harvesting specific nodes.
- Jobs: require NPC skill/stat levels before assignment or production.
- Actions: require skills/stats for placement, repair, cooking, farming, or social actions.

HQ should evaluate durable requirements when the action mutates HQ-owned state. Sunny Town should evaluate live requirements when the action depends on realtime state such as position, equipped tool, map object, or NPC job state. When both are involved, Sunny Town validates live state first, then calls a service-authenticated HQ endpoint that validates durable stats/skills in the same transaction as the mutation.

## First Vertical Slice

Recommended first implementation: mining XP for player characters.

Why this slice:

- Mining already has server-authoritative validation in Sunny Town.
- The durable actor identity is already available as `character_id`.
- The action already commits resource events to HQ with idempotent event IDs.
- It gives the character panel real player-facing progress without touching recipe balance first.

Suggested scope:

1. Add durable skill definition/state/ledger tables for `mining`.
2. Add a service-authenticated HQ endpoint to award idempotent character skill XP.
3. Award `mining` XP from successful Sunny Town resource harvest events.
4. Add a student-facing read API for the current character's stats/skills.
5. Render real `mining` progress in the character panel.

Do not add manual allocation, global character levels, broad balancing formulas, or NPC skill automation in the first slice.

## Open Questions

- Exact level curve and XP amounts.
- Whether stats gain XP directly or are derived from skill activity.
- Whether NPC jobs use the same level curve as players.
- Whether `learning` should exist as a town skill or remain academic-only.
- How equipment modifiers stack with base stats/skills.
- Whether relationship/social progression belongs in this model or a separate social graph.
