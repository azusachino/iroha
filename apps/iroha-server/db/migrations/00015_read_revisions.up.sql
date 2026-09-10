create table tb_read_revisions (
  namespace text primary key,
  revision bigint not null default 0,
  updated_at timestamptz not null default now()
);
