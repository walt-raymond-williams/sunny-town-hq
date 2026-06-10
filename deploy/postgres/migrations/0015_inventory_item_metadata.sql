alter table inventory_item_type add column if not exists icon_key text;
alter table inventory_item_type add column if not exists max_stack integer;
alter table inventory_item_type add column if not exists category text;

alter table inventory_item_type drop constraint if exists inventory_item_type_icon_key_check;
alter table inventory_item_type drop constraint if exists inventory_item_type_max_stack_check;
alter table inventory_item_type drop constraint if exists inventory_item_type_category_check;

alter table inventory_item_type
  add constraint inventory_item_type_icon_key_check
  check (icon_key is null or icon_key ~ '^[a-z][a-z0-9_]*$');

alter table inventory_item_type
  add constraint inventory_item_type_max_stack_check
  check (max_stack is null or max_stack > 0);

alter table inventory_item_type
  add constraint inventory_item_type_category_check
  check (category is null or category in ('consumable', 'gear', 'tool', 'resource', 'building'));

update inventory_item_type
set icon_key = key,
  max_stack = case
    when key in ('sunny_hoodie', 'star_cap', 'pickaxe') then 1
    else 64
  end,
  category = case
    when key = 'cookie' then 'consumable'
    when key in ('sunny_hoodie', 'star_cap') then 'gear'
    when key = 'pickaxe' then 'tool'
    when key in ('rock', 'crystal', 'flour', 'sugar') then 'resource'
    when key = 'stone_block' then 'building'
    else category
  end,
  updated_at = now()
where key in ('cookie', 'sunny_hoodie', 'star_cap', 'pickaxe', 'rock', 'crystal', 'stone_block', 'flour', 'sugar');
