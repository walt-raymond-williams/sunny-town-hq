package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	hqinventory "hq/internal/hq/inventory"
	"hq/internal/sunnytownauth"
)

type petStateResponse struct {
	Hunger      int       `json:"hunger"`
	Happiness   int       `json:"happiness"`
	Energy      int       `json:"energy"`
	Sleeping    bool      `json:"sleeping"`
	Mood        string    `json:"mood"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastDecayAt time.Time `json:"last_decay_at"`
}

type studentProfileResponse struct {
	ID          int64            `json:"id"`
	DisplayName string           `json:"display_name"`
	Cookies     int              `json:"cookies"`
	StarBalance int              `json:"star_balance"`
	PetState    petStateResponse `json:"pet_state"`
}

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

func (app *app) handleStudentProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.loadStudentProfile(r.Context(), user.ID)
	if err != nil {
		log.Printf("load student profile: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "student profile could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (app *app) handleFeedStudentPet(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.feedStudentPet(r.Context(), user.ID)
	if errors.Is(err, errNoCookies) {
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

func (app *app) handleSunnyTownSession(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, err := app.loadStudentProfile(r.Context(), user.ID)
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

func (app *app) loadStudentProfile(ctx context.Context, userID int64) (studentProfileResponse, error) {
	profile, err := app.petStore().LoadProfile(ctx, userID)
	return fromPetProfile(profile), err
}
