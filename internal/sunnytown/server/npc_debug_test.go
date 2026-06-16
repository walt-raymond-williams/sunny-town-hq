package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stconfig "hq/internal/sunnytown/config"
	"hq/internal/sunnytown/hqclient"
	stmaps "hq/internal/sunnytown/maps"
)

func TestNPCDebugSnapshotIncludesActiveGoal(t *testing.T) {
	room := testRoom(driveNPCTestMap())
	npc := room.liveNPCs[driveControlledNPCKey]
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	room.step(0.1, now)

	snapshot := room.world.npcDebugSnapshot(now)
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, driveControlledNPCKey)
	if debugNPC.ActiveDrive != string(npcDriveSocial) {
		t.Fatalf("active drive = %q, want social", debugNPC.ActiveDrive)
	}
	if debugNPC.Drives.Social <= 0 || debugNPC.Drives.Hunger <= 0 {
		t.Fatalf("drives = %#v, want populated drive values", debugNPC.Drives)
	}
	if debugNPC.Goal == nil || debugNPC.Goal.MapID != defaultMapID || debugNPC.Goal.LocationID != driveStartLocationID {
		t.Fatalf("goal = %#v, want town square social goal", debugNPC.Goal)
	}
	if debugNPC.Route == nil || debugNPC.Route.StepCount != len(npc.route.Steps) || debugNPC.Route.StepIndex != npc.routeStep {
		t.Fatalf("route = %#v, want active route metadata", debugNPC.Route)
	}
	if debugNPC.Routine.Status != "traveling" || debugNPC.Routine.TargetKey != npc.goal.failureKey() {
		t.Fatalf("routine = %#v, want traveling to active target", debugNPC.Routine)
	}
	if debugNPC.FocusUntil == "" || debugNPC.ReevaluateAt == "" || debugNPC.GoalStartedAt == "" {
		t.Fatalf("goal timestamps focus=%q reevaluate=%q started=%q, want populated", debugNPC.FocusUntil, debugNPC.ReevaluateAt, debugNPC.GoalStartedAt)
	}
}

func TestNPCDebugSnapshotIncludesAnchorsAndGoalAnchorKind(t *testing.T) {
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
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	room.step(0.1, now)

	snapshot := room.world.npcDebugSnapshot(now)
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, driveControlledNPCKey)
	if debugNPC.Anchors.Home == nil || debugNPC.Anchors.Home.LocationID != "owned-bed" || debugNPC.Anchors.Home.Source != npcAnchorSourceOwner {
		t.Fatalf("debug home anchor = %#v, want owned bed anchor", debugNPC.Anchors.Home)
	}
	if debugNPC.Goal == nil || debugNPC.Goal.AnchorKind != npcAnchorHome {
		t.Fatalf("debug goal = %#v, want home anchor kind", debugNPC.Goal)
	}
}

func TestNPCDebugSnapshotShowsIdleNPCClearly(t *testing.T) {
	room := testRoom(testMap())
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	snapshot := room.world.npcDebugSnapshot(now)
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, "guide")
	if debugNPC.ActiveDrive != "" {
		t.Fatalf("active drive = %q, want empty for idle npc", debugNPC.ActiveDrive)
	}
	if debugNPC.Goal != nil {
		t.Fatalf("goal = %#v, want nil for idle npc", debugNPC.Goal)
	}
	if debugNPC.Route != nil {
		t.Fatalf("route = %#v, want nil for idle npc", debugNPC.Route)
	}
	if debugNPC.FailureCount != 0 || len(debugNPC.FailedTargets) != 0 {
		t.Fatalf("failure state count=%d targets=%#v, want empty", debugNPC.FailureCount, debugNPC.FailedTargets)
	}
}

func TestNPCDebugSnapshotIncludesFailedTargets(t *testing.T) {
	gameMap := driveNPCTestMap()
	room := testRoom(gameMap)
	npc := room.liveNPCs[driveControlledNPCKey]
	failedAt := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	goal := npcGoal{
		drive:    npcDriveSocial,
		mapID:    room.gameMap.ID,
		location: gameMap.Locations[0],
	}
	npc.markTargetFailed(goal, failedAt)

	snapshot := room.world.npcDebugSnapshot(failedAt)
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, driveControlledNPCKey)
	if debugNPC.FailureCount != 1 {
		t.Fatalf("failure count = %d, want 1", debugNPC.FailureCount)
	}
	if len(debugNPC.FailedTargets) != 1 {
		t.Fatalf("failed targets = %#v, want one target", debugNPC.FailedTargets)
	}
	failedTarget := debugNPC.FailedTargets[0]
	if failedTarget.Key != goal.failureKey() {
		t.Fatalf("failed key = %q, want %q", failedTarget.Key, goal.failureKey())
	}
	if failedTarget.Drive != string(npcDriveSocial) || failedTarget.MapID != defaultMapID || failedTarget.LocationID != driveStartLocationID {
		t.Fatalf("failed target detail = %#v, want split target fields", failedTarget)
	}
	if failedTarget.FailedAt != formatDebugTime(failedAt) || failedTarget.RetryAt != formatDebugTime(failedAt.Add(npcFailedTargetCooldown)) {
		t.Fatalf("failed target = %#v, want failed and retry timestamps", failedTarget)
	}
	if debugNPC.Routine.Status != "blocked" || !debugNPC.Routine.RouteBlocked || debugNPC.Routine.MostRecentFailureKey != goal.failureKey() {
		t.Fatalf("routine = %#v, want blocked failed target summary", debugNPC.Routine)
	}
}

