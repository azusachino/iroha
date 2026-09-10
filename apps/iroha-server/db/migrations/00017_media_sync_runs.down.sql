alter table tb_import_jobs drop column sync_run_id;
drop index idx_tb_media_sync_runs_connector_started;
drop index uq_tb_media_sync_runs_connector_running;
drop table tb_media_sync_runs;
