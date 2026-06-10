create table if not exists student_inventory_slot (
  app_user_id bigint not null references app_user(id) on delete cascade,
  slot_index integer not null,
  item_type_id bigint null references inventory_item_type(id) on delete restrict,
  quantity integer null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (app_user_id, slot_index),
  constraint student_inventory_slot_index_check check (slot_index >= 0 and slot_index < 30),
  constraint student_inventory_slot_quantity_check check (quantity is null or quantity > 0),
  constraint student_inventory_slot_empty_or_occupied_check check (
    (item_type_id is null and quantity is null)
    or
    (item_type_id is not null and quantity is not null)
  )
);

create index if not exists student_inventory_slot_app_user_id_idx
  on student_inventory_slot (app_user_id, slot_index);

create index if not exists student_inventory_slot_item_type_id_idx
  on student_inventory_slot (app_user_id, item_type_id)
  where item_type_id is not null;

do $$
declare
  overflowing_students integer;
begin
  with expanded as (
    select
      sii.app_user_id,
      row_number() over (
        partition by sii.app_user_id
        order by iit.id, stack_parts.stack_index
      ) - 1 as slot_index
    from student_inventory_item sii
    join inventory_item_type iit on iit.id = sii.item_type_id
    cross join lateral generate_series(
      0,
      greatest(ceil(sii.quantity::numeric / greatest(coalesce(iit.max_stack, sii.quantity), 1))::integer - 1, 0)
    ) as stack_parts(stack_index)
    where sii.quantity > 0
  )
  select count(distinct app_user_id)
  into overflowing_students
  from expanded
  where slot_index >= 30;

  if overflowing_students > 0 then
    raise exception 'student_inventory_slot backfill would exceed 30 slots for % student(s)', overflowing_students;
  end if;
end $$;

with expanded as (
  select
    sii.app_user_id,
    iit.id as item_type_id,
    row_number() over (
      partition by sii.app_user_id
      order by iit.id, stack_parts.stack_index
    ) - 1 as slot_index,
    least(
      greatest(coalesce(iit.max_stack, sii.quantity), 1),
      sii.quantity - (stack_parts.stack_index * greatest(coalesce(iit.max_stack, sii.quantity), 1))
    )::integer as quantity
  from student_inventory_item sii
  join inventory_item_type iit on iit.id = sii.item_type_id
  cross join lateral generate_series(
    0,
    greatest(ceil(sii.quantity::numeric / greatest(coalesce(iit.max_stack, sii.quantity), 1))::integer - 1, 0)
  ) as stack_parts(stack_index)
  where sii.quantity > 0
)
insert into student_inventory_slot (app_user_id, slot_index, item_type_id, quantity)
select app_user_id, slot_index, item_type_id, quantity
from expanded
where slot_index < 30
on conflict (app_user_id, slot_index) do update
set item_type_id = excluded.item_type_id,
  quantity = excluded.quantity,
  updated_at = now();
