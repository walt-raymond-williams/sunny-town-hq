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

func TestNPCFocusWindowPreventsPrematureSwitching(t *testing.T) {
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
	now := time.Now()

	room.step(0.1, now)
	if npc.activeDrive != npcDriveSocial {
		t.Fatalf("active drive = %q, want social", npc.activeDrive)
	}

	npc.drives.Hunger = 10
	room.step(0.1, now.Add(time.Second))

	if npc.activeDrive != npcDriveSocial {
		t.Fatalf("active drive = %q, want focus window to keep social", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want original social goal", npc.goal)
	}
}

func TestNPCEmergencyDriveInterruptsAfterReevaluation(t *testing.T) {
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
	now := time.Now()

	room.step(0.1, now)
	npc.drives.Hunger = 10
	room.step(0.1, now.Add(npcGoalFocusDuration+npcGoalReevaluateInterval))

	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want emergency hunger interrupt", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "snack-stand" {
		t.Fatalf("goal = %#v, want snack stand hunger goal", npc.goal)
	}
}

func TestUnreachableUrgentDriveFallsBackToNextSatisfiableDrive(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.BlockedRects = []rect{{X: 192, Y: 0, Width: 32, Height: 256}}
	gameMap.Locations = append(gameMap.Locations, stmaps.Location{
		ID:     "blocked-snack-stand",
		Name:   "Blocked Snack Stand",
		X:      240,
		Y:      64,
		Radius: 32,
		Tags:   []string{"food"},
	})
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 40

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveSocial {
		t.Fatalf("active drive = %q, want social fallback", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want social town square goal", npc.goal)
	}
}

func TestFailedTargetCooldownPreventsImmediateRetry(t *testing.T) {
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
	npc.drives.Hunger = 10
	npc.drives.Social = 40
	now := time.Now()
	npc.markTargetFailed(npcGoal{
		drive:    npcDriveHunger,
		mapID:    room.gameMap.ID,
		location: gameMap.Locations[1],
	}, now)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveSocial {
		t.Fatalf("active drive = %q, want social while hunger target is cooling down", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want social town square goal", npc.goal)
	}

	room.clearNPCGoal(npc)
	room.step(0.1, now.Add(npcFailedTargetCooldown+time.Second))
	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger after cooldown expires", npc.activeDrive)
	}
}

func TestUnreplenishingTargetIsMarkedFailed(t *testing.T) {
	gameMap := driveNPCTestMap()
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	location := gameMap.Locations[0]
	now := time.Now()
	npc.drives.Hunger = 10
	npc.goal = &npcGoal{
		drive:    npcDriveHunger,
		mapID:    room.gameMap.ID,
		location: location,
	}
	npc.x = location.X
	npc.y = location.Y
	npc.goalArrivedAt = now.Add(-npcGoalGraceDuration)
	npc.goalArriveDrive = npc.drives.Hunger

	room.step(0.1, now)

	if npc.goal != nil {
		t.Fatalf("goal = %#v, want failed goal cleared", npc.goal)
	}
	if npc.failureCount != 1 {
		t.Fatalf("failure count = %d, want 1", npc.failureCount)
	}
	if !npc.targetFailedRecently(npcGoal{drive: npcDriveHunger, mapID: room.gameMap.ID, location: location}, now) {
		t.Fatal("target should be on failed cooldown")
	}
}

func TestNPCSelectsCrossMapDriveGoal(t *testing.T) {
	town, house := driveNPCCrossMapTestMaps()
	world := testWorld(town, house)
	room := world.rooms[defaultMapID]
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.mapID != house.ID || npc.goal.location.ID != "house-kitchen" {
		t.Fatalf("goal = %#v, want house kitchen", npc.goal)
	}
	if npc.route == nil || len(npc.route.Steps) != 2 || npc.route.Steps[0].PortalID != "house-door" {
		t.Fatalf("route = %#v, want portal step plus target step", npc.route)
	}
}

func TestNPCTransfersAcrossPortalRoute(t *testing.T) {
	town, house := driveNPCCrossMapTestMaps()
	world := testWorld(town, house)
	source := world.rooms[defaultMapID]
	target := world.rooms[house.ID]
	npc := source.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95
	now := time.Now()

	source.step(2, now)

	if source.liveNPCs[driveControlledNPCKey] != nil {
		t.Fatal("npc should be removed from source room after portal transfer")
	}
	transferred := target.liveNPCs[driveControlledNPCKey]
	if transferred == nil {
		t.Fatal("npc should be added to target room after portal transfer")
	}
	if transferred.mapID != house.ID || transferred.x != 64 || transferred.y != 64 || transferred.facing != "up" {
		t.Fatalf("transferred npc map=%q pos=(%v,%v) facing=%q, want house (64,64) up", transferred.mapID, transferred.x, transferred.y, transferred.facing)
	}
	if transferred.route == nil || transferred.routeStep != 1 {
		t.Fatalf("transferred route step = %d route=%#v, want final route step", transferred.routeStep, transferred.route)
	}

	source.mu.Lock()
	sourceSnapshots := source.npcSnapshotsLocked()
	source.mu.Unlock()
	target.mu.Lock()
	targetSnapshots := target.npcSnapshotsLocked()
	target.mu.Unlock()
	if len(sourceSnapshots) != 0 {
		t.Fatalf("source snapshots = %#v, want no transferred npc", sourceSnapshots)
	}
	if len(targetSnapshots) != 1 || targetSnapshots[0].ID != driveControlledNPCKey {
		t.Fatalf("target snapshots = %#v, want transferred npc", targetSnapshots)
	}
}

func TestNPCDoesNotBounceWhenLandingInsideTargetPortal(t *testing.T) {
	town, house := driveNPCCrossMapTestMaps()
	world := testWorld(town, house)
	source := world.rooms[defaultMapID]
	target := world.rooms[house.ID]
	npc := source.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95
	now := time.Now()

	source.step(2, now)
	target.step(0.5, now.Add(time.Second))

	if source.liveNPCs[driveControlledNPCKey] != nil {
		t.Fatal("npc should not bounce back to source room")
	}
	transferred := target.liveNPCs[driveControlledNPCKey]
	if transferred == nil {
		t.Fatal("npc should remain in target room")
	}
	if transferred.mapID != house.ID {
		t.Fatalf("npc map = %q, want %q", transferred.mapID, house.ID)
	}
	if transferred.x <= 64 {
		t.Fatalf("npc x = %v, want movement away from landing portal toward target", transferred.x)
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

func driveNPCCrossMapTestMaps() (gameMap, gameMap) {
	town := driveNPCTestMap()
	town.Portals = []portal{{
		ID:           "house-door",
		X:            144,
		Y:            48,
		Width:        32,
		Height:       32,
		TargetMapID:  "drive-test-house",
		TargetX:      64,
		TargetY:      64,
		TargetFacing: "up",
	}}

	house := gameMap{
		ID:       "drive-test-house",
		Name:     "Drive Test House",
		TileSize: 32,
		Width:    8,
		Height:   8,
		Spawns:   []point{{X: 64, Y: 64}},
		Portals: []portal{{
			ID:           "front-door",
			X:            48,
			Y:            48,
			Width:        32,
			Height:       32,
			TargetMapID:  defaultMapID,
			TargetX:      160,
			TargetY:      64,
			TargetFacing: "down",
		}},
		Locations: []stmaps.Location{{
			ID:     "house-kitchen",
			Name:   "House Kitchen",
			X:      160,
			Y:      64,
			Radius: 32,
			Tags:   []string{"kitchen", "food"},
		}},
	}
	return town, house
}
