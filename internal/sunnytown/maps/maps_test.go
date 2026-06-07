package maps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMapsRejectsDuplicateMapIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"duplicate","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate map ID to be rejected")
	}
}

func TestLoadMapsRejectsUnknownPortalTarget(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"portals":[{"id":"door","x":64,"y":64,"width":32,"height":32,"targetMapId":"missing","targetX":64,"targetY":64,"targetFacing":"down"}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected unknown portal target to be rejected")
	}
}

func TestLoadMapsAcceptsCheckedInMaps(t *testing.T) {
	maps, err := LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}

	classroom, ok := maps["sunny-town-classroom"]
	if !ok {
		t.Fatal("expected sunny-town-classroom map to load")
	}
	if len(classroom.NPCs) != 1 || classroom.NPCs[0].Activity == nil || classroom.NPCs[0].Activity.Type != "schoolwork" {
		t.Fatalf("classroom npcs = %#v, want teacher schoolwork npc", classroom.NPCs)
	}
	forest, ok := maps["forest-crossing-v1"]
	if !ok {
		t.Fatal("expected forest-crossing-v1 map to load")
	}
	if len(forest.ResourceNodes) < 2 {
		t.Fatalf("forest resource nodes = %#v, want at least two nodes", forest.ResourceNodes)
	}
}

func TestLoadMapsRejectsDuplicateResourceNodeIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"resourceNodes":[{"id":"rock-node","kind":"rock","x":64,"y":64,"radius":24,"interactionRadius":48,"respawnSeconds":15},{"id":"rock-node","kind":"rock","x":96,"y":64,"radius":24,"interactionRadius":48,"respawnSeconds":15}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate resource node ID to be rejected")
	}
}

func TestLoadMapsRejectsInvalidResourceNodes(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"resourceNodes":[{"id":"rock-node","kind":"rock","x":64,"y":64,"radius":0,"interactionRadius":48,"respawnSeconds":15}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected invalid resource node to be rejected")
	}
}

func TestLoadMapsAcceptsNPCDefinitions(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"activity":{"type":"schoolwork"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	maps, err := LoadMaps(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(maps["one"].NPCs) != 1 || maps["one"].NPCs[0].ID != "guide" {
		t.Fatalf("loaded npcs = %#v, want guide", maps["one"].NPCs)
	}
	if maps["one"].NPCs[0].Activity == nil || maps["one"].NPCs[0].Activity.Type != "schoolwork" {
		t.Fatalf("loaded npc activity = %#v, want schoolwork", maps["one"].NPCs[0].Activity)
	}
}

func TestLoadMapsRejectsDuplicateNPCIDs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."]},{"id":"guide","name":"Guide Again","x":96,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hi."]}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected duplicate npc ID to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCs(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."]}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCActivity(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"activity":{"type":"unknown"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc activity to be rejected")
	}
}

func TestLoadMapsRejectsInvalidNPCShop(t *testing.T) {
	dir := t.TempDir()
	mapJSON := `{"id":"one","name":"One","tileSize":32,"width":4,"height":4,"spawns":[{"x":64,"y":64}],"blockedRects":[],"starSpawns":[],"npcs":[{"id":"guide","name":"Guide","x":64,"y":64,"facing":"down","spriteKey":"guide","dialogue":["Hello."],"shop":{"id":"guide-shop","items":[{"itemKey":"cookie","name":"Cookie","description":"A treat.","priceStars":0}]}}]}`
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadMaps(dir); err == nil {
		t.Fatal("expected invalid npc shop to be rejected")
	}
}
