create table if not exists assignment (
  id bigserial primary key,
  category text not null default 'MATH',
  prompt text not null,
  expected_answer text not null,
  submitted_answer text null,
  date_submitted timestamptz not null default now(),
  passed boolean null,
  feedback text null,
  date_graded timestamptz null,
  constraint assignment_category_check check (category in ('MATH', 'SCIENCE', 'READING'))
);
