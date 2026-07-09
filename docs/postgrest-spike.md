# PostgREST Spike Notes

## Task 3: Type Generation and Web Client

PostgREST 14.14 exposes a Swagger 2.0 document at `/`. Current `openapi-typescript` rejects Swagger 2.0 directly, so the repeatable type-generation path is:

```text
PostgREST Swagger 2.0 -> swagger2openapi -> openapi-typescript -> src/lib/postgrest/types.ts
```

The web helper is `apps/iroha-web/scripts/gen-postgrest-types.mjs`, exposed as:

```bash
cd apps/iroha-web
POSTGREST_OPENAPI_URL=http://127.0.0.1:3001/ bun run postgrest:types
```

Findings:

- The generated table/view row types are useful for `api.public_activities`: the type surface includes only the sanitized view columns, so source/raw-file fields are absent at compile time.
- RPC return types are weak: `api.public_summary()` returns JSON, but the generated OpenAPI operation has no response content schema. The web client still needs a hand-written `PostgrestSummary` interface.
- PostgREST advertises `post`/`patch`/`delete` operations for views in the OpenAPI document even when database grants deny them. Type generation alone does not communicate the read-only permission boundary.
- Filters are typed as plain `string`, so operator syntax like `sport_type=eq.run` and repeated range filters must be hand-built and tested.
- Pagination is offset/limit or Range-header based. That is ergonomic for a scratch list, but it is not equivalent to iroha-server's keyset cursor API for large or mutating activity lists.
- The public route-map endpoint is not represented as a PostgREST view in this spike because the Go endpoint applies privacy trimming, private-zone masking, route splitting, and decimation before exposing coordinates.

Scratch UI:

- `/postgrest-spike` loads `api.public_activities` and `api.public_summary()` through the Vite `/postgrest` proxy.
- It intentionally uses offset pagination to show the PostgREST default DX side-by-side with the current keyset cursor client.
