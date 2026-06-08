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
