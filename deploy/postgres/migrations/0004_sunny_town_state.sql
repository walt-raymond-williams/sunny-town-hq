create table if not exists student_sunny_town_position (
  app_user_id bigint primary key references app_user(id) on delete cascade,
  room_id text not null,
  map_id text not null,
  x double precision not null,
  y double precision not null,
  facing text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint student_sunny_town_position_facing_check check (facing in ('up', 'down', 'left', 'right'))
);

create index if not exists student_sunny_town_position_updated_at_idx
  on student_sunny_town_position (updated_at desc);

create table if not exists sunny_town_map_object (
  id bigserial primary key,
  room_id text not null,
  map_id text not null,
  grid_x integer not null,
  grid_y integer not null,
  item_key text not null references inventory_item_type(key) on delete restrict,
  placed_by_app_user_id bigint not null references app_user(id) on delete cascade,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint sunny_town_map_object_grid_nonnegative check (grid_x >= 0 and grid_y >= 0),
  constraint sunny_town_map_object_item_key_check check (item_key in ('stone_block')),
  constraint sunny_town_map_object_location_key unique (room_id, map_id, grid_x, grid_y)
);

create index if not exists sunny_town_map_object_map_idx
  on sunny_town_map_object (room_id, map_id, id);
