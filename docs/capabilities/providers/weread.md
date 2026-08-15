# WeRead provider capabilities

Status: planned; export-first and governed by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

WeRead is a future reading-history provider. Its initial support should be export-based if a stable user-owned export is available; connector support is not assumed.

## Planned capabilities

| Capability     | Status   | Expected evidence                                    |
| -------------- | -------- | ---------------------------------------------------- |
| Book identity  | Planned  | Export/provider IDs and titles                       |
| Reading status | Planned  | Sourced shelf/progress observations                  |
| Ratings/notes  | Planned  | Private sourced observations when exported           |
| Exact sessions | Deferred | Only if a stable documented activity source is found |

WeRead data maps to canonical Iroha media works/items and state history; it does not overwrite AniList, Bangumi, Goodreads, or manual history. Undocumented cookie-backed web endpoints are not a stable
connector contract.
