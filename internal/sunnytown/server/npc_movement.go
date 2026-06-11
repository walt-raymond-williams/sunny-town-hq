package server

import (
	"math"
	"sort"
	"strings"
	"time"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
)

const (
	driveControlledNPCKey = "mayor-sunny"
	driveStartLocationID  = "town-square-center"
)

type npcTransfer struct {
	npc         *liveNPC
	sourceMapID string
	targetMapID string
	x           float64
	y           float64
	facing      string
}

var allNPCDrives = []npcDrive{npcDriveHunger, npcDriveEnergy, npcDriveSocial, npcDriveWork}

const (
	npcLocationCurrentMapBonus = 25.0
	npcLocationOwnerBonus      = 200.0
	npcLocationRoleBonus       = 75.0
	npcLocationAnchorBonus     = 500.0
)

var npcDriveLocationTags = map[npcDrive]map[string]bool{
	npcDriveHunger: {
		"food":        true,
		"meal":        true,
		"kitchen":     true,
		"food_source": true,
	},
	npcDriveEnergy: {
		"bed":   true,
		"sleep": true,
		"rest":  true,
		"home":  true,
	},
	npcDriveSocial: {
		"social":    true,
		"public":    true,
		"gathering": true,
	},
	npcDriveWork: {
		"work": true,
	},
	npcDriveIdle: {
		"idle":   true,
		"wander": true,
		"public": true,
		"social": true,
	},
}

func (room *room) configureNPCBehaviorLocked() {
	npc := room.liveNPCs[driveControlledNPCKey]
	if npc == nil || room.world == nil || room.world.navigation == nil {
		return
	}
	if _, ok := room.world.navigation.Location(room.gameMap.ID, driveStartLocationID); !ok {
		return
	}
	npc.drives.Social = 40
}

func (room *room) stepLiveNPCsLocked(dt float64, now time.Time) []npcTransfer {
	if dt <= 0 || room.world == nil || room.world.navigation == nil {
		return nil
	}
	transfers := []npcTransfer{}
	for _, npc := range room.liveNPCs {
		if transfer := room.stepLiveNPCLocked(npc, dt, now); transfer != nil {
			transfers = append(transfers, *transfer)
		}
	}
	return transfers
}

func (room *room) catchUpNPCsAfterNoPlayersLocked(now time.Time) {
	if room.npcPausedAt.IsZero() || !now.After(room.npcPausedAt) {
		return
	}
	elapsed := now.Sub(room.npcPausedAt)
	if elapsed > npcNoPlayerCatchUpMax {
		elapsed = npcNoPlayerCatchUpMax
	}
	if elapsed <= 0 {
		room.npcPausedAt = time.Time{}
		return
	}
	dt := elapsed.Seconds()
	productionDT := dt
	if productionDT > npcJobProductionCatchUpMax.Seconds() {
		productionDT = npcJobProductionCatchUpMax.Seconds()
	}
	var productionEvents []npcJobProductionEvent
	for _, npc := range room.liveNPCs {
		if npc == nil {
			continue
		}
		npc.depleteDrives(dt)
		room.replenishNPCDrivesLocked(npc, dt)
		if event, ok := room.advanceNPCJobProductionLocked(npc, productionDT, now); ok {
			productionEvents = append(productionEvents, event)
		}
		room.clearNPCGoal(npc)
	}
	room.npcPausedAt = time.Time{}
	queueNPCJobProductionEvents(room.npcJobEvents, productionEvents)
}

