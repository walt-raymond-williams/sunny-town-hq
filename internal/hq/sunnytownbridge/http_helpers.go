package sunnytownbridge

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"hq/internal/hq/httpapi"
)

func (handler HTTPHandler) authorized(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-HQ-Service-Secret")) == handler.serviceSecret {
		return true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "service authentication required"})
	return false
}

func parsePositiveIntQuery(r *http.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get(key)), 10, 64)
	if err != nil || value < 1 {
		return 0, errors.New("positive integer is required")
	}
	return value, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	httpapi.WriteJSON(w, status, body)
}
