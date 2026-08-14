-- Deterministic scale fixture for the v0.4.1 read-path gate.
-- It is intentionally separate from the browser fixture: the latter keeps
-- August 2026 small and legible while this covers ten years of history.

insert into tb_raw_files (
  id, sha256, original_filename, content_type, size_bytes, storage_path,
  source_kind, uploaded_via, created_at
) values (
  '70000000-0000-0000-0000-000000000001', 'release-candidate-performance-fixture',
  'release-candidate-performance.xml', 'application/xml', 1, '/tmp/release-candidate-performance.xml',
  'release-candidate-performance', 'release-candidate', '2016-01-01T00:00:00Z'
);

insert into tb_daily_metrics (
  id, day, metric, value, unit, source, first_raw_file_id, created_at, updated_at
)
select
  md5('performance-metric:' || to_char(day, 'YYYY-MM-DD'))::uuid,
  day, 'steps', 7000 + extract(day from day)::int, 'count',
  'release-candidate-performance', '70000000-0000-0000-0000-000000000001',
  day::timestamptz, day::timestamptz
from generate_series(date '2016-01-01', date '2025-12-31', interval '1 day') as days(day);

insert into tb_daily_summaries (
  id, day, move_kcal, move_goal_kcal, exercise_min, exercise_goal_min,
  stand_hours, stand_goal_hours, source, first_raw_file_id, created_at, updated_at
)
select
  md5('performance-summary:' || to_char(day, 'YYYY-MM-DD'))::uuid,
  day, 500, 600, 35, 30, 12, 12,
  'release-candidate-performance', '70000000-0000-0000-0000-000000000001',
  day::timestamptz, day::timestamptz
from generate_series(date '2016-01-01', date '2025-12-31', interval '1 day') as days(day);

insert into tb_activities (
  id, sport_type, title, started_at, ended_at, timezone, distance_m,
  duration_s, moving_time_s, elevation_gain_m, avg_hr, max_hr,
  avg_pace_s_per_km, calories_kcal, source_kind, source_activity_id,
  first_raw_file_id, created_at, updated_at
)
select
  md5('performance-activity:' || to_char(day, 'YYYY-MM-DD'))::uuid,
  case when extract(isodow from day)::int % 2 = 0 then 'run' else 'walk' end,
  'Performance activity ' || to_char(day, 'YYYY-MM-DD'),
  day + interval '08:00', day + interval '09:00', 'Asia/Tokyo',
  5000, 3600, 3500, 25, 135, 160, 432, 380,
  'release-candidate-performance', 'performance-activity-' || to_char(day, 'YYYY-MM-DD'),
  '70000000-0000-0000-0000-000000000001', day::timestamptz, day::timestamptz
from generate_series(date '2016-01-01', date '2025-12-31', interval '1 day') as days(day);

insert into tb_sleep_sessions (
  id, wake_date, started_at, ended_at, time_in_bed_s, asleep_s, efficiency,
  is_main_sleep, core_s, deep_s, rem_s, awake_s, unspecified_s, source,
  first_raw_file_id, created_at, updated_at
)
select
  md5('performance-sleep:' || to_char(day, 'YYYY-MM-DD'))::uuid,
  day, day - interval '8 hours', day, 28800, 25200, 0.875, true,
  12600, 5400, 7200, 3600, 0, 'release-candidate-performance',
  '70000000-0000-0000-0000-000000000001', day::timestamptz, day::timestamptz
from generate_series(date '2016-01-01', date '2025-12-31', interval '1 day') as days(day);

insert into tb_media_works (
  id, work_kind, primary_title, original_title, original_language,
  first_release_date, description, created_at, updated_at
)
select
  md5('performance-work:' || to_char(month, 'YYYY-MM'))::uuid,
  'book', 'Performance work ' || to_char(month, 'YYYY-MM'),
  'Performance work ' || to_char(month, 'YYYY-MM'), 'en', month,
  'Deterministic performance fixture', month::timestamptz, month::timestamptz
from generate_series(date '2016-01-01', date '2025-12-01', interval '1 month') as months(month);

insert into tb_media_items (
  id, work_id, media_type, item_role, title, created_at, updated_at
)
select
  md5('performance-item:' || to_char(month, 'YYYY-MM'))::uuid,
  md5('performance-work:' || to_char(month, 'YYYY-MM'))::uuid,
  'book', 'primary', 'Performance work ' || to_char(month, 'YYYY-MM'),
  month::timestamptz, month::timestamptz
from generate_series(date '2016-01-01', date '2025-12-01', interval '1 month') as months(month);

insert into tb_media_consumption_events (
  id, media_item_id, event_type, event_at, source_kind, source_event_id, created_at
)
select
  md5('performance-event:' || to_char(month, 'YYYY-MM'))::uuid,
  md5('performance-item:' || to_char(month, 'YYYY-MM'))::uuid,
  'completed', month + interval '15 days', 'release-candidate-performance',
  'performance-event-' || to_char(month, 'YYYY-MM'), month::timestamptz
from generate_series(date '2016-01-01', date '2025-12-01', interval '1 month') as months(month);

insert into tb_media_progress (
  media_item_id, status, finished_at, source_kind, updated_at
)
select
  md5('performance-item:' || to_char(month, 'YYYY-MM'))::uuid,
  'completed', month + interval '15 days', 'release-candidate-performance', month::timestamptz
from generate_series(date '2016-01-01', date '2025-12-01', interval '1 month') as months(month);

insert into tb_expenses (
  id, occurred_on, currency, amount_minor, category, merchant, note, items_json,
  source_kind, source_ref, create_fingerprint, created_at, updated_at
)
select
  md5('performance-expense:' || n)::uuid,
  (date '2016-01-01' + ((n - 1) / 100) * interval '1 month' + ((n - 1) % 28) * interval '1 day')::date,
  case when n % 10 = 0 then 'USD' else 'JPY' end,
  400 + (n % 900),
  (array['food', 'groceries', 'transport', 'shopping', 'entertainment', 'health'])[((n - 1) % 6) + 1],
  'Performance merchant ' || n, '', '[]'::jsonb,
  'release-candidate-performance', 'performance-expense-' || n, 'performance-fingerprint-' || n,
  '2016-01-01T00:00:00Z'::timestamptz + n * interval '1 minute',
  '2016-01-01T00:00:00Z'::timestamptz + n * interval '1 minute'
from generate_series(1, 12000) as numbers(n);
