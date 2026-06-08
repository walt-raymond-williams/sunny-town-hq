create table if not exists shop_stock_item (
  shop_id text not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  quantity integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (shop_id, item_type_id),
  constraint shop_stock_item_shop_id_check check (shop_id <> ''),
  constraint shop_stock_item_quantity_nonnegative check (quantity >= 0)
);

create table if not exists shop_stock_ledger (
  id bigserial primary key,
  event_id text not null unique,
  source text not null,
  shop_id text not null,
  item_type_id bigint not null references inventory_item_type(id) on delete restrict,
  delta integer not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  constraint shop_stock_ledger_source_check check (source <> ''),
  constraint shop_stock_ledger_shop_id_check check (shop_id <> ''),
  constraint shop_stock_ledger_delta_nonzero check (delta <> 0)
);

create index if not exists shop_stock_ledger_shop_idx
  on shop_stock_ledger (shop_id, created_at desc);
