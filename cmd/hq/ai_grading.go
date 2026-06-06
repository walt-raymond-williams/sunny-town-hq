package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hq/internal/aiapi"
	"hq/internal/serviceauth"

	"github.com/jackc/pgx/v5"
)

const (
	aiGradeStatusPending   = "pending"
	aiGradeStatusCompleted = "completed"
	aiGradeStatusFailed    = "failed"

	aiApplySkippedNotCompleted         = "not_completed"
	aiApplySkippedAutoApplyDisabled    = "auto_apply_disabled"
	aiApplySkippedTeacherOverride      = "teacher_override"
	aiApplySkippedTeacherReviewed      = "teacher_reviewed"
	aiApplySkippedAlreadyTeacherGraded = "already_teacher_graded"
)

var errAIGradeRequestAttemptMismatch = errors.New("ai grade request id belongs to a different assignment attempt")

type aiAttemptGradeState struct {
	Passed         *bool
	GradedByType   *string
	GradeSource    *string
	AIReviewStatus *string
}

func (app *app) handleInternalAIAssignmentAttempt(w http.ResponseWriter, r *http.Request) {
	if !serviceauth.Authorized(r.Header, serviceauth.HQServiceSecret, app.aiToHQServiceSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	attemptID, action, ok := parseInternalAIAssignmentAttemptPath(w, r.URL.Path)
	if !ok {
		return
	}

	switch action {
	case "grading-context":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		contextResponse, err := app.loadAIGradingContext(r.Context(), attemptID)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment attempt not found",
			})
			return
		}
		if err != nil {
			log.Printf("load ai grading context: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "grading context could not be loaded",
			})
			return
		}
		writeJSON(w, http.StatusOK, contextResponse)
	case "ai-grade":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request aiapi.AIGradeResultRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "request body must be valid JSON",
			})
			return
		}
		response, err := app.recordAIGradeResult(r.Context(), attemptID, request)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errAnsweredAssignmentNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "assignment attempt not found",
			})
			return
		}
		if err != nil {
			log.Printf("record ai grade result: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "ai grade result could not be recorded",
			})
			return
		}
		writeJSON(w, http.StatusOK, response)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment attempt action not found",
		})
	}
}

func parseInternalAIAssignmentAttemptPath(w http.ResponseWriter, path string) (int64, string, bool) {
	rest := strings.TrimPrefix(path, "/api/internal/ai/assignment-attempts/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment attempt action not found",
		})
		return 0, "", false
	}
	attemptID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || attemptID < 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "assignment attempt not found",
		})
		return 0, "", false
	}
	return attemptID, parts[1], true
}

func (app *app) loadAIGradingContext(ctx context.Context, attemptID int64) (aiapi.GradingContextResponse, error) {
	var response aiapi.GradingContextResponse
	err := app.db.QueryRow(
		ctx,
		`
			select a.id,
				aa.id,
				aa.student_user_id,
				u.display_name,
				a.category,
				'second-grade' as grade_level,
				a.prompt,
				a.expected_answer,
				'' as rubric,
				aa.submitted_answer,
				aa.attempt_number,
				aa.date_submitted
			from assignment_attempt aa
			join assignment a on a.id = aa.assignment_id
			join app_user u on u.id = aa.student_user_id
			where aa.id = $1
				and aa.reset_at is null
		`,
		attemptID,
	).Scan(
		&response.AssignmentID,
		&response.AttemptID,
		&response.StudentUserID,
		&response.StudentDisplayName,
		&response.Category,
		&response.GradeLevel,
		&response.Prompt,
		&response.ExpectedAnswer,
		&response.Rubric,
		&response.SubmittedAnswer,
		&response.AttemptNumber,
		&response.SubmittedAt,
	)
	return response, err
}

