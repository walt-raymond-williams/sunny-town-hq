package navigation

import (
	"container/heap"
	"errors"
	"fmt"
	"math"

	stmaps "hq/internal/sunnytown/maps"
)

type Point = stmaps.Point
type Rect = stmaps.Rect

const defaultAgentSize = 28.0

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
	Path         []Point
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
		path, err := graph.PlanPath(startMapID, start, target)
		if err != nil {
			return Route{}, err
		}
		return Route{Steps: []RouteStep{{
			MapID: startMapID,
			From:  start,
			To:    target,
			Path:  path,
		}}}, nil
	}

	portalPath, ok := graph.portalPath(startMapID, targetMapID)
	if !ok {
		return Route{}, fmt.Errorf("no portal route from map %q to map %q", startMapID, targetMapID)
	}

	steps := make([]RouteStep, 0, len(portalPath)+1)
	currentFrom := start
	for _, edge := range portalPath {
		path, err := graph.PlanPath(edge.SourceMapID, currentFrom, edge.SourceCenter)
		if err != nil {
			return Route{}, err
		}
		steps = append(steps, RouteStep{
			MapID:        edge.SourceMapID,
			From:         currentFrom,
			To:           edge.SourceCenter,
			Path:         path,
			PortalID:     edge.PortalID,
			TargetMapID:  edge.TargetMapID,
			TargetFacing: edge.TargetFacing,
		})
		currentFrom = edge.TargetPoint
	}
	path, err := graph.PlanPath(targetMapID, currentFrom, target)
	if err != nil {
		return Route{}, err
	}
	steps = append(steps, RouteStep{
		MapID: targetMapID,
		From:  currentFrom,
		To:    target,
		Path:  path,
	})
	return Route{Steps: steps}, nil
}

