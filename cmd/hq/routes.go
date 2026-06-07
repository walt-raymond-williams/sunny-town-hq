package main

import (
	"net/http"

	hqassignments "hq/internal/hq/assignments"
	hqpet "hq/internal/hq/pet"
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
	assignmentHandlers := hqassignments.NewHTTPHandler(hqassignments.HTTPHandlerConfig{
		Store:                      app.db,
		RequireRole:                assignmentRequireRole,
		GradeAttempt:               app.gradeAssignmentAttempt,
		TriggerAIGradingForAttempt: app.triggerAIGradingForAttempt,
	})
	apiMux.HandleFunc("/api/student/assignments/next", assignmentHandlers.HandleNextStudentAssignment)
	apiMux.HandleFunc("/api/student/assignments/graded", assignmentHandlers.HandleStudentGradedAssignments)
	apiMux.HandleFunc("/api/assignments/answered", assignmentHandlers.HandleAnsweredAssignments)
	apiMux.HandleFunc("/api/assignments", assignmentHandlers.HandleAssignments)
	apiMux.HandleFunc("/api/assignments/", assignmentHandlers.HandleAssignmentByID)
	sunnyTownBridge := app.sunnyTownBridgeHandler()
	mux.HandleFunc("/api/internal/sunny-town/reward-events", sunnyTownBridge.HandleRewardEvent)
	mux.HandleFunc("/api/internal/sunny-town/resource-events", sunnyTownBridge.HandleResourceEvent)
	mux.HandleFunc("/api/internal/sunny-town/student-equipment", sunnyTownBridge.HandleStudentEquipment)
	mux.HandleFunc("/api/internal/sunny-town/inventory-quantity", sunnyTownBridge.HandleInventoryQuantity)
	mux.HandleFunc("/api/internal/sunny-town/player-position", sunnyTownBridge.HandlePlayerPosition)
	mux.HandleFunc("/api/internal/sunny-town/map-objects", sunnyTownBridge.HandleMapObjects)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/place", sunnyTownBridge.HandlePlaceMapObject)
	mux.HandleFunc("/api/internal/sunny-town/map-objects/remove", sunnyTownBridge.HandleRemoveMapObject)
	mux.HandleFunc("/api/internal/ai/assignment-attempts/", app.handleInternalAIAssignmentAttempt)
	mux.Handle("/api/", app.authenticated(apiMux))

	petServicePath, petServiceHandler := hqpet.NewServiceHandler(hqPetBackend{app: app}, requireStudentID, errNoCookies)
	mux.Handle(petServicePath, app.authenticated(petServiceHandler))

	mux.HandleFunc("/", staticHandler(webRoot))
	return logRequests(mux)
}
