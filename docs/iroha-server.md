# Iroha Server

## Responsibility

`iroha-server` owns raw ingestion, import processing, canonical activity storage, and the private presentation API.

It should not depend on Strava, Apple, Telegram, or any other vendor as a source of truth. Those systems are producers of raw files or bundles.

## API Namespace

All HTTP endpoints live under:

```text
/api/v1
```

Use Chi for routing and middleware. Handlers should still use plain `net/http` request and response types.

## Resources

### Raw Files

```text
POST /api/v1/raw-files
GET  /api/v1/raw-files
GET  /api/v1/raw-files/{rawFileId}
```

`POST /api/v1/raw-files` accepts multipart upload first.

Request fields:

```text
file
source_kind      apple_health_export | gpx | fit | tcx | strava_export
uploaded_via     web | telegram | cli | ios_bridge
```

The server computes SHA-256, stores the bytes unchanged, and creates a `tb_raw_files` row.

### Imports

```text
POST /api/v1/imports
GET  /api/v1/imports
GET  /api/v1/imports/{importId}
```

`POST /api/v1/imports` creates a parsing job for a raw file.

Request body:

```json
{
  "raw_file_id": "raw_019f...",
  "parser_kind": "apple_health_export"
}
```

Reprocessing is modeled as another import job for the same raw file, not as mutation of an old job.

Import jobs are persisted jobs. `iroha-server` enqueues them into the durable Postgres-backed queue and the separate `iroha-job` process claims and executes them. The server and worker must share the
configured raw-file data directory.

Current behavior:

```text
POST /api/v1/imports
  -> creates queued import job
  -> returns 202 Accepted
  -> iroha-job claims the persisted job
  -> worker parses and reconciles the raw evidence
```

Queue execution is lease-based: abandoned running jobs are reclaimed after the worker lease timeout, and retryable provider errors may supply their own `Retry-After` delay. Connector sync cursors are
checkpointed per snapshot and are retained when a page fails, so a retry resumes from the failed page.

Response shape:

```json
{
  "id": "imp_019f...",
  "raw_file_id": "raw_019f...",
  "status": "completed",
  "parser_kind": "apple_health_export",
  "parser_version": "dev",
  "started_at": "2026-07-07T00:00:00Z",
  "finished_at": "2026-07-07T00:00:01Z",
  "created_at": "2026-07-07T00:00:00Z"
}
```

### Activities

```text
GET   /api/v1/activities
GET   /api/v1/activities/summary
GET   /api/v1/activities/routes
GET   /api/v1/activities/{activityId}
GET   /api/v1/activities/{activityId}/route
GET   /api/v1/activities/{activityId}/samplings
GET   /api/v1/activities/{activityId}/laps
```

Activity reads serve private canonical data. `summary` and `routes` back the dashboard/activities aggregate widgets; both reuse `apps/iroha-server/pkg/publicexport`'s sanitized query logic even though
this is a private route — the aggregates it builds never carried private fields to begin with.

### Deferred resources

Gear, privacy-zone management, published-activity mutation, and activity mutation are roadmap items. They are not currently registered routes and are intentionally excluded from the active API
contract.

