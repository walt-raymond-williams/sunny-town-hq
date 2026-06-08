package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func (client *client) refreshEquipment() {
	room := client.currentRoom()
	if room == nil || client.server == nil {
		return
	}

	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		return
	}
	userID := player.appUserID
	room.mu.Unlock()

	equipment, err := client.server.hq.LoadStudentEquipment(context.Background(), userID)
	if err != nil {
		log.Printf("refresh equipment player=%s: %v", client.id, err)
		client.trySend(serverMessage{Type: "error", Code: "equipment_refresh_failed"})
		return
	}

	room.mu.Lock()
	if player := room.players[client.id]; player != nil {
		player.equipment = equipment
	}
	room.mu.Unlock()
	room.broadcastSnapshot(time.Now())
}

func (client *client) ownsInventoryItem(ctx context.Context, appUserID int64, itemKey string) (bool, error) {
	itemKey = strings.TrimSpace(itemKey)
	if itemKey == "" || appUserID < 1 {
		return false, nil
	}
	if client.server != nil {
		quantity, err := client.server.hq.LoadStudentInventoryQuantity(ctx, appUserID, itemKey)
		if err != nil {
			return false, err
		}
		return quantity > 0, nil
	}

	room := client.currentRoom()
	if room == nil {
		return false, nil
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	player := room.players[client.id]
	if player == nil {
		return false, nil
	}
	return player.inventory[itemKey] > 0, nil
}

func (client *client) handleToolUse(message clientMessage) {
	toolKey := strings.TrimSpace(message.ToolKey)
	if toolKey == "" {
		client.trySend(serverMessage{Type: "error", Code: "missing_tool"})
		return
	}

	room := client.currentRoom()
	if room == nil {
		client.trySend(serverMessage{Type: "error", Code: "not_in_room"})
		return
	}

	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	appUserID := player.appUserID
	room.mu.Unlock()

	if toolKey != "pickaxe" {
		return
	}
	ownsTool, err := client.ownsInventoryItem(context.Background(), appUserID, toolKey)
	if err != nil {
		log.Printf("validate tool ownership player=%s tool=%s: %v", client.id, toolKey, err)
		client.trySend(serverMessage{Type: "error", Code: "tool_validation_failed"})
		return
	}
	if !ownsTool {
		client.trySend(serverMessage{Type: "error", Code: "tool_not_owned"})
		return
	}

	var event *resourceEvent
	var removeRequest *removeMapObjectRequest
	var removeTargetID string
	now := time.Now()
	room.mu.Lock()
	player = room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	if !player.lastToolUseAt.IsZero() && now.Sub(player.lastToolUseAt) < resourceToolCooldown {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "tool_cooldown"})
		return
	}
	target := room.nearestBreakableWorldObjectLocked(player, toolKey)
	if target == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "resource_node_not_found"})
		return
	}
	if target.source == worldObjectSourcePlaced && client.server == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "map_object_remove_failed"})
		return
	}
	player.lastToolUseAt = now
	target.hitCount++
	if target.hitCount >= target.hitsRequired {
		switch target.source {
		case worldObjectSourcePlaced:
			removeTargetID = target.id
			removeRequest = &removeMapObjectRequest{
				AppUserID: player.appUserID,
				RoomID:    room.id,
				MapID:     room.gameMap.ID,
				GridX:     target.gridX,
				GridY:     target.gridY,
			}
		case worldObjectSourceNatural:
			target.active = false
			target.hitCount = 0
			target.harvestSeq++
			target.respawnAt = now.Add(target.respawnDelay)
			resourceKey, amount := rollMiningDrop()
			eventID := fmt.Sprintf("%s:%s:%s:%d:%d", room.gameMap.ID, room.rewardRunID, target.id, target.harvestSeq, player.appUserID)
			event = &resourceEvent{
				eventID:     eventID,
				appUserID:   player.appUserID,
				roomID:      room.id,
				mapID:       room.gameMap.ID,
				nodeID:      target.id,
				resourceKey: resourceKey,
				amount:      amount,
				client:      player.client,
			}
		}
	}
	room.mu.Unlock()

	if removeRequest != nil {
		client.removePlacedWorldObject(room, *removeRequest, removeTargetID, now)
		return
	}
	room.broadcastSnapshot(now)
	if event == nil {
		return
	}
	select {
	case room.resourceEvents <- *event:
	default:
		log.Printf("resource queue full event=%s", event.eventID)
		client.trySend(serverMessage{
			Type:   "resource_failed",
			NodeID: event.nodeID,
			Reason: "temporary_error",
		})
	}
}

