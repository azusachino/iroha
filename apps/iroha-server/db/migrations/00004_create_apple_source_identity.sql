-- +goose Up
create table tb_import_snapshots (
  id uuid primary key,
  import_job_id uuid not null references tb_import_jobs(id),
  raw_file_id uuid not null references tb_raw_files(id),
  sha256 text not null,
  parser_version text not null,
  taken_at timestamptz,
  created_at timestamptz not null
);

create index idx_tb_import_snapshots_raw_file_created_at on tb_import_snapshots(raw_file_id, created_at desc);

create table tb_apple_source_items (
  id uuid primary key,
  source_key text not null,
  item_type text not null,
  content_hash text not null,
  activity_id uuid references tb_activities(id) on delete set null,
  last_seen_snapshot_id uuid references tb_import_snapshots(id),
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_apple_source_items_item_type_check check (item_type in ('workout', 'route', 'record')),
  unique(source_key)
);

create index idx_tb_apple_source_items_item_type on tb_apple_source_items(item_type);
create index idx_tb_apple_source_items_last_seen_snapshot_id on tb_apple_source_items(last_seen_snapshot_id);

-- +goose Down
drop table if exists tb_apple_source_items;
drop table if exists tb_import_snapshots;
