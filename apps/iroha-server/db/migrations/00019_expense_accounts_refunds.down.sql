drop index idx_tb_expenses_account_date_active;
drop index uq_tb_expenses_account_source;
alter table tb_expenses
  add constraint tb_expenses_source_kind_source_ref_key unique (source_kind, source_ref),
  drop constraint tb_expenses_refund_self_check,
  drop constraint tb_expenses_kind_check,
  drop constraint tb_expenses_account_key_check,
  drop column refund_of_expense_id,
  drop column kind,
  drop column account_key;