func TestNPCDebugSnapshotShowsScheduledRoutineState(t *testing.T) {
	now := time.Unix(0, int64(3*time.Minute))
	world := newWorldWithNPCDayLength("sunny-town-main", map[string]gameMap{
		defaultMapID: npcProductionTestMap(64, 64),
	}, 8*time.Minute)
	room := world.defaultRoom
	npc := room.liveNPCs["cookie-keeper"]
	npc.drives.Work = 95

	room.step(0.1, now)

	debugNPC := requireDebugNPC(t, world.npcDebugSnapshot(now), defaultMapID, "cookie-keeper")
	if debugNPC.Schedule.Phase != string(npcSchedulePhaseDay) || len(debugNPC.Schedule.Pressures) == 0 {
		t.Fatalf("schedule = %#v, want day pressure", debugNPC.Schedule)
	}
	if !debugNPC.Routine.Scheduled {
		t.Fatalf("routine = %#v, want scheduled target", debugNPC.Routine)
	}
}

func TestNPCDebugSnapshotIncludesProductionBlockers(t *testing.T) {
	room := testRoom(npcProductionTestMap(64, 64))

	debugNPC := requireDebugNPC(t, room.world.npcDebugSnapshot(time.Now()), defaultMapID, "cookie-keeper")
	if debugNPC.Production.Status != "blocked" || debugNPC.Production.Blocker != "not_at_work_anchor" {
		t.Fatalf("away production = %#v, want not-at-work blocker", debugNPC.Production)
	}

	room = testRoom(npcProductionTestMap(160, 160))
	debugNPC = requireDebugNPC(t, room.world.npcDebugSnapshot(time.Now()), defaultMapID, "cookie-keeper")
	if debugNPC.Production.Status != "blocked" || debugNPC.Production.Blocker != "missing_character_identity" {
		t.Fatalf("missing-character production = %#v, want missing identity blocker", debugNPC.Production)
	}
}

func TestNPCDebugSnapshotIncludesLastProductionCommitResult(t *testing.T) {
	room := testRoom(npcProductionTestMap(160, 160))
	room.world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 444,
		NPCKey:      "cookie-keeper",
		DisplayName: "Cookie Keeper",
		AvatarID:    "keeper",
	}})
	event := npcJobProductionEvent{npcKey: "cookie-keeper"}
	committedAt := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	room.world.recordNPCJobProductionCommit(event, npcJobProductionResponse{
		Accepted:      true,
		Blocked:       true,
		BlockedReason: "output_full",
	}, nil, committedAt)

	debugNPC := requireDebugNPC(t, room.world.npcDebugSnapshot(committedAt), defaultMapID, "cookie-keeper")
	if debugNPC.Production.LastCommitStatus != "blocked" || debugNPC.Production.LastBlockedReason != "output_full" {
		t.Fatalf("production = %#v, want blocked commit status", debugNPC.Production)
	}
	if debugNPC.Production.LastCommitAt != formatDebugTime(committedAt) {
		t.Fatalf("last commit at = %q, want %q", debugNPC.Production.LastCommitAt, formatDebugTime(committedAt))
	}

	room.world.recordNPCJobProductionCommit(event, npcJobProductionResponse{}, errors.New("hq unavailable"), committedAt.Add(time.Second))
	debugNPC = requireDebugNPC(t, room.world.npcDebugSnapshot(committedAt), defaultMapID, "cookie-keeper")
	if debugNPC.Production.LastCommitStatus != "error" || debugNPC.Production.LastCommitError != "hq unavailable" {
		t.Fatalf("production = %#v, want error commit status", debugNPC.Production)
	}
}

func TestHandleNPCDebugRequiresServiceSecretAndReturnsJSON(t *testing.T) {
	srv := New(stconfig.Config{ServiceSecret: "debug-secret"}, map[string]gameMap{
		defaultMapID: driveNPCTestMap(),
	})

	unauthorized := httptest.NewRecorder()
	srv.HandleNPCDebug(unauthorized, httptest.NewRequest(http.MethodGet, "/debug/npcs", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	request := httptest.NewRequest(http.MethodGet, "/debug/npcs", nil)
	request.Header.Set("X-HQ-Service-Secret", "debug-secret")
	response := httptest.NewRecorder()
	srv.HandleNPCDebug(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q, want 200", response.Code, response.Body.String())
	}
	var snapshot npcDebugResponse
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode debug response: %v", err)
	}
	debugNPC := requireDebugNPC(t, snapshot, defaultMapID, driveControlledNPCKey)
	if debugNPC.ID != driveControlledNPCKey || debugNPC.Name == "" {
		t.Fatalf("debug npc = %#v, want mayor sunny debug payload", debugNPC)
	}
}

func requireDebugNPC(t *testing.T, snapshot npcDebugResponse, mapID string, npcID string) npcDebugNPC {
	t.Helper()
	for _, debugMap := range snapshot.Maps {
		if debugMap.MapID != mapID {
			continue
		}
		for _, debugNPC := range debugMap.NPCs {
			if debugNPC.ID == npcID {
				return debugNPC
			}
		}
		t.Fatalf("map %q npcs = %#v, want npc %q", mapID, debugMap.NPCs, npcID)
	}
	t.Fatalf("debug maps = %#v, want map %q", snapshot.Maps, mapID)
	return npcDebugNPC{}
}
