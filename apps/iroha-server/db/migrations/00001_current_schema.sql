-- +goose Up

create extension if not exists postgis;

create table tb_raw_files (
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
create index idx_tb_raw_files_created_at on tb_raw_files(created_at desc);
create index idx_tb_raw_files_source_created_at on tb_raw_files(source_kind, created_at desc);

create table tb_import_jobs (
  id uuid primary key,
  raw_file_id uuid not null references tb_raw_files(id),
  status text not null,
  parser_kind text not null,
  parser_version text not null,
  error_message text,
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz not null,
  constraint tb_import_jobs_status_check check (status in ('queued', 'parsing', 'completed', 'failed'))
);
create index idx_tb_import_jobs_status on tb_import_jobs(status);
create index idx_tb_import_jobs_raw_file_created_at on tb_import_jobs(raw_file_id, created_at desc);

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

create table tb_activities (
  id uuid primary key,
  sport_type text not null,
  title text not null default '',
  started_at timestamptz not null,
  ended_at timestamptz,
  timezone text not null default '',
  distance_m numeric,
  duration_s integer,
  moving_time_s integer,
  elevation_gain_m numeric,
  avg_hr integer,
  max_hr integer,
  avg_pace_s_per_km numeric,
  calories_kcal numeric,
  source_kind text not null,
  source_activity_id text not null default '',
  first_raw_file_id uuid not null references tb_raw_files(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);
create index idx_tb_activities_started_at on tb_activities(started_at desc);
create index idx_tb_activities_sport_started on tb_activities(sport_type, started_at desc);

create table tb_external_refs (
  id uuid primary key,
  activity_id uuid not null references tb_activities(id) on delete cascade,
  provider text not null,
  external_id text not null,
  raw_file_id uuid not null references tb_raw_files(id),
  created_at timestamptz not null,
  unique(provider, external_id)
);

create table tb_activity_route_points (
  activity_id uuid not null references tb_activities(id) on delete cascade,
  seq integer not null,
  ts timestamptz,
  lat double precision not null,
  lon double precision not null,
  elevation_m numeric,
  distance_m numeric,
  speed_mps numeric,
  heart_rate integer,
  geom geography(Point, 4326) not null,
  primary key (activity_id, seq)
);
create index idx_tb_route_points_geom on tb_activity_route_points using gist(geom);

create table tb_activity_samplings (
  id uuid primary key,
  activity_id uuid not null references tb_activities(id) on delete cascade,
  sampling_type text not null,
  ts timestamptz not null,
  value numeric not null,
  unit text not null
);
create index idx_tb_samplings_activity_type_ts on tb_activity_samplings(activity_id, sampling_type, ts);

create table tb_activity_laps (
  id uuid primary key,
  activity_id uuid not null references tb_activities(id) on delete cascade,
  lap_no integer not null,
  start_ts timestamptz,
  end_ts timestamptz,
  distance_m numeric,
  duration_s integer,
  avg_hr integer,
  avg_pace_s_per_km numeric,
  calories_kcal numeric,
  unique(activity_id, lap_no)
);

create table tb_sleep_sessions (
  id uuid primary key,
  wake_date date not null,
  started_at timestamptz not null,
  ended_at timestamptz not null,
  time_in_bed_s integer not null,
  asleep_s integer not null,
  efficiency numeric not null,
  is_main_sleep boolean not null,
  core_s integer not null,
  deep_s integer not null,
  rem_s integer not null,
  awake_s integer not null,
  unspecified_s integer not null,
  source text not null,
  first_raw_file_id uuid not null references tb_raw_files(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);
create index idx_tb_sleep_sessions_wake_date on tb_sleep_sessions(wake_date desc);

create table tb_sleep_segments (
  id uuid primary key,
  session_id uuid not null references tb_sleep_sessions(id) on delete cascade,
  stage text not null,
  started_at timestamptz not null,
  ended_at timestamptz not null,
  seq integer not null,
  unique(session_id, seq)
);
create index idx_tb_sleep_segments_session_seq on tb_sleep_segments(session_id, seq);

create table tb_daily_summaries (
  id uuid primary key,
  day date not null unique,
  move_kcal double precision not null,
  move_goal_kcal double precision not null,
  exercise_min double precision not null,
  exercise_goal_min double precision not null,
  stand_hours double precision not null,
  stand_goal_hours double precision not null,
  source text not null,
  first_raw_file_id uuid not null references tb_raw_files(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);
create index idx_tb_daily_summaries_day on tb_daily_summaries(day desc);

create table tb_daily_metrics (
  id uuid primary key,
  day date not null,
  metric text not null,
  value double precision not null,
  unit text not null,
  source text not null,
  first_raw_file_id uuid not null references tb_raw_files(id),
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(day, metric)
);
create index idx_tb_daily_metrics_day_metric on tb_daily_metrics(day desc, metric);

create table tb_apple_source_items (
  id uuid primary key,
  source_key text not null,
  item_type text not null,
  content_hash text not null,
  activity_id uuid references tb_activities(id) on delete set null,
  last_seen_snapshot_id uuid references tb_import_snapshots(id),
  created_at timestamptz not null,
  updated_at timestamptz not null,
  sleep_session_id uuid references tb_sleep_sessions(id) on delete set null,
  daily_summary_id uuid references tb_daily_summaries(id) on delete set null,
  daily_metric_id uuid references tb_daily_metrics(id) on delete set null,
  constraint tb_apple_source_items_item_type_check check (item_type in ('workout', 'route', 'record', 'sleep_session', 'daily_summary', 'daily_metric')),
  unique(source_key)
);
create index idx_tb_apple_source_items_item_type on tb_apple_source_items(item_type);
create index idx_tb_apple_source_items_last_seen_snapshot_id on tb_apple_source_items(last_seen_snapshot_id);

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
create index idx_tb_intake_payloads_source_received_at on tb_intake_payloads(source_kind, received_at desc);
create unique index idx_tb_intake_payloads_source_event_id_unique on tb_intake_payloads(source_kind, source_event_id) where source_event_id <> '';

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
create index idx_tb_jobs_claimable on tb_jobs(status, run_after, priority desc, created_at) where status = 'queued';
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
create index idx_tb_job_schedules_due on tb_job_schedules(enabled, next_run_at) where enabled = true and next_run_at is not null;

-- Provider observations are source-specific records. They remain empty until
-- the application reimports raw evidence with the new persistence code.
create table tb_source_observations (
  id uuid primary key,
  provider text not null,
  source_kind text not null,
  source_key text not null,
  content_hash text not null,
  raw_file_id uuid not null references tb_raw_files(id),
  first_seen_snapshot_id uuid references tb_import_snapshots(id),
  last_seen_snapshot_id uuid references tb_import_snapshots(id),
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(provider, source_kind, source_key)
);
create index idx_tb_source_observations_provider_kind on tb_source_observations(provider, source_kind);
create index idx_tb_source_observations_last_seen on tb_source_observations(last_seen_snapshot_id);

create table tb_activity_observations (
  id uuid primary key references tb_source_observations(id) on delete cascade,
  activity_id uuid not null references tb_activities(id) on delete cascade,
  source_activity_id text not null default '',
  sport_type text not null,
  title text not null default '',
  started_at timestamptz not null,
  ended_at timestamptz,
  distance_m numeric,
  duration_s integer,
  moving_time_s integer,
  elevation_gain_m numeric,
  avg_hr integer,
  max_hr integer,
  avg_pace_s_per_km numeric,
  calories_kcal numeric,
  match_status text not null default 'canonical',
  match_confidence numeric,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_activity_observations_match_status_check check (match_status in ('unresolved', 'canonical', 'matched', 'conflict'))
);
create index idx_tb_activity_observations_activity on tb_activity_observations(activity_id);
alter table tb_activities add column selected_observation_id uuid references tb_activity_observations(id);

create table tb_activity_observation_route_points (
  activity_observation_id uuid not null references tb_activity_observations(id) on delete cascade,
  seq integer not null,
  ts timestamptz,
  lat double precision not null,
  lon double precision not null,
  elevation_m numeric,
  distance_m numeric,
  speed_mps numeric,
  heart_rate integer,
  geom geography(Point, 4326) not null,
  primary key (activity_observation_id, seq)
);
create index idx_tb_observation_route_points_geom on tb_activity_observation_route_points using gist(geom);

create table tb_activity_observation_samplings (
  id uuid primary key,
  activity_observation_id uuid not null references tb_activity_observations(id) on delete cascade,
  sampling_type text not null,
  ts timestamptz not null,
  value numeric not null,
  unit text not null
);
create index idx_tb_observation_samplings_type_ts on tb_activity_observation_samplings(activity_observation_id, sampling_type, ts);

create table tb_activity_observation_laps (
  id uuid primary key,
  activity_observation_id uuid not null references tb_activity_observations(id) on delete cascade,
  lap_no integer not null,
  start_ts timestamptz,
  end_ts timestamptz,
  distance_m numeric,
  duration_s integer,
  avg_hr integer,
  avg_pace_s_per_km numeric,
  calories_kcal numeric,
  unique(activity_observation_id, lap_no)
);

create table tb_sleep_observations (
  id uuid primary key references tb_source_observations(id) on delete cascade,
  sleep_session_id uuid not null references tb_sleep_sessions(id) on delete cascade,
  wake_date date not null,
  started_at timestamptz not null,
  ended_at timestamptz not null,
  time_in_bed_s integer not null,
  asleep_s integer not null,
  efficiency numeric not null,
  is_main_sleep boolean not null,
  core_s integer not null,
  deep_s integer not null,
  rem_s integer not null,
  awake_s integer not null,
  unspecified_s integer not null,
  source text not null,
  match_status text not null default 'canonical',
  match_confidence numeric,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_sleep_observations_match_status_check check (match_status in ('unresolved', 'canonical', 'matched', 'conflict'))
);
create index idx_tb_sleep_observations_session on tb_sleep_observations(sleep_session_id);
alter table tb_sleep_sessions add column selected_observation_id uuid references tb_sleep_observations(id);

create table tb_sleep_observation_segments (
  id uuid primary key,
  sleep_observation_id uuid not null references tb_sleep_observations(id) on delete cascade,
  stage text not null,
  started_at timestamptz not null,
  ended_at timestamptz not null,
  seq integer not null,
  unique(sleep_observation_id, seq)
);
create index idx_tb_sleep_observation_segments_seq on tb_sleep_observation_segments(sleep_observation_id, seq);

create table tb_sleep_session_observations (
  sleep_session_id uuid not null references tb_sleep_sessions(id) on delete cascade,
  sleep_observation_id uuid not null references tb_sleep_observations(id) on delete cascade,
  match_status text not null default 'canonical',
  match_confidence numeric,
  is_preferred boolean not null default false,
  primary key (sleep_session_id, sleep_observation_id),
  constraint tb_sleep_session_observations_match_status_check check (match_status in ('unresolved', 'canonical', 'matched', 'conflict'))
);

create table tb_daily_summary_observations (
  id uuid primary key references tb_source_observations(id) on delete cascade,
  daily_summary_id uuid not null references tb_daily_summaries(id) on delete cascade,
  day date not null,
  move_kcal double precision not null,
  move_goal_kcal double precision not null,
  exercise_min double precision not null,
  exercise_goal_min double precision not null,
  stand_hours double precision not null,
  stand_goal_hours double precision not null,
  source text not null,
  match_status text not null default 'canonical',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_daily_summary_observations_match_status_check check (match_status in ('unresolved', 'canonical', 'matched', 'conflict'))
);

create table tb_daily_metric_observations (
  id uuid primary key references tb_source_observations(id) on delete cascade,
  daily_metric_id uuid not null references tb_daily_metrics(id) on delete cascade,
  day date not null,
  metric text not null,
  value double precision not null,
  unit text not null,
  source text not null,
  reducer text not null default 'source_priority',
  match_status text not null default 'canonical',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_daily_metric_observations_match_status_check check (match_status in ('unresolved', 'canonical', 'matched', 'conflict'))
);
create index idx_tb_daily_metric_observations_day_metric on tb_daily_metric_observations(day desc, metric);
alter table tb_daily_summaries add column selected_observation_id uuid references tb_daily_summary_observations(id);
alter table tb_daily_metrics add column selected_observation_id uuid references tb_daily_metric_observations(id);

-- +goose Down
drop table if exists tb_daily_metric_observations;
alter table if exists tb_daily_metrics drop column if exists selected_observation_id;
drop table if exists tb_daily_summary_observations;
alter table if exists tb_daily_summaries drop column if exists selected_observation_id;
drop table if exists tb_sleep_session_observations;
drop table if exists tb_sleep_observation_segments;
alter table if exists tb_sleep_sessions drop column if exists selected_observation_id;
drop table if exists tb_sleep_observations;
drop table if exists tb_activity_observation_laps;
drop table if exists tb_activity_observation_samplings;
drop table if exists tb_activity_observation_route_points;
alter table if exists tb_activities drop column if exists selected_observation_id;
drop table if exists tb_activity_observations;
drop table if exists tb_source_observations;
drop table if exists tb_job_schedules;
drop table if exists tb_jobs;
drop table if exists tb_intake_payloads;
drop table if exists tb_apple_source_items;
drop table if exists tb_daily_metrics;
drop table if exists tb_daily_summaries;
drop table if exists tb_sleep_segments;
drop table if exists tb_sleep_sessions;
drop table if exists tb_activity_laps;
drop table if exists tb_activity_samplings;
drop table if exists tb_activity_route_points;
drop table if exists tb_external_refs;
drop table if exists tb_activities;
drop table if exists tb_import_snapshots;
drop table if exists tb_import_jobs;
drop table if exists tb_raw_files;
