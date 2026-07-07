# MVP v0 Design

## Product Direction

Iroha is not a Strava clone. It is a personal data cockpit where running is the first module.

The product promise for MVP v0 is:

```text
I own my activity history, can reprocess it forever, and can create private or public views from sanitized derived data.
```

Running is the first vertical because it exercises the core system:

- importing sensitive personal data
- preserving raw source files
- normalizing time-series and geospatial data
- rendering rich private views
- generating privacy-safe public projections

Broader personal data integrations such as photos, location history, notes, and expenses are intentionally deferred until the running module proves the architecture.

## Core Principles

### Raw Files Are Canonical Evidence

Every original upload is stored unchanged. Parser bugs are fixed by reprocessing raw files, not by asking the user to upload history again.

### Imports Are Repeatable

An import is a parsing attempt against a raw file. A raw file may have many import jobs over time as parsers improve.

### Activities Are Canonical Domain Objects

Activities, route points, samples, laps, notes, and gear are normalized into Postgres/PostGIS for query and presentation.

### Public Data Is A Sanitized Projection

Public pages must not read directly from private activity tables. A publish action writes a sanitized payload that is safe to serve independently.

## MVP v0 Scope

In scope:

- Web upload for raw fitness files.
- Raw archive storage on local filesystem.
- Apple Health export zip ingestion.
- GPX import.
- Initial FIT and TCX parser hooks, even if parser coverage starts narrow.
- Activity list.
- Activity detail page API data: summary, route, samples, laps.
- Dedupe by raw file hash and source activity references.
- Import status tracking.
- External upload client contract for an existing personal Telegram bot.

Out of scope:

- Native iOS HealthKit sync app.
- Background iOS sync.
- Strava API integration.
- Social features.
- Training plans.
- Route clustering.
- Yearbook.
- Immich, Dawarich, Obsidian, expenses, and other non-running modules.
- Full public publishing UI.
- Private frontend implementation before milestones 1-3 are complete.

## Architecture

```text
iPhone Health export / GPX / FIT / TCX / Strava archive / Telegram document
  -> iroha-server /api/v1/raw-files
  -> tb_raw_files row + immutable filesystem blob
  -> tb_import_jobs row
  -> parser creates ParsedActivity records
  -> canonical activity tables
  -> private UI and future sanitized public projections
```

## API Style

The server API is REST over `/api/v1`.

Top-level resources:

- `/api/v1/raw-files`
- `/api/v1/imports`
- `/api/v1/activities`
- `/api/v1/gear`
- `/api/v1/privacy-zones`
- `/api/v1/published-tb_activities`

The MVP can combine upload and import creation for convenience, but the internal model must still create separate `tb_raw_files` and `tb_import_jobs` records.

## Technology Choices

Preferred MVP stack:

- Backend: Go
- HTTP router: Chi
- Database: PostgreSQL + PostGIS
- Storage: local filesystem
- Frontend: SvelteKit
- Maps: MapLibre
- Charts: ECharts, Observable Plot, or uPlot
- Auth: no auth for local MVP; add bearer tokens when external upload clients are enabled
- Config: TOML file with environment variable overrides
- IDs: UUIDv7 stored in Postgres UUID columns, exposed as prefixed API IDs
- Universal tool manager: Nix flake
- Local scripts: `uv` inside the Nix-managed environment
- Go layout: `apps/iroha-server` as an in-repo Go module; add `go.work` when multiple Go modules exist
- macOS container runtime: Apple `container`, with capability checks before assuming Docker Compose parity

Milestones 1-3 are API-only. Build the private frontend after upload, import job lifecycle, and first parser path are working.

The first implementation should avoid abstractions that only make sense for multi-user SaaS. Iroha starts as a personal system.

## First Useful Experience

The first end-to-end success is:

1. Export Apple Health data from iPhone.
2. Upload `export.zip` to iroha.
3. Iroha stores the original zip unchanged.
4. Iroha parses workouts and route GPX files.
5. Iroha shows imported tb_activities.
6. Opening one activity shows route, summary metrics, and samples.

That path validates the product without requiring an Apple Developer Program subscription.
