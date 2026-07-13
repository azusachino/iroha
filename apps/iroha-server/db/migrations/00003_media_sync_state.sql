-- +goose Up

create table tb_media_sync_state (
  id uuid primary key,
  connector_id text not null unique,
  cursor_json jsonb not null default '{}'::jsonb,
  status text not null,
  last_error text,
  last_fetched_at timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create index idx_tb_media_sync_state_status on tb_media_sync_state(status);

-- +goose Down

drop table if exists tb_media_sync_state;
