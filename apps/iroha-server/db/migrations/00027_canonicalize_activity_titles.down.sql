-- Restore the exact pre-remediation title and sport slug from the audit log,
-- then drop the provenance column and the audit table.
--
-- Restoration is scoped per table by tb_activity_title_remediation.target_table
-- and uses the earliest audit row per target, so a value can never be walked
-- back past its original state. updated_at is left alone: this is an undo of a
-- data rewrite, not a new observation.

update tb_activities a
set title = r.old_title, sport_type = r.old_sport_type
from (
    select distinct on (row_id) row_id, old_title, old_sport_type
    from tb_activity_title_remediation
    where target_table = 'tb_activities'
    order by row_id, id
) r
where a.id = r.row_id;

update tb_activity_observations o
set title = r.old_title, sport_type = r.old_sport_type
from (
    select distinct on (row_id) row_id, old_title, old_sport_type
    from tb_activity_title_remediation
    where target_table = 'tb_activity_observations'
    order by row_id, id
) r
where o.id = r.row_id;

drop table if exists tb_activity_title_remediation;

alter table tb_activities drop constraint if exists tb_activities_title_origin_check;
alter table tb_activities drop column if exists title_origin;

alter table tb_activity_observations drop constraint if exists tb_activity_observations_title_origin_check;
alter table tb_activity_observations drop column if exists title_origin;
