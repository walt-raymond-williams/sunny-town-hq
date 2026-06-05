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
