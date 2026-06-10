package progression

import (
	"encoding/json"
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
	return HTTPHandler{store: config.Store, requireRole: config.RequireRole}
}

func (handler HTTPHandler) HandleStudentProgression(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response, err := LoadStudentProgression(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("load student character progression: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "character progression could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
