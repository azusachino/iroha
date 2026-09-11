create table tb_source_instances (
  id uuid primary key,
  provider text not null,
  instance_key text not null,
  display_name text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(provider, instance_key)
);

create table tb_source_receipts (
  id uuid primary key,
  source_instance_id uuid not null references tb_source_instances(id),
  raw_file_id uuid not null references tb_raw_files(id),
  source_kind text not null,
  ingestion_mode text not null default 'full_snapshot',
  scope_json jsonb not null default '{}'::jsonb,
  ordering_basis text not null default '',
  observed_at timestamptz,
  received_at timestamptz not null,
  created_at timestamptz not null,
  constraint tb_source_receipts_ingestion_mode_check check (ingestion_mode in ('full_snapshot', 'bounded_replacement', 'incremental'))
);

create index idx_tb_source_receipts_instance_received on tb_source_receipts(source_instance_id, received_at desc);
create index idx_tb_source_receipts_raw_file on tb_source_receipts(raw_file_id);
