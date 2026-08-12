# iroha

_iro & hana_ — a personal data cockpit for keeping, understanding, and selectively sharing personal history.

[![CI](https://github.com/azusachino/iroha/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/azusachino/iroha/actions/workflows/ci.yml)
[![Public site](https://github.com/azusachino/iroha/actions/workflows/public-site.yml/badge.svg?branch=main)](https://github.com/azusachino/iroha/actions/workflows/public-site.yml)
[![Release](https://img.shields.io/github/v/release/azusachino/iroha?display_name=tag&sort=semver)](https://github.com/azusachino/iroha/releases)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)

## Two surfaces

The private cockpit stores the complete personal history and runs on a local machine or private k3s/LAN deployment. The public archive is a static GitHub Pages snapshot of deliberately published data;
it has no live API and no private credentials.

| Surface         | Location                                                            | Contents                                                                        |
| --------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Private cockpit | `iroha-server`, `iroha-job`, `iroha-web`                            | Canonical Postgres/PostGIS data, routes, streams, sleep, media, expenses, monthly reports, tasks, and jobs |
| Public archive  | [`azusachino.github.io/iroha`](https://azusachino.github.io/iroha/) | Public activity snapshot with rich detail for every exported activity           |

The exporter runs inside the private deployment and pushes only the static snapshot under `apps/iroha-public-site/static/data/`. See [public-site publishing](docs/public-site-publishing.md) for the
boundary and operator workflow.

## Quick start

Requires [mise](https://mise.jdx.dev/) and Podman.

```sh
mise install
make dev-up
make check
```

Use [`docs/dev-runtime.md`](docs/dev-runtime.md) for local development and [`docs/roadmap.md`](docs/roadmap.md) for planned work.

The v0.4 local client is `scripts/iroha_cli.py`. It sends canonical JSON to the private API for expense create/list/get/update/delete operations and reads monthly reports; receipt OCR remains an external local-agent concern.

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
