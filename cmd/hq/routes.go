package main

import (
	"context"
	"net/http"

	hqai "hq/internal/hq/ai"
	hqassignments "hq/internal/hq/assignments"
	hqauth "hq/internal/hq/auth"
	hqhttpapi "hq/internal/hq/httpapi"
	hqinventory "hq/internal/hq/inventory"
	hqpet "hq/internal/hq/pet"
	hqprogression "hq/internal/hq/progression"
	hqsunnytownbridge "hq/internal/hq/sunnytownbridge"
)

func (app *app) routes(webRoot string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/me", app.handleMe)
	apiMux.HandleFunc("/api/students", app.handleStudents)
	apiMux.HandleFunc("/api/teacher/login", app.handleTeacherLogin)
	apiMux.HandleFunc("/api/teacher/logout", app.handleTeacherLogout)
	petStore := app.petStore()
	petHandlers := hqpet.NewHTTPHandler(hqpet.HTTPHandlerConfig{
		Store:          petStore,
		RequireRole:    hqauth.RequireRole,
		NoCookiesError: errNoCookies,
	})
	apiMux.HandleFunc("/api/student/profile", petHandlers.HandleStudentProfile)
	inventoryHandlers := hqinventory.NewHTTPHandler(hqinventory.HTTPHandlerConfig{
		Store:       app.db,
		RequireRole: hqauth.RequireRole,
	})
	apiMux.HandleFunc("/api/student/inventory", inventoryHandlers.HandleStudentInventory)
	apiMux.HandleFunc("/api/student/inventory/slots", inventoryHandlers.HandleStudentInventorySlots)
	apiMux.HandleFunc("/api/student/inventory/move", inventoryHandlers.HandleStudentInventoryMove)
	apiMux.HandleFunc("/api/student/hotbar", inventoryHandlers.HandleStudentHotbar)
	apiMux.HandleFunc("/api/student/crafting/recipes", inventoryHandlers.HandleStudentCraftingRecipes)
	apiMux.HandleFunc("/api/student/crafting/craft", inventoryHandlers.HandleCraftStudentRecipe)
	apiMux.HandleFunc("/api/student/equipment", inventoryHandlers.HandleStudentEquipment)
	apiMux.HandleFunc("/api/student/equipment/equip", inventoryHandlers.HandleEquipStudentItem)
	apiMux.HandleFunc("/api/student/equipment/unequip", inventoryHandlers.HandleUnequipStudentItem)
	apiMux.HandleFunc("/api/student/shop/purchase", inventoryHandlers.HandleStudentShopPurchase)
	apiMux.HandleFunc("/api/student/shop/stock", inventoryHandlers.HandleStudentShopStock)
	progressionHandlers := hqprogression.NewHTTPHandler(hqprogression.HTTPHandlerConfig{
		Store:       app.db,
		RequireRole: hqauth.RequireRole,
	})
	apiMux.HandleFunc("/api/student/sunny-town/progression", progressionHandlers.HandleStudentProgression)
	apiMux.HandleFunc("/api/student/pet/feed", petHandlers.HandleFeedStudentPet)
	apiMux.HandleFunc("/api/student/sunny-town/session", app.handleSunnyTownSession)
	assignmentHandlers := hqassignments.NewHTTPHandler(hqassignments.HTTPHandlerConfig{
		Store:       app.db,
		RequireRole: hqauth.RequireRole,
		GradeAttempt: func(ctx context.Context, command hqassignments.GradeAttemptCommand) error {
			return hqassignments.GradeAttemptInTx(ctx, app.db, command)
		},
		TriggerAIGradingForAttempt: func(attemptID int64) {
			hqai.TriggerGradeAsync(hqai.TriggerConfig{
				Enabled:       app.aiGradingEnabled,
				ServiceURL:    app.aiServiceURL,
				ServiceSecret: app.hqToAIServiceSecret,
				PromptVersion: app.aiPromptVersionGrader,
			}, attemptID)
		},
	})
	apiMux.HandleFunc("/api/student/assignments/next", assignmentHandlers.HandleNextStudentAssignment)
	apiMux.HandleFunc("/api/student/assignments/graded", assignmentHandlers.HandleStudentGradedAssignments)
	apiMux.HandleFunc("/api/assignments/answered", assignmentHandlers.HandleAnsweredAssignments)
	apiMux.HandleFunc("/api/assignments", assignmentHandlers.HandleAssignments)
	apiMux.HandleFunc("/api/assignments/", assignmentHandlers.HandleAssignmentByID)
	sunnyTownBridge := hqsunnytownbridge.NewHTTPHandler(
		hqsunnytownbridge.Store{DB: app.db},
		app.sunnyTownServiceSecret,
	)
	mux.HandleFunc("/api/internal/sunny-town/reward-events", sunnyTownBridge.HandleRewardEvent)
	mux.HandleFunc("/api/internal/sunny-town/resource-events", sunnyTownBridge.HandleResourceEvent)
	mux.HandleFunc("/api/internal/sunny-town/character-skill-xp", sunnyTownBridge.HandleCharacterSkillXP)
	mux.HandleFunc("/api/internal/sunny-town/npc-job-production", sunnyTownBridge.HandleNPCJobProduction)
	mux.HandleFunc("/api/internal/sunny-town/npc-job-production/progress", sunnyTownBridge.HandleNPCJobProductionProgress)
	mux.HandleFunc("/api/internal/sunny-town/student-equipment", sunnyTownBridge.HandleStudentEquipment)
	mux.HandleFunc("/api/internal/sunny-town/inventory-quantity", sunnyTownBridge.HandleInventoryQuantity)
	mux.HandleFunc("/api/internal/sunny-town/container-slots", sunnyTownBridge.HandleContainerSlots)
	mux.HandleFunc("/api/internal/sunny-town/container-transfer", sunnyTownBridge.HandleContainerTransfer)
	mux.HandleFunc("/api/internal/sunny-town/player-position", sunnyTownBridge.HandlePlayerPosition)
	mux.HandleFunc("/api/internal/sunny-town/map-objects", sunnyTownBridge.HandleMapObjects)
	mux.HandleFunc("/api/internal/sunny-town/npc-characters/ensure", sunnyTownBridge.HandleEnsureNPCCharacters)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/place", sunnyTownBridge.HandlePlaceMapObject)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/remove", sunnyTownBridge.HandleRemoveMapObject)
	aiHandler := hqai.NewHandler(hqai.HandlerConfig{
		Store:           app.db,
		ServiceSecret:   app.aiToHQServiceSecret,
		AutoApplyGrades: app.aiAutoApplyGrades,
	})
	mux.HandleFunc("/api/internal/ai/assignment-attempts/", aiHandler.HandleAssignmentAttempt)
	mux.Handle("/api/", app.authenticated(apiMux))

	petServicePath, petServiceHandler := hqpet.NewServiceHandler(petStore, hqauth.RequireStudentID, errNoCookies)
	mux.Handle(petServicePath, app.authenticated(petServiceHandler))

	mux.HandleFunc("/", hqhttpapi.StaticHandler(webRoot))
	return hqhttpapi.LogRequests(mux)
}
