package assignments

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAssignmentRejectsMissingFields(t *testing.T) {
	handler := HTTPHandler{
		requireRole: allowRoleUser(42),
	}
	request := httptest.NewRequest(http.MethodPost, "/api/assignments", strings.NewReader(`{"category":"math","prompt":" ","expected_answer":"4"}`))
	recorder := httptest.NewRecorder()

	handler.HandleAssignments(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "prompt and expected_answer are required") {
		t.Fatalf("body = %q, want prompt error", recorder.Body.String())
	}
}

func TestGradeAssignmentRejectsMissingPassed(t *testing.T) {
	handler := HTTPHandler{
		requireRole: allowRoleUser(42),
	}
	request := httptest.NewRequest(http.MethodPatch, "/api/assignments/7/grade", strings.NewReader(`{"attempt_id":5}`))
	recorder := httptest.NewRecorder()

	handler.gradeAssignment(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "passed is required") {
		t.Fatalf("body = %q, want passed error", recorder.Body.String())
	}
}
