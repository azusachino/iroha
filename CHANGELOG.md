# Changelog

All notable changes to this project are documented in this file.

The format loosely follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project does not yet follow strict semantic versioning guarantees — pre-1.0 releases may change the API
contract between minor versions.

## [Unreleased]

### Added

- **Private control room** — `/admin` and the front-page Daily to-go lane provide personal task tracking, recent durable-job visibility, and named AniList/Bangumi sync triggers.
- **Release identity** — the root `VERSION` file drives image tags and the frontend's subtle version note; sleep sessions now have a detail endpoint/page as well.

### Removed

- **`/public/v1`.** The sanitized public API surface, and the in-app `/share` page that rendered it, were never actually exposed to the internet — dead weight of an open-CORS route, a second
  rate-limit budget, and six themed frontend variants serving a page nobody could reach. A separate static site, deployable to GitHub Pages and kept fresh by a k3s CronJob (not a self-hosted GitHub
  Actions runner — see [roadmap Milestone 7](docs/roadmap.md#milestone-7-privacy-and-publishing)), takes over that role instead.

### Added

- `GET /api/v1/activities/summary` and `GET /api/v1/activities/routes` — private equivalents of the removed public endpoints. The dashboard and activities pages depended on the public routes directly
  for their own totals/routes-map widgets, not only the removed share page.
- `apps/iroha-server/pkg/publicexport` — the sanitized activity DTO and query logic, extracted out of the HTTP layer so both the new private routes and the static-export CLI below reuse it.
- `apps/iroha-server/cmd/iroha-export-public` — writes a static `summary.json`/`activities.json`/`routes.geojson` snapshot for the public site (`make export-public`, `OUT=...`).
- `apps/iroha-public-site` — a new SvelteKit app (adapter-static, fully prerendered) rendering that snapshot, deployable to GitHub Pages as a project page (`make public-site-build`,
  `BASE_PATH=/iroha`).
- `ops/scripts/export-public-cron.sh` and the `export-public` target in `ops/images/Containerfile.server` — the k3s CronJob container that regenerates and pushes the snapshot from inside the private
  network (the CronJob resource itself lives in harus-k3s). `.github/workflows/public-site.yml` builds and deploys the site on an ordinary GitHub-hosted runner whenever that push lands.
- `make image-server` / `image-job` / `image-db-migrate` / `image-web` / `image-export-public` / `images` — build with Podman and import straight into the local k3s node's containerd store, for the
  `azusachino.icu/iroha-*` local image naming already used elsewhere in the homelab.

### Changed

- GORM query logs are routed through the app's `*slog.Logger` instead of GORM's own ANSI-colored default logger; `iroha-server`/`iroha-job` now emit uniform JSON log lines throughout.
- `ops/local-dev/` split into `ops/local-dev/` (Podman Compose orchestration: compose files, initdb, README) and `ops/images/` (environment-agnostic build definitions: both Containerfiles, Caddyfile,
  migrate-entrypoint.sh) — these were always consumed by both local dev and manual production builds, but living under a folder named "local-dev" obscured that.
- `ops/images/Containerfile.web`'s `PUBLIC_IROHA_API_BASE` build-arg now defaults to empty (same-origin) instead of the local-dev convenience value `http://127.0.0.1:8080`, so images built for k3s no
  longer bake in a value that only makes sense on a developer's machine.
- The `migrate` Containerfile target/image is renamed `db-migrate` for clearer naming in the shared homelab image list.

### Fixed

- A media-import race: two `iroha-job` workers resolving the same `(provider, external_id)` at once could both pass a lookup-then-insert "not found" check and collide on `tb_media_external_refs`'s
  unique constraint, aborting the job transaction. Replaced with `INSERT ... ON CONFLICT (provider, external_id) DO NOTHING` plus a re-fetch.
- The `goose` binary bundled into the `db-migrate` image was built with cgo enabled in the glibc build stage and copied into the alpine/musl final stage, so it couldn't run (`goose: not found`,
  despite the file existing). Built with `CGO_ENABLED=0` instead.

## [0.1.1] — 2026-07-20

### Removed

- **JWT authentication.** `/api/v1` is now unauthenticated, matching the actual deployment model: iroha is a single-user personal project running on a private NAS, and only the (not yet exposed)
  `/public/v1` share surface is ever meant to be reachable from outside that network. The JWT layer was always a self-signed static credential standing in for network-level access control — it added a
  secret-provisioning step (`IROHA_JWT_SECRET`, token minting) without changing who could actually reach the API. Removed `golang-jwt/jwt/v5`, the `AuthConfig`/`IROHA_LOCAL_NO_AUTH`/`IROHA_JWT_*`
  config surface, and the web client's `PUBLIC_IROHA_API_TOKEN` build argument. Rate limiting is unchanged and still guards both `/api/v1` and `/public/v1`.

## [0.1.0] — 2026-07-20

First tagged release. Iroha owns personal running/fitness, sleep, daily activity, and media-consumption history end to end: raw exports in, canonical Postgres/PostGIS facts out, private and
sanitized-public read surfaces on top.

### Added

- **Import core** — raw-file archive with content-hash dedupe, a durable `tb_import_jobs` lifecycle, and a Postgres-backed `tb_jobs` queue (`FOR UPDATE SKIP LOCKED`) consumed by the standalone
  `iroha-job` worker.
- **Apple Health domains** — activities (workouts, routes, laps, per-sample streams), sleep sessions and stage segments, and daily summaries/metrics (Move/Exercise/Stand rings, steps, distance,
  flights, body vitals), all reconciled from a full-export snapshot rather than blindly appended.
- **Media sync** — AniList and Bangumi connectors (`POST /api/v1/media/sync/{connectorId}`) with canonical media items/events, aggregates, and a MAL↔AniList/Bangumi bridge cache.
- **Postgres/PostGIS cache backend** — durable, invalidation-aware cache (`tb_cache_entries`) with Valkey kept as an optional compatibility backend; durable geocode reverse-lookup with retry/backoff
  against Nominatim.
- **Private API v1 contract** — JWT authentication, per-route rate limiting, an OpenAPI v1 spec (`docs/contracts/openapi.yaml`), and a route-inventory test that fails the build if a registered route
  drifts from the contract.
- **Themed Svelte cockpit** — a switchable multi-design frontend (grapher, field journal, atlas, phenology, sound-map, archive, and default themes) sharing one typed theme registry and route renderer.
- **Sanitized public views** — `/public/v1` activity/route/summary projections, always derived, never the canonical store.
- **Local dev stack** — Podman Compose profile for Postgres/PostGIS, server, worker, and web behind a Caddy edge container, with `scripts/dev_stack.py` owning lifecycle and migrations.
- **k3s deploy readiness** — a standalone `migrate` container image (goose CLI + bundled migrations) for running as a Job/initContainer, non-root `server`/`job`/`migrate` containers, bounded-retry
  Postgres connect on startup, and graceful shutdown (SIGTERM-aware drain) in `iroha-server`.

### Changed

- Cache backend defaults to Postgres; Valkey is opt-in via `IROHA_CACHE_BACKEND`.
- `rawfiles` moved from `iroha-server` into the shared `iroha-runtime` module so both the API and the worker own the same file-storage boundary.

### Fixed

- Geocode retry storms now back off instead of hammering Nominatim on rate-limit responses.
- Local stack startup sequencing (dependencies before app containers, migrations before server).

[Unreleased]: https://github.com/azusachino/iroha/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/azusachino/iroha/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/azusachino/iroha/releases/tag/v0.1.0
