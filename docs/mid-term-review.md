# Mid-term architecture review

Date: 2026-07-13

Scope: the merged activity, sleep, daily-activity, body-vitals, cockpit, and durable-worker work currently reachable from `feat/iroha-media-sync`.

## Executive assessment

Iroha has crossed the MVP boundary successfully. The durable product shape is now:

```text
raw file on shared storage
  -> persisted import job
  -> parser and source-item reconciliation
  -> canonical Postgres/PostGIS facts
  -> private API and derived public views
  -> Svelte cockpit
```

The core design is coherent enough to continue adding domains. The main mid-term risk is not another domain schema; it is that the runtime and docs still describe the earlier MVP while the code
already assumes a separate worker, Postgres-backed cache state, absolute shared file paths, and multiple Go modules.

The local stack should use fully containerized application services. The checked-in Compose files are now consumed by `podman-compose`, while a small uv-managed runner owns readiness and migrations.

## What is working well

- Raw exports remain canonical evidence. Reprocessing is explicit through the parser version and the purge order protects against stale source-item hashes.
- Apple Health identity is source-derived rather than zip-derived. This is the correct basis for full-snapshot reconciliation.
- Sleep, daily rings, and open-ended daily metrics are separate domain facts, while the generic daily-metric table is reused successfully by body vitals.
- The durable queue is a real Postgres queue: jobs are claimed with `FOR UPDATE SKIP LOCKED`, attempts and retry timing are persisted, and the worker is a separate Go module.
- The API keeps private canonical reads separate from sanitized public views.
- The web app has a clear domain surface and the recent daily aggregation work moved rollup responsibility to the backend.

## Findings and priorities

### P0 — make one runtime contract authoritative

`make db-up`, `make run`, `make run-job`, integration tests, and the real import smoke use Podman for Postgres. `make dev-up` runs the server, worker, and web containers as well.

The next runtime milestone should define one supported local command path:

```text
make dev-up
  -> build or pull db, server, job, and web images
  -> create a shared network and persistent volumes
  -> start migrations once
  -> start server, worker, and web
```

`make db-up` can remain as a database-only compatibility target during the migration, but it should stop being the hidden prerequisite for the full app.

### P0 — make raw storage a first-class runtime contract

The server stores absolute filesystem paths and the worker opens those paths. That is correct for two host processes sharing `IROHA_DATA_DIR`, but it becomes an implicit requirement when both
processes are containers. The server and worker must mount the same named volume at the same path, and the database must never become the owner of raw bytes.

The container milestone must test the actual multipart upload -> queued job -> worker parse path with both services isolated in containers. A green database health check alone is insufficient.

### P1 — synchronize documentation with the shipped architecture

Several docs still describe an in-process worker, the original MVP roadmap, and the earlier PostGIS image. They should be refreshed after the runtime contract is chosen. In particular,
`docs/dev-runtime.md`, `docs/roadmap.md`, and `docs/iroha-server.md` should describe the current queue/worker boundary, not future-state wording that is already false.

### P1 — add missing API-level proof around the newest aggregation boundary

The daily aggregation merge helper now has a unit test, but the HTTP route and the range semantics need DB-backed coverage. The `from`/`to` filters are part of the service shape and must either be
wired into the web client or explicitly removed until supported. This is a correctness issue because the cockpit can otherwise present a range label that does not constrain the query.

### P1 — reduce import cost before adding many more Apple domains

An Apple export currently reopens and streams the large `export.xml` through separate activity, sleep, and daily passes. This is intentionally memory-safe and keeps parsers testable, so it is not an
immediate rewrite target. However, the next domain will make the repeated decompression cost more visible. Keep the current parsers as the correctness baseline, then introduce a unified streaming
dispatcher only after an import-time measurement proves it is needed.

### P2 — clarify the domain-to-evidence ownership model

`tb_apple_source_items` currently carries every Apple-derived item type and uses a global unique `source_key`. This works for the shipped data, but it is a coupling point for future domains and source
providers. Before media sync or another Apple family lands, document the invariant that makes source keys globally unique, or move toward an explicit `(provider, source_kind, source_key)` identity.

### P2 — keep optional infrastructure optional at request time

The cache is now backed by Postgres by default and remains best-effort at request time. Valkey is a compatibility backend rather than a required service; database and raw storage remain hard
dependencies.

## Recommended container target

Use these services for the first fully containerized local profile:

| Service  | Image source                                 | Persistent data                         | Network role                                  |
| -------- | -------------------------------------------- | --------------------------------------- | --------------------------------------------- |
| `db`     | PostGIS image with verified arm64 support    | named Postgres volume                   | internal, optionally host-published for debug |
| `server` | repo `Containerfile.server`                  | shared `iroha-data` volume              | HTTP API                                      |
| `job`    | repo `Containerfile.server` or worker target | same `iroha-data` volume                | queue consumer                                |
| `web`    | repo `Containerfile.web`                     | none                                    | frontend HTTP                                 |

The first implementation should use fixed service names and a private network, with the server configured for `db` service DNS. Host-published ports remain useful for browser and smoke
access. The server and worker must share the same `IROHA_DATA_DIR` mount; this is the critical boundary.

The runner should own only lifecycle mechanics that Apple `container` does not provide declaratively: network/volume creation, image build, ordered startup, health waits, migration execution, logs,
and teardown. It should not duplicate schema or application logic. A future Docker/Compose adapter can consume the same service names and environment contract.

## Proposed sequence

1. Add a runtime contract test/spec for service names, mounts, ports, required environment, and health checks. Keep the current database-only path green.
2. Add server and worker `Containerfile` targets and prove a one-shot worker can process a small fixture through the real HTTP route.
3. Keep Podman Compose as the default local runtime and retain the host-process `make db-up` compatibility path.
4. Run the real Apple Health smoke against the fully containerized server and worker, then update runtime docs and the roadmap.

## Explicit non-goals

- Do not redesign the canonical data model as part of containerization.
- Do not introduce Kubernetes, an external queue, or object storage for local development.
- Do not replace GORM plus SQL migrations with generated models or an API gateway.
- Do not unify all Apple parsing passes until measured import cost justifies the complexity.

## Review conclusion

Proceed with containerization as the next infrastructure epic, but treat it as runtime-contract work. The current domain architecture is ready for that investment; the immediate deliverable is
reproducible lifecycle and real cross-container import evidence, followed by documentation repair and the small missing aggregation/API tests.
