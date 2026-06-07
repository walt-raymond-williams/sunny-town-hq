# AI Service Implementation Plan

This plan describes how to add a dedicated AI service for HQ and future services. The first use case is automatic assignment grading. Later use cases include AI-assisted assignment generation, Sunny Town NPC/tutor behavior, and other cross-service AI workflows.

Current decision:

- Build a separate AI service now.
- Keep HQ authoritative for schoolwork state, final grade records, rewards, student-visible feedback, and teacher review.
- Let the AI service act as an internal service client and AI workflow orchestrator.
- Reuse HQ grading logic through shared internal grading commands/endpoints instead of creating a second grading path.

OpenAI docs note: current OpenAI models are exposed through the Responses API and support structured outputs. Use structured outputs for grader/generator responses so HQ does not parse free-form text. Re-check the current OpenAI docs before implementation because model names and recommended defaults change over time.

## Goals

- Automatically grade student assignment attempts with AI.
- Store who graded each attempt: teacher user or AI service.
- Let AI-graded work appear as graded to students by default.
- Let teachers review, accept, or override AI-graded work later.
- Preserve one HQ-owned final grading path so rewards and audit behavior remain consistent.
- Create a reusable AI service boundary for future assignment generation and Sunny Town AI interactions.
- Keep OpenAI credentials isolated from HQ, Sunny Town, and the browser.

## Non-Goals For First Implementation

- No AI service database writes directly into the HQ database.
- No direct browser access to the AI service.
- No AI mutation of Sunny Town realtime world state.
- No fully autonomous cross-service agent actions beyond explicitly scoped internal APIs.
- No fine-tuning or custom model training.
- No streaming UI for grading.

## Target Architecture

```text
Browser
  Student submits work
  Teacher reviews grades and overrides if needed

        |
        v

HQ Service
  owns assignments, attempts, final grades, feedback, rewards, teacher review
  exposes internal service-authenticated endpoints for AI workflows
  calls AI service after student submission or teacher request

        |
        | service-authenticated HTTP
        v

AI Service
  owns OpenAI API key
  owns prompt templates and prompt versions
  validates structured AI outputs
  logs model/prompt/request metadata
  calls HQ internal endpoints to fetch context and submit recommendations/final AI grades

        |
        v

OpenAI API
```

Boundary rule:

```text
The AI service may recommend, draft, grade through approved HQ commands, explain, and orchestrate.
Domain services remain authoritative for validation, persistence, rewards, and final state changes.
```

## Service Ownership

### HQ Owns

- `assignment`
- `assignment_attempt`
- final `passed` / `feedback` values
- grader identity and review status
- student-visible grade results
- cookie/star reward rules
- assignment drafts once persisted
- teacher review and override state
- internal endpoints that expose only the context/actions AI is allowed to use

### AI Service Owns

- OpenAI API integration
- model configuration
- prompt templates
- prompt version names
- structured output schemas
- AI job orchestration
- retries/backoff for provider calls
- AI request ids and idempotency keys
- AI-specific logs, token usage, and cost metadata
- future cross-service AI workflows

### Sunny Town Owns

- realtime world state
- map/player/NPC interaction validation
- deciding whether a player is allowed to trigger an AI interaction
- future routing to HQ or AI through explicit internal APIs

## Repository Changes

Add:

```text
cmd/ai/
  main.go
  openai_client.go
  grader.go
  generator.go
  prompts/

internal/aiapi/
  grading.go
  generation.go
  errors.go

internal/serviceauth/
  service_auth.go

deploy/ai/Dockerfile
```

Update:

```text
deploy/docker-compose.yml
cmd/hq/main.go
internal/hq/users/
cmd/hq/reward_test.go or new assignment tests
docs/PROJECT_SETUP.md
docs/TESTING_GUIDELINES.md
docs/USER_STORIES_AND_REQUIREMENTS.md
docs/TEST_CASE_COVERAGE_MATRIX.md
```

## Configuration

AI service environment:

```text
AI_PORT=18083
OPENAI_API_KEY=...
OPENAI_MODEL=...
AI_TO_HQ_SERVICE_SECRET=...
HQ_INTERNAL_BASE_URL=http://hq:8080
AI_PROMPT_VERSION_GRADER=assignment-grader-v1
AI_PROMPT_VERSION_GENERATOR=assignment-generator-v1
```

HQ environment:

```text
AI_SERVICE_URL=http://ai:18083
HQ_TO_AI_SERVICE_SECRET=...
AI_TO_HQ_SERVICE_SECRET=...
AI_GRADING_ENABLED=true
AI_AUTO_APPLY_GRADES=true
```

Local host ports:

```text
HQ:          http://localhost:18080
Keycloak:    http://localhost:18081
Sunny Town:  http://localhost:18082
AI service:  http://localhost:18083
```

