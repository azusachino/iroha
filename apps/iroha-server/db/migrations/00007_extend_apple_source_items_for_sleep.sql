-- +goose Up
alter table tb_apple_source_items
  drop constraint tb_apple_source_items_item_type_check;

alter table tb_apple_source_items
  add constraint tb_apple_source_items_item_type_check
  check (item_type in ('workout', 'route', 'record', 'sleep_session'));

alter table tb_apple_source_items
  add column sleep_session_id uuid references tb_sleep_sessions(id) on delete set null;

-- +goose Down
alter table tb_apple_source_items
  drop column if exists sleep_session_id;

alter table tb_apple_source_items
  drop constraint tb_apple_source_items_item_type_check;

alter table tb_apple_source_items
  add constraint tb_apple_source_items_item_type_check
  check (item_type in ('workout', 'route', 'record'));
