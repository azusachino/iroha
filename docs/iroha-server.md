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
source_kind      apple_health_export | health_auto_export | gpx | fit | tcx | strava_export
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

### Automatic Apple Health intake

```text
POST /api/v1/intake/health
```

Accepts automated Health Auto Export Format v2 JSON payloads up to 10 MiB (`healthIntakeMaxBytes`). The endpoint validates the payload and capture timestamp; the source instance is
`iphone-hae:<credential name>`, taken from the authenticated credential, and bounded date range before storing raw JSON and enqueuing a `health_auto_export` import job.

Every request must carry the intake credential as `Authorization: Bearer <token>`, regardless of network origin. The `requireIntakeCredential` middleware checks it before the handler reads the body.
Each device has its own credential in `tb_intake_credentials`; only a SHA-256 verifier of the `iroha_hae_`-prefixed token is stored, with `last_used_at` and `revoked_at`. Until an active credential
exists the endpoint fails closed with `503 intake_not_provisioned`; a missing, wrong, or revoked token gets `401`. Manage credentials with the operator-only
`iroha-admin intake-token issue [-name <device>]`, `list`, and `revoke <cred_id>`; `issue` prints the token once, and older credentials stay active until revoked.

Intake uses bounded replacement (`bounded_replacement`), not complete-export reconciliation. Each payload's window (HAE sends the previous 7 days of daily metrics and the default range of workouts)
adds or updates evidence for that window without deleting older activities, sleep sessions, or daily metrics outside it. Replaying the same payload is idempotent because the raw-file content hash is
deduplicated.

Import jobs are persisted jobs. `iroha-server` enqueues them into the durable Postgres-backed queue and the separate `iroha-job` process claims and executes them. The server and worker must share the
configured raw-file data directory.

Connector-created imports also expose `sync_run_id`. A media fetch run is marked successful when its raw evidence and child import jobs are durably recorded; child parsing remains a separate outcome
and can be retried from retained evidence.

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

### Connections and agent actions

```text
GET  /api/v1/connections
POST /api/v1/media/matching-decisions
```

`GET /api/v1/connections` is the operational summary for an agent or shell client. Each source instance reports its latest receipt, latest import, evidence-backed coverage, and an explicit
`next_actions` list. A failed import returns a replayable `retry_import` action; configured media providers expose a `sync` action; Apple Health sources expose the bounded intake action. The summary
intentionally reports `unknown` when credentials, cadence, or collection coverage are not evidenced. It is not a human resolution inbox.

Provider conflicts are resolved through the transactional matching-decision endpoint. `attach`, `keep_separate`, and `undo` decisions become durable lineage used by later replays, so an agent can
resolve a conflict without editing the database or waiting for manual intervention.

Read contracts keep status dimensions separate. Briefing sections expose `availability`, `collection`, `operation`, and `freshness`; metric series expose observation coverage separately from
`collection_completeness`; monthly reports expose calendar closure, canonical observation state, and collection completeness. A closed calendar period is not evidence that the source covered it, so
collection remains `unknown` until a source-scoped assertion is available.

### Normalized expense statements

Monthly bank/card data uses a deliberately small normalized CSV contract rather than a provider-specific parser:

```text
POST /api/v1/expenses/statements/preview
POST /api/v1/expenses/statements
```

Both endpoints accept a JSON body with `manifest` and `csv`:

```json
{
  "manifest": {
    "account_key": "card-main",
    "source_kind": "bank_csv",
    "statement_ref": "2026-08",
    "period_from": "2026-08-01",
    "period_to": "2026-09-01",
    "completeness": "complete",
    "revision": 1
  },
  "csv": "transaction_id,occurred_on,currency,amount_minor,kind,category,merchant,note,original_transaction_ref\n..."
}
```

The required columns are `transaction_id`, `occurred_on`, `currency`, `amount_minor`, `kind`, `category`, and `merchant`. `note` and `original_transaction_ref` are optional. Amounts are positive minor
units; `kind` is `expense` or `refund`, and transfers are rejected. A refund may omit its original transaction reference.

The preview validates every row and writes nothing. Import identity is `(account_key, source_kind, transaction_id)`; an exact revision and CSV hash replay is idempotent, while a changed revision
updates only that statement lineage. `partial` statements never delete omitted rows. `complete` statements tombstone omitted rows only inside the declared account/source/period scope. Manual expenses
outside that lineage are not deleted or overwritten. Statement revisions and row tombstones remain in the fresh SQLx schema for audit and replay.

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

The local client exposes the same agent-facing recovery and resolution contracts without requiring database access:

```text
uv run python scripts/iroha_cli.py connection list
uv run python scripts/iroha_cli.py connection action /api/v1/imports --input retry.json
uv run python scripts/iroha_cli.py media-write decide bangumi <external-id> <media-id> attach
```

The `connection action` path must come from the server's `next_actions` response and is restricted to `/api/v1/`. Public publishing remains a separate sanitized projection: `make export-public` or
`make public-site-build` uses `iroha-export-public`, never the private API response cache or expense/report records. The export validator and atomic directory swap preserve the previous public
snapshot if generation fails.

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

