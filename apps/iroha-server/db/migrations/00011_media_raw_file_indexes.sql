-- +goose Up

-- purgeDerivedForRawFile (apps/iroha-imports/reprocess.go) deletes from both
-- tables by raw_file_id alone on every reprocess. Neither table had an index
-- usable for that predicate: tb_media_state_history's only raw_file_id-
-- bearing index (00009) has it as the last column of a 5-column composite,
-- which Postgres can't use for a standalone equality lookup (leftmost-prefix
-- rule), and tb_media_consumption_events had no raw_file_id index at all.
create index idx_tb_media_events_raw_file on tb_media_consumption_events(raw_file_id);
create index idx_tb_media_state_history_raw_file on tb_media_state_history(raw_file_id);

-- +goose Down

drop index if exists idx_tb_media_events_raw_file;
drop index if exists idx_tb_media_state_history_raw_file;
