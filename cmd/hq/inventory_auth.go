package main

import (
	"net/http"

	hqinventory "hq/internal/hq/inventory"
)

func inventoryRequireRole(w http.ResponseWriter, r *http.Request, role string) (hqinventory.RoleUser, bool) {
	user, ok := requireRole(w, r, role)
	if !ok {
		return hqinventory.RoleUser{}, false
	}
	return hqinventory.RoleUser{ID: user.ID}, true
}
