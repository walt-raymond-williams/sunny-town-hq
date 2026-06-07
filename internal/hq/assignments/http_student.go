package assignments

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (handler HTTPHandler) HandleNextStudentAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	if category != "" {
		if !IsValidCategory(category) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "category must be MATH, SCIENCE, or READING",
			})
			return
		}
	}

	assignments, err := ListNextForStudent(r.Context(), handler.store, user.ID, category)
	if err != nil {
		log.Printf("load next student assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be loaded",
		})
		return
	}

	if len(assignments) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"assignment": nil,
		})
		return
	}

	RestrictToStudent(assignments, user.ID)
	writeJSON(w, http.StatusOK, map[string]Response{
		"assignment": assignments[0],
	})
}

func (handler HTTPHandler) HandleStudentGradedAssignments(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := ListGradedForStudent(r.Context(), handler.store, user.ID)
	if err != nil {
		log.Printf("list student graded assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "graded assignments could not be loaded",
		})
		return
	}

	for index := range assignments {
		assignments[index].Attempts = AttemptsForStudent(assignments[index].Attempts, user.ID)
		assignments[index].Attempts = GradedAttempts(assignments[index].Attempts)
		assignments[index].CurrentAttempt = CurrentAttempt(assignments[index].Attempts)
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (handler HTTPHandler) submitAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "student")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/submit")
	if !ok {
		return
	}

	var request SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.SubmittedAnswer = strings.TrimSpace(request.SubmittedAnswer)
	if request.SubmittedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "submitted_answer is required",
		})
		return
	}

	assignment, attemptID, err := Submit(r.Context(), handler.store, user.ID, id, request)
	if err != nil {
		var loadErr *LoadAfterWriteError
		if errors.As(err, &loadErr) {
			log.Printf("load submitted assignment %d after attempt %d: %v", id, attemptID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "answer was saved but could not be loaded",
			})
			return
		}

		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment not found or already submitted",
			})
			return
		}

		log.Printf("submit assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answer could not be submitted",
		})
		return
	}

	if handler.triggerAIGradingForAttempt != nil {
		handler.triggerAIGradingForAttempt(attemptID)
	}

	writeJSON(w, http.StatusOK, assignment)
}
