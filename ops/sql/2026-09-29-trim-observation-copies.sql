-- One-off backfill for docs/plans/2026-09-29-raw-evidence-retention-and-source-coverage.md.
-- Observation child rows were copies of the canonical rows. Import no longer
-- writes them, so empty the tables to reclaim the space. Back up first
-- (pg_dump) and run in a quiet window: TRUNCATE takes an exclusive lock.
--
-- Refuses to run when any activity or sleep session has more than one
-- observation, because then a copy could be the only surviving record.
begin;

do $$
begin
  if exists (select 1 from tb_activity_observations group by activity_id having count(*) > 1)
     or exists (select 1 from tb_sleep_observations group by sleep_session_id having count(*) > 1) then
    raise exception 'multi-observation activity or sleep session found; do not truncate';
  end if;
end $$;

truncate tb_activity_observation_route_points, tb_activity_observation_samplings,
  tb_activity_observation_laps, tb_sleep_observation_segments;

commit;
