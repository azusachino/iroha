-- Cumulative reading duration (in seconds) reported by reading gateways
-- (such as Drifter / WeRead). Stored on the progress projection for reading
-- items so dashboard and library views can surface actual time invested.
alter table tb_media_progress
  add column reading_time_seconds integer check (reading_time_seconds is null or reading_time_seconds >= 0);
