package server

import (
	"path/filepath"
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	"hq/internal/sunnytownauth"
)

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
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
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
