package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hq/internal/aiapi"
	hqassignments "hq/internal/hq/assignments"
	"hq/internal/serviceauth"
)

func TestApplySkipReason(t *testing.T) {
	teacher := hqassignments.GraderTypeTeacher
	teacherOverride := hqassignments.GradeSourceTeacherOverride
	reviewed := hqassignments.AIReviewStatusReviewed
	overridden := hqassignments.AIReviewStatusOverridden

	tests := []struct {
		name            string
		state           AttemptGradeState
		autoApplyGrades bool
		want            string
	}{
		{
			name: "teacher override review status wins",
			state: AttemptGradeState{
				AIReviewStatus: &overridden,
			},
			autoApplyGrades: true,
			want:            ApplySkippedTeacherOverride,
		},
		{
			name: "teacher override grade source wins",
			state: AttemptGradeState{
				GradeSource: &teacherOverride,
			},
			autoApplyGrades: true,
			want:            ApplySkippedTeacherOverride,
		},
		{
			name: "reviewed attempt is preserved",
			state: AttemptGradeState{
				AIReviewStatus: &reviewed,
			},
			autoApplyGrades: true,
			want:            ApplySkippedTeacherReviewed,
		},
		{
			name: "teacher graded attempt is preserved",
			state: AttemptGradeState{
				GradedByType: &teacher,
			},
			autoApplyGrades: true,
			want:            ApplySkippedAlreadyTeacherGraded,
		},
		{
			name:            "auto apply disabled stores without applying",
			autoApplyGrades: false,
			want:            ApplySkippedAutoApplyDisabled,
		},
		{
			name:            "auto apply enabled can apply",
			autoApplyGrades: true,
			want:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplySkipReason(tt.state, tt.autoApplyGrades)
			if got != tt.want {
				t.Fatalf("ApplySkipReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequestGradeSendsServiceAuthenticatedRequest(t *testing.T) {
	var gotRequest aiapi.GradeAssignmentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/ai/grade-assignment" {
			t.Fatalf("path = %q, want /internal/ai/grade-assignment", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.Header.Get(serviceauth.AIServiceSecret) != "secret" {
			t.Fatalf("service secret header = %q, want secret", r.Header.Get(serviceauth.AIServiceSecret))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	err := RequestGrade(context.Background(), server.Client(), server.URL+"/", "secret", "grader-v2", 42)
	if err != nil {
		t.Fatalf("RequestGrade() error = %v", err)
	}
	if gotRequest.AttemptID != 42 || gotRequest.RequestID != "assignment-attempt-42:grader-v2" {
		t.Fatalf("request = %#v, want attempt 42 with prompt version request id", gotRequest)
	}
}

func TestRequestGradeReportsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	err := RequestGrade(context.Background(), server.Client(), server.URL, "secret", "grader-v2", 42)
	if err == nil {
		t.Fatal("RequestGrade() error = nil, want status error")
	}
}

func TestTriggerGradeAsyncSkipsIncompleteConfig(t *testing.T) {
	TriggerGradeAsync(TriggerConfig{Enabled: false}, 42)
	TriggerGradeAsync(TriggerConfig{Enabled: true}, 42)
	TriggerGradeAsync(TriggerConfig{Enabled: true, ServiceURL: "http://127.0.0.1:1"}, 42)
}
