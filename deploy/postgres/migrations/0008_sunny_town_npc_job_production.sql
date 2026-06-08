create table if not exists sunny_town_npc_job_production_ledger (
  id bigserial primary key,
  event_id text not null unique,
  character_id bigint not null references sunny_town_character(id) on delete cascade,
  room_id text not null,
  map_id text not null,
  npc_key text not null,
  job_key text not null,
  location_id text not null,
  output_key text not null,
  amount integer not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint sunny_town_npc_job_production_room_id_check check (room_id <> ''),
  constraint sunny_town_npc_job_production_map_id_check check (map_id <> ''),
  constraint sunny_town_npc_job_production_npc_key_check check (npc_key <> ''),
  constraint sunny_town_npc_job_production_job_key_check check (job_key <> ''),
  constraint sunny_town_npc_job_production_location_id_check check (location_id <> ''),
  constraint sunny_town_npc_job_production_output_key_check check (output_key <> ''),
  constraint sunny_town_npc_job_production_amount_positive check (amount > 0)
);

create index if not exists sunny_town_npc_job_production_character_idx
  on sunny_town_npc_job_production_ledger (character_id, created_at desc);

create index if not exists sunny_town_npc_job_production_job_idx
  on sunny_town_npc_job_production_ledger (room_id, job_key, created_at desc);
