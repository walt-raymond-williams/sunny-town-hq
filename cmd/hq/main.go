package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"time"

	hqapp "hq/internal/hq/app"
	hqauth "hq/internal/hq/auth"
	hqschema "hq/internal/hq/schema"
	hqusers "hq/internal/hq/users"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db                     *pgxpool.Pool
	auth                   *hqauth.Verifier
	userStore              *hqusers.Store
	sunnyTownJoinSecret    string
	sunnyTownServiceSecret string
	sunnyTownWebSocketURL  string
	aiServiceURL           string
	hqToAIServiceSecret    string
	aiToHQServiceSecret    string
	aiGradingEnabled       bool
	aiAutoApplyGrades      bool
	aiPromptVersionGrader  string
}

var (
	errNoCookies = errors.New("no cookies available")
)

func main() {
	ctx := context.Background()
	cfg := hqapp.LoadConfig()
	addr := cfg.Address()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required, for example: postgres://hq:hq@localhost:55432/hq?sslmode=disable")
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	app := &app{
		db:                     db,
		auth:                   hqauth.NewVerifier(cfg.KeycloakIssuer, cfg.KeycloakAudience, cfg.KeycloakJWKSURL),
		userStore:              hqusers.NewStore(db),
		sunnyTownJoinSecret:    cfg.SunnyTownJoinSecret,
		sunnyTownServiceSecret: cfg.SunnyTownServiceSecret,
		sunnyTownWebSocketURL:  cfg.SunnyTownWebSocketURL,
		aiServiceURL:           cfg.AIServiceURL,
		hqToAIServiceSecret:    cfg.HQToAIServiceSecret,
		aiToHQServiceSecret:    cfg.AIToHQServiceSecret,
		aiGradingEnabled:       cfg.AIGradingEnabled,
		aiAutoApplyGrades:      cfg.AIAutoApplyGrades,
		aiPromptVersionGrader:  cfg.AIPromptVersionGrader,
	}
	if err := hqschema.RunMigrations(ctx, db, cfg.MigrationsDir); err != nil {
		log.Fatalf("run schema migrations: %v", err)
	}
	app.petStore().StartDecayTicker(ctx)

	webRoot := filepath.Join(".", "web")

	server := &http.Server{
		Addr:              addr,
		Handler:           app.routes(webRoot),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HQ server listening on http://%s", addr)
	for _, ip := range localIPv4Addresses() {
		log.Printf("Try from another device on Wi-Fi: http://%s:%s", ip, cfg.Port)
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}
