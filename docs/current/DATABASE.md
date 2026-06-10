# Current Database

The app database is PostgreSQL 16. Docker Compose starts it as the `postgres` service on host port `55432`.

Fresh database initialization currently starts from:

```text
deploy/postgres/001_run_migrations.sh
deploy/postgres/migrations/
```

The HQ service also applies these same migrations at startup through:

```text
internal/hq/schema
```

Applied versions are recorded in the `schema_migration` table.

Schema ownership is tracked in:

```text
docs/current/SCHEMA_OWNERSHIP.md
```

## Main Tables

- `app_user`: local app user profile synced from Keycloak subject
- `app_user_role`: app copy of Keycloak `student` and `teacher` roles
- `assignment`: teacher-created questions
- `assignment_attempt`: per-student answers, grading, feedback, resets, and AI review markers
- `assignment_ai_grade`: AI grading request/result records
- `pet_state`: per-student virtual pet state
- `student_wallet`: current per-student star balance
- `student_star_ledger`: idempotent star reward and spend records
- `inventory_item_type`: inventory catalog
- `student_inventory_item`: current per-student item quantities
- `student_inventory_slot`: durable per-student inventory slot layout
- `student_inventory_ledger`: idempotent resource and inventory event records
- `shop_input_storage_item`: durable shop-owned ingredient/input storage quantities
- `student_equipped_item`: current gear/accessory/tool equipment
- `student_hotbar_slot`: current student hotbar slots
- `shop_stock_item`: current HQ-owned shop item quantities
- `shop_stock_ledger`: idempotent shop stock production/adjustment records
- `storage_container`: durable container/chest identity, ownership/access metadata, slot count, and revision
- `storage_container_slot`: durable container/chest item slots
- `student_sunny_town_position`: last accepted Sunny Town map position
- `sunny_town_map_object`: persisted placed map objects
- `sunny_town_character`: shared Sunny Town character identity for player-controlled and future NPC actors
- `sunny_town_npc_character`: durable NPC character mapping by room and NPC key
- `sunny_town_npc_job_production_ledger`: idempotent durable NPC job production events
- `sunny_town_npc_job_production_blocked_ledger`: idempotent durable NPC job production attempts blocked by storage/recipe state

Stats and skills progression is not implemented yet. The accepted design in `docs/current/STATS_SKILLS_PROGRESSION.md` expects future durable state and ledger tables to key progression by `sunny_town_character.id`, not by player-only or NPC-only identities.

## Migration Layout

Current structure:

```text
deploy/postgres/migrations/
  0001_initial.sql
  0002_keycloak_users.sql
  0003_inventory_equipment.sql
  0004_sunny_town_state.sql
  0005_ai_grading.sql
  0006_sunny_town_characters.sql
  0007_sunny_town_npc_characters.sql
  0008_sunny_town_npc_job_production.sql
  0009_shop_stock.sql
  0010_seed_cookie_keeper_shop_stock.sql
  0011_shop_input_storage.sql
  0012_cookie_recipe_inputs.sql
  0013_npc_job_production_blocked.sql
  0014_seed_cookie_keeper_input_storage.sql
  0015_inventory_item_metadata.sql
  0016_student_inventory_slots.sql
  0017_storage_containers.sql
```

Fresh Docker databases apply the ordered SQL files through the Postgres init entrypoint. Existing databases are upgraded by the HQ startup migration runner using the same files.

## Migration Workflow

Add schema changes as new ordered SQL files in:

```text
deploy/postgres/migrations/
```

Use the next numeric prefix and a short domain name, for example:

```text
0006_add_assignment_due_dates.sql
```

Keep migrations idempotent where practical because the ordered files also support legacy local databases. HQ applies unapplied files at startup and records each version in `schema_migration`.

Check applied versions:

```powershell
docker exec hq-postgres psql -U hq -d hq -c "select version, applied_at from schema_migration order by version"
```

Smoke-test migrations against a disposable database:

```powershell
docker run -d --name hq-migration-smoke -e POSTGRES_DB=hq -e POSTGRES_USER=hq -e POSTGRES_PASSWORD=hq -p 55543:5432 postgres:16-alpine
$env:HQ_SCHEMA_TEST_DATABASE_URL="postgres://hq:hq@127.0.0.1:55543/hq?sslmode=disable"
go test ./internal/hq/schema -run TestRunMigrationsIntegration -count=1
docker rm -f hq-migration-smoke
```

## Reset Workflow

Reset only the app database:

```powershell
docker compose -f deploy\docker-compose.yml down
docker volume rm deploy_hq-postgres-data
docker compose -f deploy\docker-compose.yml up -d --build postgres hq
```

Reset app database, Keycloak database, and runtime containers:

```powershell
docker compose -f deploy\docker-compose.yml down
docker volume rm deploy_hq-postgres-data deploy_hq-keycloak-postgres-data
docker compose -f deploy\docker-compose.yml up -d --build
```

The Postgres init entrypoint only runs on an empty database volume. Existing volumes are migrated by HQ startup instead.

## Inventory Catalog Metadata

`inventory_item_type` includes item display and grid-inventory metadata:

- `icon_key`: inventory UI icon or asset key, separate from Sunny Town equipment `visual_key`.
- `max_stack`: maximum stack size for future slotted inventory operations.
- `category`: item grouping for inventory UI and validation, currently `consumable`, `gear`, `tool`, `resource`, or `building`.

Current seeded items use `max_stack = 1` for starter equipment/tools and `max_stack = 64` for stackable consumables, resources, and placed blocks.

## Slotted Student Inventory

`student_inventory_slot` stores the durable player inventory grid for Sunny Town.

Current rules:

- Player inventory has 30 slots.
- Slot indexes are zero-based: `0` through `29`.
- Occupied slots store `item_type_id` and positive `quantity`.
- Empty slots are represented in API responses; empty rows do not need to be stored.
- The same item type may exist in multiple slots.
- `student_inventory_item` remains as an aggregate compatibility table while equipment, hotbar, pet, placement, and Sunny Town quantity flows are migrated safely.
- Student crafting recipe availability and execution use `student_inventory_slot` as the authoritative quantity source.
- Inventory mutation helpers update slot rows and aggregate rows in the same transaction.

## Container Storage

General chest/container storage is implemented according to `docs/current/CONTAINER_STORAGE.md`.

Current schema:

- `storage_container` owns stable container identity, access policy, slot count, revision, and fixture/placed-object/shop metadata.
- `storage_container_slot` owns durable item stacks for each container slot.
- Authored fixture containers should use deterministic IDs such as `fixture:<room_id>:<map_id>:<fixture_id>`.
- Placed object containers should use IDs derived from the persisted placed object ID.
- Cookie Shop `shop_stock_item` and `shop_input_storage_item` remain the current specialized backing tables until a deliberate migration folds them into general container slots.
