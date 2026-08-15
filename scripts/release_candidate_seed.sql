insert into tb_raw_files (
  id, sha256, original_filename, content_type, size_bytes, storage_path,
  source_kind, uploaded_via, created_at
) values (
  '10000000-0000-0000-0000-000000000001', 'release-candidate-fixture',
  'release-candidate.xml', 'application/xml', 1, '/tmp/release-candidate.xml',
  'apple_health_export', 'release-candidate', '2026-08-01T00:00:00Z'
);

insert into tb_activities (
  id, sport_type, title, started_at, ended_at, timezone, distance_m,
  duration_s, moving_time_s, elevation_gain_m, avg_hr, max_hr,
  avg_pace_s_per_km, calories_kcal, source_kind, source_activity_id,
  first_raw_file_id, created_at, updated_at
) values (
  '20000000-0000-0000-0000-000000000001', 'run', 'Timezone edge run',
  '2026-07-31T15:30:00Z', '2026-07-31T16:15:00Z', 'Asia/Tokyo', 8000,
  2700, 2650, 42, 145, 172, 337.5, 510, 'release-candidate', 'edge-run',
  '10000000-0000-0000-0000-000000000001', '2026-08-01T00:00:00Z',
  '2026-08-01T00:00:00Z'
);

insert into tb_sleep_sessions (
  id, wake_date, started_at, ended_at, time_in_bed_s, asleep_s, efficiency,
  is_main_sleep, core_s, deep_s, rem_s, awake_s, unspecified_s, source,
  first_raw_file_id, created_at, updated_at
) values (
  '30000000-0000-0000-0000-000000000001', '2026-08-02',
  '2026-08-01T14:30:00Z', '2026-08-01T22:30:00Z', 28800, 25200, 0.875,
  true, 12600, 5400, 7200, 3600, 0, 'release-candidate',
  '10000000-0000-0000-0000-000000000001', '2026-08-02T00:00:00Z',
  '2026-08-02T00:00:00Z'
);

insert into tb_daily_summaries (
  id, day, move_kcal, move_goal_kcal, exercise_min, exercise_goal_min,
  stand_hours, stand_goal_hours, source, first_raw_file_id, created_at, updated_at
) values (
  '40000000-0000-0000-0000-000000000001', '2026-08-02', 620, 500, 48, 30,
  13, 12, 'release-candidate', '10000000-0000-0000-0000-000000000001',
  '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'
);

insert into tb_daily_metrics (
  id, day, metric, value, unit, source, first_raw_file_id, created_at, updated_at
) values
  ('41000000-0000-0000-0000-000000000001', '2026-08-02', 'steps', 12345, 'count', 'release-candidate', '10000000-0000-0000-0000-000000000001', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z'),
  ('41000000-0000-0000-0000-000000000002', '2026-08-02', 'resting_hr', 54, 'bpm', 'release-candidate', '10000000-0000-0000-0000-000000000001', '2026-08-02T00:00:00Z', '2026-08-02T00:00:00Z');

insert into tb_media_works (
  id, work_kind, primary_title, original_title, original_language,
  first_release_date, description, created_at, updated_at
) values (
  '50000000-0000-0000-0000-000000000001', 'anime', 'Release Candidate Story',
  'Release Candidate Story', 'ja', '2026-01-01', 'Deterministic fixture',
  '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z'
);

insert into tb_media_items (
  id, work_id, media_type, item_role, title, created_at, updated_at
) values (
  '51000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000001', 'anime_season', 'primary',
  'Release Candidate Story', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z'
);

insert into tb_media_consumption_events (
  id, media_item_id, event_type, event_at, source_kind, source_event_id,
  created_at
) values (
  '52000000-0000-0000-0000-000000000001',
  '51000000-0000-0000-0000-000000000001', 'completed',
  '2026-08-03T12:00:00Z', 'release-candidate', 'completed-1',
  '2026-08-03T12:00:00Z'
);

insert into tb_media_progress (
  media_item_id, status, completed_on_value, completed_on_precision,
  source_kind, updated_at
) values (
  '51000000-0000-0000-0000-000000000001', 'completed',
  '2026-08-03', 'day', 'release-candidate', '2026-08-03T12:00:00Z'
);

insert into tb_expenses (
  id, occurred_on, currency, amount_minor, category, merchant, note, items_json,
  source_kind, source_ref, create_fingerprint, created_at, updated_at
)
select
  md5('release-candidate-expense-' || n)::uuid,
  date '2026-08-01' + ((n - 1) % 20),
  'JPY', 500 + n * 10,
  (array['food', 'groceries', 'transport', 'shopping', 'entertainment'])[((n - 1) % 5) + 1],
  'Fixture merchant ' || n, '', '[]'::jsonb, 'release-candidate',
  'expense-' || n, 'fixture-' || n,
  '2026-08-01T00:00:00Z'::timestamptz + n * interval '1 minute',
  '2026-08-01T00:00:00Z'::timestamptz + n * interval '1 minute'
from generate_series(1, 55) as n;

insert into tb_expenses (
  id, occurred_on, currency, amount_minor, category, merchant, note, items_json,
  source_kind, source_ref, create_fingerprint, created_at, updated_at
) values (
  '60000000-0000-0000-0000-000000000001', '2026-08-05', 'USD', 2500,
  'subscriptions', 'Fixture USD', '', '[{"name":"service","quantity":1,"amount_minor":2500}]',
  'release-candidate', 'expense-usd', 'fixture-usd',
  '2026-08-05T00:00:00Z', '2026-08-05T00:00:00Z'
);
