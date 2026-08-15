# Kindle provider capabilities

Status: planned; artifact/export-first and governed by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

Amazon documents Kindle synchronization for reading position, notes, and highlights across devices and apps. Those synchronized values are useful current state and annotation evidence, but they do not
by themselves provide a stable per-session history.

| Capability            | Status   | Expected evidence                                                       |
| --------------------- | -------- | ----------------------------------------------------------------------- |
| Book identity         | Planned  | ASIN/ISBN, title/author, Amazon personal-data export or local artifact  |
| Library and position  | Planned  | Current library, furthest position, percent/location where present      |
| Annotations           | Planned  | Kindle highlights, notes, bookmarks, and `My Clippings`-style artifacts |
| Completion/date facts | Planned  | Only when the export explicitly supplies a date; preserve its precision |
| Exact sessions        | Deferred | No inferred timestamps from sync or position changes                    |

The adapter must keep ASIN/ISBN as external references and distinguish a Kindle edition from its parent work. Sideloaded books without a reliable identifier create a resolution task rather than a
silent title merge.
