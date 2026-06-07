package assignments

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func IsValidCategory(category string) bool {
	return category == "MATH" || category == "SCIENCE" || category == "READING"
}

func parseAssignmentID(w http.ResponseWriter, path string, suffix string) (int64, bool) {
	idText := strings.TrimPrefix(path, "/api/assignments/")
	if suffix != "" {
		idText = strings.TrimSuffix(idText, suffix)
	}
	idText = strings.Trim(idText, "/")
	if idText == "" || strings.Contains(idText, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return 0, false
	}

	return id, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
