# Daily Activity Module Plan

Daily activity (Apple's Move/Exercise/Stand rings + daily step/distance/flight totals) is the third first-class domain after running/fitness and sleep. It follows the same shape as every module (see
`personal-data-cockpit.md`):

```text
raw Apple Health export  ->  durable import job  ->  canonical daily records  ->  private views
```

Daily activity is **not** an activity (no route/pace/laps) and **not** a sleep session. It is one aggregate row per calendar day. Per the data-model rule ("use first-class tables for stable domains"),
it gets its own table rather than being forced into `tb_activities`.

## What the data looks like

Grounded by `scripts/activity_explore.py` against a real export (2023-09 .. 2026-07), the export carries **two distinct shapes**:

### 1. `<ActivitySummary>` — the rings (clean, one row per day)

- **1,040 daily elements**, `2023-09-02 .. 2026-07-07`. History starts at the Apple Watch era (rings did not exist before), so this domain is shorter than sleep's 2017+ — a data-era caveat, not a
  problem.
- Each element is already aggregated by Apple: `activeEnergyBurned`+ `activeEnergyBurnedGoal` (Move, kcal), `appleExerciseTime`+goal (min), `appleStandHours`+goal (h), keyed by `dateComponents` (the
  local date).
- Real averages: 635 kcal Move / 56.6 min Exercise / 12.0 h Stand; Move ring closed 91% of days, Exercise 78%.
- **These map 1:1 to `tb_daily_summaries` rows with no processing.** Apple has already deduped Move across sources.

### 2. Daily cumulative `Record`s — steps / distance / flights (double-counted)

