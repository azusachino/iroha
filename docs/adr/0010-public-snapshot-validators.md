# ADR-0010: Encode Public Snapshots Once and Validate Their Exact Bytes

## Status

Proposed for PR #125 under the owner-approved local #116 scope. Independent contract verification is required before acceptance.

## Date

2026-10-04

## Context

The public snapshot cache held decoded projections and encoded them on every response. There was no ETag or conditional GET. Activity revision changes and the 24-hour TTL rebuild the projection; the latter changes metadata generation time even without a revision change. A revision-only validator would therefore incorrectly identify different metadata bytes as identical.

## Decision

Cache the four snapshot representations (`summary`, `activities`, `routes`, `meta`) as immutable JSON encoder output, including the existing trailing newline. Encode each representation once when rebuilding the projection and compute a full SHA-256 strong ETag from those exact bytes. Publish a rebuilt cache only after every encoding succeeds. Keep the existing activity revision, mutex, TTL, sanitized projection, rate limit, CORS and `public, max-age=86400` policy.

Use Go's `http.ServeContent` on the cached bytes, with the existing JSON content type, ETag and cache policy. Normalize repeated entity-tag field lines on a cloned request. Ignore Range/If-Range by removing them on that clone and suppress the resulting Accept-Ranges header: this API continues serving whole JSON documents, not partial representations. Do not advertise Last-Modified; metadata generation time is not the representation's modification time.

Go evaluates If-Match before If-None-Match. A failed If-Match yields an empty 412 with `Cache-Control: no-store` and without representation Content-Type/ETag; RFC 9111 §3 otherwise permits a shared cache to store a conditional failure under the resource's cache key. This was identified during independent contract research and has a regression assertion. A 412 must not poison subsequent unconditional reads; a matching If-None-Match on GET yields bodyless 304 using weak comparison, including tag lists and `*`. Snapshot/build failures and existing router/rate-limit failures win before conditional evaluation. Do not add HEAD routes or validators to uncached activity details in this change.

## Primary sources and rejected alternative

[RFC 9110 §13.1.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-13.1.1) requires origin servers to evaluate If-Match, including on representation-selecting methods. An If-None-Match-only implementation that silently ignores If-Match is therefore rejected. [§13.1.2](https://www.rfc-editor.org/rfc/rfc9110.html#section-13.1.2) requires weak comparison; [§13.2.1–2](https://www.rfc-editor.org/rfc/rfc9110.html#section-13.2) establishes normal-check precedence and conditional evaluation order. [§15.4.5](https://www.rfc-editor.org/rfc/rfc9110.html#section-15.4.5) requires bodyless 304 with applicable validators/cache metadata. [§14.2](https://www.rfc-editor.org/rfc/rfc9110.html#section-14.2) permits ignoring Range. Go's [ServeContent documentation](https://pkg.go.dev/net/http#ServeContent) describes its validator handling.

A custom entity-tag parser would duplicate the standard library's syntax, weak/strong comparison and precedence rules. Unmodified ServeContent would introduce byte-range responses, so the small adapter deliberately retains whole-document semantics. A prospective source test must cover If-Match precedence, weak/list/wildcard matching, repeated fields, changed bytes, matching metadata, errors before validators, unchanged method handling, and ignored ranges.

## Consequences and evidence boundary

Validators reveal only the sanitized representation's byte identity. Encoding errors cannot leave a partially rebuilt cache. Successful 304 responses retain origin cache/CORS metadata; Go supplies Date on actual HTTP responses. An isolated origin/proxy fixture must verify compression and validator round trips; source review does not prove Caddy's deployed behavior. Live edge policy, publication, deployment and removal/disclosure-policy changes remain explicitly deferred.
