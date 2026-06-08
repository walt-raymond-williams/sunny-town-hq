package server

import (
	"math"
	"sort"
	"strings"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
)

const (
	driveControlledNPCKey = "mayor-sunny"
	driveStartLocationID  = "town-square-center"
)

var allNPCDrives = []npcDrive{npcDriveHunger, npcDriveEnergy, npcDriveSocial, npcDriveWork}

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

func (room *room) stepLiveNPCsLocked(dt float64) {
	if dt <= 0 || room.world == nil || room.world.navigation == nil {
		return
	}
	for _, npc := range room.liveNPCs {
		room.stepLiveNPCLocked(npc, dt)
	}
}

func (room *room) stepLiveNPCLocked(npc *liveNPC, dt float64) {
	if npc == nil {
		return
	}
	npc.depleteDrives(dt)
	room.replenishNPCDrivesLocked(npc, dt)
	room.completeNPCGoalIfSatisfiedLocked(npc)

	if npc.route == nil && npc.goal == nil {
		room.chooseNPCDriveGoalLocked(npc)
	}
	if npc.route == nil {
		npc.moving = false
		return
	}

	remaining := npcSpeed * dt
	for remaining > 0 && npc.route != nil {
		step := npc.route.Steps[npc.routeStep]
		if step.MapID != room.gameMap.ID || step.PortalID != "" {
			room.clearNPCGoal(npc)
			return
		}
		if npc.pathIndex >= len(step.Path) {
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
				return
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
			return
		}
		npc.x = nextX
		npc.y = nextY
		npc.moving = true
		remaining = 0
	}
}

func (room *room) chooseNPCDriveGoalLocked(npc *liveNPC) {
	for _, drive := range npc.drivesByUrgency() {
		if npc.driveValue(drive) >= npcDriveThreshold {
			continue
		}
		goal, route, ok := room.routeToDriveLocationLocked(npc, drive)
		if !ok {
			continue
		}
		npc.activeDrive = drive
		npc.goal = &goal
		npc.route = &route
		npc.routeStep = 0
		npc.pathIndex = firstWaypointIndex(route.Steps[0].Path)
		return
	}
}

func (room *room) routeToDriveLocationLocked(npc *liveNPC, drive npcDrive) (npcGoal, stnavigation.Route, bool) {
	start := stnavigation.Point{X: npc.x, Y: npc.y}
	for _, location := range room.gameMap.Locations {
		if !locationMatchesDrive(location, drive) {
			continue
		}
		if pointWithinLocation(start, location) {
			return npcGoal{}, stnavigation.Route{}, false
		}
		route, err := room.world.navigation.PlanRouteToLocation(room.gameMap.ID, start, room.gameMap.ID, location.ID)
		if err != nil || len(route.Steps) == 0 {
			continue
		}
		return npcGoal{
			drive:    drive,
			mapID:    room.gameMap.ID,
			location: location,
		}, route, true
	}
	return npcGoal{}, stnavigation.Route{}, false
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
	if npc.driveValue(npc.goal.drive) < npcDriveThreshold {
		return
	}
	if !pointWithinLocation(stnavigation.Point{X: npc.x, Y: npc.y}, npc.goal.location) {
		return
	}
	room.clearNPCGoal(npc)
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

func (npc *liveNPC) drivesByUrgency() []npcDrive {
	drives := append([]npcDrive(nil), allNPCDrives...)
	sort.SliceStable(drives, func(i int, j int) bool {
		return npc.driveValue(drives[i]) < npc.driveValue(drives[j])
	})
	return drives
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

func locationMatchesDrive(location stmaps.Location, drive npcDrive) bool {
	for _, tag := range location.Tags {
		if npcDriveLocationTags[drive][strings.ToLower(strings.TrimSpace(tag))] {
			return true
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
