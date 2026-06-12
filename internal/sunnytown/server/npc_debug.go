package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

type npcDebugResponse struct {
	RoomID string        `json:"roomId"`
	Maps   []npcDebugMap `json:"maps"`
}

type npcDebugMap struct {
	MapID string        `json:"mapId"`
	Tick  int64         `json:"tick"`
	NPCs  []npcDebugNPC `json:"npcs"`
}

type npcDebugNPC struct {
	ID            string                 `json:"id"`
	CharacterID   int64                  `json:"characterId,omitempty"`
	Name          string                 `json:"name"`
	MapID         string                 `json:"mapId"`
	X             float64                `json:"x"`
	Y             float64                `json:"y"`
	Facing        string                 `json:"facing"`
	Moving        bool                   `json:"moving"`
	Drives        npcDebugDrives         `json:"drives"`
	Anchors       npcDebugAnchors        `json:"anchors"`
	Schedule      npcDebugSchedule       `json:"schedule"`
	Routine       npcDebugRoutine        `json:"routine"`
	Production    npcDebugProduction     `json:"production"`
	ActiveDrive   string                 `json:"activeDrive,omitempty"`
	Goal          *npcDebugGoal          `json:"goal,omitempty"`
	Route         *npcDebugRoute         `json:"route,omitempty"`
	FocusUntil    string                 `json:"focusUntil,omitempty"`
	ReevaluateAt  string                 `json:"reevaluateAt,omitempty"`
	GoalStartedAt string                 `json:"goalStartedAt,omitempty"`
	GoalArrivedAt string                 `json:"goalArrivedAt,omitempty"`
	GoalDriveAt   float64                `json:"goalDriveAt,omitempty"`
	FailureCount  int                    `json:"failureCount"`
	FailedTargets []npcDebugFailedTarget `json:"failedTargets,omitempty"`
}

type npcDebugDrives struct {
	Hunger float64 `json:"hunger"`
	Energy float64 `json:"energy"`
	Social float64 `json:"social"`
	Work   float64 `json:"work"`
}

type npcDebugAnchors struct {
	Home   *npcDebugAnchor `json:"home,omitempty"`
	Work   *npcDebugAnchor `json:"work,omitempty"`
	Food   *npcDebugAnchor `json:"food,omitempty"`
	Social *npcDebugAnchor `json:"social,omitempty"`
}

type npcDebugSchedule struct {
	Phase     string                  `json:"phase"`
	Pressures []npcDebugDrivePressure `json:"pressures,omitempty"`
}

type npcDebugProduction struct {
	Eligible          bool    `json:"eligible"`
	Status            string  `json:"status"`
	Blocker           string  `json:"blocker,omitempty"`
	JobKey            string  `json:"jobKey,omitempty"`
	OutputKey         string  `json:"outputKey,omitempty"`
	ProgressSeconds   float64 `json:"progressSeconds,omitempty"`
	LastAt            string  `json:"lastAt,omitempty"`
	LastEvent         string  `json:"lastEvent,omitempty"`
	LastCommitAt      string  `json:"lastCommitAt,omitempty"`
	LastCommitStatus  string  `json:"lastCommitStatus,omitempty"`
	LastBlockedReason string  `json:"lastBlockedReason,omitempty"`
	LastCommitError   string  `json:"lastCommitError,omitempty"`
}

type npcDebugRoutine struct {
	Status               string `json:"status"`
	TargetKey            string `json:"targetKey,omitempty"`
	Scheduled            bool   `json:"scheduled"`
	RouteBlocked         bool   `json:"routeBlocked"`
	CurrentFailureKey    string `json:"currentFailureKey,omitempty"`
	MostRecentFailureKey string `json:"mostRecentFailureKey,omitempty"`
}

type npcDebugDrivePressure struct {
	Drive          string  `json:"drive"`
	Pressure       float64 `json:"pressure"`
	Value          float64 `json:"value"`
	SelectionValue float64 `json:"selectionValue"`
}

