package server

import (
	"math"
	"time"
)

func (room *room) broadcastSnapshot(now time.Time) {
	room.mu.Lock()
	message := serverMessage{
		Type:          "snapshot",
		MapID:         room.gameMap.ID,
		Tick:          room.tick,
		ServerTimeMS:  now.UnixMilli(),
		Players:       room.snapshotsLocked(),
		Collectibles:  room.collectibleSnapshotsLocked(),
		ResourceNodes: room.resourceNodeSnapshotsLocked(),
		PlacedObjects: room.placedObjectSnapshotsLocked(),
		WorldObjects:  room.worldObjectSnapshotsLocked(),
	}
	clients := make([]*client, 0, len(room.players))
	for _, player := range room.players {
		clients = append(clients, player.client)
	}
	room.mu.Unlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			room.leave(client)
		}
	}
}

func (room *room) broadcast(message serverMessage) {
	room.mu.Lock()
	clients := make([]*client, 0, len(room.players))
	for _, player := range room.players {
		clients = append(clients, player.client)
	}
	room.mu.Unlock()

	for _, client := range clients {
		client.trySend(message)
	}
}

func (room *room) snapshotsLocked() []playerSnapshot {
	snapshots := make([]playerSnapshot, 0, len(room.players))
	for _, player := range room.players {
		snapshots = append(snapshots, playerSnapshot{
			ID:               player.id,
			CharacterID:      player.characterID,
			DisplayName:      player.displayName,
			X:                math.Round(player.x*10) / 10,
			Y:                math.Round(player.y*10) / 10,
			Facing:           player.facing,
			Moving:           player.moving,
			AvatarID:         player.avatarID,
			Equipment:        cloneEquipment(player.equipment),
			LastProcessedSeq: player.lastMoveSeq,
		})
	}
	return snapshots
}

func cloneEquipment(equipment equipmentSnapshot) equipmentSnapshot {
	if len(equipment) == 0 {
		return equipmentSnapshot{}
	}
	cloned := make(equipmentSnapshot, len(equipment))
	for slot, visualKey := range equipment {
		cloned[slot] = visualKey
	}
	return cloned
}

func inventoryFromEquipment(equipment equipmentSnapshot) inventorySnapshot {
	inventory := inventorySnapshot{}
	if equipment[equipmentSlotTool] == "pickaxe" {
		inventory["pickaxe"] = 1
	}
	return inventory
}

func (room *room) collectibleSnapshotsLocked() []collectibleSnapshot {
	snapshots := make([]collectibleSnapshot, 0, len(room.collectibles))
	for _, collectible := range room.collectibles {
		snapshots = append(snapshots, collectibleSnapshot{
			ID:     collectible.id,
			Kind:   collectible.kind,
			X:      collectible.x,
			Y:      collectible.y,
			Active: collectible.active,
		})
	}
	return snapshots
}

func (room *room) resourceNodeSnapshotsLocked() []resourceNodeSnapshot {
	snapshots := make([]resourceNodeSnapshot, 0, len(room.resourceNodes))
	for _, node := range room.resourceNodes {
		snapshots = append(snapshots, resourceNodeSnapshot{
			ID:     node.id,
			Kind:   node.resourceKind,
			X:      node.x,
			Y:      node.y,
			Radius: node.radius,
			Active: node.active,
			Hits:   node.hitCount,
			Needed: node.hitsRequired,
		})
	}
	return snapshots
}

func (room *room) placedObjectSnapshotsLocked() []placedObjectSnapshot {
	snapshots := make([]placedObjectSnapshot, 0, len(room.placedObjects))
	for _, object := range room.placedObjects {
		snapshots = append(snapshots, object.snapshot())
	}
	return snapshots
}

func (room *room) worldObjectSnapshotsLocked() []worldObjectSnapshot {
	snapshots := make([]worldObjectSnapshot, 0, len(room.worldObjects))
	for _, object := range room.worldObjects {
		snapshots = append(snapshots, object.worldObjectSnapshot())
	}
	return snapshots
}

func (object *placedObject) snapshot() placedObjectSnapshot {
	return placedObjectSnapshot{
		ID:                object.id,
		ItemKey:           object.itemKey,
		GridX:             object.gridX,
		GridY:             object.gridY,
		X:                 object.x,
		Y:                 object.y,
		Width:             object.width,
		Height:            object.height,
		PlacedByAppUserID: object.placedByAppUserID,
	}
}

func (object *worldObject) worldObjectSnapshot() worldObjectSnapshot {
	return worldObjectSnapshot{
		ID:                object.id,
		Kind:              object.kind,
		Source:            object.source,
		ItemKey:           object.itemKey,
		ResourceKind:      object.resourceKind,
		X:                 object.x,
		Y:                 object.y,
		Width:             object.width,
		Height:            object.height,
		Radius:            object.radius,
		Active:            object.active,
		Collision:         object.collision,
		Breakable:         object.breakable,
		ReservesPlacement: object.reservesPlacement,
		Hits:              object.hitCount,
		Needed:            object.hitsRequired,
		GridX:             object.gridX,
		GridY:             object.gridY,
		PlacedByAppUserID: object.placedByAppUserID,
	}
}
