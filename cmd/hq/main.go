package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hq/internal/sunnytownauth"
	petv1connect "hq/proto/hq/pet/v1/petv1connect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	db                     *pgxpool.Pool
	auth                   *authVerifier
	sunnyTownJoinSecret    string
	sunnyTownServiceSecret string
	sunnyTownWebSocketURL  string
}

var (
	errInsufficientStars = errors.New("not enough stars")
	errNoCookies         = errors.New("no cookies available")
)

type createAssignmentRequest struct {
	Category       string `json:"category"`
	Prompt         string `json:"prompt"`
	ExpectedAnswer string `json:"expected_answer"`
}

type submitAssignmentRequest struct {
	SubmittedAnswer string `json:"submitted_answer"`
}

type gradeAssignmentRequest struct {
	AttemptID int64  `json:"attempt_id"`
	Passed    *bool  `json:"passed"`
	Feedback  string `json:"feedback"`
}

type resetAssignmentRequest struct {
	AttemptID int64  `json:"attempt_id"`
	Feedback  string `json:"feedback"`
}

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

type sunnyTownRewardEventRequest struct {
	EventID       string `json:"event_id"`
	AppUserID     int64  `json:"app_user_id"`
	RoomID        string `json:"room_id"`
	MapID         string `json:"map_id"`
	CollectibleID string `json:"collectible_id"`
	RewardKind    string `json:"reward_kind"`
	Amount        int    `json:"amount"`
}

type sunnyTownRewardEventResponse struct {
	Accepted       bool `json:"accepted"`
	Duplicate      bool `json:"duplicate"`
	NewStarBalance int  `json:"new_star_balance"`
}

type sunnyTownResourceEventRequest struct {
	EventID     string `json:"event_id"`
	AppUserID   int64  `json:"app_user_id"`
	Source      string `json:"source"`
	RoomID      string `json:"room_id"`
	MapID       string `json:"map_id"`
	NodeID      string `json:"node_id"`
	ResourceKey string `json:"resource_key"`
	Amount      int    `json:"amount"`
}

type sunnyTownResourceEventResponse struct {
	Accepted    bool   `json:"accepted"`
	Duplicate   bool   `json:"duplicate"`
	ResourceKey string `json:"resource_key"`
	Quantity    int    `json:"quantity"`
}

type sunnyTownPositionRequest struct {
	AppUserID int64   `json:"app_user_id"`
	RoomID    string  `json:"room_id"`
	MapID     string  `json:"map_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Facing    string  `json:"facing"`
}

type sunnyTownPositionResponse struct {
	Found     bool      `json:"found"`
	AppUserID int64     `json:"app_user_id,omitempty"`
	RoomID    string    `json:"room_id,omitempty"`
	MapID     string    `json:"map_id,omitempty"`
	X         float64   `json:"x,omitempty"`
	Y         float64   `json:"y,omitempty"`
	Facing    string    `json:"facing,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type shopPurchaseRequest struct {
	ShopID   string `json:"shopId"`
	ItemKey  string `json:"itemKey"`
	Quantity int    `json:"quantity"`
}

type shopPurchaseResponse struct {
	StarBalance int                      `json:"starBalance"`
	Inventory   studentInventoryResponse `json:"inventory"`
}

type starRewardRequest struct {
	EventID       string
	AppUserID     int64
	Source        string
	Delta         int
	RoomID        string
	MapID         string
	CollectibleID string
}

type inventoryLedgerRequest struct {
	EventID   string
	AppUserID int64
	Source    string
	ItemKey   string
	Delta     int
	RoomID    string
	MapID     string
	NodeID    string
}

type assignmentAttemptResponse struct {
	ID              int64      `json:"id"`
	AssignmentID    int64      `json:"assignment_id"`
	StudentUserID   int64      `json:"student_user_id"`
	StudentName     string     `json:"student_display_name"`
	AttemptNumber   int        `json:"attempt_number"`
	SubmittedAnswer string     `json:"submitted_answer"`
	DateSubmitted   time.Time  `json:"date_submitted"`
	Passed          *bool      `json:"passed"`
	Feedback        *string    `json:"feedback"`
	DateGraded      *time.Time `json:"date_graded"`
	CookieAwarded   bool       `json:"cookie_awarded"`
	ResetAt         *time.Time `json:"reset_at"`
}

type assignmentResponse struct {
	ID             int64                       `json:"id"`
	Category       string                      `json:"category"`
	Prompt         string                      `json:"prompt"`
	ExpectedAnswer string                      `json:"expected_answer"`
	CreatedAt      time.Time                   `json:"created_at"`
	CurrentAttempt *assignmentAttemptResponse  `json:"current_attempt"`
	Attempts       []assignmentAttemptResponse `json:"attempts"`
}

