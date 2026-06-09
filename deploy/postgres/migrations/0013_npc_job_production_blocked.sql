create table if not exists sunny_town_npc_job_production_blocked_ledger (
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
  reason text not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint sunny_town_npc_job_production_blocked_reason_check check (reason <> '')
);

create index if not exists sunny_town_npc_job_production_blocked_room_idx
  on sunny_town_npc_job_production_blocked_ledger (room_id, created_at desc);