## Data Model Changes

### Assignment Attempt Grader Metadata

Add columns to `assignment_attempt`:

```text
graded_by_type text null
  values: teacher, ai

graded_by_user_id bigint null
  references app_user(id)
  set when a human teacher grades or overrides

graded_by_service text null
  example: ai

grade_source text null
  values: manual, ai_auto, ai_override, teacher_override

ai_review_status text null
  values: pending_review, reviewed, overridden

ai_grade_id bigint null
  references assignment_ai_grade(id)
```

Rules:

- Manual teacher grading sets `graded_by_type = 'teacher'`.
- Automatic AI grading sets `graded_by_type = 'ai'`, `grade_source = 'ai_auto'`, and `ai_review_status = 'pending_review'`.
- Teacher reviewing without changes sets `ai_review_status = 'reviewed'`.
- Teacher changing AI grade/feedback sets `graded_by_type = 'teacher'`, `grade_source = 'teacher_override'`, and `ai_review_status = 'overridden'`.
- Passing rewards must still be idempotent and run through the existing HQ grading path.

### AI Grade Audit Table

Create:

```text
assignment_ai_grade
  id bigserial primary key
  assignment_attempt_id bigint not null references assignment_attempt(id) on delete cascade
  request_id text not null unique
  status text not null
    values: pending, completed, failed
  recommended_passed boolean null
  recommended_feedback text null
  confidence numeric null
  rubric_scores jsonb not null default '[]'::jsonb
  model text null
  prompt_version text not null
  raw_response jsonb null
  error_message text null
  created_at timestamptz not null default now()
  completed_at timestamptz null
```

Purpose:

- Preserve AI output separate from the current final grade.
- Support teacher review/override without losing the AI recommendation.
- Support prompt/model evaluation later.
- Keep idempotency through `request_id`.

### Assignment Drafts For Generation

Create later, when implementing AI question generation:

```text
assignment_draft
  id bigserial primary key
  created_by_user_id bigint null references app_user(id)
  category text not null
  prompt text not null
  expected_answer text not null
  rubric text null
  source text not null
    values: manual, ai
  ai_model text null
  prompt_version text null
  status text not null
    values: draft, published, discarded
  published_assignment_id bigint null references assignment(id)
  created_at timestamptz not null default now()
  updated_at timestamptz not null default now()
```

## HQ Internal API For AI

Use service-authenticated endpoints. Do not expose these to the browser.

Auth header:

```text
X-HQ-Service-Name: ai
X-HQ-Service-Secret: ...
```

### Get Grading Context

```text
GET /api/internal/ai/assignment-attempts/{attempt_id}/grading-context
```

Response:

```json
{
  "assignment_id": 42,
  "attempt_id": 123,
  "student_user_id": 7,
  "student_display_name": "Student",
  "category": "MATH",
  "grade_level": "second-grade",
  "prompt": "What is 7 + 5?",
  "expected_answer": "12",
  "rubric": "Accept numerically equivalent answers.",
  "submitted_answer": "twelve",
  "attempt_number": 1,
  "submitted_at": "2026-06-06T16:00:00Z"
}
```

### Record AI Grade Result

```text
POST /api/internal/ai/assignment-attempts/{attempt_id}/ai-grade
```

Request:

```json
{
  "request_id": "assignment-attempt-123:assignment-grader-v1",
  "status": "completed",
  "recommended_passed": true,
  "recommended_feedback": "Correct. Twelve is equivalent to 12.",
  "confidence": 0.96,
  "rubric_scores": [
    {
      "name": "correctness",
      "score": 1,
      "reason": "The answer is numerically correct."
    }
  ],
  "model": "configured-model",
  "prompt_version": "assignment-grader-v1",
  "raw_response": {}
}
```

HQ behavior:

- Validate service auth.
- Validate attempt exists and is active.
- Upsert/insert `assignment_ai_grade` by `request_id`.
- If `AI_AUTO_APPLY_GRADES` is enabled and result is completed, call the same shared grading command used by teachers.
- Mark attempt as AI graded and pending teacher review.
- If status is failed, store failure but leave attempt manually reviewable.

### AI Final Grade Command

Optional separate endpoint if we want to split recommendation storage from final grade application:

```text
POST /api/internal/ai/assignment-attempts/{attempt_id}/grade
```

This endpoint should call the same internal HQ grading command as the teacher endpoint.

Prefer starting with `ai-grade` doing both recommendation storage and auto-apply when configured, so the state transition is easy to reason about.

## AI Service API

These endpoints are called by HQ or future services. They should require a service secret from the caller.

Auth header:

```text
X-AI-Service-Secret: ...
```

### Health

```text
GET /healthz
```

Returns:

