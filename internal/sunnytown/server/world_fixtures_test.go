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
	if snapshot.Source != worldObjectSourceFixture || snapshot.Name != "Cookie Shop Output Chest" || snapshot.LocationID != "cookie-shop" || snapshot.ShopID != "cookie-keeper-shop" || snapshot.StorageRole != "output" || snapshot.ItemKey != "cookie" {
		t.Fatalf("fixture snapshot = %#v, want output chest metadata", snapshot)
	}
}
