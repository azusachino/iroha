# AniList provider capabilities

Status: planned; time semantics are defined by [ADR-0005](../../adr/0005-media-provider-time-semantics.md).

AniList is a media metadata and list-state provider, not the canonical media database. Its records should link to canonical Iroha works/items through provider-scoped external references.

## Planned capabilities

| Capability              | Status  | Initial expectation                                                                 |
| ----------------------- | ------- | ----------------------------------------------------------------------------------- |
| Media identity          | Planned | Anime/manga records and aliases                                                     |
| Relations               | Planned | Adaptations, sequels, and provider graph edges                                      |
| Current library state   | Planned | Normalize list status, progress, ratings, notes, repeat count, and fuzzy dates      |
| Provider update history | Planned | Import `ListActivity` with its exact `createdAt` when available                     |
| Consumption sessions    | Planned | Only exact manual/playback/provider evidence; never infer from a full list snapshot |
| Metadata enrichment     | Planned | Preserve provider responses as evidence                                             |

AniList `MediaList` state must not overwrite Iroha consumption history. `startedAt` and `completedAt` are fuzzy dates, and `updatedAt` is a record-update instant; neither becomes an exact consumption
event. The optional `ListActivity` feed is stored as explicitly labeled provider-update history. A later sync must retain the original provider snapshot and use activity IDs/state fingerprints for
idempotence.
