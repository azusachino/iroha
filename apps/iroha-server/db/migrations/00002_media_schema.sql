-- +goose Up

create table tb_media_works (
  id uuid primary key,
  work_kind text not null,
  primary_title text not null,
  original_title text not null default '',
  original_language text not null default '',
  first_release_date date,
  description text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_items (
  id uuid primary key,
  work_id uuid references tb_media_works(id),
  parent_item_id uuid references tb_media_items(id),
  media_type text not null,
  item_role text not null,
  title text not null,
  sort_title text not null default '',
  original_title text not null default '',
  description text not null default '',
  release_date date,
  season_number integer,
  episode_number integer,
  chapter_number numeric,
  volume_number numeric,
  duration_seconds integer,
  page_count integer,
  episode_count integer,
  chapter_count integer,
  language text not null default '',
  country text not null default '',
  cover_image_url text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);
create index idx_tb_media_items_work on tb_media_items(work_id);
create index idx_tb_media_items_parent on tb_media_items(parent_item_id);

create table tb_media_titles (
  id uuid primary key,
  scope_type text not null,
  scope_id uuid not null,
  title text not null,
  language text not null default '',
  script text not null default '',
  region text not null default '',
  title_kind text not null,
  provider text not null default '',
  is_primary boolean not null default false,
  confidence numeric,
  created_at timestamptz not null
);
create index idx_tb_media_titles_title on tb_media_titles using gin(to_tsvector('simple', title));
create index idx_tb_media_titles_scope on tb_media_titles(scope_type, scope_id);

create table tb_media_relations (
  id uuid primary key,
  from_type text not null,
  from_id uuid not null,
  to_type text not null,
  to_id uuid not null,
  relation_type text not null,
  provider text not null default '',
  confidence numeric,
  created_at timestamptz not null,
  unique(from_type, from_id, to_type, to_id, relation_type, provider)
);
create index idx_tb_media_relations_from on tb_media_relations(from_type, from_id);
create index idx_tb_media_relations_to on tb_media_relations(to_type, to_id);

create table tb_media_external_refs (
  id uuid primary key,
  scope_type text not null,
  scope_id uuid not null,
  provider text not null,
  external_id text not null,
  external_url text not null default '',
  confidence numeric,
  matched_by text not null default '',
  created_at timestamptz not null,
  unique(provider, external_id)
);
create index idx_tb_media_external_refs_scope on tb_media_external_refs(scope_type, scope_id);

create table tb_media_creators (
  id uuid primary key,
  name text not null,
  sort_name text not null default '',
  original_name text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_creator_roles (
  id uuid primary key,
  creator_id uuid not null references tb_media_creators(id) on delete cascade,
  scope_type text not null,
  scope_id uuid not null,
  role text not null,
  provider text not null default '',
  created_at timestamptz not null
);
create index idx_tb_media_creator_roles_scope on tb_media_creator_roles(scope_type, scope_id);

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

create table tb_media_progress (
  media_item_id uuid primary key references tb_media_items(id) on delete cascade,
  status text not null,
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  started_at timestamptz,
  last_update_at timestamptz,
  finished_at timestamptz,
  play_count integer not null default 0,
  hidden_from_continue boolean not null default false,
  source_kind text not null default '',
  updated_at timestamptz not null
);

create table tb_media_lists (
  id uuid primary key,
  name text not null,
  list_kind text not null,
  source_kind text not null default '',
  external_ref_id uuid references tb_media_external_refs(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_list_items (
  id uuid primary key,
  list_id uuid not null references tb_media_lists(id) on delete cascade,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  position numeric,
  created_at timestamptz not null,
  unique(list_id, media_item_id)
);
create index idx_tb_media_list_items_item on tb_media_list_items(media_item_id);

create table tb_media_resolution_tasks (
  id uuid primary key,
  task_type text not null,
  status text not null,
  candidates_json jsonb not null default '[]'::jsonb,
  resolution_json jsonb not null default '{}'::jsonb,
  created_at timestamptz not null,
  resolved_at timestamptz
);
create index idx_tb_media_resolution_tasks_status on tb_media_resolution_tasks(status, created_at);

-- +goose Down

drop table if exists tb_media_resolution_tasks;
drop table if exists tb_media_list_items;
drop table if exists tb_media_lists;
drop table if exists tb_media_progress;
drop table if exists tb_media_consumption_events;
drop table if exists tb_media_creator_roles;
drop table if exists tb_media_creators;
drop table if exists tb_media_external_refs;
drop table if exists tb_media_relations;
drop table if exists tb_media_titles;
drop table if exists tb_media_items;
drop table if exists tb_media_works;
