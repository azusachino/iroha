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
GET   /api/v1/activities/{activityId}
GET   /api/v1/activities/{activityId}/route
GET   /api/v1/activities/{activityId}/samplings
GET   /api/v1/activities/{activityId}/laps
```

Activity reads serve private canonical data.

### Deferred resources

Gear, privacy-zone management, published-activity mutation, and activity mutation are roadmap items. They are not currently registered routes and are intentionally excluded from the active API
contract.

Public pages read the existing sanitized `/public/v1` projections rather than these deferred mutation resources.

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

All write requests carry the bearer token when auth is enabled (see [Auth](#auth)):

```text
Authorization: Bearer <import_token>
```

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

MVP auth is intentionally simple:

- no auth for local-only development (`local_no_auth = true`)
- bearer token when external upload clients such as a personal Telegram bot are enabled (`local_no_auth = false`, `import_token = "<secret>"`)

Only the write endpoints are guarded:

```text
POST /api/v1/raw-files   requires Authorization: Bearer <import_token>
POST /api/v1/imports     requires Authorization: Bearer <import_token>
```

The read endpoints (`GET /api/v1/raw-files`, `GET /api/v1/imports`, `GET /api/v1/activities...`) are not guarded. A missing or wrong bearer token on a guarded endpoint returns `401 Unauthorized`:

```json
{ "error": "unauthorized" }
```

OIDC can be added later if iroha becomes network-exposed or multi-device access becomes awkward.

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

[auth]
local_no_auth = true
import_token = ""
```

Environment variables override TOML:

```text
IROHA_SERVER_ADDR
IROHA_DATABASE_URL
IROHA_DATA_DIR
IROHA_LOCAL_NO_AUTH
IROHA_IMPORT_TOKEN
```
