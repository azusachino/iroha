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

The server computes SHA-256, stores the bytes unchanged, and creates a `raw_files` row.

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

Import jobs are persisted jobs. MVP execution can be an in-process worker, but job state must live in Postgres so status survives process crashes and future worker extraction.

### Activities

```text
GET   /api/v1/activities
GET   /api/v1/activities/{activityId}
PATCH /api/v1/activities/{activityId}
GET   /api/v1/activities/{activityId}/route
GET   /api/v1/activities/{activityId}/samples
GET   /api/v1/activities/{activityId}/laps
```

Activity reads serve private canonical data.

### Gear

```text
GET    /api/v1/gear
POST   /api/v1/gear
PATCH  /api/v1/gear/{gearId}
POST   /api/v1/activities/{activityId}/gear
DELETE /api/v1/activities/{activityId}/gear/{gearId}
```

Gear is deferred until import and activity detail work.

### Privacy Zones

```text
GET    /api/v1/privacy-zones
POST   /api/v1/privacy-zones
PATCH  /api/v1/privacy-zones/{privacyZoneId}
DELETE /api/v1/privacy-zones/{privacyZoneId}
```

Privacy zones define locations to remove or blur before publishing.

### Published Activities

```text
POST   /api/v1/published-activities
GET    /api/v1/published-activities/{publishedActivityId}
DELETE /api/v1/published-activities/{publishedActivityId}
```

Publishing writes a sanitized projection. Public pages should read this projection, not private activity tables.

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

The personal Telegram bot is an external upload client, not an in-repo component and not an importer.

```text
Telegram document
  -> bot validates sender
  -> bot downloads file if size allows
  -> bot POSTs to /api/v1/raw-files
  -> bot POSTs to /api/v1/imports
  -> bot reports import status
```

Normal Telegram Bot API file downloads may be too small for full Apple Health exports. For large exports, use web upload or a local Telegram Bot API server.

## Auth

MVP auth is intentionally simple:

- no auth for local-only development
- bearer token when external upload clients such as a personal Telegram bot are enabled

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
