
drop index if exists uq_tb_media_state_history_snapshot_fingerprint;
create unique index uq_tb_media_state_history_fingerprint
  on tb_media_state_history(source_kind, media_item_id, source_event_id, state_fingerprint);
