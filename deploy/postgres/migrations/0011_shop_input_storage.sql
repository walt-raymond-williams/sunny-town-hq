create table if not exists shop_input_storage_item (
  shop_id text not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  quantity integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (shop_id, item_type_id),
  constraint shop_input_storage_item_shop_id_check check (shop_id <> ''),
  constraint shop_input_storage_item_quantity_nonnegative check (quantity >= 0)
);
