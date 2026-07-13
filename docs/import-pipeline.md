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

The importer should parse `export.xml` for workouts and samplings, then link workout route GPX files where possible.

Initial parser behavior:

- Parse `Workout` records from `export.xml` into canonical activities.
- Parse GPX files under `workout-routes/` as route-backed tb_activities.
- Defer precise Apple workout-to-route association until real exports are inspected.
- Defer heart-rate and other sampling extraction until the first route/activity import loop is stable.

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

Telegram is an optional inbox. The personal bot is an external upload client only: it forwards files to iroha-server and does not parse them.

The bot uploads with `uploaded_via=telegram`, sending a bearer token when auth is enabled, then creates an import job and polls its status. All parsing and dedupe stay inside iroha-server. See the
External Upload Client Contract in `iroha-server.md` for the exact request and response shapes.

## Real Apple Health Export Findings

A real Apple Health export zip exercises the product route through the HTTP API: raw-file multipart upload, import job creation, import status polling, and read API queries.

Current support:

- `export.xml` workout rows are parsed into activities.
- `workout-routes/*.gpx` files are parsed into route-backed activities.
- Large route imports work against PostGIS.

Known gaps:

- Route GPX files are imported as separate `gpx` activities. They are not yet matched back to their corresponding Apple workout rows.
- HealthKit `Record` samples are not parsed yet, so heart rate and other time-series samplings remain empty.
- Laps/splits are not parsed yet.
- Duplicate raw-file uploads dedupe the stored raw file, but creating another import job still reprocesses the full export.
- Route points are inserted one row at a time; large exports should use batched inserts or copy-style loading.
- Expected lookup misses currently produce noisy GORM `record not found` SQL logs; quiet those before regular real-data testing.

## Pipeline Stages

```text
receive upload
  -> create tb_raw_files row
  -> store raw bytes unchanged
  -> create tb_import_jobs row (status: queued)
  -> enqueue background job in tb_jobs (kind: apple_import_parse)
  -> [iroha-job worker claims job from tb_jobs]
  -> detect parser
  -> parse into ParsedActivity[]
  -> dedupe and upsert canonical activities
  -> mark import completed or failed in tb_import_jobs
  -> mark background job completed or failed in tb_jobs
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
  samplings[]
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
- samplings can be replaced by activity and sampling type
- old import jobs remain as audit history

## Error Handling

Import jobs should expose:

- `queued`
- `parsing`
- `completed`
- `failed`

Failures should keep the raw file and error message. Failed imports can be retried by creating another import job.

## Adding a New Data Domain

When introducing a new type of data to be parsed (for example, media/photo data or a new workout type):

1. **Extend Parser Cases**: Implement the file format handler inside the `pkg/parsers` package.
2. **Define SQL Tables**: Add appropriate goose SQL migrations under `apps/iroha-server/db/migrations/` to create the domain tables (e.g. `tb_activity_<domain>`).
3. **Register Job Kind**: Define a job kind constant in `pkg/jobs/service.go` (e.g. `KindMediaIntakeParse`).
4. **Register Job Handler**: Wire the job kind to its respective handler function inside `apps/iroha-job/main.go`.
5. **Enqueue Job**: Map the parser kind or file type to the newly registered job kind and enqueue it inside the `imports.Service.Create` function (or a separate ingestion service).

## Retry Safety and Idempotency

All background import jobs are processed by the worker with `max_attempts = 3`. Therefore, the import execution pipeline must be entirely retry-safe and idempotent:

1. **Atomic Ingestion**: The database operations inside `imports.Service.process()` run in a transaction. If a worker attempt dies midway, the database rolls back to keep state consistent.
2. **Same-Version Skip**: The `dispositionSkip` rule compares the file's SHA256 and the parser version of prior completed imports. If the same parser has already completed the work, a retry attempt
   will instantly short-circuit and succeed.
3. **Reprocess Purge Sequence**: If reprocessing is triggered, any previously persisted records derived from that raw file are purged first. Deleting source items _first_ is load-bearing so GORM
   doesn't skip recreating activities due to cache hits or duplicate unique keys.
