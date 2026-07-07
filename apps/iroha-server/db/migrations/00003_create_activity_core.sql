-- +goose Up
create table activities (
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
  source_kind text not null,
  source_activity_id text not null default '',
  first_raw_file_id uuid not null references raw_files(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create index idx_activities_started_at on activities(started_at desc);
create index idx_activities_sport_started on activities(sport_type, started_at desc);

create table external_refs (
  id uuid primary key,
  activity_id uuid not null references activities(id) on delete cascade,
  provider text not null,
  external_id text not null,
  raw_file_id uuid not null references raw_files(id),
  created_at timestamptz not null,
  unique(provider, external_id)
);

create table activity_route_points (
  activity_id uuid not null references activities(id) on delete cascade,
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

create index idx_route_points_geom on activity_route_points using gist(geom);

create table activity_samples (
  id uuid primary key,
  activity_id uuid not null references activities(id) on delete cascade,
  sample_type text not null,
  ts timestamptz not null,
  value numeric not null,
  unit text not null
);

create index idx_samples_activity_type_ts on activity_samples(activity_id, sample_type, ts);

create table activity_laps (
  id uuid primary key,
  activity_id uuid not null references activities(id) on delete cascade,
  lap_no integer not null,
  start_ts timestamptz,
  end_ts timestamptz,
  distance_m numeric,
  duration_s integer,
  avg_hr integer,
  avg_pace_s_per_km numeric,
  unique(activity_id, lap_no)
);

-- +goose Down
drop table if exists activity_laps;
drop table if exists activity_samples;
drop table if exists activity_route_points;
drop table if exists external_refs;
drop table if exists activities;
