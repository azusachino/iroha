# Goodreads provider capabilities

Status: deferred.

Goodreads is a future book-history provider. The first expected integration is
an export-based intake rather than a live connector.

## Deferred capabilities

| Capability | Status | Expected evidence |
| --- | --- | --- |
| Book identity | Deferred | Goodreads CSV export and external IDs |
| Reading status | Deferred | Shelves and status snapshots |
| Ratings | Deferred | Sourced rating observations |
| Reading events | Deferred | Dates/progress where the export provides them |

Goodreads records should map to canonical books/editions and preserve the
original export row as evidence. They must not become a second canonical book
table.
