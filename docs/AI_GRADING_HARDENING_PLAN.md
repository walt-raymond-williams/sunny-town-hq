# AI Grading Hardening Plan

This plan tracks the follow-up work from the AI service code review. The goal is to make AI grading safer under retries, teacher overrides, and future async/background execution without changing the existing manual teacher grading workflow or the current student assignment flow.

## Current Status

- Status: implementation in progress
- Owner: engineering
- Scope: HQ AI grading persistence/apply path, AI service retry behavior, tests, and commit hygiene notes
- Related docs:
  - `docs/AI_SERVICE_IMPLEMENTATION_PLAN.md`
  - `docs/TEST_CASE_COVERAGE_MATRIX.md`
  - `docs/USER_STORIES_AND_REQUIREMENTS.md`

## Goals

- Keep HQ authoritative for final assignment grading state.
- Let AI recommendations be stored safely even when they are not applied.
- Prevent AI retries from overwriting teacher-reviewed or teacher-overridden grades.
- Keep `request_id` idempotency while preventing cross-attempt grade linkage.
- Add focused tests around retry and override cases.
- Avoid breaking existing manual assignment creation, submission, grading, reset, rewards, and student-visible result behavior.

## Non-Goals

- Do not add OpenAI API calls yet.
- Do not change the browser-facing assignment APIs unless required for safe metadata visibility.
- Do not require teachers to review every AI grade before it can appear as graded.
- Do not let the AI service write directly to the HQ database.
- Do not change Sunny Town behavior as part of this hardening pass.

## Work Tracker

| ID | Status | Area | Work |
| --- | --- | --- | --- |
| H1 | Implemented | HQ grading | Prevent AI auto-apply from overwriting teacher grades or teacher review decisions. |
| H2 | Implemented | HQ persistence | Make AI result upserts reject duplicate `request_id` values that belong to a different attempt. |
| H3 | Implemented | HQ persistence | Make storing an AI result and deciding whether to apply it transactional. |
| H4 | Implemented | API contract | Return an apply skip reason when an AI result is stored but not applied. |
| H5 | Implemented | Tests | Add retry and idempotency regression tests for AI grading. |
| H6 | Planned | Commit hygiene | Separate or document unrelated Sunny Town/frontend changes in the AI foundation commit. |

## Detailed Plan

### H1: Protect Teacher Authority

Current risk:

- AI auto-apply can run after a teacher has overridden or reviewed an AI grade.
- A retry from the AI service could change the attempt back to the AI result.

Implementation:

- Before auto-applying a completed AI result, load the current attempt grading metadata.
- Allow AI auto-apply only when the attempt is:
  - ungraded, or
  - already AI-graded and still pending review.
- Skip AI auto-apply when any of these are true:
  - `graded_by_type = 'teacher'`
  - `grade_source = 'teacher_override'`
  - `ai_review_status in ('reviewed', 'overridden')`
- Still store the AI result row for audit/debugging when apply is skipped.

Expected behavior:

- Manual teacher grading remains unchanged.
- Teacher override remains final unless a teacher manually grades again.
- AI recommendations can still be recorded after a teacher override, but they cannot mutate the final grade.

Implementation note:

- Completed in HQ by checking current attempt metadata before AI auto-apply. Teacher override, teacher-reviewed, and teacher-graded states now skip AI mutation while preserving the stored AI result.

### H2: Make `request_id` Idempotency Attempt-Safe

Current risk:

- `assignment_ai_grade.request_id` is unique, but a duplicate request id submitted for another attempt could return the existing AI grade row and link it to the wrong attempt.

Implementation:

- Keep `request_id` unique.
- On upsert, only update the existing row when `assignment_ai_grade.assignment_attempt_id = excluded.assignment_attempt_id`.
- If the conflict belongs to a different attempt, return a validation error instead of linking or applying the result.
- Consider a named error such as `errAIGradeRequestAttemptMismatch` for tests and handler response clarity.

Expected behavior:

- Retrying the same request for the same attempt remains safe.
- Reusing a request id for a different attempt is rejected.

Implementation note:

- Completed by constraining the `request_id` upsert to the same `assignment_attempt_id` and returning a named mismatch error when a request id belongs to another attempt.

### H3: Transactional Store-And-Apply

Current risk:

- AI result insert/update, attempt metadata update, and optional auto-apply currently happen as separate operations.
- A failure between steps can leave partial state that is hard to reason about.

Implementation:

- Wrap AI result record/update, apply eligibility check, attempt metadata update, and optional grade application in one transaction.
- Reuse the existing shared grade logic, but allow it to operate inside an existing transaction.
- Keep cookie reward idempotency exactly as it is today.

Expected behavior:

- Either the whole AI result/apply decision succeeds, or none of it mutates final grading state.
- Existing teacher grading behavior continues to use the same shared grading logic.

Implementation note:

- Completed by letting the shared grading command run inside an existing transaction for the AI result path while preserving the standalone transaction wrapper used by teacher grading.

### H4: Return Apply Skip Reason

Current limitation:

- The AI result response only says whether the result was applied.
- It does not explain why an apply was skipped.

Implementation:

- Add `apply_skipped_reason` to `AIGradeResultResponse`.
- Suggested values:
  - `auto_apply_disabled`
  - `teacher_override`
  - `teacher_reviewed`
  - `already_teacher_graded`
  - `not_completed`
- Keep the field omitted or empty when `applied = true`.

Expected behavior:

- AI service logs and future background workers can distinguish expected skips from real failures.
- No existing client should break because this is an additive response field on an internal API.

Implementation note:

- Completed with an additive `apply_skipped_reason` field on the internal AI grade response.

### H5: Add Regression Tests

Required tests:

- Completed: AI auto-applies, teacher overrides, same AI request retries, teacher grade remains unchanged.
- Completed: AI auto-applies, teacher marks review complete, AI retry does not overwrite review.
- Completed: Duplicate `request_id` for the same attempt remains idempotent.
- Completed: Duplicate `request_id` for a different attempt is rejected and does not relink `ai_grade_id`.
- Completed: AI result is stored but not applied when auto-apply is disabled, with `apply_skipped_reason = auto_apply_disabled`.

Useful additional tests:

- Failed AI result is stored but never applied.
- Pending AI result is stored but never applied.
- Teacher manual grading with no AI grade still records `graded_by_type = teacher` and awards cookies as before.

Verification:

```powershell
go test ./cmd/hq
go test ./cmd/ai
go test ./...
docker compose -f deploy\docker-compose.yml config --quiet
```

### H6: Commit Hygiene

Current issue:

- The AI foundation commit also includes Sunny Town and frontend resource-node changes.

Options:

- Preferred: split unrelated Sunny Town/frontend files into a separate commit if it can be done without disrupting the other active agent.
- Acceptable: leave the commit as-is but document that the commit includes unrelated Sunny Town resource-node collision/rendering changes.

Constraint:

- Do not modify or revert active Sunny Town/frontend work unless explicitly coordinated.

## Compatibility Checklist

Before marking this hardening pass complete, verify:

- Manual teacher grading still passes existing tests.
- Teacher reset workflow still works.
- Student submission still returns the assignment immediately.
- AI grading still remains disabled by default in Docker Compose.
- AI service still works with the fake provider and no OpenAI key.
- Cookie awards remain idempotent.
- Teacher override cannot be reversed by AI retry.
- Duplicate AI request ids cannot cross-link attempts.

## Completion Criteria

- H1-H5 are implemented and covered by tests.
- Full backend test suite passes.
- Docker Compose config remains valid.
- Any commit hygiene decision for H6 is documented before committing follow-up work.