type npcDebugAnchor struct {
	Kind         string   `json:"kind"`
	MapID        string   `json:"mapId"`
	LocationID   string   `json:"locationId"`
	LocationName string   `json:"locationName,omitempty"`
	Source       string   `json:"source"`
	Tags         []string `json:"tags,omitempty"`
}

type npcDebugGoal struct {
	Drive        string   `json:"drive"`
	MapID        string   `json:"mapId"`
	LocationID   string   `json:"locationId"`
	LocationName string   `json:"locationName,omitempty"`
	AnchorKind   string   `json:"anchorKind,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type npcDebugRoute struct {
	StepIndex       int    `json:"stepIndex"`
	PathIndex       int    `json:"pathIndex"`
	StepCount       int    `json:"stepCount"`
	CurrentMapID    string `json:"currentMapId,omitempty"`
	CurrentPortalID string `json:"currentPortalId,omitempty"`
	TargetMapID     string `json:"targetMapId,omitempty"`
}

type npcDebugFailedTarget struct {
	Key        string `json:"key"`
	Drive      string `json:"drive,omitempty"`
	MapID      string `json:"mapId,omitempty"`
	LocationID string `json:"locationId,omitempty"`
	FailedAt   string `json:"failedAt"`
	RetryAt    string `json:"retryAt"`
}

func (srv *Server) HandleNPCDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if srv.config.ServiceSecret != "" && r.Header.Get("X-HQ-Service-Secret") != srv.config.ServiceSecret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(srv.npcDebugSnapshot(time.Now())); err != nil {
		http.Error(w, "encode npc debug snapshot", http.StatusInternalServerError)
	}
}

func (srv *Server) npcDebugSnapshot(now time.Time) npcDebugResponse {
	if srv == nil || srv.world == nil {
		return npcDebugResponse{}
	}
	return srv.world.npcDebugSnapshot(now)
}

func (world *world) npcDebugSnapshot(now time.Time) npcDebugResponse {
	response := npcDebugResponse{
		RoomID: world.roomID,
		Maps:   make([]npcDebugMap, 0, len(world.rooms)),
	}
	mapIDs := make([]string, 0, len(world.rooms))
	for mapID := range world.rooms {
		mapIDs = append(mapIDs, mapID)
	}
	sort.Strings(mapIDs)

	for _, mapID := range mapIDs {
		room := world.rooms[mapID]
		if room == nil {
			continue
		}
		room.mu.Lock()
		debugMap := npcDebugMap{
			MapID: room.gameMap.ID,
			Tick:  room.tick,
			NPCs:  make([]npcDebugNPC, 0, len(room.liveNPCs)),
		}
		npcKeys := make([]string, 0, len(room.liveNPCs))
		for npcKey := range room.liveNPCs {
			npcKeys = append(npcKeys, npcKey)
		}
		sort.Strings(npcKeys)
		for _, npcKey := range npcKeys {
			debugMap.NPCs = append(debugMap.NPCs, room.npcDebugSnapshotLocked(room.liveNPCs[npcKey], now))
		}
		room.mu.Unlock()
		response.Maps = append(response.Maps, debugMap)
	}
	return response
}

func (room *room) npcDebugSnapshotLocked(liveNPC *liveNPC, now time.Time) npcDebugNPC {
	character, hasCharacter := room.world.npcCharacter(liveNPC.npcKey)
	publicSnapshot := liveNPC.snapshot(character, hasCharacter, now)
	debugNPC := npcDebugNPC{
		ID:            publicSnapshot.ID,
		CharacterID:   publicSnapshot.CharacterID,
		Name:          publicSnapshot.Name,
		MapID:         liveNPC.mapID,
		X:             publicSnapshot.X,
		Y:             publicSnapshot.Y,
		Facing:        publicSnapshot.Facing,
		Moving:        publicSnapshot.Moving,
		Drives:        npcDebugDrives(liveNPC.drives),
		Anchors:       debugAnchors(liveNPC.anchors),
		Schedule:      liveNPC.debugSchedule(now, room.scheduleDayLength()),
		Routine:       liveNPC.debugRoutine(now, room.scheduleDayLength()),
		Production:    room.npcProductionDebugSnapshotLocked(liveNPC),
		ActiveDrive:   string(liveNPC.activeDrive),
		FocusUntil:    formatDebugTime(liveNPC.focusUntil),
		ReevaluateAt:  formatDebugTime(liveNPC.reevaluateAt),
		GoalStartedAt: formatDebugTime(liveNPC.goalStartedAt),
		GoalArrivedAt: formatDebugTime(liveNPC.goalArrivedAt),
		GoalDriveAt:   liveNPC.goalArriveDrive,
		FailureCount:  liveNPC.failureCount,
		FailedTargets: liveNPC.failedTargetsDebugSnapshot(),
	}
	if liveNPC.goal != nil {
		debugNPC.Goal = &npcDebugGoal{
			Drive:        string(liveNPC.goal.drive),
			MapID:        liveNPC.goal.mapID,
			LocationID:   liveNPC.goal.location.ID,
			LocationName: liveNPC.goal.location.Name,
			AnchorKind:   liveNPC.goal.anchorKind,
			Tags:         append([]string(nil), liveNPC.goal.location.Tags...),
		}
	}
	if liveNPC.route != nil {
		debugNPC.Route = &npcDebugRoute{
			StepIndex: liveNPC.routeStep,
			PathIndex: liveNPC.pathIndex,
			StepCount: len(liveNPC.route.Steps),
		}
		if liveNPC.routeStep >= 0 && liveNPC.routeStep < len(liveNPC.route.Steps) {
			step := liveNPC.route.Steps[liveNPC.routeStep]
			debugNPC.Route.CurrentMapID = step.MapID
			debugNPC.Route.CurrentPortalID = step.PortalID
			debugNPC.Route.TargetMapID = step.TargetMapID
		}
	}
	return debugNPC
}

func (room *room) npcProductionDebugSnapshotLocked(liveNPC *liveNPC) npcDebugProduction {
	job, ok := liveNPC.jobDefinition()
	if !ok {
		return npcDebugProduction{Status: "no_job", Blocker: "no_job"}
	}
	status := "building_progress"
	blocker := ""
	eligible := room.npcAtWorkAnchorLocked(liveNPC)
	if liveNPC.anchors.Work == nil {
		status = "blocked"
		blocker = "no_work_anchor"
	} else if !eligible {
		status = "blocked"
		blocker = "not_at_work_anchor"
	} else if room.npcProductionCharacterIDLocked(liveNPC) < 1 {
		status = "blocked"
		blocker = "missing_character_identity"
	} else if liveNPC.jobProduction.Progress >= npcJobProductionInterval.Seconds() {
		status = "ready_to_queue"
	} else {
		status = "eligible"
	}
	return npcDebugProduction{
		Eligible:          eligible,
		Status:            status,
		Blocker:           blocker,
		JobKey:            job.JobKey,
		OutputKey:         job.OutputKey,
		ProgressSeconds:   liveNPC.jobProduction.Progress,
		LastAt:            formatDebugTime(liveNPC.jobProduction.LastAt),
		LastEvent:         liveNPC.jobProduction.LastEvent,
		LastCommitAt:      formatDebugTime(liveNPC.jobProduction.LastCommitAt),
		LastCommitStatus:  liveNPC.jobProduction.LastCommitStatus,
		LastBlockedReason: liveNPC.jobProduction.LastBlockedReason,
		LastCommitError:   liveNPC.jobProduction.LastCommitError,
	}
}

func (room *room) npcProductionCharacterIDLocked(liveNPC *liveNPC) int64 {
	if liveNPC == nil {
		return 0
	}
	if liveNPC.characterID > 0 {
		return liveNPC.characterID
	}
	if room.world == nil {
		return 0
	}
	character, ok := room.world.npcCharacter(liveNPC.npcKey)
	if !ok {
		return 0
	}
	return character.characterID
}

func (npc *liveNPC) debugRoutine(now time.Time, dayLength time.Duration) npcDebugRoutine {
	routine := npcDebugRoutine{
		Status:               "idle",
		MostRecentFailureKey: npc.mostRecentFailedTargetKey(),
	}
	if npc.goal != nil {
		routine.TargetKey = npc.goal.failureKey()
		routine.CurrentFailureKey = routine.TargetKey
		routine.RouteBlocked = npc.targetFailedRecentlyDebug(*npc.goal, now)
		routine.Scheduled = npc.scheduleDrivePressure(npc.goal.drive, now, dayLength) > 0
		switch {
		case npc.route != nil:
			routine.Status = "traveling"
		case !npc.goalArrivedAt.IsZero():
			routine.Status = "arrived"
		default:
			routine.Status = "targeting"
		}
		return routine
	}
	if routine.MostRecentFailureKey != "" {
		routine.Status = "blocked"
		routine.RouteBlocked = true
	}
	return routine
}

func (npc *liveNPC) targetFailedRecentlyDebug(goal npcGoal, now time.Time) bool {
	if len(npc.failedTargets) == 0 {
		return false
	}
	failedAt, ok := npc.failedTargets[goal.failureKey()]
	return ok && now.Sub(failedAt) < npcFailedTargetCooldown
}

func (npc *liveNPC) debugSchedule(now time.Time, dayLength time.Duration) npcDebugSchedule {
	debugSchedule := npcDebugSchedule{
		Phase: string(npcSchedulePhaseAt(now, dayLength)),
	}
	for _, drive := range allNPCDrives {
		pressure := npc.scheduleDrivePressure(drive, now, dayLength)
		if pressure <= 0 {
			continue
		}
		debugSchedule.Pressures = append(debugSchedule.Pressures, npcDebugDrivePressure{
			Drive:          string(drive),
			Pressure:       pressure,
			Value:          npc.driveValue(drive),
			SelectionValue: npc.driveSelectionValue(drive, now, dayLength),
		})
	}
	return debugSchedule
}

func debugAnchors(anchors npcRoutineAnchors) npcDebugAnchors {
	return npcDebugAnchors{
		Home:   debugAnchor(anchors.Home),
		Work:   debugAnchor(anchors.Work),
		Food:   debugAnchor(anchors.Food),
		Social: debugAnchor(anchors.Social),
	}
}

func debugAnchor(anchor *npcLocationAnchor) *npcDebugAnchor {
	if anchor == nil {
		return nil
	}
	return &npcDebugAnchor{
		Kind:         anchor.Kind,
		MapID:        anchor.MapID,
		LocationID:   anchor.LocationID,
		LocationName: anchor.LocationName,
		Source:       anchor.Source,
		Tags:         append([]string(nil), anchor.Tags...),
	}
}

func (npc *liveNPC) failedTargetsDebugSnapshot() []npcDebugFailedTarget {
	if len(npc.failedTargets) == 0 {
		return nil
	}
	keys := make([]string, 0, len(npc.failedTargets))
	for key := range npc.failedTargets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	targets := make([]npcDebugFailedTarget, 0, len(keys))
	for _, key := range keys {
		failedAt := npc.failedTargets[key]
		drive, mapID, locationID := splitFailedTargetKey(key)
		targets = append(targets, npcDebugFailedTarget{
			Key:        key,
			Drive:      drive,
			MapID:      mapID,
			LocationID: locationID,
			FailedAt:   formatDebugTime(failedAt),
			RetryAt:    formatDebugTime(failedAt.Add(npcFailedTargetCooldown)),
		})
	}
	return targets
}

func (npc *liveNPC) mostRecentFailedTargetKey() string {
	if len(npc.failedTargets) == 0 {
		return ""
	}
	var newestKey string
	var newestAt time.Time
	for key, failedAt := range npc.failedTargets {
		if newestKey == "" || failedAt.After(newestAt) || (failedAt.Equal(newestAt) && key < newestKey) {
			newestKey = key
			newestAt = failedAt
		}
	}
	return newestKey
}

func splitFailedTargetKey(key string) (string, string, string) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}

func formatDebugTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