- `HKQuantityTypeIdentifierStepCount`, `DistanceWalkingRunning`, `FlightsClimbed` arrive as many small interval records throughout the day.
- **Cross-source double-count is real and large** (the analogue of sleep's `InBed` overlap): naively summing all sources overcounts **steps by 42%** (18.3M -> 10.6M deduped), distance by 40%, flights
  by 45%.
- **926 of 1,037 days have multiple sources** — almost every day. Observed sources: `iPhone` (1024d), `iWatch` (871d), a second `Apple Watch` (69d, device replacement), plus `Nike Run Club` (2d,
  distance only).
- The daily totals therefore **require** cross-source dedup before rollup.

## Source dedup (decision: `daily-source-dedup`)

- Steps/distance/flights totals use **greedy interval-union with source priority**: sort a day's records by source priority (Watch > iPhone > third-party), accept each record only if its interval does
  not overlap an already-accepted one, then sum accepted values. This drops the lower-priority duplicate on overlap while still keeping non-overlapping contributions from a second source on hybrid
  days (watch on charger part of the day). It reuses the overlap-safe interval logic the sleep parser already has for `InBed`.
- A per-day "max single source" rollup is only a rough approximation (used by `activity_explore.py` to _expose_ the overcount); it undercounts hybrid days where different sources covered different
  parts of the day. Do not ship it.
- Rings (`ActivitySummary`) need **no** dedup — Apple already deduped Move.
- Attribution: a day belongs to the local calendar date (the `startDate` date prefix for records, `dateComponents` for summaries).

## Schema (new migration `00007_create_daily_activity.sql`, decision: `daily-metrics-hybrid`)

Two tables, split by data character: the rings are a fixed, coherent triplet (value + goal) so they stay **structured**; every other daily scalar is open-ended (dozens of HK quantity types), so it
goes in one **generic** table that the next module (body-vitals) reuses with **zero schema change**.

```text
tb_daily_summaries          -- the rings, one structured row per day
  id                uuid pk
  day               date          -- local calendar date (attribution), unique
  move_kcal         double        -- ActivitySummary activeEnergyBurned
  move_goal_kcal    double
  exercise_min      double        -- appleExerciseTime
  exercise_goal_min double
  stand_hours       double        -- appleStandHours
  stand_goal_hours  double
  source            text          -- primary source device for the day
  first_raw_file_id uuid  fk -> tb_raw_files (for reprocess purge)
  created_at / updated_at

tb_daily_metrics            -- open-ended daily scalars, one row per (day, metric)
  id                uuid pk
  day               date
  metric            text          -- 'steps' | 'distance_km' | 'flights' (+ later
                                  --   'resting_hr' | 'hrv_sdnn' | 'vo2max' |
                                  --   'body_mass_kg' | 'spo2' | 'respiratory_rate' ...)
  value             double        -- interval-union-deduped daily total/value
  unit              text          -- 'count' | 'km' | 'ml/kg/min' | ...
  source            text          -- winning source device for the day
  first_raw_file_id uuid  fk -> tb_raw_files
  created_at / updated_at
  unique(day, metric)
```

Rings stay a single-row read for ring-close trends; `tb_daily_metrics` is the extensible spine for steps/distance/flights **and the future body-vitals module** — a new metric is just more rows, never
an `ALTER TABLE`. Per-record step intervals are collapsed at parse time (unlike sleep, whose segment timeline is worth keeping for the per-night render).

## Reconcile / reprocess

Reuse the existing import machinery unchanged in spirit:

- Extend `tb_apple_source_items.item_type` to `"daily_summary"` (rings) and `"daily_metric"` (one per day+metric).
- Stable source key: the local `day` for a summary; `day|metric` for a metric (deterministic; survives re-exports; independent of zip hash).
- `content_hash` over the day's rolled-up values so an unchanged day/metric skips re-persist and a changed one upserts — same skip/upsert/insert logic as workouts and sleep.
- Reprocess purge extends to `tb_daily_summaries` and `tb_daily_metrics` in the load-bearing order (source items first, then the derived rows).

## Parser

Add a daily-activity pass to `pkg/parsers/apple_health.go` (streaming, alongside the existing workout + sampling + sleep passes — do not buffer the ~900MB file):

1. Stream `<ActivitySummary ...>` -> per-day rings (direct field copy) -> `ParsedDailySummary`.
2. Stream `<Record type="...StepCount|DistanceWalkingRunning|FlightsClimbed">` -> collect `(start, end, source, value)` per day per metric.
3. Greedy interval-union with source priority to get deduped daily totals -> `ParsedDailyMetric` rows (`day, metric, value, unit, source`).
4. Emit `ParsedDailySummary` (rings) + `[]ParsedDailyMetric` (scalars) — parallel to `ParsedActivity` / `ParsedSleepSession`, not folded into either. The metric emitter is metric-agnostic so the
   body-vitals module later adds quantity types to the same path with no new parser structure.

## Read API

- `GET /api/v1/daily?from=&to=` — per-day rows joining rings (from `tb_daily_summaries`) with that day's scalars (from `tb_daily_metrics`, pivoted into the row), keyset paginated.
- Aggregates (weekly/monthly ring-close rate, step/distance trends) follow, as they did for sleep; the first read API ships daily rows before the aggregate API exists.

## Verification

Extend `scripts/real_import_smoke.py` (or a sibling) with daily assertions: summary count > 0, deduped step total within tolerance of `activity_explore.py`'s per-day-max-source figure but strictly
less than the naive all-source sum (proves dedup ran), re-import reuse yields zero dup growth, reprocess keeps counts stable. Cross-check ring averages against `activity_explore.py`.

## Task breakdown

Tracked as the `iroha:daily-activity` epic:

1. Migration `00007_create_daily_activity.sql` + Go model.
2. Parser: daily-activity pass + `ParsedDailySummary` + interval-union-with- priority helper (unit tested, DB-free; reuse sleep's union).
3. Persist + reconcile: `item_type='daily_summary'`, reprocess purge extension.
4. Read API: `/patterns`.
5. Smoke assertions + cross-check vs `activity_explore.py`.

Web render (activity page, ring + trend charts) is a separate later epic, mirroring how running and sleep shipped their data layers before the cockpit dataviz.
