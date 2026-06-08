package server

import (
	"encoding/json"
	"net/http"
	"sort"
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
	Key      string `json:"key"`
	FailedAt string `json:"failedAt"`
	RetryAt  string `json:"retryAt"`
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
	publicSnapshot := liveNPC.snapshot(character, hasCharacter)
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
		targets = append(targets, npcDebugFailedTarget{
			Key:      key,
			FailedAt: formatDebugTime(failedAt),
			RetryAt:  formatDebugTime(failedAt.Add(npcFailedTargetCooldown)),
		})
	}
	return targets
}

func formatDebugTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
