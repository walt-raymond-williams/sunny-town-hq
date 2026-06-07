package main

import (
	"log"
	"net/http"

	hqauth "hq/internal/hq/auth"
	hqhttpapi "hq/internal/hq/httpapi"
)

func (app *app) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := app.auth.AuthenticateRequest(r.Context(), r)
		if err != nil {
			hqhttpapi.WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "login required",
			})
			return
		}

		user, err = app.userStore.SyncAuthenticated(r.Context(), user)
		if err != nil {
			log.Printf("sync authenticated user: %v", err)
			hqhttpapi.WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "user could not be synced",
			})
			return
		}

		ctx := hqauth.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
