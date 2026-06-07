# Current Database

The app database is PostgreSQL 16. Docker Compose starts it as the `postgres` service on host port `55432`.

Fresh database initialization currently starts from:

```text
deploy/postgres/init/001_create_assignment.sql
```

The HQ service also runs schema-safety upgrades at startup in:

```text
cmd/hq/schema.go
```

This mixed model is acceptable for early development but should be replaced by explicit ordered migrations after active AI schema work settles.

Schema ownership for the migration conversion is tracked in:

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
- `student_equipped_item`: current gear/accessory/tool equipment
- `student_hotbar_slot`: current student hotbar slots
- `student_sunny_town_position`: last accepted Sunny Town map position
- `sunny_town_map_object`: persisted placed map objects

## Migration Direction

Target structure:

```text
deploy/postgres/migrations/
  0001_initial.sql
  0002_keycloak_users.sql
  0003_inventory_equipment.sql
  0004_sunny_town_state.sql
  0005_ai_grading.sql
```

Do not start this migration conversion until the current HQ package-boundary cleanup is finished. Use `docs/current/SCHEMA_OWNERSHIP.md` as the table and migration ownership map.
