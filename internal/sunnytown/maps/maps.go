package maps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GameMap struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	TileSize      int                      `json:"tileSize"`
	Width         int                      `json:"width"`
	Height        int                      `json:"height"`
	Spawns        []Point                  `json:"spawns"`
	BlockedRects  []Rect                   `json:"blockedRects"`
	StarSpawns    []Point                  `json:"starSpawns"`
	Portals       []Portal                 `json:"portals"`
	NPCs          []NPC                    `json:"npcs"`
	ResourceNodes []ResourceNodeDefinition `json:"resourceNodes"`
	Locations     []Location               `json:"locations"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Portal struct {
	ID           string  `json:"id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	TargetMapID  string  `json:"targetMapId"`
	TargetX      float64 `json:"targetX"`
	TargetY      float64 `json:"targetY"`
	TargetFacing string  `json:"targetFacing"`
}

type Location struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	X           float64  `json:"x"`
	Y           float64  `json:"y"`
	Radius      float64  `json:"radius"`
	Tags        []string `json:"tags"`
	OwnerNPCKey string   `json:"ownerNpcKey,omitempty"`
	Capacity    int      `json:"capacity,omitempty"`
}

type NPC struct {
	ID          string    `json:"id"`
	CharacterID int64     `json:"characterId,omitempty"`
	Name        string    `json:"name"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Facing      string    `json:"facing"`
	SpriteKey   string    `json:"spriteKey"`
	Dialogue    []string  `json:"dialogue"`
	Shop        *Shop     `json:"shop,omitempty"`
	Activity    *Activity `json:"activity,omitempty"`
}

type Activity struct {
	Type string `json:"type"`
}

type Shop struct {
	ID    string     `json:"id"`
	Items []ShopItem `json:"items"`
}

type ShopItem struct {
	ItemKey     string `json:"itemKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceStars  int    `json:"priceStars"`
}

