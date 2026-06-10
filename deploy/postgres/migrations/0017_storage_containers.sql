create table if not exists storage_container (
  id text primary key,
  kind text not null,
  room_id text not null,
  map_id text not null,
  fixture_id text null,
  placed_object_id text null,
  shop_id text null,
  storage_role text null,
  location_id text null,
  owner_app_user_id bigint null references app_user(id) on delete cascade,
  slot_count integer not null,
  access_policy text not null,
  revision bigint not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint storage_container_kind_check check (kind in ('fixture', 'placed', 'shop')),
  constraint storage_container_storage_role_check check (storage_role is null or storage_role in ('input', 'output', 'general')),
  constraint storage_container_slot_count_check check (slot_count > 0 and slot_count <= 120),
  constraint storage_container_access_policy_check check (access_policy in ('shop_public_read', 'shop_input_deposit', 'owner_private', 'room_shared')),
  constraint storage_container_fixture_identity_check check (
    (kind <> 'fixture') or (fixture_id is not null and room_id <> '' and map_id <> '')
  ),
  constraint storage_container_placed_identity_check check (
    (kind <> 'placed') or placed_object_id is not null
  ),
  constraint storage_container_shop_identity_check check (
    (kind <> 'shop') or (shop_id is not null and storage_role is not null)
  )
);

create unique index if not exists storage_container_fixture_key
  on storage_container (room_id, map_id, fixture_id)
  where fixture_id is not null;

create unique index if not exists storage_container_placed_key
  on storage_container (placed_object_id)
  where placed_object_id is not null;

create unique index if not exists storage_container_shop_key
  on storage_container (shop_id, storage_role)
  where shop_id is not null and storage_role is not null;

create table if not exists storage_container_slot (
  container_id text not null references storage_container(id) on delete cascade,
  slot_index integer not null,
  item_type_id bigint null references inventory_item_type(id) on delete restrict,
  quantity integer null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (container_id, slot_index),
  constraint storage_container_slot_index_check check (slot_index >= 0),
  constraint storage_container_slot_quantity_check check (quantity is null or quantity > 0),
  constraint storage_container_slot_empty_or_occupied_check check (
    (item_type_id is null and quantity is null)
    or
    (item_type_id is not null and quantity is not null)
  )
);

insert into storage_container (
  id,
  kind,
  room_id,
  map_id,
  fixture_id,
  shop_id,
  storage_role,
  location_id,
  slot_count,
  access_policy
)
values
  (
    'fixture:sunny-town-main:sunny-town-house-1:cookie-shop-output-chest',
    'fixture',
    'sunny-town-main',
    'sunny-town-house-1',
    'cookie-shop-output-chest',
    'cookie-keeper-shop',
    'output',
    'cookie-shop',
    30,
    'shop_public_read'
  ),
  (
    'fixture:sunny-town-main:sunny-town-house-1:cookie-shop-input-chest',
    'fixture',
    'sunny-town-main',
    'sunny-town-house-1',
    'cookie-shop-input-chest',
    'cookie-keeper-shop',
    'input',
    'cookie-shop',
    30,
    'shop_input_deposit'
  )
on conflict (id) do update
set kind = excluded.kind,
  room_id = excluded.room_id,
  map_id = excluded.map_id,
  fixture_id = excluded.fixture_id,
  shop_id = excluded.shop_id,
  storage_role = excluded.storage_role,
  location_id = excluded.location_id,
  slot_count = excluded.slot_count,
  access_policy = excluded.access_policy,
  updated_at = now();
