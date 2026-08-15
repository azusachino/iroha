# Goodreads provider capabilities

Status: planned; export-first and governed by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

Goodreads is a future book-history provider. The first expected integration is an export-based intake rather than a live connector.

## Planned capabilities

| Capability     | Status   | Expected evidence                             |
| -------------- | -------- | --------------------------------------------- |
| Book identity  | Planned  | CSV title/author/ISBN/provider identifiers    |
| Reading status | Planned  | Exclusive shelf plus preserved custom shelves |
| Ratings/notes  | Planned  | Sourced rating, review, tags, and notes       |
| Day facts      | Planned  | `Date Read` as `source_date` when valid       |
| Exact sessions | Deferred | No stable exact-time session feed assumed     |

Goodreads records map to canonical books/editions and preserve the original CSV row as evidence. `read`, `currently-reading`, and `to-read` are the only default exclusive-shelf mappings; custom
shelves remain source labels until explicitly configured. `Date Read` is a calendar fact, not an exact timestamp.
