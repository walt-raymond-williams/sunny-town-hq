package server

import (
	"crypto/rand"
	"fmt"
	"math"
	"strconv"
	"time"
)

func (room *room) nearestBreakableWorldObjectLocked(player *player, toolKey string) *worldObject {
	var nearest *worldObject
	nearestDistance := math.MaxFloat64
	for _, object := range room.worldObjects {
		if !object.active || !object.breakable || object.toolKey != toolKey {
			continue
		}
		centerX, centerY := object.center()
		distance := math.Hypot(player.x-centerX, player.y-centerY)
		if distance > object.interactionRadius || distance >= nearestDistance {
			continue
		}
		nearest = object
		nearestDistance = distance
	}
	return nearest
}

func (room *room) validatedContainerAccessLocked(player *player, objectSource string, objectID string, action string) (string, bool) {
	if player == nil || objectID == "" {
		return "", false
	}
	object := room.worldObjects[worldObjectKey(objectSource, objectID)]
	if object == nil || !object.active || object.kind != worldObjectKindChest {
		return "", false
	}
	if object.mapID != room.gameMap.ID {
		return "", false
	}
	if !containerActionAllowed(object, action) {
		return "", false
	}
	centerX, centerY := object.center()
	if math.Hypot(player.x-centerX, player.y-centerY) > object.interactionRadius {
		return "", false
	}
	switch object.source {
	case worldObjectSourceFixture:
		return fmt.Sprintf("fixture:%s:%s:%s", room.id, room.gameMap.ID, object.id), true
	case worldObjectSourcePlaced:
		return "placed:" + object.id, true
	default:
		return "", false
	}
}

func containerActionAllowed(object *worldObject, action string) bool {
	switch action {
	case "read":
		return object.storageRole == "output" || object.storageRole == "input" || object.storageRole == "general"
	case "deposit":
		return object.storageRole == "input" || object.storageRole == "general"
	case "withdraw":
		return object.storageRole == "general"
	default:
		return false
	}
}

func (room *room) collectStarsLocked(player *player, now time.Time) []rewardEvent {
	rewards := []rewardEvent{}
	for _, collectible := range room.collectibles {
		if !collectible.active || collectible.kind != "star" {
			continue
		}
		if math.Hypot(player.x-collectible.x, player.y-collectible.y) > starPickupRadius {
			continue
		}

		collectible.active = false
		collectible.spawnSeq++
		collectible.respawnAt = now.Add(starRespawnDelay)
		eventID := fmt.Sprintf("%s:%s:%s:%d:%d", room.id, room.rewardRunID, collectible.id, collectible.spawnSeq, player.appUserID)
		rewards = append(rewards, rewardEvent{
			eventID:       eventID,
			appUserID:     player.appUserID,
			roomID:        room.id,
			mapID:         room.gameMap.ID,
			collectibleID: collectible.id,
			kind:          collectible.kind,
			amount:        1,
			client:        player.client,
		})
	}
	return rewards
}

func (room *room) respawnCollectiblesLocked(now time.Time) {
	for _, collectible := range room.collectibles {
		if collectible.active || collectible.respawnAt.IsZero() || now.Before(collectible.respawnAt) {
			continue
		}
		collectible.active = true
		collectible.respawnAt = time.Time{}
	}
}

func (room *room) respawnResourceNodesLocked(now time.Time) {
	for _, node := range room.resourceNodes {
		if node.active || node.respawnAt.IsZero() || now.Before(node.respawnAt) {
			continue
		}
		node.active = true
		node.hitCount = 0
		node.respawnAt = time.Time{}
	}
}

func (room *room) canPlaceObjectLocked(gridX int, gridY int, itemKey string) bool {
	if itemKey != "stone_block" || gridX < 0 || gridY < 0 {
		return false
	}
	if gridX >= room.gameMap.Width || gridY >= room.gameMap.Height {
		return false
	}
	objectRect := gridRect(room.gameMap, gridX, gridY)
	for _, blocked := range room.gameMap.BlockedRects {
		if rectsOverlap(objectRect, blocked) {
			return false
		}
	}
	for _, portal := range room.gameMap.Portals {
		if rectsOverlap(objectRect, rect{X: portal.X, Y: portal.Y, Width: portal.Width, Height: portal.Height}) {
			return false
		}
	}
	for _, npc := range room.gameMap.NPCs {
		if rectsOverlap(objectRect, rect{X: npc.X - playerSize/2, Y: npc.Y - playerSize/2, Width: playerSize, Height: playerSize}) {
			return false
		}
	}
	for _, object := range room.worldObjects {
		if object.reservesPlacement && rectsOverlap(objectRect, object.rect()) {
			return false
		}
	}
	for _, player := range room.players {
		if rectsOverlap(objectRect, rect{X: player.x - playerSize/2, Y: player.y - playerSize/2, Width: playerSize, Height: playerSize}) {
			return false
		}
	}
	return true
}

