package server

import "hq/internal/sunnytown/hqclient"

func (world *world) setNPCCharacters(characters []hqclient.NPCCharacterResponse) {
	world.npcMu.Lock()
	defer world.npcMu.Unlock()

	next := make(map[string]npcCharacter, len(characters))
	for _, character := range characters {
		if character.NPCKey == "" || character.CharacterID < 1 {
			continue
		}
		next[character.NPCKey] = npcCharacter{
			characterID: character.CharacterID,
			displayName: character.DisplayName,
			avatarID:    character.AvatarID,
		}
	}
	world.npcCharacters = next
}

func (world *world) npcCharacter(npcKey string) (npcCharacter, bool) {
	world.npcMu.RLock()
	defer world.npcMu.RUnlock()
	character, ok := world.npcCharacters[npcKey]
	return character, ok
}

func initialLiveNPCs(gameMap gameMap) map[string]*liveNPC {
	liveNPCs := make(map[string]*liveNPC, len(gameMap.NPCs))
	for _, mapNPC := range gameMap.NPCs {
		liveNPCs[mapNPC.ID] = &liveNPC{
			characterID: mapNPC.CharacterID,
			npcKey:      mapNPC.ID,
			displayName: mapNPC.Name,
			spriteKey:   mapNPC.SpriteKey,
			mapID:       gameMap.ID,
			x:           mapNPC.X,
			y:           mapNPC.Y,
			facing:      mapNPC.Facing,
			dialogue:    append([]string(nil), mapNPC.Dialogue...),
			shop:        cloneShop(mapNPC.Shop),
			activity:    cloneActivity(mapNPC.Activity),
		}
	}
	return liveNPCs
}

func cloneShop(shopSnapshot *shop) *shop {
	if shopSnapshot == nil {
		return nil
	}
	cloned := &shop{
		ID:    shopSnapshot.ID,
		Items: append([]shopItem(nil), shopSnapshot.Items...),
	}
	return cloned
}

func cloneActivity(activitySnapshot *activity) *activity {
	if activitySnapshot == nil {
		return nil
	}
	return &activity{Type: activitySnapshot.Type}
}

func (room *room) mapSnapshotLocked() gameMap {
	snapshot := room.gameMap
	if len(snapshot.NPCs) == 0 {
		return snapshot
	}

	snapshot.NPCs = append([]npc(nil), snapshot.NPCs...)
	room.world.npcMu.RLock()
	defer room.world.npcMu.RUnlock()
	for index := range snapshot.NPCs {
		character, ok := room.world.npcCharacters[snapshot.NPCs[index].ID]
		if !ok {
			continue
		}
		snapshot.NPCs[index].CharacterID = character.characterID
		if character.displayName != "" {
			snapshot.NPCs[index].Name = character.displayName
		}
		if character.avatarID != "" {
			snapshot.NPCs[index].SpriteKey = character.avatarID
		}
	}
	return snapshot
}
