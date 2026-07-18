-- +goose Up

create table tb_cache_namespaces (
  namespace text primary key,
  generation bigint not null default 1,
  updated_at timestamptz not null,
  constraint tb_cache_namespaces_generation_check check (generation > 0)
);

create table tb_cache_entries (
  namespace text not null references tb_cache_namespaces(namespace) on delete cascade,
  cache_key text not null,
  generation bigint not null,
  value_json jsonb not null,
  expires_at timestamptz not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  primary key (namespace, cache_key),
  constraint tb_cache_entries_generation_check check (generation > 0)
);

create index idx_tb_cache_entries_expiry on tb_cache_entries(expires_at);
create index idx_tb_cache_entries_generation on tb_cache_entries(namespace, generation);

-- +goose Down

drop table if exists tb_cache_entries;
drop table if exists tb_cache_namespaces;
