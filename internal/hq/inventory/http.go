package inventory

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	hqauth "hq/internal/hq/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleUser = hqauth.RoleUser
type RequireRoleFunc = hqauth.RequireRoleFunc

type HTTPHandler struct {
	store       *pgxpool.Pool
	requireRole RequireRoleFunc
}

type HTTPHandlerConfig struct {
	Store       *pgxpool.Pool
	RequireRole RequireRoleFunc
}

func NewHTTPHandler(config HTTPHandlerConfig) HTTPHandler {
	return HTTPHandler{
		store:       config.Store,
		requireRole: config.RequireRole,
	}
}

func (handler HTTPHandler) HandleStudentInventory(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inventory, err := LoadStudent(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("load student inventory: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "inventory could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, inventory)
}

func (handler HTTPHandler) HandleStudentInventorySlots(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inventory, err := LoadStudentSlots(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("load student inventory slots: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "inventory slots could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, inventory)
}

func (handler HTTPHandler) HandleStudentInventoryMove(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request InventoryMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	inventory, err := MoveStudentInventoryStack(r.Context(), handler.store, user.ID, request)
	if err != nil {
		status := http.StatusBadRequest
		if !IsInventoryMoveClientError(err) {
			status = http.StatusInternalServerError
			log.Printf("move student inventory stack: %v", err)
		}
		writeJSON(w, status, map[string]string{
			"error": InventoryMoveErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, inventory)
}

func (handler HTTPHandler) HandleStudentHotbar(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		hotbar, err := LoadStudentHotbar(r.Context(), handler.store, user.ID)
		if err != nil {
			log.Printf("load student hotbar: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "hotbar could not be loaded",
			})
			return
		}
		writeJSON(w, http.StatusOK, hotbar)
	case http.MethodPut:
		var request HotbarSlotRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid hotbar request",
			})
			return
		}
		hotbar, err := SetStudentHotbarSlot(r.Context(), handler.store, user.ID, request)
		if err != nil {
			status := http.StatusBadRequest
			if !errors.Is(err, ErrInvalidHotbarSlot) && !errors.Is(err, ErrHotbarItemNotOwned) {
				status = http.StatusInternalServerError
				log.Printf("set student hotbar: %v", err)
			}
			writeJSON(w, status, map[string]string{
				"error": HotbarErrorMessage(err),
			})
			return
		}
		writeJSON(w, http.StatusOK, hotbar)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler HTTPHandler) HandleStudentCraftingRecipes(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipes, err := LoadCraftingRecipes(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("load crafting recipes: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "crafting recipes could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, recipes)
}

func (handler HTTPHandler) HandleCraftStudentRecipe(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CraftRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := CraftStudentRecipe(r.Context(), handler.store, user.ID, request)
	if err != nil {
		log.Printf("craft student recipe: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": CraftingErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (handler HTTPHandler) HandleStudentEquipment(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	equipment, err := LoadStudentEquipment(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("load student equipment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "equipment could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (handler HTTPHandler) HandleEquipStudentItem(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request EquipmentChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	equipment, err := EquipStudentItem(r.Context(), handler.store, user.ID, request)
	if err != nil {
		log.Printf("equip student item: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": EquipmentErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (handler HTTPHandler) HandleUnequipStudentItem(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request EquipmentChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	equipment, err := UnequipStudentItem(r.Context(), handler.store, user.ID, request)
	if err != nil {
		log.Printf("unequip student item: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": EquipmentErrorMessage(err),
		})
		return
	}

	writeJSON(w, http.StatusOK, equipment)
}

func (handler HTTPHandler) HandleStudentShopPurchase(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request ShopPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	response, err := PurchaseStudentShopItem(r.Context(), handler.store, user.ID, request)
	if errors.Is(err, ErrInsufficientStars) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "not enough stars",
		})
		return
	}
	if errors.Is(err, ErrInsufficientShopStock) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "shop item is out of stock",
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

func (handler HTTPHandler) HandleStudentShopStock(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response, err := LoadShopStock(r.Context(), handler.store, r.URL.Query().Get("shop_id"))
	if err != nil {
		log.Printf("load student shop stock: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "shop stock could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func IsInventoryMoveClientError(err error) bool {
	return errors.Is(err, ErrUnsupportedInventoryMoveMode) ||
		errors.Is(err, ErrUnsupportedInventoryStorage) ||
		errors.Is(err, ErrInvalidInventorySlot) ||
		errors.Is(err, ErrInventorySourceEmpty) ||
		errors.Is(err, ErrInventoryDestinationEmpty) ||
		errors.Is(err, ErrInventoryDestinationOccupied) ||
		errors.Is(err, ErrInventoryIncompatibleMerge) ||
		errors.Is(err, ErrInventoryStackFull) ||
		errors.Is(err, ErrInvalidInventorySplitQuantity)
}

func InventoryMoveErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrUnsupportedInventoryMoveMode):
		return "unsupported inventory move"
	case errors.Is(err, ErrUnsupportedInventoryStorage):
		return "unsupported inventory storage"
	case errors.Is(err, ErrInvalidInventorySlot):
		return "invalid inventory slot"
	case errors.Is(err, ErrInventorySourceEmpty):
		return "source slot is empty"
	case errors.Is(err, ErrInventoryDestinationEmpty):
		return "destination slot is empty"
	case errors.Is(err, ErrInventoryDestinationOccupied):
		return "destination slot is occupied"
	case errors.Is(err, ErrInventoryIncompatibleMerge):
		return "stacks cannot be merged"
	case errors.Is(err, ErrInventoryStackFull):
		return "destination stack is full"
	case errors.Is(err, ErrInvalidInventorySplitQuantity):
		return "invalid split quantity"
	default:
		return "inventory move could not be completed"
	}
}

func StatusForInventoryStorageError(err error) int {
	if IsInventoryMoveClientError(err) ||
		errors.Is(err, ErrContainerNotFound) ||
		errors.Is(err, ErrInvalidContainer) ||
		errors.Is(err, ErrInventoryFull) ||
		errors.Is(err, ErrShopInputStorageFull) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func InventoryStorageErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrContainerNotFound):
		return "container not found"
	case errors.Is(err, ErrInvalidContainer):
		return "invalid container"
	case errors.Is(err, ErrInventoryFull):
		return "not enough room in inventory"
	case errors.Is(err, ErrShopInputStorageFull):
		return "shop input storage is full"
	default:
		return InventoryMoveErrorMessage(err)
	}
}

func EquipmentErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidEquipmentSlot):
		return "invalid equipment slot"
	case errors.Is(err, ErrItemNotEquippable):
		return "item cannot be equipped in that slot"
	case errors.Is(err, ErrItemNotOwned):
		return "item is not in your inventory"
	default:
		return "equipment could not be updated"
	}
}

func HotbarErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidHotbarSlot):
		return "invalid hotbar slot"
	case errors.Is(err, ErrHotbarItemNotOwned):
		return "item is not in your inventory"
	default:
		return "hotbar could not be updated"
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
