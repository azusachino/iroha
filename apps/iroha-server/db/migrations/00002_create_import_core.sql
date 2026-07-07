-- +goose Up
create table raw_files (
  id uuid primary key,
  sha256 text not null unique,
  original_filename text not null,
  content_type text not null default '',
  size_bytes bigint not null,
  storage_path text not null,
  source_kind text not null,
  uploaded_via text not null,
  created_at timestamptz not null
);

create index idx_raw_files_created_at on raw_files(created_at desc);
create index idx_raw_files_source_created_at on raw_files(source_kind, created_at desc);

create table import_jobs (
  id uuid primary key,
  raw_file_id uuid not null references raw_files(id),
  status text not null,
  parser_kind text not null,
  parser_version text not null,
  error_message text,
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz not null,
  constraint import_jobs_status_check check (status in ('queued', 'parsing', 'completed', 'failed'))
);

create index idx_import_jobs_status on import_jobs(status);
create index idx_import_jobs_raw_file_created_at on import_jobs(raw_file_id, created_at desc);

-- +goose Down
drop table if exists import_jobs;
drop table if exists raw_files;
