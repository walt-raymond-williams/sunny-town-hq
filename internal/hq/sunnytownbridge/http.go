package sunnytownbridge

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	hqinventory "hq/internal/hq/inventory"
)

type HTTPHandler struct {
	store         Store
	serviceSecret string
}

func NewHTTPHandler(store Store, serviceSecret string) HTTPHandler {
	return HTTPHandler{store: store, serviceSecret: serviceSecret}
}

func (handler HTTPHandler) HandleRewardEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request RewardEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.CommitReward(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town reward: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reward event could not be accepted"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleResourceEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request ResourceEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.CommitResource(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town resource: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resource event could not be accepted"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleNPCJobProduction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request NPCJobProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.CommitNPCJobProduction(r.Context(), request)
	if err != nil {
		log.Printf("commit sunny town npc job production: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "npc job production event could not be accepted"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleStudentEquipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	userID, err := parsePositiveIntQuery(r, "app_user_id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
		return
	}

	equipment, err := hqinventory.LoadStudentEquipment(r.Context(), handler.store.DB, userID)
	if err != nil {
		log.Printf("load internal sunny town student equipment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "equipment could not be loaded"})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (handler HTTPHandler) HandleInventoryQuantity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	userID, err := parsePositiveIntQuery(r, "app_user_id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
		return
	}
	itemKey := strings.TrimSpace(r.URL.Query().Get("item_key"))
	if itemKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "item_key is required"})
		return
	}

	quantity, err := LoadStudentInventoryQuantity(r.Context(), handler.store.DB, userID, itemKey)
	if err != nil {
		log.Printf("load internal sunny town inventory quantity: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inventory quantity could not be loaded"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"item_key": itemKey, "quantity": quantity})
}

func (handler HTTPHandler) HandlePlayerPosition(w http.ResponseWriter, r *http.Request) {
	if !handler.authorized(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		userID, err := parsePositiveIntQuery(r, "app_user_id")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "app_user_id is required"})
			return
		}
		position, err := handler.store.LoadPosition(r.Context(), userID)
		if err != nil {
			log.Printf("load internal sunny town position: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "position could not be loaded"})
			return
		}
		writeJSON(w, http.StatusOK, position)
	case http.MethodPost:
		var request PositionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
			return
		}
		position, err := handler.store.SavePosition(r.Context(), request)
		if err != nil {
			log.Printf("save internal sunny town position: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "position could not be saved"})
			return
		}
		writeJSON(w, http.StatusOK, position)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler HTTPHandler) HandleMapObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	response, err := handler.store.LoadMapObjects(r.Context(), r.URL.Query().Get("room_id"), r.URL.Query().Get("map_id"))
	if err != nil {
		log.Printf("load internal sunny town map objects: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "map objects could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleEnsureNPCCharacters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request EnsureNPCCharactersRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.EnsureNPCCharacters(r.Context(), request)
	if err != nil {
		log.Printf("ensure internal sunny town npc characters: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "npc characters could not be ensured"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandlePlaceMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request PlaceMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.PlaceMapObject(r.Context(), request)
	if err != nil {
		log.Printf("place internal sunny town map object: %v", err)
		writeJSON(w, StatusForMapObjectError(err), map[string]string{"error": MapObjectErrorMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleRemoveMapObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !handler.authorized(w, r) {
		return
	}

	var request RemoveMapObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}

	response, err := handler.store.RemoveMapObject(r.Context(), request)
	if err != nil {
		log.Printf("remove internal sunny town map object: %v", err)
		writeJSON(w, StatusForMapObjectError(err), map[string]string{"error": MapObjectErrorMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, response)
}
