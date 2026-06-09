insert into shop_input_storage_item (shop_id, item_type_id, quantity)
select 'cookie-keeper-shop', id, 16
from inventory_item_type
where key in ('flour', 'sugar')
on conflict (shop_id, item_type_id) do update
set quantity = greatest(shop_input_storage_item.quantity, excluded.quantity),
  updated_at = now();
