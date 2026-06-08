package navigation

import (
	"errors"
	"fmt"

	stmaps "hq/internal/sunnytown/maps"
)

type Point = stmaps.Point
type Rect = stmaps.Rect

type Graph struct {
	maps  map[string]stmaps.GameMap
	edges map[string][]PortalEdge
}

type PortalEdge struct {
	PortalID     string
	SourceMapID  string
	SourceRect   Rect
	SourceCenter Point
	TargetMapID  string
	TargetPoint  Point
	TargetFacing string
}

type Route struct {
	Steps []RouteStep
}

type RouteStep struct {
	MapID        string
	From         Point
	To           Point
	PortalID     string
	TargetMapID  string
	TargetFacing string
}

func NewGraph(maps map[string]stmaps.GameMap) (*Graph, error) {
	if len(maps) == 0 {
		return nil, errors.New("navigation graph requires at least one map")
	}

	copiedMaps := make(map[string]stmaps.GameMap, len(maps))
	edges := make(map[string][]PortalEdge, len(maps))
	for mapID, gameMap := range maps {
		if mapID == "" || gameMap.ID == "" {
			return nil, errors.New("navigation graph map IDs must be non-empty")
		}
		if mapID != gameMap.ID {
			return nil, fmt.Errorf("navigation graph map key %q does not match map id %q", mapID, gameMap.ID)
		}
		copiedMaps[mapID] = gameMap
		for _, portal := range gameMap.Portals {
			if _, ok := maps[portal.TargetMapID]; !ok {
				return nil, fmt.Errorf("map %q portal %q targets unknown map %q", gameMap.ID, portal.ID, portal.TargetMapID)
			}
			edges[mapID] = append(edges[mapID], PortalEdge{
				PortalID:    portal.ID,
				SourceMapID: gameMap.ID,
				SourceRect: Rect{
					X:      portal.X,
					Y:      portal.Y,
					Width:  portal.Width,
					Height: portal.Height,
				},
				SourceCenter: Point{
					X: portal.X + portal.Width/2,
					Y: portal.Y + portal.Height/2,
				},
				TargetMapID:  portal.TargetMapID,
				TargetPoint:  Point{X: portal.TargetX, Y: portal.TargetY},
				TargetFacing: portal.TargetFacing,
			})
		}
	}

	return &Graph{
		maps:  copiedMaps,
		edges: edges,
	}, nil
}

func (graph *Graph) PortalEdges(mapID string) []PortalEdge {
	if graph == nil {
		return nil
	}
	edges := graph.edges[mapID]
	copied := make([]PortalEdge, len(edges))
	copy(copied, edges)
	return copied
}

func (graph *Graph) Location(mapID string, locationID string) (stmaps.Location, bool) {
	if graph == nil {
		return stmaps.Location{}, false
	}
	gameMap, ok := graph.maps[mapID]
	if !ok {
		return stmaps.Location{}, false
	}
	for _, location := range gameMap.Locations {
		if location.ID == locationID {
			return location, true
		}
	}
	return stmaps.Location{}, false
}

func (graph *Graph) PlanRouteToLocation(startMapID string, start Point, targetMapID string, locationID string) (Route, error) {
	location, ok := graph.Location(targetMapID, locationID)
	if !ok {
		return Route{}, fmt.Errorf("unknown location %q on map %q", locationID, targetMapID)
	}
	return graph.PlanRoute(startMapID, start, targetMapID, Point{X: location.X, Y: location.Y})
}

func (graph *Graph) PlanRoute(startMapID string, start Point, targetMapID string, target Point) (Route, error) {
	if graph == nil {
		return Route{}, errors.New("navigation graph is nil")
	}
	if _, ok := graph.maps[startMapID]; !ok {
		return Route{}, fmt.Errorf("unknown start map %q", startMapID)
	}
	if _, ok := graph.maps[targetMapID]; !ok {
		return Route{}, fmt.Errorf("unknown target map %q", targetMapID)
	}
	if startMapID == targetMapID {
		return Route{Steps: []RouteStep{{
			MapID: startMapID,
			From:  start,
			To:    target,
		}}}, nil
	}

	portalPath, ok := graph.portalPath(startMapID, targetMapID)
	if !ok {
		return Route{}, fmt.Errorf("no portal route from map %q to map %q", startMapID, targetMapID)
	}

	steps := make([]RouteStep, 0, len(portalPath)+1)
	currentFrom := start
	for _, edge := range portalPath {
		steps = append(steps, RouteStep{
			MapID:        edge.SourceMapID,
			From:         currentFrom,
			To:           edge.SourceCenter,
			PortalID:     edge.PortalID,
			TargetMapID:  edge.TargetMapID,
			TargetFacing: edge.TargetFacing,
		})
		currentFrom = edge.TargetPoint
	}
	steps = append(steps, RouteStep{
		MapID: targetMapID,
		From:  currentFrom,
		To:    target,
	})
	return Route{Steps: steps}, nil
}

func (graph *Graph) portalPath(startMapID string, targetMapID string) ([]PortalEdge, bool) {
	type queuedMap struct {
		mapID string
		path  []PortalEdge
	}

	visited := map[string]bool{startMapID: true}
	queue := []queuedMap{{mapID: startMapID}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range graph.edges[current.mapID] {
			if visited[edge.TargetMapID] {
				continue
			}
			nextPath := append(append([]PortalEdge(nil), current.path...), edge)
			if edge.TargetMapID == targetMapID {
				return nextPath, true
			}
			visited[edge.TargetMapID] = true
			queue = append(queue, queuedMap{
				mapID: edge.TargetMapID,
				path:  nextPath,
			})
		}
	}
	return nil, false
}
