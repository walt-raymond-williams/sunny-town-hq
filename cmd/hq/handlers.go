package main

import (
	"log"
	"net/http"
	"time"

	hqauth "hq/internal/hq/auth"
	hqcharacters "hq/internal/hq/characters"
	hqhttpapi "hq/internal/hq/httpapi"
	hqinventory "hq/internal/hq/inventory"
	hqsunnytownbridge "hq/internal/hq/sunnytownbridge"
	"hq/internal/sunnytownauth"
)

type sunnyTownSessionResponse struct {
	RoomID       string                            `json:"room_id"`
	MapID        string                            `json:"map_id"`
	CharacterID  int64                             `json:"character_id"`
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
	if _, ok := hqauth.RequireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hqhttpapi.WriteJSON(w, http.StatusOK, map[string]string{
		"role": "teacher",
	})
}

func (app *app) handleTeacherLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hqhttpapi.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "logged out",
	})
}

func (app *app) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := hqauth.UserFromContext(r.Context())
	if !ok {
		hqhttpapi.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "login required",
		})
		return
	}

	hqhttpapi.WriteJSON(w, http.StatusOK, user)
}

func (app *app) handleStudents(w http.ResponseWriter, r *http.Request) {
	if _, ok := hqauth.RequireRole(w, r, "teacher"); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	students, err := app.userStore.LoadStudents(r.Context())
	if err != nil {
		log.Printf("load students: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "students could not be loaded",
		})
		return
	}

	hqhttpapi.WriteJSON(w, http.StatusOK, students)
}

func (app *app) handleSunnyTownSession(w http.ResponseWriter, r *http.Request) {
	roleUser, ok := hqauth.RequireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := hqauth.UserFromContext(r.Context())
	if !ok {
		hqhttpapi.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "login required",
		})
		return
	}

	profile, err := app.petStore().LoadProfile(r.Context(), roleUser.ID)
	if err != nil {
		log.Printf("load sunny town student profile: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	position, err := hqsunnytownbridge.Store{DB: app.db}.LoadPosition(r.Context(), roleUser.ID)
	if err != nil {
		log.Printf("load sunny town position: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
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
	character, err := hqcharacters.EnsurePlayer(r.Context(), app.db, roleUser.ID, profile.DisplayName, avatarID)
	if err != nil {
		log.Printf("ensure sunny town player character: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	token, err := sunnytownauth.Sign(sunnytownauth.Claims{
		AppUserID:       roleUser.ID,
		CharacterID:     character.ID,
		KeycloakSubject: user.KeycloakSubject,
		DisplayName:     character.DisplayName,
		Roles:           user.Roles,
		RoomID:          roomID,
		MapID:           mapID,
		AvatarID:        character.AvatarID,
		ExpiresAt:       expiresAt.Unix(),
	}, app.sunnyTownJoinSecret)
	if err != nil {
		log.Printf("sign sunny town token: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	starBalance, err := hqinventory.EnsureStudentWallet(r.Context(), app.db, roleUser.ID)
	if err != nil {
		log.Printf("load sunny town wallet: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	inventory, err := hqinventory.LoadStudent(r.Context(), app.db, roleUser.ID)
	if err != nil {
		log.Printf("load sunny town inventory: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}
	hotbar, err := hqinventory.LoadStudentHotbar(r.Context(), app.db, roleUser.ID)
	if err != nil {
		log.Printf("load sunny town hotbar: %v", err)
		hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "sunny town session could not be created",
		})
		return
	}

	hqhttpapi.WriteJSON(w, http.StatusOK, sunnyTownSessionResponse{
		RoomID:       roomID,
		MapID:        mapID,
		CharacterID:  character.ID,
		AvatarID:     character.AvatarID,
		WebSocketURL: app.sunnyTownWebSocketURL,
		JoinToken:    token,
		ExpiresAt:    expiresAt,
		Wallet:       sunnyTownWalletResponse{StarBalance: starBalance},
		Inventory:    inventory,
		Hotbar:       hotbar,
	})
}
