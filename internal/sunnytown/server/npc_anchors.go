package server

import (
	"math"
	"sort"
	"strings"

	stmaps "hq/internal/sunnytown/maps"
	stnavigation "hq/internal/sunnytown/navigation"
)

const (
	npcAnchorHome   = "home"
	npcAnchorWork   = "work"
	npcAnchorFood   = "food"
	npcAnchorSocial = "social"

	npcAnchorSourceOwner   = "owner"
	npcAnchorSourceRole    = "role"
	npcAnchorSourceGeneric = "generic"
)

var npcAnchorTags = map[string]map[string]bool{
	npcAnchorHome: {
		"home":  true,
		"bed":   true,
		"sleep": true,
		"rest":  true,
	},
	npcAnchorWork: {
		"work":     true,
		"school":   true,
		"shop":     true,
		"merchant": true,
		"farm":     true,
		"mine":     true,
	},
	npcAnchorFood: {
		"food":    true,
		"kitchen": true,
		"meal":    true,
	},
	npcAnchorSocial: {
		"public":    true,
		"social":    true,
		"idle":      true,
		"gathering": true,
	},
}

type npcAnchorCandidate struct {
	anchor   npcLocationAnchor
	score    float64
	mapID    string
	location stmaps.Location
}

func (world *world) configureNPCRoutineAnchors() {
	if world == nil {
		return
	}
	for _, room := range world.rooms {
		if room == nil {
			continue
		}
		room.mu.Lock()
		for _, npc := range room.liveNPCs {
			npc.anchors = world.resolveNPCRoutineAnchors(npc, room.gameMap.ID, stnavigation.Point{X: npc.x, Y: npc.y})
		}
		room.mu.Unlock()
	}
}

func (world *world) resolveNPCRoutineAnchors(npc *liveNPC, startMapID string, start stnavigation.Point) npcRoutineAnchors {
	if npc == nil || world == nil {
		return npcRoutineAnchors{}
	}
	return npcRoutineAnchors{
		Home:   world.resolveNPCAnchor(npc, startMapID, start, npcAnchorHome),
		Work:   world.resolveNPCAnchor(npc, startMapID, start, npcAnchorWork),
		Food:   world.resolveNPCAnchor(npc, startMapID, start, npcAnchorFood),
		Social: world.resolveNPCAnchor(npc, startMapID, start, npcAnchorSocial),
	}
}

func (world *world) resolveNPCAnchor(npc *liveNPC, startMapID string, start stnavigation.Point, kind string) *npcLocationAnchor {
	candidates := []npcAnchorCandidate{}
	for _, gameMap := range world.sortedGameMaps() {
		for _, location := range gameMap.Locations {
			if !locationHasAnyTag(location, npcAnchorTags[kind]) {
				continue
			}
			source := npcAnchorSourceGeneric
			score := 0.0
			if location.OwnerNPCKey != "" && location.OwnerNPCKey == npc.npcKey {
				source = npcAnchorSourceOwner
				score += 10000
			} else if kind == npcAnchorWork && locationRoleMatchesNPC(npc, location) {
				source = npcAnchorSourceRole
				score += 5000
			}
			if gameMap.ID == startMapID {
				score += 100
			}
			score -= math.Hypot(location.X-start.X, location.Y-start.Y)
			candidates = append(candidates, npcAnchorCandidate{
				anchor: npcLocationAnchor{
					Kind:         kind,
					MapID:        gameMap.ID,
					LocationID:   location.ID,
					LocationName: location.Name,
					Source:       source,
					Tags:         append([]string(nil), location.Tags...),
				},
				score:    score,
				mapID:    gameMap.ID,
				location: location,
			})
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i int, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.score != right.score {
			return left.score > right.score
		}
		if left.mapID != right.mapID {
			return left.mapID < right.mapID
		}
		return left.location.ID < right.location.ID
	})
	return &candidates[0].anchor
}

func (world *world) sortedGameMaps() []gameMap {
	mapIDs := make([]string, 0, len(world.rooms))
	for mapID := range world.rooms {
		mapIDs = append(mapIDs, mapID)
	}
	sort.Strings(mapIDs)
	gameMaps := make([]gameMap, 0, len(mapIDs))
	for _, mapID := range mapIDs {
		if room := world.rooms[mapID]; room != nil {
			gameMaps = append(gameMaps, room.gameMap)
		}
	}
	return gameMaps
}

func locationHasAnyTag(location stmaps.Location, tags map[string]bool) bool {
	for _, tag := range location.Tags {
		if tags[strings.ToLower(strings.TrimSpace(tag))] {
			return true
		}
	}
	return false
}

func (anchors npcRoutineAnchors) matchingAnchor(drive npcDrive, mapID string, locationID string) *npcLocationAnchor {
	for _, anchor := range anchors.forDrive(drive) {
		if anchor != nil && anchor.MapID == mapID && anchor.LocationID == locationID {
			return anchor
		}
	}
	return nil
}

func (anchors npcRoutineAnchors) forDrive(drive npcDrive) []*npcLocationAnchor {
	switch drive {
	case npcDriveEnergy:
		return []*npcLocationAnchor{anchors.Home}
	case npcDriveWork:
		return []*npcLocationAnchor{anchors.Work}
	case npcDriveHunger:
		return []*npcLocationAnchor{anchors.Food}
	case npcDriveSocial, npcDriveIdle:
		return []*npcLocationAnchor{anchors.Social}
	default:
		return nil
	}
}
