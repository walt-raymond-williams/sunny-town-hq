package server

import (
	"testing"
	"time"
)

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
		t.Fatal("node should stay active without owned pickaxe")
	}
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected resource event: %#v", event)
	default:
	}
	if message := <-client.send; message.Type != "hello" {
		t.Fatalf("first message type = %q, want hello", message.Type)
	}
	if message := <-client.send; message.Type != "error" || message.Code != "tool_not_available" {
		t.Fatalf("tool error = %#v, want tool_not_available", message)
	}
}

func TestMiningUsesOwnedPickaxeWithoutEquipmentSlot(t *testing.T) {
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
	select {
	case event := <-room.resourceEvents:
		t.Fatalf("unexpected resource event: %#v", event)
	default:
	}
}

func TestMiningRequiresPlayerInRange(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.joinWithInventory(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{}, inventorySnapshot{"pickaxe": 1})
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
	room.rewardRunID = "test-run"
	client := testClient(room, "42")
	room.joinWithInventory(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{}, inventorySnapshot{"pickaxe": 1})
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
		if event.eventID != "sunny-town-v1:test-run:rock-node-001:1:42" || event.nodeID != "rock-node-001" || event.amount < 1 {
			t.Fatalf("resource event = %#v", event)
		}
	default:
		t.Fatal("expected resource event")
	}
}

func TestMiningRequiresActiveNodeAndRespawns(t *testing.T) {
	room := testRoom(miningTestMap())
	client := testClient(room, "42")
	room.joinWithInventory(client, testClaims(42), equipmentSnapshot{}, studentPositionResponse{}, inventorySnapshot{"pickaxe": 1})
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