func (room *room) stepLiveNPCLocked(npc *liveNPC, dt float64, now time.Time) *npcTransfer {
	if npc == nil {
		return nil
	}
	npc.depleteDrives(dt)
	room.replenishNPCDrivesLocked(npc, dt)
	room.completeNPCGoalIfSatisfiedLocked(npc)
	room.failNPCGoalIfStaleLocked(npc, now)
	room.reevaluateNPCGoalLocked(npc, now)

	if npc.route == nil && npc.goal == nil {
		room.chooseNPCDriveGoalLocked(npc, now)
	}
	if npc.route == nil {
		npc.moving = false
		room.noteNPCGoalArrivalLocked(npc, now)
		return nil
	}

	remaining := npcSpeed * dt
	for remaining > 0 && npc.route != nil {
		step := npc.route.Steps[npc.routeStep]
		if step.MapID != room.gameMap.ID {
			room.clearNPCGoal(npc)
			return nil
		}
		if npc.pathIndex >= len(step.Path) {
			if step.PortalID != "" {
				return room.transferNPCThroughPortalLocked(npc, step)
			}
			room.advanceLiveNPCRouteLocked(npc)
			continue
		}

		waypoint := step.Path[npc.pathIndex]
		dx := waypoint.X - npc.x
		dy := waypoint.Y - npc.y
		distance := math.Hypot(dx, dy)
		if distance <= 0.001 {
			npc.x = waypoint.X
			npc.y = waypoint.Y
			npc.pathIndex++
			continue
		}

		npc.facing = facingForDelta(dx, dy, npc.facing)
		if remaining >= distance {
			if room.collidesLocked(waypoint.X, waypoint.Y) {
				room.clearNPCGoal(npc)
				return nil
			}
			npc.x = waypoint.X
			npc.y = waypoint.Y
			npc.pathIndex++
			remaining -= distance
			npc.moving = true
			continue
		}

		ratio := remaining / distance
		nextX := npc.x + dx*ratio
		nextY := npc.y + dy*ratio
		if room.collidesLocked(nextX, nextY) {
			room.clearNPCGoal(npc)
			return nil
		}
		npc.x = nextX
		npc.y = nextY
		npc.moving = true
		remaining = 0
	}
	room.noteNPCGoalArrivalLocked(npc, now)
	return nil
}

func (room *room) chooseNPCDriveGoalLocked(npc *liveNPC, now time.Time) {
	goal, route, ok := room.nextNPCDriveGoalLocked(npc, now, false)
	if !ok {
		goal, route, ok = room.nextNPCFallbackGoalLocked(npc, now)
	}
	if !ok {
		return
	}
	room.assignNPCGoal(npc, goal, route, now)
}

func (room *room) nextNPCDriveGoalLocked(npc *liveNPC, now time.Time, emergencyOnly bool) (npcGoal, stnavigation.Route, bool) {
	dayLength := room.scheduleDayLength()
	for _, drive := range npc.drivesByUrgency(now, dayLength) {
		if emergencyOnly && npc.driveValue(drive) >= npcEmergencyDriveThreshold {
			continue
		}
		if !emergencyOnly && npc.driveSelectionValue(drive, now, dayLength) >= npcDriveThreshold {
			continue
		}
		goal, route, ok := room.routeToDriveLocationLocked(npc, drive, now)
		if !ok {
			continue
		}
		return goal, route, true
	}
	return npcGoal{}, stnavigation.Route{}, false
}

func (room *room) nextNPCFallbackGoalLocked(npc *liveNPC, now time.Time) (npcGoal, stnavigation.Route, bool) {
	return room.routeToDriveLocationLocked(npc, npcDriveIdle, now)
}

func (room *room) routeToDriveLocationLocked(npc *liveNPC, drive npcDrive, now time.Time) (npcGoal, stnavigation.Route, bool) {
	type scoredDriveLocation struct {
		goal  npcGoal
		route stnavigation.Route
		score float64
	}

	start := stnavigation.Point{X: npc.x, Y: npc.y}
	candidates := []scoredDriveLocation{}
	for _, gameMap := range room.driveTargetMaps() {
		for _, location := range gameMap.Locations {
			if !locationMatchesDrive(location, drive) {
				continue
			}
			goal := npcGoal{drive: drive, mapID: gameMap.ID, location: location}
			if anchor := npc.anchors.matchingAnchor(drive, gameMap.ID, location.ID); anchor != nil {
				goal.anchorKind = anchor.Kind
			}
			if npc.targetFailedRecently(goal, now) {
				continue
			}
			if gameMap.ID == room.gameMap.ID && pointWithinLocation(start, location) {
				return npcGoal{}, stnavigation.Route{}, false
			}
			route, err := room.world.navigation.PlanRouteToLocationWithBlockedRects(room.gameMap.ID, start, gameMap.ID, location.ID, room.npcRouteBlockedRectsLocked())
			if err != nil || len(route.Steps) == 0 {
				npc.markTargetFailed(goal, now)
				continue
			}
			candidates = append(candidates, scoredDriveLocation{
				goal:  goal,
				route: route,
				score: room.scoreNPCDriveLocation(npc, drive, gameMap.ID, location, route),
			})
		}
	}
	if len(candidates) == 0 {
		return npcGoal{}, stnavigation.Route{}, false
	}
	sort.SliceStable(candidates, func(i int, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.score != right.score {
			return left.score > right.score
		}
		if left.goal.mapID != right.goal.mapID {
			return left.goal.mapID < right.goal.mapID
		}
		return left.goal.location.ID < right.goal.location.ID
	})
	return candidates[0].goal, candidates[0].route, true
}

