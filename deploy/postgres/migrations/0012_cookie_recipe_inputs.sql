insert into inventory_item_type (key, name, description)
values
  ('flour', 'Flour', 'A basic baking ingredient.'),
  ('sugar', 'Sugar', 'A sweet baking ingredient.')
on conflict (key) do nothing;
