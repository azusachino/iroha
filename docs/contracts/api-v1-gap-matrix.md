# API v1 contract gap matrix

Status: active development, v0.1.4 release candidate

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
| Control room     | tasks, durable jobs, and allowlisted actions under `/api/v1/tasks`, `/api/v1/jobs`, and `/api/v1/actions` | PRESENT |
| Public views     | sanitized JSON/GeoJSON export consumed by the static GitHub Pages site                                    | PRESENT |

Gear, privacy-zone management, published-activity mutation, and activity mutation are roadmap items, not shipped routes.

## Contract evidence

| Area                      | Status          | Evidence and gap                                                                                                                                                                                            |
| ------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Route inventory           | PRESENT         | Router is explicit in `pkg/httpapi/server.go`; `make contract-check` guards the active inventory.                                                                                                           |
| Request and response DTOs | PRESENT         | Handler-local Go DTOs define the current JSON output. They are not yet an external contract artifact.                                                                                                       |
| OpenAPI                   | PRESENT         | `docs/contracts/openapi.yaml` is checked in and the active route inventory is tested.                                                                                                                       |
| IDs and timestamps        | PARTIAL         | IDs are encoded consistently and timestamps use Go JSON time encoding; the client-facing rules are undocumented.                                                                                            |
| Pagination                | PARTIAL         | List responses use `items`, `next_cursor`, and `has_more`; cursor opacity, limits, and invalid-cursor behavior are undocumented.                                                                            |
| Errors                    | PARTIAL         | Errors currently use `{ "error": "..." }`; error codes, request correlation, and the complete status matrix are not defined.                                                                                |
| Private/public projection | PRESENT         | Public DTOs are separate from private activity DTOs; the static export remains separate from the private API.                                                                                               |
| Authentication            | N/A             | `/api/v1` is intentionally unauthenticated; the deployment's network boundary is the security control (see `api-v1-decisions.md#authentication`).                                                           |
| Rate limiting             | PRESENT/PARTIAL | IP-based limits exist for private and geocode routes. The `429` response and `Retry-After` contract are covered but still need broader fixture coverage.                                                    |
| Frontend compatibility    | PRESENT         | The Svelte client consumes private reads directly; no auth headers are required.                                                                                                                            |
| Runtime contract          | PARTIAL         | Podman Compose runs server, worker, web, database, and Valkey; local startup now gates application services on database readiness, while the full upload-to-worker contract still needs an executable gate. |
| Release stability         | CANDIDATE       | v0.1.4 has passed `make check`, `make validate`, image build, k3s rollout, and live browser request audit.                                                                                                  |

## Required next decisions

1. Keep the OpenAPI artifact and route-inventory test synchronized when handlers change.
2. Add schema-aware validation and response fixtures for the remaining resource families.
3. Rehearse the real upload-to-worker workflow before each production release.
