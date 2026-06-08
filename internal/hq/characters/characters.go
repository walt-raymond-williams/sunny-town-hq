package characters

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	TypePlayer    = "player"
	TypeNPC       = "npc"
	DefaultRoomID = "sunny-town-main"
	DefaultAvatar = "pet-default"
)

type Character struct {
	ID          int64
	Type        string
	AppUserID   int64
	RoomID      string
	DisplayName string
	AvatarID    string
}

type NPCCharacter struct {
	Character
	NPCKey string
}

type NPCSpec struct {
	RoomID      string
	NPCKey      string
	DisplayName string
	AvatarID    string
}

type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func EnsurePlayer(ctx context.Context, querier Querier, appUserID int64, displayName string, avatarID string) (Character, error) {
	displayName = strings.TrimSpace(displayName)
	avatarID = strings.TrimSpace(avatarID)
	if appUserID < 1 || displayName == "" {
		return Character{}, errors.New("player character is missing required fields")
	}
	if avatarID == "" {
		avatarID = DefaultAvatar
	}

	var character Character
	row := querier.QueryRow(
		ctx,
		`
			insert into sunny_town_character (
				character_type,
				app_user_id,
				room_id,
				display_name,
				avatar_id
			)
			values ('player', $1, $2, $3, $4)
			on conflict (app_user_id) do update
			set display_name = excluded.display_name,
				avatar_id = excluded.avatar_id,
				updated_at = now()
			returning id, character_type, app_user_id, room_id, display_name, avatar_id
		`,
		appUserID,
		DefaultRoomID,
		displayName,
		avatarID,
	)
	if err := scanCharacter(row, &character); err != nil {
		return Character{}, err
	}
	return character, nil
}

func EnsureNPC(ctx context.Context, querier Querier, spec NPCSpec) (NPCCharacter, error) {
	spec.RoomID = strings.TrimSpace(spec.RoomID)
	spec.NPCKey = strings.TrimSpace(spec.NPCKey)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.AvatarID = strings.TrimSpace(spec.AvatarID)
	if spec.RoomID == "" || spec.NPCKey == "" || spec.DisplayName == "" {
		return NPCCharacter{}, errors.New("npc character is missing required fields")
	}
	if spec.AvatarID == "" {
		spec.AvatarID = DefaultAvatar
	}

	var character NPCCharacter
	row := querier.QueryRow(
		ctx,
		`
			with existing_mapping as (
				select character_id
				from sunny_town_npc_character
				where room_id = $1 and npc_key = $2
			),
			inserted_character as (
				insert into sunny_town_character (
					character_type,
					app_user_id,
					room_id,
					display_name,
					avatar_id
				)
				select 'npc', null, $1, $3, $4
				where not exists (select 1 from existing_mapping)
				returning id
			),
			target_character as (
				select character_id as id from existing_mapping
				union all
				select id from inserted_character
			),
			inserted_mapping as (
				insert into sunny_town_npc_character (
					character_id,
					room_id,
					npc_key
				)
				select id, $1, $2
				from inserted_character
				on conflict (room_id, npc_key) do nothing
			)
			update sunny_town_character c
			set display_name = $3,
				avatar_id = $4,
				updated_at = now()
			from target_character t
			where c.id = t.id
			returning c.id, c.character_type, c.app_user_id, c.room_id, c.display_name, c.avatar_id
		`,
		spec.RoomID,
		spec.NPCKey,
		spec.DisplayName,
		spec.AvatarID,
	)
	if err := scanCharacter(row, &character.Character); err != nil {
		return NPCCharacter{}, err
	}
	character.NPCKey = spec.NPCKey
	return character, nil
}

func EnsureNPCs(ctx context.Context, querier Querier, specs []NPCSpec) ([]NPCCharacter, error) {
	characters := make([]NPCCharacter, 0, len(specs))
	for _, spec := range specs {
		character, err := EnsureNPC(ctx, querier, spec)
		if err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}
	return characters, nil
}

func scanCharacter(row pgx.Row, character *Character) error {
	var appUserID sql.NullInt64
	if err := row.Scan(
		&character.ID,
		&character.Type,
		&appUserID,
		&character.RoomID,
		&character.DisplayName,
		&character.AvatarID,
	); err != nil {
		return err
	}
	if appUserID.Valid {
		character.AppUserID = appUserID.Int64
	}
	return nil
}
