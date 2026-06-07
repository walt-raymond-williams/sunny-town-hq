package assignments

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	hqauth "hq/internal/hq/auth"

	"github.com/jackc/pgx/v5"
)

type RoleUser = hqauth.RoleUser
type RequireRoleFunc = hqauth.RequireRoleFunc
type GradeAttemptFunc func(context.Context, GradeAttemptCommand) error

type HTTPHandler struct {
	store                      Store
	requireRole                RequireRoleFunc
	gradeAttempt               GradeAttemptFunc
	triggerAIGradingForAttempt func(int64)
}

type HTTPHandlerConfig struct {
	Store                      Store
	RequireRole                RequireRoleFunc
	GradeAttempt               GradeAttemptFunc
	TriggerAIGradingForAttempt func(int64)
}

func NewHTTPHandler(config HTTPHandlerConfig) HTTPHandler {
	return HTTPHandler{
		store:                      config.Store,
		requireRole:                config.RequireRole,
		gradeAttempt:               config.GradeAttempt,
		triggerAIGradingForAttempt: config.TriggerAIGradingForAttempt,
	}
}

func (handler HTTPHandler) HandleAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := handler.requireRole(w, r, "teacher"); !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		handler.listAssignments(w, r)
	case http.MethodPost:
		handler.createAssignment(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler HTTPHandler) HandleAssignmentByID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/submit") {
		handler.submitAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/grade") {
		handler.gradeAssignment(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/reset") {
		handler.resetAssignment(w, r)
		return
	}

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