```text
ok
```

### Grade Assignment

```text
POST /internal/ai/grade-assignment
```

Request:

```json
{
  "request_id": "assignment-attempt-123:assignment-grader-v1",
  "attempt_id": 123
}
```

AI service behavior:

1. Validate caller service auth.
2. Fetch grading context from HQ.
3. Build the prompt for `assignment-grader-v1`.
4. Call OpenAI Responses API with a structured output schema.
5. Validate the structured output.
6. Post the completed or failed AI grade result back to HQ.
7. Return the final AI job status to HQ.

Response:

```json
{
  "request_id": "assignment-attempt-123:assignment-grader-v1",
  "status": "completed"
}
```

### Generate Assignment

Implement after grading.

```text
POST /internal/ai/generate-assignment
```

Request:

```json
{
  "request_id": "assignment-generation-abc",
  "category": "READING",
  "grade_level": "second-grade",
  "count": 5,
  "constraints": {
    "topic": "main idea",
    "difficulty": "easy"
  }
}
```

AI service behavior:

1. Validate caller.
2. Fetch any needed generation context from HQ.
3. Generate structured assignment drafts.
4. Post drafts to HQ.
5. Teacher reviews drafts before publishing.

## Shared HQ Grading Command

Refactor current teacher grading logic into one internal command:

```go
type GradeAttemptCommand struct {
	AssignmentID       int64
	AttemptID          int64
	Passed             bool
	Feedback           string
	GradedByType       string
	GradedByUserID     *int64
	GradedByService    string
	GradeSource        string
	AIGradeID          *int64
	AIReviewStatus     string
}
```

Callers:

- Teacher endpoint: `PATCH /api/assignments/:id/grade`
- AI internal endpoint: `POST /api/internal/ai/assignment-attempts/:attempt_id/ai-grade`
- Future teacher review endpoint

Command responsibilities:

- Validate the active attempt exists.
- Write `passed`, `feedback`, `date_graded`, and grader metadata.
- Award cookie only once when the final result is passing.
- Return the updated assignment/attempt state.

## Teacher Review UX Changes

Teacher assignment lists should distinguish:

- Needs human review: submitted but ungraded.
- AI graded pending review: already visible as graded to student, but teacher may inspect.
- Reviewed: teacher accepted AI grade.
- Overridden: teacher changed AI grade or feedback.
- Manual: teacher graded directly.

Potential filters:

```text
Needs Review
AI Graded
Overridden
All
Graded
```

Teacher actions:

- Accept AI grade.
- Override pass/fail.
- Edit feedback.
- Reset assignment for retry.

Student behavior:

- AI-graded attempts appear in the graded view.
- If a teacher later overrides the grade, the student sees the latest final grade and feedback.
- Attempt history remains preserved.

## Implementation Phases

### Phase 1: HQ Grading Foundation

1. Add `assignment_attempt` grader metadata columns.
2. Add `assignment_ai_grade`.
3. Refactor teacher grading into a shared HQ grading command.
4. Update existing teacher grade endpoint to use the shared command.
5. Add schoolwork workflow tests.
6. Add idempotent reward tests for manual and AI grade paths.

Exit criteria:

- Manual teacher grading still works.
- Existing reward behavior is unchanged.
- Tests cover submit, grade, reset, review, and reward idempotency.

### Phase 2: AI Service Skeleton

1. Add `cmd/ai`.
2. Add `/healthz`.
3. Add service-auth middleware.
4. Add shared `internal/aiapi` request/response structs.
5. Add Dockerfile and Compose service on `18083`.
6. Add AI service config validation.
7. Add fake/noop grader mode for development without an OpenAI key.

Exit criteria:

- AI service starts in Docker Compose.
- Health check returns `ok`.
- HQ and AI can authenticate to each other with service secrets.

### Phase 3: HQ Internal AI Endpoints

1. Add grading-context endpoint.
2. Add AI grade result endpoint.
3. Implement idempotent `request_id` handling.
4. Add auto-apply behavior behind `AI_AUTO_APPLY_GRADES`.
5. Add tests with service-authenticated requests.

Exit criteria:

- AI recommendation can be stored.
- Completed AI recommendation can auto-grade through the shared command.
- Failed AI recommendation leaves attempt manually reviewable.

### Phase 4: OpenAI Grader

1. Add OpenAI client to AI service.
2. Use Responses API.
3. Require structured output with fields:
   - `passed`
   - `feedback`
   - `confidence`
   - `rubric_scores`
4. Validate model response before posting to HQ.
5. Store model and prompt version in HQ.
6. Add timeout/retry/backoff behavior.
7. Add provider error handling.

Exit criteria:

