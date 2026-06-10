package progression

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MiningSkillKey        = "mining"
	MiningHarvestXP       = 10
	defaultXPPerLevel     = 100
	sourceSunnyTownMining = "sunny_town_mining"
)

type AwardSkillXPRequest struct {
	EventID     string `json:"event_id"`
	CharacterID int64  `json:"character_id"`
	Source      string `json:"source"`
	ActivityKey string `json:"activity_key"`
	SkillKey    string `json:"skill_key"`
	XPAmount    int    `json:"xp_amount"`
	RoomID      string `json:"room_id,omitempty"`
	MapID       string `json:"map_id,omitempty"`
	NodeID      string `json:"node_id,omitempty"`
}

type AwardSkillXPResponse struct {
	Accepted  bool               `json:"accepted"`
	Duplicate bool               `json:"duplicate"`
	Skill     SkillStateResponse `json:"skill"`
}

type CharacterProgressionResponse struct {
	CharacterID int64                `json:"characterId"`
	Skills      []SkillStateResponse `json:"skills"`
}

type SkillStateResponse struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	XP             int    `json:"xp"`
	Level          int    `json:"level"`
	CurrentLevelXP int    `json:"currentLevelXp"`
	NextLevelXP    int    `json:"nextLevelXp"`
}

type mutationQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func AwardSkillXP(ctx context.Context, db *pgxpool.Pool, request AwardSkillXPRequest) (AwardSkillXPResponse, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return AwardSkillXPResponse{}, err
	}
	defer tx.Rollback(ctx)

	response, err := AwardSkillXPInTx(ctx, tx, request)
	if err != nil {
		return AwardSkillXPResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AwardSkillXPResponse{}, err
	}
	return response, nil
}

func AwardMiningHarvestXPInTx(ctx context.Context, querier mutationQuerier, eventID string, characterID int64, roomID string, mapID string, nodeID string) (AwardSkillXPResponse, error) {
	return AwardSkillXPInTx(ctx, querier, AwardSkillXPRequest{
		EventID:     "mining-xp:" + strings.TrimSpace(eventID),
		CharacterID: characterID,
		Source:      sourceSunnyTownMining,
		ActivityKey: "resource_harvest",
		SkillKey:    MiningSkillKey,
		XPAmount:    MiningHarvestXP,
		RoomID:      roomID,
		MapID:       mapID,
		NodeID:      nodeID,
	})
}

func AwardSkillXPInTx(ctx context.Context, querier mutationQuerier, request AwardSkillXPRequest) (AwardSkillXPResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ActivityKey = strings.TrimSpace(request.ActivityKey)
	request.SkillKey = strings.TrimSpace(request.SkillKey)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NodeID = strings.TrimSpace(request.NodeID)
	if request.EventID == "" || request.CharacterID < 1 || request.Source == "" || request.ActivityKey == "" || request.SkillKey == "" || request.XPAmount < 1 {
		return AwardSkillXPResponse{}, errors.New("skill xp award is missing required fields")
	}
	if request.SkillKey != MiningSkillKey {
		return AwardSkillXPResponse{}, errors.New("unsupported skill")
	}

	if err := lockSkillXPEvent(ctx, querier, request.EventID); err != nil {
		return AwardSkillXPResponse{}, err
	}

	var inserted bool
	err := querier.QueryRow(
		ctx,
		`
			with skill_definition as (
				select skill_key, xp_per_level
				from sunny_town_skill_definition
				where skill_key = $5
			),
			inserted_ledger as (
				insert into sunny_town_character_skill_xp_ledger (
					event_id,
					character_id,
					source,
					activity_key,
					skill_key,
					xp_amount,
					room_id,
					map_id,
					node_id
				)
				select $1, $2, $3, $4, skill_key, $6, nullif($7, ''), nullif($8, ''), nullif($9, '')
				from skill_definition
				on conflict (event_id) do nothing
				returning character_id, skill_key, xp_amount
			),
			updated_skill as (
				insert into sunny_town_character_skill (
					character_id,
					skill_key,
					xp,
					level
				)
				select character_id,
					skill_key,
					xp_amount,
					1 + (xp_amount / (select xp_per_level from skill_definition))
				from inserted_ledger
				on conflict (character_id, skill_key) do update
				set xp = sunny_town_character_skill.xp + excluded.xp,
					level = 1 + ((sunny_town_character_skill.xp + excluded.xp) / (select xp_per_level from skill_definition)),
					updated_at = now()
				returning character_id
			)
			select exists(select 1 from inserted_ledger) as inserted
		`,
		request.EventID,
		request.CharacterID,
		request.Source,
		request.ActivityKey,
		request.SkillKey,
		request.XPAmount,
		request.RoomID,
		request.MapID,
		request.NodeID,
	).Scan(&inserted)
	if err != nil {
		return AwardSkillXPResponse{}, err
	}

	skill, err := loadCharacterSkill(ctx, querier, request.CharacterID, request.SkillKey)
	if err != nil {
		return AwardSkillXPResponse{}, err
	}
	return AwardSkillXPResponse{Accepted: true, Duplicate: !inserted, Skill: skill}, nil
}

