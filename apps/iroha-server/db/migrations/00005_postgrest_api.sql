-- +goose Up
create schema if not exists api;

-- +goose StatementBegin
do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'web_anon') then
    create role web_anon nologin;
  end if;
end
$$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'authenticator') then
    create role authenticator noinherit login password 'iroha_dev';
  end if;
end
$$;
-- +goose StatementEnd

grant web_anon to authenticator;

create or replace view api.activities as
select
  id,
  sport_type,
  title,
  started_at,
  ended_at,
  timezone,
  distance_m,
  duration_s,
  moving_time_s,
  elevation_gain_m,
  avg_hr,
  max_hr,
  avg_pace_s_per_km
from tb_activities;

create or replace view api.public_activities as
select
  'act_' || id::text as id,
  sport_type,
  title,
  started_at,
  ended_at,
  timezone,
  distance_m,
  duration_s,
  moving_time_s,
  elevation_gain_m,
  avg_hr,
  max_hr,
  avg_pace_s_per_km
from tb_activities;

-- +goose StatementBegin
create or replace function api.public_summary()
returns jsonb
language sql
stable
security definer
set search_path = public, pg_temp
as $$
with totals as (
  select
    count(*)::integer as activity_count,
    coalesce(sum(distance_m), 0)::double precision as distance_m,
    coalesce(sum(duration_s), 0)::integer as duration_s,
    coalesce(sum(moving_time_s), 0)::integer as moving_time_s
  from tb_activities
),
by_year as (
  select
    extract(year from started_at)::text as key,
    count(*)::integer as activity_count,
    coalesce(sum(distance_m), 0)::double precision as distance_m,
    coalesce(sum(duration_s), 0)::integer as duration_s,
    coalesce(sum(moving_time_s), 0)::integer as moving_time_s
  from tb_activities
  group by key
  order by key desc
),
by_month as (
  select
    to_char(started_at, 'YYYY-MM') as key,
    count(*)::integer as activity_count,
    coalesce(sum(distance_m), 0)::double precision as distance_m,
    coalesce(sum(duration_s), 0)::integer as duration_s,
    coalesce(sum(moving_time_s), 0)::integer as moving_time_s
  from tb_activities
  group by key
  order by key desc
),
by_sport as (
  select
    sport_type as key,
    count(*)::integer as activity_count,
    coalesce(sum(distance_m), 0)::double precision as distance_m,
    coalesce(sum(duration_s), 0)::integer as duration_s,
    coalesce(sum(moving_time_s), 0)::integer as moving_time_s
  from tb_activities
  group by sport_type
  order by activity_count desc
)
select jsonb_build_object(
  'totals', (select to_jsonb(totals) from totals),
  'by_year', coalesce((select jsonb_agg(to_jsonb(by_year)) from by_year), '[]'::jsonb),
  'by_month', coalesce((select jsonb_agg(to_jsonb(by_month)) from by_month), '[]'::jsonb),
  'by_sport', coalesce((select jsonb_agg(to_jsonb(by_sport)) from by_sport), '[]'::jsonb)
);
$$;
-- +goose StatementEnd

grant usage on schema api to web_anon;
grant select on api.activities to web_anon;
grant select on api.public_activities to web_anon;
grant execute on function api.public_summary() to web_anon;

-- +goose Down
drop function if exists api.public_summary();
drop view if exists api.public_activities;
drop view if exists api.activities;

drop schema if exists api cascade;

revoke web_anon from authenticator;

drop role if exists authenticator;
drop role if exists web_anon;
