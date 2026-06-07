package pet

import (
	"context"
	"database/sql"
	"math"
	"time"
)

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
