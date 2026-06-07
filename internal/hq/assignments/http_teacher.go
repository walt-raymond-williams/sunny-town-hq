package assignments

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (handler HTTPHandler) HandleAnsweredAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := handler.requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	assignments, err := ListAnswered(r.Context(), handler.store)
	if err != nil {
		log.Printf("list answered assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "answered assignments could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (handler HTTPHandler) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := handler.requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "")
	if !ok {
		return
	}

	deleted, err := Delete(r.Context(), handler.store, id)
	if err != nil {
		log.Printf("delete assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be deleted",
		})
		return
	}

	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment not found",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler HTTPHandler) gradeAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := handler.requireRole(w, r, "teacher")
	if !ok {
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/grade")
	if !ok {
		return
	}

	var request GradeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Feedback = strings.TrimSpace(request.Feedback)
	if request.Passed == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "passed is required",
		})
		return
	}
	if request.AttemptID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "attempt_id is required",
		})
		return
	}

	err := handler.gradeAttempt(r.Context(), GradeAttemptCommand{
		AssignmentID:     id,
		AttemptID:        request.AttemptID,
		Passed:           *request.Passed,
		Feedback:         request.Feedback,
		GradedByType:     GraderTypeTeacher,
		GradedByUserID:   &user.ID,
		GradeSource:      GradeSourceManual,
		PreserveAIReview: false,
	})
	if errors.Is(err, ErrAnsweredAssignmentNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}
	if err != nil {
		log.Printf("grade assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result could not be saved",
		})
		return
	}

	assignment, err := LoadByID(r.Context(), handler.store, id)
	if err != nil {
		log.Printf("load graded assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "result was saved but could not be loaded",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (handler HTTPHandler) resetAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := handler.requireRole(w, r, "teacher"); !ok {
		return
	}

	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseAssignmentID(w, r.URL.Path, "/reset")
	if !ok {
		return
	}

	var request ResetRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}
	}
	request.Feedback = strings.TrimSpace(request.Feedback)
	if request.AttemptID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "attempt_id is required",
		})
		return
	}

	assignment, err := Reset(r.Context(), handler.store, id, request)
	if errors.Is(err, ErrAnsweredAssignmentNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "answered assignment not found",
		})
		return
	}
	if err != nil {
		var loadErr *LoadAfterWriteError
		if errors.As(err, &loadErr) {
			log.Printf("load reset assignment: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "assignment was reset but could not be loaded",
			})
			return
		}

		log.Printf("reset assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be reset",
		})
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func (handler HTTPHandler) listAssignments(w http.ResponseWriter, r *http.Request) {
	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	studentIDText := strings.TrimSpace(r.URL.Query().Get("student_id"))

	var assignments []Response
	var err error
	if category != "" {
		if !IsValidCategory(category) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "category must be MATH, SCIENCE, or READING",
			})
			return
		}
		assignments, err = ListByCategory(r.Context(), handler.store, category)
	} else {
		assignments, err = ListAll(r.Context(), handler.store)
	}

	if err != nil {
		log.Printf("list assignments: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignments could not be loaded",
		})
		return
	}

	if studentIDText != "" && studentIDText != "ALL" {
		studentID, err := strconv.ParseInt(studentIDText, 10, 64)
		if err != nil || studentID < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "student_id must be a positive integer",
			})
			return
		}
		RestrictToStudent(assignments, studentID)
	}

	writeJSON(w, http.StatusOK, assignments)
}

func (handler HTTPHandler) createAssignment(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must be valid JSON",
		})
		return
	}

	request.Category = strings.ToUpper(strings.TrimSpace(request.Category))
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.ExpectedAnswer = strings.TrimSpace(request.ExpectedAnswer)

	if !IsValidCategory(request.Category) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "category must be MATH, SCIENCE, or READING",
		})
		return
	}

	if request.Prompt == "" || request.ExpectedAnswer == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "prompt and expected_answer are required",
		})
		return
	}

	assignment, err := Create(r.Context(), handler.store, request)
	if err != nil {
		var loadErr *LoadAfterWriteError
		if errors.As(err, &loadErr) {
			log.Printf("load created assignment: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "assignment was saved but could not be loaded",
			})
			return
		}

		log.Printf("insert assignment: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "assignment could not be saved",
		})
		return
	}

	writeJSON(w, http.StatusCreated, assignment)
}