func main() {
	ctx := context.Background()
	host := envOrDefault("HQ_HOST", "0.0.0.0")
	port := envOrDefault("HQ_PORT", "8080")
	addr := host + ":" + port
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required, for example: postgres://hq:hq@localhost:55432/hq?sslmode=disable")
	}

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	keycloakIssuer := envOrDefault("KEYCLOAK_ISSUER", "http://localhost:18081/realms/hq")
	keycloakAudience := envOrDefault("KEYCLOAK_AUDIENCE", "hq-web")
	keycloakJWKSURL := os.Getenv("KEYCLOAK_JWKS_URL")
	app := &app{
		db:                     db,
		auth:                   newAuthVerifier(keycloakIssuer, keycloakAudience, keycloakJWKSURL),
		sunnyTownJoinSecret:    envOrDefault("SUNNY_TOWN_JOIN_SECRET", "local-dev-secret"),
		sunnyTownServiceSecret: envOrDefault("SUNNY_TOWN_SERVICE_SECRET", "local-dev-service-secret"),
		sunnyTownWebSocketURL:  envOrDefault("SUNNY_TOWN_WS_URL", "ws://127.0.0.1:18082/sunny-town/ws"),
	}
	if err := app.ensureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	app.startPetDecayTicker(ctx)

	webRoot := filepath.Join(".", "web")
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/me", app.handleMe)
	apiMux.HandleFunc("/api/students", app.handleStudents)
	apiMux.HandleFunc("/api/teacher/login", app.handleTeacherLogin)
	apiMux.HandleFunc("/api/teacher/logout", app.handleTeacherLogout)
	apiMux.HandleFunc("/api/student/profile", app.handleStudentProfile)
	apiMux.HandleFunc("/api/student/inventory", app.handleStudentInventory)
	apiMux.HandleFunc("/api/student/hotbar", app.handleStudentHotbar)
	apiMux.HandleFunc("/api/student/crafting/recipes", app.handleStudentCraftingRecipes)
	apiMux.HandleFunc("/api/student/crafting/craft", app.handleCraftStudentRecipe)
	apiMux.HandleFunc("/api/student/equipment", app.handleStudentEquipment)
	apiMux.HandleFunc("/api/student/equipment/equip", app.handleEquipStudentItem)
	apiMux.HandleFunc("/api/student/equipment/unequip", app.handleUnequipStudentItem)
	apiMux.HandleFunc("/api/student/shop/purchase", app.handleStudentShopPurchase)
	apiMux.HandleFunc("/api/student/pet/feed", app.handleFeedStudentPet)
	apiMux.HandleFunc("/api/student/sunny-town/session", app.handleSunnyTownSession)
	apiMux.HandleFunc("/api/student/assignments/next", app.handleNextStudentAssignment)
	apiMux.HandleFunc("/api/student/assignments/graded", app.handleStudentGradedAssignments)
	apiMux.HandleFunc("/api/assignments/answered", app.handleAnsweredAssignments)
	apiMux.HandleFunc("/api/assignments", app.handleAssignments)
	apiMux.HandleFunc("/api/assignments/", app.handleAssignmentByID)
	mux.HandleFunc("/api/internal/sunny-town/reward-events", app.handleSunnyTownRewardEvent)
	mux.HandleFunc("/api/internal/sunny-town/resource-events", app.handleSunnyTownResourceEvent)
	mux.HandleFunc("/api/internal/sunny-town/student-equipment", app.handleInternalSunnyTownStudentEquipment)
	mux.HandleFunc("/api/internal/sunny-town/inventory-quantity", app.handleInternalSunnyTownInventoryQuantity)
	mux.HandleFunc("/api/internal/sunny-town/player-position", app.handleInternalSunnyTownPlayerPosition)
	mux.HandleFunc("/api/internal/sunny-town/map-objects", app.handleInternalSunnyTownMapObjects)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/place", app.handleInternalSunnyTownPlaceMapObject)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/remove", app.handleInternalSunnyTownRemoveMapObject)
	mux.Handle("/api/", app.authenticated(apiMux))

	petServicePath, petServiceHandler := petv1connect.NewPetServiceHandler(&petService{app: app})
	mux.Handle(petServicePath, app.authenticated(petServiceHandler))

	mux.HandleFunc("/", staticHandler(webRoot))

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HQ server listening on http://%s", addr)
	for _, ip := range localIPv4Addresses() {
		log.Printf("Try from another device on Wi-Fi: http://%s:%s", ip, port)
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

func (app *app) handleAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		app.listAssignments(w, r)
	case http.MethodPost:
		app.createAssignment(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *app) handleAssignmentByID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/submit") {
		app.submitAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/grade") {
		app.gradeAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/reset") {
		app.resetAssignment(w, r)
		return
	}

	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "")
	if !ok {
		return
	}

	result, err := app.db.Exec(r.Context(), "delete from assignment where id = $1", id)
	if err != nil {
		log.Printf("delete assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be deleted",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (app *app) handleSunnyTownRewardEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	var request sunnyTownRewardEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.commitSunnyTownReward(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town reward: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "reward event could not be accepted",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleSunnyTownResourceEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	var request sunnyTownResourceEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.commitSunnyTownResource(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town resource: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "resource event could not be accepted",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleInternalSunnyTownStudentEquipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	userID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("app_user_id")), 10, 64)
	if err != nil || userID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "app_user_id is required",
		})
		return
	}

	equipment, err := app.loadStudentEquipment(r.Context(), userID)
	if err != nil {
		log.Printf("load internal sunny town student equipment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "equipment could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (app *app) handleInternalSunnyTownInventoryQuantity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	userID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("app_user_id")), 10, 64)
	if err != nil || userID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "app_user_id is required",
		})
		return
	}
	itemKey := strings.TrimSpace(r.URL.Query().Get("item_key"))
	if itemKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "item_key is required",
		})
		return
	}

	quantity, err := loadStudentInventoryQuantity(r.Context(), app.db, userID, itemKey)
	if err != nil {
		log.Printf("load internal sunny town inventory quantity: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "inventory quantity could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"item_key": itemKey,
		"quantity": quantity,
	})
}

