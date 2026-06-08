package characters

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	TypePlayer    = "player"
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
	err := querier.QueryRow(
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
	).Scan(
		&character.ID,
		&character.Type,
		&character.AppUserID,
		&character.RoomID,
		&character.DisplayName,
		&character.AvatarID,
	)
	if err != nil {
		return Character{}, err
	}
	return character, nil
}
