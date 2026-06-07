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
type assignmentLoadAfterWriteError = hqassignments.LoadAfterWriteError

func (app *app) loadAssignments(ctx context.Context, suffix string, args ...any) ([]assignmentResponse, error) {
	return hqassignments.Load(ctx, app.db, suffix, args...)
}

func (app *app) loadAssignmentByID(ctx context.Context, id int64) (assignmentResponse, error) {
	return hqassignments.LoadByID(ctx, app.db, id)
}

func (app *app) createAssignmentRecord(ctx context.Context, request createAssignmentRequest) (assignmentResponse, error) {
	return hqassignments.Create(ctx, app.db, request)
}

func (app *app) submitAssignmentAttempt(ctx context.Context, studentUserID int64, assignmentID int64, request submitAssignmentRequest) (assignmentResponse, int64, error) {
	return hqassignments.Submit(ctx, app.db, studentUserID, assignmentID, request)
}

func (app *app) resetAssignmentAttempt(ctx context.Context, assignmentID int64, request resetAssignmentRequest) (assignmentResponse, error) {
	return hqassignments.Reset(ctx, app.db, assignmentID, request)
}

func (app *app) deleteAssignmentRecord(ctx context.Context, assignmentID int64) (bool, error) {
	return hqassignments.Delete(ctx, app.db, assignmentID)
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
