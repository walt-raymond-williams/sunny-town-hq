package server

import (
	"path/filepath"
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
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

func TestNPCSelectsCloserDriveLocation(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:     "far-snack-stand",
			Name:   "Far Snack Stand",
			X:      224,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
		stmaps.Location{
			ID:     "near-snack-stand",
			Name:   "Near Snack Stand",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
	)
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "near-snack-stand" {
		t.Fatalf("goal = %#v, want closer snack stand", npc.goal)
	}
}

func TestNPCSelectsOwnedDriveLocation(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:     "guest-bed",
			Name:   "Guest Bed",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"bed", "sleep"},
		},
		stmaps.Location{
			ID:          "mayor-bed",
			Name:        "Mayor Bed",
			X:           160,
			Y:           64,
			Radius:      16,
			Tags:        []string{"bed", "sleep", "home"},
			OwnerNPCKey: driveControlledNPCKey,
		},
	)
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Energy = 10
	npc.drives.Social = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveEnergy {
		t.Fatalf("active drive = %q, want energy", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "mayor-bed" {
		t.Fatalf("goal = %#v, want owned mayor bed", npc.goal)
	}
}

func TestNPCDriveLocationTieBreaksByMapAndLocationID(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:     "z-snack-stand",
			Name:   "Z Snack Stand",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
		stmaps.Location{
			ID:     "a-snack-stand",
			Name:   "A Snack Stand",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
	)
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95

	room.step(0.1, time.Now())

	if npc.goal == nil || npc.goal.location.ID != "a-snack-stand" {
		t.Fatalf("goal = %#v, want deterministic location ID tie-break", npc.goal)
	}
}

func TestNPCChoosesIdleFallbackWhenNoDriveIsUrgent(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveIdle {
		t.Fatalf("active drive = %q, want idle fallback", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want town square idle fallback", npc.goal)
	}
}

func TestNPCIdleFallbackPersistsAfterArrival(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Now()

	room.step(2, now)
	if npc.route != nil {
		t.Fatalf("route = %#v, want cleared after arriving at idle fallback", npc.route)
	}
	if npc.goal == nil || npc.goal.drive != npcDriveIdle {
		t.Fatalf("goal = %#v, want idle goal to remain after arrival", npc.goal)
	}

	goalStartedAt := npc.goalStartedAt
	room.step(0.1, now.Add(time.Second))
	if npc.goal == nil || npc.goal.drive != npcDriveIdle {
		t.Fatalf("goal = %#v, want idle fallback to persist", npc.goal)
	}
	if !npc.goalStartedAt.Equal(goalStartedAt) {
		t.Fatalf("goal started at = %v, want unchanged %v", npc.goalStartedAt, goalStartedAt)
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

func TestUnavailableUrgentDriveFallsBackToIdle(t *testing.T) {
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
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveIdle {
		t.Fatalf("active drive = %q, want idle fallback", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != driveStartLocationID {
		t.Fatalf("goal = %#v, want town square idle fallback", npc.goal)
	}
}

func TestNPCDriveRouteAvoidsActiveCollisionWorldObject(t *testing.T) {
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
	now := time.Now()
	room.worldObjects["placed:blocking-wall"] = &worldObject{
		id:        "blocking-wall",
		source:    worldObjectSourcePlaced,
		mapID:     room.gameMap.ID,
		x:         96,
		y:         0,
		width:     32,
		height:    256,
		active:    true,
		collision: true,
	}

	if goal, route, ok := room.routeToDriveLocationLocked(npc, npcDriveHunger, now); ok {
		t.Fatalf("goal = %#v route = %#v, want active collision object to block route", goal, route)
	}
	if npc.failureCount != 1 {
		t.Fatalf("failure count = %d, want one failed target after blocked route", npc.failureCount)
	}
}

func TestNPCDriveRouteSkipsFailedDynamicBlockTargetUntilCooldown(t *testing.T) {
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
	room.worldObjects["placed:blocking-wall"] = &worldObject{
		id:        "blocking-wall",
		source:    worldObjectSourcePlaced,
		mapID:     room.gameMap.ID,
		x:         96,
		y:         0,
		width:     32,
		height:    256,
		active:    true,
		collision: true,
	}

	if _, _, ok := room.routeToDriveLocationLocked(npc, npcDriveHunger, now); ok {
		t.Fatal("expected active collision object to block route")
	}
	if _, _, ok := room.routeToDriveLocationLocked(npc, npcDriveHunger, now.Add(time.Second)); ok {
		t.Fatal("expected failed target cooldown to skip blocked route retry")
	}
	if npc.failureCount != 1 {
		t.Fatalf("failure count = %d, want cooldown to prevent repeated failure marks", npc.failureCount)
	}
}

func TestNPCDriveRouteIgnoresInactiveCollisionWorldObjectAfterCooldown(t *testing.T) {
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
	blocker := &worldObject{
		id:        "blocking-wall",
		source:    worldObjectSourcePlaced,
		mapID:     room.gameMap.ID,
		x:         96,
		y:         0,
		width:     32,
		height:    256,
		active:    true,
		collision: true,
	}
	room.worldObjects["placed:blocking-wall"] = blocker

	if _, _, ok := room.routeToDriveLocationLocked(npc, npcDriveHunger, now); ok {
		t.Fatal("expected active collision object to block route")
	}
	blocker.active = false

	goal, route, ok := room.routeToDriveLocationLocked(npc, npcDriveHunger, now.Add(npcFailedTargetCooldown+time.Second))
	if !ok {
		t.Fatal("expected inactive collision object to stop blocking route after cooldown")
	}
	if goal.location.ID != "snack-stand" || len(route.Steps) != 1 {
		t.Fatalf("goal = %#v route = %#v, want snack stand same-map route", goal, route)
	}
}

func TestPausedRoomSkipsExactNPCMovementWhenNoPlayersRemain(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	startX := npc.x
	startY := npc.y
	now := time.Now()
	room.npcPausedAt = now

	room.step(2, now.Add(2*time.Second))

	if npc.x != startX || npc.y != startY {
		t.Fatalf("npc position = (%v,%v), want unchanged (%v,%v)", npc.x, npc.y, startX, startY)
	}
	if npc.activeDrive != "" || npc.goal != nil || npc.route != nil || npc.moving {
		t.Fatalf("npc state activeDrive=%q goal=%#v route=%#v moving=%v, want no exact movement while paused", npc.activeDrive, npc.goal, npc.route, npc.moving)
	}
}

func TestNoPlayerCatchUpIsBoundedAndClearsTransientNPCGoal(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	now := time.Now()
	goal, route, ok := room.nextNPCFallbackGoalLocked(npc, now)
	if !ok {
		t.Fatal("expected fallback goal")
	}
	room.assignNPCGoal(npc, goal, route, now)
	room.npcPausedAt = now.Add(-10 * time.Minute)

	room.catchUpNPCsAfterNoPlayersLocked(now)

	wantDrive := npcDriveDefault - npcDriveDepletePerSecond*npcNoPlayerCatchUpMax.Seconds()
	if npc.drives.Hunger != wantDrive || npc.drives.Energy != wantDrive || npc.drives.Work != wantDrive {
		t.Fatalf("drives = %#v, want bounded depletion to %v", npc.drives, wantDrive)
	}
	if npc.goal != nil || npc.route != nil || npc.activeDrive != "" || npc.moving {
		t.Fatalf("npc state activeDrive=%q goal=%#v route=%#v moving=%v, want transient goal cleared", npc.activeDrive, npc.goal, npc.route, npc.moving)
	}
	if !room.npcPausedAt.IsZero() {
		t.Fatalf("npcPausedAt = %v, want reset after catch-up", room.npcPausedAt)
	}
}

func TestNoPlayerCatchUpReplenishesCurrentLocation(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.NPCs[0].X = 160
	gameMap.NPCs[0].Y = 64
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Social = 40
	now := time.Now()
	room.npcPausedAt = now.Add(-time.Minute)

	room.catchUpNPCsAfterNoPlayersLocked(now)

	if npc.drives.Social != 100 {
		t.Fatalf("social = %v, want replenished to 100 at current social location", npc.drives.Social)
	}
	if npc.drives.Hunger >= npcDriveDefault {
		t.Fatalf("hunger = %v, want nonmatching drive to deplete", npc.drives.Hunger)
	}
}

func TestUrgentDriveInterruptsIdleFallbackAfterReevaluation(t *testing.T) {
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
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Now()

	room.step(0.1, now)
	if npc.activeDrive != npcDriveIdle {
		t.Fatalf("active drive = %q, want idle fallback", npc.activeDrive)
	}

	npc.drives.Hunger = 30
	room.step(0.1, now.Add(time.Second))
	if npc.activeDrive != npcDriveIdle {
		t.Fatalf("active drive = %q, want focus window to keep idle fallback", npc.activeDrive)
	}

	room.step(0.1, now.Add(npcGoalFocusDuration+npcGoalReevaluateInterval))
	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger to interrupt idle fallback", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "snack-stand" {
		t.Fatalf("goal = %#v, want snack stand hunger goal", npc.goal)
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

func TestNPCDriveLocationScoringSkipsFailedTarget(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:     "near-snack-stand",
			Name:   "Near Snack Stand",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
		stmaps.Location{
			ID:     "far-snack-stand",
			Name:   "Far Snack Stand",
			X:      224,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food"},
		},
	)
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Social = 95
	now := time.Now()
	npc.markTargetFailed(npcGoal{
		drive:    npcDriveHunger,
		mapID:    room.gameMap.ID,
		location: gameMap.Locations[1],
	}, now)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "far-snack-stand" {
		t.Fatalf("goal = %#v, want farther snack stand while nearer target cools down", npc.goal)
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

func TestMayorSelectsAuthoredOwnedBedInCheckedInMaps(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorld(defaultRoomID, maps)
	room := world.rooms[defaultMapID]
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 10
	npc.drives.Social = 95
	npc.drives.Work = 95

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveEnergy {
		t.Fatalf("active drive = %q, want energy", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.mapID != "sunny-town-house-1" || npc.goal.location.ID != "mayor-sunny-bed" {
		t.Fatalf("goal = %#v, want authored owned mayor bed", npc.goal)
	}
}

func TestTeacherSelectsAuthoredSchoolWorkLocation(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	classroom := maps["sunny-town-classroom"]
	classroom.NPCs[0].X = 320
	classroom.NPCs[0].Y = 384
	maps["sunny-town-classroom"] = classroom
	world := newWorld(defaultRoomID, maps)
	room := world.rooms["sunny-town-classroom"]
	npc := room.liveNPCs["teacher"]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 10

	room.step(0.1, time.Now())

	if npc.activeDrive != npcDriveWork {
		t.Fatalf("active drive = %q, want work", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.mapID != "sunny-town-classroom" || npc.goal.location.ID != "teacher-desk-work" {
		t.Fatalf("goal = %#v, want authored teacher desk work location", npc.goal)
	}
}

func TestCookieKeeperDayScheduleHoldsWorkAnchorWhenAlreadyThere(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorldWithNPCDayLength(defaultRoomID, maps, 8*time.Minute)
	room := world.rooms["sunny-town-house-1"]
	npc := room.liveNPCs["cookie-keeper"]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := npcScheduleTestBaseTime().Add(3 * time.Minute)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveWork {
		t.Fatalf("active drive = %q, want scheduled work", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.mapID != "sunny-town-house-1" || npc.goal.location.ID != "cookie-keeper-counter" || npc.goal.anchorKind != npcAnchorWork {
		t.Fatalf("goal = %#v, want stationary counter work goal", npc.goal)
	}
	if npc.route != nil || npc.moving {
		t.Fatalf("route = %#v moving=%v, want stationary work goal at counter", npc.route, npc.moving)
	}

	room.step(0.1, now.Add(time.Second))

	if npc.goal == nil || npc.goal.location.ID != "cookie-keeper-counter" {
		t.Fatalf("goal = %#v, want scheduled work goal to persist while day pressure is active", npc.goal)
	}
}

func TestCookieKeeperDemoCadenceRoutesHomeThenBackToWork(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorldWithNPCDayLength(defaultRoomID, maps, 8*time.Minute)
	house := world.rooms["sunny-town-house-1"]
	keeper := house.liveNPCs["cookie-keeper"]
	keeper.drives.Hunger = 95
	keeper.drives.Energy = 95
	keeper.drives.Social = 95
	keeper.drives.Work = 95
	night := npcScheduleTestBaseTime().Add(7 * time.Minute)

	stepWorldForTest(world, 0.1, night)

	if keeper.activeDrive != npcDriveEnergy {
		t.Fatalf("active drive = %q, want scheduled home/rest", keeper.activeDrive)
	}
	if keeper.goal == nil || keeper.goal.mapID != "sunny-town-cookie-keeper-home" || keeper.goal.location.ID != "cookie-keeper-bed" || keeper.goal.anchorKind != npcAnchorHome {
		t.Fatalf("goal = %#v, want owned Cookie Keeper bed", keeper.goal)
	}
	if keeper.route == nil || len(keeper.route.Steps) < 3 {
		t.Fatalf("route = %#v, want cross-map route from work to home bed", keeper.route)
	}

	stepWorldForTest(world, 35, night.Add(time.Second))

	home := world.rooms["sunny-town-cookie-keeper-home"]
	keeper = home.liveNPCs["cookie-keeper"]
	if keeper == nil {
		t.Fatalf("keeper location after night route: house=%v town=%v home=%v, want home", house.liveNPCs["cookie-keeper"] != nil, world.rooms[defaultMapID].liveNPCs["cookie-keeper"] != nil, home.liveNPCs["cookie-keeper"] != nil)
	}
	if keeper.goal == nil || keeper.goal.location.ID != "cookie-keeper-bed" || keeper.route != nil {
		t.Fatalf("keeper goal=%#v route=%#v, want arrived at bed", keeper.goal, keeper.route)
	}
	bed, ok := world.navigation.Location("sunny-town-cookie-keeper-home", "cookie-keeper-bed")
	if !ok || !pointWithinLocation(stnavigation.Point{X: keeper.x, Y: keeper.y}, bed) {
		t.Fatalf("keeper position = (%v,%v), want inside Cookie Keeper bed", keeper.x, keeper.y)
	}

	keeper.drives.Hunger = 95
	keeper.drives.Energy = 95
	keeper.drives.Social = 95
	keeper.drives.Work = 95
	day := npcScheduleTestBaseTime().Add(10 * time.Minute)
	stepWorldForTest(world, 0.1, day)

	if keeper.activeDrive != npcDriveWork {
		t.Fatalf("active drive = %q, want scheduled work after day phase starts", keeper.activeDrive)
	}
	if keeper.goal == nil || keeper.goal.mapID != "sunny-town-house-1" || keeper.goal.location.ID != "cookie-keeper-counter" || keeper.goal.anchorKind != npcAnchorWork {
		t.Fatalf("goal = %#v, want counter work goal from home", keeper.goal)
	}
	if keeper.route == nil || len(keeper.route.Steps) < 3 {
		t.Fatalf("route = %#v, want cross-map route from home bed to work counter", keeper.route)
	}

	stepWorldForTest(world, 35, day.Add(time.Second))

	keeper = house.liveNPCs["cookie-keeper"]
	if keeper == nil {
		t.Fatalf("keeper location after day route: house=%v town=%v home=%v, want house", house.liveNPCs["cookie-keeper"] != nil, world.rooms[defaultMapID].liveNPCs["cookie-keeper"] != nil, home.liveNPCs["cookie-keeper"] != nil)
	}
	counter, ok := world.navigation.Location("sunny-town-house-1", "cookie-keeper-counter")
	if !ok || !pointWithinLocation(stnavigation.Point{X: keeper.x, Y: keeper.y}, counter) {
		t.Fatalf("keeper position = (%v,%v), want inside Cookie Keeper counter", keeper.x, keeper.y)
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

func stepWorldForTest(world *world, dt float64, now time.Time) {
	if world == nil || dt <= 0 {
		return
	}
	remaining := dt
	for remaining > 0 {
		step := remaining
		if step > 0.1 {
			step = 0.1
		}
		for _, room := range world.rooms {
			room.step(step, now)
		}
		remaining -= step
		now = now.Add(time.Duration(step * float64(time.Second)))
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