func (app *app) recordAIGradeResult(ctx context.Context, attemptID int64, request aiapi.AIGradeResultRequest) (aiapi.AIGradeResultResponse, error) {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.Status = strings.TrimSpace(request.Status)
	request.RecommendedFeedback = strings.TrimSpace(request.RecommendedFeedback)
	request.Model = strings.TrimSpace(request.Model)
	request.PromptVersion = strings.TrimSpace(request.PromptVersion)
	request.ErrorMessage = strings.TrimSpace(request.ErrorMessage)
	if request.RequestID == "" || request.PromptVersion == "" {
		return aiapi.AIGradeResultResponse{}, errors.New("ai grade result is missing required fields")
	}
	if request.Status != aiGradeStatusPending && request.Status != aiGradeStatusCompleted && request.Status != aiGradeStatusFailed {
		return aiapi.AIGradeResultResponse{}, errors.New("invalid ai grade status")
	}
	if request.Status == aiGradeStatusCompleted && request.RecommendedPassed == nil {
		return aiapi.AIGradeResultResponse{}, errors.New("completed ai grade is missing recommended_passed")
	}

	contextResponse, err := app.loadAIGradingContext(ctx, attemptID)
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}

	rubricScores, err := json.Marshal(request.RubricScores)
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}
	rawResponse, err := json.Marshal(request.RawResponse)
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}
	if request.RawResponse == nil {
		rawResponse = nil
	}

	tx, err := app.db.Begin(ctx)
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var id int64
	err = tx.QueryRow(
		ctx,
		`
			insert into assignment_ai_grade (
				assignment_attempt_id,
				request_id,
				status,
				recommended_passed,
				recommended_feedback,
				confidence,
				rubric_scores,
				model,
				prompt_version,
				raw_response,
				error_message,
				completed_at
			)
			values (
				$1,
				$2,
				$3,
				$4,
				nullif($5, ''),
				$6,
				$7::jsonb,
				nullif($8, ''),
				$9,
				$10::jsonb,
				nullif($11, ''),
				case when $3 in ('completed', 'failed') then now() else null end
			)
			on conflict (request_id) do update
			set status = excluded.status,
				recommended_passed = excluded.recommended_passed,
				recommended_feedback = excluded.recommended_feedback,
				confidence = excluded.confidence,
				rubric_scores = excluded.rubric_scores,
				model = excluded.model,
				prompt_version = excluded.prompt_version,
				raw_response = excluded.raw_response,
				error_message = excluded.error_message,
				completed_at = excluded.completed_at
			where assignment_ai_grade.assignment_attempt_id = excluded.assignment_attempt_id
			returning id
		`,
		attemptID,
		request.RequestID,
		request.Status,
		request.RecommendedPassed,
		request.RecommendedFeedback,
		request.Confidence,
		rubricScores,
		request.Model,
		request.PromptVersion,
		rawResponse,
		request.ErrorMessage,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return aiapi.AIGradeResultResponse{}, errAIGradeRequestAttemptMismatch
	}
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}

	applied := false
	skipReason := aiApplySkippedNotCompleted
	if request.Status == aiGradeStatusCompleted {
		state, err := loadAIAttemptGradeState(ctx, tx, attemptID)
		if errors.Is(err, pgx.ErrNoRows) {
			return aiapi.AIGradeResultResponse{}, errAnsweredAssignmentNotFound
		}
		if err != nil {
			return aiapi.AIGradeResultResponse{}, err
		}

		skipReason = app.aiApplySkipReason(state)
		if skipReason == "" {
			if err := gradeAssignmentAttempt(ctx, tx, gradeAttemptCommand{
				AssignmentID:     contextResponse.AssignmentID,
				AttemptID:        attemptID,
				Passed:           *request.RecommendedPassed,
				Feedback:         request.RecommendedFeedback,
				GradedByType:     graderTypeAI,
				GradedByService:  "ai",
				GradeSource:      gradeSourceAIAuto,
				AIGradeID:        &id,
				AIReviewStatus:   aiReviewStatusPending,
				PreserveAIReview: true,
			}); err != nil {
				return aiapi.AIGradeResultResponse{}, err
			}
			applied = true
		} else if skipReason == aiApplySkippedAutoApplyDisabled {
			if _, err := tx.Exec(
				ctx,
				`
					update assignment_attempt
					set ai_grade_id = $2,
						ai_review_status = coalesce(ai_review_status, 'pending_review')
					where id = $1
				`,
				attemptID,
				id,
			); err != nil {
				return aiapi.AIGradeResultResponse{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}

	return aiapi.AIGradeResultResponse{
		ID:                 id,
		RequestID:          request.RequestID,
		Status:             request.Status,
		Applied:            applied,
		ApplySkippedReason: skipReason,
	}, nil
}

func loadAIAttemptGradeState(ctx context.Context, querier assignmentGradeQuerier, attemptID int64) (aiAttemptGradeState, error) {
	var state aiAttemptGradeState
	err := querier.QueryRow(
		ctx,
		`
			select passed, graded_by_type, grade_source, ai_review_status
			from assignment_attempt
			where id = $1
				and reset_at is null
			for update
		`,
		attemptID,
	).Scan(&state.Passed, &state.GradedByType, &state.GradeSource, &state.AIReviewStatus)
	return state, err
}

func (app *app) aiApplySkipReason(state aiAttemptGradeState) string {
	if stringPtrEquals(state.AIReviewStatus, aiReviewStatusOverridden) || stringPtrEquals(state.GradeSource, gradeSourceTeacherOverride) {
		return aiApplySkippedTeacherOverride
	}
	if stringPtrEquals(state.AIReviewStatus, aiReviewStatusReviewed) {
		return aiApplySkippedTeacherReviewed
	}
	if stringPtrEquals(state.GradedByType, graderTypeTeacher) {
		return aiApplySkippedAlreadyTeacherGraded
	}
	if !app.aiAutoApplyGrades {
		return aiApplySkippedAutoApplyDisabled
	}
	return ""
}

func stringPtrEquals(value *string, expected string) bool {
	return value != nil && *value == expected
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
	requestBody := aiapi.GradeAssignmentRequest{
		RequestID: fmt.Sprintf("assignment-attempt-%d:%s", attemptID, app.aiPromptVersionGrader),
		AttemptID: attemptID,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, app.aiServiceURL+"/internal/ai/grade-assignment", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(serviceauth.AIServiceSecret, app.hqToAIServiceSecret)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ai service returned status %d", response.StatusCode)
	}
	return nil
}
