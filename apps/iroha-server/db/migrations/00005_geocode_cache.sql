-- +goose Up

create table tb_geocode_cache (
  coordinate_key text primary key,
  latitude double precision not null,
  longitude double precision not null,
  provider text not null,
  city text not null,
  response_json jsonb not null,
  fetched_at timestamptz not null,
  expires_at timestamptz not null,
  refresh_queued_at timestamptz,
  last_error text,
  created_at timestamptz not null,
  updated_at timestamptz not null
);
create index idx_tb_geocode_cache_expiry on tb_geocode_cache(expires_at);
create index idx_tb_geocode_cache_refresh_queued on tb_geocode_cache(refresh_queued_at);

-- +goose Down

drop table if exists tb_geocode_cache;
