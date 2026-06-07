package assignments

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleNextStudentAssignmentRejectsInvalidCategory(t *testing.T) {
	handler := HTTPHandler{
		requireRole: allowRoleUser(42),
	}
	request := httptest.NewRequest(http.MethodGet, "/api/student/assignments/next?category=writing", nil)
	recorder := httptest.NewRecorder()

	handler.HandleNextStudentAssignment(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "category must be MATH, SCIENCE, or READING") {
		t.Fatalf("body = %q, want category error", recorder.Body.String())
	}
}

func TestSubmitAssignmentRejectsBlankAnswer(t *testing.T) {
	handler := HTTPHandler{
		requireRole: allowRoleUser(42),
	}
	request := httptest.NewRequest(http.MethodPost, "/api/assignments/7/submit", strings.NewReader(`{"submitted_answer":"   "}`))
	recorder := httptest.NewRecorder()

	handler.submitAssignment(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "submitted_answer is required") {
		t.Fatalf("body = %q, want submitted_answer error", recorder.Body.String())
	}
}

func allowRoleUser(id int64) RequireRoleFunc {
	return func(http.ResponseWriter, *http.Request, string) (RoleUser, bool) {
		return RoleUser{ID: id}, true
	}
}
