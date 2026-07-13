# Bangumi provider capabilities

Status: planned.

Bangumi is a media catalog and list-state provider. Its anime, manga, book,
and related records link to canonical Iroha media works/items through
provider-scoped external references.

## Planned capabilities

| Capability | Status | Initial expectation |
| --- | --- | --- |
| Media identity | Planned | Titles, aliases, subjects, and editions |
| Relations | Planned | Provider relations and adaptation links |
| Consumption state | Planned | Convert collection/progress state into sourced events |
| Metadata enrichment | Planned | Preserve provider responses as evidence |

Bangumi state must not overwrite Iroha's append-only consumption history.