func (room *room) scoreNPCDriveLocation(npc *liveNPC, drive npcDrive, mapID string, location stmaps.Location, route stnavigation.Route) float64 {
	score := 0.0
	if drive != npcDriveIdle {
		score = 100 - npc.driveValue(drive)
	}
	score -= routePathCost(route)
	if mapID == room.gameMap.ID {
		score += npcLocationCurrentMapBonus
	}
	if location.OwnerNPCKey != "" && location.OwnerNPCKey == npc.npcKey {
		score += npcLocationOwnerBonus
	}
	if locationRoleMatchesNPC(npc, location) {
		score += npcLocationRoleBonus
	}
	if npc.anchors.matchingAnchor(drive, mapID, location.ID) != nil {
		score += npcLocationAnchorBonus
	}
	return score
}

func routePathCost(route stnavigation.Route) float64 {
	var cost float64
	for _, step := range route.Steps {
		if len(step.Path) < 2 {
			continue
		}
		for index := 1; index < len(step.Path); index++ {
			from := step.Path[index-1]
			to := step.Path[index]
			cost += math.Hypot(to.X-from.X, to.Y-from.Y)
		}
	}
	return cost
}

func (room *room) npcRouteBlockedRectsLocked() map[string][]stnavigation.Rect {
	blockedRects := room.activeCollisionRectsLocked()
	if len(blockedRects) == 0 {
		return nil
	}
	return map[string][]stnavigation.Rect{
		room.gameMap.ID: blockedRects,
	}
}

func (room *room) activeCollisionRectsLocked() []stnavigation.Rect {
	blockedRects := []stnavigation.Rect{}
	for _, object := range room.worldObjects {
		if !object.active || !object.collision {
			continue
		}
		blockedRects = append(blockedRects, object.rect())
	}
	return blockedRects
}

func (room *room) driveTargetMaps() []gameMap {
	if room.world == nil {
		return []gameMap{room.gameMap}
	}
	mapIDs := make([]string, 0, len(room.world.rooms))
	for mapID := range room.world.rooms {
		if mapID != room.gameMap.ID {
			mapIDs = append(mapIDs, mapID)
		}
	}
	sort.Strings(mapIDs)

	gameMaps := []gameMap{room.gameMap}
	for _, mapID := range mapIDs {
		target := room.world.rooms[mapID]
		if target == nil {
			continue
		}
		gameMaps = append(gameMaps, target.gameMap)
	}
	return gameMaps
}

func (room *room) transferNPCThroughPortalLocked(npc *liveNPC, step stnavigation.RouteStep) *npcTransfer {
	nextStepIndex := npc.routeStep + 1
	if npc.route == nil || step.TargetMapID == "" || nextStepIndex >= len(npc.route.Steps) {
		room.clearNPCGoal(npc)
		return nil
	}
	target := room.world.rooms[step.TargetMapID]
	if target == nil {
		room.clearNPCGoal(npc)
		return nil
	}
	nextStep := npc.route.Steps[nextStepIndex]
	if nextStep.MapID != step.TargetMapID {
		room.clearNPCGoal(npc)
		return nil
	}
	delete(room.liveNPCs, npc.npcKey)
	npc.mapID = step.TargetMapID
	npc.x = nextStep.From.X
	npc.y = nextStep.From.Y
	if stmaps.IsFacing(step.TargetFacing) {
		npc.facing = step.TargetFacing
	}
	npc.moving = false
	npc.routeStep = nextStepIndex
	npc.pathIndex = firstWaypointIndex(nextStep.Path)
	return &npcTransfer{
		npc:         npc,
		sourceMapID: room.gameMap.ID,
		targetMapID: step.TargetMapID,
		x:           npc.x,
		y:           npc.y,
		facing:      npc.facing,
	}
}

