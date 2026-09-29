-- Raw file bytes are deleted after the retention window; the row stays so
-- duplicate uploads are still recognised by sha256.
alter table tb_raw_files add column purged_at timestamptz;
create index idx_tb_raw_files_live on tb_raw_files(created_at) where purged_at is null;
