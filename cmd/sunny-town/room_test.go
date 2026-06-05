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

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 7, 120, 100, "right", true, now)

	if player.x <= 100 {
		t.Fatalf("player x = %v, want movement to the right", player.x)
	}
	if player.facing != "right" || !player.moving {
		t.Fatalf("player state facing=%q moving=%v, want right and moving", player.facing, player.moving)
	}
}

func TestRoomAcceptsClientPositionInsideBlockedGeometry(t *testing.T) {
	gameMap := testMap()
	gameMap.BlockedRects = []rect{{X: 130, Y: 80, Width: 60, Height: 60}}
	room := newRoom(defaultRoomID, gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 7, 150, 100, "right", true, now)

	if player.x != 150 || player.y != 100 {
		t.Fatalf("player position = (%v,%v), want accepted client position (150,100)", player.x, player.y)
	}
}

func TestRoomClampsBounds(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	player := room.players["42"]
	player.x = 20
	now := time.Now()
	player.lastMoveAt = now.Add(-time.Second)
	room.updateMove("42", 7, -100, player.y, "left", true, now)

	if player.x < playerSize/2 {
		t.Fatalf("player x = %v, want clamped to map bounds", player.x)
	}
}

func TestRoomAcceptsLargeClientMoveSamples(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-50 * time.Millisecond)
	room.updateMove("42", 12, 300, 100, "right", true, now)

	if player.x != 300 {
		t.Fatalf("player x = %v, want accepted client position 300", player.x)
	}
}

func TestRoomSnapshotIncludesLastProcessedSeq(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	now := time.Now()
	room.players["42"].lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 12, 100, 120, "down", true, now)
	snapshots := room.snapshotsLocked()

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].LastProcessedSeq != 12 {
		t.Fatalf("LastProcessedSeq = %d, want 12", snapshots[0].LastProcessedSeq)
	}
}

func TestRoomIgnoresOutOfOrderMove(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	now := time.Now()
	room.players["42"].lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 12, 120, 100, "right", true, now)
	room.updateMove("42", 11, 80, 100, "left", true, now.Add(50*time.Millisecond))

	if room.players["42"].lastMoveSeq != 12 {
		t.Fatalf("lastMoveSeq = %d, want 12", room.players["42"].lastMoveSeq)
	}
	if room.players["42"].facing != "right" {
		t.Fatalf("facing = %q, want right", room.players["42"].facing)
	}
}

func TestRoomDoesNotDriftWithoutNewMoveSamples(t *testing.T) {
	room := newRoom(defaultRoomID, testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	start := time.Now()
	player := room.players["42"]
	player.lastMoveAt = start.Add(-200 * time.Millisecond)
	room.updateMove("42", 12, 120, 100, "right", true, start)
	x := player.x
	y := player.y

	room.step(0.05, start.Add(50*time.Millisecond))
	if player.x != x || player.y != y {
		t.Fatalf("player moved without a new sample from (%v,%v) to (%v,%v)", x, y, player.x, player.y)
	}
	room.step(0.05, start.Add(movingStateTTL+time.Millisecond))
	if player.moving {
		t.Fatal("player should stop showing movement after stale samples")
	}
}

func TestRoomPickupEnqueuesRewardFromServerOverlap(t *testing.T) {
	gameMap := testMap()
	gameMap.StarSpawns = []point{{X: 100, Y: 100}}
	room := newRoom(defaultRoomID, gameMap)
	room.rewardRunID = "test-run"
	client := testClient(room, "42")
	room.join(client, testClaims(42))

	room.step(0, time.Now())

	select {
	case event := <-room.rewardEvents:
		if event.eventID != "sunny-town-main:test-run:star-0001:1:42" {
			t.Fatalf("eventID = %q, want sunny-town-main:test-run:star-0001:1:42", event.eventID)
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
