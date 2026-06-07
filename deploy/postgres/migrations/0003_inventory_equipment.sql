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

create index if not exists student_star_ledger_app_user_id_idx
  on student_star_ledger (app_user_id, created_at desc);

create table if not exists inventory_item_type (
  id bigserial primary key,
  key text not null unique,
  name text not null,
  description text not null default '',
  equip_slot text null,
  visual_key text null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint inventory_item_type_key_check check (key ~ '^[a-z][a-z0-9_]*$'),
  constraint inventory_item_type_equip_slot_check check (equip_slot is null or equip_slot in ('gear', 'accessory', 'tool'))
);

alter table inventory_item_type add column if not exists equip_slot text;
alter table inventory_item_type add column if not exists visual_key text;
alter table inventory_item_type drop constraint if exists inventory_item_type_equip_slot_check;

alter table inventory_item_type
  add constraint inventory_item_type_equip_slot_check
  check (equip_slot is null or equip_slot in ('gear', 'accessory', 'tool'));

insert into inventory_item_type (key, name, description)
values ('cookie', 'Cookie', 'A treat for your pet.')
on conflict (key) do update
set name = excluded.name,
  description = excluded.description,
  equip_slot = null,
  visual_key = null,
  updated_at = now();

insert into inventory_item_type (key, name, description, equip_slot, visual_key)
values
  ('sunny_hoodie', 'Sunny Hoodie', 'A cozy hoodie for Sunny Town.', 'gear', 'sunny_hoodie'),
  ('star_cap', 'Star Cap', 'A bright cap for sunny adventures.', 'accessory', 'star_cap'),
  ('pickaxe', 'Pickaxe', 'A sturdy starter tool.', 'tool', 'pickaxe')
on conflict (key) do update
set name = excluded.name,
  description = excluded.description,
  equip_slot = excluded.equip_slot,
  visual_key = excluded.visual_key,
  updated_at = now();

insert into inventory_item_type (key, name, description)
values
  ('rock', 'Rock', 'A sturdy rock from Forest Crossing.'),
  ('crystal', 'Crystal', 'A bright crystal from Forest Crossing.'),
  ('stone_block', 'Stone Block', 'A solid block crafted from stone.')
on conflict (key) do update
set name = excluded.name,
  description = excluded.description,
  equip_slot = null,
  visual_key = null,
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

create table if not exists student_inventory_ledger (
  id bigserial primary key,
  app_user_id bigint not null references app_user(id) on delete cascade,
  event_id text not null unique,
  source text not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  delta integer not null,
  room_id text null,
  map_id text null,
  node_id text null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint student_inventory_ledger_delta_nonzero check (delta <> 0)
);

create index if not exists student_inventory_ledger_app_user_id_idx
  on student_inventory_ledger (app_user_id, created_at desc);

insert into student_inventory_item (app_user_id, item_type_id, quantity)
select u.id, iit.id, u.cookies
from app_user u
cross join inventory_item_type iit
where iit.key = 'cookie'
  and u.cookies > 0
on conflict (app_user_id, item_type_id) do update
set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
  updated_at = now();

update app_user set cookies = 0 where cookies > 0;

insert into student_inventory_item (app_user_id, item_type_id, quantity)
select u.id, iit.id, 1
from app_user u
join app_user_role ur on ur.user_id = u.id and ur.role = 'student'
cross join inventory_item_type iit
where iit.key in ('sunny_hoodie', 'star_cap', 'pickaxe')
on conflict (app_user_id, item_type_id) do update
set quantity = greatest(student_inventory_item.quantity, excluded.quantity),
  updated_at = now();

create table if not exists student_equipped_item (
  app_user_id bigint not null references app_user(id) on delete cascade,
  slot text not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (app_user_id, slot),
  constraint student_equipped_item_slot_check check (slot in ('gear', 'accessory', 'tool'))
);

alter table student_equipped_item drop constraint if exists student_equipped_item_slot_check;

alter table student_equipped_item
  add constraint student_equipped_item_slot_check
  check (slot in ('gear', 'accessory', 'tool'));

create index if not exists student_equipped_item_app_user_id_idx
  on student_equipped_item (app_user_id);

create table if not exists student_hotbar_slot (
  app_user_id bigint not null references app_user(id) on delete cascade,
  slot_index integer not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (app_user_id, slot_index),
  constraint student_hotbar_slot_index_check check (slot_index between 1 and 5)
);

create index if not exists student_hotbar_slot_app_user_id_idx
  on student_hotbar_slot (app_user_id);
