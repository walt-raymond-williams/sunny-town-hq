package users

import (
	"context"
	"strings"

	hqauth "hq/internal/hq/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (store *Store) SyncAuthenticated(ctx context.Context, user hqauth.User) (hqauth.User, error) {
	if user.KeycloakSubject == "" {
		return hqauth.User{}, hqauth.ErrInvalidToken
	}

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return hqauth.User{}, err
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
		return hqauth.User{}, err
	}

	if _, err := tx.Exec(ctx, "delete from app_user_role where user_id = $1", user.ID); err != nil {
		return hqauth.User{}, err
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
			return hqauth.User{}, err
		}
	}

	if hqauth.HasRole(user, "student") {
		if _, err := tx.Exec(
			ctx,
			`
				insert into pet_state (user_id, hunger, happiness, energy)
				values ($1, 50, 50, 50)
				on conflict (user_id) do nothing
			`,
			user.ID,
		); err != nil {
			return hqauth.User{}, err
		}
		if _, err := tx.Exec(
			ctx,
			`
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				select $1, iit.id, 1
				from inventory_item_type iit
				where iit.key in ('sunny_hoodie', 'star_cap', 'pickaxe')
				on conflict (app_user_id, item_type_id) do update
				set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
					updated_at = now()
			`,
			user.ID,
		); err != nil {
			return hqauth.User{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return hqauth.User{}, err
	}

	return user, nil
}

func (store *Store) LoadStudents(ctx context.Context) ([]hqauth.User, error) {
	rows, err := store.db.Query(
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

	students := []hqauth.User{}
	for rows.Next() {
		var student hqauth.User
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

func Require(ctx context.Context) (hqauth.User, error) {
	user, ok := hqauth.UserFromContext(ctx)
	if !ok {
		return hqauth.User{}, pgx.ErrNoRows
	}
	return user, nil
}
