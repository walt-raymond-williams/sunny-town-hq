package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	"hq/internal/sunnytownauth"
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

func TestRoomPickupEnqueuesRewardFromServerOverlap(t *testing.T) {
	gameMap := testMap()
	gameMap.StarSpawns = []point{{X: 100, Y: 100}}
	room := testRoom(gameMap)
	room.rewardRunID = "test-run"
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

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
	room := testRoom(gameMap)
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	room.step(0, time.Now())

	select {
	case event := <-room.rewardEvents:
		t.Fatalf("unexpected reward event: %#v", event)
	default:
	}
}

func TestWorldTransfersPlayerThroughPortal(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	client := testClient(nil, "42")
	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	source := world.rooms[defaultMapID]
	source.updateMove("42", 7, 100, 100, "down", true, time.Now())

	if client.currentRoom().gameMap.ID != "test-house" {
		t.Fatalf("current map = %q, want test-house", client.currentRoom().gameMap.ID)
	}
	if _, ok := source.players["42"]; ok {
		t.Fatal("player should leave source room")
	}
	if _, ok := world.rooms["test-house"].players["42"]; !ok {
		t.Fatal("player should enter target room")
	}

	select {
	case message := <-client.send:
		if message.Type != "hello" {
			t.Fatalf("first message type = %q, want hello", message.Type)
		}
	default:
		t.Fatal("expected hello message")
	}
	select {
	case message := <-client.send:
		if message.Type != "map_changed" || message.MapID != "test-house" || message.Map == nil {
			t.Fatalf("transition message = %#v", message)
		}
		if len(message.Map.NPCs) != 1 || message.Map.NPCs[0].ID != "indoor-npc" {
			t.Fatalf("transition npcs = %#v, want indoor-npc", message.Map.NPCs)
		}
	default:
		t.Fatal("expected map_changed message")
	}
}

func TestWorldTransfersPlayerToForestCrossing(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorld(defaultRoomID, maps)
	client := testClient(nil, "42")
	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	world.rooms[defaultMapID].updateMove("42", 7, 1224, 480, "right", true, time.Now())

	if client.currentRoom().gameMap.ID != "forest-crossing-v1" {
		t.Fatalf("current map = %q, want forest-crossing-v1", client.currentRoom().gameMap.ID)
	}
}

func TestWorldDoesNotBouncePlayerFromTargetPortal(t *testing.T) {
	town := testMap()
	town.Portals = []portal{{
		ID:           "to-forest",
		X:            100,
		Y:            100,
		Width:        32,
		Height:       32,
		TargetMapID:  "forest",
		TargetX:      100,
		TargetY:      100,
		TargetFacing: "right",
	}}
	forest := gameMap{
		ID:       "forest",
		Name:     "Forest",
		TileSize: 32,
		Width:    10,
		Height:   10,
		Spawns:   []point{{X: 200, Y: 200}},
		Portals: []portal{{
			ID:           "to-town",
			X:            84,
			Y:            84,
			Width:        64,
			Height:       64,
			TargetMapID:  defaultMapID,
			TargetX:      160,
			TargetY:      160,
			TargetFacing: "left",
		}},
	}
	world := testWorld(town, forest)
	client := testClient(nil, "42")
	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	now := time.Now()
	world.rooms[defaultMapID].updateMove("42", 7, 100, 100, "right", true, now)
	if client.currentRoom().gameMap.ID != "forest" {
		t.Fatalf("current map after first transfer = %q, want forest", client.currentRoom().gameMap.ID)
	}

	world.rooms["forest"].updateMove("42", 8, 101, 100, "right", true, now.Add(50*time.Millisecond))
	if client.currentRoom().gameMap.ID != "forest" {
		t.Fatalf("current map after moving inside target portal = %q, want forest", client.currentRoom().gameMap.ID)
	}

	world.rooms["forest"].updateMove("42", 9, 200, 200, "right", true, now.Add(100*time.Millisecond))
	world.rooms["forest"].updateMove("42", 10, 100, 100, "left", true, now.Add(150*time.Millisecond))
	if client.currentRoom().gameMap.ID != defaultMapID {
		t.Fatalf("current map after leaving and re-entering portal = %q, want %s", client.currentRoom().gameMap.ID, defaultMapID)
	}
}

