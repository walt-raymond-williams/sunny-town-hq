package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"hq/internal/aiapi"
	hqassignments "hq/internal/hq/assignments"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecordAIGradeResultStoresRecommendationWithoutAutoApply(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	assignmentID, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	confidence := 0.91

	response, err := RecordGradeResult(ctx, db, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Confidence:          &confidence,
		RubricScores: []aiapi.RubricScore{{
			Name:   "correctness",
			Score:  1,
			Reason: "Matches expected answer.",
		}},
		Model:         "fake-grader",
		PromptVersion: "assignment-grader-v1",
	}, false)
	if err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}
	if response.Applied {
		t.Fatal("expected AI result to be stored without auto-applying grade")
	}
	if response.ApplySkippedReason != ApplySkippedAutoApplyDisabled {
		t.Fatalf("apply skipped reason = %q, want %q", response.ApplySkippedReason, ApplySkippedAutoApplyDisabled)
	}

	var count int
	var currentPassed *bool
	if err := db.QueryRow(ctx, "select count(*) from assignment_ai_grade where assignment_attempt_id = $1", attemptID).Scan(&count); err != nil {
		t.Fatalf("count ai grades: %v", err)
	}
	if err := db.QueryRow(ctx, "select passed from assignment_attempt where id = $1 and assignment_id = $2", attemptID, assignmentID).Scan(&currentPassed); err != nil {
		t.Fatalf("load attempt passed: %v", err)
	}
	if count != 1 || currentPassed != nil {
		t.Fatalf("ai grade count=%d passed=%v, want stored recommendation and ungraded attempt", count, currentPassed)
	}
}

