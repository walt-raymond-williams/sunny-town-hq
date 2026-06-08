package server

import (
	"math"
	"time"

	stnavigation "hq/internal/sunnytown/navigation"
)

const scriptedSmokeNPCKey = "mayor-sunny"
const scriptedSmokeLocationID = "town-square-center"

func (room *room) configureScriptedNPCsLocked() {
	npc := room.liveNPCs[scriptedSmokeNPCKey]
	if npc == nil || room.world == nil || room.world.navigation == nil {
		return
	}
	if _, ok := room.world.navigation.Location(room.gameMap.ID, scriptedSmokeLocationID); !ok {
		return
	}
	npc.scriptedTargets = []scriptedNPCTarget{
		{
			mapID:      room.gameMap.ID,
			locationID: scriptedSmokeLocationID,
		},
		{
			mapID: room.gameMap.ID,
			x:     npc.x,
			y:     npc.y,
		},
	}
}

func (room *room) stepLiveNPCsLocked(dt float64, now time.Time) {
	if dt <= 0 || room.world == nil || room.world.navigation == nil {
		return
	}
	for _, npc := range room.liveNPCs {
		room.stepLiveNPCLocked(npc, dt, now)
	}
}

func (room *room) stepLiveNPCLocked(npc *liveNPC, dt float64, now time.Time) {
	if npc == nil || len(npc.scriptedTargets) == 0 {
		return
	}
	if now.Before(npc.scriptedWaitUntil) {
		npc.moving = false
		return
	}
	if npc.route == nil && !room.planLiveNPCRouteLocked(npc) {
		npc.moving = false
		return
	}

	remaining := npcSpeed * dt
	for remaining > 0 && npc.route != nil {
		step := npc.route.Steps[npc.routeStep]
		if step.MapID != room.gameMap.ID || step.PortalID != "" {
			npc.moving = false
			npc.route = nil
			npc.routeBlocked = true
			return
		}
		if npc.pathIndex >= len(step.Path) {
			room.advanceLiveNPCRouteLocked(npc, now)
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
				room.blockLiveNPCRoute(npc)
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
			room.blockLiveNPCRoute(npc)
			return
		}
		npc.x = nextX
		npc.y = nextY
		npc.moving = true
		remaining = 0
	}
}

func (room *room) planLiveNPCRouteLocked(npc *liveNPC) bool {
	if npc.routeBlocked || room.world == nil || room.world.navigation == nil {
		return false
	}
	target := npc.scriptedTargets[npc.scriptedTarget]
	start := stnavigation.Point{X: npc.x, Y: npc.y}
	var (
		route stnavigation.Route
		err   error
	)
	if target.locationID != "" {
		route, err = room.world.navigation.PlanRouteToLocation(room.gameMap.ID, start, target.mapID, target.locationID)
	} else {
		route, err = room.world.navigation.PlanRoute(room.gameMap.ID, start, target.mapID, stnavigation.Point{X: target.x, Y: target.y})
	}
	if err != nil || len(route.Steps) == 0 {
		npc.routeBlocked = true
		return false
	}
	npc.route = &route
	npc.routeStep = 0
	npc.pathIndex = firstWaypointIndex(route.Steps[0].Path)
	return true
}

func (room *room) advanceLiveNPCRouteLocked(npc *liveNPC, now time.Time) {
	npc.routeStep++
	if npc.route == nil || npc.routeStep >= len(npc.route.Steps) {
		npc.route = nil
		npc.routeStep = 0
		npc.pathIndex = 0
		npc.routeBlocked = false
		npc.moving = false
		npc.scriptedTarget = (npc.scriptedTarget + 1) % len(npc.scriptedTargets)
		npc.scriptedWaitUntil = now.Add(npcScriptedPause)
		return
	}
	npc.pathIndex = firstWaypointIndex(npc.route.Steps[npc.routeStep].Path)
}

func firstWaypointIndex(path []stnavigation.Point) int {
	if len(path) > 1 {
		return 1
	}
	return 0
}

func (room *room) blockLiveNPCRoute(npc *liveNPC) {
	npc.moving = false
	npc.route = nil
	npc.routeStep = 0
	npc.pathIndex = 0
	npc.routeBlocked = true
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
