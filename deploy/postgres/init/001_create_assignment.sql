create table if not exists assignment (
  id bigserial primary key,
  category text not null default 'MATH',
  prompt text not null,
  expected_answer text not null,
  created_at timestamptz not null default now(),
  constraint assignment_category_check check (category in ('MATH', 'SCIENCE', 'READING'))
);

create table if not exists assignment_attempt (
  id bigserial primary key,
  assignment_id bigint not null references assignment(id) on delete cascade,
  attempt_number integer not null,
  submitted_answer text not null,
  date_submitted timestamptz not null default now(),
  passed boolean null,
  feedback text null,
  date_graded timestamptz null,
  reset_at timestamptz null,
  constraint assignment_attempt_number_unique unique (assignment_id, attempt_number)
);

create index if not exists assignment_attempt_assignment_id_idx
  on assignment_attempt (assignment_id);

create index if not exists assignment_attempt_active_review_idx
  on assignment_attempt (assignment_id, date_submitted desc)
  where reset_at is null;
