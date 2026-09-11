alter table tb_source_observations
  add column source_instance_id uuid references tb_source_instances(id);

-- Existing observations are attached to the latest real receipt for their
-- evidence blob. A source observation without receipt context is not valid in
-- the fresh cut-over schema and aborts the install rather than inventing
-- provenance.
update tb_source_observations so
set source_instance_id = receipts.source_instance_id
from (
  select distinct on (raw_file_id) raw_file_id, source_instance_id
  from tb_source_receipts
  order by raw_file_id, received_at desc, id desc
) receipts
where so.raw_file_id = receipts.raw_file_id
  and so.source_instance_id is null;

do $$
begin
  if exists (select 1 from tb_source_observations where source_instance_id is null) then
    raise exception 'source observations without receipt context cannot be cut over; replay raw evidence first';
  end if;
end
$$;

alter table tb_source_observations
  alter column source_instance_id set not null;

alter table tb_source_observations
  drop constraint if exists tb_source_observations_provider_source_kind_source_key_key;

alter table tb_source_observations
  add constraint uq_tb_source_observations_instance_kind_key
  unique (source_instance_id, source_kind, source_key);

alter table tb_source_observations
  add constraint uq_tb_source_observations_instance_id
  unique (source_instance_id, id);

alter table tb_source_receipts
  add constraint uq_tb_source_receipts_instance_id
  unique (source_instance_id, id);

create table tb_source_observation_receipts (
  source_instance_id uuid not null,
  source_observation_id uuid not null,
  source_receipt_id uuid not null,
  import_snapshot_id uuid not null references tb_import_snapshots(id),
  content_hash text not null,
  created_at timestamptz not null,
  primary key (source_receipt_id, source_observation_id, import_snapshot_id),
  foreign key (source_instance_id, source_observation_id)
    references tb_source_observations(source_instance_id, id)
    on delete cascade,
  foreign key (source_instance_id, source_receipt_id)
    references tb_source_receipts(source_instance_id, id)
    on delete cascade
);
create index idx_tb_source_observation_receipts_observation
  on tb_source_observation_receipts(source_observation_id, created_at desc);

insert into tb_source_observation_receipts (
  source_instance_id,
  source_observation_id,
  source_receipt_id,
  import_snapshot_id,
  content_hash,
  created_at
)
select so.source_instance_id,
       so.id,
       receipt.id,
       coalesce(so.last_seen_snapshot_id, so.first_seen_snapshot_id),
       so.content_hash,
       coalesce(receipt.created_at, so.created_at)
from tb_source_observations so
join tb_source_receipts receipt
  on receipt.raw_file_id = so.raw_file_id
 and receipt.source_instance_id = so.source_instance_id
where coalesce(so.last_seen_snapshot_id, so.first_seen_snapshot_id) is not null
on conflict do nothing;

alter table tb_activity_observations
  add constraint uq_tb_activity_observations_activity_id
  unique (activity_id, id);
alter table tb_activities
  add constraint fk_tb_activities_selected_observation_owner
  foreign key (id, selected_observation_id)
  references tb_activity_observations(activity_id, id);

alter table tb_sleep_observations
  add constraint uq_tb_sleep_observations_session_id
  unique (sleep_session_id, id);
alter table tb_sleep_sessions
  add constraint fk_tb_sleep_sessions_selected_observation_owner
  foreign key (id, selected_observation_id)
  references tb_sleep_observations(sleep_session_id, id);
alter table tb_sleep_session_observations
  add constraint fk_tb_sleep_session_observations_owner
  foreign key (sleep_session_id, sleep_observation_id)
  references tb_sleep_observations(sleep_session_id, id);

alter table tb_daily_summary_observations
  add constraint uq_tb_daily_summary_observations_summary_id
  unique (daily_summary_id, id);
alter table tb_daily_summaries
  add constraint fk_tb_daily_summaries_selected_observation_owner
  foreign key (id, selected_observation_id)
  references tb_daily_summary_observations(daily_summary_id, id);

alter table tb_daily_metric_observations
  add constraint uq_tb_daily_metric_observations_metric_id
  unique (daily_metric_id, id);
alter table tb_daily_metrics
  add constraint fk_tb_daily_metrics_selected_observation_owner
  foreign key (id, selected_observation_id)
  references tb_daily_metric_observations(daily_metric_id, id);

alter table tb_daily_metric_observations
  add constraint tb_daily_metric_observations_unit_check check (unit <> '');
alter table tb_daily_metrics
  add constraint tb_daily_metrics_unit_check check (unit <> '');
