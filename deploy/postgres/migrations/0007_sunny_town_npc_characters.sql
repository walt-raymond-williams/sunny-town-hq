create table if not exists sunny_town_npc_character (
  character_id bigint primary key references sunny_town_character(id) on delete cascade,
  room_id text not null,
  npc_key text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint sunny_town_npc_character_room_id_check check (room_id <> ''),
  constraint sunny_town_npc_character_npc_key_check check (npc_key <> ''),
  constraint sunny_town_npc_character_room_key unique (room_id, npc_key)
);

create index if not exists sunny_town_npc_character_room_idx
  on sunny_town_npc_character (room_id, npc_key);

with character_row as (
  insert into sunny_town_character (
    character_type,
    app_user_id,
    room_id,
    display_name,
    avatar_id
  )
  select
    'npc',
    null,
    'sunny-town-main',
    'Mayor Sunny',
    'mayor'
  where not exists (
    select 1
    from sunny_town_npc_character
    where room_id = 'sunny-town-main'
      and npc_key = 'mayor-sunny'
  )
  returning id
)
insert into sunny_town_npc_character (
  character_id,
  room_id,
  npc_key
)
select
  id,
  'sunny-town-main',
  'mayor-sunny'
from character_row
on conflict (room_id, npc_key) do nothing;
