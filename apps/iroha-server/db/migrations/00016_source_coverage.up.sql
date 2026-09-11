create table tb_source_coverage_assertions (
  id uuid primary key,
  source_instance_id uuid not null references tb_source_instances(id),
  source_receipt_id uuid references tb_source_receipts(id),
  import_snapshot_id uuid references tb_import_snapshots(id),
  category text not null,
  scope_json jsonb not null default '{}'::jsonb,
  interval_start timestamptz not null,
  interval_end timestamptz not null,
  timezone text not null,
  ingestion_mode text not null,
  completeness text not null,
  recorded_at timestamptz not null,
  created_at timestamptz not null,
  constraint tb_source_coverage_evidence_check check (source_receipt_id is not null or import_snapshot_id is not null),
  constraint tb_source_coverage_interval_check check (interval_start < interval_end),
  constraint tb_source_coverage_scope_check check (jsonb_typeof(scope_json) = 'object'),
  constraint tb_source_coverage_ingestion_mode_check check (ingestion_mode in ('full_snapshot', 'bounded_replacement', 'incremental')),
  constraint tb_source_coverage_completeness_check check (completeness in ('unknown', 'partial', 'covered', 'covered_empty'))
);

create index idx_tb_source_coverage_scope_interval
  on tb_source_coverage_assertions(source_instance_id, category, interval_start, interval_end);
create index idx_tb_source_coverage_recorded
  on tb_source_coverage_assertions(source_instance_id, category, recorded_at desc, created_at desc);
