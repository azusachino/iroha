# API v1 verification gate

Status: release-candidate verification for v0.1.4

The contract gate verifies the active `/api/v1` surface in place. It is not a backward-compatibility gate for a released v1 and does not require an `/api/v2`.

## Gate layers

### 1. Route inventory

The registered routes in `apps/iroha-server/pkg/httpapi/server.go` must match the active paths in `docs/contracts/openapi.yaml`.

The first executable slice is `make contract-check`, which walks the Chi router and asserts the current active method/path inventory. The OpenAPI path set remains a reviewed artifact until a
schema-aware validator is added.

- deferred roadmap routes must not appear in the active OpenAPI paths;
- the static public export must remain sanitized and separate from private HTTP routes;
- health must remain separate from application data routes;
- a route added during active development must update the OpenAPI artifact in the same change.

### 2. Schema and example validation

Validate the OpenAPI document and every committed example as JSON/YAML. Each example must identify its target operation and decode against the referenced schema. Examples are canonical wire evidence,
not prose-only documentation.

### 3. HTTP response fixtures

The HTTP test suite must cover, for each resource family:

- successful list/detail/aggregate responses;
- required fields and field names;
- omitted optional values and explicit `null` values;
- malformed IDs, dates, filters, cursors, and request bodies;
- not-found and persistence failures;
- pagination continuation and terminal pages.

### 4. Projection safety

Private response fields must never appear in the static public export. The test must inspect serialized JSON, not only Go structs, and must cover activities, routes, summaries, and future public media
projections.

### 5. Authentication

`/api/v1` and `/healthz` are intentionally unauthenticated (see `api-v1-decisions.md#authentication`). The public site is a static export, not an HTTP API. There is no token/scope behavior to verify;
tests instead confirm no credential material (tokens, secrets) appears in logs or error bodies, since none should ever be sent.

### 6. Rate-limit behavior

Tests must verify the contract rather than only the configured number:

- the correct bucket is selected for private, public, geocode, and upload traffic, keyed by client IP;
- exhaustion returns `429`, the common error body, and `Retry-After`;
- rate limiting does not mutate canonical data or job state.

### 7. Real workflow

The release-candidate rehearsal must exercise:

```text
client
  -> upload raw file
  -> create import
  -> worker claims and processes job
  -> poll import status
  -> read private projection
  -> read sanitized public projection
```

The same fixture must run against the supported local runtime path and the containerized server/worker path. A green database check alone is insufficient.

## Current implementation evidence

- `make contract-check` passes against the registered private route inventory.
- Rate-limit tests cover `429`, the common error body, and `Retry-After`.
- `make check` passes, including the frontend formatter, Svelte check, and frontend tests.

The end-to-end worker/import rehearsal remains a release-candidate check because it requires the supported local and containerized runtime paths.

## Completion rule

The route inventory and contract decisions must be reviewed before implementation. Rate limiting may be considered contract-gated once the checks above remain green and the release-candidate rehearsal
is completed.
