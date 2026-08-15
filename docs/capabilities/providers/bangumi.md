# Bangumi provider capabilities

Status: planned; time semantics are defined by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

Bangumi is a media catalog and list-state provider. Its anime, manga, book, and related records link to canonical Iroha media works/items through provider-scoped external references.

## Planned capabilities

| Capability            | Status  | Initial expectation                                                               |
| --------------------- | ------- | --------------------------------------------------------------------------------- |
| Media identity        | Planned | Titles, aliases, subjects, and editions                                           |
| Relations             | Planned | Provider relations and adaptation links                                           |
| Current library state | Planned | Normalize collection status, episode/volume progress, ratings, tags, and comments |
| State history         | Planned | Append Iroha-observed diffs with explicit observation basis                       |
| Consumption sessions  | Planned | Only exact external/manual evidence; collection snapshots are not sessions        |
| Metadata enrichment   | Planned | Preserve provider responses as evidence                                           |

Bangumi collection state must not overwrite Iroha's append-only consumption history. The official subject-collection contract warns that `updated_at` is unreliable, and episode collection timestamps
may be `0` (unknown); neither is used as a canonical consumption time. Sync-only changes remain state history with `iroha_observed` timing.
