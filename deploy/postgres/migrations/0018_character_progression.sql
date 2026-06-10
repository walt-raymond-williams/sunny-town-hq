create table if not exists sunny_town_skill_definition (
  skill_key text primary key,
  display_name text not null,
  description text not null default '',
  xp_per_level integer not null default 100,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint sunny_town_skill_definition_key_check check (skill_key <> ''),
  constraint sunny_town_skill_definition_display_name_check check (display_name <> ''),
  constraint sunny_town_skill_definition_xp_per_level_check check (xp_per_level > 0)
);

create table if not exists sunny_town_character_skill (
  character_id bigint not null references sunny_town_character(id) on delete cascade,
  skill_key text not null references sunny_town_skill_definition(skill_key) on delete restrict,
  xp integer not null default 0,
  level integer not null default 1,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (character_id, skill_key),
  constraint sunny_town_character_skill_xp_check check (xp >= 0),
  constraint sunny_town_character_skill_level_check check (level >= 1)
);

create table if not exists sunny_town_character_skill_xp_ledger (
  id bigserial primary key,
  event_id text not null unique,
  character_id bigint not null references sunny_town_character(id) on delete cascade,
  source text not null,
  activity_key text not null,
  skill_key text not null references sunny_town_skill_definition(skill_key) on delete restrict,
  xp_amount integer not null,
  room_id text null,
  map_id text null,
  node_id text null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint sunny_town_character_skill_xp_event_id_check check (event_id <> ''),
  constraint sunny_town_character_skill_xp_source_check check (source <> ''),
  constraint sunny_town_character_skill_xp_activity_key_check check (activity_key <> ''),
  constraint sunny_town_character_skill_xp_amount_check check (xp_amount > 0)
);

create index if not exists sunny_town_character_skill_xp_ledger_character_idx
  on sunny_town_character_skill_xp_ledger (character_id, created_at desc);

insert into sunny_town_skill_definition (
  skill_key,
  display_name,
  description,
  xp_per_level
)
values (
  'mining',
  'Mining',
  'Breaking rocks, harvesting stone and crystal, and using pickaxes.',
  100
)
on conflict (skill_key) do update
set display_name = excluded.display_name,
  description = excluded.description,
  xp_per_level = excluded.xp_per_level,
  updated_at = now();
