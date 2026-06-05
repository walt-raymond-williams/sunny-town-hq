package main

import (
	"testing"
	"time"

	"hq/internal/sunnytownauth"
)

func TestRoomJoinAndLeave(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")

	room.join(client, testClaims(42))
	if len(room.players) != 1 {
		t.Fatalf("player count after join = %d, want 1", len(room.players))
	}

	room.leave(client)
	if len(room.players) != 0 {
		t.Fatalf("player count after leave = %d, want 0", len(room.players))
	}
}

func TestRoomMovement(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.updateInput("42", 7, inputState{Right: true})
	room.step(0.2, time.Now())

	player := room.players["42"]
	if player.x <= 100 {
		t.Fatalf("player x = %v, want movement to the right", player.x)
	}
	if player.facing != "right" || !player.moving {
		t.Fatalf("player state facing=%q moving=%v, want right and moving", player.facing, player.moving)
	}
}

func TestRoomBlocksCollision(t *testing.T) {
	gameMap := testMap()
	gameMap.BlockedRects = []rect{{X: 130, Y: 80, Width: 60, Height: 60}}
	room := newRoom(defaultRoomID, gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.updateInput("42", 7, inputState{Right: true})
	room.step(0.2, time.Now())

	player := room.players["42"]
	if player.x >= 130 {
		t.Fatalf("player x = %v, want collision to block movement before obstacle", player.x)
	}
}

func TestRoomClampsBounds(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	player := room.players["42"]
	player.x = 20
	room.updateInput("42", 7, inputState{Left: true})
	room.step(1, time.Now())

	if player.x < playerSize/2 {
		t.Fatalf("player x = %v, want clamped to map bounds", player.x)
	}
}

func TestRoomSnapshotIncludesLastProcessedSeq(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.updateInput("42", 12, inputState{Down: true})
	snapshots := room.snapshotsLocked()

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].LastProcessedSeq != 0 {
		t.Fatalf("LastProcessedSeq before simulation = %d, want 0", snapshots[0].LastProcessedSeq)
	}

	room.step(0.05, time.Now())
	snapshots = room.snapshotsLocked()
	if snapshots[0].LastProcessedSeq != 12 {
		t.Fatalf("LastProcessedSeq after simulation = %d, want 12", snapshots[0].LastProcessedSeq)
	}
}

func TestRoomSnapshotDoesNotAcknowledgeInputUntilNextSimulationStep(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.updateInput("42", 12, inputState{Right: true})
	room.step(0.05, time.Now())

	room.updateInput("42", 13, inputState{})
	snapshots := room.snapshotsLocked()
	if snapshots[0].LastProcessedSeq != 12 {
		t.Fatalf("LastProcessedSeq before next simulation = %d, want 12", snapshots[0].LastProcessedSeq)
	}

	room.step(0.05, time.Now())
	snapshots = room.snapshotsLocked()
	if snapshots[0].LastProcessedSeq != 13 {
		t.Fatalf("LastProcessedSeq after next simulation = %d, want 13", snapshots[0].LastProcessedSeq)
	}
}

func TestRoomPickupEnqueuesRewardFromServerOverlap(t *testing.T) {
	gameMap := testMap()
	gameMap.StarSpawns = []point{{X: 100, Y: 100}}
	room := newRoom(defaultRoomID, gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.step(0, time.Now())

	select {
	case event := <-room.rewardEvents:
		if event.eventID != "sunny-town-main:star-0001:1:42" {
			t.Fatalf("eventID = %q, want sunny-town-main:star-0001:1:42", event.eventID)
		}
		if event.appUserID != 42 || event.collectibleID != "star-0001" || event.kind != "star" {
			t.Fatalf("reward event = %#v", event)
		}
	default:
		t.Fatal("expected reward event")
	}

	if room.collectibles["star-0001"].active {
		t.Fatal("star should be inactive after pickup")
	}
}

func TestRoomPickupRequiresOverlap(t *testing.T) {
	gameMap := testMap()
	gameMap.StarSpawns = []point{{X: 250, Y: 250}}
	room := newRoom(defaultRoomID, gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.step(0, time.Now())

	select {
	case event := <-room.rewardEvents:
		t.Fatalf("unexpected reward event: %#v", event)
	default:
	}
}

func TestValidateJoinTargetRejectsUnknownRoomOrMap(t *testing.T) {
	claims := testClaims(42)
	claims.RoomID = "other-room"
	if err := validateJoinTarget(claims, defaultRoomID, defaultMapID); err == nil {
		t.Fatal("expected unknown room to be rejected")
	}

	claims = testClaims(42)
	claims.MapID = "other-map"
	if err := validateJoinTarget(claims, defaultRoomID, defaultMapID); err == nil {
		t.Fatal("expected unknown map to be rejected")
	}
}

func testClient(room *room, id string) *client {
	return &client{
		send: make(chan serverMessage, 4),
		room: room,
		id:   id,
	}
}

func testClaims(appUserID int64) sunnytownauth.Claims {
	return sunnytownauth.Claims{
		AppUserID:       appUserID,
		KeycloakSubject: "subject",
		DisplayName:     "Student",
		Roles:           []string{"student"},
		RoomID:          defaultRoomID,
		MapID:           defaultMapID,
		AvatarID:        "pet-default",
		ExpiresAt:       time.Now().Add(time.Minute).Unix(),
	}
}

func testMap() gameMap {
	return gameMap{
		ID:       defaultMapID,
		Name:     "Test Town",
		TileSize: 32,
		Width:    10,
		Height:   10,
		Spawns:   []point{{X: 100, Y: 100}},
	}
}
