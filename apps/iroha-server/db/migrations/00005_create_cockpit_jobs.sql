-- +goose Up
create table tb_intake_payloads (
  id uuid primary key,
  source_kind text not null,
  source_actor text not null,
  source_event_id text not null default '',
  content_type text not null default '',
  sha256 text not null,
  size_bytes bigint not null default 0,
  storage_path text not null default '',
  payload_json jsonb,
  received_at timestamptz not null,
  parsed_at timestamptz,
  created_at timestamptz not null,
  constraint tb_intake_payloads_source_actor_check check (source_actor in ('user', 'connector', 'enrichment', 'system')),
  constraint tb_intake_payloads_payload_or_storage_check check (storage_path <> '' or payload_json is not null)
);

create index idx_tb_intake_payloads_source_received_at
  on tb_intake_payloads(source_kind, received_at desc);
create unique index idx_tb_intake_payloads_source_event_id_unique
  on tb_intake_payloads(source_kind, source_event_id)
  where source_event_id <> '';

create table tb_jobs (
  id uuid primary key,
  kind text not null,
  status text not null,
  priority integer not null default 0,
  payload_json jsonb not null default '{}'::jsonb,
  attempts integer not null default 0,
  max_attempts integer not null default 3,
  run_after timestamptz not null,
  locked_by text,
  locked_at timestamptz,
  error_message text,
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_jobs_status_check check (status in ('queued', 'running', 'completed', 'failed', 'canceled')),
  constraint tb_jobs_attempts_check check (attempts >= 0 and max_attempts > 0)
);

create index idx_tb_jobs_claimable
  on tb_jobs(status, run_after, priority desc, created_at)
  where status = 'queued';
create index idx_tb_jobs_kind_created_at on tb_jobs(kind, created_at desc);
create index idx_tb_jobs_status_updated_at on tb_jobs(status, updated_at desc);

create table tb_job_schedules (
  id uuid primary key,
  kind text not null,
  enabled boolean not null default true,
  schedule_kind text not null,
  schedule_expr text not null,
  payload_json jsonb not null default '{}'::jsonb,
  next_run_at timestamptz,
  last_run_at timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_job_schedules_schedule_kind_check check (schedule_kind in ('interval', 'manual'))
);

create index idx_tb_job_schedules_due
  on tb_job_schedules(enabled, next_run_at)
  where enabled = true and next_run_at is not null;

-- +goose Down
drop table if exists tb_job_schedules;
drop table if exists tb_jobs;
drop table if exists tb_intake_payloads;
