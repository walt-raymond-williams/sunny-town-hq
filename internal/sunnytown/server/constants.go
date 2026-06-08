package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"hq/internal/sunnytownauth"
)

const (
	defaultRoomID              = "sunny-town-main"
	DefaultMapID               = "sunny-town-v1"
	defaultMapID               = DefaultMapID
	equipmentSlotGear          = "gear"
	equipmentSlotAccessory     = "accessory"
	equipmentSlotTool          = "tool"
	playerSize                 = 28.0
	playerSpeed                = 150.0
	npcSpeed                   = 80.0
	npcDriveDefault            = 95.0
	npcDriveThreshold          = 50.0
	npcEmergencyDriveThreshold = 20.0
	npcDriveDepletePerSecond   = 0.1
	npcDriveReplenishPerSecond = 8.0
	npcGoalFocusDuration       = 5 * time.Second
	npcGoalReevaluateInterval  = 2 * time.Second
	npcGoalGraceDuration       = 2 * time.Second
	npcFailedTargetCooldown    = 10 * time.Second
	npcNoPlayerCatchUpMax      = 5 * time.Minute
	starPickupRadius           = 30.0
	resourceCommitQueueSize    = 32
	npcJobProductionQueueSize  = 32
	npcJobProductionInterval   = 20 * time.Second
	npcJobProductionCatchUpMax = 5 * time.Minute
	npcJobProductionUnit       = 1
	resourceHitsRequired       = 3
	resourceToolCooldown       = 500 * time.Millisecond
	simulationInterval         = 50 * time.Millisecond
	snapshotInterval           = 100 * time.Millisecond
	movingStateTTL             = 250 * time.Millisecond
	starRespawnDelay           = 10 * time.Second
)

const (
	worldObjectSourceNatural = "natural"
	worldObjectSourcePlaced  = "placed"
	worldObjectSourceFixture = "fixture"

	worldObjectKindRockNode   = "rock_node"
	worldObjectKindStoneBlock = "stone_block"
	worldObjectKindChest      = "chest"
)

func (room *room) step(dt float64, now time.Time) {
	room.mu.Lock()

	room.tick++
	var rewards []rewardEvent
	for _, player := range room.players {
		if !player.lastMoveAt.IsZero() && now.Sub(player.lastMoveAt) > movingStateTTL {
			player.moving = false
		}

		rewards = append(rewards, room.collectStarsLocked(player, now)...)
	}

	var npcTransfers []npcTransfer
	var npcJobEvents []npcJobProductionEvent
	if len(room.players) > 0 || room.npcPausedAt.IsZero() {
		npcTransfers = room.stepLiveNPCsLocked(dt, now)
		npcJobEvents = room.collectNPCJobProductionLocked(dt, now)
	}
	room.respawnCollectiblesLocked(now)
	room.respawnResourceNodesLocked(now)
	room.mu.Unlock()

	room.world.applyNPCTransfers(npcTransfers, now)

	for _, reward := range rewards {
		select {
		case room.rewardEvents <- reward:
		default:
			log.Printf("reward queue full event=%s", reward.eventID)
			reward.client.trySend(serverMessage{
				Type:          "reward_failed",
				CollectibleID: reward.collectibleID,
				Reason:        "temporary_error",
			})
		}
	}
	queueNPCJobProductionEvents(room.npcJobEvents, npcJobEvents)
}

func newRewardRunID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func validateJoinTarget(claims sunnytownauth.Claims, roomID string, rooms map[string]*room) error {
	if claims.RoomID != roomID {
		return errors.New("unknown room or map")
	}
	if rooms[claims.MapID] == nil {
		return errors.New("unknown room or map")
	}
	return nil
}