func (world *world) applyNPCTransfers(transfers []npcTransfer, now time.Time) {
	for _, transfer := range transfers {
		if transfer.npc == nil {
			continue
		}
		target := world.rooms[transfer.targetMapID]
		if target == nil {
			continue
		}
		target.mu.Lock()
		transfer.npc.mapID = transfer.targetMapID
		transfer.npc.x = transfer.x
		transfer.npc.y = transfer.y
		transfer.npc.facing = transfer.facing
		target.liveNPCs[transfer.npc.npcKey] = transfer.npc
		target.mu.Unlock()

		if source := world.rooms[transfer.sourceMapID]; source != nil {
			source.broadcastSnapshot(now)
		}
		target.broadcastSnapshot(now)
	}
}

func (room *room) assignNPCGoal(npc *liveNPC, goal npcGoal, route stnavigation.Route, now time.Time) {
	npc.activeDrive = goal.drive
	npc.goal = &goal
	npc.route = &route
	npc.routeStep = 0
	npc.pathIndex = firstWaypointIndex(route.Steps[0].Path)
	npc.goalStartedAt = now
	npc.focusUntil = now.Add(npcGoalFocusDuration)
	npc.reevaluateAt = now.Add(npcGoalFocusDuration + npcGoalReevaluateInterval)
	npc.goalArrivedAt = time.Time{}
}

func (room *room) replenishNPCDrivesLocked(npc *liveNPC, dt float64) {
	position := stnavigation.Point{X: npc.x, Y: npc.y}
	for _, location := range room.gameMap.Locations {
		if !pointWithinLocation(position, location) {
			continue
		}
		for _, drive := range allNPCDrives {
			if locationMatchesDrive(location, drive) {
				npc.setDriveValue(drive, npc.driveValue(drive)+npcDriveReplenishPerSecond*dt)
			}
		}
	}
}

func (room *room) completeNPCGoalIfSatisfiedLocked(npc *liveNPC) {
	if npc.goal == nil {
		return
	}
	if npc.goal.drive == npcDriveIdle {
		return
	}
	if npc.driveValue(npc.goal.drive) < npcDriveThreshold {
		return
	}
	if !pointWithinLocation(stnavigation.Point{X: npc.x, Y: npc.y}, npc.goal.location) {
		return
	}
	room.clearNPCGoal(npc)
}

func (room *room) failNPCGoalIfStaleLocked(npc *liveNPC, now time.Time) {
	if npc.goal == nil || npc.route != nil {
		return
	}
	if npc.goal.drive == npcDriveIdle {
		return
	}
	if !pointWithinLocation(stnavigation.Point{X: npc.x, Y: npc.y}, npc.goal.location) {
		return
	}
	if npc.goalArrivedAt.IsZero() || now.Sub(npc.goalArrivedAt) < npcGoalGraceDuration {
		return
	}
	if npc.driveValue(npc.goal.drive) >= npcDriveThreshold {
		return
	}
	if npc.driveValue(npc.goal.drive) > npc.goalArriveDrive+0.001 {
		return
	}
	npc.markTargetFailed(*npc.goal, now)
	room.clearNPCGoal(npc)
}

func (room *room) reevaluateNPCGoalLocked(npc *liveNPC, now time.Time) {
	if npc.goal == nil || now.Before(npc.focusUntil) || now.Before(npc.reevaluateAt) {
		return
	}
	npc.reevaluateAt = now.Add(npcGoalReevaluateInterval)
	emergencyOnly := npc.goal.drive != npcDriveIdle
	goal, route, ok := room.nextNPCDriveGoalLocked(npc, now, emergencyOnly)
	if !ok {
		return
	}
	if npc.goal != nil && npc.goal.failureKey() == goal.failureKey() {
		return
	}
	room.assignNPCGoal(npc, goal, route, now)
}

func (room *room) noteNPCGoalArrivalLocked(npc *liveNPC, now time.Time) {
	if npc.goal == nil || npc.route != nil || !npc.goalArrivedAt.IsZero() {
		return
	}
	if pointWithinLocation(stnavigation.Point{X: npc.x, Y: npc.y}, npc.goal.location) {
		npc.goalArrivedAt = now
		npc.goalArriveDrive = npc.driveValue(npc.goal.drive)
	}
}

func (room *room) advanceLiveNPCRouteLocked(npc *liveNPC) {
	npc.routeStep++
	if npc.route == nil || npc.routeStep >= len(npc.route.Steps) {
		npc.route = nil
		npc.routeStep = 0
		npc.pathIndex = 0
		npc.moving = false
		return
	}
	npc.pathIndex = firstWaypointIndex(npc.route.Steps[npc.routeStep].Path)
}

