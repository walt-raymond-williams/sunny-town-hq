package main

import (
	"context"
	"database/sql"
	"math"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	petv1 "hq/proto/hq/pet/v1"
)

const (
	petUserID                int64 = 1
	petDailyDecayPoints            = 144.0
	petSleepRecoveryDuration       = 10 * time.Minute
	petDecayTickInterval           = 5 * time.Minute
	petGameTargetScore             = 10
	petGameEnergyCost              = 5
)

type petService struct {
	app *app
}

func (service *petService) GetPetState(ctx context.Context, _ *connect.Request[petv1.GetPetStateRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.loadStudentProfile(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) FeedPet(ctx context.Context, _ *connect.Request[petv1.FeedPetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.feedStudentPet(ctx)
	if errNoCookies == err {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) PlayWithPet(ctx context.Context, _ *connect.Request[petv1.PlayWithPetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.playWithStudentPet(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) ApplyGameResult(ctx context.Context, request *connect.Request[petv1.ApplyGameResultRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.applyGameResult(ctx, int(request.Msg.GetScore()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) PutPetToSleep(ctx context.Context, _ *connect.Request[petv1.PutPetToSleepRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.putStudentPetToSleep(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) WakePet(ctx context.Context, _ *connect.Request[petv1.WakePetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	profile, err := service.app.wakeStudentPet(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *petService) WatchPetState(ctx context.Context, _ *connect.Request[petv1.WatchPetStateRequest], stream *connect.ServerStream[petv1.PetStateResponse]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		profile, err := service.app.loadStudentProfile(ctx)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}

		if err := stream.Send(profile.toProto()); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (app *app) startPetDecayTicker(ctx context.Context) {
	ticker := time.NewTicker(petDecayTickInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := app.applyPetDecay(context.Background()); err != nil {
					// Decay is opportunistic; request-time decay will catch up later.
					continue
				}
			}
		}
	}()
}

func (app *app) applyPetDecay(ctx context.Context) error {
	tx, err := app.db.Begin(ctx)
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
		petUserID,
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
		dailyDecay := int(math.Floor(elapsed.Hours() * petDailyDecayPoints / 24.0))
		if dailyDecay > 0 {
			hunger = clampPetStat(hunger - dailyDecay)
			happiness = clampPetStat(happiness - dailyDecay)
			if !sleeping {
				energy = clampPetStat(energy - dailyDecay)
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
			happiness = clampPetStat(happiness + 10)
			sleepStartedAt = sql.NullTime{}
			sleepStartedEnergy = sql.NullInt64{}
			changed = true
		} else {
			sleepElapsed := now.Sub(startedAt)
			if sleepElapsed < 0 {
				sleepElapsed = 0
			}

			if sleepElapsed >= petSleepRecoveryDuration {
				energy = 100
				sleeping = false
				happiness = clampPetStat(happiness + 10)
				sleepStartedAt = sql.NullTime{}
				sleepStartedEnergy = sql.NullInt64{}
				changed = true
			} else {
				progress := sleepElapsed.Seconds() / petSleepRecoveryDuration.Seconds()
				recovered := startEnergy + int(math.Ceil(float64(100-startEnergy)*progress))
				recovered = clampPetStat(recovered)
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
		petUserID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (app *app) playWithStudentPet(ctx context.Context) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx); err != nil {
		return studentProfileResponse{}, err
	}

	_, err := app.db.Exec(
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
		petUserID,
	)
	if err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfile(ctx)
}

func (app *app) applyGameResult(ctx context.Context, score int) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx); err != nil {
		return studentProfileResponse{}, err
	}

	if score < 0 {
		score = 0
	}

	won := score >= petGameTargetScore
	happinessDelta := score
	if won {
		happinessDelta = score * 2
		if happinessDelta > 20 {
			happinessDelta = 20
		}
	} else if happinessDelta > 8 {
		happinessDelta = 8
	}

	_, err := app.db.Exec(
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
		petGameEnergyCost,
		petUserID,
	)
	if err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfile(ctx)
}

func (app *app) putStudentPetToSleep(ctx context.Context) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx); err != nil {
		return studentProfileResponse{}, err
	}

	_, err := app.db.Exec(
		ctx,
		`
			update pet_state
			set sleeping = true,
				sleep_started_at = now(),
				sleep_started_energy = energy,
				updated_at = now()
			where user_id = $1
		`,
		petUserID,
	)
	if err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfile(ctx)
}

func (app *app) wakeStudentPet(ctx context.Context) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx); err != nil {
		return studentProfileResponse{}, err
	}

	_, err := app.db.Exec(
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
		petUserID,
	)
	if err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfile(ctx)
}

func clampPetStat(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (profile studentProfileResponse) toProto() *petv1.PetStateResponse {
	return &petv1.PetStateResponse{
		UserId:      profile.ID,
		DisplayName: profile.DisplayName,
		Cookies:     int32(profile.Cookies),
		PetState: &petv1.PetState{
			Hunger:      int32(profile.PetState.Hunger),
			Happiness:   int32(profile.PetState.Happiness),
			Energy:      int32(profile.PetState.Energy),
			Sleeping:    profile.PetState.Sleeping,
			Mood:        profile.PetState.protoMood(),
			UpdatedAt:   timestamppb.New(profile.PetState.UpdatedAt),
			LastDecayAt: timestamppb.New(profile.PetState.LastDecayAt),
		},
	}
}

func (state petStateResponse) protoMood() petv1.PetMood {
	switch state.Mood {
	case "sleeping":
		return petv1.PetMood_PET_MOOD_SLEEPING
	case "hungry":
		return petv1.PetMood_PET_MOOD_HUNGRY
	case "sad":
		return petv1.PetMood_PET_MOOD_SAD
	case "happy":
		return petv1.PetMood_PET_MOOD_HAPPY
	default:
		return petv1.PetMood_PET_MOOD_IDLE
	}
}
