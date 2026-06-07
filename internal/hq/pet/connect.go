package pet

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	petv1 "hq/proto/hq/pet/v1"
	petv1connect "hq/proto/hq/pet/v1/petv1connect"
)

type State struct {
	Hunger      int       `json:"hunger"`
	Happiness   int       `json:"happiness"`
	Energy      int       `json:"energy"`
	Sleeping    bool      `json:"sleeping"`
	Mood        string    `json:"mood"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastDecayAt time.Time `json:"last_decay_at"`
}

type Profile struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Cookies     int    `json:"cookies"`
	StarBalance int    `json:"star_balance"`
	PetState    State  `json:"pet_state"`
}

type Backend interface {
	LoadProfile(ctx context.Context, userID int64) (Profile, error)
	Feed(ctx context.Context, userID int64) (Profile, error)
	Play(ctx context.Context, userID int64) (Profile, error)
	ApplyGameResult(ctx context.Context, userID int64, score int, starsCollected int, roundID string) (Profile, error)
	PutToSleep(ctx context.Context, userID int64) (Profile, error)
	Wake(ctx context.Context, userID int64) (Profile, error)
}

type RequireStudentFunc func(ctx context.Context) (int64, error)

type connectService struct {
	backend        Backend
	requireStudent RequireStudentFunc
	noCookiesError error
}

func NewServiceHandler(backend Backend, requireStudent RequireStudentFunc, noCookiesError error) (string, http.Handler) {
	return petv1connect.NewPetServiceHandler(&connectService{
		backend:        backend,
		requireStudent: requireStudent,
		noCookiesError: noCookiesError,
	})
}

func (service *connectService) GetPetState(ctx context.Context, _ *connect.Request[petv1.GetPetStateRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.LoadProfile(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) FeedPet(ctx context.Context, _ *connect.Request[petv1.FeedPetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.Feed(ctx, userID)
	if errors.Is(err, service.noCookiesError) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) PlayWithPet(ctx context.Context, _ *connect.Request[petv1.PlayWithPetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.Play(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) ApplyGameResult(ctx context.Context, request *connect.Request[petv1.ApplyGameResultRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.ApplyGameResult(
		ctx,
		userID,
		int(request.Msg.GetScore()),
		int(request.Msg.GetStarsCollected()),
		request.Msg.GetRoundId(),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) PutPetToSleep(ctx context.Context, _ *connect.Request[petv1.PutPetToSleepRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.PutToSleep(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) WakePet(ctx context.Context, _ *connect.Request[petv1.WakePetRequest]) (*connect.Response[petv1.PetStateResponse], error) {
	userID, err := service.studentID(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := service.backend.Wake(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(profile.toProto()), nil
}

func (service *connectService) WatchPetState(ctx context.Context, _ *connect.Request[petv1.WatchPetStateRequest], stream *connect.ServerStream[petv1.PetStateResponse]) error {
	userID, err := service.studentID(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		profile, err := service.backend.LoadProfile(ctx, userID)
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

func (service *connectService) studentID(ctx context.Context) (int64, error) {
	userID, err := service.requireStudent(ctx)
	if err != nil {
		return 0, connect.NewError(connect.CodePermissionDenied, err)
	}
	return userID, nil
}

func (profile Profile) toProto() *petv1.PetStateResponse {
	return &petv1.PetStateResponse{
		UserId:      profile.ID,
		DisplayName: profile.DisplayName,
		Cookies:     int32(profile.Cookies),
		StarBalance: int32(profile.StarBalance),
		PetState: &petv1.PetState{
			Hunger:      int32(profile.PetState.Hunger),
			Happiness:   int32(profile.PetState.Happiness),
			Energy:      int32(profile.PetState.Energy),
			Sleeping:    profile.PetState.Sleeping,
			Mood:        ProtoMood(profile.PetState.Mood),
			UpdatedAt:   timestamppb.New(profile.PetState.UpdatedAt),
			LastDecayAt: timestamppb.New(profile.PetState.LastDecayAt),
		},
	}
}
