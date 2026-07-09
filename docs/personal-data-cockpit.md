# Personal Data Cockpit Model

Iroha should grow from a running importer into a personal data cockpit. The
shape stays stable across modules:

```text
raw evidence and connector snapshots
  -> queued durable jobs
  -> canonical domain records
  -> private views and optional public projections
```

The server remains the boundary for ingestion, privacy, canonicalization, and
read APIs. Frontends, bots, browser extensions, and admin tools should submit
intent or uploads; they should not decide canonical identity or write directly
to domain tables.

## Core Concepts

### Evidence

Evidence is the original material Iroha can reprocess later:

- Apple Health export zips.
- GPX, FIT, TCX, CSV, JSON, and service exports.
- Manual web entries.
- Telegram or share-sheet intent payloads.
- Connector snapshots from services such as AniList, Bangumi, Jellyfin, Komga,
  Audiobookshelf, RSS, or future HealthKit clients.
- Provider enrichment responses from services such as TMDb, Open Library, or
  media metadata APIs.
- User corrections and confirmations.

`tb_raw_files` should keep large immutable files. A generalized
`tb_intake_payloads` table should cover non-file evidence and connector
snapshots.

Suggested first shape:

```text
tb_intake_payloads
  id
  source_kind
  source_actor
  source_event_id
  content_type
  sha256
  size_bytes
  storage_path
  payload_json
  received_at
  parsed_at
  created_at
```

Large payloads should be stored in the filesystem and referenced by
`storage_path`. Small manual or webhook payloads may also keep `payload_json`
for quick inspection. Either way, parser improvements should not require the
user to re-enter history.

The first schema lives in
`apps/iroha-server/db/migrations/00006_create_cockpit_jobs.sql`.

### Jobs

Iroha needs a worker process for durable background work:

```text
apps/iroha-server/cmd/iroha-server  HTTP API, auth boundary, upload/intake/read APIs
apps/iroha-job                      queued imports, connector syncs, triggers, projections
```

`iroha-server` accepts evidence and creates jobs. `iroha-job` claims jobs from
Postgres and performs parsing, sync, canonicalization, enrichment, reprocess,
and projection refreshes.

The first implementation should use Postgres-backed jobs, not Redis or a
distributed workflow engine. The database is already the durable source of
truth, is easy to inspect, and fits the personal-scale deployment.

Suggested first shape:

```text
tb_jobs
  id
  kind
  status
  priority
  payload_json
  attempts
  max_attempts
  run_after
  locked_by
  locked_at
  error_message
  started_at
  finished_at
  created_at
  updated_at
```

Job kinds should be explicit. Start with:

```text
apple_import_parse
media_intake_parse
media_connector_sync
health_full_dump_request
projection_refresh
public_summary_refresh
parser_reprocess
```

The same queue should handle one-shot work and scheduled work. A "full dump of
health data" is a user-triggered sync job: it may call a future trusted device
client, wait for a new export payload, or enqueue a parser job once evidence is
available. The trigger is durable even when the actual data source is outside
the server.

### Syncs and Triggers

Jobs are not only parser runs. They also represent planned or requested work:

- Regular connector syncs, for example nightly AniList or Bangumi list pulls.
- Manual "sync now" actions from the web UI.
- User-triggered full dumps, such as "capture a fresh Health export".
- Backfills after adding a new parser or metadata provider.
- Projection refreshes after privacy rules change.
- Retryable enrichment calls.
- Confirmation follow-ups after ambiguous media matches.

Scheduled work can be represented by a small schedule table that enqueues
ordinary `tb_jobs` rows:

```text
tb_job_schedules
  id
  kind
  enabled
  schedule_kind
  schedule_expr
  payload_json
  next_run_at
  last_run_at
  created_at
  updated_at
```

The scheduler should be boring: one worker tick finds due schedules, inserts
jobs, and advances `next_run_at`. Domain logic belongs in job handlers, not in
the scheduler.

### Canonical Records

Canonical tables should stay domain-shaped where the domain needs structure.

Running and fitness should keep first-class activity, route, sampling, lap, and
publication tables. Media should keep works, items, titles, relations, external
refs, consumption events, progress, and notes.

A flexible metadata layer is still useful for long-tail categories, but it
should not flatten high-value domains into a generic key-value store.

Suggested shared primitives:

```text
source
intake_payload
job
entity
event
attribute
external_ref
projection
visibility
```

The rule is:

```text
use first-class tables for stable domains
use flexible attributes for long-tail or evolving metadata
```

### Views

Iroha should provide NocoDB-like affordances without making NocoDB the product:

- table-like private browsing for canonical records
- saved filters and dashboards
- inboxes for unresolved imports, conflicts, and confirmations
- public/private visibility on derived projections
- sanitized public summaries

NocoDB, Directus, Grist, Metabase, Appsmith, or ToolJet can be evaluated as
local admin or exploration tools. They should not replace the server-owned
ingestion and privacy boundary.

## First Vertical After Fitness

The next vertical should be media history because it exercises the generalized
model without requiring new hardware access:

1. Add `tb_intake_payloads`.
2. Add `tb_jobs` and the first `iroha-job` worker binary.
3. Add the minimum media schema from `media-history-research.md`.
4. Add `POST /api/v1/media/intake`.
5. Create `media_intake_parse` jobs from manual web or bot payloads.
6. Build a private media inbox and history list.
7. Add a local NocoDB experiment only as an admin/exploration sidecar.

This keeps Iroha's center of gravity in the product: capture data, preserve
evidence, normalize it, and create useful private or public views.

## First Implementation Slice

The first implementation is intentionally small:

- `tb_intake_payloads` preserves non-file evidence and connector snapshots.
- `tb_jobs` is the durable queue.
- `tb_job_schedules` turns regular syncs and one-shot triggers into queued jobs.
- `internal/jobs` owns enqueue, claim, retry, complete, fail, and due-schedule
  enqueueing.
- `apps/iroha-job` runs the worker loop.
- `make run-job` starts a local worker.

The skeleton should fail unknown job kinds visibly until concrete handlers are
registered. That makes the queue safe to inspect before media, sync, and health
dump handlers exist.