func TestRecordAIGradeResultAutoAppliesThroughSharedGradeCommand(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	_, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	confidence := 0.97

	response, err := RecordGradeResult(ctx, db, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Confidence:          &confidence,
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}, true)
	if err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}
	if !response.Applied {
		t.Fatal("expected AI grade to auto-apply")
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var cookieAwarded bool
	var cookieQuantity int
	if err := db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, cookie_awarded
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &cookieAwarded); err != nil {
		t.Fatalf("load graded attempt metadata: %v", err)
	}
	if err := db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'cookie'
		`,
	).Scan(&cookieQuantity); err != nil {
		t.Fatalf("load cookie quantity: %v", err)
	}
	if gradedByType != hqassignments.GraderTypeAI || gradeSource != hqassignments.GradeSourceAIAuto || reviewStatus != hqassignments.AIReviewStatusPending || !cookieAwarded || cookieQuantity != 1 {
		t.Fatalf("metadata type=%q source=%q review=%q cookieAwarded=%v cookies=%d, want AI auto grade with one cookie", gradedByType, gradeSource, reviewStatus, cookieAwarded, cookieQuantity)
	}
}

func TestTeacherOverrideMarksAIAttemptOverridden(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	assignmentID, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	if _, err := RecordGradeResult(ctx, db, attemptID, aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}, true); err != nil {
		t.Fatalf("record ai grade result: %v", err)
	}

	teacherID := int64(124)
	if _, err := db.Exec(ctx, "insert into app_user (id, display_name) values ($1, 'Teacher')", teacherID); err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := hqassignments.GradeAttemptInTx(ctx, db, hqassignments.GradeAttemptCommand{
		AssignmentID:     assignmentID,
		AttemptID:        attemptID,
		Passed:           false,
		Feedback:         "Try again.",
		GradedByType:     hqassignments.GraderTypeTeacher,
		GradedByUserID:   &teacherID,
		GradeSource:      hqassignments.GradeSourceManual,
		PreserveAIReview: false,
	}); err != nil {
		t.Fatalf("teacher override grade: %v", err)
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var currentPassed bool
	if err := db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, passed
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load override metadata: %v", err)
	}
	if gradedByType != hqassignments.GraderTypeTeacher || gradeSource != hqassignments.GradeSourceTeacherOverride || reviewStatus != hqassignments.AIReviewStatusOverridden || currentPassed {
		t.Fatalf("override metadata type=%q source=%q review=%q passed=%v, want teacher override failure", gradedByType, gradeSource, reviewStatus, currentPassed)
	}
}

func TestAIGradeRetryDoesNotOverwriteTeacherOverride(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	assignmentID, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := RecordGradeResult(ctx, db, attemptID, request, true); err != nil {
		t.Fatalf("record initial ai grade result: %v", err)
	}

	teacherID := int64(124)
	if _, err := db.Exec(ctx, "insert into app_user (id, display_name) values ($1, 'Teacher')", teacherID); err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := hqassignments.GradeAttemptInTx(ctx, db, hqassignments.GradeAttemptCommand{
		AssignmentID:     assignmentID,
		AttemptID:        attemptID,
		Passed:           false,
		Feedback:         "Try again.",
		GradedByType:     hqassignments.GraderTypeTeacher,
		GradedByUserID:   &teacherID,
		GradeSource:      hqassignments.GradeSourceManual,
		PreserveAIReview: false,
	}); err != nil {
		t.Fatalf("teacher override grade: %v", err)
	}

	response, err := RecordGradeResult(ctx, db, attemptID, request, true)
	if err != nil {
		t.Fatalf("retry ai grade result: %v", err)
	}
	if response.Applied || response.ApplySkippedReason != ApplySkippedTeacherOverride {
		t.Fatalf("retry applied=%v reason=%q, want skipped teacher override", response.Applied, response.ApplySkippedReason)
	}

	var gradedByType string
	var gradeSource string
	var reviewStatus string
	var currentPassed bool
	if err := db.QueryRow(
		ctx,
		`
			select graded_by_type, grade_source, ai_review_status, passed
			from assignment_attempt
			where id = $1
		`,
		attemptID,
	).Scan(&gradedByType, &gradeSource, &reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load retry metadata: %v", err)
	}
	if gradedByType != hqassignments.GraderTypeTeacher || gradeSource != hqassignments.GradeSourceTeacherOverride || reviewStatus != hqassignments.AIReviewStatusOverridden || currentPassed {
		t.Fatalf("retry metadata type=%q source=%q review=%q passed=%v, want teacher override preserved", gradedByType, gradeSource, reviewStatus, currentPassed)
	}
}

func TestAIGradeRetryDoesNotOverwriteReviewedAttempt(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	_, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := RecordGradeResult(ctx, db, attemptID, request, true); err != nil {
		t.Fatalf("record initial ai grade result: %v", err)
	}
	if _, err := db.Exec(ctx, "update assignment_attempt set ai_review_status = $2 where id = $1", attemptID, hqassignments.AIReviewStatusReviewed); err != nil {
		t.Fatalf("mark ai grade reviewed: %v", err)
	}

	passed = false
	request.RecommendedPassed = &passed
	request.RecommendedFeedback = "Incorrect."
	response, err := RecordGradeResult(ctx, db, attemptID, request, true)
	if err != nil {
		t.Fatalf("retry ai grade result: %v", err)
	}
	if response.Applied || response.ApplySkippedReason != ApplySkippedTeacherReviewed {
		t.Fatalf("retry applied=%v reason=%q, want skipped teacher reviewed", response.Applied, response.ApplySkippedReason)
	}

	var reviewStatus string
	var currentPassed bool
	if err := db.QueryRow(ctx, "select ai_review_status, passed from assignment_attempt where id = $1", attemptID).Scan(&reviewStatus, &currentPassed); err != nil {
		t.Fatalf("load reviewed attempt: %v", err)
	}
	if reviewStatus != hqassignments.AIReviewStatusReviewed || !currentPassed {
		t.Fatalf("review=%q passed=%v, want reviewed AI pass preserved", reviewStatus, currentPassed)
	}
}

func TestAIGradeDuplicateRequestIDSameAttemptIsIdempotent(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	_, attemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	first, err := RecordGradeResult(ctx, db, attemptID, request, true)
	if err != nil {
		t.Fatalf("record first ai grade result: %v", err)
	}
	request.RecommendedFeedback = "Still correct."
	second, err := RecordGradeResult(ctx, db, attemptID, request, true)
	if err != nil {
		t.Fatalf("record second ai grade result: %v", err)
	}
	if !first.Applied || !second.Applied || first.ID != second.ID {
		t.Fatalf("first=%#v second=%#v, want same applied AI grade", first, second)
	}

	var aiGradeCount int
	var cookieQuantity int
	if err := db.QueryRow(ctx, "select count(*) from assignment_ai_grade where assignment_attempt_id = $1", attemptID).Scan(&aiGradeCount); err != nil {
		t.Fatalf("count ai grades: %v", err)
	}
	if err := db.QueryRow(
		ctx,
		`
			select sii.quantity
			from student_inventory_item sii
			join inventory_item_type iit on iit.id = sii.item_type_id
			where sii.app_user_id = 123 and iit.key = 'cookie'
		`,
	).Scan(&cookieQuantity); err != nil {
		t.Fatalf("load cookie quantity: %v", err)
	}
	if aiGradeCount != 1 || cookieQuantity != 1 {
		t.Fatalf("aiGradeCount=%d cookieQuantity=%d, want one idempotent AI grade and one cookie", aiGradeCount, cookieQuantity)
	}
}

func TestAIGradeDuplicateRequestIDDifferentAttemptIsRejected(t *testing.T) {
	db, cleanup := testAIDB(t)
	defer cleanup()

	ctx := context.Background()
	_, firstAttemptID := seedAIAssignmentAttempt(t, db)
	_, secondAttemptID := seedAIAssignmentAttempt(t, db)
	passed := true
	request := aiapi.AIGradeResultRequest{
		RequestID:           "assignment-attempt-1:assignment-grader-v1",
		Status:              GradeStatusCompleted,
		RecommendedPassed:   &passed,
		RecommendedFeedback: "Correct.",
		Model:               "fake-grader",
		PromptVersion:       "assignment-grader-v1",
	}
	if _, err := RecordGradeResult(ctx, db, firstAttemptID, request, true); err != nil {
		t.Fatalf("record first ai grade result: %v", err)
	}

	_, err := RecordGradeResult(ctx, db, secondAttemptID, request, true)
	if !errors.Is(err, ErrGradeRequestAttemptMismatch) {
		t.Fatalf("second ai grade error = %v, want request attempt mismatch", err)
	}

	var secondAIGradeID *int64
	var secondPassed *bool
	if err := db.QueryRow(ctx, "select ai_grade_id, passed from assignment_attempt where id = $1", secondAttemptID).Scan(&secondAIGradeID, &secondPassed); err != nil {
		t.Fatalf("load second attempt: %v", err)
	}
	if secondAIGradeID != nil || secondPassed != nil {
		t.Fatalf("second ai_grade_id=%v passed=%v, want untouched second attempt", secondAIGradeID, secondPassed)
	}
}

func seedAIAssignmentAttempt(t *testing.T, db *pgxpool.Pool) (int64, int64) {
	t.Helper()
	ctx := context.Background()

	var assignmentID int64
	if err := db.QueryRow(
		ctx,
		"insert into assignment (category, prompt, expected_answer) values ('MATH', 'What is 2 + 2?', '4') returning id",
	).Scan(&assignmentID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	var attemptID int64
	if err := db.QueryRow(
		ctx,
		`
			insert into assignment_attempt (
				assignment_id,
				student_user_id,
				attempt_number,
				submitted_answer
			)
			values ($1, 123, 1, '4')
			returning id
		`,
		assignmentID,
	).Scan(&attemptID); err != nil {
		t.Fatalf("seed assignment attempt: %v", err)
	}

	return assignmentID, attemptID
}

func testAIDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	databaseURL := os.Getenv("HQ_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("HQ_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}

	schema := fmt.Sprintf("ai_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		adminPool.Close()
		t.Fatalf("create schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		adminPool.Close()
		t.Fatalf("parse pool config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		adminPool.Close()
		t.Fatalf("connect test pool: %v", err)
	}

	statements := []string{
		`create table app_user (
			id bigint primary key,
			display_name text not null
		)`,
		`create table assignment (
			id bigserial primary key,
			category text not null,
			prompt text not null,
			expected_answer text not null,
			created_at timestamptz not null default now()
		)`,
		`create table assignment_attempt (
			id bigserial primary key,
			assignment_id bigint not null references assignment(id) on delete cascade,
			student_user_id bigint not null references app_user(id) on delete cascade,
			attempt_number integer not null,
			submitted_answer text not null,
			date_submitted timestamptz not null default now(),
			passed boolean null,
			feedback text null,
			date_graded timestamptz null,
			cookie_awarded boolean not null default false,
			reset_at timestamptz null,
			graded_by_type text null,
			graded_by_user_id bigint null references app_user(id) on delete set null,
			graded_by_service text null,
			grade_source text null,
			ai_review_status text null,
			ai_grade_id bigint null,
			unique (assignment_id, student_user_id, attempt_number)
		)`,
		`create table assignment_ai_grade (
			id bigserial primary key,
			assignment_attempt_id bigint not null references assignment_attempt(id) on delete cascade,
			request_id text not null unique,
			status text not null,
			recommended_passed boolean null,
			recommended_feedback text null,
			confidence numeric null,
			rubric_scores jsonb not null default '[]'::jsonb,
			model text null,
			prompt_version text not null,
			raw_response jsonb null,
			error_message text null,
			created_at timestamptz not null default now(),
			completed_at timestamptz null
		)`,
		`alter table assignment_attempt
			add constraint assignment_attempt_ai_grade_id_fkey
			foreign key (ai_grade_id) references assignment_ai_grade(id) on delete set null`,
		`create table inventory_item_type (
			id bigserial primary key,
			key text not null unique,
			name text not null,
			description text not null default '',
			equip_slot text null,
			visual_key text null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)`,
		`insert into inventory_item_type (key, name, description)
			values ('cookie', 'Cookie', 'A treat for your pet.')`,
		`create table student_inventory_item (
			app_user_id bigint not null references app_user(id) on delete cascade,
			item_type_id bigint not null references inventory_item_type(id) on delete restrict,
			quantity integer not null default 0,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			primary key (app_user_id, item_type_id),
			constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
		)`,
		`insert into app_user (id, display_name) values (123, 'Student')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			db.Close()
			_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
			adminPool.Close()
			t.Fatalf("setup statement failed: %v", err)
		}
	}

	return db, func() {
		db.Close()
		_, _ = adminPool.Exec(ctx, "drop schema "+schema+" cascade")
		adminPool.Close()
	}
}
