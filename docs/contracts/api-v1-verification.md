# API v1 verification gate

Status: Gate A approved; v0.4.1 released locally

## Gate A — freeze repaired existing contracts

Gate A is the owner approval checkpoint between existing-contract repairs and new expense/report implementation. It answers one question:

> Are the repaired existing `/api/v1` wire contracts acceptable as the v0.4 baseline?

Gate A covers only the existing data domains and shared HTTP behavior. The owner reviews and approves these decisions:

- calendar-only values use `YYYY-MM-DD`; instants use RFC 3339; date ranges use half-open `[from,to)` semantics;
- report, activity, sleep, daily, and expense period adapters use explicit half-open `[from,to)` ranges internally;
- daily rows use `ring: null` when no ring record exists, and daily aggregate metrics are explicit `(metric, unit, observed_days)` entries;
- sleep aggregates report `nap_count` separately and calculate averages/stages from main sleep only;
- media event IDs are distinct from media item IDs, undated events remain undated, and undated events are excluded from time-based reporting;
- list limits are omitted/defaulted to 50 or explicitly bounded to 1..100; cursors remain opaque; errors use `{code,message,request_id}`;
- web mutations support PUT/DELETE and 204 responses, and private CORS allows their preflights;
- the OpenAPI document, representative fixtures, and registered Chi routes remain in parity;
- the cache namespace is versioned for the repaired contract; successful private GET reads, including direct expense records, use the shared cache module, while canonical mutations invalidate their
  dependent namespaces;
- `/api/v1` remains unauthenticated but private-network-only, and the sanitized public export remains a separate projection.

Gate A does not approve the expense data model, monthly report response, CLI workflow, cockpit UX, Telegram, Suzuran, OCR, or scheduled report delivery. Those remain later implementation decisions.

At the time of the Gate A review, the gate was complete when the owner said `Gate A approved` (or supplied corrections); only then could tasks 12–21, which add expenses and monthly reports, be
dispatched. Those tasks are now part of the released v0.4.1 surface.

Gate A is approved. The expense, monthly-report, CLI, and cockpit work described below was implemented after that approval. The v0.4.1 branch completed the canonical projection, request fan-out,
date-range, cache, media-history, release-candidate, and local deployment checks. The local `v0.4.1` tag records this release boundary; remote merge/publication and any production image rollout remain
external handoffs.

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
- The previous local release-candidate evidence is retained in the dated audit record; the final release evidence is in [the v0.4.1 release audit](../audits/2026-08-15-v0.4.1-release.md).
- The final release reran `make check`, `make validate`, and `make release-candidate` after the projection, contract, media-history, and detail changes.
- Local k3s deployment and live cache/browser smoke are recorded separately from the source release; the current cluster remains on explicitly named dev image pins.

The end-to-end worker/import rehearsal is covered by the release-candidate target; repeat it for each future production release.

## Completion rule

The route inventory and contract decisions were reviewed before implementation. The attached release evidence is complete for the local `v0.4.1` source tag. This document does not promise backward
compatibility for future pre-1.0 releases.
