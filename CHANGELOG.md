# Changelog

All notable changes to this project are documented in this file.

The format loosely follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project does not yet
follow strict semantic versioning guarantees — pre-1.0 releases may change the API contract between minor versions.

## [Unreleased]

## [0.1.1] — 2026-07-20

### Removed

- **JWT authentication.** `/api/v1` is now unauthenticated, matching the actual deployment model: iroha is a single-user personal project running on a private NAS, and only the (not yet exposed)
  `/public/v1` share surface is ever meant to be reachable from outside that network. The JWT layer was always a self-signed static credential standing in for network-level access control — it added
  a secret-provisioning step (`IROHA_JWT_SECRET`, token minting) without changing who could actually reach the API. Removed `golang-jwt/jwt/v5`, the `AuthConfig`/`IROHA_LOCAL_NO_AUTH`/`IROHA_JWT_*`
  config surface, and the web client's `PUBLIC_IROHA_API_TOKEN` build argument. Rate limiting is unchanged and still guards both `/api/v1` and `/public/v1`.

## [0.1.0] — 2026-07-20

First tagged release. Iroha owns personal running/fitness, sleep, daily activity, and media-consumption history
end to end: raw exports in, canonical Postgres/PostGIS facts out, private and sanitized-public read surfaces on top.

### Added

- **Import core** — raw-file archive with content-hash dedupe, a durable `tb_import_jobs` lifecycle, and a
  Postgres-backed `tb_jobs` queue (`FOR UPDATE SKIP LOCKED`) consumed by the standalone `iroha-job` worker.
- **Apple Health domains** — activities (workouts, routes, laps, per-sample streams), sleep sessions and stage
  segments, and daily summaries/metrics (Move/Exercise/Stand rings, steps, distance, flights, body vitals), all
  reconciled from a full-export snapshot rather than blindly appended.
- **Media sync** — AniList and Bangumi connectors (`POST /api/v1/media/sync/{connectorId}`) with canonical media
  items/events, aggregates, and a MAL↔AniList/Bangumi bridge cache.
- **Postgres/PostGIS cache backend** — durable, invalidation-aware cache (`tb_cache_entries`) with Valkey kept as an
  optional compatibility backend; durable geocode reverse-lookup with retry/backoff against Nominatim.
- **Private API v1 contract** — JWT authentication, per-route rate limiting, an OpenAPI v1 spec
  (`docs/contracts/openapi.yaml`), and a route-inventory test that fails the build if a registered route drifts
  from the contract.
- **Themed Svelte cockpit** — a switchable multi-design frontend (grapher, field journal, atlas, phenology,
  sound-map, archive, and default themes) sharing one typed theme registry and route renderer.
- **Sanitized public views** — `/public/v1` activity/route/summary projections, always derived, never the
  canonical store.
- **Local dev stack** — Podman Compose profile for Postgres/PostGIS, server, worker, and web behind a Caddy edge
  container, with `scripts/dev_stack.py` owning lifecycle and migrations.
- **k3s deploy readiness** — a standalone `migrate` container image (goose CLI + bundled migrations) for running as
  a Job/initContainer, non-root `server`/`job`/`migrate` containers, bounded-retry Postgres connect on startup, and
  graceful shutdown (SIGTERM-aware drain) in `iroha-server`.

### Changed

- Cache backend defaults to Postgres; Valkey is opt-in via `IROHA_CACHE_BACKEND`.
- `rawfiles` moved from `iroha-server` into the shared `iroha-runtime` module so both the API and the worker own
  the same file-storage boundary.

### Fixed

- Geocode retry storms now back off instead of hammering Nominatim on rate-limit responses.
- Local stack startup sequencing (dependencies before app containers, migrations before server).

[Unreleased]: https://github.com/azusachino/iroha/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/azusachino/iroha/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/azusachino/iroha/releases/tag/v0.1.0
