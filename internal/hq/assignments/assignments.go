package assignments

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type CreateRequest struct {
	Category       string `json:"category"`
	Prompt         string `json:"prompt"`
	ExpectedAnswer string `json:"expected_answer"`
}

type SubmitRequest struct {
	SubmittedAnswer string `json:"submitted_answer"`
}

type GradeRequest struct {
	AttemptID int64  `json:"attempt_id"`
	Passed    *bool  `json:"passed"`
	Feedback  string `json:"feedback"`
}

type ResetRequest struct {
	AttemptID int64  `json:"attempt_id"`
	Feedback  string `json:"feedback"`
}

type AttemptResponse struct {
	ID              int64      `json:"id"`
	AssignmentID    int64      `json:"assignment_id"`
	StudentUserID   int64      `json:"student_user_id"`
	StudentName     string     `json:"student_display_name"`
	AttemptNumber   int        `json:"attempt_number"`
	SubmittedAnswer string     `json:"submitted_answer"`
	DateSubmitted   time.Time  `json:"date_submitted"`
	Passed          *bool      `json:"passed"`
	Feedback        *string    `json:"feedback"`
	DateGraded      *time.Time `json:"date_graded"`
	CookieAwarded   bool       `json:"cookie_awarded"`
	ResetAt         *time.Time `json:"reset_at"`
	GradedByType    *string    `json:"graded_by_type,omitempty"`
	GradedByUserID  *int64     `json:"graded_by_user_id,omitempty"`
	GradedByService *string    `json:"graded_by_service,omitempty"`
	GradeSource     *string    `json:"grade_source,omitempty"`
	AIReviewStatus  *string    `json:"ai_review_status,omitempty"`
	AIGradeID       *int64     `json:"ai_grade_id,omitempty"`
}

type Response struct {
	ID             int64             `json:"id"`
	Category       string            `json:"category"`
	Prompt         string            `json:"prompt"`
	ExpectedAnswer string            `json:"expected_answer"`
	CreatedAt      time.Time         `json:"created_at"`
	CurrentAttempt *AttemptResponse  `json:"current_attempt"`
	Attempts       []AttemptResponse `json:"attempts"`
}

type Loader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func Load(ctx context.Context, loader Loader, suffix string, args ...any) ([]Response, error) {
	query := `
		select a.id, a.category, a.prompt, a.expected_answer, a.created_at
		from assignment a
		` + suffix

	rows, err := loader.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := []Response{}
	assignmentIndexes := map[int64]int{}
	for rows.Next() {
		var assignment Response
		if err := rows.Scan(
			&assignment.ID,
			&assignment.Category,
			&assignment.Prompt,
			&assignment.ExpectedAnswer,
			&assignment.CreatedAt,
		); err != nil {
			return nil, err
		}

		assignment.Attempts = []AttemptResponse{}
		assignmentIndexes[assignment.ID] = len(assignments)
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(assignments) == 0 {
		return assignments, nil
	}

	ids := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ID)
	}

	attemptRows, err := loader.Query(
		ctx,
		`
			select aa.id,
				aa.assignment_id,
				aa.student_user_id,
				u.display_name,
				aa.attempt_number,
				aa.submitted_answer,
				aa.date_submitted,
				aa.passed,
				aa.feedback,
				aa.date_graded,
				aa.cookie_awarded,
				aa.reset_at,
				aa.graded_by_type,
				aa.graded_by_user_id,
				aa.graded_by_service,
				aa.grade_source,
				aa.ai_review_status,
				aa.ai_grade_id
			from assignment_attempt aa
			join app_user u on u.id = aa.student_user_id
			where aa.assignment_id = any($1)
			order by aa.assignment_id asc, u.display_name asc, aa.attempt_number asc
		`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer attemptRows.Close()

	for attemptRows.Next() {
		var attempt AttemptResponse
		if err := attemptRows.Scan(
			&attempt.ID,
			&attempt.AssignmentID,
			&attempt.StudentUserID,
			&attempt.StudentName,
			&attempt.AttemptNumber,
			&attempt.SubmittedAnswer,
			&attempt.DateSubmitted,
			&attempt.Passed,
			&attempt.Feedback,
			&attempt.DateGraded,
			&attempt.CookieAwarded,
			&attempt.ResetAt,
			&attempt.GradedByType,
			&attempt.GradedByUserID,
			&attempt.GradedByService,
			&attempt.GradeSource,
			&attempt.AIReviewStatus,
			&attempt.AIGradeID,
		); err != nil {
			return nil, err
		}

		index, ok := assignmentIndexes[attempt.AssignmentID]
		if !ok {
			continue
		}
		assignments[index].Attempts = append(assignments[index].Attempts, attempt)
	}

	if err := attemptRows.Err(); err != nil {
		return nil, err
	}

	for index := range assignments {
		assignments[index].CurrentAttempt = CurrentAttempt(assignments[index].Attempts)
	}

	return assignments, nil
}

func CurrentAttempt(attempts []AttemptResponse) *AttemptResponse {
	for index := len(attempts) - 1; index >= 0; index-- {
		if attempts[index].ResetAt == nil {
			return &attempts[index]
		}
	}

	return nil
}

func RestrictToStudent(assignments []Response, studentID int64) {
	for index := range assignments {
		assignments[index].Attempts = AttemptsForStudent(assignments[index].Attempts, studentID)
		assignments[index].CurrentAttempt = CurrentAttempt(assignments[index].Attempts)
	}
}

func AttemptsForStudent(attempts []AttemptResponse, studentID int64) []AttemptResponse {
	filtered := []AttemptResponse{}
	for _, attempt := range attempts {
		if attempt.StudentUserID == studentID {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func GradedAttempts(attempts []AttemptResponse) []AttemptResponse {
	graded := []AttemptResponse{}
	for _, attempt := range attempts {
		if attempt.Passed != nil {
			graded = append(graded, attempt)
		}
	}

	return graded
}
