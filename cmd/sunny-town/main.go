package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	stconfig "hq/internal/sunnytown/config"
	stmaps "hq/internal/sunnytown/maps"
	"hq/internal/sunnytownauth"
)

const (
	defaultRoomID           = "sunny-town-main"
	defaultMapID            = "sunny-town-v1"
	equipmentSlotGear       = "gear"
	equipmentSlotAccessory  = "accessory"
	equipmentSlotTool       = "tool"
	playerSize              = 28.0
	playerSpeed             = 150.0
	starPickupRadius        = 30.0
	resourceCommitQueueSize = 32
	resourceHitsRequired    = 3
	resourceToolCooldown    = 500 * time.Millisecond
	simulationInterval      = 50 * time.Millisecond
	snapshotInterval        = 100 * time.Millisecond
	movingStateTTL          = 250 * time.Millisecond
	starRespawnDelay        = 10 * time.Second
)

const (
	worldObjectSourceNatural = "natural"
	worldObjectSourcePlaced  = "placed"

	worldObjectKindRockNode   = "rock_node"
	worldObjectKindStoneBlock = "stone_block"
)

func main() {
	cfg := stconfig.Load()
	maps, err := stmaps.LoadMaps(cfg.MapsDir)
	if err != nil {
		log.Fatalf("load maps: %v", err)
	}
	if _, ok := maps[defaultMapID]; !ok {
		log.Fatalf("default map %q was not loaded", defaultMapID)
	}

	world := newWorld(defaultRoomID, maps)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for _, room := range world.rooms {
		go room.run(ctx)
	}

	srv := newServer(cfg, world)
	if err := srv.loadInitialMapObjects(ctx); err != nil {
		log.Printf("load initial sunny town map objects: %v", err)
	}
	go srv.runRewardWorker(ctx)
	go srv.runResourceWorker(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/sunny-town/ws", srv.handleWebSocket)

	httpServer := &http.Server{
		Addr:              cfg.Host + ":" + cfg.Port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("Sunny Town listening on http://%s:%s", cfg.Host, cfg.Port)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func (room *room) step(dt float64, now time.Time) {
	room.mu.Lock()

	room.tick++
	var rewards []rewardEvent
	for _, player := range room.players {
		if !player.lastMoveAt.IsZero() && now.Sub(player.lastMoveAt) > movingStateTTL {
			player.moving = false
		}

		rewards = append(rewards, room.collectStarsLocked(player, now)...)
	}

	room.respawnCollectiblesLocked(now)
	room.respawnResourceNodesLocked(now)
	room.mu.Unlock()

	for _, reward := range rewards {
		select {
		case room.rewardEvents <- reward:
		default:
			log.Printf("reward queue full event=%s", reward.eventID)
			reward.client.trySend(serverMessage{
				Type:          "reward_failed",
				CollectibleID: reward.collectibleID,
				Reason:        "temporary_error",
			})
		}
	}
}

func newRewardRunID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func validateJoinTarget(claims sunnytownauth.Claims, roomID string, rooms map[string]*room) error {
	if claims.RoomID != roomID {
		return errors.New("unknown room or map")
	}
	if rooms[claims.MapID] == nil {
		return errors.New("unknown room or map")
	}
	return nil
}