func (client *client) removePlacedWorldObject(room *room, request removeMapObjectRequest, targetID string, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	removed, quantity, err := client.server.removeMapObject(ctx, request, room.gameMap)
	cancel()
	if err != nil {
		log.Printf("remove placed object player=%s object=%s: %v", client.id, targetID, err)
		client.trySend(serverMessage{Type: "error", Code: "map_object_remove_failed"})
		return
	}

	var snapshot placedObjectSnapshot
	var worldSnapshot worldObjectSnapshot
	room.mu.Lock()
	room.removePlacedObjectLocked(removed.id)
	snapshot = removed.snapshot()
	worldSnapshot = removed.worldObjectSnapshot()
	room.mu.Unlock()

	room.broadcastSnapshot(now)
	room.broadcast(serverMessage{
		Type:         "map_object_removed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
	})
	client.trySend(serverMessage{
		Type:         "map_object_removed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
		ResourceKey:  removed.itemKey,
		Quantity:     quantity,
	})
}

func (client *client) handlePlaceObject(message clientMessage) {
	itemKey := strings.TrimSpace(message.ItemKey)
	if itemKey != "stone_block" {
		client.trySend(serverMessage{Type: "error", Code: "unsupported_map_object"})
		return
	}
	room := client.currentRoom()
	if room == nil {
		client.trySend(serverMessage{Type: "error", Code: "not_in_room"})
		return
	}
	if client.server == nil {
		client.trySend(serverMessage{Type: "error", Code: "map_object_place_failed"})
		return
	}

	var request placeMapObjectRequest
	room.mu.Lock()
	player := room.players[client.id]
	if player == nil {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "player_not_found"})
		return
	}
	if !room.canPlaceObjectLocked(message.GridX, message.GridY, itemKey) {
		room.mu.Unlock()
		client.trySend(serverMessage{Type: "error", Code: "invalid_map_object_location"})
		return
	}
	request = placeMapObjectRequest{
		AppUserID: player.appUserID,
		RoomID:    room.id,
		MapID:     room.gameMap.ID,
		GridX:     message.GridX,
		GridY:     message.GridY,
		ItemKey:   itemKey,
	}
	room.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	placed, quantity, err := client.server.placeMapObject(ctx, request, room.gameMap)
	cancel()
	if err != nil {
		log.Printf("place object player=%s map=%s grid=(%d,%d): %v", client.id, room.gameMap.ID, message.GridX, message.GridY, err)
		client.trySend(serverMessage{Type: "error", Code: "map_object_place_failed"})
		return
	}

	room.mu.Lock()
	if !room.canPlaceObjectLocked(placed.gridX, placed.gridY, placed.itemKey) {
		deleteRequest := removeMapObjectRequest{
			AppUserID: request.AppUserID,
			RoomID:    request.RoomID,
			MapID:     request.MapID,
			GridX:     request.GridX,
			GridY:     request.GridY,
		}
		room.mu.Unlock()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _, _ = client.server.removeMapObject(cleanupCtx, deleteRequest, room.gameMap)
		cleanupCancel()
		client.trySend(serverMessage{Type: "error", Code: "invalid_map_object_location"})
		return
	}
	room.addPlacedObjectLocked(placed)
	snapshot := placed.snapshot()
	worldSnapshot := placed.worldObjectSnapshot()
	room.mu.Unlock()

	room.broadcastSnapshot(time.Now())
	room.broadcast(serverMessage{
		Type:         "map_object_placed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
	})
	client.trySend(serverMessage{
		Type:         "map_object_placed",
		MapID:        room.gameMap.ID,
		PlacedObject: &snapshot,
		WorldObject:  &worldSnapshot,
		ResourceKey:  placed.itemKey,
		Quantity:     quantity,
	})
}
