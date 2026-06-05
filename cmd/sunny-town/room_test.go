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
	room.step(0.2)

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
	room.step(0.2)

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
	room.step(1)

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
	if snapshots[0].LastProcessedSeq != 12 {
		t.Fatalf("LastProcessedSeq = %d, want 12", snapshots[0].LastProcessedSeq)
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
