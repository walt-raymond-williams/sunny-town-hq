package main

import (
	"context"

	hqassignments "hq/internal/hq/assignments"
)

type createAssignmentRequest = hqassignments.CreateRequest
type submitAssignmentRequest = hqassignments.SubmitRequest
type gradeAssignmentRequest = hqassignments.GradeRequest
type resetAssignmentRequest = hqassignments.ResetRequest
type assignmentAttemptResponse = hqassignments.AttemptResponse
type assignmentResponse = hqassignments.Response

func (app *app) loadAssignments(ctx context.Context, suffix string, args ...any) ([]assignmentResponse, error) {
	return hqassignments.Load(ctx, app.db, suffix, args...)
}

func currentAttempt(attempts []assignmentAttemptResponse) *assignmentAttemptResponse {
	return hqassignments.CurrentAttempt(attempts)
}

func restrictAssignmentsToStudent(assignments []assignmentResponse, studentID int64) {
	hqassignments.RestrictToStudent(assignments, studentID)
}

func attemptsForStudent(attempts []assignmentAttemptResponse, studentID int64) []assignmentAttemptResponse {
	return hqassignments.AttemptsForStudent(attempts, studentID)
}

func gradedAttempts(attempts []assignmentAttemptResponse) []assignmentAttemptResponse {
	return hqassignments.GradedAttempts(attempts)
}
