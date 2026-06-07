package main

import (
	"log"
	"net/http"
	"time"

	hqinventory "hq/internal/hq/inventory"
	"hq/internal/sunnytownauth"
)

type sunnyTownSessionResponse struct {
	RoomID       string                            `json:"room_id"`
	MapID        string                            `json:"map_id"`
	AvatarID     string                            `json:"avatar_id"`
	WebSocketURL string                            `json:"websocket_url"`
	JoinToken    string                            `json:"join_token"`
	ExpiresAt    time.Time                         `json:"expires_at"`
	Wallet       sunnyTownWalletResponse           `json:"wallet"`
	Inventory    hqinventory.StudentResponse       `json:"inventory"`
	Hotbar       hqinventory.StudentHotbarResponse `json:"hotbar"`
}

type sunnyTownWalletResponse struct {
	StarBalance int `json:"star_balance"`
}

func (app *app) handleTeacherLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"role": "teacher",
	})
}

func (app *app) handleTeacherLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "logged out",
	})
}

func (app *app) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "login required",
		})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (app *app) handleStudents(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	students, err := app.loadStudents(r.Context())
	if err != nil {
		log.Printf("load students: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "students could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, students)
}

func (app *app) handleSunnyTownSession(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.petStore().LoadProfile(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town student profile: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	position, err := app.loadSunnyTownPosition(r.Context(), user.ID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	roomID := "sunny-town-main"
	mapID := "sunny-town-v1"
	if position.Found {
		roomID = position.RoomID
		mapID = position.MapID
	}

	expiresAt := time.Now().UTC().Add(time.Minute)
	avatarID := "pet-default"
	token, err := sunnytownauth.Sign(sunnytownauth.Claims{
		AppUserID:       user.ID,
		KeycloakSubject: user.KeycloakSubject,
		DisplayName:     profile.DisplayName,
		Roles:           user.Roles,
		RoomID:          roomID,
		MapID:           mapID,
		AvatarID:        avatarID,
		ExpiresAt:       expiresAt.Unix(),
	}, app.sunnyTownJoinSecret)
	if err != nil {
		log.Printf("sign sunny town token: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	starBalance, err := hqinventory.EnsureStudentWallet(r.Context(), app.db, user.ID)
	if err != nil {
		log.Printf("load sunny town wallet: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	inventory, err := hqinventory.LoadStudent(r.Context(), app.db, user.ID)
	if err != nil {
		log.Printf("load sunny town inventory: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	hotbar, err := hqinventory.LoadStudentHotbar(r.Context(), app.db, user.ID)
	if err != nil {
		log.Printf("load sunny town hotbar: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	writeJSON(w, http.StatusOK, sunnyTownSessionResponse{
		RoomID:       roomID,
		MapID:        mapID,
		AvatarID:     avatarID,
		WebSocketURL: app.sunnyTownWebSocketURL,
		JoinToken:    token,
		ExpiresAt:    expiresAt,
		Wallet:       sunnyTownWalletResponse{StarBalance: starBalance},
		Inventory:    inventory,
		Hotbar:       hotbar,
	})
}