func (app *app) handleInternalSunnyTownPlayerPosition(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		userID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("app_user_id")), 10, 64)
		if err != nil || userID < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "app_user_id is required",
			})
			return
		}
		position, err := app.loadSunnyTownPosition(r.Context(), userID)
		if err != nil {
			log.Printf("load internal sunny town position: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "position could not be loaded",
			})
			return
		}
		writeJSON(w, http.StatusOK, position)
	case http.MethodPost:
		var request sunnyTownPositionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}
		position, err := app.saveSunnyTownPosition(r.Context(), request)
		if err != nil {
			log.Printf("save internal sunny town position: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "position could not be saved",
			})
			return
		}
		writeJSON(w, http.StatusOK, position)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *app) handleInternalSunnyTownMapObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	response, err := app.loadSunnyTownMapObjects(
		r.Context(),
		r.URL.Query().Get("room_id"),
		r.URL.Query().Get("map_id"),
	)
	if err != nil {
		log.Printf("load internal sunny town map objects: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "map objects could not be loaded",
		})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleInternalSunnyTownPlaceMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	var request sunnyTownPlaceMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.placeSunnyTownMapObject(r.Context(), request)
	if err != nil {
		log.Printf("place internal sunny town map object: %v", err)
		writeJSON(w, statusForMapObjectError(err), map[string]string{
			"error": mapObjectErrorMessage(err),
		})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (app *app) handleInternalSunnyTownRemoveMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) != app.sunnyTownServiceSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	var request sunnyTownRemoveMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := app.removeSunnyTownMapObject(r.Context(), request)
	if err != nil {
		log.Printf("remove internal sunny town map object: %v", err)
		writeJSON(w, statusForMapObjectError(err), map[string]string{
			"error": mapObjectErrorMessage(err),
		})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func statusForMapObjectError(err error) int {
	switch {
	case errors.Is(err, errMapObjectNotFound):
		return http.StatusNotFound
	case errors.Is(err, errMapObjectOccupied), errors.Is(err, errMapObjectNotOwned):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func mapObjectErrorMessage(err error) string {
	switch {
	case errors.Is(err, errMapObjectOccupied):
		return "location is occupied"
	case errors.Is(err, errMapObjectNotFound):
		return "map object not found"
	case errors.Is(err, errMapObjectNotOwned):
		return "item is not in inventory"
	case errors.Is(err, errUnsupportedMapObject):
		return "unsupported map object"
	default:
		return "map object could not be updated"
	}
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

func (app *app) handleNextStudentAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	suffixArgs := []any{}
	assignmentSuffix := `
		where not exists (
			select 1
			from assignment_attempt aa
			where aa.assignment_id = a.id
				and aa.student_user_id = $1
				and aa.reset_at is null
		)
		and a.category = (
			select eligible_categories.category
			from (
				select c.category
				from assignment c
				where not exists (
					select 1
					from assignment_attempt caa
					where caa.assignment_id = c.id
						and caa.student_user_id = $1
						and caa.reset_at is null
				)
				group by c.category
			) eligible_categories
			order by random()
			limit 1
		)
		order by a.id asc
		limit 1
	`
	if category != "" {
		if !isValidCategory(category) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "category must be MATH, SCIENCE, or READING",
			})
			return
		}

		assignmentSuffix = `
			where not exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.student_user_id = $1
					and aa.reset_at is null
			)
			and a.category = $2
			order by a.id asc
			limit 1
		`
		suffixArgs = append(suffixArgs, category)
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		assignmentSuffix,
		append([]any{user.ID}, suffixArgs...)...,
	)
	if err != nil {
		log.Printf("load next student assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be loaded",
		})
		return
	}

	if len(assignments) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"assignment": nil,
		})
		return
	}

	restrictAssignmentsToStudent(assignments, user.ID)
	writeJSON(w, http.StatusOK, map[string]assignmentResponse{
		"assignment": assignments[0],
	})
}

