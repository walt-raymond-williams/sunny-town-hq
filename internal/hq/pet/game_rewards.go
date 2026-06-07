package pet

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (store *Store) ApplyGameResult(ctx context.Context, userID int64, score int, starsCollected int, roundID string) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	score, starsCollected = NormalizeGameResult(score, starsCollected)
	happinessDelta := GameHappinessDelta(score)

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sleeping bool
	if err := tx.QueryRow(
		ctx,
		`
			select sleeping
			from pet_state
			where user_id = $1
			for update
		`,
		userID,
	).Scan(&sleeping); err != nil {
		return Profile{}, err
	}
	if sleeping {
		if err := tx.Commit(ctx); err != nil {
			return Profile{}, err
		}
		return store.LoadProfile(ctx, userID)
	}

	applyPetResult := true
	if starsCollected > 0 {
		roundID = strings.TrimSpace(roundID)
		if roundID == "" {
			roundID = fmt.Sprintf("legacy-%d", time.Now().UTC().UnixNano())
		}
		inserted, _, err := store.CommitStudentStarOnce(ctx, tx, StarRewardRequest{
			EventID:       fmt.Sprintf("pet-falling-stars:%d:%s", userID, roundID),
			AppUserID:     userID,
			Source:        "pet_falling_stars",
			Delta:         starsCollected,
			CollectibleID: roundID,
		})
		if err != nil {
			return Profile{}, err
		}
		applyPetResult = inserted
	} else if _, err := tx.Exec(
		ctx,
		`
			insert into student_wallet (app_user_id)
			values ($1)
			on conflict (app_user_id) do nothing
		`,
		userID,
	); err != nil {
		return Profile{}, err
	}

	if applyPetResult {
		_, err = tx.Exec(
			ctx,
			`
				update pet_state
				set happiness = least(happiness + $1, 100),
					energy = greatest(energy - $2, 0),
					sleeping = case when energy <= $2 then true else false end,
					sleep_started_at = case when energy <= $2 then now() else null end,
					sleep_started_energy = case when energy <= $2 then 0 else null end,
					updated_at = now()
				where user_id = $3
					and not sleeping
			`,
			happinessDelta,
			GameEnergyCost,
			userID,
		)
		if err != nil {
			return Profile{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}

	return store.LoadProfile(ctx, userID)
}
