# API v1 contract decisions

Status: release-candidate decision record for v0.1.4

These decisions apply to the current `/api/v1` surface in place. They do not freeze a released v1 and do not introduce `/api/v2`. Until the first release, the contract, implementation, and frontend
may change together when the verification gate is updated in the same change.

## Surface ownership

- `/api/v1` is the private application API.
- The sanitized public projection is a static JSON/GeoJSON export consumed by `apps/iroha-public-site`; there is no live `/public/v1` API.
- `/healthz` is an anonymous liveness endpoint and is not an application data contract.
- Roadmap-only resources are excluded from OpenAPI until their handlers exist.

## Resource and wire rules

- JSON field names use `snake_case`.
- IDs are opaque strings. Clients must not parse the embedded UUID or depend on the prefix beyond validating the documented resource type.
- Instants use RFC 3339 JSON timestamps. Calendar-only values use `YYYY-MM-DD` and are not midnight timestamps.
- Daily and sleep list/aggregate `from` and `to` filters accept `YYYY-MM-DD`; both bounds are currently inclusive for these existing endpoints. New period-based report methods use explicit half-open `[from,to)` boundaries internally and do not reuse these list filters.
- Monthly aggregate periods serialize as `YYYY-MM`; yearly aggregate periods serialize as `YYYY`.
- Daily rows expose ring measurements as a nullable `ring` object; metric-only days return `ring: null` rather than fabricated zero ring values. Daily aggregate `metrics` is always an array of `{metric, value, unit, observed_days}` entries, grouped by metric and unit.
- Optional values are omitted when absent. A response field that is always present but can be empty uses an explicit empty string, empty array, empty object, or `null` according to its schema.
- Response objects use named schemas rather than exposing persistence models.
- Unknown response fields must be ignored by clients; unknown request fields are rejected once request validation is added.

## Pagination

List endpoints use this envelope:

```json
{
  "items": [],
  "next_cursor": null,
  "has_more": false
}
```

- `limit` is optional and defaults to 50 for paginated domain lists.
- Explicit `limit` values must be integers from 1 through 100; non-numeric, zero, negative, and larger values return `400`.
- Cursors are opaque, endpoint-specific, and must be returned unchanged to continue a listing.
- Invalid cursors return `400` with the common error schema.
- A cursor does not grant access and contains no client-authoritative filter state.
- Aggregate endpoints are not paginated unless their individual contract says otherwise.

## Errors

The common error response is:

```json
{
  "code": "invalid_cursor",
  "message": "cursor is invalid",
  "request_id": "req_01..."
}
```

`code` is a stable machine value for the endpoint family; `message` is for diagnostics and may change. `request_id` correlates a response with server logs. The OpenAPI document defines the status/code
matrix for each route.

## Authentication

`/api/v1` and `/healthz` are unauthenticated. iroha is a single-user personal deployment (private LAN/NAS); the network boundary is the security control, not an application-level credential. A prior
revision of this contract carried JWT bearer authentication on `/api/v1`, but the only clients are the operator's own private-network web viewer and personal automation (e.g. a Telegram bot) — neither
obtains credentials through a login flow, so the token was always a self-issued static secret standing in for network-level access control. It added an extra secret-provisioning step without changing
who could reach the API. The static export site is the only surface designed for public exposure, and it only ever serves sanitized data.

A future multi-user or public-write deployment would need a real login/session flow rather than reintroducing a static bearer token.

## Rate limiting

Rate limiting is part of the HTTP behavior, keyed by client IP:

- private API: 6000 requests per minute — generous, since the API is reachable only from the operator's own private network;
- static public export: no live request budget;
- geocode: 10 requests per minute;
- exhausted budgets return `429 Too Many Requests`, the common error body, and `Retry-After` in seconds.

These are initial defaults, not domain invariants. They are configurable by deployment, while the status/header/error behavior remains stable.