func (app *app) handleStudentGradedAssignments(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		`
			where exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.student_user_id = $1
					and aa.passed is not null
			)
			order by a.category asc, a.id desc
		`,
		user.ID,
	)
	if err != nil {
		log.Printf("list student graded assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "graded assignments could not be loaded",
		})
		return
	}

	for index := range assignments {
		assignments[index].Attempts = attemptsForStudent(assignments[index].Attempts, user.ID)
		assignments[index].Attempts = gradedAttempts(assignments[index].Attempts)
		assignments[index].CurrentAttempt = currentAttempt(assignments[index].Attempts)
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) submitAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/submit")
	if !ok {
		return
	}

	var request submitAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.SubmittedAnswer = strings.TrimSpace(request.SubmittedAnswer)
	if request.SubmittedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "submitted_answer is required",
		})
		return
	}

	var attemptID int64
	err := app.db.QueryRow(
		r.Context(),
		`
			insert into assignment_attempt (assignment_id, student_user_id, attempt_number, submitted_answer)
			select a.id,
				$1,
				coalesce(max(aa.attempt_number), 0) + 1,
				$2
			from assignment a
			left join assignment_attempt aa on aa.assignment_id = a.id
				and aa.student_user_id = $1
			where a.id = $3
				and not exists (
					select 1
					from assignment_attempt active_attempt
					where active_attempt.assignment_id = a.id
						and active_attempt.student_user_id = $1
						and active_attempt.reset_at is null
				)
			group by a.id
			returning id
		`,
		user.ID,
		request.SubmittedAnswer,
		id,
	).Scan(&attemptID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment not found or already submitted",
			})
			return
		}

		log.Printf("submit assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answer could not be submitted",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load submitted assignment %d after attempt %d: %v", id, attemptID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answer was saved but could not be loaded",
		})
		return
	}
	assignment.Attempts = attemptsForStudent(assignment.Attempts, user.ID)
	assignment.CurrentAttempt = currentAttempt(assignment.Attempts)

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) handleAnsweredAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := app.loadAssignments(
		r.Context(),
		`
			where exists (
				select 1
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
			)
			order by (
				select case when aa.passed is null then 0 else 1 end
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			), (
				select aa.date_submitted
				from assignment_attempt aa
				where aa.assignment_id = a.id
					and aa.reset_at is null
				order by aa.attempt_number desc
				limit 1
			) desc, a.id desc
		`,
	)
	if err != nil {
		log.Printf("list answered assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answered assignments could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) gradeAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/grade")
	if !ok {
		return
	}

	var request gradeAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Feedback = strings.TrimSpace(request.Feedback)
	if request.Passed == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "passed is required",
		})
		return
	}
	if request.AttemptID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "attempt_id is required",
		})
		return
	}

	tx, err := app.db.Begin(r.Context())
	if err != nil {
		log.Printf("begin grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}
	defer func() {
		_ = tx.Rollback(r.Context())
	}()

	var attemptID int64
	var studentUserID int64
	var cookieAwarded bool
	err = tx.QueryRow(
		r.Context(),
		`
			select aa.id, aa.student_user_id, aa.cookie_awarded
			from assignment_attempt aa
			where aa.id = $1
				and aa.assignment_id = $2
				and aa.reset_at is null
		`,
		request.AttemptID,
		id,
	).Scan(&attemptID, &studentUserID, &cookieAwarded)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}
	if err != nil {
		log.Printf("load attempt for grading: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}

	awardCookie := *request.Passed && !cookieAwarded
	result, err := tx.Exec(
		r.Context(),
		`
			update assignment_attempt
			set passed = $1,
				feedback = nullif($2, ''),
				date_graded = now(),
				cookie_awarded = case
					when $1 and not cookie_awarded then true
					else cookie_awarded
				end
			where id = $3
		`,
		*request.Passed,
		request.Feedback,
		attemptID,
	)
	if err != nil {
		log.Printf("grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}

	if awardCookie {
		if err := incrementStudentInventoryItem(
			r.Context(),
			tx,
			studentUserID,
			cookieInventoryKey,
			1,
		); err != nil {
			log.Printf("award cookie: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "cookie could not be awarded",
			})
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		log.Printf("commit grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load graded assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) resetAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/reset")
	if !ok {
		return
	}

	var request resetAssignmentRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}
	}
	request.Feedback = strings.TrimSpace(request.Feedback)
	if request.AttemptID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "attempt_id is required",
		})
		return
	}

	result, err := app.db.Exec(
		r.Context(),
		`
			update assignment_attempt
			set feedback = nullif($1, ''),
				reset_at = now()
			where id = $2
				and assignment_id = $3
				and reset_at is null
		`,
		request.Feedback,
		request.AttemptID,
		id,
	)
	if err != nil {
		log.Printf("reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be reset",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment was reset but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (app *app) listAssignments(w http.ResponseWriter, r *http.Request) {
	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	studentIDText := strings.TrimSpace(r.URL.Query().Get("student_id"))

	suffix := "order by a.id desc"
	args := []any{}
	if category != "" {
		if !isValidCategory(category) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "category must be MATH, SCIENCE, or READING",
			})
			return
		}
		suffix = "where a.category = $1 order by a.id desc"
		args = append(args, category)
	}

	assignments, err := app.loadAssignments(r.Context(), suffix, args...)
	if err != nil {
		log.Printf("list assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignments could not be loaded",
		})
		return
	}

	if studentIDText != "" && studentIDText != "ALL" {
		studentID, err := strconv.ParseInt(studentIDText, 10, 64)
		if err != nil || studentID < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "student_id must be a positive integer",
			})
			return
		}
		restrictAssignmentsToStudent(assignments, studentID)
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (app *app) createAssignment(w http.ResponseWriter, r *http.Request) {
	var request createAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Category = strings.ToUpper(strings.TrimSpace(request.Category))
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.ExpectedAnswer = strings.TrimSpace(request.ExpectedAnswer)

	if !isValidCategory(request.Category) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "category must be MATH, SCIENCE, or READING",
		})
		return
	}

	if request.Prompt == "" || request.ExpectedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "prompt and expected_answer are required",
		})
		return
	}

	var id int64
	err := app.db.QueryRow(
		r.Context(),
		`
			insert into assignment (category, prompt, expected_answer)
			values ($1, $2, $3)
			returning id
		`,
		request.Category,
		request.Prompt,
		request.ExpectedAnswer,
	).Scan(&id)
	if err != nil {
		log.Printf("insert assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be saved",
		})
		return
	}

	assignment, err := app.loadAssignmentByID(r.Context(), id)
	if err != nil {
		log.Printf("load created assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusCreated, assignment)
}

func (app *app) loadAssignmentByID(ctx context.Context, id int64) (assignmentResponse, error) {
	assignments, err := app.loadAssignments(ctx, "where a.id = $1", id)
	if err != nil {
		return assignmentResponse{}, err
	}

	if len(assignments) == 0 {
		return assignmentResponse{}, pgx.ErrNoRows
	}

	return assignments[0], nil
}

func (app *app) loadStudentProfile(ctx context.Context, userID int64) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx, userID); err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfileWithoutDecay(ctx, userID)
}

