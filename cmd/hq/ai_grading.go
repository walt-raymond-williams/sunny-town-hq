package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"hq/internal/aiapi"
	hqai "hq/internal/hq/ai"
)

const (
	aiGradeStatusPending   = hqai.GradeStatusPending
	aiGradeStatusCompleted = hqai.GradeStatusCompleted
	aiGradeStatusFailed    = hqai.GradeStatusFailed

	aiApplySkippedNotCompleted         = hqai.ApplySkippedNotCompleted
	aiApplySkippedAutoApplyDisabled    = hqai.ApplySkippedAutoApplyDisabled
	aiApplySkippedTeacherOverride      = hqai.ApplySkippedTeacherOverride
	aiApplySkippedTeacherReviewed      = hqai.ApplySkippedTeacherReviewed
	aiApplySkippedAlreadyTeacherGraded = hqai.ApplySkippedAlreadyTeacherGraded
)

var errAIGradeRequestAttemptMismatch = hqai.ErrGradeRequestAttemptMismatch

type aiAttemptGradeState = hqai.AttemptGradeState

func (app *app) aiGradingHandler() *hqai.Handler {
	return hqai.NewHandler(hqai.HandlerConfig{
		Store:           app.db,
		ServiceSecret:   app.aiToHQServiceSecret,
		AutoApplyGrades: app.aiAutoApplyGrades,
	})
}

func (app *app) handleInternalAIAssignmentAttempt(w http.ResponseWriter, r *http.Request) {
	app.aiGradingHandler().HandleAssignmentAttempt(w, r)
}

func (app *app) loadAIGradingContext(ctx context.Context, attemptID int64) (aiapi.GradingContextResponse, error) {
	return hqai.LoadGradingContext(ctx, app.db, attemptID)
}

func (app *app) recordAIGradeResult(ctx context.Context, attemptID int64, request aiapi.AIGradeResultRequest) (aiapi.AIGradeResultResponse, error) {
	return hqai.RecordGradeResult(ctx, app.db, attemptID, request, app.aiAutoApplyGrades)
}

func loadAIAttemptGradeState(ctx context.Context, querier assignmentGradeQuerier, attemptID int64) (aiAttemptGradeState, error) {
	return hqai.LoadAttemptGradeState(ctx, querier, attemptID)
}

func (app *app) aiApplySkipReason(state aiAttemptGradeState) string {
	return hqai.ApplySkipReason(state, app.aiAutoApplyGrades)
}

func (app *app) triggerAIGradingForAttempt(attemptID int64) {
	if !app.aiGradingEnabled || app.aiServiceURL == "" || app.hqToAIServiceSecret == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.requestAIGrade(ctx, attemptID); err != nil {
			log.Printf("request ai grade for attempt %d: %v", attemptID, err)
		}
	}()
}

func (app *app) requestAIGrade(ctx context.Context, attemptID int64) error {
	return hqai.RequestGrade(ctx, http.DefaultClient, app.aiServiceURL, app.hqToAIServiceSecret, app.aiPromptVersionGrader, attemptID)
}
