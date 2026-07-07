# Import Pipeline

## Goal

The import pipeline turns raw source files into canonical activities while preserving the original bytes forever.

## Supported MVP Sources

### Apple Health Export Zip

Primary iPhone import path for MVP v0.

User flow:

```text
Health app
  -> profile icon
  -> Export All Health Data
  -> save export.zip
  -> upload to iroha
```

Expected contents:

```text
apple_health_export/
  export.xml
  workout-routes/
    route_*.gpx
```

The importer should parse `export.xml` for workouts and samples, then link workout route GPX files where possible.

Initial parser behavior:

- Parse `Workout` records from `export.xml` into canonical activities.
- Parse GPX files under `workout-routes/` as route-backed tb_activities.
- Defer precise Apple workout-to-route association until real exports are inspected.
- Defer heart-rate and other sample extraction until the first route/activity import loop is stable.

### GPX

GPX is the simplest route-first import source. It should be supported early to validate route rendering and PostGIS storage.

Initial parser behavior:

- Each GPX track becomes one activity.
- Track points become `tb_activity_route_points`.
- The parser defaults `sport_type` to `run`.
- Reprocessing dedupes by external reference derived from the raw file hash.

### FIT and TCX

FIT and TCX are in MVP scope, but parser depth can improve incrementally. The data model should not assume all sources provide the same fields.

### Strava Export Zip

Strava is a legacy import adapter only. Strava IDs may be stored as external references but must not become primary IDs.

### Telegram Document

Telegram is an optional inbox. The bot forwards files to iroha-server and does not parse them.

## Pipeline Stages

```text
receive upload
  -> create tb_raw_files row
  -> store raw bytes unchanged
  -> create tb_import_jobs row
  -> detect parser
  -> parse into ParsedActivity[]
  -> dedupe and upsert canonical activities
  -> mark import completed or failed
```

## Parser Contract

Each parser should return source-independent records:

```text
ParsedActivity
  source_kind
  source_activity_id
  sport_type
  title
  started_at
  ended_at
  timezone
  summary metrics
  route_points[]
  samples[]
  laps[]
  metadata
```

The parser should not write canonical tables directly. A normalization step handles dedupe and persistence.

## Dedupe Rules

Use multiple signals:

- `tb_raw_files.sha256` prevents storing identical files twice.
- `tb_external_refs(provider, external_id)` prevents duplicate source activities.
- fallback matching may use sport type, start time, duration, and distance when source IDs are absent.

Fallback matching should be conservative. It is better to surface a duplicate candidate than silently merge unrelated activities.

## Reprocessing

Parser improvements should create a new `tb_import_jobs` row for the same `raw_file_id`.

Reprocessing must be idempotent:

- same source identity updates the same activity
- route points for an activity can be replaced transactionally
- samples can be replaced by activity and sample type
- old import jobs remain as audit history

## Error Handling

Import jobs should expose:

- `queued`
- `parsing`
- `completed`
- `failed`

Failures should keep the raw file and error message. Failed imports can be retried by creating another import job.