func TestRoomResourceNodeSnapshots(t *testing.T) {
	gameMap := testMap()
	gameMap.ResourceNodes = []resourceNodeDefinition{{
		ID:                "rock-node-001",
		Kind:              "rock",
		X:                 140,
		Y:                 160,
		Radius:            24,
		InteractionRadius: 48,
		RespawnSeconds:    15,
	}}
	room := testRoom(gameMap)

	snapshots := room.resourceNodeSnapshotsLocked()
	if len(snapshots) != 1 || snapshots[0].ID != "rock-node-001" || !snapshots[0].Active {
		t.Fatalf("resource snapshots = %#v, want active rock-node-001", snapshots)
	}
}

func TestRoomTargetsNearestBreakableWorldObject(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	room.addPlacedObjectLocked(placedObjectFromResponse(room.gameMap, mapObjectResponse{
		ID:                1,
		MapID:             room.gameMap.ID,
		GridX:             2,
		GridY:             2,
		ItemKey:           "stone_block",
		PlacedByAppUserID: 42,
	}))

	player := room.players["42"]
	player.x = 80
	player.y = 80

	target := room.nearestBreakableWorldObjectLocked(player, "pickaxe")
	if target == nil || target.source != worldObjectSourcePlaced || target.itemKey != "stone_block" {
		t.Fatalf("nearest target = %#v, want placed stone block", target)
	}

	player.x = 140
	player.y = 160

	target = room.nearestBreakableWorldObjectLocked(player, "pickaxe")
	if target == nil || target.source != worldObjectSourceNatural || target.resourceKind != "rock" {
		t.Fatalf("nearest target = %#v, want natural rock", target)
	}
}

func TestMiningRequiresOwnedPickaxe(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	room.players["42"].x = 140
	room.players["42"].y = 160

	client.handleToolUse(clientMessage{Type: "tool_use", ToolKey: "pickaxe"})

	if !room.resourceNodes["rock-node-001"].active {
		t.Fatal("node should stay active without equipped pickaxe")
	}
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected resource event: %#v", event)
	default:
	}
}

func TestMiningAllowsOwnedPickaxeWithoutEquipmentSlot(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	room.players["42"].inventory["pickaxe"] = 1
	room.players["42"].x = 140
	room.players["42"].y = 160

	client.handleToolUse(clientMessage{Type: "tool_use", ToolKey: "pickaxe"})

	if room.resourceNodes["rock-node-001"].hitCount != 1 {
		t.Fatalf("hitCount = %d, want 1", room.resourceNodes["rock-node-001"].hitCount)
	}
}

func TestMiningRequiresPlayerInRange(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{equipmentSlotTool: "pickaxe"}, studentPositionResponse{})
	room.players["42"].x = 260
	room.players["42"].y = 260

	client.handleToolUse(clientMessage{Type: "tool_use", ToolKey: "pickaxe"})

	if !room.resourceNodes["rock-node-001"].active {
		t.Fatal("node should stay active when player is too far away")
	}
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected resource event: %#v", event)
	default:
	}
}

func TestMiningRequiresThreeSwings(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{equipmentSlotTool: "pickaxe"}, studentPositionResponse{})
	room.players["42"].x = 140
	room.players["42"].y = 160

	node := room.resourceNodes["rock-node-001"]
	swingPickaxe(room, client)
	if !node.active || node.hitCount != 1 || node.harvestSeq != 0 {
		t.Fatalf("node after first swing = %#v, want active with 1 hit", node)
	}
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected event after first swing: %#v", event)
	default:
	}

	swingPickaxe(room, client)
	if !node.active || node.hitCount != 2 || node.harvestSeq != 0 {
		t.Fatalf("node after second swing = %#v, want active with 2 hits", node)
	}
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected event after second swing: %#v", event)
	default:
	}

	swingPickaxe(room, client)
	if node.active || node.harvestSeq != 1 || node.respawnAt.IsZero() {
		t.Fatalf("node after mining = %#v, want inactive harvest seq 1 with respawn", node)
	}
	select {
	case event := <-room.resourceEvents:
		if event.eventID != "sunny-town-v1:rock-node-001:1:42" || event.nodeID != "rock-node-001" || event.amount < 1 {
			t.Fatalf("resource event = %#v", event)
		}
	default:
		t.Fatal("expected resource event")
	}
}

