# iroha

_iro & hana_ — a personal data cockpit for keeping, understanding, and selectively sharing personal history.

[![CI](https://github.com/azusachino/iroha/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/azusachino/iroha/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/azusachino/iroha?display_name=tag&sort=semver)](https://github.com/azusachino/iroha/releases)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

## Two surfaces

The private cockpit stores personal history and runs on a local machine or private k3s/LAN deployment. The public archive is a separate static client that reads Iroha's validated, sanitized `/public/v1` projection. It has no private credentials and does not proxy the private `/api/v1` API.

| Surface         | Location                                                | Contents                                                                                                   |
| --------------- | ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Private cockpit | `iroha-server`, `iroha-job`, `iroha-web`                | Canonical Postgres/PostGIS data, routes, streams, sleep, media, expenses, monthly reports, tasks, and jobs |
| Public archive  | [`iroha.azusachino.com`](https://iroha.azusachino.com/) | Sanitized public activity projection with rich detail for available activities                             |

The public site is served from the deployment cluster; its Caddy proxy exposes only `/public/v1/*` from `iroha-server`. GitHub Pages at `azusachino.github.io/iroha` is retired and no longer updates. See [public-site publishing](docs/public-site-publishing.md) for the current pipeline, privacy boundary, and operator workflow.

## Quick start

Requires [mise](https://mise.jdx.dev/) and Podman.

```sh
mise install
make dev-up
make check
```

Use [`docs/dev-runtime.md`](docs/dev-runtime.md) for local development, [`ops/k8s/README.md`](ops/k8s/README.md) for Kubernetes deployment, and [`docs/roadmap.md`](docs/roadmap.md) for planned work.

The v0.5 local client is `scripts/iroha_cli.py`. It uploads files into the canonical import pipeline, reads activities, sleep, daily health, media, metrics, and monthly reports, manages expenses,
lists source attention, and applies agent-owned connection or media-matching actions. JSON is preserved by default; receipt OCR remains an external local-agent concern. Run
`uv run python scripts/iroha_cli.py --help` for the exact commands.

The v0.5 runtime cache is a shared, backend-neutral disposable read layer for the private cockpit. Canonical records remain in Postgres; Postgres is also the default cache backend, Valkey is supported
for the k3s compatibility deployment, and `none` disables caching. There is no production process-memory cache and no scheduled aggregate table in this release. The public site remains a separate
static client; it reads the validated sanitized public API projection and does not use the private cache.

v0.5 is a fresh-schema cut-over. The release path replays the complete original raw evidence set into SQLx migrations 1–25; it does not migrate the legacy schema and does not adopt Goose.

After a local k3s rollout, run `make smoke-k3s-cache` for a non-mutating deployment check. It verifies the k3s ConfigMap selects Valkey and that two identical monthly-report reads return the expected
cache hit on the second request.

## Data Ingestion

Iroha supports a two-tier ingestion model for health, fitness, and location data:

1. **Continuous Daily Intake (Health Auto Export)**:
   - Configure [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) on iOS/watchOS to POST Format v2 JSON to `POST /api/v1/intake/health`.
   - Imports summarized daily metrics and sleep, plus workouts with optional GPS routes and heart-rate series. The HAE metric mapping is partial; see the [provider reference](docs/capabilities/providers/health-auto-export.md) for supported names and known gaps.
   - Every request carries a dedicated intake token (`Authorization: Bearer`), issued per device on the Admin page; intake is refused until one is issued.
   - See [Setting up Health Auto Export HTTP Intake](docs/health-auto-export-setup.md).

2. **Historical Bulk Backfill**:
   - Upload full Apple Health `export.zip` archives, Garmin/Wahoo `fit` / `tcx` files, or `gpx` tracks via `POST /api/v1/raw-files` and `POST /api/v1/imports`.
   - See [Import Pipeline](docs/import-pipeline.md).

## References

- [API contract](docs/contracts/openapi.yaml)
- [Data model](docs/data-model.md)
- [Import pipeline](docs/import-pipeline.md)
- [Frontend design contract](docs/frontend-design-contract.md)
- [Theme architecture](docs/frontend-theme-architecture.md)
- [Contributing](CONTRIBUTING.md)
- [Agent/project conventions](AGENTS.md)
- [Changelog](CHANGELOG.md)

`VERSION` is the product release version; releases use matching tags such as `v0.3.1`. `IROHA_PARSER_VERSION` is separate and changes only when import semantics require reprocessing.

## License

Licensed under the **GNU Affero General Public License v3.0** — see [LICENSE](LICENSE).
