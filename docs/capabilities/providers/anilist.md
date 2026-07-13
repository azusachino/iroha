# AniList provider capabilities

Status: planned.

AniList is a media metadata and list-state provider, not the canonical media
database. Its records should link to canonical Iroha works/items through
provider-scoped external references.

## Planned capabilities

| Capability | Status | Initial expectation |
| --- | --- | --- |
| Media identity | Planned | Anime/manga records and aliases |
| Relations | Planned | Adaptations, sequels, and provider graph edges |
| Consumption state | Planned | Convert list/progress state into sourced events |
| Metadata enrichment | Planned | Preserve provider responses as evidence |

AniList state must not overwrite Iroha event history. A later sync may append a
new observation or event and must retain the original provider snapshot.
