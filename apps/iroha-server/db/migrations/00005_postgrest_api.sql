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

grant usage on schema api to web_anon;
grant select on api.activities to web_anon;

-- +goose Down
drop view if exists api.activities;

drop schema if exists api cascade;

revoke web_anon from authenticator;

drop role if exists authenticator;
drop role if exists web_anon;
