package sunnytownbridge

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestIsFacing(t *testing.T) {
	for _, value := range []string{"up", "down", "left", "right"} {
		if !IsFacing(value) {
			t.Fatalf("IsFacing(%q) = false, want true", value)
		}
	}

	if IsFacing("north") {
		t.Fatal("IsFacing(\"north\") = true, want false")
	}
}

func TestMapObjectErrorMapping(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{name: "not found", err: ErrMapObjectNotFound, status: http.StatusNotFound, message: "map object not found"},
		{name: "occupied", err: ErrMapObjectOccupied, status: http.StatusConflict, message: "location is occupied"},
		{name: "not owned", err: ErrMapObjectNotOwned, status: http.StatusConflict, message: "item is not in inventory"},
		{name: "unsupported", err: ErrUnsupportedMapObject, status: http.StatusBadRequest, message: "unsupported map object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatusForMapObjectError(tt.err); got != tt.status {
				t.Fatalf("status = %d, want %d", got, tt.status)
			}
			if got := MapObjectErrorMessage(tt.err); got != tt.message {
				t.Fatalf("message = %q, want %q", got, tt.message)
			}
		})
	}
}

func TestCommitRewardRejectsInvalidRequestBeforeDB(t *testing.T) {
	_, err := (Store{}).CommitReward(context.Background(), RewardEventRequest{
		EventID:    "event-1",
		AppUserID:  123,
		RoomID:     "sunny-town-main",
		MapID:      "sunny-town-v1",
		RewardKind: "star",
		Amount:     1,
	})
	if err == nil {
		t.Fatal("expected invalid reward request error")
	}
	if !strings.Contains(err.Error(), "missing required fields") {
		t.Fatalf("error = %v, want missing required fields", err)
	}
}

func TestCommitResourceRejectsInvalidRequestBeforeDB(t *testing.T) {
	_, err := (Store{}).CommitResource(context.Background(), ResourceEventRequest{
		EventID:     "event-1",
		AppUserID:   123,
		Source:      "other",
		RoomID:      "sunny-town-main",
		MapID:       "sunny-town-v1",
		NodeID:      "node-1",
		ResourceKey: "rock",
		Amount:      1,
	})
	if err == nil {
		t.Fatal("expected invalid resource request error")
	}
	if !strings.Contains(err.Error(), "unsupported resource event source") {
		t.Fatalf("error = %v, want unsupported source", err)
	}
}

func TestCommitNPCJobProductionRejectsInvalidRequestBeforeDB(t *testing.T) {
	_, err := (Store{}).CommitNPCJobProduction(context.Background(), NPCJobProductionRequest{
		EventID:     "event-1",
		CharacterID: 123,
		RoomID:      "sunny-town-main",
		MapID:       "sunny-town-house-1",
		NPCKey:      "cookie-keeper",
		JobKey:      "other",
		LocationID:  "cookie-keeper-counter",
		OutputKey:   "shop_stock_progress",
		Amount:      1,
	})
	if err == nil {
		t.Fatal("expected invalid npc job production request error")
	}
	if !strings.Contains(err.Error(), "unsupported npc job") {
		t.Fatalf("error = %v, want unsupported npc job", err)
	}
}

func TestLoadNPCJobProductionProgressRejectsInvalidRequestBeforeDB(t *testing.T) {
	_, err := (Store{}).LoadNPCJobProductionProgress(context.Background(), NPCJobProductionProgressRequest{
		RoomID: "sunny-town-main",
		JobKey: "other",
	})
	if err == nil {
		t.Fatal("expected invalid npc job production progress request error")
	}
	if !strings.Contains(err.Error(), "unsupported npc job") {
		t.Fatalf("error = %v, want unsupported npc job", err)
	}
}
