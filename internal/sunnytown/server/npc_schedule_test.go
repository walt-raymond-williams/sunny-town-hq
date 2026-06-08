package server

import (
	"testing"
	"time"

	stmaps "hq/internal/sunnytown/maps"
)

func TestNPCSchedulePhaseAt(t *testing.T) {
	tests := []struct {
		name string
		hour int
		want npcSchedulePhase
	}{
		{name: "night before morning", hour: 5, want: npcSchedulePhaseNight},
		{name: "morning start", hour: 6, want: npcSchedulePhaseMorning},
		{name: "day start", hour: 10, want: npcSchedulePhaseDay},
		{name: "evening start", hour: 17, want: npcSchedulePhaseEvening},
		{name: "night start", hour: 21, want: npcSchedulePhaseNight},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 6, 8, test.hour, 0, 0, 0, time.UTC)
			if got := npcSchedulePhaseAt(now); got != test.want {
				t.Fatalf("phase = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDayScheduleBiasesNPCWithWorkAnchorTowardWork(t *testing.T) {
	room := testRoom(scheduleNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveWork {
		t.Fatalf("active drive = %q, want work", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "owned-workbench" || npc.goal.anchorKind != npcAnchorWork {
		t.Fatalf("goal = %#v, want owned work anchor", npc.goal)
	}
}

func TestNightScheduleBiasesNPCWithHomeAnchorTowardRest(t *testing.T) {
	room := testRoom(scheduleNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Date(2026, 6, 8, 22, 0, 0, 0, time.UTC)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveEnergy {
		t.Fatalf("active drive = %q, want energy", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "owned-bed" || npc.goal.anchorKind != npcAnchorHome {
		t.Fatalf("goal = %#v, want owned home anchor", npc.goal)
	}
}

func TestUrgentHungerOverridesSchedulePressure(t *testing.T) {
	room := testRoom(scheduleNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 10
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	room.step(0.1, now)

	if npc.activeDrive != npcDriveHunger {
		t.Fatalf("active drive = %q, want hunger", npc.activeDrive)
	}
	if npc.goal == nil || npc.goal.location.ID != "snack-table" {
		t.Fatalf("goal = %#v, want snack table", npc.goal)
	}
}

func TestNPCDebugSnapshotIncludesScheduleState(t *testing.T) {
	room := testRoom(scheduleNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	npc.drives.Hunger = 95
	npc.drives.Energy = 95
	npc.drives.Social = 95
	npc.drives.Work = 95
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	snapshot := room.world.npcDebugSnapshot(now)
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, driveControlledNPCKey)
	if debugNPC.Schedule.Phase != string(npcSchedulePhaseDay) {
		t.Fatalf("phase = %q, want day", debugNPC.Schedule.Phase)
	}
	if len(debugNPC.Schedule.Pressures) != 1 {
		t.Fatalf("pressures = %#v, want one work pressure", debugNPC.Schedule.Pressures)
	}
	pressure := debugNPC.Schedule.Pressures[0]
	if pressure.Drive != string(npcDriveWork) || pressure.Pressure != npcScheduleMajorPressure || pressure.SelectionValue >= npcDriveThreshold {
		t.Fatalf("pressure = %#v, want work pressure below selection threshold", pressure)
	}
}

func scheduleNPCTestMap() gameMap {
	gameMap := driveNPCTestMap()
	gameMap.Locations = []stmaps.Location{
		{
			ID:          "owned-workbench",
			Name:        "Owned Workbench",
			X:           160,
			Y:           64,
			Radius:      16,
			Tags:        []string{"work"},
			OwnerNPCKey: driveControlledNPCKey,
		},
		{
			ID:          "owned-bed",
			Name:        "Owned Bed",
			X:           224,
			Y:           64,
			Radius:      16,
			Tags:        []string{"home", "bed", "sleep"},
			OwnerNPCKey: driveControlledNPCKey,
		},
		{
			ID:     "snack-table",
			Name:   "Snack Table",
			X:      128,
			Y:      64,
			Radius: 16,
			Tags:   []string{"food", "meal"},
		},
	}
	return gameMap
}