The private API requires an owner session (see [Auth](#auth)). A non-browser client has no supported way to obtain one; ADR-0008 requires a separately approved design for it.

Step 1 — push the raw file as multipart form data:

```text
POST /api/v1/raw-files
Content-Type: multipart/form-data

file          the raw bytes (e.g. export.zip)
source_kind   apple_health_export | health_auto_export | gpx | fit | tcx | strava_export
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

Iroha has one owner account (ADR-0008); tailnet membership is not identity. Every `/api/v1` route requires an owner session except:

- `GET /api/v1/auth/session`, `POST /api/v1/auth/setup`, and `POST /api/v1/auth/login`;
- `POST /api/v1/intake/health`, which requires its own intake credential (see [health intake](#automatic-apple-health-intake));
- `/healthz` and `/readyz`, which sit outside `/api/v1`.

**Setup.** On a database with no owner, the web app shows a one-time setup screen that creates the owner and logs in. The single-owner unique index on `tb_users` makes any later
`POST /api/v1/auth/setup` return `409`. Setup must stay reachable only through the tailnet ingress.

**Sessions.** Setup and login set the `iroha_session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`, 30-day absolute expiry). The server stores only the SHA-256 of the cookie value (`tb_sessions`), so
logout, expiry, and password reset revoke sessions server-side. Passwords are stored as Argon2id hashes (`tb_user_passwords`). Setup and login accept only `application/json`.

**CSRF.** State-changing requests must send the session's CSRF token in `X-CSRF-Token`; `GET /api/v1/auth/session` returns it to the web app, which keeps it in memory only. A missing or wrong token
gets `403 csrf_failed`.

**Passkeys.** Set `IROHA_WEBAUTHN_RP_ID` (the site's host name, e.g. `iroha.example.com`) and `IROHA_WEBAUTHN_ORIGINS` (comma-separated exact origins, e.g. `https://iroha.example.com`) to enable them;
unset, passkey routes answer `503 passkeys_disabled` and the UI hides them. The owner adds, renames, and removes passkeys under **Account settings → Security**; adding or removing needs the password
re-confirmed within the last 5 minutes (`POST /api/v1/account/reauth`). Sign-in is username-less (discoverable credentials, user verification required). Only public-key credential state is stored
(`tb_passkeys`); ceremony challenges are single-use, in memory, bound to an `HttpOnly` `SameSite=Strict` cookie, and expire after 5 minutes. The library checks origin and RP ID against the configured
values, never forwarded headers.

**HAE tokens.** The owner issues and revokes intake tokens on the Admin page (`/api/v1/admin/intake-credentials`); the operator command `iroha-admin intake-token` does the same from the server
container.

**Admin operations.** `GET /api/v1/admin/system` reports parser version, schema migration level, database and raw-file storage size, cache backend, and timezone. `GET /api/v1/admin/schedules`,
`PATCH /api/v1/admin/schedules/{kind}` (`{"enabled": bool}`), and `POST /api/v1/admin/schedules/{kind}/run` list, pause or resume, and trigger recurring jobs; run-now makes the schedule due, and the
worker enqueues it on its next poll. `POST /api/v1/jobs/{id}/retry` queues a fresh copy of a failed or canceled job (the original row stays as history), and `POST /api/v1/jobs/{id}/cancel` stops a
queued job.

**Password reset.** There is no in-app recovery. The operator-only break-glass reset reads a new password from stdin and revokes every session:

```bash
kubectl -n harus-core exec -i deploy/iroha-server -- iroha-admin password reset
```

**Rate limits.** Limits are per client: 6000 requests/minute across `/api/v1`, 10/minute for login and setup, and 30/minute for HAE intake before the credential is checked. After it, each intake
credential gets 120 requests/hour and 512 MiB/day, which caps what a leaked token can write. The client is the socket peer, unless that peer is inside `IROHA_TRUSTED_PROXY_CIDRS` (comma-separated
CIDRs); then the `Cf-Connecting-Ip` header set by Cloudflare is used instead. Only list proxies that strip or overwrite that header for everything they forward.

**Public exposure.** Only `POST /api/v1/intake/health` may be public (ADR-0008 §3); the deployment routes nothing else on the public hostname. Setup and login stay on the tailnet. Passkeys are not
implemented yet.

Per-IP rate limiting still applies to `/api/v1` as a basic abuse guard; see [HTTP hardening](#http-hardening).

## HTTP hardening

- Configured private origins may use `GET`, `POST`, and `OPTIONS` with `Accept` and `Content-Type` headers.
- JSON request bodies are limited to 1 MiB and reject unknown fields and trailing JSON values. Multipart raw-file uploads use the separate configured upload limit.
- The server applies a 10-second header timeout, a 15-minute request-read timeout for large uploads, a 2-minute write/idle timeout, and a 1 MiB maximum header size.
- Structured access logs include the request ID, route, status, duration, and response size. Request bodies are not logged.

## Configuration

Use TOML config with environment variable overrides.

When `IROHA_ANILIST_USERNAME` or `IROHA_BANGUMI_USERNAME` is configured, `iroha-job` creates one enabled daily sync schedule for that provider. Override the cadence with `IROHA_ANILIST_SYNC_INTERVAL`
or `IROHA_BANGUMI_SYNC_INTERVAL` (Go duration such as `12h`); set either to `off` to disable its default schedule. The schedules are independent: neither provider silently wins a media conflict;
explicit matching decisions remain authoritative. Concurrent runs for one connector are rejected, while the other connector remains independent.

Default lookup:

```text
./iroha.toml
```

Example:

```toml
[server]
addr = "127.0.0.1:8080"
# IANA timezone used for calendar scopes and timestamp bucketing.
timezone = "Asia/Tokyo"

[database]
url = "postgres://iroha:iroha_dev@localhost:5432/iroha"

[storage]
data_dir = ".iroha-data"
```

Environment variables override TOML:

```text
IROHA_SERVER_ADDR
IROHA_TIMEZONE
IROHA_DATABASE_URL
IROHA_DATA_DIR
```