func TestMiningRequiresActiveNodeAndRespawns(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{equipmentSlotTool: "pickaxe"}, studentPositionResponse{})
	room.players["42"].x = 140
	room.players["42"].y = 160

	now := time.Now()
	node := room.resourceNodes["rock-node-001"]
	node.active = false
	node.respawnAt = now.Add(time.Second)
	client.handleToolUse(clientMessage{Type: "tool_use", ToolKey: "pickaxe"})
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected resource event while inactive: %#v", event)
	default:
	}

	room.step(0, now.Add(2*time.Second))
	if !node.active {
		t.Fatal("node should respawn after cooldown")
	}
}

func TestWorldJoinUsesClaimMap(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	client := testClient(nil, "42")
	claims := testClaims(42)
	claims.MapID = "test-house"

	world.join(client, claims, equipmentSnapshot{}, studentPositionResponse{
		Found:     true,
		AppUserID: 42,
		RoomID:    defaultRoomID,
		MapID:     "test-house",
		X:         180,
		Y:         190,
		Facing:    "up",
	})

	if client.currentRoom().gameMap.ID != "test-house" {
		t.Fatalf("current map = %q, want test-house", client.currentRoom().gameMap.ID)
	}
	player := world.rooms["test-house"].players["42"]
	if player == nil || player.x != 180 || player.y != 190 || player.facing != "up" {
		t.Fatalf("joined player = %#v, want saved indoor position", player)
	}
}

func TestJoinHelloIncludesMapNPCs(t *testing.T) {
	room := testRoom(testMap())
	client := testClient(room, "42")
	room.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	select {
	case message := <-client.send:
		if message.Type != "hello" || message.Map == nil {
			t.Fatalf("hello message = %#v", message)
		}
		if len(message.Map.NPCs) != 1 || message.Map.NPCs[0].ID != "guide" {
			t.Fatalf("hello npcs = %#v, want guide", message.Map.NPCs)
		}
	default:
		t.Fatal("expected hello message")
	}
}

