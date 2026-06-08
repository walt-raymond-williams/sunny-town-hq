create table if not exists sunny_town_character (
  id bigserial primary key,
  character_type text not null,
  app_user_id bigint null unique references app_user(id) on delete cascade,
  room_id text not null default 'sunny-town-main',
  display_name text not null,
  avatar_id text not null default 'pet-default',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint sunny_town_character_type_check check (character_type in ('player', 'npc')),
  constraint sunny_town_character_owner_check check (
    (character_type = 'player' and app_user_id is not null) or
    (character_type = 'npc' and app_user_id is null)
  ),
  constraint sunny_town_character_room_id_check check (room_id <> ''),
  constraint sunny_town_character_display_name_check check (display_name <> ''),
  constraint sunny_town_character_avatar_id_check check (avatar_id <> '')
);

create index if not exists sunny_town_character_room_idx
  on sunny_town_character (room_id, character_type, id);

insert into sunny_town_character (
  character_type,
  app_user_id,
  room_id,
  display_name,
  avatar_id
)
select
  'player',
  u.id,
  'sunny-town-main',
  u.display_name,
  'pet-default'
from app_user u
join app_user_role ur on ur.user_id = u.id and ur.role = 'student'
on conflict (app_user_id) do update
set display_name = excluded.display_name,
  updated_at = now();