func initialCollectibles(gameMap gameMap) map[string]*collectible {
	collectibles := map[string]*collectible{}
	for index, spawn := range gameMap.StarSpawns {
		id := fmt.Sprintf("star-%04d", index+1)
		collectibles[id] = &collectible{
			id:       id,
			kind:     "star",
			x:        spawn.X,
			y:        spawn.Y,
			active:   true,
			spawnSeq: 0,
		}
	}
	return collectibles
}

func initialResourceNodes(gameMap gameMap) map[string]*resourceNode {
	nodes := map[string]*resourceNode{}
	for _, definition := range gameMap.ResourceNodes {
		nodes[definition.ID] = &resourceNode{
			id:                definition.ID,
			kind:              worldObjectKindRockNode,
			source:            worldObjectSourceNatural,
			resourceKind:      definition.Kind,
			mapID:             gameMap.ID,
			x:                 definition.X,
			y:                 definition.Y,
			radius:            definition.Radius,
			interactionRadius: definition.InteractionRadius,
			collision:         true,
			breakable:         true,
			toolKey:           "pickaxe",
			hitsRequired:      resourceHitsRequired,
			reservesPlacement: true,
			respawnDelay:      time.Duration(definition.RespawnSeconds) * time.Second,
			active:            true,
		}
	}
	return nodes
}

func initialFixtureObjects(gameMap gameMap) map[string]*worldObject {
	objects := map[string]*worldObject{}
	for _, definition := range gameMap.Fixtures {
		objects[definition.ID] = fixtureObjectFromDefinition(gameMap, definition)
	}
	return objects
}

func fixtureObjectFromDefinition(gameMap gameMap, definition fixtureDefinition) *worldObject {
	return &worldObject{
		id:                definition.ID,
		kind:              definition.Kind,
		source:            worldObjectSourceFixture,
		itemKey:           definition.ItemKey,
		name:              definition.Name,
		locationID:        definition.LocationID,
		shopID:            definition.ShopID,
		storageRole:       definition.StorageRole,
		tags:              append([]string(nil), definition.Tags...),
		mapID:             gameMap.ID,
		x:                 definition.X,
		y:                 definition.Y,
		width:             definition.Width,
		height:            definition.Height,
		interactionRadius: definition.InteractionRadius,
		collision:         definition.Collision,
		breakable:         false,
		reservesPlacement: definition.ReservesPlacement,
		active:            true,
	}
}

func placedObjectFromResponse(gameMap gameMap, response mapObjectResponse) *placedObject {
	objectRect := gridRect(gameMap, response.GridX, response.GridY)
	return &placedObject{
		id:                strconv.FormatInt(response.ID, 10),
		kind:              worldObjectKindStoneBlock,
		source:            worldObjectSourcePlaced,
		itemKey:           response.ItemKey,
		mapID:             response.MapID,
		gridX:             response.GridX,
		gridY:             response.GridY,
		x:                 objectRect.X,
		y:                 objectRect.Y,
		width:             objectRect.Width,
		height:            objectRect.Height,
		interactionRadius: 58,
		collision:         true,
		breakable:         true,
		toolKey:           "pickaxe",
		hitsRequired:      1,
		reservesPlacement: true,
		active:            true,
		placedByAppUserID: response.PlacedByAppUserID,
	}
}

func gridRect(gameMap gameMap, gridX int, gridY int) rect {
	tileSize := float64(gameMap.TileSize)
	return rect{
		X:      float64(gridX) * tileSize,
		Y:      float64(gridY) * tileSize,
		Width:  tileSize,
		Height: tileSize,
	}
}

func (object *placedObject) rect() rect {
	if object.radius > 0 {
		return rect{
			X:      object.x - object.radius,
			Y:      object.y - object.radius,
			Width:  object.radius * 2,
			Height: object.radius * 2,
		}
	}
	return rect{
		X:      object.x,
		Y:      object.y,
		Width:  object.width,
		Height: object.height,
	}
}

func (object *worldObject) center() (float64, float64) {
	if object.radius > 0 {
		return object.x, object.y
	}
	return object.x + object.width/2, object.y + object.height/2
}

func worldObjectKey(source string, id string) string {
	return source + ":" + id
}

func (room *room) setPlacedObjectsLocked(objects map[string]*placedObject) {
	for _, object := range room.placedObjects {
		delete(room.worldObjects, worldObjectKey(object.source, object.id))
	}
	room.placedObjects = objects
	for _, object := range objects {
		room.addPlacedObjectLocked(object)
	}
}

func (room *room) addPlacedObjectLocked(object *placedObject) {
	room.placedObjects[object.id] = object
	room.worldObjects[worldObjectKey(object.source, object.id)] = object
}

func (room *room) removePlacedObjectLocked(id string) {
	if object := room.placedObjects[id]; object != nil {
		delete(room.worldObjects, worldObjectKey(object.source, object.id))
	}
	delete(room.placedObjects, id)
}

func rollMiningDrop() (string, int) {
	roll := randomInt(100)
	if roll < 85 {
		return "rock", 1
	}
	if roll < 95 {
		return "rock", 2
	}
	return "crystal", 1
}

func randomInt(max int) int {
	if max <= 1 {
		return 0
	}
	var bytes [1]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return int(bytes[0]) % max
	}
	return int(time.Now().UnixNano() % int64(max))
}
