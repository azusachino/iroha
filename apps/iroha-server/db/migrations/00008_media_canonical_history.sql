-- +goose Up

-- Connector observation time is distinct from raw-file storage time.
alter table tb_raw_files add column observed_at timestamptz;

alter table tb_media_items
  add column release_date_precision text not null default 'day';
alter table tb_media_items
  add constraint tb_media_items_release_date_precision_check
  check (release_date_precision in ('', 'year', 'month', 'day'));

-- These columns were allowed to turn AniList fuzzy dates into false instants.
-- Provider state is re-importable from the retained raw snapshots, so the
-- derived projection is rebuilt with the canonical partial-date contract.
alter table tb_media_progress
  add column started_on_value date,
  add column started_on_precision text not null default '',
  add column completed_on_value date,
  add column completed_on_precision text not null default '';
alter table tb_media_progress
  drop column started_at,
  drop column finished_at;
alter table tb_media_progress
  add constraint tb_media_progress_started_on_check check (
    (started_on_value is null and started_on_precision = '')
    or (started_on_value is not null and started_on_precision in ('year', 'month', 'day'))
  ),
  add constraint tb_media_progress_completed_on_check check (
    (completed_on_value is null and completed_on_precision = '')
    or (completed_on_value is not null and completed_on_precision in ('year', 'month', 'day'))
  );

-- The old table was an importer-derived mixed-semantics log. Rebuild it so a
-- null event time cannot be written again, and a provider list snapshot is
-- structurally unable to masquerade as a consumption event.
-- This is intentionally destructive: the two legacy synthetic rewatch rows
-- are not canonical evidence and must remain available only in raw files.
drop table if exists tb_media_consumption_events;
create table tb_media_consumption_events (
  id uuid primary key,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  event_type text not null,
  event_at timestamptz not null,
  source_kind text not null,
  source_event_id text not null default '',
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  rating numeric,
  rating_scale numeric,
  note text not null default '',
  raw_file_id uuid references tb_raw_files(id),
  created_at timestamptz not null,
  constraint tb_media_consumption_events_type_check check (event_type <> 'list_state')
);
create index idx_tb_media_events_item_at on tb_media_consumption_events(media_item_id, event_at desc);
create index idx_tb_media_events_source on tb_media_consumption_events(source_kind, source_event_id);
create unique index uq_tb_media_events_idempotency
  on tb_media_consumption_events(source_kind, source_event_id, event_type)
  where source_event_id <> '';

create table tb_media_state_history (
  id uuid primary key,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  source_kind text not null,
  source_event_id text not null default '',
  observed_at timestamptz not null,
  effective_at timestamptz,
  time_basis text not null,
  change_kind text not null,
  state_fingerprint text not null,
  status text not null default '',
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  rating numeric,
  rating_scale numeric,
  note text not null default '',
  repeat_count integer not null default 0,
  started_on_value date,
  started_on_precision text not null default '',
  completed_on_value date,
  completed_on_precision text not null default '',
  effective_on_value date,
  effective_on_precision text not null default '',
  provider_recorded_at timestamptz,
  raw_file_id uuid references tb_raw_files(id),
  created_at timestamptz not null,
  constraint tb_media_state_history_time_basis_check check (
    time_basis in ('manual_exact', 'provider_activity', 'provider_recorded', 'iroha_observed', 'source_fuzzy_date', 'source_date')
  ),
  constraint tb_media_state_history_change_kind_check check (
    change_kind in ('snapshot', 'changed', 'removed', 'provider_activity')
  ),
  constraint tb_media_state_history_started_on_check check (
    (started_on_value is null and started_on_precision = '')
    or (started_on_value is not null and started_on_precision in ('year', 'month', 'day'))
  ),
  constraint tb_media_state_history_completed_on_check check (
    (completed_on_value is null and completed_on_precision = '')
    or (completed_on_value is not null and completed_on_precision in ('year', 'month', 'day'))
  ),
  constraint tb_media_state_history_effective_on_check check (
    (effective_on_value is null and effective_on_precision = '')
    or (effective_on_value is not null and effective_on_precision in ('year', 'month', 'day'))
  )
);
create index idx_tb_media_state_history_item_observed
  on tb_media_state_history(media_item_id, observed_at desc, id desc);
create index idx_tb_media_state_history_source
  on tb_media_state_history(source_kind, source_event_id, observed_at desc);
create unique index uq_tb_media_state_history_fingerprint
  on tb_media_state_history(source_kind, media_item_id, source_event_id, state_fingerprint);

-- +goose Down

drop table if exists tb_media_state_history;
drop table if exists tb_media_consumption_events;

alter table tb_media_progress
  drop constraint if exists tb_media_progress_started_on_check,
  drop constraint if exists tb_media_progress_completed_on_check,
  drop column if exists started_on_value,
  drop column if exists started_on_precision,
  drop column if exists completed_on_value,
  drop column if exists completed_on_precision,
  add column started_at timestamptz,
  add column finished_at timestamptz;

alter table tb_media_items
  drop constraint if exists tb_media_items_release_date_precision_check,
  drop column if exists release_date_precision;
alter table tb_raw_files drop column if exists observed_at;

create table tb_media_consumption_events (
  id uuid primary key,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  event_type text not null,
  event_at timestamptz,
  source_kind text not null,
  source_event_id text not null default '',
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  rating numeric,
  rating_scale numeric,
  note text not null default '',
  raw_file_id uuid references tb_raw_files(id),
  created_at timestamptz not null
);
create index idx_tb_media_events_item_at on tb_media_consumption_events(media_item_id, event_at desc);
create index idx_tb_media_events_source on tb_media_consumption_events(source_kind, source_event_id);
