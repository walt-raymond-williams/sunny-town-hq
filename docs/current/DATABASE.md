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
- `student_inventory_ledger`: idempotent resource and inventory event records
- `shop_input_storage_item`: durable shop-owned ingredient/input storage quantities
- `student_equipped_item`: current gear/accessory/tool equipment
- `student_hotbar_slot`: current student hotbar slots
- `shop_stock_item`: current HQ-owned shop item quantities
- `shop_stock_ledger`: idempotent shop stock production/adjustment records
- `student_sunny_town_position`: last accepted Sunny Town map position
- `sunny_town_map_object`: persisted placed map objects
- `sunny_town_character`: shared Sunny Town character identity for player-controlled and future NPC actors
- `sunny_town_npc_character`: durable NPC character mapping by room and NPC key
- `sunny_town_npc_job_production_ledger`: idempotent durable NPC job production events
- `sunny_town_npc_job_production_blocked_ledger`: idempotent durable NPC job production attempts blocked by storage/recipe state

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
