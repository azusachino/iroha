# Apple Books / iBooks provider capabilities

Status: planned; snapshot-first and governed by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

`ibooks` is an intake alias for the canonical provider ID `apple_books`. Apple documents synchronization of books, audiobooks, collections, reading position, bookmarks, notes, and highlights through
iCloud, but this does not establish a public per-session history feed.

| Capability                  | Status   | Expected evidence                                                |
| --------------------------- | -------- | ---------------------------------------------------------------- |
| Book/audiobook/PDF identity | Planned  | Local/export snapshot, ISBN/store/asset ID where present         |
| Library and collections     | Planned  | Current library, Want to Read, Finished, and custom collections  |
| Position                    | Planned  | Current page/percent/location when the snapshot exposes it       |
| Annotations                 | Planned  | Bookmarks, notes, highlights when exported                       |
| Exact sessions              | Deferred | No invented session time from current progress or finished state |

The adapter must preserve the local snapshot and map each concrete edition to a canonical media item. A finished state is current state; it is not an exact reading event. A source-provided calendar
completion date, if a future export exposes one, uses `source_date`.