func LoadStudentProgression(ctx context.Context, db *pgxpool.Pool, appUserID int64) (CharacterProgressionResponse, error) {
	if appUserID < 1 {
		return CharacterProgressionResponse{}, errors.New("app_user_id is required")
	}

	var characterID int64
	err := db.QueryRow(
		ctx,
		`
			select id
			from sunny_town_character
			where app_user_id = $1 and character_type = 'player'
		`,
		appUserID,
	).Scan(&characterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CharacterProgressionResponse{Skills: []SkillStateResponse{}}, nil
	}
	if err != nil {
		return CharacterProgressionResponse{}, err
	}

	rows, err := db.Query(
		ctx,
		`
			select sd.skill_key,
				sd.display_name,
				sd.description,
				coalesce(cs.xp, 0) as xp,
				coalesce(cs.level, 1) as level,
				sd.xp_per_level
			from sunny_town_skill_definition sd
			left join sunny_town_character_skill cs on cs.skill_key = sd.skill_key
				and cs.character_id = $1
			where sd.skill_key = $2
			order by sd.skill_key
		`,
		characterID,
		MiningSkillKey,
	)
	if err != nil {
		return CharacterProgressionResponse{}, err
	}
	defer rows.Close()

	response := CharacterProgressionResponse{CharacterID: characterID, Skills: []SkillStateResponse{}}
	for rows.Next() {
		var skill SkillStateResponse
		var xpPerLevel int
		if err := rows.Scan(&skill.Key, &skill.Name, &skill.Description, &skill.XP, &skill.Level, &xpPerLevel); err != nil {
			return CharacterProgressionResponse{}, err
		}
		applyLevelProgress(&skill, xpPerLevel)
		response.Skills = append(response.Skills, skill)
	}
	if err := rows.Err(); err != nil {
		return CharacterProgressionResponse{}, err
	}
	return response, nil
}

func lockSkillXPEvent(ctx context.Context, querier mutationQuerier, eventID string) error {
	var locked int
	return querier.QueryRow(
		ctx,
		`select 1 from (select pg_advisory_xact_lock(hashtext('skill-xp:' || $1))) as locked`,
		eventID,
	).Scan(&locked)
}

func loadCharacterSkill(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, characterID int64, skillKey string) (SkillStateResponse, error) {
	var skill SkillStateResponse
	var xpPerLevel int
	err := querier.QueryRow(
		ctx,
		`
			select sd.skill_key,
				sd.display_name,
				sd.description,
				coalesce(cs.xp, 0) as xp,
				coalesce(cs.level, 1) as level,
				sd.xp_per_level
			from sunny_town_skill_definition sd
			left join sunny_town_character_skill cs on cs.skill_key = sd.skill_key
				and cs.character_id = $1
			where sd.skill_key = $2
		`,
		characterID,
		skillKey,
	).Scan(&skill.Key, &skill.Name, &skill.Description, &skill.XP, &skill.Level, &xpPerLevel)
	if err != nil {
		return SkillStateResponse{}, err
	}
	applyLevelProgress(&skill, xpPerLevel)
	return skill, nil
}

func applyLevelProgress(skill *SkillStateResponse, xpPerLevel int) {
	if xpPerLevel < 1 {
		xpPerLevel = defaultXPPerLevel
	}
	skill.CurrentLevelXP = skill.XP % xpPerLevel
	skill.NextLevelXP = xpPerLevel
}
