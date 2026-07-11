-- +goose Up
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

-- +goose Down
drop table if exists tb_sleep_segments;
drop table if exists tb_sleep_sessions;
