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
  student_user_id bigint not null,
  attempt_number integer not null,
  submitted_answer text not null,
  date_submitted timestamptz not null default now(),
  passed boolean null,
  feedback text null,
  date_graded timestamptz null,
  cookie_awarded boolean not null default false,
  reset_at timestamptz null,
  constraint assignment_attempt_number_unique unique (assignment_id, student_user_id, attempt_number)
);

create table if not exists app_user (
  id bigserial primary key,
  keycloak_subject text not null unique,
  display_name text not null,
  email text null,
  cookies integer not null default 0,
  constraint app_user_cookies_nonnegative check (cookies >= 0)
);

alter table assignment_attempt
  add constraint assignment_attempt_student_user_id_fkey
  foreign key (student_user_id) references app_user(id) on delete cascade;

create table if not exists app_user_role (
  user_id bigint not null references app_user(id) on delete cascade,
  role text not null,
  primary key (user_id, role),
  constraint app_user_role_role_check check (role in ('student', 'teacher'))
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

create table if not exists student_wallet (
  app_user_id bigint primary key references app_user(id) on delete cascade,
  star_balance integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint student_wallet_star_balance_nonnegative check (star_balance >= 0)
);

create table if not exists student_star_ledger (
  id bigserial primary key,
  app_user_id bigint not null references app_user(id) on delete cascade,
  event_id text not null unique,
  source text not null,
  delta integer not null,
  room_id text null,
  map_id text null,
  collectible_id text null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint student_star_ledger_delta_nonzero check (delta <> 0)
);

create index if not exists assignment_attempt_assignment_id_idx
  on assignment_attempt (assignment_id);

create index if not exists assignment_attempt_student_user_id_idx
  on assignment_attempt (student_user_id);

create index if not exists assignment_attempt_active_review_idx
  on assignment_attempt (assignment_id, student_user_id, date_submitted desc)
  where reset_at is null;

create index if not exists student_star_ledger_app_user_id_idx
  on student_star_ledger (app_user_id, created_at desc);

create table if not exists inventory_item_type (
  id bigserial primary key,
  key text not null unique,
  name text not null,
  description text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint inventory_item_type_key_check check (key ~ '^[a-z][a-z0-9_]*$')
);

insert into inventory_item_type (key, name, description)
values ('cookie', 'Cookie', 'A treat for your pet.')
on conflict (key) do update
set name = excluded.name,
  description = excluded.description,
  updated_at = now();

create table if not exists student_inventory_item (
  app_user_id bigint not null references app_user(id) on delete cascade,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  quantity integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (app_user_id, item_type_id),
  constraint student_inventory_item_quantity_nonnegative check (quantity >= 0)
);