func (app *app) ensureStudentWallet(ctx context.Context, userID int64) (int, error) {
	var balance int
	err := app.db.QueryRow(
		ctx,
		`
			insert into student_wallet (app_user_id)
			values ($1)
			on conflict (app_user_id) do update
			set updated_at = student_wallet.updated_at
			returning star_balance
		`,
		userID,
	).Scan(&balance)
	return balance, err
}

func (app *app) purchaseStudentShopItem(ctx context.Context, userID int64, request shopPurchaseRequest) (shopPurchaseResponse, error) {
	request.ShopID = strings.TrimSpace(request.ShopID)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if userID < 1 || request.Quantity < 1 {
		return shopPurchaseResponse{}, errors.New("shop purchase is missing required fields")
	}
	if request.ShopID != "cookie-keeper-shop" || request.ItemKey != cookieInventoryKey {
		return shopPurchaseResponse{}, errors.New("unsupported shop purchase")
	}

	totalPrice := 50 * request.Quantity
	tx, err := app.db.Begin(ctx)
	if err != nil {
		return shopPurchaseResponse{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`insert into student_wallet (app_user_id) values ($1) on conflict (app_user_id) do nothing`,
		userID,
	); err != nil {
		return shopPurchaseResponse{}, err
	}

	var starBalance int
	err = tx.QueryRow(
		ctx,
		`
			update student_wallet
			set star_balance = star_balance - $2,
				updated_at = now()
			where app_user_id = $1
				and star_balance >= $2
			returning star_balance
		`,
		userID,
		totalPrice,
	).Scan(&starBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return shopPurchaseResponse{}, errInsufficientStars
	}
	if err != nil {
		return shopPurchaseResponse{}, err
	}

	if _, err := tx.Exec(
		ctx,
		`
			insert into student_star_ledger (
				app_user_id,
				event_id,
				source,
				delta,
				metadata
			)
			values (
				$1,
				$2,
				'shop_purchase',
				$3,
				jsonb_build_object(
					'shop_id', $4::text,
					'item_key', $5::text,
					'quantity', $6::integer
				)
			)
		`,
		userID,
		fmt.Sprintf("shop-purchase:%d:%d", userID, time.Now().UnixNano()),
		-totalPrice,
		request.ShopID,
		request.ItemKey,
		request.Quantity,
	); err != nil {
		return shopPurchaseResponse{}, err
	}

	if err := incrementStudentInventoryItem(ctx, tx, userID, request.ItemKey, request.Quantity); err != nil {
		return shopPurchaseResponse{}, err
	}
	inventory, err := loadStudentInventory(ctx, tx, userID)
	if err != nil {
		return shopPurchaseResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return shopPurchaseResponse{}, err
	}

	return shopPurchaseResponse{
		StarBalance: starBalance,
		Inventory:   inventory,
	}, nil
}

func (app *app) commitSunnyTownReward(ctx context.Context, request sunnyTownRewardEventRequest) (sunnyTownRewardEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.CollectibleID = strings.TrimSpace(request.CollectibleID)
	request.RewardKind = strings.TrimSpace(request.RewardKind)
	if request.EventID == "" || request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" || request.CollectibleID == "" {
		return sunnyTownRewardEventResponse{}, errors.New("reward event is missing required fields")
	}
	if request.RewardKind != "star" || request.Amount != 1 {
		return sunnyTownRewardEventResponse{}, errors.New("unsupported sunny town reward")
	}

	inserted, balance, err := commitStudentStarReward(ctx, app.db, starRewardRequest{
		EventID:       request.EventID,
		AppUserID:     request.AppUserID,
		Source:        "sunny_town_star_collect",
		Delta:         request.Amount,
		RoomID:        request.RoomID,
		MapID:         request.MapID,
		CollectibleID: request.CollectibleID,
	})
	if err != nil {
		return sunnyTownRewardEventResponse{}, err
	}

	return sunnyTownRewardEventResponse{
		Accepted:       true,
		Duplicate:      !inserted,
		NewStarBalance: balance,
	}, nil
}

