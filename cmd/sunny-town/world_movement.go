package main

import (
	"math"
	"time"

	stmaps "hq/internal/sunnytown/maps"
)

func (room *room) updateMove(playerID string, seq int64, x float64, y float64, facing string, moving bool, now time.Time) {
	var triggered *portal
	room.mu.Lock()
	if player := room.players[playerID]; player != nil {
		if seq <= player.lastMoveSeq {
			room.mu.Unlock()
			return
		}
		acceptedX, acceptedY, ok := room.acceptedMoveLocked(player, x, y)
		if ok {
			player.x = acceptedX
			player.y = acceptedY
			player.lastMoveAt = now
		}
		if stmaps.IsFacing(facing) {
			player.facing = facing
		}
		player.moving = moving && ok
		player.lastMoveSeq = seq
		if ok {
			currentPortal := room.portalForPlayerLocked(player)
			if currentPortal == nil {
				player.portalLocked = false
			} else if !player.portalLocked {
				triggered = currentPortal
			}
		}
	}
	room.mu.Unlock()

	if triggered != nil {
		room.world.transferPlayer(room.gameMap.ID, playerID, *triggered, seq, now)
	}
}

func (room *room) acceptedMoveLocked(player *player, proposedX float64, proposedY float64) (float64, float64, bool) {
	if math.IsNaN(proposedX) || math.IsNaN(proposedY) || math.IsInf(proposedX, 0) || math.IsInf(proposedY, 0) {
		return player.x, player.y, false
	}

	x := room.clampXLocked(proposedX)
	y := room.clampYLocked(proposedY)
	if room.collidesLocked(x, y) {
		x = player.x
		y = player.y
	}
	return x, y, true
}

func (room *room) spawnPointLocked() point {
	if len(room.gameMap.Spawns) == 0 {
		return point{X: 64, Y: 64}
	}
	index := len(room.players) % len(room.gameMap.Spawns)
	return room.gameMap.Spawns[index]
}

func (room *room) collidesLocked(x float64, y float64) bool {
	playerRect := rect{
		X:      x - playerSize/2,
		Y:      y - playerSize/2,
		Width:  playerSize,
		Height: playerSize,
	}
	for _, blocked := range room.gameMap.BlockedRects {
		if rectsOverlap(playerRect, blocked) {
			return true
		}
	}
	for _, object := range room.worldObjects {
		if object.active && object.collision && rectsOverlap(playerRect, object.rect()) {
			return true
		}
	}
	return false
}

func (room *room) clampXLocked(x float64) float64 {
	return room.clampX(x)
}

func (room *room) clampYLocked(y float64) float64 {
	return room.clampY(y)
}

func (room *room) clampX(x float64) float64 {
	maxX := float64(room.gameMap.Width*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxX, x))
}

func (room *room) clampY(y float64) float64 {
	maxY := float64(room.gameMap.Height*room.gameMap.TileSize) - playerSize/2
	return math.Max(playerSize/2, math.Min(maxY, y))
}

func (room *room) portalForPlayerLocked(player *player) *portal {
	playerRect := rect{
		X:      player.x - playerSize/2,
		Y:      player.y - playerSize/2,
		Width:  playerSize,
		Height: playerSize,
	}
	for index := range room.gameMap.Portals {
		portal := room.gameMap.Portals[index]
		if rectsOverlap(playerRect, rect{
			X:      portal.X,
			Y:      portal.Y,
			Width:  portal.Width,
			Height: portal.Height,
		}) {
			return &room.gameMap.Portals[index]
		}
	}
	return nil
}

func rectsOverlap(a rect, b rect) bool {
	return a.X < b.X+b.Width &&
		a.X+a.Width > b.X &&
		a.Y < b.Y+b.Height &&
		a.Y+a.Height > b.Y
}
