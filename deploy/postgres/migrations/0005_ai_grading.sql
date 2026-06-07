create table if not exists assignment_ai_grade (
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
  completed_at timestamptz null,
  constraint assignment_ai_grade_status_check check (status in ('pending', 'completed', 'failed'))
);

create index if not exists assignment_ai_grade_attempt_idx
  on assignment_ai_grade (assignment_attempt_id, created_at desc);

alter table assignment_attempt add column if not exists graded_by_type text;
alter table assignment_attempt add column if not exists graded_by_user_id bigint;
alter table assignment_attempt add column if not exists graded_by_service text;
alter table assignment_attempt add column if not exists grade_source text;
alter table assignment_attempt add column if not exists ai_review_status text;
alter table assignment_attempt add column if not exists ai_grade_id bigint;

alter table assignment_attempt drop constraint if exists assignment_attempt_graded_by_type_check;

alter table assignment_attempt
  add constraint assignment_attempt_graded_by_type_check
  check (graded_by_type is null or graded_by_type in ('teacher', 'ai'));

alter table assignment_attempt drop constraint if exists assignment_attempt_grade_source_check;

alter table assignment_attempt
  add constraint assignment_attempt_grade_source_check
  check (grade_source is null or grade_source in ('manual', 'ai_auto', 'ai_override', 'teacher_override'));

alter table assignment_attempt drop constraint if exists assignment_attempt_ai_review_status_check;

alter table assignment_attempt
  add constraint assignment_attempt_ai_review_status_check
  check (ai_review_status is null or ai_review_status in ('pending_review', 'reviewed', 'overridden'));

do $$
begin
  if not exists (
    select 1 from pg_constraint where conname = 'assignment_attempt_graded_by_user_id_fkey'
  ) then
    alter table assignment_attempt
      add constraint assignment_attempt_graded_by_user_id_fkey
      foreign key (graded_by_user_id) references app_user(id) on delete set null;
  end if;
  if not exists (
    select 1 from pg_constraint where conname = 'assignment_attempt_ai_grade_id_fkey'
  ) then
    alter table assignment_attempt
      add constraint assignment_attempt_ai_grade_id_fkey
      foreign key (ai_grade_id) references assignment_ai_grade(id) on delete set null;
  end if;
end $$;
