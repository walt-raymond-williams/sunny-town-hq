package server

import (
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
)

func TestDrivenNPCMovesAlongPath(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
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

func TestDrivenNPCReachesTargetAndStops(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]

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

func TestDrivenNPCDoesNotMoveWhenNoPathExists(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.BlockedRects = []rect{{X: 96, Y: 0, Width: 32, Height: 256}}
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]

	room.step(0.5, time.Now())

	if npc.x != 64 || npc.y != 64 {
		t.Fatalf("npc position = (%v,%v), want unchanged (64,64)", npc.x, npc.y)
	}
	if npc.moving {
		t.Fatal("npc should not move when route planning fails")
	}
}

func TestSnapshotIncludesMovedDrivenNPCPosition(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	room.step(0.5, time.Now())

	room.mu.Lock()
	snapshots := room.npcSnapshotsLocked()
	room.mu.Unlock()

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].ID != driveControlledNPCKey || snapshots[0].X <= 64 || snapshots[0].Moving != true {
		t.Fatalf("npc snapshot = %#v, want moved mayor snapshot", snapshots[0])
	}
}

func TestNPCDrivesDeplete(t *testing.T) {
	room := testRoom(testMap())
	npc := room.liveNPCs["guide"]
	start := npc.drives.Hunger

	room.step(1, time.Now())

	if npc.drives.Hunger >= start {
		t.Fatalf("hunger = %v, want less than %v", npc.drives.Hunger, start)
	}
}

func TestMatchingLocationReplenishesDrive(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.NPCs[0].X = 160
	gameMap.NPCs[0].Y = 64
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Social = 40

	room.step(1, time.Now())

	if npc.drives.Social <= 40 {
		t.Fatalf("social = %v, want replenished above 40", npc.drives.Social)
	}
}

func TestNonMatchingLocationDoesNotReplenishDrive(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.NPCs[0].X = 160
	gameMap.NPCs[0].Y = 64
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 40

	room.step(1, time.Now())

	if npc.drives.Hunger >= 40 {
		t.Fatalf("hunger = %v, want depletion without social-location replenishment", npc.drives.Hunger)
	}
}

func TestLowestSatisfiableDriveIsSelected(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations, stmaps.Location{
		ID:     "snack-stand",
		Name:   "Snack Stand",
		X:      224,
		Y:      64,
		Radius: 32,
		Tags:   []string{"food"},
	})
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 40
	npc.drives.Social = 30

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveSocial {
		t.Fatalf("active drive = %q, want social", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want town square social location", npc.goal)
	}
}

func driveNPCTestMap() gameMap {
	return gameMap{
		ID:       defaultMapID,
		Name:     "Drive NPC Test Town",
		TileSize: 32,
		Width:    8,
		Height:   8,
		Spawns:   []point{{X: 64, Y: 96}},
		NPCs: []npc{{
			ID:        driveControlledNPCKey,
			Name:      "Mayor Sunny",
			X:         64,
			Y:         64,
			Facing:    "down",
			SpriteKey: "mayor",
			Dialogue:  []string{"Hello."},
		}},
		Locations: []stmaps.Location{{
			ID:     driveStartLocationID,
			Name:   "Town Square Center",
			X:      160,
			Y:      64,
			Radius: 32,
			Tags:   []string{"public", "social", "idle"},
		}},
	}
}
