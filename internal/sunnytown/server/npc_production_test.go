package server

import (
	"testing"
	"time"

	"hq/internal/sunnytown/hqclient"
	stmaps "hq/internal/sunnytown/maps"
)

func TestNPCAtWorkAnchorQueuesJobProduction(t *testing.T) {
	room := testRoom(npcProductionTestMap(160, 160))
	room.world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 444,
		NPCKey:      "cookie-keeper",
		DisplayName: "Cookie Keeper",
		AvatarID:    "keeper",
	}})

	room.step(npcJobProductionInterval.Seconds()+0.1, time.Now())

	select {
	case event := <-room.npcJobEvents:
		if event.characterID != 444 || event.npcKey != "cookie-keeper" || event.jobKey != "shopkeeper_stock" || event.outputKey != "shop_stock_progress" {
			t.Fatalf("production event = %#v, want cookie keeper stock event", event)
		}
		if event.locationID != "cookie-keeper-counter" || event.amount != npcJobProductionUnit {
			t.Fatalf("production event = %#v, want counter unit production", event)
		}
	default:
		t.Fatal("expected npc job production event")
	}
}

func TestNPCAwayFromWorkAnchorDoesNotProduce(t *testing.T) {
	room := testRoom(npcProductionTestMap(64, 64))
	room.world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 444,
		NPCKey:      "cookie-keeper",
		DisplayName: "Cookie Keeper",
		AvatarID:    "keeper",
	}})

	room.step(npcJobProductionInterval.Seconds()+0.1, time.Now())

	select {
	case event := <-room.npcJobEvents:
		t.Fatalf("unexpected npc job production event: %#v", event)
	default:
	}
}

func TestNPCProductionRequiresDurableCharacterIdentity(t *testing.T) {
	room := testRoom(npcProductionTestMap(160, 160))

	room.step(npcJobProductionInterval.Seconds()+0.1, time.Now())

	select {
	case event := <-room.npcJobEvents:
		t.Fatalf("unexpected npc job production event without character id: %#v", event)
	default:
	}
}

func TestNoPlayerCatchUpCanQueueBoundedNPCProduction(t *testing.T) {
	now := time.Now()
	room := testRoom(npcProductionTestMap(160, 160))
	room.world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 444,
		NPCKey:      "cookie-keeper",
		DisplayName: "Cookie Keeper",
		AvatarID:    "keeper",
	}})
	room.npcPausedAt = now.Add(-10 * time.Minute)

	room.catchUpNPCsAfterNoPlayersLocked(now)

	select {
	case event := <-room.npcJobEvents:
		if event.npcKey != "cookie-keeper" || event.jobKey != "shopkeeper_stock" {
			t.Fatalf("production event = %#v, want bounded catch-up stock event", event)
		}
	default:
		t.Fatal("expected bounded catch-up production event")
	}
}

func TestNPCDebugIncludesProductionState(t *testing.T) {
	room := testRoom(npcProductionTestMap(160, 160))
	room.world.setNPCCharacters([]hqclient.NPCCharacterResponse{{
		CharacterID: 444,
		NPCKey:      "cookie-keeper",
		DisplayName: "Cookie Keeper",
		AvatarID:    "keeper",
	}})

	debugNPC := requireDebugNPC(t, room.world.npcDebugSnapshot(time.Now()), defaultMapID, "cookie-keeper")

	if !debugNPC.Production.Eligible || debugNPC.Production.JobKey != "shopkeeper_stock" || debugNPC.Production.OutputKey != "shop_stock_progress" {
		t.Fatalf("debug production = %#v, want eligible shopkeeper production", debugNPC.Production)
	}
}

func npcProductionTestMap(npcX float64, npcY float64) gameMap {
	return gameMap{
		ID:       defaultMapID,
		Name:     "Production Test Town",
		TileSize: 32,
		Width:    10,
		Height:   10,
		Spawns:   []point{{X: 100, Y: 100}},
		Locations: []stmaps.Location{{
			ID:          "cookie-keeper-counter",
			Name:        "Cookie Keeper Counter",
			X:           160,
			Y:           160,
			Radius:      24,
			Tags:        []string{"work", "shop", "merchant"},
			OwnerNPCKey: "cookie-keeper",
		}},
		NPCs: []npc{{
			ID:        "cookie-keeper",
			Name:      "Cookie Keeper",
			X:         npcX,
			Y:         npcY,
			Facing:    "down",
			SpriteKey: "keeper",
			Dialogue:  []string{"Cookies."},
			Shop: &shop{
				ID: "cookie-keeper-shop",
				Items: []shopItem{{
					ItemKey:     "cookie",
					Name:        "Cookie",
					Description: "A treat.",
					PriceStars:  50,
				}},
			},
		}},
	}
}
