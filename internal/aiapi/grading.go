package aiapi

import "time"

type GradingContextResponse struct {
	AssignmentID       int64     `json:"assignment_id"`
	AttemptID          int64     `json:"attempt_id"`
	StudentUserID      int64     `json:"student_user_id"`
	StudentDisplayName string    `json:"student_display_name"`
	Category           string    `json:"category"`
	GradeLevel         string    `json:"grade_level"`
	Prompt             string    `json:"prompt"`
	ExpectedAnswer     string    `json:"expected_answer"`
	Rubric             string    `json:"rubric"`
	SubmittedAnswer    string    `json:"submitted_answer"`
	AttemptNumber      int       `json:"attempt_number"`
	SubmittedAt        time.Time `json:"submitted_at"`
}

type RubricScore struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type AIGradeResultRequest struct {
	RequestID           string        `json:"request_id"`
	Status              string        `json:"status"`
	RecommendedPassed   *bool         `json:"recommended_passed"`
	RecommendedFeedback string        `json:"recommended_feedback"`
	Confidence          *float64      `json:"confidence"`
	RubricScores        []RubricScore `json:"rubric_scores"`
	Model               string        `json:"model"`
	PromptVersion       string        `json:"prompt_version"`
	RawResponse         any           `json:"raw_response,omitempty"`
	ErrorMessage        string        `json:"error_message,omitempty"`
}

type AIGradeResultResponse struct {
	ID        int64  `json:"id"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Applied   bool   `json:"applied"`
}

type GradeAssignmentRequest struct {
	RequestID string `json:"request_id"`
	AttemptID int64  `json:"attempt_id"`
}

type GradeAssignmentResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}
