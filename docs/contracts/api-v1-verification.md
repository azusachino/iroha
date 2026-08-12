# API v1 verification gate

Status: active-development verification for v0.4.0

## Gate A — freeze repaired existing contracts

Gate A is the owner approval checkpoint between existing-contract repairs and new expense/report implementation. It answers one question:

> Are the repaired existing `/api/v1` wire contracts acceptable as the v0.4 baseline?

Gate A covers only the existing data domains and shared HTTP behavior. The owner reviews and approves these decisions:

- calendar-only values use `YYYY-MM-DD`; instants use RFC 3339; existing daily/sleep list filters remain inclusive;
- future report adapters use explicit half-open `[from,to)` ranges internally;
- daily rows use `ring: null` when no ring record exists, and daily aggregate metrics are explicit `(metric, unit, observed_days)` entries;
- sleep aggregates report `nap_count` separately and calculate averages/stages from main sleep only;
- media event IDs are distinct from media item IDs, undated events remain undated, and undated events are excluded from time-based reporting;
- list limits are omitted/defaulted to 50 or explicitly bounded to 1..100; cursors remain opaque; errors use `{code,message,request_id}`;
- web mutations support PUT/DELETE and 204 responses, and private CORS allows their preflights;
- the OpenAPI document, representative fixtures, and registered Chi routes remain in parity;
- the cache namespace is versioned for the repaired contract, while future expense/report routes are uncached;
- `/api/v1` remains unauthenticated but private-network-only, and the sanitized public export remains a separate projection.

Gate A does not approve the expense data model, monthly report response, CLI workflow, cockpit UX, Telegram, Suzuran, OCR, or scheduled report delivery. Those remain later implementation decisions.

The gate is complete when the owner says `Gate A approved` (or supplies corrections). Only then may tasks 12–21, which add expenses and monthly reports, be dispatched.

The contract gate verifies the active `/api/v1` surface in place. It is not a backward-compatibility gate for a released v1 and does not require an `/api/v2`.

## Gate layers

### 1. Route inventory

The registered routes in `apps/iroha-server/pkg/httpapi/server.go` must match the active paths in `docs/contracts/openapi.yaml`.

`make contract-check` walks the Chi router, parses `docs/contracts/openapi.yaml`, and asserts method/path parity. It also validates the committed example manifest: every example names an active
operation and an existing schema, parses as JSON, and contains that schema's required top-level fields.

- deferred roadmap routes must not appear in the active OpenAPI paths;
- the static public export must remain sanitized and separate from private HTTP routes;
- health must remain separate from application data routes;
- a route added during active development must update the OpenAPI artifact in the same change.

### 2. Schema and example validation

Validate the OpenAPI document and every committed example as JSON/YAML. Each example must identify its target operation in `examples/manifest.json` and decode as canonical wire evidence, not
prose-only documentation. The Go contract test currently validates the representative top-level shapes; endpoint integration tests remain responsible for full response behavior.

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

- `make contract-check` passes against the registered private route inventory and OpenAPI path set.
- Rate-limit tests cover `429`, the common error body, and `Retry-After`.
- `make check` passes, including the frontend formatter, Svelte check, and frontend tests.

The end-to-end worker/import rehearsal remains a release-candidate check because it requires the supported local and containerized runtime paths.

## Completion rule

The route inventory and contract decisions must be reviewed before implementation. Rate limiting may be considered contract-gated once the checks above remain green and the release-candidate rehearsal
is completed.
