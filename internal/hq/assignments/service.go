package assignments

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store interface {
	Loader
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type LoadAfterWriteError struct {
	Err error
}

func (err *LoadAfterWriteError) Error() string {
	return err.Err.Error()
}

func (err *LoadAfterWriteError) Unwrap() error {
	return err.Err
}

func LoadByID(ctx context.Context, store Store, id int64) (Response, error) {
	assignments, err := Load(ctx, store, "where a.id = $1", id)
	if err != nil {
		return Response{}, err
	}

	if len(assignments) == 0 {
		return Response{}, pgx.ErrNoRows
	}

	return assignments[0], nil
}

func Create(ctx context.Context, store Store, request CreateRequest) (Response, error) {
	request.Category = strings.ToUpper(strings.TrimSpace(request.Category))
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.ExpectedAnswer = strings.TrimSpace(request.ExpectedAnswer)

	var id int64
	err := store.QueryRow(
		ctx,
		`
			insert into assignment (category, prompt, expected_answer)
			values ($1, $2, $3)
			returning id
		`,
		request.Category,
		request.Prompt,
		request.ExpectedAnswer,
	).Scan(&id)
	if err != nil {
		return Response{}, err
	}

	assignment, err := LoadByID(ctx, store, id)
	if err != nil {
		return Response{}, &LoadAfterWriteError{Err: err}
	}

	return assignment, nil
}

func Submit(ctx context.Context, store Store, studentUserID int64, assignmentID int64, request SubmitRequest) (Response, int64, error) {
	request.SubmittedAnswer = strings.TrimSpace(request.SubmittedAnswer)

	var attemptID int64
	err := store.QueryRow(
		ctx,
		`
			insert into assignment_attempt (assignment_id, student_user_id, attempt_number, submitted_answer)
			select a.id,
				$1,
				coalesce(max(aa.attempt_number), 0) + 1,
				$2
			from assignment a
			left join assignment_attempt aa on aa.assignment_id = a.id
				and aa.student_user_id = $1
			where a.id = $3
				and not exists (
					select 1
					from assignment_attempt active_attempt
					where active_attempt.assignment_id = a.id
						and active_attempt.student_user_id = $1
						and active_attempt.reset_at is null
				)
			group by a.id
			returning id
		`,
		studentUserID,
		request.SubmittedAnswer,
		assignmentID,
	).Scan(&attemptID)
	if err != nil {
		return Response{}, 0, err
	}

	assignment, err := LoadByID(ctx, store, assignmentID)
	if err != nil {
		return Response{}, attemptID, &LoadAfterWriteError{Err: err}
	}

	assignment.Attempts = AttemptsForStudent(assignment.Attempts, studentUserID)
	assignment.CurrentAttempt = CurrentAttempt(assignment.Attempts)
	return assignment, attemptID, nil
}

func Reset(ctx context.Context, store Store, assignmentID int64, request ResetRequest) (Response, error) {
	request.Feedback = strings.TrimSpace(request.Feedback)

	result, err := store.Exec(
		ctx,
		`
			update assignment_attempt
			set feedback = nullif($1, ''),
				reset_at = now()
			where id = $2
				and assignment_id = $3
				and reset_at is null
		`,
		request.Feedback,
		request.AttemptID,
		assignmentID,
	)
	if err != nil {
		return Response{}, err
	}

	if result.RowsAffected() == 0 {
		return Response{}, ErrAnsweredAssignmentNotFound
	}

	assignment, err := LoadByID(ctx, store, assignmentID)
	if err != nil {
		return Response{}, &LoadAfterWriteError{Err: err}
	}

	return assignment, nil
}

func Delete(ctx context.Context, store Store, assignmentID int64) (bool, error) {
	result, err := store.Exec(ctx, "delete from assignment where id = $1", assignmentID)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}
