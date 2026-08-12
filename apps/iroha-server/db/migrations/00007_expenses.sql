-- +goose Up

create table tb_expenses (
  id uuid primary key,
  occurred_on date not null,
  currency text not null,
  amount_minor bigint not null,
  category text not null,
  merchant text not null default '',
  note text not null default '',
  items_json jsonb not null default '[]'::jsonb,
  source_kind text not null,
  source_ref text not null,
  create_fingerprint text not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint tb_expenses_amount_check check (amount_minor > 0),
  constraint tb_expenses_currency_check check (currency in ('JPY', 'USD', 'EUR', 'GBP')),
  constraint tb_expenses_category_check check (category in ('food', 'groceries', 'transport', 'shopping', 'housing', 'utilities', 'health', 'entertainment', 'subscriptions', 'work', 'other')),
  constraint tb_expenses_items_array_check check (jsonb_typeof(items_json) = 'array'),
  constraint tb_expenses_items_limit_check check (jsonb_array_length(items_json) <= 32),
  constraint tb_expenses_source_kind_check check (source_kind <> ''),
  constraint tb_expenses_source_ref_check check (source_ref <> ''),
  constraint tb_expenses_fingerprint_check check (create_fingerprint <> ''),
  constraint tb_expenses_timestamps_check check (updated_at >= created_at and (deleted_at is null or deleted_at >= created_at)),
  unique (source_kind, source_ref)
);
create index idx_tb_expenses_occurred_on_active on tb_expenses(occurred_on desc, id desc) where deleted_at is null;
create index idx_tb_expenses_currency_category_active on tb_expenses(currency, category, occurred_on desc) where deleted_at is null;

-- +goose Down

drop table if exists tb_expenses;
