package server

import (
	"path/filepath"
	"testing"

	stmaps "hq/internal/sunnytown/maps"
)

func TestNewRoomInitializesMapFixturesAsWorldObjects(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}

	room := newRoom("room-1", maps["sunny-town-house-1"], make(chan rewardEvent, 1), make(chan resourceEvent, 1), make(chan npcJobProductionEvent, 1), nil)
	object := room.worldObjects[worldObjectKey(worldObjectSourceFixture, "cookie-shop-output-chest")]
	if object == nil {
		t.Fatal("expected cookie shop output chest fixture world object")
	}
	if object.kind != worldObjectKindChest || object.shopID != "cookie-keeper-shop" || object.storageRole != "output" || object.itemKey != "cookie" {
		t.Fatalf("fixture object = %#v, want chest-backed Cookie Shop output storage", object)
	}
	if !object.active || !object.collision || !object.reservesPlacement || object.breakable {
		t.Fatalf("fixture object flags = active:%v collision:%v reserves:%v breakable:%v, want active blocking non-breakable chest", object.active, object.collision, object.reservesPlacement, object.breakable)
	}

	snapshot := object.worldObjectSnapshot()
	if snapshot.Source != worldObjectSourceFixture || snapshot.Name != "Cookie Shop Output Chest" || snapshot.LocationID != "cookie-shop" || snapshot.ShopID != "cookie-keeper-shop" || snapshot.StorageRole != "output" || snapshot.ItemKey != "cookie" || snapshot.InteractionRadius != 56 {
		t.Fatalf("fixture snapshot = %#v, want output chest metadata", snapshot)
	}

	inputObject := room.worldObjects[worldObjectKey(worldObjectSourceFixture, "cookie-shop-input-chest")]
	if inputObject == nil {
		t.Fatal("expected cookie shop input chest fixture world object")
	}
	if inputObject.kind != worldObjectKindChest || inputObject.shopID != "cookie-keeper-shop" || inputObject.storageRole != "input" || inputObject.itemKey != "" {
		t.Fatalf("input fixture object = %#v, want chest-backed Cookie Shop input storage identity", inputObject)
	}
	inputSnapshot := inputObject.worldObjectSnapshot()
	if inputSnapshot.Source != worldObjectSourceFixture || inputSnapshot.Name != "Cookie Shop Input Chest" || inputSnapshot.LocationID != "cookie-shop" || inputSnapshot.ShopID != "cookie-keeper-shop" || inputSnapshot.StorageRole != "input" || inputSnapshot.ItemKey != "" || inputSnapshot.InteractionRadius != 56 {
		t.Fatalf("input fixture snapshot = %#v, want input chest metadata", inputSnapshot)
	}
}

func TestValidatedContainerAccessUsesServerWorldStateAndAcceptedPosition(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}

	room := newRoom("sunny-town-main", maps["sunny-town-house-1"], make(chan rewardEvent, 1), make(chan resourceEvent, 1), make(chan npcJobProductionEvent, 1), nil)
	player := &player{id: "player-1", appUserID: 123, x: 368, y: 304}

	containerID, ok := room.validatedContainerAccessLocked(player, worldObjectSourceFixture, "cookie-shop-input-chest", "deposit")
	if !ok {
		t.Fatal("expected nearby player to access input chest for deposit")
	}
	if containerID != "fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest" {
		t.Fatalf("containerID = %q, want stable fixture container id", containerID)
	}

	if _, ok := room.validatedContainerAccessLocked(player, worldObjectSourceFixture, "cookie-shop-output-chest", "withdraw"); ok {
		t.Fatal("expected output chest withdraw to be rejected")
	}
	player.x = 64
	player.y = 64
	if _, ok := room.validatedContainerAccessLocked(player, worldObjectSourceFixture, "cookie-shop-input-chest", "deposit"); ok {
		t.Fatal("expected far player to be rejected")
	}
	if _, ok := room.validatedContainerAccessLocked(player, worldObjectSourceFixture, "missing", "deposit"); ok {
		t.Fatal("expected missing client-supplied object id to be rejected")
	}
}
