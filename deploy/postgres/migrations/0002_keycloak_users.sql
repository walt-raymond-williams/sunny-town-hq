alter table app_user add column if not exists keycloak_subject text;
alter table app_user add column if not exists email text;

update app_user
set keycloak_subject = 'legacy-user-' || id
where keycloak_subject is null;

alter table app_user alter column keycloak_subject set not null;

create unique index if not exists app_user_keycloak_subject_key
  on app_user (keycloak_subject);

create table if not exists app_user_role (
  user_id bigint not null references app_user(id) on delete cascade,
  role text not null,
  primary key (user_id, role),
  constraint app_user_role_role_check check (role in ('student', 'teacher'))
);

insert into app_user_role (user_id, role)
select id, 'student' from app_user
on conflict do nothing;

alter table assignment_attempt add column if not exists student_user_id bigint;

update assignment_attempt
set student_user_id = 1
where student_user_id is null;

alter table assignment_attempt alter column student_user_id set not null;

do $$
begin
  if not exists (
    select 1 from pg_constraint where conname = 'assignment_attempt_student_user_id_fkey'
  ) then
    alter table assignment_attempt
      add constraint assignment_attempt_student_user_id_fkey
      foreign key (student_user_id) references app_user(id) on delete cascade;
  end if;
end $$;

alter table assignment_attempt drop constraint if exists assignment_attempt_number_unique;

alter table assignment_attempt
  add constraint assignment_attempt_number_unique unique (assignment_id, student_user_id, attempt_number);

create index if not exists assignment_attempt_student_user_id_idx
  on assignment_attempt (student_user_id);

drop index if exists assignment_attempt_active_review_idx;

create index if not exists assignment_attempt_active_review_idx
  on assignment_attempt (assignment_id, student_user_id, date_submitted desc)
  where reset_at is null;