func TestWorldSnapshotsStayWithinCurrentCell(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	firstClient := testClient(nil, "42")
	secondClient := testClient(nil, "43")
	world.join(firstClient, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	world.join(secondClient, testClaims(43), equipmentSnapshot{}, studentPositionResponse{})

	world.rooms[defaultMapID].updateMove("42", 7, 100, 100, "down", true, time.Now())

	outdoorSnapshots := world.rooms[defaultMapID].snapshotsLocked()
	if len(outdoorSnapshots) != 1 || outdoorSnapshots[0].ID != "43" {
		t.Fatalf("outdoor snapshots = %#v, want only player 43", outdoorSnapshots)
	}

	indoorSnapshots := world.rooms["test-house"].snapshotsLocked()
	if len(indoorSnapshots) != 1 || indoorSnapshots[0].ID != "42" {
		t.Fatalf("indoor snapshots = %#v, want only player 42", indoorSnapshots)
	}
}

func TestMapNPCsStayScopedToCurrentCell(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	client := testClient(nil, "42")
	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	select {
	case message := <-client.send:
		if message.Map == nil || len(message.Map.NPCs) != 1 || message.Map.NPCs[0].ID != "outdoor-npc" {
			t.Fatalf("outdoor hello npcs = %#v, want outdoor-npc", message.Map)
		}
	default:
		t.Fatal("expected hello message")
	}

	world.rooms[defaultMapID].updateMove("42", 7, 100, 100, "down", true, time.Now())

	select {
	case message := <-client.send:
		if message.Type != "map_changed" || message.Map == nil {
			t.Fatalf("transition message = %#v", message)
		}
		if len(message.Map.NPCs) != 1 || message.Map.NPCs[0].ID != "indoor-npc" {
			t.Fatalf("indoor map npcs = %#v, want indoor-npc", message.Map.NPCs)
		}
	default:
		t.Fatal("expected map_changed message")
	}
}

func TestWorldPlayersShareInteriorCell(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	firstClient := testClient(nil, "42")
	secondClient := testClient(nil, "43")
	world.join(firstClient, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	world.join(secondClient, testClaims(43), equipmentSnapshot{}, studentPositionResponse{})

	now := time.Now()
	world.rooms[defaultMapID].updateMove("42", 7, 100, 100, "down", true, now)
	world.rooms[defaultMapID].updateMove("43", 7, 100, 100, "down", true, now)

	indoorSnapshots := world.rooms["test-house"].snapshotsLocked()
	if len(indoorSnapshots) != 2 {
		t.Fatalf("indoor snapshot count = %d, want 2", len(indoorSnapshots))
	}
}

func TestIndoorMapDoesNotCollectStars(t *testing.T) {
	world := testWorld(outdoorTestMap(), indoorTestMap())
	client := testClient(nil, "42")
	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})
	world.rooms[defaultMapID].updateMove("42", 7, 100, 100, "down", true, time.Now())

	world.rooms["test-house"].step(0, time.Now())

	select {
	case event := <-world.rewardEvents:
		t.Fatalf("unexpected indoor reward event: %#v", event)
	default:
	}
}

func TestValidateJoinTargetRejectsUnknownRoomOrMap(t *testing.T) {
	world := testWorld()
	claims := testClaims(42)
	claims.RoomID = "other-room"
	if err := validateJoinTarget(claims, defaultRoomID, world.rooms); err == nil {
		t.Fatal("expected unknown room to be rejected")
	}

	claims = testClaims(42)
	claims.MapID = "other-map"
	if err := validateJoinTarget(claims, defaultRoomID, world.rooms); err == nil {
		t.Fatal("expected unknown map to be rejected")
	}
}

func TestLoadMapsRejectsDuplicateMapIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"duplicate","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate map ID to be rejected")
	}
}

func TestLoadMapsRejectsUnknownPortalTarget(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"portals":[{"id":"door","x":64,"y":64,"width":32,"height":32,"targetMapId":"missing","targetX":64,"targetY":64,"targetFacing":"down"}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected unknown portal target to be rejected")
	}
}

func TestLoadMapsAcceptsCheckedInMaps(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}

	classroom, ok := maps["sunny-town-classroom"]
	if !ok {
		t.Fatal("expected sunny-town-classroom map to load")
	}
	if len(classroom.NPCs) != 1 || classroom.NPCs[0].Activity == nil || classroom.NPCs[0].Activity.Type != "schoolwork" {
		t.Fatalf("classroom npcs = %#v, want teacher schoolwork npc", classroom.NPCs)
	}
	forest, ok := maps["forest-crossing-v1"]
	if !ok {
		t.Fatal("expected forest-crossing-v1 map to load")
	}
	if len(forest.ResourceNodes) < 2 {
		t.Fatalf("forest resource nodes = %#v, want at least two nodes", forest.ResourceNodes)
	}
}

func TestLoadMapsRejectsDuplicateResourceNodeIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"resourceNodes":[{"id":"rock-node","kind":"rock","x":64,"y":64,"radius":24,"interactionRadius":48,"respawnSeconds":15},{"id":"rock-node","kind":"rock","x":96,"y":64,"radius":24,"interactionRadius":48,"respawnSeconds":15}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate resource node ID to be rejected")
	}
}

func TestLoadMapsRejectsInvalidResourceNodes(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"resourceNodes":[{"id":"rock-node","kind":"rock","x":64,"y":64,"radius":0,"interactionRadius":48,"respawnSeconds":15}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected invalid resource node to be rejected")
	}
}