func (graph *Graph) PlanPath(mapID string, start Point, target Point) ([]Point, error) {
	if graph == nil {
		return nil, errors.New("navigation graph is nil")
	}
	gameMap, ok := graph.maps[mapID]
	if !ok {
		return nil, fmt.Errorf("unknown map %q", mapID)
	}
	grid, err := newNavGrid(gameMap, defaultAgentSize)
	if err != nil {
		return nil, err
	}
	return grid.plan(start, target)
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

type gridCell struct {
	x int
	y int
}

type navGrid struct {
	gameMap   stmaps.GameMap
	cellSize  float64
	agentSize float64
	width     int
	height    int
	blocked   [][]bool
}

func newNavGrid(gameMap stmaps.GameMap, agentSize float64) (navGrid, error) {
	if gameMap.TileSize < 1 || gameMap.Width < 1 || gameMap.Height < 1 {
		return navGrid{}, fmt.Errorf("map %q has invalid navigation dimensions", gameMap.ID)
	}
	grid := navGrid{
		gameMap:   gameMap,
		cellSize:  float64(gameMap.TileSize),
		agentSize: agentSize,
		width:     gameMap.Width,
		height:    gameMap.Height,
		blocked:   make([][]bool, gameMap.Height),
	}
	for y := 0; y < gameMap.Height; y++ {
		grid.blocked[y] = make([]bool, gameMap.Width)
		for x := 0; x < gameMap.Width; x++ {
			grid.blocked[y][x] = grid.cellBlocked(gridCell{x: x, y: y})
		}
	}
	return grid, nil
}

func (grid navGrid) plan(start Point, target Point) ([]Point, error) {
	startCell, ok := grid.pointCell(start)
	if !ok {
		return nil, fmt.Errorf("start point %.2f,%.2f is outside map %q", start.X, start.Y, grid.gameMap.ID)
	}
	targetCell, ok := grid.pointCell(target)
	if !ok {
		return nil, fmt.Errorf("target point %.2f,%.2f is outside map %q", target.X, target.Y, grid.gameMap.ID)
	}
	if !grid.walkable(startCell) {
		return nil, fmt.Errorf("start point %.2f,%.2f is blocked on map %q", start.X, start.Y, grid.gameMap.ID)
	}
	if !grid.walkable(targetCell) {
		return nil, fmt.Errorf("target point %.2f,%.2f is blocked on map %q", target.X, target.Y, grid.gameMap.ID)
	}
	if startCell == targetCell {
		return []Point{start, target}, nil
	}

	cells, ok := grid.aStar(startCell, targetCell)
	if !ok {
		return nil, fmt.Errorf("no walkable path on map %q from %.2f,%.2f to %.2f,%.2f", grid.gameMap.ID, start.X, start.Y, target.X, target.Y)
	}

	points := make([]Point, 0, len(cells)+2)
	points = append(points, start)
	for _, cell := range cells[1 : len(cells)-1] {
		points = append(points, grid.cellCenter(cell))
	}
	points = append(points, target)
	return points, nil
}

func (grid navGrid) aStar(start gridCell, target gridCell) ([]gridCell, bool) {
	open := &cellPriorityQueue{}
	heap.Init(open)
	heap.Push(open, &cellQueueItem{cell: start, priority: grid.heuristic(start, target)})

	cameFrom := map[gridCell]gridCell{}
	costSoFar := map[gridCell]float64{start: 0}
	closed := map[gridCell]bool{}

	for open.Len() > 0 {
		current := heap.Pop(open).(*cellQueueItem).cell
		if closed[current] {
			continue
		}
		if current == target {
			return reconstructCells(cameFrom, start, target), true
		}
		closed[current] = true

		for _, next := range grid.neighbors(current) {
			newCost := costSoFar[current] + 1
			if existingCost, ok := costSoFar[next]; ok && newCost >= existingCost {
				continue
			}
			costSoFar[next] = newCost
			cameFrom[next] = current
			heap.Push(open, &cellQueueItem{
				cell:     next,
				priority: newCost + grid.heuristic(next, target),
			})
		}
	}
	return nil, false
}

func reconstructCells(cameFrom map[gridCell]gridCell, start gridCell, target gridCell) []gridCell {
	path := []gridCell{target}
	for current := target; current != start; {
		current = cameFrom[current]
		path = append(path, current)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func (grid navGrid) neighbors(cell gridCell) []gridCell {
	candidates := []gridCell{
		{x: cell.x, y: cell.y - 1},
		{x: cell.x + 1, y: cell.y},
		{x: cell.x, y: cell.y + 1},
		{x: cell.x - 1, y: cell.y},
	}
	neighbors := make([]gridCell, 0, len(candidates))
	for _, candidate := range candidates {
		if grid.walkable(candidate) {
			neighbors = append(neighbors, candidate)
		}
	}
	return neighbors
}

func (grid navGrid) heuristic(a gridCell, b gridCell) float64 {
	return math.Abs(float64(a.x-b.x)) + math.Abs(float64(a.y-b.y))
}

func (grid navGrid) pointCell(point Point) (gridCell, bool) {
	maxX := float64(grid.width) * grid.cellSize
	maxY := float64(grid.height) * grid.cellSize
	if point.X < 0 || point.Y < 0 || point.X > maxX || point.Y > maxY {
		return gridCell{}, false
	}
	x := int(math.Floor(point.X / grid.cellSize))
	y := int(math.Floor(point.Y / grid.cellSize))
	if x == grid.width {
		x = grid.width - 1
	}
	if y == grid.height {
		y = grid.height - 1
	}
	return gridCell{x: x, y: y}, true
}

func (grid navGrid) walkable(cell gridCell) bool {
	if cell.x < 0 || cell.y < 0 || cell.x >= grid.width || cell.y >= grid.height {
		return false
	}
	return !grid.blocked[cell.y][cell.x]
}

func (grid navGrid) cellCenter(cell gridCell) Point {
	return Point{
		X: (float64(cell.x) + 0.5) * grid.cellSize,
		Y: (float64(cell.y) + 0.5) * grid.cellSize,
	}
}

func (grid navGrid) cellBlocked(cell gridCell) bool {
	center := grid.cellCenter(cell)
	agentRect := Rect{
		X:      center.X - grid.agentSize/2,
		Y:      center.Y - grid.agentSize/2,
		Width:  grid.agentSize,
		Height: grid.agentSize,
	}
	for _, blocked := range grid.gameMap.BlockedRects {
		if rectsOverlap(agentRect, blocked) {
			return true
		}
	}
	return false
}

func rectsOverlap(a Rect, b Rect) bool {
	return a.X < b.X+b.Width &&
		a.X+a.Width > b.X &&
		a.Y < b.Y+b.Height &&
		a.Y+a.Height > b.Y
}

type cellQueueItem struct {
	cell     gridCell
	priority float64
	index    int
}

type cellPriorityQueue []*cellQueueItem

func (queue cellPriorityQueue) Len() int {
	return len(queue)
}

func (queue cellPriorityQueue) Less(i int, j int) bool {
	return queue[i].priority < queue[j].priority
}

func (queue cellPriorityQueue) Swap(i int, j int) {
	queue[i], queue[j] = queue[j], queue[i]
	queue[i].index = i
	queue[j].index = j
}

func (queue *cellPriorityQueue) Push(item any) {
	queued := item.(*cellQueueItem)
	queued.index = len(*queue)
	*queue = append(*queue, queued)
}

func (queue *cellPriorityQueue) Pop() any {
	old := *queue
	item := old[len(old)-1]
	old[len(old)-1] = nil
	item.index = -1
	*queue = old[:len(old)-1]
	return item
}
