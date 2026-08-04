-- +goose Up

create table tb_tasks (
  id uuid primary key,
  title text not null,
  notes text not null default '',
  status text not null default 'open',
  due_date date,
  priority integer not null default 0,
  source text not null default 'web',
  completed_at timestamptz,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint tb_tasks_status_check check (status in ('open', 'completed', 'canceled'))
);
create index idx_tb_tasks_open_due on tb_tasks(status, due_date, priority desc, created_at desc);

-- +goose Down

drop table if exists tb_tasks;
