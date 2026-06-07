package main

import (
	"context"

	hqassignments "hq/internal/hq/assignments"
)

const (
	graderTypeTeacher = hqassignments.GraderTypeTeacher
	graderTypeAI      = hqassignments.GraderTypeAI

	gradeSourceManual          = hqassignments.GradeSourceManual
	gradeSourceAIAuto          = hqassignments.GradeSourceAIAuto
	gradeSourceTeacherOverride = hqassignments.GradeSourceTeacherOverride

	aiReviewStatusPending    = hqassignments.AIReviewStatusPending
	aiReviewStatusReviewed   = hqassignments.AIReviewStatusReviewed
	aiReviewStatusOverridden = hqassignments.AIReviewStatusOverridden
)

var errAnsweredAssignmentNotFound = hqassignments.ErrAnsweredAssignmentNotFound

type gradeAttemptCommand = hqassignments.GradeAttemptCommand
type assignmentGradeQuerier = hqassignments.GradeQuerier

func (app *app) gradeAssignmentAttempt(ctx context.Context, command gradeAttemptCommand) error {
	tx, err := app.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := gradeAssignmentAttempt(ctx, tx, command); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func gradeAssignmentAttempt(ctx context.Context, querier assignmentGradeQuerier, command gradeAttemptCommand) error {
	return hqassignments.GradeAttempt(ctx, querier, command)
}
