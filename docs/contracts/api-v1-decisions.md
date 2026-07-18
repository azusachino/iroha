# API v1 contract decisions

Status: pre-release design for active development

These decisions apply to the current `/api/v1` surface in place. They do not freeze a released v1 and do not introduce `/api/v2`. Until the first release, the contract, implementation, and frontend
may change together when the verification gate is updated in the same change.

## Surface ownership

- `/api/v1` is the private application API.
- `/public/v1` is an anonymous sanitized projection API.
- `/healthz` is an anonymous liveness endpoint and is not an application data contract.
- Roadmap-only resources are excluded from OpenAPI until their handlers exist.

## Resource and wire rules

- JSON field names use `snake_case`.
- IDs are opaque strings. Clients must not parse the embedded UUID or depend on the prefix beyond validating the documented resource type.
- Instants use RFC 3339 JSON timestamps. Calendar-only values use `YYYY-MM-DD` and are not midnight timestamps.
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

`code` is a stable machine value for the endpoint family; `message` is for diagnostics and may change. `request_id` correlates a response with server logs. The OpenAPI document will define the
status/code matrix for each route.

## JWT authentication

Authentication is deployment-configurable:

- authenticated mode requires `Authorization: Bearer <JWT>` on `/api/v1`;
- trusted local mode may bypass authentication with `IROHA_LOCAL_NO_AUTH=true`;
- `/public/v1` and `/healthz` remain anonymous;
- read operations require `iroha:read` and write operations require `iroha:write` scopes.

The required claims are `iss`, `sub`, `aud`, `iat`, `exp`, `jti`, and `scope`. Signing algorithm, key rotation, and token issuance are deployment concerns, not domain response semantics. Expired,
malformed, wrong-audience, and insufficient-scope tokens return the common `401`/`403` errors.

The current browser client supports a deployment-provided bearer token through `PUBLIC_IROHA_API_TOKEN`. This is suitable only for a trusted private self-hosted network: a token embedded in public
static JavaScript is not considered a secret. A future multi-user deployment needs a real login/session flow and must not reuse this static-token path.

## Rate limiting

Rate limiting is part of the HTTP behavior:

- private API: 120 requests per minute in authenticated deployment mode;
- public API: 60 requests per minute;
- geocode: 10 requests per minute;
- local no-auth development may use the existing elevated private/public budget;
- authenticated requests are keyed by JWT subject where available, otherwise by client IP;
- anonymous requests are keyed by client IP;
- exhausted budgets return `429 Too Many Requests`, the common error body, and `Retry-After` in seconds.

These are initial defaults, not domain invariants. They are configurable by deployment, while the status/header/error behavior remains stable.
