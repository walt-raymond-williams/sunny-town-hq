package assignments

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		want     bool
	}{
		{category: "MATH", want: true},
		{category: "SCIENCE", want: true},
		{category: "READING", want: true},
		{category: "WRITING", want: false},
		{category: "math", want: false},
		{category: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := IsValidCategory(tt.category); got != tt.want {
				t.Fatalf("IsValidCategory(%q) = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestParseAssignmentID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		suffix     string
		wantID     int64
		wantOK     bool
		wantStatus int
	}{
		{
			name:   "plain assignment path",
			path:   "/api/assignments/42",
			wantID: 42,
			wantOK: true,
		},
		{
			name:   "assignment action path",
			path:   "/api/assignments/42/submit",
			suffix: "/submit",
			wantID: 42,
			wantOK: true,
		},
		{
			name:       "missing id",
			path:       "/api/assignments/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non numeric id",
			path:       "/api/assignments/not-a-number",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "nested path",
			path:       "/api/assignments/42/extra",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "zero id",
			path:       "/api/assignments/0",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			id, ok := parseAssignmentID(recorder, tt.path, tt.suffix)
			if id != tt.wantID || ok != tt.wantOK {
				t.Fatalf("parseAssignmentID() = (%d, %v), want (%d, %v)", id, ok, tt.wantID, tt.wantOK)
			}
			if tt.wantStatus != 0 && recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}
