package inventory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPHandlerRequiresStudentRole(t *testing.T) {
	handler := NewHTTPHandler(HTTPHandlerConfig{
		RequireRole: func(w http.ResponseWriter, _ *http.Request, role string) (RoleUser, bool) {
			if role != "student" {
				t.Fatalf("role = %q, want student", role)
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "student role required"})
			return RoleUser{}, false
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/student/inventory", nil)
	response := httptest.NewRecorder()

	handler.HandleStudentInventory(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Body.String(), "student role required") {
		t.Fatalf("body = %q, want role error", response.Body.String())
	}
}

func TestHTTPHandlerRejectsWrongMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		handle func(HTTPHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "inventory only allows get",
			method: http.MethodPost,
			path:   "/api/student/inventory",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentInventory(w, r)
			},
		},
		{
			name:   "crafting recipes only allows get",
			method: http.MethodPost,
			path:   "/api/student/crafting/recipes",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentCraftingRecipes(w, r)
			},
		},
		{
			name:   "craft recipe only allows post",
			method: http.MethodGet,
			path:   "/api/student/crafting/craft",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleCraftStudentRecipe(w, r)
			},
		},
		{
			name:   "equipment only allows get",
			method: http.MethodPost,
			path:   "/api/student/equipment",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentEquipment(w, r)
			},
		},
		{
			name:   "equip only allows post",
			method: http.MethodGet,
			path:   "/api/student/equipment/equip",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleEquipStudentItem(w, r)
			},
		},
		{
			name:   "unequip only allows post",
			method: http.MethodGet,
			path:   "/api/student/equipment/unequip",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleUnequipStudentItem(w, r)
			},
		},
		{
			name:   "shop purchase only allows post",
			method: http.MethodGet,
			path:   "/api/student/shop/purchase",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentShopPurchase(w, r)
			},
		},
		{
			name:   "hotbar rejects delete",
			method: http.MethodDelete,
			path:   "/api/student/hotbar",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentHotbar(w, r)
			},
		},
	}

	handler := handlerWithStudentRole()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()

			test.handle(handler, response, request)

			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHTTPHandlerRejectsBadJSON(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		handle    func(HTTPHandler, http.ResponseWriter, *http.Request)
		wantError string
	}{
		{
			name: "hotbar",
			path: "/api/student/hotbar",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentHotbar(w, r)
			},
			wantError: "invalid hotbar request",
		},
		{
			name: "craft recipe",
			path: "/api/student/crafting/craft",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleCraftStudentRecipe(w, r)
			},
			wantError: "request body must be valid JSON",
		},
		{
			name: "equip",
			path: "/api/student/equipment/equip",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleEquipStudentItem(w, r)
			},
			wantError: "request body must be valid JSON",
		},
		{
			name: "unequip",
			path: "/api/student/equipment/unequip",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleUnequipStudentItem(w, r)
			},
			wantError: "request body must be valid JSON",
		},
		{
			name: "shop purchase",
			path: "/api/student/shop/purchase",
			handle: func(handler HTTPHandler, w http.ResponseWriter, r *http.Request) {
				handler.HandleStudentShopPurchase(w, r)
			},
			wantError: "request body must be valid JSON",
		},
	}

	handler := handlerWithStudentRole()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader("{"))
			if test.path == "/api/student/hotbar" {
				request.Method = http.MethodPut
			}
			response := httptest.NewRecorder()

			test.handle(handler, response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if !strings.Contains(response.Body.String(), test.wantError) {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantError)
			}
		})
	}
}

func TestInventoryHTTPErrorMessages(t *testing.T) {
	if got := EquipmentErrorMessage(ErrInvalidEquipmentSlot); got != "invalid equipment slot" {
		t.Fatalf("EquipmentErrorMessage invalid slot = %q", got)
	}
	if got := EquipmentErrorMessage(ErrItemNotEquippable); got != "item cannot be equipped in that slot" {
		t.Fatalf("EquipmentErrorMessage not equippable = %q", got)
	}
	if got := EquipmentErrorMessage(ErrItemNotOwned); got != "item is not in your inventory" {
		t.Fatalf("EquipmentErrorMessage not owned = %q", got)
	}
	if got := HotbarErrorMessage(ErrInvalidHotbarSlot); got != "invalid hotbar slot" {
		t.Fatalf("HotbarErrorMessage invalid slot = %q", got)
	}
	if got := HotbarErrorMessage(ErrHotbarItemNotOwned); got != "item is not in your inventory" {
		t.Fatalf("HotbarErrorMessage not owned = %q", got)
	}
}

func handlerWithStudentRole() HTTPHandler {
	return NewHTTPHandler(HTTPHandlerConfig{
		RequireRole: func(_ http.ResponseWriter, _ *http.Request, role string) (RoleUser, bool) {
			if role != "student" {
				return RoleUser{}, false
			}
			return RoleUser{ID: 123}, true
		},
	})
}
