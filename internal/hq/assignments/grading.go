package assignments

import (
	"context"
	"errors"
	"strings"

	hqinventory "hq/internal/hq/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	GraderTypeTeacher = "teacher"
	GraderTypeAI      = "ai"

	GradeSourceManual          = "manual"
	GradeSourceAIAuto          = "ai_auto"
	GradeSourceTeacherOverride = "teacher_override"

	AIReviewStatusPending    = "pending_review"
	AIReviewStatusReviewed   = "reviewed"
	AIReviewStatusOverridden = "overridden"
)

var ErrAnsweredAssignmentNotFound = errors.New("answered assignment not found")

type GradeAttemptCommand struct {
	AssignmentID     int64
	AttemptID        int64
	Passed           bool
	Feedback         string
	GradedByType     string
	GradedByUserID   *int64
	GradedByService  string
	GradeSource      string
	AIGradeID        *int64
	AIReviewStatus   string
	PreserveAIReview bool
}

type GradeQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func GradeAttempt(ctx context.Context, querier GradeQuerier, command GradeAttemptCommand) error {
	command.Feedback = strings.TrimSpace(command.Feedback)
	command.GradedByType = strings.TrimSpace(command.GradedByType)
	command.GradedByService = strings.TrimSpace(command.GradedByService)
	command.GradeSource = strings.TrimSpace(command.GradeSource)
	command.AIReviewStatus = strings.TrimSpace(command.AIReviewStatus)
	if command.AssignmentID < 1 || command.AttemptID < 1 {
		return ErrAnsweredAssignmentNotFound
	}
	if command.GradedByType != GraderTypeTeacher && command.GradedByType != GraderTypeAI {
		return errors.New("invalid grader type")
	}
	if command.GradeSource == "" {
		command.GradeSource = GradeSourceManual
	}
	if command.GradedByType == GraderTypeAI && command.AIReviewStatus == "" {
		command.AIReviewStatus = AIReviewStatusPending
	}

	var studentUserID int64
	var cookieAwarded bool
	var existingAIGradeID *int64
	err := querier.QueryRow(
		ctx,
		`
			select aa.student_user_id, aa.cookie_awarded, aa.ai_grade_id
			from assignment_attempt aa
			where aa.id = $1
				and aa.assignment_id = $2
				and aa.reset_at is null
		`,
		command.AttemptID,
		command.AssignmentID,
	).Scan(&studentUserID, &cookieAwarded, &existingAIGradeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAnsweredAssignmentNotFound
	}
	if err != nil {
		return err
	}

	reviewStatus := command.AIReviewStatus
	if command.GradedByType == GraderTypeTeacher && existingAIGradeID != nil && !command.PreserveAIReview {
		reviewStatus = AIReviewStatusOverridden
		if command.GradeSource == GradeSourceManual {
			command.GradeSource = GradeSourceTeacherOverride
		}
	}

	_, err = querier.Exec(
		ctx,
		`
			update assignment_attempt
			set passed = $1,
				feedback = nullif($2, ''),
				date_graded = now(),
				cookie_awarded = case
					when $1 and not cookie_awarded then true
					else cookie_awarded
				end,
				graded_by_type = $4,
				graded_by_user_id = $5,
				graded_by_service = nullif($6, ''),
				grade_source = nullif($7, ''),
				ai_grade_id = coalesce($8, ai_grade_id),
				ai_review_status = nullif($9, '')
			where id = $3
		`,
		command.Passed,
		command.Feedback,
		command.AttemptID,
		command.GradedByType,
		command.GradedByUserID,
		command.GradedByService,
		command.GradeSource,
		command.AIGradeID,
		reviewStatus,
	)
	if err != nil {
		return err
	}

	if command.Passed && !cookieAwarded {
		if err := hqinventory.IncrementStudentItem(ctx, querier, studentUserID, hqinventory.CookieKey, 1); err != nil {
			return err
		}
	}

	return nil
}
