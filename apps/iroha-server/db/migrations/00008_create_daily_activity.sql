-- +goose Up
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

-- +goose Down
drop table if exists tb_daily_metrics;
drop table if exists tb_daily_summaries;
