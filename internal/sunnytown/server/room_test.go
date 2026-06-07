package server

import (
	"testing"
	"time"

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