type ResourceNodeDefinition struct {
	ID                string  `json:"id"`
	Kind              string  `json:"kind"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Radius            float64 `json:"radius"`
	InteractionRadius float64 `json:"interactionRadius"`
	RespawnSeconds    int     `json:"respawnSeconds"`
}

func LoadMap(path string) (GameMap, error) {
	file, err := os.Open(path)
	if err != nil {
		return GameMap{}, err
	}
	defer file.Close()

	var loaded GameMap
	if err := json.NewDecoder(file).Decode(&loaded); err != nil {
		return GameMap{}, err
	}
	if loaded.ID == "" || loaded.TileSize < 1 || loaded.Width < 1 || loaded.Height < 1 {
		return GameMap{}, errors.New("map is missing required dimensions")
	}
	for _, portal := range loaded.Portals {
		if portal.ID == "" || portal.Width <= 0 || portal.Height <= 0 || portal.TargetMapID == "" {
			return GameMap{}, fmt.Errorf("map %q has an invalid portal", loaded.ID)
		}
		if !IsFacing(portal.TargetFacing) {
			return GameMap{}, fmt.Errorf("map %q portal %q has invalid target facing", loaded.ID, portal.ID)
		}
	}
	if err := validateLocations(loaded); err != nil {
		return GameMap{}, err
	}
	npcIDs := map[string]bool{}
	for _, loadedNPC := range loaded.NPCs {
		if loadedNPC.ID == "" || loadedNPC.Name == "" || len(loadedNPC.Dialogue) == 0 {
			return GameMap{}, fmt.Errorf("map %q has an invalid npc", loaded.ID)
		}
		if npcIDs[loadedNPC.ID] {
			return GameMap{}, fmt.Errorf("map %q has duplicate npc id %q", loaded.ID, loadedNPC.ID)
		}
		npcIDs[loadedNPC.ID] = true
		if loadedNPC.Facing != "" && !IsFacing(loadedNPC.Facing) {
			return GameMap{}, fmt.Errorf("map %q npc %q has invalid facing", loaded.ID, loadedNPC.ID)
		}
		for _, line := range loadedNPC.Dialogue {
			if strings.TrimSpace(line) == "" {
				return GameMap{}, fmt.Errorf("map %q npc %q has blank dialogue", loaded.ID, loadedNPC.ID)
			}
		}
		if loadedNPC.Shop != nil {
			if loadedNPC.Shop.ID == "" || len(loadedNPC.Shop.Items) == 0 {
				return GameMap{}, fmt.Errorf("map %q npc %q has an invalid shop", loaded.ID, loadedNPC.ID)
			}
			shopItemKeys := map[string]bool{}
			for _, item := range loadedNPC.Shop.Items {
				if item.ItemKey == "" || item.Name == "" || item.PriceStars < 1 {
					return GameMap{}, fmt.Errorf("map %q npc %q has an invalid shop item", loaded.ID, loadedNPC.ID)
				}
				if shopItemKeys[item.ItemKey] {
					return GameMap{}, fmt.Errorf("map %q npc %q has duplicate shop item %q", loaded.ID, loadedNPC.ID, item.ItemKey)
				}
				shopItemKeys[item.ItemKey] = true
			}
		}
		if loadedNPC.Activity != nil && loadedNPC.Activity.Type != "schoolwork" {
			return GameMap{}, fmt.Errorf("map %q npc %q has invalid activity type %q", loaded.ID, loadedNPC.ID, loadedNPC.Activity.Type)
		}
	}
	resourceNodeIDs := map[string]bool{}
	for _, node := range loaded.ResourceNodes {
		if node.ID == "" || node.Kind == "" || node.X < 0 || node.Y < 0 || node.Radius <= 0 || node.InteractionRadius <= 0 || node.RespawnSeconds < 1 {
			return GameMap{}, fmt.Errorf("map %q has an invalid resource node", loaded.ID)
		}
		if resourceNodeIDs[node.ID] {
			return GameMap{}, fmt.Errorf("map %q has duplicate resource node id %q", loaded.ID, node.ID)
		}
		resourceNodeIDs[node.ID] = true
		if node.Kind != "rock" {
			return GameMap{}, fmt.Errorf("map %q resource node %q has unsupported kind %q", loaded.ID, node.ID, node.Kind)
		}
	}
	return loaded, nil
}

func validateLocations(loaded GameMap) error {
	locationIDs := map[string]bool{}
	maxX := float64(loaded.Width * loaded.TileSize)
	maxY := float64(loaded.Height * loaded.TileSize)
	for _, location := range loaded.Locations {
		if strings.TrimSpace(location.ID) == "" {
			return fmt.Errorf("map %q has a location with blank id", loaded.ID)
		}
		if locationIDs[location.ID] {
			return fmt.Errorf("map %q has duplicate location id %q", loaded.ID, location.ID)
		}
		locationIDs[location.ID] = true
		if location.X < 0 || location.Y < 0 || location.X > maxX || location.Y > maxY {
			return fmt.Errorf("map %q location %q is outside map bounds", loaded.ID, location.ID)
		}
		if location.Radius <= 0 {
			return fmt.Errorf("map %q location %q has invalid radius", loaded.ID, location.ID)
		}
		if len(location.Tags) == 0 {
			return fmt.Errorf("map %q location %q has no tags", loaded.ID, location.ID)
		}
		for _, tag := range location.Tags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("map %q location %q has a blank tag", loaded.ID, location.ID)
			}
		}
		if location.Capacity < 0 {
			return fmt.Errorf("map %q location %q has invalid capacity", loaded.ID, location.ID)
		}
	}
	return nil
}

func LoadMaps(dir string) (map[string]GameMap, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	maps := map[string]GameMap{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		loaded, err := LoadMap(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, exists := maps[loaded.ID]; exists {
			return nil, fmt.Errorf("duplicate map id %q", loaded.ID)
		}
		maps[loaded.ID] = loaded
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("no maps found in %s", dir)
	}
	for _, loaded := range maps {
		for _, portal := range loaded.Portals {
			if _, ok := maps[portal.TargetMapID]; !ok {
				return nil, fmt.Errorf("map %q portal %q targets unknown map %q", loaded.ID, portal.ID, portal.TargetMapID)
			}
		}
	}
	return maps, nil
}

func IsFacing(value string) bool {
	return value == "up" || value == "down" || value == "left" || value == "right"
}
