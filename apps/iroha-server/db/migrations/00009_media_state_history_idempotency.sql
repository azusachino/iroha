-- +goose Up

-- A state can legitimately return to an earlier value (A -> B -> A). The
-- v0.4.1 initial index was global to the source item and therefore rejected
-- that valid history. Scope deduplication to the raw snapshot that supplied
-- the state, while the importer compares the latest fingerprint across
-- observations to suppress unchanged syncs.
drop index if exists uq_tb_media_state_history_fingerprint;
create unique index uq_tb_media_state_history_snapshot_fingerprint
  on tb_media_state_history(source_kind, media_item_id, source_event_id, state_fingerprint, raw_file_id)
  where raw_file_id is not null;

-- +goose Down

drop index if exists uq_tb_media_state_history_snapshot_fingerprint;
create unique index uq_tb_media_state_history_fingerprint
  on tb_media_state_history(source_kind, media_item_id, source_event_id, state_fingerprint);
