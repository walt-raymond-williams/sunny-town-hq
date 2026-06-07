package main

import (
	"context"
	"net/http"

	hqpet "hq/internal/hq/pet"
)

func petRequireRole(w http.ResponseWriter, r *http.Request, role string) (hqpet.RoleUser, bool) {
	user, ok := requireRole(w, r, role)
	if !ok {
		return hqpet.RoleUser{}, false
	}
	return hqpet.RoleUser{ID: user.ID}, true
}

func requireStudentID(ctx context.Context) (int64, error) {
	user, ok := userFromContext(ctx)
	if !ok || !hasRole(user, "student") {
		return 0, errInvalidToken
	}
	return user.ID, nil
}
