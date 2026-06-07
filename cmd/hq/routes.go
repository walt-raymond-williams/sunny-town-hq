package main

import (
	"net/http"

	petv1connect "hq/proto/hq/pet/v1/petv1connect"
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
	mux.HandleFunc("/api/internal/ai/assignment-attempts/", app.handleInternalAIAssignmentAttempt)
	mux.Handle("/api/", app.authenticated(apiMux))

	petServicePath, petServiceHandler := petv1connect.NewPetServiceHandler(&petService{app: app})
	mux.Handle(petServicePath, app.authenticated(petServiceHandler))

	mux.HandleFunc("/", staticHandler(webRoot))
	return logRequests(mux)
}
