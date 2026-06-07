package main

import (
	"net/http"

	hqassignments "hq/internal/hq/assignments"
)

func assignmentRequireRole(w http.ResponseWriter, r *http.Request, role string) (hqassignments.RoleUser, bool) {
	user, ok := requireRole(w, r, role)
	if !ok {
		return hqassignments.RoleUser{}, false
	}
	return hqassignments.RoleUser{ID: user.ID}, true
}
