package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	hqapp "hq/internal/hq/app"
	"hq/internal/sunnytownauth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db                     *pgxpool.Pool
	auth                   *authVerifier
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

type petStateResponse struct {
	Hunger      int       `json:"hunger"`
	Happiness   int       `json:"happiness"`
	Energy      int       `json:"energy"`
	Sleeping    bool      `json:"sleeping"`
	Mood        string    `json:"mood"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastDecayAt time.Time `json:"last_decay_at"`
}

type studentProfileResponse struct {
	ID          int64            `json:"id"`
	DisplayName string           `json:"display_name"`
	Cookies     int              `json:"cookies"`
	StarBalance int              `json:"star_balance"`
	PetState    petStateResponse `json:"pet_state"`
}

type sunnyTownSessionResponse struct {
	RoomID       string                   `json:"room_id"`
	MapID        string                   `json:"map_id"`
	AvatarID     string                   `json:"avatar_id"`
	WebSocketURL string                   `json:"websocket_url"`
	JoinToken    string                   `json:"join_token"`
	ExpiresAt    time.Time                `json:"expires_at"`
	Wallet       sunnyTownWalletResponse  `json:"wallet"`
	Inventory    studentInventoryResponse `json:"inventory"`
	Hotbar       studentHotbarResponse    `json:"hotbar"`
}

type sunnyTownWalletResponse struct {
	StarBalance int `json:"star_balance"`
}

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
		auth:                   newAuthVerifier(cfg.KeycloakIssuer, cfg.KeycloakAudience, cfg.KeycloakJWKSURL),
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
	if err := app.ensureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	app.startPetDecayTicker(ctx)

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

func (app *app) handleTeacherLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"role": "teacher",
	})
}

func (app *app) handleTeacherLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "logged out",
	})
}

func (app *app) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "login required",
		})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (app *app) handleStudents(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	students, err := app.loadStudents(r.Context())
	if err != nil {
		log.Printf("load students: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "students could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, students)
}

func (app *app) handleStudentProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.loadStudentProfile(r.Context(), user.ID)
	if err != nil {
		log.Printf("load student profile: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "student profile could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (app *app) handleStudentInventory(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inventory, err := app.loadStudentInventory(r.Context(), user.ID)
	if err != nil {
		log.Printf("load student inventory: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "inventory could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, inventory)
}

func (app *app) handleStudentHotbar(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		hotbar, err := app.loadStudentHotbar(r.Context(), user.ID)
		if err != nil {
			log.Printf("load student hotbar: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "hotbar could not be loaded",
			})
			return
		}
		writeJSON(w, http.StatusOK, hotbar)
	case http.MethodPut:
		var request hotbarSlotRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid hotbar request",
			})
			return
		}
		hotbar, err := app.setStudentHotbarSlot(r.Context(), user.ID, request)
		if err != nil {
			status := http.StatusBadRequest
			if !errors.Is(err, errInvalidHotbarSlot) && !errors.Is(err, errHotbarItemNotOwned) {
				status = http.StatusInternalServerError
				log.Printf("set student hotbar: %v", err)
			}
			writeJSON(w, status, map[string]string{
				"error": hotbarErrorMessage(err),
			})
			return
		}
		writeJSON(w, http.StatusOK, hotbar)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *app) handleStudentCraftingRecipes(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipes, err := app.loadCraftingRecipes(r.Context(), user.ID)
	if err != nil {
		log.Printf("load crafting recipes: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "crafting recipes could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, recipes)
}

func (app *app) handleCraftStudentRecipe(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request craftRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.craftStudentRecipe(r.Context(), user.ID, request)
	if err != nil {
		log.Printf("craft student recipe: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": craftingErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleStudentEquipment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	equipment, err := app.loadStudentEquipment(r.Context(), user.ID)
	if err != nil {
		log.Printf("load student equipment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "equipment could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (app *app) handleEquipStudentItem(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request equipmentChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	equipment, err := app.equipStudentItem(r.Context(), user.ID, request)
	if err != nil {
		log.Printf("equip student item: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": equipmentErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (app *app) handleUnequipStudentItem(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request equipmentChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	equipment, err := app.unequipStudentItem(r.Context(), user.ID, request)
	if err != nil {
		log.Printf("unequip student item: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": equipmentErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (app *app) handleStudentShopPurchase(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request shopPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.purchaseStudentShopItem(r.Context(), user.ID, request)
	if errors.Is(err, errInsufficientStars) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "not enough stars",
		})
		return
	}
	if err != nil {
		log.Printf("purchase student shop item: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "shop purchase could not be completed",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleFeedStudentPet(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.feedStudentPet(r.Context(), user.ID)
	if errors.Is(err, errNoCookies) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "no cookies available",
		})
		return
	}
	if err != nil {
		log.Printf("feed student pet: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "pet could not be fed",
		})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (app *app) handleSunnyTownSession(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.loadStudentProfile(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town student profile: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	position, err := app.loadSunnyTownPosition(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	roomID := "sunny-town-main"
	mapID := "sunny-town-v1"
	if position.Found {
		roomID = position.RoomID
		mapID = position.MapID
	}

	expiresAt := time.Now().UTC().Add(time.Minute)
	avatarID := "pet-default"
	token, err := sunnytownauth.Sign(sunnytownauth.Claims{
		AppUserID:       user.ID,
		KeycloakSubject: user.KeycloakSubject,
		DisplayName:     profile.DisplayName,
		Roles:           user.Roles,
		RoomID:          roomID,
		MapID:           mapID,
		AvatarID:        avatarID,
		ExpiresAt:       expiresAt.Unix(),
	}, app.sunnyTownJoinSecret)
	if err != nil {
		log.Printf("sign sunny town token: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	starBalance, err := app.ensureStudentWallet(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town wallet: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	inventory, err := app.loadStudentInventory(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town inventory: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	hotbar, err := app.loadStudentHotbar(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town hotbar: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	writeJSON(w, http.StatusOK, sunnyTownSessionResponse{
		RoomID:       roomID,
		MapID:        mapID,
		AvatarID:     avatarID,
		WebSocketURL: app.sunnyTownWebSocketURL,
		JoinToken:    token,
		ExpiresAt:    expiresAt,
		Wallet:       sunnyTownWalletResponse{StarBalance: starBalance},
		Inventory:    inventory,
		Hotbar:       hotbar,
	})
}

func equipmentErrorMessage(err error) string {
	switch {
	case errors.Is(err, errInvalidEquipmentSlot):
		return "invalid equipment slot"
	case errors.Is(err, errItemNotEquippable):
		return "item cannot be equipped in that slot"
	case errors.Is(err, errItemNotOwned):
		return "item is not in your inventory"
	default:
		return "equipment could not be updated"
	}
}

func hotbarErrorMessage(err error) string {
	switch {
	case errors.Is(err, errInvalidHotbarSlot):
		return "invalid hotbar slot"
	case errors.Is(err, errHotbarItemNotOwned):
		return "item is not in your inventory"
	default:
		return "hotbar could not be updated"
	}
}

func (app *app) loadStudentProfile(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().LoadProfile(ctx, userID)
	return fromPetProfile(profile), err
}

func (app *app) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := app.auth.authenticateRequest(r.Context(), r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "login required",
			})
			return
		}

		user, err = app.syncAuthenticatedUser(r.Context(), user)
		if err != nil {
			log.Printf("sync authenticated user: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "user could not be synced",
			})
			return
		}

		ctx := context.WithValue(r.Context(), authUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireRole(w http.ResponseWriter, r *http.Request, role string) (authUser, bool) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "login required",
		})
		return authUser{}, false
	}

	if !hasRole(user, role) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": role + " role required",
		})
		return authUser{}, false
	}

	return user, true
}

func staticHandler(webRoot string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(webRoot))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := filepath.Clean(r.URL.Path)
		if path == "." || path == string(filepath.Separator) {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		fullPath := filepath.Join(webRoot, strings.TrimPrefix(path, string(filepath.Separator)))
		if _, err := os.Stat(fullPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(webRoot, "index.html"))
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func localIPv4Addresses() []string {
	var addresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		ifaceAddresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, ifaceAddress := range ifaceAddresses {
			ipNet, ok := ifaceAddress.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			addresses = append(addresses, fmt.Sprintf("%s", ip))
		}
	}

	return addresses
}
