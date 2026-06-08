package navigation

import (
	"path/filepath"
	"testing"

	stmaps "hq/internal/sunnytown/maps"
)

func TestPlanRouteToSameMapLocation(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := NewGraph(maps)
	if err != nil {
		t.Fatal(err)
	}

	route, err := graph.PlanRouteToLocation("sunny-town-v1", Point{X: 736, Y: 512}, "sunny-town-v1", "town-square-center")
	if err != nil {
		t.Fatal(err)
	}

	if len(route.Steps) != 1 {
		t.Fatalf("route steps = %#v, want one same-map step", route.Steps)
	}
	step := route.Steps[0]
	if step.MapID != "sunny-town-v1" || step.PortalID != "" {
		t.Fatalf("same-map step = %#v, want sunny-town-v1 without portal", step)
	}
	if step.To != (Point{X: 640, Y: 512}) {
		t.Fatalf("same-map destination = %#v, want town-square-center", step.To)
	}
}

func TestPlanRouteUsesExistingForestPortal(t *testing.T) {
	maps, err := stmaps.LoadMaps(filepath.Join("..", "..", "..", "sunny-town", "maps"))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := NewGraph(maps)
	if err != nil {
		t.Fatal(err)
	}

	route, err := graph.PlanRoute("sunny-town-v1", Point{X: 736, Y: 512}, "forest-crossing-v1", Point{X: 704, Y: 352})
	if err != nil {
		t.Fatal(err)
	}

	if len(route.Steps) != 2 {
		t.Fatalf("route steps = %#v, want portal step plus target-map step", route.Steps)
	}
	portalStep := route.Steps[0]
	if portalStep.MapID != "sunny-town-v1" || portalStep.PortalID != "forest-crossing-path" {
		t.Fatalf("portal step = %#v, want sunny town forest portal", portalStep)
	}
	if portalStep.TargetMapID != "forest-crossing-v1" || portalStep.TargetFacing != "right" {
		t.Fatalf("portal target = %#v, want forest-crossing-v1 facing right", portalStep)
	}
	if portalStep.To != (Point{X: 1232, Y: 482}) {
		t.Fatalf("portal center = %#v, want center of forest portal", portalStep.To)
	}
	finalStep := route.Steps[1]
	if finalStep.MapID != "forest-crossing-v1" || finalStep.PortalID != "" {
		t.Fatalf("final step = %#v, want forest target-map step", finalStep)
	}
	if finalStep.From != (Point{X: 96, Y: 480}) || finalStep.To != (Point{X: 704, Y: 352}) {
		t.Fatalf("final step endpoints = %#v, want portal landing to target", finalStep)
	}
}

func TestPlanRouteFailsForUnknownTargetMap(t *testing.T) {
	graph, err := NewGraph(map[string]stmaps.GameMap{
		"one": {ID: "one", TileSize: 32, Width: 4, Height: 4},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := graph.PlanRoute("one", Point{X: 64, Y: 64}, "missing", Point{X: 64, Y: 64}); err == nil {
		t.Fatal("expected unknown target map to fail")
	}
}

func TestPlanRouteFailsForUnconnectedTargetMap(t *testing.T) {
	graph, err := NewGraph(map[string]stmaps.GameMap{
		"one": {ID: "one", TileSize: 32, Width: 4, Height: 4},
		"two": {ID: "two", TileSize: 32, Width: 4, Height: 4},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := graph.PlanRoute("one", Point{X: 64, Y: 64}, "two", Point{X: 64, Y: 64}); err == nil {
		t.Fatal("expected unconnected target map to fail")
	}
}

func TestNewGraphRejectsUnknownPortalTarget(t *testing.T) {
	_, err := NewGraph(map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    4,
			Height:   4,
			Portals: []stmaps.Portal{{
				ID:           "door",
				X:            32,
				Y:            32,
				Width:        32,
				Height:       32,
				TargetMapID:  "missing",
				TargetX:      64,
				TargetY:      64,
				TargetFacing: "down",
			}},
		},
	})
	if err == nil {
		t.Fatal("expected unknown portal target to fail")
	}
}
