create table tb_media_sync_runs (
  id uuid primary key,
  connector_id text not null,
  status text not null,
  started_at timestamptz not null,
  finished_at timestamptz,
  error_message text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_media_sync_runs_status_check check (status in ('running', 'completed', 'failed'))
);

create unique index uq_tb_media_sync_runs_connector_running
  on tb_media_sync_runs(connector_id)
  where status = 'running';
create index idx_tb_media_sync_runs_connector_started
  on tb_media_sync_runs(connector_id, started_at desc);

alter table tb_import_jobs
  add column sync_run_id uuid references tb_media_sync_runs(id);
create index idx_tb_import_jobs_sync_run on tb_import_jobs(sync_run_id, created_at desc);
