package main

import (
	"context"

	hqinventory "hq/internal/hq/inventory"
	hqpet "hq/internal/hq/pet"

	"github.com/jackc/pgx/v5"
)

func (app *app) petStore() *hqpet.Store {
	return &hqpet.Store{
		DB:                    app.db,
		CookieInventoryKey:    hqinventory.CookieKey,
		NoCookiesError:        errNoCookies,
		ConsumeInventoryItem:  consumePetInventoryItem,
		CommitStudentStarOnce: commitPetStarReward,
	}
}

func consumePetInventoryItem(ctx context.Context, tx pgx.Tx, userID int64, itemKey string, quantity int) (bool, error) {
	return hqinventory.ConsumeStudentItem(ctx, tx, userID, itemKey, quantity)
}

func (app *app) startPetDecayTicker(ctx context.Context) {
	app.petStore().StartDecayTicker(ctx)
}

func (app *app) applyPetDecayForAll(ctx context.Context) error {
	return app.petStore().ApplyDecayForAll(ctx)
}

func (app *app) applyPetDecay(ctx context.Context, userID int64) error {
	return app.petStore().ApplyDecay(ctx, userID)
}

func (app *app) feedStudentPet(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().Feed(ctx, userID)
	return fromPetProfile(profile), err
}

func (app *app) playWithStudentPet(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().Play(ctx, userID)
	return fromPetProfile(profile), err
}

func (app *app) applyGameResult(ctx context.Context, userID int64, score int, starsCollected int, roundID string) (studentProfileResponse, error) {
	profile, err := app.petStore().ApplyGameResult(ctx, userID, score, starsCollected, roundID)
	return fromPetProfile(profile), err
}

func (app *app) putStudentPetToSleep(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().PutToSleep(ctx, userID)
	return fromPetProfile(profile), err
}

func (app *app) wakeStudentPet(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().Wake(ctx, userID)
	return fromPetProfile(profile), err
}

func requireStudentUser(ctx context.Context) (authUser, error) {
	user, ok := userFromContext(ctx)
	if !ok || !hasRole(user, "student") {
		return authUser{}, errInvalidToken
	}
	return user, nil
}
