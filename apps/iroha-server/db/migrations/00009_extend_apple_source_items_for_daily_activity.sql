-- +goose Up
alter table tb_apple_source_items
  drop constraint tb_apple_source_items_item_type_check;

alter table tb_apple_source_items
  add constraint tb_apple_source_items_item_type_check
  check (item_type in ('workout', 'route', 'record', 'sleep_session', 'daily_summary', 'daily_metric'));

alter table tb_apple_source_items
  add column daily_summary_id uuid references tb_daily_summaries(id) on delete set null,
  add column daily_metric_id uuid references tb_daily_metrics(id) on delete set null;

-- +goose Down
alter table tb_apple_source_items
  drop column if exists daily_metric_id,
  drop column if exists daily_summary_id;

alter table tb_apple_source_items
  drop constraint tb_apple_source_items_item_type_check;

alter table tb_apple_source_items
  add constraint tb_apple_source_items_item_type_check
  check (item_type in ('workout', 'route', 'record', 'sleep_session'));
