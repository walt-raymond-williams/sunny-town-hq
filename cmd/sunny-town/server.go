package main

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

type server struct {
	config   stconfig.Config
	world    *world
	hq       *hqclient.Client
	upgrader websocket.Upgrader
}

func newServer(cfg stconfig.Config, world *world) *server {
	srv := &server{
		config: cfg,
		world:  world,
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

func (srv *server) loadInitialMapObjects(ctx context.Context) error {
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

func (srv *server) loadMapObjects(ctx context.Context, roomID string, mapID string) (map[string]*placedObject, error) {
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

func (srv *server) refreshRoomMapObjects(ctx context.Context, mapID string) error {
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

func (srv *server) placeMapObject(ctx context.Context, request placeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	placed, err := srv.hq.PlaceMapObject(ctx, request)
	if err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, placed), placed.RemainingItemAmount, nil
}

func (srv *server) removeMapObject(ctx context.Context, request removeMapObjectRequest, gameMap gameMap) (*placedObject, int, error) {
	removed, err := srv.hq.RemoveMapObject(ctx, request)
	if err != nil {
		return nil, 0, err
	}
	return placedObjectFromResponse(gameMap, removed), removed.RemainingItemAmount, nil
}

func (srv *server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
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

	playerID := sunnytownauth.PlayerID(claims.AppUserID)
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

	position, err := srv.hq.LoadStudentPosition(r.Context(), claims.AppUserID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		position = studentPositionResponse{}
	}
	if err := srv.refreshRoomMapObjects(r.Context(), claims.MapID); err != nil {
		log.Printf("refresh sunny town map objects map=%s: %v", claims.MapID, err)
	}

	srv.world.join(client, claims, equipment, position)
	log.Printf("player joined room=%s map=%s player=%s", srv.world.roomID, claims.MapID, playerID)

	go client.writePump()
	client.readPump()
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
