package pet

import "context"

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