func (app *app) commitSunnyTownResource(ctx context.Context, request sunnyTownResourceEventRequest) (sunnyTownResourceEventResponse, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.NodeID = strings.TrimSpace(request.NodeID)
	request.ResourceKey = strings.TrimSpace(request.ResourceKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.RoomID == "" || request.MapID == "" || request.NodeID == "" {
		return sunnyTownResourceEventResponse{}, errors.New("resource event is missing required fields")
	}
	if request.Source != "sunny_town_mining" {
		return sunnyTownResourceEventResponse{}, errors.New("unsupported resource event source")
	}
	if request.ResourceKey != "rock" && request.ResourceKey != "crystal" {
		return sunnyTownResourceEventResponse{}, errors.New("unsupported resource")
	}
	if request.Amount < 1 {
		return sunnyTownResourceEventResponse{}, errors.New("resource amount must be positive")
	}

	inserted, quantity, err := commitStudentInventoryLedgerDelta(ctx, app.db, inventoryLedgerRequest{
		EventID:   request.EventID,
		AppUserID: request.AppUserID,
		Source:    request.Source,
		ItemKey:   request.ResourceKey,
		Delta:     request.Amount,
		RoomID:    request.RoomID,
		MapID:     request.MapID,
		NodeID:    request.NodeID,
	})
	if err != nil {
		return sunnyTownResourceEventResponse{}, err
	}

	return sunnyTownResourceEventResponse{
		Accepted:    true,
		Duplicate:   !inserted,
		ResourceKey: request.ResourceKey,
		Quantity:    quantity,
	}, nil
}

func (app *app) loadSunnyTownPosition(ctx context.Context, appUserID int64) (sunnyTownPositionResponse, error) {
	if appUserID < 1 {
		return sunnyTownPositionResponse{}, errors.New("app_user_id is required")
	}

	var position sunnyTownPositionResponse
	err := app.db.QueryRow(
		ctx,
		`
			select app_user_id, room_id, map_id, x, y, facing, updated_at
			from student_sunny_town_position
			where app_user_id = $1
		`,
		appUserID,
	).Scan(
		&position.AppUserID,
		&position.RoomID,
		&position.MapID,
		&position.X,
		&position.Y,
		&position.Facing,
		&position.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sunnyTownPositionResponse{Found: false}, nil
	}
	if err != nil {
		return sunnyTownPositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func (app *app) saveSunnyTownPosition(ctx context.Context, request sunnyTownPositionRequest) (sunnyTownPositionResponse, error) {
	request.RoomID = strings.TrimSpace(request.RoomID)
	request.MapID = strings.TrimSpace(request.MapID)
	request.Facing = strings.TrimSpace(request.Facing)
	if request.AppUserID < 1 || request.RoomID == "" || request.MapID == "" {
		return sunnyTownPositionResponse{}, errors.New("position is missing required fields")
	}
	if !isSunnyTownFacing(request.Facing) {
		return sunnyTownPositionResponse{}, errors.New("invalid sunny town facing")
	}
	if math.IsNaN(request.X) || math.IsInf(request.X, 0) || math.IsNaN(request.Y) || math.IsInf(request.Y, 0) {
		return sunnyTownPositionResponse{}, errors.New("invalid sunny town coordinates")
	}

	var position sunnyTownPositionResponse
	err := app.db.QueryRow(
		ctx,
		`
			insert into student_sunny_town_position (
				app_user_id,
				room_id,
				map_id,
				x,
				y,
				facing
			)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (app_user_id) do update
			set room_id = excluded.room_id,
				map_id = excluded.map_id,
				x = excluded.x,
				y = excluded.y,
				facing = excluded.facing,
				updated_at = now()
			returning app_user_id, room_id, map_id, x, y, facing, updated_at
		`,
		request.AppUserID,
		request.RoomID,
		request.MapID,
		request.X,
		request.Y,
		request.Facing,
	).Scan(
		&position.AppUserID,
		&position.RoomID,
		&position.MapID,
		&position.X,
		&position.Y,
		&position.Facing,
		&position.UpdatedAt,
	)
	if err != nil {
		return sunnyTownPositionResponse{}, err
	}
	position.Found = true
	return position, nil
}

func isSunnyTownFacing(value string) bool {
	return value == "up" || value == "down" || value == "left" || value == "right"
}

type starRewardQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func commitStudentStarReward(ctx context.Context, querier starRewardQuerier, request starRewardRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.Delta == 0 {
		return false, 0, errors.New("star reward is missing required fields")
	}

	var inserted bool
	var balance int
	err := querier.QueryRow(
		ctx,
		`
			with inserted as (
				insert into student_star_ledger (
					app_user_id,
					event_id,
					source,
					delta,
					room_id,
					map_id,
					collectible_id
				)
				values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''), nullif($7, ''))
				on conflict (event_id) do nothing
				returning app_user_id, delta
			),
			updated_wallet as (
				insert into student_wallet (app_user_id, star_balance)
				select app_user_id, delta from inserted
				on conflict (app_user_id) do update
				set star_balance = student_wallet.star_balance + excluded.star_balance,
					updated_at = now()
				returning star_balance
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select star_balance from updated_wallet),
					(select star_balance from student_wallet where app_user_id = $1),
					0
				) as star_balance
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.CollectibleID,
	).Scan(&inserted, &balance)
	return inserted, balance, err
}

func commitStudentInventoryLedgerDelta(ctx context.Context, querier starRewardQuerier, request inventoryLedgerRequest) (bool, int, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.Source = strings.TrimSpace(request.Source)
	request.ItemKey = strings.TrimSpace(request.ItemKey)
	if request.EventID == "" || request.AppUserID < 1 || request.Source == "" || request.ItemKey == "" || request.Delta == 0 {
		return false, 0, errors.New("inventory ledger event is missing required fields")
	}

	var inserted bool
	var quantity int
	err := querier.QueryRow(
		ctx,
		`
			with item_type as (
				select id
				from inventory_item_type
				where key = $4
			),
			inserted as (
				insert into student_inventory_ledger (
					app_user_id,
					event_id,
					source,
					item_type_id,
					delta,
					room_id,
					map_id,
					node_id
				)
				select $1, $2, $3, id, $5, nullif($6, ''), nullif($7, ''), nullif($8, '')
				from item_type
				on conflict (event_id) do nothing
				returning app_user_id, item_type_id, delta
			),
			updated_inventory as (
				insert into student_inventory_item (app_user_id, item_type_id, quantity)
				select app_user_id, item_type_id, delta from inserted
				on conflict (app_user_id, item_type_id) do update
				set quantity = student_inventory_item.quantity + excluded.quantity,
					updated_at = now()
				returning quantity
			)
			select exists(select 1 from inserted) as inserted,
				coalesce(
					(select quantity from updated_inventory),
					(
						select sii.quantity
						from student_inventory_item sii
						join item_type on item_type.id = sii.item_type_id
						where sii.app_user_id = $1
					),
					0
				) as quantity
		`,
		request.AppUserID,
		request.EventID,
		request.Source,
		request.ItemKey,
		request.Delta,
		request.RoomID,
		request.MapID,
		request.NodeID,
	).Scan(&inserted, &quantity)
	return inserted, quantity, err
}

func (app *app) loadStudentProfileWithoutDecay(ctx context.Context, userID int64) (studentProfileResponse, error) {
	var profile studentProfileResponse
	err := app.db.QueryRow(
		ctx,
		`
			select u.id,
				u.display_name,
				coalesce(cookie_inventory.quantity, 0),
				coalesce(sw.star_balance, 0),
				ps.hunger,
				ps.happiness,
				ps.energy,
				ps.sleeping,
				ps.updated_at,
				ps.last_decay_at,
				case
					when ps.sleeping then 'sleeping'
					when ps.hunger = 0 then 'hungry'
					when ps.happiness = 0 then 'sad'
					else 'idle'
				end as mood
			from app_user u
			join pet_state ps on ps.user_id = u.id
			left join student_wallet sw on sw.app_user_id = u.id
			left join inventory_item_type cookie_type on cookie_type.key = 'cookie'
			left join student_inventory_item cookie_inventory on cookie_inventory.app_user_id = u.id
				and cookie_inventory.item_type_id = cookie_type.id
			where u.id = $1
		`,
		userID,
	).Scan(
		&profile.ID,
		&profile.DisplayName,
		&profile.Cookies,
		&profile.StarBalance,
		&profile.PetState.Hunger,
		&profile.PetState.Happiness,
		&profile.PetState.Energy,
		&profile.PetState.Sleeping,
		&profile.PetState.UpdatedAt,
		&profile.PetState.LastDecayAt,
		&profile.PetState.Mood,
	)
	return profile, err
}

func (app *app) feedStudentPet(ctx context.Context, userID int64) (studentProfileResponse, error) {
	if err := app.applyPetDecay(ctx, userID); err != nil {
		return studentProfileResponse{}, err
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return studentProfileResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	consumed, err := consumeStudentInventoryItem(
		ctx,
		tx,
		userID,
		cookieInventoryKey,
		1,
	)
	if err != nil {
		return studentProfileResponse{}, err
	}
	if !consumed {
		return studentProfileResponse{}, errNoCookies
	}

	if _, err := tx.Exec(
		ctx,
		`
			update pet_state
			set hunger = least(hunger + 10, 100),
				updated_at = now()
			where user_id = $1
		`,
		userID,
	); err != nil {
		return studentProfileResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return studentProfileResponse{}, err
	}

	return app.loadStudentProfile(ctx, userID)
}

func (app *app) loadAssignments(ctx context.Context, suffix string, args ...any) ([]assignmentResponse, error) {
	query := `
		select a.id, a.category, a.prompt, a.expected_answer, a.created_at
		from assignment a
		` + suffix

	rows, err := app.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := []assignmentResponse{}
	assignmentIndexes := map[int64]int{}
	for rows.Next() {
		var assignment assignmentResponse
		if err := rows.Scan(
			&assignment.ID,
			&assignment.Category,
			&assignment.Prompt,
			&assignment.ExpectedAnswer,
			&assignment.CreatedAt,
		); err != nil {
			return nil, err
		}

		assignment.Attempts = []assignmentAttemptResponse{}
		assignmentIndexes[assignment.ID] = len(assignments)
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(assignments) == 0 {
		return assignments, nil
	}

	ids := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ID)
	}

	attemptRows, err := app.db.Query(
		ctx,
		`
			select aa.id,
				aa.assignment_id,
				aa.student_user_id,
				u.display_name,
				aa.attempt_number,
				aa.submitted_answer,
				aa.date_submitted,
				aa.passed,
				aa.feedback,
				aa.date_graded,
				aa.cookie_awarded,
				aa.reset_at
			from assignment_attempt aa
			join app_user u on u.id = aa.student_user_id
			where aa.assignment_id = any($1)
			order by aa.assignment_id asc, u.display_name asc, aa.attempt_number asc
		`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer attemptRows.Close()

	for attemptRows.Next() {
		var attempt assignmentAttemptResponse
		if err := attemptRows.Scan(
			&attempt.ID,
			&attempt.AssignmentID,
			&attempt.StudentUserID,
			&attempt.StudentName,
			&attempt.AttemptNumber,
			&attempt.SubmittedAnswer,
			&attempt.DateSubmitted,
			&attempt.Passed,
			&attempt.Feedback,
			&attempt.DateGraded,
			&attempt.CookieAwarded,
			&attempt.ResetAt,
		); err != nil {
			return nil, err
		}

		index, ok := assignmentIndexes[attempt.AssignmentID]
		if !ok {
			continue
		}
		assignments[index].Attempts = append(assignments[index].Attempts, attempt)
	}

	if err := attemptRows.Err(); err != nil {
		return nil, err
	}

	for index := range assignments {
		assignments[index].CurrentAttempt = currentAttempt(assignments[index].Attempts)
	}

	return assignments, nil
}

func currentAttempt(attempts []assignmentAttemptResponse) *assignmentAttemptResponse {
	for index := len(attempts) - 1; index >= 0; index-- {
		if attempts[index].ResetAt == nil {
			return &attempts[index]
		}
	}

	return nil
}

func restrictAssignmentsToStudent(assignments []assignmentResponse, studentID int64) {
	for index := range assignments {
		assignments[index].Attempts = attemptsForStudent(assignments[index].Attempts, studentID)
		assignments[index].CurrentAttempt = currentAttempt(assignments[index].Attempts)
	}
}

func attemptsForStudent(attempts []assignmentAttemptResponse, studentID int64) []assignmentAttemptResponse {
	filtered := []assignmentAttemptResponse{}
	for _, attempt := range attempts {
		if attempt.StudentUserID == studentID {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func gradedAttempts(attempts []assignmentAttemptResponse) []assignmentAttemptResponse {
	graded := []assignmentAttemptResponse{}
	for _, attempt := range attempts {
		if attempt.Passed != nil {
			graded = append(graded, attempt)
		}
	}

	return graded
}

func parseAssignmentID(w http.ResponseWriter, path string, suffix string) (int64, bool) {
	idText := strings.TrimPrefix(path, "/api/assignments/")
	if suffix != "" {
		idText = strings.TrimSuffix(idText, suffix)
	}
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	return id, true
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

func isValidCategory(category string) bool {
	return category == "MATH" || category == "SCIENCE" || category == "READING"
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

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
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
