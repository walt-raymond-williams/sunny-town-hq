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
  cookie_awarded boolean not null default false,
  reset_at timestamptz null,
  constraint assignment_attempt_number_unique unique (assignment_id, attempt_number)
);

create table if not exists app_user (
  id bigserial primary key,
  display_name text not null,
  cookies integer not null default 0,
  constraint app_user_cookies_nonnegative check (cookies >= 0)
);

create table if not exists pet_state (
  id bigserial primary key,
  user_id bigint not null references app_user(id) on delete cascade,
  hunger integer not null default 50,
  happiness integer not null default 50,
  energy integer not null default 50,
  sleeping boolean not null default false,
  sleep_started_at timestamptz null,
  sleep_started_energy integer null,
  updated_at timestamptz not null default now(),
  last_decay_at timestamptz not null default now(),
  constraint pet_state_user_unique unique (user_id),
  constraint pet_state_hunger_range check (hunger between 0 and 100),
  constraint pet_state_happiness_range check (happiness between 0 and 100),
  constraint pet_state_energy_range check (energy between 0 and 100),
  constraint pet_state_sleep_started_energy_range check (
    sleep_started_energy is null or sleep_started_energy between 0 and 100
  )
);

insert into app_user (id, display_name, cookies)
values (1, 'Student', 0)
on conflict (id) do nothing;

select setval(pg_get_serial_sequence('app_user', 'id'), 1, true);

insert into pet_state (user_id, hunger, happiness, energy)
values (1, 50, 50, 50)
on conflict (user_id) do nothing;

create index if not exists assignment_attempt_assignment_id_idx
  on assignment_attempt (assignment_id);

create index if not exists assignment_attempt_active_review_idx
  on assignment_attempt (assignment_id, date_submitted desc)
  where reset_at is null;
