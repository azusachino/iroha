# API v1 contract gap matrix

Status: v0.4.1 release-candidate contract inventory

This matrix records the current contract evidence for the private `/api/v1` surface. The static public export is documented separately; this document does not establish a released compatibility
promise or introduce an `/api/v2` policy.

## Current surfaces

The source of truth for the route inventory is `apps/iroha-server/pkg/httpapi/server.go`.

| Surface          | Current routes                                                                                            | Result  |
| ---------------- | --------------------------------------------------------------------------------------------------------- | ------- |
| Health           | `GET /healthz`                                                                                            | PRESENT |
| Private briefing | `GET /api/v1/briefing`                                                                                    | PRESENT |
| Raw files        | `POST /api/v1/raw-files`, `GET /api/v1/raw-files`, `GET /api/v1/raw-files/{rawFileId}`                    | PRESENT |
| Imports          | `POST /api/v1/imports`, `GET /api/v1/imports`, `GET /api/v1/imports/{importId}`                           | PRESENT |
| Activities       | `GET /api/v1/activities`, detail, route, samplings, laps                                                  | PRESENT |
| Sleep            | list, aggregates, and segments under `/api/v1/sleep`                                                      | PRESENT |
| Daily            | list and aggregates under `/api/v1/daily`                                                                 | PRESENT |
| Media            | list, detail, events, aggregates, and `POST /api/v1/media/sync/{connectorId}` under `/api/v1/media`       | PRESENT |
| Metrics          | catalog, definition, and deterministic series under `/api/v1/metrics`                                     | PRESENT |
| Expenses         | create, list, detail, replace, and tombstone-delete under `/api/v1/expenses`                              | PRESENT |
| Reports          | monthly envelope and twelve-month series under `/api/v1/reports`                                          | PRESENT |
| Control room     | tasks, durable jobs, and allowlisted actions under `/api/v1/tasks`, `/api/v1/jobs`, and `/api/v1/actions` | PRESENT |
| Public views     | sanitized JSON/GeoJSON export consumed by the static GitHub Pages site                                    | PRESENT |

Gear, privacy-zone management, published-activity mutation, and activity mutation are roadmap items, not shipped routes.

## Contract evidence

| Area                      | Status          | Evidence and gap                                                                                                                                                                                        |
| ------------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Route inventory           | PRESENT         | Router is explicit in `pkg/httpapi/server.go`; `make contract-check` guards the active inventory.                                                                                                       |
| Request and response DTOs | PRESENT         | Handler-local Go DTOs and the v0.4 OpenAPI schemas define the current JSON output; integration fixtures cover representative resource families.                                                         |
| OpenAPI                   | PRESENT         | `docs/contracts/openapi.yaml` is parsed in `make contract-check`; its method/path inventory must equal the registered Chi routes.                                                                       |
| IDs and timestamps        | PRESENT         | IDs are opaque prefixed strings, instants use RFC3339, and calendar values use `YYYY-MM-DD`; the decisions and schemas encode these rules.                                                              |
| Pagination                | PRESENT         | List responses use `items`, `next_cursor`, and `has_more`; explicit limits are 1..100 and invalid cursors/limits return the common error schema.                                                        |
| Errors                    | PRESENT         | Errors use `{ "code": "...", "message": "...", "request_id": "..." }`; the common schema and web client preserve these fields.                                                                          |
| Private/public projection | PRESENT         | Public DTOs are separate from private activity DTOs; the static export remains separate from the private API.                                                                                           |
| Authentication            | N/A             | `/api/v1` is intentionally unauthenticated; the deployment's network boundary is the security control (see `api-v1-decisions.md#authentication`).                                                       |
| Rate limiting             | PRESENT/PARTIAL | IP-based limits exist for private and geocode routes. The `429` response and `Retry-After` contract are covered but still need broader fixture coverage.                                                |
| Frontend compatibility    | PRESENT         | The Svelte client consumes private reads directly; no auth headers are required.                                                                                                                        |
| Runtime contract          | PRESENT         | `make release-candidate` exercises isolated migrations, the upload/import/worker path, API assertions, production build, and seeded browser checks.                                                     |
| Release stability         | CANDIDATE       | v0.4.1 cache/report implementation passes `make check`, `make validate`, the release-candidate gate, public-site checks/build, and local k3s rollout; owner release review and remote merge/tag remain. |

## Required next decisions

1. Keep the OpenAPI artifact, route-inventory test, and example manifest synchronized when handlers change.
2. Add full response-schema validation when a dependency policy for JSON Schema/OpenAPI tooling is chosen.
3. Rehearse the real upload-to-worker workflow before each production release; the v0.4.1 release-candidate rehearsal is recorded, not a permanent compatibility guarantee.
4. Complete the owner release review before tagging v0.4.1.
