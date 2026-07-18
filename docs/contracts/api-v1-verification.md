# API v1 verification gate

Status: pre-release design

The contract gate verifies the active `/api/v1` surface in place. It is not a backward-compatibility gate for a released v1 and does not require an `/api/v2`.

## Gate layers

### 1. Route inventory

The registered routes in `apps/iroha-server/pkg/httpapi/server.go` must match the active paths in `docs/contracts/openapi.yaml`.

- deferred roadmap routes must not appear in the active OpenAPI paths;
- public routes must remain explicitly anonymous;
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

Private response fields must never appear in public projections. The test must inspect serialized JSON, not only Go structs, and must cover activities, routes, summaries, and future public media
projections.

### 5. Authentication behavior

Once JWT is implemented, tests must cover:

- local no-auth bypass;
- missing, malformed, expired, wrong-issuer, wrong-audience, and invalid-signature tokens;
- read and write scope enforcement;
- anonymous access to `/public/v1` and `/healthz`;
- rejection of private `/api/v1` access without a token in authenticated mode;
- no token or credential material in logs and error bodies.

### 6. Rate-limit behavior

Tests must verify the contract rather than only the configured number:

- the correct bucket is selected for private, public, geocode, and upload traffic;
- authenticated traffic is keyed by subject and anonymous traffic by IP;
- exhaustion returns `429`, the common error body, and `Retry-After`;
- local development uses its documented elevated budget;
- rate limiting does not mutate canonical data or job state.

### 7. Real workflow

The release-candidate rehearsal must exercise:

```text
JWT client
  -> upload raw file
  -> create import
  -> worker claims and processes job
  -> poll import status
  -> read private projection
  -> read sanitized public projection
```

The same fixture must run against the supported local runtime path and the containerized server/worker path. A green database check alone is insufficient.

## Completion rule

Task 5 may implement the gate only when the route inventory and contract decisions are reviewed. Task 6 may implement JWT and rate limiting only when the contract gate is green and the frontend
authentication flow is specified.
