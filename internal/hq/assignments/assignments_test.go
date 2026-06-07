package assignments

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestCurrentAttemptSkipsResetAttempts(t *testing.T) {
	resetAt := time.Now()
	attempts := []AttemptResponse{
		{ID: 1, ResetAt: nil},
		{ID: 2, ResetAt: &resetAt},
	}

	current := CurrentAttempt(attempts)
	if current == nil {
		t.Fatal("expected current attempt")
	}
	if current.ID != 1 {
		t.Fatalf("current attempt ID = %d, want 1", current.ID)
	}
}

func TestTypedListQueriesOwnSQLShape(t *testing.T) {
	tests := []struct {
		name      string
		run       func(context.Context, Loader) ([]Response, error)
		wantQuery string
		wantArgs  []any
	}{
		{
			name:      "all",
			run:       func(ctx context.Context, loader Loader) ([]Response, error) { return ListAll(ctx, loader) },
			wantQuery: "order by a.id desc",
			wantArgs:  []any{},
		},
		{
			name: "category",
			run: func(ctx context.Context, loader Loader) ([]Response, error) {
				return ListByCategory(ctx, loader, "MATH")
			},
			wantQuery: "where a.category = $1 order by a.id desc",
			wantArgs:  []any{"MATH"},
		},
		{
			name: "next student category",
			run: func(ctx context.Context, loader Loader) ([]Response, error) {
				return ListNextForStudent(ctx, loader, 42, "READING")
			},
			wantQuery: "and a.category = $2",
			wantArgs:  []any{int64(42), "READING"},
		},
		{
			name: "graded student",
			run: func(ctx context.Context, loader Loader) ([]Response, error) {
				return ListGradedForStudent(ctx, loader, 42)
			},
			wantQuery: "and aa.passed is not null",
			wantArgs:  []any{int64(42)},
		},
		{
			name:      "answered",
			run:       func(ctx context.Context, loader Loader) ([]Response, error) { return ListAnswered(ctx, loader) },
			wantQuery: "order by (",
			wantArgs:  []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &recordingLoader{err: errStopQuery}
			_, err := tt.run(context.Background(), loader)
			if !errors.Is(err, errStopQuery) {
				t.Fatalf("error = %v, want %v", err, errStopQuery)
			}
			if !strings.Contains(loader.query, tt.wantQuery) {
				t.Fatalf("query = %q, want to contain %q", loader.query, tt.wantQuery)
			}
			if len(loader.args) != len(tt.wantArgs) {
				t.Fatalf("arg count = %d, want %d", len(loader.args), len(tt.wantArgs))
			}
			for index, want := range tt.wantArgs {
				if loader.args[index] != want {
					t.Fatalf("arg %d = %#v, want %#v", index, loader.args[index], want)
				}
			}
		})
	}
}

func TestRestrictToStudentFiltersAttemptsAndCurrentAttempt(t *testing.T) {
	assignments := []Response{
		{
			ID: 10,
			Attempts: []AttemptResponse{
				{ID: 1, StudentUserID: 100},
				{ID: 2, StudentUserID: 200},
				{ID: 3, StudentUserID: 100},
			},
		},
	}

	RestrictToStudent(assignments, 100)

	if len(assignments[0].Attempts) != 2 {
		t.Fatalf("attempt count = %d, want 2", len(assignments[0].Attempts))
	}
	if assignments[0].Attempts[0].ID != 1 || assignments[0].Attempts[1].ID != 3 {
		t.Fatalf("filtered attempts = %#v, want IDs 1 and 3", assignments[0].Attempts)
	}
	if assignments[0].CurrentAttempt == nil || assignments[0].CurrentAttempt.ID != 3 {
		t.Fatalf("current attempt = %#v, want ID 3", assignments[0].CurrentAttempt)
	}
}

func TestGradedAttemptsOnlyKeepsGraded(t *testing.T) {
	passed := true
	failed := false
	attempts := []AttemptResponse{
		{ID: 1, Passed: nil},
		{ID: 2, Passed: &passed},
		{ID: 3, Passed: &failed},
	}

	graded := GradedAttempts(attempts)

	if len(graded) != 2 {
		t.Fatalf("graded count = %d, want 2", len(graded))
	}
	if graded[0].ID != 2 || graded[1].ID != 3 {
		t.Fatalf("graded attempts = %#v, want IDs 2 and 3", graded)
	}
}

var errStopQuery = errors.New("stop query")

type recordingLoader struct {
	query string
	args  []any
	err   error
}

func (loader *recordingLoader) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	loader.query = query
	loader.args = args
	return nil, loader.err
}
