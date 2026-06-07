package ai

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
	hqassignments "hq/internal/hq/assignments"
	"hq/internal/serviceauth"

	"github.com/jackc/pgx/v5"
)

const (
	GradeStatusPending   = "pending"
	GradeStatusCompleted = "completed"
	GradeStatusFailed    = "failed"

	ApplySkippedNotCompleted         = "not_completed"
	ApplySkippedAutoApplyDisabled    = "auto_apply_disabled"
	ApplySkippedTeacherOverride      = "teacher_override"
	ApplySkippedTeacherReviewed      = "teacher_reviewed"
	ApplySkippedAlreadyTeacherGraded = "already_teacher_graded"
)

var ErrGradeRequestAttemptMismatch = errors.New("ai grade request id belongs to a different assignment attempt")

type Store interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

type AttemptGradeState struct {
	Passed         *bool
	GradedByType   *string
	GradeSource    *string
	AIReviewStatus *string
}

type Handler struct {
	store           Store
	serviceSecret   string
	autoApplyGrades bool
}

type HandlerConfig struct {
	Store           Store
	ServiceSecret   string
	AutoApplyGrades bool
}

type TriggerConfig struct {
	Enabled       bool
	ServiceURL    string
	ServiceSecret string
	PromptVersion string
	Client        *http.Client
	Timeout       time.Duration
}

func NewHandler(config HandlerConfig) *Handler {
	return &Handler{
		store:           config.Store,
		serviceSecret:   config.ServiceSecret,
		autoApplyGrades: config.AutoApplyGrades,
	}
}

func (handler *Handler) HandleAssignmentAttempt(w http.ResponseWriter, r *http.Request) {
	if !serviceauth.Authorized(r.Header, serviceauth.HQServiceSecret, handler.serviceSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "service authentication required",
		})
		return
	}

	attemptID, action, ok := parseAssignmentAttemptPath(w, r.URL.Path)
	if !ok {
		return
	}

	switch action {
	case "grading-context":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		contextResponse, err := LoadGradingContext(r.Context(), handler.store, attemptID)
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
		response, err := RecordGradeResult(r.Context(), handler.store, attemptID, request, handler.autoApplyGrades)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, hqassignments.ErrAnsweredAssignmentNotFound) {
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

func parseAssignmentAttemptPath(w http.ResponseWriter, path string) (int64, string, bool) {
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

func LoadGradingContext(ctx context.Context, store Store, attemptID int64) (aiapi.GradingContextResponse, error) {
	var response aiapi.GradingContextResponse
	err := store.QueryRow(
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

func RecordGradeResult(ctx context.Context, store Store, attemptID int64, request aiapi.AIGradeResultRequest, autoApplyGrades bool) (aiapi.AIGradeResultResponse, error) {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.Status = strings.TrimSpace(request.Status)
	request.RecommendedFeedback = strings.TrimSpace(request.RecommendedFeedback)
	request.Model = strings.TrimSpace(request.Model)
	request.PromptVersion = strings.TrimSpace(request.PromptVersion)
	request.ErrorMessage = strings.TrimSpace(request.ErrorMessage)
	if request.RequestID == "" || request.PromptVersion == "" {
		return aiapi.AIGradeResultResponse{}, errors.New("ai grade result is missing required fields")
	}
	if request.Status != GradeStatusPending && request.Status != GradeStatusCompleted && request.Status != GradeStatusFailed {
		return aiapi.AIGradeResultResponse{}, errors.New("invalid ai grade status")
	}
	if request.Status == GradeStatusCompleted && request.RecommendedPassed == nil {
		return aiapi.AIGradeResultResponse{}, errors.New("completed ai grade is missing recommended_passed")
	}

	contextResponse, err := LoadGradingContext(ctx, store, attemptID)
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

	tx, err := store.Begin(ctx)
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
		return aiapi.AIGradeResultResponse{}, ErrGradeRequestAttemptMismatch
	}
	if err != nil {
		return aiapi.AIGradeResultResponse{}, err
	}

	applied := false
	skipReason := ApplySkippedNotCompleted
	if request.Status == GradeStatusCompleted {
		state, err := LoadAttemptGradeState(ctx, tx, attemptID)
		if errors.Is(err, pgx.ErrNoRows) {
			return aiapi.AIGradeResultResponse{}, hqassignments.ErrAnsweredAssignmentNotFound
		}
		if err != nil {
			return aiapi.AIGradeResultResponse{}, err
		}

		skipReason = ApplySkipReason(state, autoApplyGrades)
		if skipReason == "" {
			if err := hqassignments.GradeAttempt(ctx, tx, hqassignments.GradeAttemptCommand{
				AssignmentID:     contextResponse.AssignmentID,
				AttemptID:        attemptID,
				Passed:           *request.RecommendedPassed,
				Feedback:         request.RecommendedFeedback,
				GradedByType:     hqassignments.GraderTypeAI,
				GradedByService:  "ai",
				GradeSource:      hqassignments.GradeSourceAIAuto,
				AIGradeID:        &id,
				AIReviewStatus:   hqassignments.AIReviewStatusPending,
				PreserveAIReview: true,
			}); err != nil {
				return aiapi.AIGradeResultResponse{}, err
			}
			applied = true
		} else if skipReason == ApplySkippedAutoApplyDisabled {
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

func LoadAttemptGradeState(ctx context.Context, querier hqassignments.GradeQuerier, attemptID int64) (AttemptGradeState, error) {
	var state AttemptGradeState
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

func ApplySkipReason(state AttemptGradeState, autoApplyGrades bool) string {
	if stringPtrEquals(state.AIReviewStatus, hqassignments.AIReviewStatusOverridden) || stringPtrEquals(state.GradeSource, hqassignments.GradeSourceTeacherOverride) {
		return ApplySkippedTeacherOverride
	}
	if stringPtrEquals(state.AIReviewStatus, hqassignments.AIReviewStatusReviewed) {
		return ApplySkippedTeacherReviewed
	}
	if stringPtrEquals(state.GradedByType, hqassignments.GraderTypeTeacher) {
		return ApplySkippedAlreadyTeacherGraded
	}
	if !autoApplyGrades {
		return ApplySkippedAutoApplyDisabled
	}
	return ""
}

func RequestGrade(ctx context.Context, client *http.Client, serviceURL, serviceSecret, promptVersion string, attemptID int64) error {
	requestBody := aiapi.GradeAssignmentRequest{
		RequestID: fmt.Sprintf("assignment-attempt-%d:%s", attemptID, promptVersion),
		AttemptID: attemptID,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(serviceURL, "/")+"/internal/ai/grade-assignment", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(serviceauth.AIServiceSecret, serviceSecret)

	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ai service returned status %d", response.StatusCode)
	}
	return nil
}

func TriggerGradeAsync(config TriggerConfig, attemptID int64) {
	if !config.Enabled || config.ServiceURL == "" || config.ServiceSecret == "" {
		return
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := RequestGrade(ctx, config.Client, config.ServiceURL, config.ServiceSecret, config.PromptVersion, attemptID); err != nil {
			log.Printf("request ai grade for attempt %d: %v", attemptID, err)
		}
	}()
}

func stringPtrEquals(value *string, expected string) bool {
	return value != nil && *value == expected
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
