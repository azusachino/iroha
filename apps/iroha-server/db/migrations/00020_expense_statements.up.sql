alter table tb_expenses
  add column original_transaction_ref text not null default '';

create table tb_expense_statements (
  id uuid primary key,
  account_key text not null,
  source_kind text not null,
  statement_ref text not null,
  period_from date not null,
  period_to date not null,
  completeness text not null,
  revision bigint not null,
  csv_sha256 text not null,
  row_count integer not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_expense_statements_period_check check (period_from < period_to),
  constraint tb_expense_statements_completeness_check check (completeness in ('partial', 'complete')),
  constraint tb_expense_statements_revision_check check (revision > 0),
  constraint tb_expense_statements_sha_check check (csv_sha256 <> ''),
  constraint tb_expense_statements_row_count_check check (row_count >= 0),
  unique (account_key, source_kind, statement_ref, revision)
);

create table tb_expense_statement_rows (
  id uuid primary key,
  statement_id uuid not null references tb_expense_statements(id),
  account_key text not null,
  source_kind text not null,
  transaction_id text not null,
  expense_id uuid not null references tb_expenses(id),
  row_fingerprint text not null,
  occurred_on date not null,
  tombstoned_at timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique (statement_id, transaction_id)
);

create index idx_tb_expense_statements_scope
  on tb_expense_statements(account_key, source_kind, period_from, period_to, revision desc);
create index idx_tb_expense_statement_rows_identity
  on tb_expense_statement_rows(account_key, source_kind, transaction_id, created_at desc);
create index idx_tb_expense_statement_rows_period
  on tb_expense_statement_rows(account_key, source_kind, occurred_on);
