
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
