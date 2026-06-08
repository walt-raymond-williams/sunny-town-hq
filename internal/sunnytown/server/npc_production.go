package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	stnavigation "hq/internal/sunnytown/navigation"
)

type npcJobDefinition struct {
	JobKey    string
	OutputKey string
}

func (room *room) collectNPCJobProductionLocked(dt float64, now time.Time) []npcJobProductionEvent {
	if dt <= 0 {
		return nil
	}
	events := []npcJobProductionEvent{}
	for _, npc := range room.liveNPCs {
		if event, ok := room.advanceNPCJobProductionLocked(npc, dt, now); ok {
			events = append(events, event)
		}
	}
	return events
}

func (room *room) advanceNPCJobProductionLocked(npc *liveNPC, dt float64, now time.Time) (npcJobProductionEvent, bool) {
	if npc == nil || room.world == nil || npc.anchors.Work == nil {
		return npcJobProductionEvent{}, false
	}
	job, ok := npc.jobDefinition()
	if !ok {
		npc.jobProduction.Progress = 0
		return npcJobProductionEvent{}, false
	}
	if !room.npcAtWorkAnchorLocked(npc) {
		npc.jobProduction.Progress = 0
		return npcJobProductionEvent{}, false
	}
	characterID := npc.characterID
	if characterID < 1 {
		if character, ok := room.world.npcCharacter(npc.npcKey); ok {
			characterID = character.characterID
		}
	}
	if characterID < 1 {
		return npcJobProductionEvent{}, false
	}

	npc.jobProduction.Progress += dt
	if npc.jobProduction.Progress < npcJobProductionInterval.Seconds() {
		return npcJobProductionEvent{}, false
	}
	npc.jobProduction.Progress -= npcJobProductionInterval.Seconds()
	npc.jobProduction.Sequence++
	eventID := fmt.Sprintf("%s:%s:%s:%s:%d", room.id, npc.npcKey, job.JobKey, npc.anchors.Work.LocationID, npc.jobProduction.Sequence)
	npc.jobProduction.LastAt = now
	npc.jobProduction.LastEvent = eventID

	return npcJobProductionEvent{
		eventID:     eventID,
		characterID: characterID,
		roomID:      room.id,
		mapID:       room.gameMap.ID,
		npcKey:      npc.npcKey,
		jobKey:      job.JobKey,
		locationID:  npc.anchors.Work.LocationID,
		outputKey:   job.OutputKey,
		amount:      npcJobProductionUnit,
	}, true
}

func (room *room) npcAtWorkAnchorLocked(npc *liveNPC) bool {
	if npc == nil || npc.anchors.Work == nil || npc.anchors.Work.MapID != room.gameMap.ID {
		return false
	}
	if room.world == nil || room.world.navigation == nil {
		return false
	}
	location, ok := room.world.navigation.Location(room.gameMap.ID, npc.anchors.Work.LocationID)
	if !ok {
		return false
	}
	return pointWithinLocation(stnavigation.Point{X: npc.x, Y: npc.y}, location)
}

func (npc *liveNPC) jobDefinition() (npcJobDefinition, bool) {
	if npc == nil {
		return npcJobDefinition{}, false
	}
	if npc.shop != nil {
		return npcJobDefinition{JobKey: "shopkeeper_stock", OutputKey: "shop_stock_progress"}, true
	}
	if npc.activity != nil {
		switch strings.TrimSpace(strings.ToLower(npc.activity.Type)) {
		case "schoolwork":
			return npcJobDefinition{JobKey: "teacher_lesson_prep", OutputKey: "lesson_prep_progress"}, true
		}
	}
	return npcJobDefinition{}, false
}

func queueNPCJobProductionEvents(events chan npcJobProductionEvent, pending []npcJobProductionEvent) {
	for _, event := range pending {
		select {
		case events <- event:
		default:
			log.Printf("npc job production queue full event=%s npc=%s", event.eventID, event.npcKey)
		}
	}
}
