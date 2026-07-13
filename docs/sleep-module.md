# Sleep Module Plan

Sleep is the second first-class domain after running/fitness. It follows the same shape as every module (see `personal-data-cockpit.md`):

```text
raw Apple Health export  ->  durable import job  ->  canonical sleep records  ->  private views
```

Sleep is **not** an activity. It has no distance, pace, or route; it is a nightly session made of many contiguous stage segments. Per the data-model rule ("use first-class tables for stable domains"),
it gets its own tables rather than being forced into `tb_activities`.

## What the data looks like

Grounded by `scripts/sleep_explore.py` against a real export (32,156 `HKCategoryTypeIdentifierSleepAnalysis` records, 2017–2026):

- Each record is a short **stage segment** with `startDate`, `endDate`, and a `value` (the stage). A night is dozens of them.
- HealthKit stage values map to slugs: `InBed`, `AsleepUnspecified`, `Awake`, `AsleepCore` (core), `AsleepDeep` (deep), `AsleepREM` (rem).
- **Overlap caveat**: `InBed` is a coarse envelope that overlaps the fine stage values. Per-category durations must be an **interval union**, never a naive sum (a naive sum blows past 100% and is
  wrong).
- **Two eras**: granular stages (Core/Deep/REM) only exist from ~2023 (watchOS 9+). 2017–2020 is `InBed`/`AsleepUnspecified` only — the model must handle both.

Exploration output (60-min gap): 1,326 sessions, 936 main-sleep nights (near one per day across 2024–2026), avg 6.71 h asleep / 7.12 h in bed / 94.8% efficiency, stage mix core 60% / deep 13% / rem
23% / awake 4%. The asleep-duration histogram is cleanly bimodal, confirming the threshold separates real nights from fragments/naps.

## Sessionization (decision: `sleep-sessionization`)

- **Gap-merge**: consecutive segments join one session while the inter-segment gap stays under a threshold (default **60 min**); a larger gap starts a new session. Threshold is tunable.
- **Attribution**: a session belongs to its **wake date** (`ended_at` date).
- **Main vs nap**: `is_main_sleep = asleep >= 3h`, else nap/short.
- **Per-category durations** use interval union (overlap-safe).

## Schema (new migration `00006_create_sleep_core.sql`)

```text
tb_sleep_sessions
  id                uuid pk
  wake_date         date          -- attribution (date of ended_at, local)
  started_at        timestamptz   -- bedtime (first segment start)
  ended_at          timestamptz   -- wake (last segment end)
  time_in_bed_s     int           -- InBed union, else session span
  asleep_s          int           -- union of asleep stages
  efficiency        double        -- asleep_s / time_in_bed_s
  is_main_sleep     bool
  core_s / deep_s / rem_s / awake_s / unspecified_s  int  -- denormalized rollup
  source            text          -- sourceName (device)
  first_raw_file_id uuid  fk -> tb_raw_files (for reprocess purge)
  created_at / updated_at

tb_sleep_segments
  id           uuid pk
  session_id   uuid fk -> tb_sleep_sessions (on delete cascade)
  stage        text          -- in_bed | awake | core | deep | rem | asleep_unspecified
  started_at / ended_at  timestamptz
  seq          int
```

Rollup columns on the session make the common query (nightly trends, stage mix) a single-row read; `tb_sleep_segments` backs the per-night timeline render.

## Reconcile / reprocess

Reuse the existing import machinery unchanged in spirit:

- Extend `tb_apple_source_items.item_type` beyond `"workout"` to `"sleep_session"`.
- Stable source key per session: `sourceName | wake_date | started_at | ended_at` (survives re-exports; independent of zip hash).
- `content_hash` over the session's segments so an unchanged night skips re-persist and a changed night upserts — same skip/upsert/insert logic as workouts.
- Reprocess purge extends to sleep tables in the load-bearing order (source items first, then sessions cascade to segments).

## Parser

Add a sleep pass to `pkg/parsers/apple_health.go` (streaming, alongside the existing workout + sampling passes — do not buffer the ~900MB file):

1. Stream `<Record type="HKCategoryTypeIdentifierSleepAnalysis">`, keep only selected stage values, decode `start`/`end`/`value`/`sourceName`.
2. Gap-merge into sessions; compute overlap-safe rollups.
3. Emit a new `ParsedSleepSession` (with segments) — parallel to `ParsedActivity`, not folded into it.

## Read API

- `GET /api/v1/sleep?from=&to=` — nightly sessions (rollups), keyset paginated.
- `GET /api/v1/sleep/{id}/segments` — the stage timeline for one night.
- Aggregates (weekly/monthly stage mix, duration trend) are a required follow-up for the sleep cockpit; the first read API intentionally ships nightly rows before the aggregate API exists.

## User requirements side note (2026-07-11)

The sleep cockpit must provide yearly and monthly aggregation across the full sleep history, not only statistics calculated from the currently loaded recent sessions page. At minimum, the aggregate
view needs session/night count, average asleep duration, average time in bed, efficiency, main-sleep count, and stage-duration trends for the selected year/month range. The nightly timeline remains
the drill-down view beneath those aggregates.

## Verification

Extend `scripts/real_import_smoke.py` (or a sibling) with sleep assertions: session count > 0, segments > 0, stage rollups sum consistent (asleep ≈ core+deep+rem+unspecified within tolerance),
re-import reuse yields zero dup growth, reprocess keeps counts stable. Cross-check totals against `sleep_explore.py`.

## Task breakdown

Tracked as the `iroha:sleep-module` epic:

1. Migration `00006_create_sleep_core.sql` + Go models.
2. Parser: sleep pass + `ParsedSleepSession` + gap-merge/union helpers (unit tested, DB-free).
3. Persist + reconcile: `item_type='sleep_session'`, reprocess purge extension.
4. Read API: `/sleep` + `/sleep/{id}/segments`.
5. Smoke assertions + cross-check vs `sleep_explore.py`.

Web render (sleep page, stage/duration charts) is a separate later epic, mirroring how running shipped its data layer before the cockpit dataviz.
