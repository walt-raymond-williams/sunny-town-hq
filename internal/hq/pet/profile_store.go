package pet

import "context"

func (store *Store) LoadProfile(ctx context.Context, userID int64) (Profile, error) {
	if err := store.ApplyDecay(ctx, userID); err != nil {
		return Profile{}, err
	}

	return store.loadProfileWithoutDecay(ctx, userID)
}

func (store *Store) loadProfileWithoutDecay(ctx context.Context, userID int64) (Profile, error) {
	var profile Profile
	err := store.DB.QueryRow(
		ctx,
		`
			select u.id,
				u.display_name,
				coalesce(cookie_inventory.quantity, 0),
				coalesce(sw.star_balance, 0),
				ps.hunger,
				ps.happiness,
				ps.energy,
				ps.sleeping,
				ps.updated_at,
				ps.last_decay_at,
				case
					when ps.sleeping then 'sleeping'
					when ps.hunger = 0 then 'hungry'
					when ps.happiness = 0 then 'sad'
					else 'idle'
				end as mood
			from app_user u
			join pet_state ps on ps.user_id = u.id
			left join student_wallet sw on sw.app_user_id = u.id
			left join inventory_item_type cookie_type on cookie_type.key = 'cookie'
			left join student_inventory_item cookie_inventory on cookie_inventory.app_user_id = u.id
				and cookie_inventory.item_type_id = cookie_type.id
			where u.id = $1
		`,
		userID,
	).Scan(
		&profile.ID,
		&profile.DisplayName,
		&profile.Cookies,
		&profile.StarBalance,
		&profile.PetState.Hunger,
		&profile.PetState.Happiness,
		&profile.PetState.Energy,
		&profile.PetState.Sleeping,
		&profile.PetState.UpdatedAt,
		&profile.PetState.LastDecayAt,
		&profile.PetState.Mood,
	)
	return profile, err
}
