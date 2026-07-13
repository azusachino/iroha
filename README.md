# iroha

_iro & hana_ — a personal data cockpit.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](apps/iroha-server/go.mod)
[![Postgres](https://img.shields.io/badge/Postgres-18%20%2F%20PostGIS-336791?logo=postgresql&logoColor=white)](docs/data-model.md)
[![Built with Nix](https://img.shields.io/badge/built%20with-Nix-5277C3?logo=nixos&logoColor=white)](flake.nix)

Iroha lets you own your personal data end to end: keep the **raw exports**, normalize them into a **durable Postgres/PostGIS store**, and publish only **sanitized derived views**. Running and fitness,
sleep, and daily Apple Health activity are the current data domains; the architecture generalizes to reading, watching, and other personal-history sources.

## Architecture

Raw files are canonical evidence. The server stores each upload, creates a durable import job, and the separate `iroha-job` worker parses it into typed domain records. Reconciliation uses stable
source keys and content hashes, so unchanged imports are skipped and parser-version changes purge-and-reprocess derived rows instead of appending duplicates.

```text
raw export
  -> tb_raw_files + stored bytes
  -> tb_import_jobs -> tb_jobs
  -> iroha-job / iroha-server parser
  -> reconcile into Postgres/PostGIS
  -> private API and web cockpit
```

The current canonical domains are:

- `tb_activities` — workouts, routes, laps, and time-series samplings.
- `tb_sleep_sessions` — sleep sessions and stage segments.
- `tb_daily_summaries` + `tb_daily_metrics` — Apple Move/Exercise/Stand rings and cross-source-deduplicated daily steps, distance, and flights.

Private reads are served under `/api/v1/activities`, `/api/v1/sleep`, and `/api/v1/daily`. Sanitized activity and route projections are also available under `/public/v1`; the public surface is a
derived view, never the canonical store.

## Features

- **Raw-file archive** — every import preserves the original evidence (Apple Health zip, GPX); raw files are the canonical source and are deduplicated by content hash.
- **Incremental Apple Health ingestion** — a full export is treated as a complete snapshot and _reconciled_, not blindly appended: stable per-workout source identity, content-hash change detection,
  and idempotent re-import. A parser-version bump triggers a purge-then-repersist _reprocess_ so counts stay stable instead of duplicating.
- **Multiple Apple Health domains** — the import pipeline extracts workouts, sleep sessions, and daily activity. Daily steps, distance, and flights use source-priority interval deduplication; Apple’s
  ActivitySummary rings are persisted as structured daily summaries.
- **High fidelity** — workout routes linked to their workout (not standalone GPX), HR/pace/distance summaries, laps, and per-sample streams (heart rate, running power/speed, stride, energy) parsed
  from ~millions of `Record` rows via streaming.
- **Background jobs + read surfaces** — `iroha-server` owns ingestion and reads, `iroha-job` claims persisted jobs, and the Svelte cockpit consumes the private API. Public activity/route views are
  sanitized projections.
- **PostGIS canonical store** — Strava is a legacy import/export adapter only.

## Tech stack

| Layer    | Choice                                                                        |
| -------- | ----------------------------------------------------------------------------- |
| Services | Go 1.26 (`apps/iroha-server`, `apps/iroha-job`), GORM                         |
| Database | PostgreSQL 18 + PostGIS, [goose](https://github.com/pressly/goose) migrations |
| Web      | Svelte 5 + Vite (`apps/iroha-web`, [bun](https://bun.sh))                     |
| Tooling  | Nix devShell, `make` task runner, `uv` for dev scripts                        |

## Quickstart

Requires [Nix](https://nixos.org/download) with flakes enabled.

```sh
nix develop            # enter the dev shell (all tools come from here)
make db-up             # start Postgres/PostGIS and apply migrations
make check             # fmt-check + vet + tests + web checks
make build             # build server and web

# run the server against the dev database (terminal 1)
IROHA_DATABASE_URL="postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable" \
  go -C apps/iroha-server run ./cmd/iroha-server

# run the persisted import worker in another terminal
make run-job
```

The server is configured via `iroha.toml` and/or environment variables:

| Env var                | Purpose                              | Default                        |
| ---------------------- | ------------------------------------ | ------------------------------ |
| `IROHA_SERVER_ADDR`    | Listen address                       | `127.0.0.1:8080`               |
| `IROHA_DATABASE_URL`   | Postgres DSN                         | local dev DSN                  |
| `IROHA_DATA_DIR`       | Raw-file storage dir                 | `.iroha-data`                  |
| `IROHA_LOCAL_NO_AUTH`  | Disable auth for local dev           | `true`                         |
| `IROHA_IMPORT_TOKEN`   | Bearer token for the upload contract | —                              |
| `IROHA_PARSER_VERSION` | Parser build id; bump to reprocess   | `imports.DefaultParserVersion` |
| `IROHA_ANILIST_USERNAME` | Public AniList username to sync       | —                              |
| `IROHA_ANILIST_TOKEN`  | Optional AniList OAuth token           | —                              |

Smoke-test a real import end to end:

```sh
make smoke-real-import FILE=.iroha-data/imports/your_export.zip
# stronger reconciliation/reprocess checks:
uv run python scripts/real_import_smoke.py .iroha-data/imports/your_export.zip --assert
```

## Documentation

- [MVP v0 Design](docs/mvp-v0-design.md)
- [Iroha Server](docs/iroha-server.md)
- [Import Pipeline](docs/import-pipeline.md)
- [Data Model](docs/data-model.md)
- [Daily Activity Module](docs/activity-module.md)
- [Sleep Module](docs/sleep-module.md)
- [Reading and Watching History Research](docs/media-history-research.md)
- [Development Runtime](docs/dev-runtime.md)
- [Roadmap](docs/roadmap.md)

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for the dev workflow, commit conventions, and code style (oriented from the [Uber Go Style Guide](https://github.com/uber-go/guide)).
Project conventions for humans and agents live in [AGENTS.md](AGENTS.md).

## License

Licensed under the **GNU Affero General Public License v3.0** — see [LICENSE](LICENSE). © 2026 haru.
