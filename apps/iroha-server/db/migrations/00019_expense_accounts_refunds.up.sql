alter table tb_expenses
  add column account_key text not null default 'default',
  add column kind text not null default 'expense',
  add column refund_of_expense_id uuid references tb_expenses(id);

alter table tb_expenses
  add constraint tb_expenses_account_key_check check (account_key <> ''),
  add constraint tb_expenses_kind_check check (kind in ('expense', 'refund')),
  add constraint tb_expenses_refund_self_check check (refund_of_expense_id is null or refund_of_expense_id <> id);

alter table tb_expenses drop constraint tb_expenses_source_kind_source_ref_key;
create unique index uq_tb_expenses_account_source on tb_expenses(account_key, source_kind, source_ref);
create index idx_tb_expenses_account_date_active on tb_expenses(account_key, occurred_on desc, id desc) where deleted_at is null;
