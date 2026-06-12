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
		NPCs:          room.npcSnapshotsLocked(now),
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

func (room *room) npcSnapshotsLocked(now time.Time) []npcSnapshot {
	snapshots := make([]npcSnapshot, 0, len(room.liveNPCs))
	for _, liveNPC := range room.liveNPCs {
		character, hasCharacter := room.world.npcCharacter(liveNPC.npcKey)
		snapshots = append(snapshots, liveNPC.snapshot(character, hasCharacter, now))
	}
	return snapshots
}

func (liveNPC *liveNPC) snapshot(character npcCharacter, hasCharacter bool, now time.Time) npcSnapshot {
	name := liveNPC.displayName
	spriteKey := liveNPC.spriteKey
	characterID := liveNPC.characterID
	if hasCharacter {
		characterID = character.characterID
		if character.displayName != "" {
			name = character.displayName
		}
		if character.avatarID != "" {
			spriteKey = character.avatarID
		}
	}
	return npcSnapshot{
		ID:            liveNPC.npcKey,
		CharacterID:   characterID,
		Name:          name,
		X:             math.Round(liveNPC.x*10) / 10,
		Y:             math.Round(liveNPC.y*10) / 10,
		Facing:        liveNPC.facing,
		Moving:        liveNPC.moving,
		SpriteKey:     spriteKey,
		Dialogue:      append([]string(nil), liveNPC.dialogue...),
		RoutineStatus: liveNPC.publicRoutineStatus(now),
		Shop:          cloneShop(liveNPC.shop),
		Activity:      cloneActivity(liveNPC.activity),
	}
}

func (liveNPC *liveNPC) publicRoutineStatus(now time.Time) string {
	if liveNPC == nil {
		return ""
	}
	if liveNPC.goal != nil {
		if liveNPC.targetFailedRecentlyDebug(*liveNPC.goal, now) {
			return "blocked"
		}
		if liveNPC.route != nil {
			return "traveling"
		}
		if !liveNPC.goalArrivedAt.IsZero() {
			return publicRoutineStatusForGoal(*liveNPC.goal)
		}
		return ""
	}
	if liveNPC.mostRecentFailedTargetKey() != "" {
		return "blocked"
	}
	return ""
}

func publicRoutineStatusForGoal(goal npcGoal) string {
	switch {
	case goal.anchorKind == npcAnchorHome || goal.drive == npcDriveEnergy:
		return "resting"
	case goal.anchorKind == npcAnchorWork || goal.drive == npcDriveWork:
		return "working"
	default:
		return ""
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
		Name:              object.name,
		LocationID:        object.locationID,
		ShopID:            object.shopID,
		StorageRole:       object.storageRole,
		X:                 object.x,
		Y:                 object.y,
		Width:             object.width,
		Height:            object.height,
		Radius:            object.radius,
		InteractionRadius: object.interactionRadius,
		Active:            object.active,
		Collision:         object.collision,
		Breakable:         object.breakable,
		ReservesPlacement: object.reservesPlacement,
		Hits:              object.hitCount,
		Needed:            object.hitsRequired,
		GridX:             object.gridX,
		GridY:             object.gridY,
		PlacedByAppUserID: object.placedByAppUserID,
		Tags:              append([]string(nil), object.tags...),
	}
}
