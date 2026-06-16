package server

import (
	"path/filepath"
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
)

func TestRuntimeAnchorsResolveFromCheckedInMaps(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorld(defaultRoomID, maps)

	mayor := world.rooms[defaultMapID].liveNPCs[driveControlledNPCKey]
	if mayor.anchors.Home == nil || mayor.anchors.Home.MapID != "sunny-town-house-1" || mayor.anchors.Home.LocationID != "mayor-sunny-bed" || mayor.anchors.Home.Source != npcAnchorSourceOwner {
		t.Fatalf("mayor home anchor = %#v, want owned mayor bed", mayor.anchors.Home)
	}

	teacher := world.rooms["sunny-town-classroom"].liveNPCs["teacher"]
	if teacher.anchors.Work == nil || teacher.anchors.Work.MapID != "sunny-town-classroom" || teacher.anchors.Work.LocationID != "teacher-desk-work" || teacher.anchors.Work.Source != npcAnchorSourceOwner {
		t.Fatalf("teacher work anchor = %#v, want owned teacher desk", teacher.anchors.Work)
	}

	keeper := world.rooms["sunny-town-house-1"].liveNPCs["cookie-keeper"]
	if keeper.anchors.Work == nil || keeper.anchors.Work.MapID != "sunny-town-house-1" || keeper.anchors.Work.LocationID != "cookie-keeper-counter" || keeper.anchors.Work.Source != npcAnchorSourceOwner {
		t.Fatalf("keeper work anchor = %#v, want owned merchant counter", keeper.anchors.Work)
	}
	if keeper.anchors.Home == nil || keeper.anchors.Home.MapID != "sunny-town-cookie-keeper-home" || keeper.anchors.Home.LocationID != "cookie-keeper-bed" || keeper.anchors.Home.Source != npcAnchorSourceOwner {
		t.Fatalf("keeper home anchor = %#v, want owned Cookie Keeper bed", keeper.anchors.Home)
	}
}

func TestCookieKeeperCanRouteBetweenHomeAndWork(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	world := newWorld(defaultRoomID, maps)

	homeToWork, err := world.navigation.PlanRouteToLocation("sunny-town-cookie-keeper-home", stnavigation.Point{X: 160, Y: 224}, "sunny-town-house-1", "cookie-keeper-counter")
	if err != nil {
		t.Fatalf("plan home to work route: %v", err)
	}
	if len(homeToWork.Steps) < 3 {
		t.Fatalf("home to work route steps = %#v, want cross-map route through town", homeToWork.Steps)
	}

	workToHome, err := world.navigation.PlanRouteToLocation("sunny-town-house-1", stnavigation.Point{X: 224, Y: 256}, "sunny-town-cookie-keeper-home", "cookie-keeper-bed")
	if err != nil {
		t.Fatalf("plan work to home route: %v", err)
	}
	if len(workToHome.Steps) < 3 {
		t.Fatalf("work to home route steps = %#v, want cross-map route through town", workToHome.Steps)
	}
}

func TestRuntimeAnchorResolutionPrefersOwnerOverGeneric(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:     "generic-bed",
			Name:   "Generic Bed",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"bed", "sleep"},
		},
		stmaps.Location{
			ID:          "owned-bed",
			Name:        "Owned Bed",
			X:           224,
			Y:           64,
			Radius:      16,
			Tags:        []string{"bed", "sleep"},
			OwnerNPCKey: driveControlledNPCKey,
		},
	)
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]

	if npc.anchors.Home == nil || npc.anchors.Home.LocationID != "owned-bed" || npc.anchors.Home.Source != npcAnchorSourceOwner {
		t.Fatalf("home anchor = %#v, want owned bed over closer generic bed", npc.anchors.Home)
	}
}

func TestRuntimeAnchorResolutionUsesRoleForWork(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.NPCs[0].Activity = &activity{Type: "schoolwork"}
	gameMap.Locations = append(gameMap.Locations, stmaps.Location{
		ID:     "school-desk",
		Name:   "School Desk",
		X:      224,
		Y:      64,
		Radius: 16,
		Tags:   []string{"school", "work"},
	})
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]

	if npc.anchors.Work == nil || npc.anchors.Work.LocationID != "school-desk" || npc.anchors.Work.Source != npcAnchorSourceRole {
		t.Fatalf("work anchor = %#v, want role-matched school desk", npc.anchors.Work)
	}
}

func TestAnchorPreferredDriveSelectionFallsBackWhenAnchorUnreachable(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.BlockedRects = []rect{{X: 192, Y: 0, Width: 32, Height: 256}}
	gameMap.Locations = append(gameMap.Locations,
		stmaps.Location{
			ID:          "blocked-owned-bed",
			Name:        "Blocked Owned Bed",
			X:           240,
			Y:           64,
			Radius:      16,
			Tags:        []string{"bed", "sleep"},
			OwnerNPCKey: driveControlledNPCKey,
		},
		stmaps.Location{
			ID:     "reachable-bed",
			Name:   "Reachable Bed",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"bed", "sleep"},
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
	if npc.goal == nil || npc.goal.location.ID != "reachable-bed" {
		t.Fatalf("goal = %#v, want reachable generic bed when anchor is blocked", npc.goal)
	}
	if npc.goal.anchorKind != "" {
		t.Fatalf("anchor kind = %q, want generic fallback goal without anchor kind", npc.goal.anchorKind)
	}
}

func TestAnchorPreferredDriveSelectionMarksAnchorGoal(t *testing.T) {
	gameMap := driveNPCTestMap()
	gameMap.Locations = append(gameMap.Locations, stmaps.Location{
		ID:          "owned-bed",
		Name:        "Owned Bed",
		X:           224,
		Y:           64,
		Radius:      16,
		Tags:        []string{"bed", "sleep"},
		OwnerNPCKey: driveControlledNPCKey,
	})
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Energy = 10
	npc.drives.Social = 95

	room.step(0.1, time.Now())

	if npc.goal == nil || npc.goal.location.ID != "owned-bed" {
		t.Fatalf("goal = %#v, want owned bed anchor goal", npc.goal)
	}
	if npc.goal.anchorKind != npcAnchorHome {
		t.Fatalf("anchor kind = %q, want home", npc.goal.anchorKind)
	}
}
