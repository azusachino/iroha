create table tb_media_matching_decisions (
  id uuid primary key,
  provider text not null,
  external_id text not null,
  source_item_id uuid not null references tb_media_items(id),
  target_item_id uuid not null references tb_media_items(id),
  decision_kind text not null,
  previous_matched_by text not null default '',
  previous_confidence numeric,
  created_at timestamptz not null,
  constraint tb_media_matching_decisions_kind_check
    check (decision_kind in ('attach', 'keep_separate', 'undo'))
);

create index idx_tb_media_matching_decisions_lookup
  on tb_media_matching_decisions(provider, external_id, created_at desc, id desc);
