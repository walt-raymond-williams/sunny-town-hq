package server

import (
	"context"
	"log"
	"time"
)

func (srv *Server) RunRewardWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-srv.world.rewardEvents:
			response, err := srv.commitRewardWithRetry(ctx, event)
			if err != nil {
				log.Printf("reward commit failed event=%s player=%d: %v", event.eventID, event.appUserID, err)
				event.client.trySend(serverMessage{
					Type:          "reward_failed",
					CollectibleID: event.collectibleID,
					Reason:        "temporary_error",
				})
				continue
			}
			log.Printf("reward commit success event=%s player=%d duplicate=%v", event.eventID, event.appUserID, response.Duplicate)
			event.client.trySend(serverMessage{
				Type:           "reward_committed",
				EventID:        event.eventID,
				Kind:           event.kind,
				Amount:         event.amount,
				NewStarBalance: response.NewStarBalance,
			})
		}
	}
}

func (srv *Server) commitRewardWithRetry(ctx context.Context, event rewardEvent) (rewardCommitResponse, error) {
	backoffs := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		response, err := srv.commitReward(ctx, event)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == len(backoffs) {
			break
		}
		select {
		case <-ctx.Done():
			return rewardCommitResponse{}, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}
	return rewardCommitResponse{}, lastErr
}

func (srv *Server) commitReward(ctx context.Context, event rewardEvent) (rewardCommitResponse, error) {
	return srv.hq.CommitReward(ctx, rewardCommitRequest{
		EventID:       event.eventID,
		AppUserID:     event.appUserID,
		RoomID:        event.roomID,
		MapID:         event.mapID,
		CollectibleID: event.collectibleID,
		RewardKind:    event.kind,
		Amount:        event.amount,
	})
}

func (srv *Server) RunResourceWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-srv.world.resourceEvents:
			response, err := srv.commitResourceWithRetry(ctx, event)
			if err != nil {
				log.Printf("resource commit failed event=%s player=%d: %v", event.eventID, event.appUserID, err)
				event.client.trySend(serverMessage{
					Type:   "resource_failed",
					NodeID: event.nodeID,
					Reason: "temporary_error",
				})
				continue
			}
			log.Printf("resource commit success event=%s player=%d duplicate=%v", event.eventID, event.appUserID, response.Duplicate)
			event.client.trySend(serverMessage{
				Type:        "resource_committed",
				EventID:     event.eventID,
				NodeID:      event.nodeID,
				ResourceKey: response.ResourceKey,
				Amount:      event.amount,
				Quantity:    response.Quantity,
			})
		}
	}
}

func (srv *Server) RunNPCJobProductionWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-srv.world.npcJobEvents:
			response, err := srv.commitNPCJobProductionWithRetry(ctx, event)
			if err != nil {
				srv.world.recordNPCJobProductionCommit(event, npcJobProductionResponse{}, err, time.Now())
				log.Printf("npc job production commit failed event=%s npc=%s character=%d: %v", event.eventID, event.npcKey, event.characterID, err)
				continue
			}
			srv.world.recordNPCJobProductionCommit(event, response, nil, time.Now())
			if response.Blocked {
				log.Printf("npc job production blocked event=%s npc=%s character=%d duplicate=%v reason=%s", event.eventID, event.npcKey, event.characterID, response.Duplicate, response.BlockedReason)
				continue
			}
			log.Printf("npc job production commit success event=%s npc=%s character=%d duplicate=%v", event.eventID, event.npcKey, event.characterID, response.Duplicate)
		}
	}
}

func (srv *Server) commitNPCJobProductionWithRetry(ctx context.Context, event npcJobProductionEvent) (npcJobProductionResponse, error) {
	backoffs := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		response, err := srv.commitNPCJobProduction(ctx, event)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == len(backoffs) {
			break
		}
		select {
		case <-ctx.Done():
			return npcJobProductionResponse{}, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}
	return npcJobProductionResponse{}, lastErr
}

func (srv *Server) commitNPCJobProduction(ctx context.Context, event npcJobProductionEvent) (npcJobProductionResponse, error) {
	return srv.hq.CommitNPCJobProduction(ctx, npcJobProductionRequest{
		EventID:     event.eventID,
		CharacterID: event.characterID,
		RoomID:      event.roomID,
		MapID:       event.mapID,
		NPCKey:      event.npcKey,
		JobKey:      event.jobKey,
		LocationID:  event.locationID,
		OutputKey:   event.outputKey,
		Amount:      event.amount,
	})
}

func (world *world) recordNPCJobProductionCommit(event npcJobProductionEvent, response npcJobProductionResponse, commitErr error, now time.Time) {
	if world == nil {
		return
	}
	for _, room := range world.rooms {
		if room == nil {
			continue
		}
		room.mu.Lock()
		npc := room.liveNPCs[event.npcKey]
		if npc != nil {
			npc.jobProduction.LastCommitAt = now
			npc.jobProduction.LastBlockedReason = ""
			npc.jobProduction.LastCommitError = ""
			switch {
			case commitErr != nil:
				npc.jobProduction.LastCommitStatus = "error"
				npc.jobProduction.LastCommitError = commitErr.Error()
			case response.Blocked:
				npc.jobProduction.LastCommitStatus = "blocked"
				npc.jobProduction.LastBlockedReason = response.BlockedReason
			case response.Duplicate:
				npc.jobProduction.LastCommitStatus = "duplicate"
			default:
				npc.jobProduction.LastCommitStatus = "accepted"
			}
			room.mu.Unlock()
			return
		}
		room.mu.Unlock()
	}
}

func (srv *Server) commitResourceWithRetry(ctx context.Context, event resourceEvent) (resourceCommitResponse, error) {
	backoffs := []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		response, err := srv.commitResource(ctx, event)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == len(backoffs) {
			break
		}
		select {
		case <-ctx.Done():
			return resourceCommitResponse{}, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}
	return resourceCommitResponse{}, lastErr
}

func (srv *Server) commitResource(ctx context.Context, event resourceEvent) (resourceCommitResponse, error) {
	return srv.hq.CommitResource(ctx, resourceCommitRequest{
		EventID:     event.eventID,
		AppUserID:   event.appUserID,
		CharacterID: event.characterID,
		Source:      "sunny_town_mining",
		RoomID:      event.roomID,
		MapID:       event.mapID,
		NodeID:      event.nodeID,
		ResourceKey: event.resourceKey,
		Amount:      event.amount,
	})
}