func (room *room) clearNPCGoal(npc *liveNPC) {
	npc.moving = false
	npc.activeDrive = ""
	npc.goal = nil
	npc.route = nil
	npc.routeStep = 0
	npc.pathIndex = 0
	npc.goalStartedAt = time.Time{}
	npc.focusUntil = time.Time{}
	npc.reevaluateAt = time.Time{}
	npc.goalArrivedAt = time.Time{}
	npc.goalArriveDrive = 0
}

func (drives npcDrives) withDefaults() npcDrives {
	if drives.Hunger == 0 {
		drives.Hunger = npcDriveDefault
	}
	if drives.Energy == 0 {
		drives.Energy = npcDriveDefault
	}
	if drives.Social == 0 {
		drives.Social = npcDriveDefault
	}
	if drives.Work == 0 {
		drives.Work = npcDriveDefault
	}
	return drives
}

func (npc *liveNPC) depleteDrives(dt float64) {
	for _, drive := range allNPCDrives {
		npc.setDriveValue(drive, npc.driveValue(drive)-npcDriveDepletePerSecond*dt)
	}
}

func (npc *liveNPC) driveValue(drive npcDrive) float64 {
	switch drive {
	case npcDriveHunger:
		return npc.drives.Hunger
	case npcDriveEnergy:
		return npc.drives.Energy
	case npcDriveSocial:
		return npc.drives.Social
	case npcDriveWork:
		return npc.drives.Work
	default:
		return 0
	}
}

func (npc *liveNPC) setDriveValue(drive npcDrive, value float64) {
	value = math.Max(0, math.Min(100, value))
	switch drive {
	case npcDriveHunger:
		npc.drives.Hunger = value
	case npcDriveEnergy:
		npc.drives.Energy = value
	case npcDriveSocial:
		npc.drives.Social = value
	case npcDriveWork:
		npc.drives.Work = value
	}
}

func (npc *liveNPC) targetFailedRecently(goal npcGoal, now time.Time) bool {
	if len(npc.failedTargets) == 0 {
		return false
	}
	failedAt, ok := npc.failedTargets[goal.failureKey()]
	if !ok {
		return false
	}
	if now.Sub(failedAt) < npcFailedTargetCooldown {
		return true
	}
	delete(npc.failedTargets, goal.failureKey())
	return false
}

func (npc *liveNPC) markTargetFailed(goal npcGoal, now time.Time) {
	if npc.failedTargets == nil {
		npc.failedTargets = map[string]time.Time{}
	}
	npc.failedTargets[goal.failureKey()] = now
	npc.failureCount++
}

func (goal npcGoal) failureKey() string {
	return string(goal.drive) + ":" + goal.mapID + ":" + goal.location.ID
}

func locationMatchesDrive(location stmaps.Location, drive npcDrive) bool {
	for _, tag := range location.Tags {
		if npcDriveLocationTags[drive][strings.ToLower(strings.TrimSpace(tag))] {
			return true
		}
	}
	return false
}

func locationRoleMatchesNPC(npc *liveNPC, location stmaps.Location) bool {
	if npc == nil {
		return false
	}
	tags := map[string]bool{}
	for _, tag := range location.Tags {
		tags[strings.ToLower(strings.TrimSpace(tag))] = true
	}
	if npc.shop != nil && (tags["shop"] || tags["merchant"] || tags["work"]) {
		return true
	}
	if npc.activity != nil {
		switch npc.activity.Type {
		case "schoolwork":
			return tags["school"] || tags["work"]
		}
	}
	return false
}

func pointWithinLocation(point stnavigation.Point, location stmaps.Location) bool {
	return math.Hypot(point.X-location.X, point.Y-location.Y) <= location.Radius
}

func firstWaypointIndex(path []stnavigation.Point) int {
	if len(path) > 1 {
		return 1
	}
	return 0
}

func facingForDelta(dx float64, dy float64, fallback string) string {
	if math.Abs(dx) >= math.Abs(dy) {
		if dx < 0 {
			return "left"
		}
		if dx > 0 {
			return "right"
		}
	} else {
		if dy < 0 {
			return "up"
		}
		if dy > 0 {
			return "down"
		}
	}
	return fallback
}
