package main

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	graderTypeTeacher = "teacher"
	graderTypeAI      = "ai"

	gradeSourceManual          = "manual"
	gradeSourceAIAuto          = "ai_auto"
	gradeSourceTeacherOverride = "teacher_override"

	aiReviewStatusPending    = "pending_review"
	aiReviewStatusOverridden = "overridden"
)

var errAnsweredAssignmentNotFound = errors.New("answered assignment not found")

type gradeAttemptCommand struct {
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

func (app *app) gradeAssignmentAttempt(ctx context.Context, command gradeAttemptCommand) error {
	command.Feedback = strings.TrimSpace(command.Feedback)
	command.GradedByType = strings.TrimSpace(command.GradedByType)
	command.GradedByService = strings.TrimSpace(command.GradedByService)
	command.GradeSource = strings.TrimSpace(command.GradeSource)
	command.AIReviewStatus = strings.TrimSpace(command.AIReviewStatus)
	if command.AssignmentID < 1 || command.AttemptID < 1 {
		return errAnsweredAssignmentNotFound
	}
	if command.GradedByType != graderTypeTeacher && command.GradedByType != graderTypeAI {
		return errors.New("invalid grader type")
	}
	if command.GradeSource == "" {
		command.GradeSource = gradeSourceManual
	}
	if command.GradedByType == graderTypeAI && command.AIReviewStatus == "" {
		command.AIReviewStatus = aiReviewStatusPending
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var studentUserID int64
	var cookieAwarded bool
	var existingAIGradeID *int64
	err = tx.QueryRow(
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
		return errAnsweredAssignmentNotFound
	}
	if err != nil {
		return err
	}

	reviewStatus := command.AIReviewStatus
	if command.GradedByType == graderTypeTeacher && existingAIGradeID != nil && !command.PreserveAIReview {
		reviewStatus = aiReviewStatusOverridden
		if command.GradeSource == gradeSourceManual {
			command.GradeSource = gradeSourceTeacherOverride
		}
	}

	_, err = tx.Exec(
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
		if err := incrementStudentInventoryItem(ctx, tx, studentUserID, cookieInventoryKey, 1); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
