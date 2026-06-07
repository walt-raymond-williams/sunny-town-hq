package pet

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StarRewardRequest struct {
	EventID       string
	AppUserID     int64
	Source        string
	Delta         int
	RoomID        string
	MapID         string
	CollectibleID string
}

type InventoryConsumer func(ctx context.Context, tx pgx.Tx, userID int64, itemKey string, quantity int) (bool, error)

type StarRewardCommitter func(ctx context.Context, tx pgx.Tx, request StarRewardRequest) (bool, int, error)

type Store struct {
	DB                    *pgxpool.Pool
	CookieInventoryKey    string
	NoCookiesError        error
	ConsumeInventoryItem  InventoryConsumer
	CommitStudentStarOnce StarRewardCommitter
}

func (store *Store) StartDecayTicker(ctx context.Context) {
	ticker := time.NewTicker(DecayTickInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				_ = store.ApplyDecayForAll(context.Background())
			}
		}
	}()
}

func (store *Store) Feed(ctx context.Context, userID int64) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	consumed, err := store.ConsumeInventoryItem(ctx, tx, userID, store.CookieInventoryKey, 1)
	if err != nil {
		return Profile{}, err
	}
	if !consumed {
		return Profile{}, store.NoCookiesError
	}

	if _, err := tx.Exec(
		ctx,
		`
			update pet_state
			set hunger = least(hunger + 10, 100),
				updated_at = now()
			where user_id = $1
		`,
		userID,
	); err != nil {
		return Profile{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}

	return store.LoadProfile(ctx, userID)
}

func (store *Store) Play(ctx context.Context, userID int64) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	if _, err := store.DB.Exec(
		ctx,
		`
			update pet_state
			set happiness = least(happiness + 10, 100),
				energy = greatest(energy - 10, 0),
				sleeping = case when energy <= 10 then true else false end,
				sleep_started_at = case when energy <= 10 then now() else null end,
				sleep_started_energy = case when energy <= 10 then 0 else null end,
				updated_at = now()
			where user_id = $1
				and not sleeping
				and energy >= 10
		`,
		userID,
	); err != nil {
		return Profile{}, err
	}

	return store.LoadProfile(ctx, userID)
}

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

func (store *Store) PutToSleep(ctx context.Context, userID int64) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	if _, err := store.DB.Exec(
		ctx,
		`
			update pet_state
			set sleeping = true,
				sleep_started_at = now(),
				sleep_started_energy = energy,
				updated_at = now()
			where user_id = $1
		`,
		userID,
	); err != nil {
		return Profile{}, err
	}

	return store.LoadProfile(ctx, userID)
}

func (store *Store) Wake(ctx context.Context, userID int64) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	if _, err := store.DB.Exec(
		ctx,
		`
			update pet_state
			set sleeping = false,
				happiness = greatest(happiness - 30, 0),
				sleep_started_at = null,
				sleep_started_energy = null,
				updated_at = now()
			where user_id = $1
				and sleeping
		`,
		userID,
	); err != nil {
		return Profile{}, err
	}

	return store.LoadProfile(ctx, userID)
}

func (store *Store) ApplyDecayForAll(ctx context.Context) error {
	rows, err := store.DB.Query(ctx, "select user_id from pet_state")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		if err := store.ApplyDecay(ctx, userID); err != nil {
			continue
		}
	}
	return rows.Err()
}

func (store *Store) ApplyDecay(ctx context.Context, userID int64) error {
	tx, err := store.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var hunger int
	var happiness int
	var energy int
	var sleeping bool
	var lastDecayAt time.Time
	var sleepStartedAt sql.NullTime
	var sleepStartedEnergy sql.NullInt64
	if err := tx.QueryRow(
		ctx,
		`
			select hunger, happiness, energy, sleeping, last_decay_at, sleep_started_at, sleep_started_energy
			from pet_state
			where user_id = $1
			for update
		`,
		userID,
	).Scan(
		&hunger,
		&happiness,
		&energy,
		&sleeping,
		&lastDecayAt,
		&sleepStartedAt,
		&sleepStartedEnergy,
	); err != nil {
		return err
	}

	now := time.Now().UTC()
	changed := false
	decayApplied := false

	if now.After(lastDecayAt) {
		elapsed := now.Sub(lastDecayAt)
		dailyDecay := int(math.Floor(elapsed.Hours() * DailyDecayPoints / 24.0))
		if dailyDecay > 0 {
			hunger = ClampStat(hunger - dailyDecay)
			happiness = ClampStat(happiness - dailyDecay)
			if !sleeping {
				energy = ClampStat(energy - dailyDecay)
			}
			changed = true
			decayApplied = true
		}
	}

	if sleeping {
		startedAt := sleepStartedAt.Time
		if !sleepStartedAt.Valid {
			startedAt = now
			sleepStartedAt = sql.NullTime{Time: startedAt, Valid: true}
			changed = true
		}

		startEnergy := energy
		if sleepStartedEnergy.Valid {
			startEnergy = int(sleepStartedEnergy.Int64)
		} else {
			sleepStartedEnergy = sql.NullInt64{Int64: int64(startEnergy), Valid: true}
			changed = true
		}

		if startEnergy >= 100 {
			energy = 100
			sleeping = false
			happiness = ClampStat(happiness + 10)
			sleepStartedAt = sql.NullTime{}
			sleepStartedEnergy = sql.NullInt64{}
			changed = true
		} else {
			sleepElapsed := now.Sub(startedAt)
			if sleepElapsed < 0 {
				sleepElapsed = 0
			}

			if sleepElapsed >= SleepRecoveryDuration {
				energy = 100
				sleeping = false
				happiness = ClampStat(happiness + 10)
				sleepStartedAt = sql.NullTime{}
				sleepStartedEnergy = sql.NullInt64{}
				changed = true
			} else {
				progress := sleepElapsed.Seconds() / SleepRecoveryDuration.Seconds()
				recovered := startEnergy + int(math.Ceil(float64(100-startEnergy)*progress))
				recovered = ClampStat(recovered)
				if recovered > energy {
					energy = recovered
					changed = true
				}
			}
		}
	}

	if !sleeping && energy == 0 {
		sleeping = true
		sleepStartedAt = sql.NullTime{Time: now, Valid: true}
		sleepStartedEnergy = sql.NullInt64{Int64: 0, Valid: true}
		changed = true
	}

	if !changed {
		return tx.Commit(ctx)
	}

	var sleepStartedAtValue any
	if sleepStartedAt.Valid {
		sleepStartedAtValue = sleepStartedAt.Time
	}

	var sleepStartedEnergyValue any
	if sleepStartedEnergy.Valid {
		sleepStartedEnergyValue = int(sleepStartedEnergy.Int64)
	}

	_, err = tx.Exec(
		ctx,
		`
			update pet_state
			set hunger = $1,
				happiness = $2,
				energy = $3,
				sleeping = $4,
				updated_at = now(),
				sleep_started_at = $5,
				sleep_started_energy = $6,
				last_decay_at = case when $7 then now() else last_decay_at end
			where user_id = $8
		`,
		hunger,
		happiness,
		energy,
		sleeping,
		sleepStartedAtValue,
		sleepStartedEnergyValue,
		decayApplied,
		userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
