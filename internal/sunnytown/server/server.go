package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	stconfig "hq/internal/sunnytown/config"
	"hq/internal/sunnytown/hqclient"
	"hq/internal/sunnytownauth"

	"github.com/gorilla/websocket"
)

type Server struct {
	config   stconfig.Config
	world    *world
	hq       *hqclient.Client
	upgrader websocket.Upgrader
}

func New(cfg stconfig.Config, maps map[string]gameMap) *Server {
	srv := &Server{
		config: cfg,
		world:  newWorldWithNPCDayLength(defaultRoomID, maps, cfg.NPCDayLength),
		hq:     hqclient.New(cfg.HQInternalURL, cfg.ServiceSecret, 3*time.Second),
	}
	srv.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			return origin == "" || cfg.AllowedOrigins["*"] || cfg.AllowedOrigins[origin]
		},
	}
	return srv
}

func (srv *Server) StartRooms(ctx context.Context) {
	for _, room := range srv.world.rooms {
		go room.run(ctx)
	}
}

func (srv *Server) LoadInitialMapObjects(ctx context.Context) error {
	for _, room := range srv.world.rooms {
		loadCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		objects, err := srv.loadMapObjects(loadCtx, room.id, room.gameMap.ID)
		cancel()
		if err != nil {
			return err
		}
		room.mu.Lock()
		room.setPlacedObjectsLocked(objects)
		room.mu.Unlock()
	}
	return nil
}

func (srv *Server) LoadNPCCharacters(ctx context.Context) error {
	inputs := make([]hqclient.EnsureNPCCharacterInput, 0)
	seen := map[string]bool{}
	for _, room := range srv.world.rooms {
		for _, npc := range room.gameMap.NPCs {
			if strings.TrimSpace(npc.ID) == "" || seen[npc.ID] {
				continue
			}
			seen[npc.ID] = true
			inputs = append(inputs, hqclient.EnsureNPCCharacterInput{
				NPCKey:      npc.ID,
				DisplayName: npc.Name,
				AvatarID:    npc.SpriteKey,
			})
		}
	}
	if len(inputs) == 0 {
		return nil
	}

	loaded, err := srv.hq.EnsureNPCCharacters(ctx, hqclient.EnsureNPCCharactersRequest{
		RoomID: srv.world.roomID,
		NPCs:   inputs,
	})
	if err != nil {
		return err
	}

	srv.world.setNPCCharacters(loaded.NPCs)
	return nil
}

func (srv *Server) loadMapObjects(ctx context.Context, roomID string, mapID string) (map[string]*placedObject, error) {
	loaded, err := srv.hq.LoadMapObjects(ctx, roomID, mapID)
	if err != nil {
		return nil, err
	}

	room := srv.world.rooms[mapID]
	if room == nil {
		return nil, fmt.Errorf("unknown map %q", mapID)
	}
	objects := map[string]*placedObject{}
	for _, object := range loaded.Objects {
		placed := placedObjectFromResponse(room.gameMap, object)
		objects[placed.id] = placed
	}
	return objects, nil
}

func (srv *Server) refreshRoomMapObjects(ctx context.Context, mapID string) error {
	room := srv.world.rooms[mapID]
	if room == nil {
		return fmt.Errorf("unknown map %q", mapID)
	}
	objects, err := srv.loadMapObjects(ctx, room.id, room.gameMap.ID)
	if err != nil {
		return err
	}
	room.mu.Lock()
	room.setPlacedObjectsLocked(objects)
	room.mu.Unlock()
	return nil
}

func (srv *Server) placeMapObject(ctx context.Context, request placeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	placed, err := srv.hq.PlaceMapObject(ctx, request)
	if err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, placed), placed.RemainingItemAmount, nil
}

func (srv *Server) removeMapObject(ctx context.Context, request removeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	removed, err := srv.hq.RemoveMapObject(ctx, request)
	if err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, removed), removed.RemainingItemAmount, nil
}

func (srv *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := sunnytownauth.Verify(r.URL.Query().Get("token"), srv.config.JoinSecret, time.Now())
	if err != nil {
		log.Printf("sunny town auth failed: %v", err)
		http.Error(w, "invalid sunny town token", http.StatusUnauthorized)
		return
	}
	if err := validateJoinTarget(claims, srv.world.roomID, srv.world.rooms); err != nil {
		log.Printf("sunny town auth failed: unknown room=%q map=%q", claims.RoomID, claims.MapID)
		http.Error(w, "unknown room or map", http.StatusBadRequest)
		return
	}

	conn, err := srv.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade websocket: %v", err)
		return
	}

	playerID := sunnytownauth.PlayerID(claims.CharacterID)
	client := &client{
		conn:   conn,
		send:   make(chan serverMessage, 16),
		server: srv,
		id:     playerID,
	}

	equipment, err := srv.hq.LoadStudentEquipment(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town equipment: %v", err)
		equipment = equipmentSnapshot{}
	}
	inventory, err := srv.loadInitialToolInventory(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town tool inventory: %v", err)
		inventory = inventorySnapshot{}
	}

	position, err := srv.hq.LoadStudentPosition(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		position = studentPositionResponse{}
	}
	if err := srv.refreshRoomMapObjects(r.Context(), claims.MapID); err != nil {
		log.Printf("refresh sunny town map objects map=%s: %v", claims.MapID, err)
	}
	if err := srv.LoadNPCCharacters(r.Context()); err != nil {
		log.Printf("refresh sunny town npc characters: %v", err)
	}

	srv.world.joinWithInventory(client, claims, equipment, position, inventory)
	log.Printf("player joined room=%s map=%s player=%s", srv.world.roomID, claims.MapID, playerID)

	go client.writePump()
	client.readPump()
}

func (srv *Server) loadInitialToolInventory(ctx context.Context, appUserID int64) (inventorySnapshot, error) {
	quantity, err := srv.hq.LoadStudentInventoryQuantity(ctx, appUserID, "pickaxe")
	if err != nil {
		return inventorySnapshot{}, err
	}
	inventory := inventorySnapshot{}
	if quantity > 0 {
		inventory["pickaxe"] = quantity
	}
	return inventory, nil
}

func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