- With `OPENAI_API_KEY` set, submitted assignments can be AI graded.
- Without `OPENAI_API_KEY`, fake/noop mode remains available for tests/dev.
- Invalid AI output becomes a failed AI grade result, not a final grade.

### Phase 5: Teacher Review UI

1. Show AI graded status in teacher workspace.
2. Add filter for AI-graded pending review.
3. Add accept AI grade action.
4. Add override action.
5. Show AI feedback/model/prompt metadata in a compact review panel.

Exit criteria:

- Teachers can review AI-graded work.
- Teacher override updates final grade and marks AI status overridden.
- Student graded view reflects the latest final grade.

### Phase 6: Assignment Generation

1. Add `assignment_draft`.
2. Add AI generation endpoint.
3. Add HQ draft creation endpoint.
4. Add teacher draft review/publish UI.
5. Publish creates normal `assignment` rows.

Exit criteria:

- AI can create draft questions.
- Teachers can edit/publish/discard drafts.
- Published assignments behave exactly like manually created assignments.

### Phase 7: Sunny Town AI Hooks

Do not start until grading/generation boundaries are stable.

Potential workflows:

- NPC tutor explanation for a missed assignment.
- NPC-generated practice question.
- Context-aware dialogue from approved game state.

Rules:

- Sunny Town validates interaction eligibility.
- AI service does not mutate live world state.
- HQ persists any durable schoolwork/tutor artifacts.

## Testing Plan

### HQ Tests

Add or update:

```text
cmd/hq/assignment_test.go
cmd/hq/ai_grading_test.go
cmd/hq/auth_handler_test.go
```

Required cases:

- Student submits answer and creates active attempt.
- Duplicate active submission is rejected.
- Teacher manual pass/fail uses shared grading command.
- Passing grade awards cookie once.
- Reset preserves history and makes assignment answerable.
- AI grade result is stored with `request_id`.
- Duplicate AI grade result is idempotent.
- AI completed result auto-applies grade when enabled.
- AI completed result does not auto-apply when disabled.
- AI failed result leaves attempt manually reviewable.
- Teacher accepts AI grade and marks reviewed.
- Teacher overrides AI grade and marks overridden.
- Internal AI endpoints reject missing/wrong service secret.

### AI Service Tests

Add:

```text
cmd/ai/grader_test.go
cmd/ai/http_test.go
```

Required cases:

- Health check returns `ok`.
- Grade endpoint rejects missing auth.
- Grade endpoint rejects missing `attempt_id`.
- Grade endpoint fetches context from fake HQ.
- Valid fake/OpenAI response posts completed grade to fake HQ.
- Invalid structured response posts failed grade to fake HQ.
- HQ fetch failure returns failed status.
- Provider timeout returns failed status.
- Request id is passed through unchanged.

### Integration Tests

Use Docker Compose:

```powershell
docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town ai
```

Health checks:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18083/healthz | Select-Object -ExpandProperty Content
```

Expected response for all is:

```text
ok
```

Manual smoke:

1. Log in as student.
2. Submit an assignment.
3. Confirm AI service grades it.
4. Confirm student graded view shows the AI grade.
5. Log in as teacher.
6. Filter AI-graded work.
7. Accept one AI grade.
8. Override one AI grade.
9. Confirm student view updates.

## Security And Safety

- Store `OPENAI_API_KEY` only in AI service environment.
- Never send OpenAI keys to browser, HQ frontend assets, Sunny Town, or logs.
- Use internal service secrets between HQ and AI.
- Log request ids, model, prompt version, latency, and status.
- Avoid logging full student answers in normal logs; keep detailed raw responses behind explicit debug/audit storage.
- Validate all structured AI outputs before applying grades.
- Keep grade auto-apply behind config.
- Make teacher override always possible.
- Keep prompt versions stable and explicit.

## Open Questions

- Should AI grading run synchronously during submit, or asynchronously after submit?
  - Recommended: start synchronous from HQ to AI if latency is acceptable; move to queued jobs if grading delays the student flow.
- Should all categories auto-grade, or only categories with strong rubrics?
  - Recommended: make this configurable by assignment/category.
- Should low-confidence AI grades stay ungraded until teacher review?
  - Recommended: yes. Add a confidence threshold before auto-apply.
- Should teacher edits feed future evals?
  - Recommended: store override metadata now so later evals can compare AI output to teacher final grades.

## Documentation Updates After Implementation

Update:

- `docs/USER_STORIES_AND_REQUIREMENTS.md`
- `docs/TEST_CASE_COVERAGE_MATRIX.md`
- `docs/PROJECT_SETUP.md`
- `docs/TESTING_GUIDELINES.md`
- `README.md`

Add:

- AI service runtime instructions.
- OpenAI key setup instructions.
- AI grading user stories.
- AI grading test cases.
- Teacher review/override acceptance criteria.