There is no separate public-facing HTTP surface for these sanitized projections (the previous `/public/v1` was removed — it was never actually exposed to the internet). The replacement is implemented
as a standalone export built on `publicexport` that produces a static snapshot for a separate GitHub Pages site instead of a second live API — see
[roadmap Milestone 7](roadmap.md#milestone-7-privacy-and-publishing).

The following routes remain planned and are not part of the active contract:

```text
GET    /api/v1/gear
POST   /api/v1/gear
PATCH  /api/v1/gear/{gearId}
POST   /api/v1/activities/{activityId}/gear
DELETE /api/v1/activities/{activityId}/gear/{gearId}

GET    /api/v1/privacy-zones
POST   /api/v1/privacy-zones
PATCH  /api/v1/privacy-zones/{privacyZoneId}
DELETE /api/v1/privacy-zones/{privacyZoneId}

POST   /api/v1/published-tb_activities
GET    /api/v1/published-tb_activities/{publishedActivityId}
DELETE /api/v1/published-tb_activities/{publishedActivityId}
```

## Upload Flow

MVP direct upload:

```text
client
  -> POST /api/v1/raw-files
  -> POST /api/v1/imports
  -> poll GET /api/v1/imports/{id}
  -> open GET /api/v1/activities/{id}
```

Later large-file upload:

```text
client
  -> POST /api/v1/raw-files/upload-intents
  -> PUT bytes to short-lived upload URL
  -> POST /api/v1/imports
```

The large-file flow is deferred until direct multipart upload becomes painful.

## External Telegram Bot Boundary

The personal Telegram bot is an external upload client, not an in-repo component and not an importer. It pushes raw bytes plus metadata and lets iroha-server own all parsing and dedupe.

```text
Telegram document
  -> bot validates sender
  -> bot downloads file if size allows
  -> bot POSTs to /api/v1/raw-files with uploaded_via=telegram
  -> bot POSTs to /api/v1/imports
  -> bot polls import status
```

Normal Telegram Bot API file downloads may be too small for full Apple Health exports. For large exports, use web upload or a local Telegram Bot API server.

### Upload Contract

The private API is unauthenticated (see [Auth](#auth)); the bot only needs network access to `iroha-server`.

Step 1 — push the raw file as multipart form data:

```text
POST /api/v1/raw-files
Content-Type: multipart/form-data

file          the raw bytes (e.g. export.zip)
source_kind   apple_health_export | gpx | fit | tcx | strava_export
uploaded_via  telegram
```

Response (`201 Created`, or `200 OK` with `"duplicate": true` when the sha256 already exists):

```json
{
  "id": "raw_019f...",
  "sha256": "b1946ac9...",
  "original_filename": "export.zip",
  "content_type": "application/zip",
  "size_bytes": 20480,
  "source_kind": "apple_health_export",
  "uploaded_via": "telegram",
  "created_at": "2026-07-07T00:00:00Z"
}
```

Step 2 — create the import job for that raw file:

```text
POST /api/v1/imports
Content-Type: application/json
```

```json
{
  "raw_file_id": "raw_019f...",
  "parser_kind": "apple_health_export"
}
```

Response (`202 Accepted`):

```json
{
  "id": "imp_019f...",
  "raw_file_id": "raw_019f...",
  "status": "queued",
  "parser_kind": "apple_health_export",
  "parser_version": "dev",
  "created_at": "2026-07-07T00:00:00Z"
}
```

Step 3 — poll import status until it reaches `completed` or `failed`:

```text
GET /api/v1/imports/{importId}
```

```json
{
  "id": "imp_019f...",
  "raw_file_id": "raw_019f...",
  "status": "completed",
  "parser_kind": "apple_health_export",
  "parser_version": "dev",
  "started_at": "2026-07-07T00:00:00Z",
  "finished_at": "2026-07-07T00:00:01Z",
  "created_at": "2026-07-07T00:00:00Z"
}
```

`error_message` is present only on `failed` jobs. The `id`, `status`, and `raw_file_id` fields are the stable contract an external client depends on.

## Auth

`/api/v1` and `/healthz` are the only surfaces this process serves, and both are unauthenticated. iroha is a single-user personal deployment (private LAN/NAS); the network boundary is the security
control, not an application-level credential. Do not expose `iroha-server` to an untrusted network.

Per-IP rate limiting still applies to `/api/v1` as a basic abuse guard; see [HTTP hardening](#http-hardening).

## HTTP hardening

- Configured private origins may use `GET`, `POST`, and `OPTIONS` with `Accept` and `Content-Type` headers.
- JSON request bodies are limited to 1 MiB and reject unknown fields and trailing JSON values. Multipart raw-file uploads use the separate configured upload limit.
- The server applies a 10-second header timeout, a 15-minute request-read timeout for large uploads, a 2-minute write/idle timeout, and a 1 MiB maximum header size.
- Structured access logs include the request ID, route, status, duration, and response size. Request bodies are not logged.

## Configuration

Use TOML config with environment variable overrides.

Default lookup:

```text
./iroha.toml
```

Example:

```toml
[server]
addr = "127.0.0.1:8080"

[database]
url = "postgres://iroha:iroha_dev@localhost:5432/iroha"

[storage]
data_dir = ".iroha-data"
```

Environment variables override TOML:

```text
IROHA_SERVER_ADDR
IROHA_DATABASE_URL
IROHA_DATA_DIR
```
