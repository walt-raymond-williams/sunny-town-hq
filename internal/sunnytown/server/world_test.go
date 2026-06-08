package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	stconfig "hq/internal/sunnytown/config"
	"hq/internal/sunnytown/hqclient"
	stmaps "hq/internal/sunnytown/maps"
)

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

func TestLoadNPCCharactersMergesDurableIdentity(t *testing.T) {
	hq := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/internal/sunny-town/npc-characters/ensure" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-HQ-Service-Secret") != "test-secret" {
			t.Fatalf("service secret = %q, want test-secret", r.Header.Get("X-HQ-Service-Secret"))
		}
		var request struct {
			RoomID string `json:"room_id"`
			NPCs   []struct {
				NPCKey      string `json:"npc_key"`
				DisplayName string `json:"display_name"`
				AvatarID    string `json:"avatar_id"`
			} `json:"npcs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.RoomID != defaultRoomID || len(request.NPCs) != 1 || request.NPCs[0].NPCKey != "guide" {
			t.Fatalf("request = %#v, want guide in default room", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"npcs":[{"character_id":777,"room_id":"sunny-town-main","npc_key":"guide","display_name":"Guide","avatar_id":"guide"}]}`))
	}))
	defer hq.Close()

	srv := New(stconfig.Config{
		HQInternalURL:  hq.URL,
		ServiceSecret:  "test-secret",
		AllowedOrigins: map[string]bool{},
	}, map[string]gameMap{defaultMapID: testMap()})

	if err := srv.LoadNPCCharacters(context.Background()); err != nil {
		t.Fatalf("LoadNPCCharacters() error = %v", err)
	}
	staticNPCs := srv.world.rooms[defaultMapID].gameMap.NPCs
	if len(staticNPCs) != 1 || staticNPCs[0].CharacterID != 0 {
		t.Fatalf("static npcs = %#v, want unchanged map definition without character id", staticNPCs)
	}

	room := srv.world.rooms[defaultMapID]
	room.mu.Lock()
	snapshot := room.mapSnapshotLocked()
	room.mu.Unlock()
	if len(snapshot.NPCs) != 1 || snapshot.NPCs[0].ID != "guide" || snapshot.NPCs[0].CharacterID != 777 {
		t.Fatalf("snapshot npcs = %#v, want guide with character id 777", snapshot.NPCs)
	}
}

func TestJoinHelloUsesNPCCharacterSnapshotWithoutMutatingMap(t *testing.T) {
	world := testWorld(testMap())
	world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 888,
		NPCKey:      "guide",
		DisplayName: "Durable Guide",
		AvatarID:    "durable-guide",
	}})
	client := testClient(nil, "42")

	world.join(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{})

	select {
	case message := <-client.send:
		if message.Type != "hello" || message.Map == nil || len(message.Map.NPCs) != 1 {
			t.Fatalf("hello message = %#v", message)
		}
		npc := message.Map.NPCs[0]
		if npc.ID != "guide" || npc.CharacterID != 888 || npc.Name != "Durable Guide" || npc.SpriteKey != "durable-guide" {
			t.Fatalf("hello npc = %#v, want durable identity overlay", npc)
		}
	default:
		t.Fatal("expected hello message")
	}
	staticNPC := world.rooms[defaultMapID].gameMap.NPCs[0]
	if staticNPC.CharacterID != 0 || staticNPC.Name != "Guide" || staticNPC.SpriteKey != "guide" {
		t.Fatalf("static npc = %#v, want original map data unchanged", staticNPC)
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
