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
	if len(step.Path) < 2 || step.Path[0] != step.From || step.Path[len(step.Path)-1] != step.To {
		t.Fatalf("same-map path = %#v, want endpoints from route step", step.Path)
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
	if len(portalStep.Path) < 2 || portalStep.Path[len(portalStep.Path)-1] != portalStep.To {
		t.Fatalf("portal step path = %#v, want path ending at portal", portalStep.Path)
	}
	if len(finalStep.Path) < 2 || finalStep.Path[0] != finalStep.From || finalStep.Path[len(finalStep.Path)-1] != finalStep.To {
		t.Fatalf("final step path = %#v, want endpoints from route step", finalStep.Path)
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

func TestPlanPathStraightReachable(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {ID: "one", TileSize: 32, Width: 4, Height: 4},
	})

	path, err := graph.PlanPath("one", Point{X: 16, Y: 48}, Point{X: 112, Y: 48})
	if err != nil {
		t.Fatal(err)
	}

	if len(path) < 2 {
		t.Fatalf("path = %#v, want at least start and target", path)
	}
	if path[0] != (Point{X: 16, Y: 48}) || path[len(path)-1] != (Point{X: 112, Y: 48}) {
		t.Fatalf("path endpoints = %#v, want requested start and target", path)
	}
}

func TestPlanPathRoutesAroundBlockedRect(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    5,
			Height:   5,
			BlockedRects: []stmaps.Rect{{
				X:      64,
				Y:      64,
				Width:  32,
				Height: 32,
			}},
		},
	})

	path, err := graph.PlanPath("one", Point{X: 48, Y: 80}, Point{X: 112, Y: 80})
	if err != nil {
		t.Fatal(err)
	}

	if len(path) <= 2 {
		t.Fatalf("path = %#v, want detour waypoints", path)
	}
	for _, point := range path {
		if point == (Point{X: 80, Y: 80}) {
			t.Fatalf("path = %#v, should avoid blocked cell center", path)
		}
	}
}

func TestPlanPathRoutesAroundExtraBlockedRect(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    5,
			Height:   5,
		},
	})

	path, err := graph.PlanPathWithBlockedRects("one", Point{X: 48, Y: 80}, Point{X: 112, Y: 80}, []Rect{{
		X:      64,
		Y:      64,
		Width:  32,
		Height: 32,
	}})
	if err != nil {
		t.Fatal(err)
	}

	if len(path) <= 2 {
		t.Fatalf("path = %#v, want detour waypoints", path)
	}
	for _, point := range path {
		if point == (Point{X: 80, Y: 80}) {
			t.Fatalf("path = %#v, should avoid extra blocked cell center", path)
		}
	}
}

func TestPlanPathFailsWhenTargetUnreachable(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    3,
			Height:   3,
			BlockedRects: []stmaps.Rect{{
				X:      32,
				Y:      0,
				Width:  32,
				Height: 96,
			}},
		},
	})

	if _, err := graph.PlanPath("one", Point{X: 16, Y: 16}, Point{X: 80, Y: 16}); err == nil {
		t.Fatal("expected unreachable target to fail")
	}
}

func TestPlanPathFailsWhenExtraBlockedRectCutsOffTarget(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    3,
			Height:   3,
		},
	})

	if _, err := graph.PlanPathWithBlockedRects("one", Point{X: 16, Y: 16}, Point{X: 80, Y: 16}, []Rect{{
		X:      32,
		Y:      0,
		Width:  32,
		Height: 96,
	}}); err == nil {
		t.Fatal("expected extra blocked rect to make target unreachable")
	}
}

func TestPlanRoutePathToPortalEntryPoint(t *testing.T) {
	graph := mustTestGraph(t, map[string]stmaps.GameMap{
		"one": {
			ID:       "one",
			TileSize: 32,
			Width:    6,
			Height:   5,
			Portals: []stmaps.Portal{{
				ID:           "door",
				X:            128,
				Y:            64,
				Width:        32,
				Height:       32,
				TargetMapID:  "two",
				TargetX:      16,
				TargetY:      16,
				TargetFacing: "down",
			}},
		},
		"two": {ID: "two", TileSize: 32, Width: 4, Height: 4},
	})

	route, err := graph.PlanRoute("one", Point{X: 16, Y: 80}, "two", Point{X: 80, Y: 16})
	if err != nil {
		t.Fatal(err)
	}

	if len(route.Steps) != 2 {
		t.Fatalf("route = %#v, want portal step plus final step", route.Steps)
	}
	portalPath := route.Steps[0].Path
	if len(portalPath) < 2 || portalPath[len(portalPath)-1] != (Point{X: 144, Y: 80}) {
		t.Fatalf("portal path = %#v, want path to portal center", portalPath)
	}
}

func mustTestGraph(t *testing.T, maps map[string]stmaps.GameMap) *Graph {
	t.Helper()
	graph, err := NewGraph(maps)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}
