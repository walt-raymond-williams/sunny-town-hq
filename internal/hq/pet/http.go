package pet

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type RoleUser struct {
	ID int64
}

type RequireRoleFunc func(http.ResponseWriter, *http.Request, string) (RoleUser, bool)

type HTTPStore interface {
	LoadProfile(ctx context.Context, userID int64) (Profile, error)
	Feed(ctx context.Context, userID int64) (Profile, error)
}

type HTTPHandler struct {
	store          HTTPStore
	requireRole    RequireRoleFunc
	noCookiesError error
}

type HTTPHandlerConfig struct {
	Store          HTTPStore
	RequireRole    RequireRoleFunc
	NoCookiesError error
}

func NewHTTPHandler(config HTTPHandlerConfig) HTTPHandler {
	return HTTPHandler{
		store:          config.Store,
		requireRole:    config.RequireRole,
		noCookiesError: config.NoCookiesError,
	}
}

func (handler HTTPHandler) HandleStudentProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := handler.store.LoadProfile(r.Context(), user.ID)
	if err != nil {
		log.Printf("load student profile: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "student profile could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (handler HTTPHandler) HandleFeedStudentPet(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := handler.store.Feed(r.Context(), user.ID)
	if errors.Is(err, handler.noCookiesError) {
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
