with item_type as (
  select id
  from inventory_item_type
  where key = 'cookie'
),
inserted as (
  insert into shop_stock_ledger (
    event_id,
    source,
    shop_id,
    item_type_id,
    delta
  )
  select
    'shop-stock-seed:cookie-keeper-shop:cookie:v1',
    'seed',
    'cookie-keeper-shop',
    id,
    5
  from item_type
  on conflict (event_id) do nothing
  returning shop_id, item_type_id, delta
)
insert into shop_stock_item (shop_id, item_type_id, quantity)
select shop_id, item_type_id, delta
from inserted
on conflict (shop_id, item_type_id) do update
set quantity = shop_stock_item.quantity + excluded.quantity,
  updated_at = now();
