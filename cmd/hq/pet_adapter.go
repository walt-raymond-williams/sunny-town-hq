package main

import (
	"context"

	hqpet "hq/internal/hq/pet"
)

type hqPetBackend struct {
	app *app
}

func requireStudentID(ctx context.Context) (int64, error) {
	user, err := requireStudentUser(ctx)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (backend hqPetBackend) LoadProfile(ctx context.Context, userID int64) (hqpet.Profile, error) {
	profile, err := backend.app.loadStudentProfile(ctx, userID)
	return toPetProfile(profile), err
}

func (backend hqPetBackend) Feed(ctx context.Context, userID int64) (hqpet.Profile, error) {
	profile, err := backend.app.feedStudentPet(ctx, userID)
	return toPetProfile(profile), err
}

func (backend hqPetBackend) Play(ctx context.Context, userID int64) (hqpet.Profile, error) {
	profile, err := backend.app.playWithStudentPet(ctx, userID)
	return toPetProfile(profile), err
}

func (backend hqPetBackend) ApplyGameResult(ctx context.Context, userID int64, score int, starsCollected int, roundID string) (hqpet.Profile, error) {
	profile, err := backend.app.applyGameResult(ctx, userID, score, starsCollected, roundID)
	return toPetProfile(profile), err
}

func (backend hqPetBackend) PutToSleep(ctx context.Context, userID int64) (hqpet.Profile, error) {
	profile, err := backend.app.putStudentPetToSleep(ctx, userID)
	return toPetProfile(profile), err
}

func (backend hqPetBackend) Wake(ctx context.Context, userID int64) (hqpet.Profile, error) {
	profile, err := backend.app.wakeStudentPet(ctx, userID)
	return toPetProfile(profile), err
}

func toPetProfile(profile studentProfileResponse) hqpet.Profile {
	return hqpet.Profile{
		ID:          profile.ID,
		DisplayName: profile.DisplayName,
		Cookies:     profile.Cookies,
		StarBalance: profile.StarBalance,
		PetState: hqpet.State{
			Hunger:      profile.PetState.Hunger,
			Happiness:   profile.PetState.Happiness,
			Energy:      profile.PetState.Energy,
			Sleeping:    profile.PetState.Sleeping,
			Mood:        profile.PetState.Mood,
			UpdatedAt:   profile.PetState.UpdatedAt,
			LastDecayAt: profile.PetState.LastDecayAt,
		},
	}
}

func fromPetProfile(profile hqpet.Profile) studentProfileResponse {
	return studentProfileResponse{
		ID:          profile.ID,
		DisplayName: profile.DisplayName,
		Cookies:     profile.Cookies,
		StarBalance: profile.StarBalance,
		PetState: petStateResponse{
			Hunger:      profile.PetState.Hunger,
			Happiness:   profile.PetState.Happiness,
			Energy:      profile.PetState.Energy,
			Sleeping:    profile.PetState.Sleeping,
			Mood:        profile.PetState.Mood,
			UpdatedAt:   profile.PetState.UpdatedAt,
			LastDecayAt: profile.PetState.LastDecayAt,
		},
	}
}
