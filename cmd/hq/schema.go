package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (app *app) ensureSchema(ctx context.Context) error {
	statements := []string{
		`alter table app_user add column if not exists keycloak_subject text`,
		`alter table app_user add column if not exists email text`,
		`update app_user set keycloak_subject = 'legacy-user-' || id where keycloak_subject is null`,
		`alter table app_user alter column keycloak_subject set not null`,
		`create unique index if not exists app_user_keycloak_subject_key on app_user (keycloak_subject)`,
		`create table if not exists app_user_role (
			user_id bigint not null references app_user(id) on delete cascade,
			role text not null,
			primary key (user_id, role),
			constraint app_user_role_role_check check (role in ('student', 'teacher'))
		)`,
		`insert into app_user_role (user_id, role)
			select id, 'student' from app_user
			on conflict do nothing`,
		`alter table assignment_attempt add column if not exists student_user_id bigint`,
		`update assignment_attempt set student_user_id = 1 where student_user_id is null`,
		`alter table assignment_attempt alter column student_user_id set not null`,
		`do $$
		begin
			if not exists (
				select 1 from pg_constraint where conname = 'assignment_attempt_student_user_id_fkey'
			) then
				alter table assignment_attempt
					add constraint assignment_attempt_student_user_id_fkey
					foreign key (student_user_id) references app_user(id) on delete cascade;
			end if;
		end $$`,
		`alter table assignment_attempt drop constraint if exists assignment_attempt_number_unique`,
		`alter table assignment_attempt
			add constraint assignment_attempt_number_unique unique (assignment_id, student_user_id, attempt_number)`,
		`create index if not exists assignment_attempt_student_user_id_idx on assignment_attempt (student_user_id)`,
		`drop index if exists assignment_attempt_active_review_idx`,
		`create index if not exists assignment_attempt_active_review_idx
			on assignment_attempt (assignment_id, student_user_id, date_submitted desc)
			where reset_at is null`,
		`create table if not exists student_wallet (
			app_user_id bigint primary key references app_user(id) on delete cascade,
			star_balance integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint student_wallet_star_balance_nonnegative check (star_balance >= 0)
		)`,
		`create table if not exists student_star_ledger (
			id bigserial primary key,
			app_user_id bigint not null references app_user(id) on delete cascade,
			event_id text not null unique,
			source text not null,
			delta integer not null,
			room_id text null,
			map_id text null,
			collectible_id text null,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now(),
			constraint student_star_ledger_delta_nonzero check (delta <> 0)
		)`,
		`create index if not exists student_star_ledger_app_user_id_idx
			on student_star_ledger (app_user_id, created_at desc)`,
		`create table if not exists inventory_item_type (
			id bigserial primary key,
			key text not null unique,
			name text not null,
			description text not null default '',
			equip_slot text null,
			visual_key text null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			constraint inventory_item_type_key_check check (key ~ '^[a-z][a-z0-9_]*$'),
			constraint inventory_item_type_equip_slot_check check (equip_slot is null or equip_slot in ('gear', 'accessory'))
		)`,
		`alter table inventory_item_type add column if not exists equip_slot text`,
		`alter table inventory_item_type add column if not exists visual_key text`,
		`do $$
		begin
			if not exists (
				select 1 from pg_constraint where conname = 'inventory_item_type_equip_slot_check'
			) then
				alter table inventory_item_type
					add constraint inventory_item_type_equip_slot_check
					check (equip_slot is null or equip_slot in ('gear', 'accessory'));
			end if;
		end $$`,
		`insert into inventory_item_type (key, name, description)
			values ('cookie', 'Cookie', 'A treat for your pet.')
			on conflict (key) do update
			set name = excluded.name,
				description = excluded.description,
				equip_slot = null,
				visual_key = null,
				updated_at = now()`,
		`insert into inventory_item_type (key, name, description, equip_slot, visual_key)
			values
				('sunny_hoodie', 'Sunny Hoodie', 'A cozy hoodie for Sunny Town.', 'gear', 'sunny_hoodie'),
				('star_cap', 'Star Cap', 'A bright cap for sunny adventures.', 'accessory', 'star_cap')
			on conflict (key) do update
			set name = excluded.name,
				description = excluded.description,
				equip_slot = excluded.equip_slot,
				visual_key = excluded.visual_key,
				updated_at = now()`,
		`create table if not exists student_inventory_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, item_type_id),
			constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
		)`,
		`insert into student_inventory_item (app_user_id, item_type_id, quantity)
			select u.id, iit.id, u.cookies
			from app_user u
			cross join inventory_item_type iit
			where iit.key = 'cookie'
				and u.cookies > 0
			on conflict (app_user_id, item_type_id) do update
			set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
				updated_at = now()`,
		`update app_user set cookies = 0 where cookies > 0`,
		`insert into student_inventory_item (app_user_id, item_type_id, quantity)
			select u.id, iit.id, 1
			from app_user u
			join app_user_role ur on ur.user_id = u.id and ur.role = 'student'
			cross join inventory_item_type iit
			where iit.key in ('sunny_hoodie', 'star_cap')
			on conflict (app_user_id, item_type_id) do update
			set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
				updated_at = now()`,
		`create table if not exists student_equipped_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			slot text not null,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, slot),
			constraint student_equipped_item_slot_check check (slot in ('gear', 'accessory'))
		)`,
		`create index if not exists student_equipped_item_app_user_id_idx
			on student_equipped_item (app_user_id)`,
	}

	for _, statement := range statements {
		if _, err := app.db.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) syncAuthenticatedUser(ctx context.Context, user authUser) (authUser, error) {
	if user.KeycloakSubject == "" {
		return authUser{}, errInvalidToken
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return authUser{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = user.KeycloakSubject
	}

	err = tx.QueryRow(
		ctx,
		`
			insert into app_user (keycloak_subject, display_name, email)
			values ($1, $2, nullif($3, ''))
			on conflict (keycloak_subject) do update
			set display_name = excluded.display_name,
				email = excluded.email
			returning id, display_name, coalesce(email, '')
		`,
		user.KeycloakSubject,
		displayName,
		strings.TrimSpace(user.Email),
	).Scan(&user.ID, &user.DisplayName, &user.Email)
	if err != nil {
		return authUser{}, err
	}

	if _, err := tx.Exec(ctx, "delete from app_user_role where user_id = $1", user.ID); err != nil {
		return authUser{}, err
	}

	for _, role := range user.Roles {
		if role != "student" && role != "teacher" {
			continue
		}
		if _, err := tx.Exec(
			ctx,
			"insert into app_user_role (user_id, role) values ($1, $2) on conflict do nothing",
			user.ID,
			role,
		); err != nil {
			return authUser{}, err
		}
	}

	if hasRole(user, "student") {
		if _, err := tx.Exec(
			ctx,
			`
				insert into pet_state (user_id, hunger, happiness, energy)
				values ($1, 50, 50, 50)
				on conflict (user_id) do nothing
			`,
			user.ID,
		); err != nil {
			return authUser{}, err
		}
		if _, err := tx.Exec(
			ctx,
			`
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				select $1, iit.id, 1
				from inventory_item_type iit
				where iit.key in ('sunny_hoodie', 'star_cap')
				on conflict (app_user_id, item_type_id) do update
				set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
					updated_at = now()
			`,
			user.ID,
		); err != nil {
			return authUser{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return authUser{}, err
	}

	return user, nil
}

func (app *app) loadStudents(ctx context.Context) ([]authUser, error) {
	rows, err := app.db.Query(
		ctx,
		`
			select u.id, u.keycloak_subject, u.display_name, coalesce(u.email, '')
			from app_user u
			join app_user_role ur on ur.user_id = u.id and ur.role = 'student'
			order by lower(u.display_name), u.id
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []authUser{}
	for rows.Next() {
		var student authUser
		if err := rows.Scan(
			&student.ID,
			&student.KeycloakSubject,
			&student.DisplayName,
			&student.Email,
		); err != nil {
			return nil, err
		}
		student.Roles = []string{"student"}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return students, nil
}

func requireUser(ctx context.Context) (authUser, error) {
	user, ok := userFromContext(ctx)
	if !ok {
		return authUser{}, pgx.ErrNoRows
	}
	return user, nil
}
