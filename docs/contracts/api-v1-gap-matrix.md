# API v1 contract gap matrix

Status: active development, pre-release

This matrix records the current contract evidence before OpenAPI, JWT, and rate-limit work. `/api/v1` is intentionally evolved in place until the first release; this document does not establish a
released compatibility promise or introduce an `/api/v2` policy.

## Current surfaces

The source of truth for the route inventory is `apps/iroha-server/pkg/httpapi/server.go`.

| Surface          | Current routes                                                                         | Result  |
| ---------------- | -------------------------------------------------------------------------------------- | ------- |
| Health           | `GET /healthz`                                                                         | PRESENT |
| Private briefing | `GET /api/v1/briefing`                                                                 | PRESENT |
| Raw files        | `POST /api/v1/raw-files`, `GET /api/v1/raw-files`, `GET /api/v1/raw-files/{rawFileId}` | PRESENT |
| Imports          | `POST /api/v1/imports`, `GET /api/v1/imports`, `GET /api/v1/imports/{importId}`        | PRESENT |
| Activities       | `GET /api/v1/activities`, detail, route, samplings, laps                               | PRESENT |
| Sleep            | list, aggregates, and segments under `/api/v1/sleep`                                   | PRESENT |
| Daily            | list and aggregates under `/api/v1/daily`                                              | PRESENT |
| Media            | list, detail, events, and aggregates under `/api/v1/media`                             | PRESENT |
| Public views     | summary, activities, routes, and geocode under `/public/v1`                            | PRESENT |

Gear, privacy-zone management, published-activity mutation, and activity mutation are roadmap items, not shipped routes.

## Contract evidence

| Area                      | Status          | Evidence and gap                                                                                                                          |
| ------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Route inventory           | PRESENT         | Router is explicit in `pkg/httpapi/server.go`; no machine-readable inventory exists yet.                                                  |
| Request and response DTOs | PRESENT         | Handler-local Go DTOs define the current JSON output. They are not yet an external contract artifact.                                     |
| OpenAPI                   | MISSING         | No OpenAPI document or canonical examples are checked in.                                                                                 |
| IDs and timestamps        | PARTIAL         | IDs are encoded consistently and timestamps use Go JSON time encoding; the client-facing rules are undocumented.                          |
| Pagination                | PARTIAL         | List responses use `items`, `next_cursor`, and `has_more`; cursor opacity, limits, and invalid-cursor behavior are undocumented.          |
| Errors                    | PARTIAL         | Errors currently use `{ "error": "..." }`; error codes, request correlation, and the complete status matrix are not defined.              |
| Private/public projection | PRESENT         | Public DTOs are separate from private activity DTOs and have a leakage test. Other public response contracts still need fixture coverage. |
| JWT authentication        | PRESENT/PARTIAL | JWT validation now protects private API mode; token issuance, rotation, and browser/session policy remain deployment concerns.            |
| Rate limiting             | PRESENT/PARTIAL | IP-based limits exist for private, public, and geocode routes. The `429` response and `Retry-After` contract are not defined.             |
| Frontend compatibility    | PARTIAL         | The Svelte client consumes private reads without auth headers; JWT adoption requires an explicit browser/session flow.                    |
| Runtime contract          | PARTIAL         | Podman Compose runs server, worker, web, database, and Valkey; the full upload-to-worker contract still needs an executable gate.         |
| Release stability         | UNKNOWN         | Stability is evaluated against the release candidate, not the current active-development branch.                                          |

## Required next decisions

1. Define the current `/api/v1` resource and field semantics in OpenAPI.
2. Define common pagination and error schemas.
3. Define JWT claims and private/public security requirements.
4. Define rate-limit buckets, identity, `429`, and `Retry-After` behavior.
5. Add contract fixtures and drift checks before changing handlers.