func TestLoadMapsAcceptsNPCDefinitions(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"activity":{"type":"schoolwork"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	maps, err := stmaps.LoadMaps(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(maps["one"].NPCs) != 1 || maps["one"].NPCs[0].ID != "guide" {
		t.Fatalf("loaded npcs = %#v, want guide", maps["one"].NPCs)
	}
	if maps["one"].NPCs[0].Activity == nil || maps["one"].NPCs[0].Activity.Type != "schoolwork" {
		t.Fatalf("loaded npc activity = %#v, want schoolwork", maps["one"].NPCs[0].Activity)
	}
}

func TestLoadMapsRejectsDuplicateNPCIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."]},{"id":"guide","name":"Guide Again","x":96,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hi."]}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate npc ID to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."]}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCActivity(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"activity":{"type":"unknown"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc activity to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCShop(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"shop":{"id":"guide-shop","items":[{"itemKey":"cookie","name":"Cookie","description":"A treat.","priceStars":0}]}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := stmaps.LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc shop to be rejected")
	}
}

func testClient(room *room, id string) *client {
	client := &client{
		send: make(chan serverMessage, 4),
		id:   id,
	}
	client.setRoom(room)
	return client
}

func testRoom(gameMap gameMap) *room {
	world := testWorld(gameMap)
	return world.rooms[gameMap.ID]
}

func testWorld(maps ...gameMap) *world {
	byID := map[string]gameMap{}
	for _, gameMap := range maps {
		byID[gameMap.ID] = gameMap
	}
	if _, ok := byID[defaultMapID]; !ok {
		byID[defaultMapID] = testMap()
	}
	return newWorld(defaultRoomID, byID)
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
		NPCs: []npc{{
			ID:        "guide",
			Name:      "Guide",
			X:         160,
			Y:         160,
			Facing:    "down",
			SpriteKey: "guide",
			Dialogue:  []string{"Hello."},
		}},
	}
}

func outdoorTestMap() gameMap {
	gameMap := testMap()
	gameMap.Portals = []portal{{
		ID:           "test-door",
		X:            86,
		Y:            86,
		Width:        32,
		Height:       32,
		TargetMapID:  "test-house",
		TargetX:      160,
		TargetY:      160,
		TargetFacing: "up",
	}}
	gameMap.NPCs = []npc{{
		ID:        "outdoor-npc",
		Name:      "Outdoor NPC",
		X:         140,
		Y:         140,
		Facing:    "down",
		SpriteKey: "guide",
		Dialogue:  []string{"Outside."},
	}}
	return gameMap
}

func indoorTestMap() gameMap {
	return gameMap{
		ID:       "test-house",
		Name:     "Test House",
		TileSize: 32,
		Width:    10,
		Height:   10,
		Spawns:   []point{{X: 160, Y: 160}},
		NPCs: []npc{{
			ID:        "indoor-npc",
			Name:      "Indoor NPC",
			X:         200,
			Y:         200,
			Facing:    "down",
			SpriteKey: "keeper",
			Dialogue:  []string{"Inside."},
			Shop: &shop{
				ID: "cookie-keeper-shop",
				Items: []shopItem{{
					ItemKey:     "cookie",
					Name:        "Cookie",
					Description: "A treat.",
					PriceStars:  50,
				}},
			},
		}},
	}
}

func miningTestMap() gameMap {
	gameMap := testMap()
	gameMap.ResourceNodes = []resourceNodeDefinition{{
		ID:                "rock-node-001",
		Kind:              "rock",
		X:                 140,
		Y:                 160,
		Radius:            24,
		InteractionRadius: 48,
		RespawnSeconds:    1,
	}}
	return gameMap
}

func swingPickaxe(room *room, client *client) {
	if player := room.players[client.id]; player != nil {
		player.lastToolUseAt = time.Now().Add(-resourceToolCooldown)
	}
	client.handleToolUse(clientMessage{Type: "tool_use", ToolKey: "pickaxe"})
}
