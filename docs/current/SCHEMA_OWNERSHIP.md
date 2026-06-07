# HQ Schema Ownership

This map captures the current HQ schema ownership boundaries for ordered migrations.

## Ownership Rules

- HQ owns all durable identity, assignment, grading, inventory, wallet, pet, and Sunny Town persistence tables.
- Sunny Town owns live realtime world state and must not write the HQ database directly.
- AI grading writes HQ assignment grading state only through HQ-owned internal endpoints.
- Pet remains an HQ domain package. AI may consume pet context later, but pet tables stay under HQ ownership.

## Domain Groups

### Identity And Auth

Owner: `internal/hq/auth` for Keycloak token verification and auth user types, and `internal/hq/users` for authenticated user persistence, role sync, default student provisioning, and student listing.

Tables and indexes:

- `app_user`
- `app_user_role`
- `app_user_keycloak_subject_key`

Migration target: `0002_keycloak_users.sql`

### Assignments

Owner: `internal/hq/assignments`

Tables, columns, constraints, and indexes:

- `assignment`
- `assignment_attempt`
- `assignment_attempt.student_user_id`
- `assignment_attempt.graded_by_type`
- `assignment_attempt.graded_by_user_id`
- `assignment_attempt.graded_by_service`
- `assignment_attempt.grade_source`
- `assignment_attempt.ai_review_status`
- `assignment_attempt.ai_grade_id`
- `assignment_attempt_number_unique`
- `assignment_attempt_student_user_id_idx`
- `assignment_attempt_active_review_idx`
- `assignment_attempt_student_user_id_fkey`
- `assignment_attempt_graded_by_user_id_fkey`
- `assignment_attempt_ai_grade_id_fkey`
- assignment grading check constraints

Migration target: `0001_initial.sql` for base assignment tables, then `0002_keycloak_users.sql` for student ownership, and `0005_ai_grading.sql` for AI grading columns.

### AI Grading

Owner: `internal/hq/ai`, coordinated with `internal/hq/assignments`

Tables, constraints, and indexes:

- `assignment_ai_grade`
- `assignment_ai_grade_status_check`
- `assignment_ai_grade_attempt_idx`

Migration target: `0005_ai_grading.sql`

### Inventory And Economy

Owner: `internal/hq/inventory`

Tables, constraints, and indexes:

- `student_wallet`
- `student_star_ledger`
- `student_star_ledger_app_user_id_idx`
- `inventory_item_type`
- `inventory_item_type_key_check`
- `inventory_item_type_equip_slot_check`
- `student_inventory_item`
- `student_inventory_ledger`
- `student_inventory_ledger_app_user_id_idx`
- `student_equipped_item`
- `student_equipped_item_slot_check`
- `student_equipped_item_app_user_id_idx`
- `student_hotbar_slot`
- `student_hotbar_slot_index_check`
- `student_hotbar_slot_app_user_id_idx`

Migration target: `0003_inventory_equipment.sql`

### Pet

Owner: `internal/hq/pet`

Tables:

- `pet_state`

Migration target: keep in `0001_initial.sql` unless pet schema expands enough to justify a dedicated migration file.

### Sunny Town Persistence

Owner: `internal/hq/sunnytownbridge`

Tables, constraints, and indexes:

- `student_sunny_town_position`
- `student_sunny_town_position_facing_check`
- `student_sunny_town_position_updated_at_idx`
- `sunny_town_map_object`
- `sunny_town_map_object_grid_nonnegative`
- `sunny_town_map_object_item_key_check`
- `sunny_town_map_object_location_key`
- `sunny_town_map_object_map_idx`

Migration target: `0004_sunny_town_state.sql`

## Migration Notes

- Preserve statement order when adding future schema migrations.
- Keep data backfills near the schema change they support, for example cookie migration into inventory and default Sunny Town starter items.
- Keep compatibility constraints explicit; do not rely on application validation alone.
- Runtime schema patching has been replaced by the HQ startup migration runner in `internal/hq/schema`.
