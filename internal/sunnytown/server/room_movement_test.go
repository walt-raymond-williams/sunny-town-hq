package server

import (
	"testing"
	"time"
)

func TestRoomJoinAndLeave(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")

	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	if len(room.players) != 1 {
		t.Fatalf("player count after join = %d, want 1", len(room.players))
	}

	room.leave(client)
	if len(room.players) != 0 {
		t.Fatalf("player count after leave = %d, want 0", len(room.players))
	}
}

func TestRoomJoinUsesSavedPosition(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")

	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{
		Found:     true,
		AppUserID: 42,
		RoomID:    defaultRoomID,
		MapID:     defaultMapID,
		X:         220,
		Y:         230,
		Facing:    "left",
	})

	player := room.players["42"]
	if player.x != 220 || player.y != 230 || player.facing != "left" {
		t.Fatalf("player spawn = (%v,%v,%s), want saved position (220,230,left)", player.x, player.y, player.facing)
	}
}

func TestRoomMovement(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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

func TestRoomRejectsClientPositionInsideBlockedGeometry(t *testing.T) {
	gameMap := testMap()
	gameMap.BlockedRects = []rect{{X: 130, Y: 80, Width: 60, Height: 60}}
	room := testRoom(gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 7, 150, 100, "right", true, now)

	if player.x != 100 || player.y != 100 {
		t.Fatalf("player position = (%v,%v), want previous position (100,100)", player.x, player.y)
	}
}

func TestRoomRejectsClientPositionInsidePlacedObject(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	room.addPlacedObjectLocked(placedObjectFromResponse(room.gameMap, mapObjectResponse{
		ID:                1,
		MapID:             room.gameMap.ID,
		GridX:             4,
		GridY:             3,
		ItemKey:           "stone_block",
		PlacedByAppUserID: 42,
	}))

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 7, 144, 112, "right", true, now)

	if player.x != 100 || player.y != 100 {
		t.Fatalf("player position = (%v,%v), want previous position (100,100)", player.x, player.y)
	}
}

func TestRoomRejectsClientPositionInsideActiveResourceNode(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 7, 140, 160, "right", true, now)

	if player.x != 100 || player.y != 100 {
		t.Fatalf("player position = (%v,%v), want previous position (100,100)", player.x, player.y)
	}

	room.resourceNodes["rock-node-001"].active = false
	player.lastMoveAt = now.Add(-200 * time.Millisecond)
	room.updateMove("42", 8, 140, 160, "right", true, now.Add(time.Second))

	if player.x != 140 || player.y != 160 {
		t.Fatalf("player position = (%v,%v), want movement through depleted node", player.x, player.y)
	}
}

func TestRoomCanPlaceObjectRejectsBlockedAndOccupiedTiles(t *testing.T) {
	gameMap := testMap()
	gameMap.BlockedRects = []rect{{X: 192, Y: 96, Width: 32, Height: 32}}
	room := testRoom(gameMap)

	if !room.canPlaceObjectLocked(2, 2, "stone_block") {
		t.Fatal("expected empty tile to allow stone block placement")
	}
	if room.canPlaceObjectLocked(6, 3, "stone_block") {
		t.Fatal("expected blocked tile to reject stone block placement")
	}

	room.addPlacedObjectLocked(placedObjectFromResponse(room.gameMap, mapObjectResponse{
		ID:                1,
		MapID:             room.gameMap.ID,
		GridX:             2,
		GridY:             2,
		ItemKey:           "stone_block",
		PlacedByAppUserID: 42,
	}))
	if room.canPlaceObjectLocked(2, 2, "stone_block") {
		t.Fatal("expected occupied tile to reject stone block placement")
	}
}

func TestRoomClampsBounds(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	now := time.Now()
	player := room.players["42"]
	player.lastMoveAt = now.Add(-50 * time.Millisecond)
	room.updateMove("42", 12, 300, 100, "right", true, now)

	if player.x != 300 {
		t.Fatalf("player x = %v, want accepted client position 300", player.x)
	}
}

func TestRoomSnapshotIncludesLastProcessedSeq(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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
