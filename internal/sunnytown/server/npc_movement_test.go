package server

import (
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
)

func TestScriptedNPCMovesAlongPath(t *testing.T) {
	room := testRoom(scriptedNPCTestMap())
	npc := room.liveNPCs[scriptedSmokeNPCKey]
	startX := npc.x
	startY := npc.y

	room.step(0.5, time.Now())

	if npc.x <= startX {
		t.Fatalf("npc x = %v, want movement to the right from %v", npc.x, startX)
	}
	if npc.y < startY {
		t.Fatalf("npc y = %v, want movement to stay on or below start y %v", npc.y, startY)
	}
	if npc.facing != "right" || !npc.moving {
		t.Fatalf("npc state facing=%q moving=%v, want right and moving", npc.facing, npc.moving)
	}
}

func TestScriptedNPCReachesTargetAndStops(t *testing.T) {
	room := testRoom(scriptedNPCTestMap())
	npc := room.liveNPCs[scriptedSmokeNPCKey]

	now := time.Now()
	room.step(2, now)

	if npc.x != 160 || npc.y != 64 {
		t.Fatalf("npc position = (%v,%v), want target (160,64)", npc.x, npc.y)
	}
	if npc.moving {
		t.Fatal("npc should stop after reaching scripted target")
	}
	if npc.route != nil {
		t.Fatalf("npc route = %#v, want cleared after arrival", npc.route)
	}
}

func TestScriptedNPCDoesNotMoveWhenNoPathExists(t *testing.T) {
	gameMap := scriptedNPCTestMap()
	gameMap.BlockedRects = []rect{{X: 96, Y: 0, Width: 32, Height: 256}}
	room := testRoom(gameMap)
	npc := room.liveNPCs[scriptedSmokeNPCKey]

	room.step(0.5, time.Now())

	if npc.x != 64 || npc.y != 64 {
		t.Fatalf("npc position = (%v,%v), want unchanged (64,64)", npc.x, npc.y)
	}
	if npc.moving {
		t.Fatal("npc should not move when route planning fails")
	}
}

func TestSnapshotIncludesMovedScriptedNPCPosition(t *testing.T) {
	room := testRoom(scriptedNPCTestMap())
	room.step(0.5, time.Now())

	room.mu.Lock()
	snapshots := room.npcSnapshotsLocked()
	room.mu.Unlock()

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].ID != scriptedSmokeNPCKey || snapshots[0].X <= 64 || snapshots[0].Moving != true {
		t.Fatalf("npc snapshot = %#v, want moved mayor snapshot", snapshots[0])
	}
}

func scriptedNPCTestMap() gameMap {
	return gameMap{
		ID:       defaultMapID,
		Name:     "Scripted NPC Test Town",
		TileSize: 32,
		Width:    8,
		Height:   8,
		Spawns:   []point{{X: 64, Y: 96}},
		NPCs: []npc{{
			ID:        scriptedSmokeNPCKey,
			Name:      "Mayor Sunny",
			X:         64,
			Y:         64,
			Facing:    "down",
			SpriteKey: "mayor",
			Dialogue:  []string{"Hello."},
		}},
		Locations: []stmaps.Location{{
			ID:     scriptedSmokeLocationID,
			Name:   "Town Square Center",
			X:      160,
			Y:      64,
			Radius: 32,
			Tags:   []string{"public", "social", "idle"},
		}},
	}
}
