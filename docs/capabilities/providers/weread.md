# WeRead provider capabilities

Status: deferred.

WeRead is a future reading-history provider. Its initial support should be
export-based if a stable user-owned export is available; connector support is
not assumed.

## Deferred capabilities

| Capability | Status | Expected evidence |
| --- | --- | --- |
| Book identity | Deferred | WeRead export or connector snapshot |
| Reading status | Deferred | Sourced list/progress observations |
| Reading events | Deferred | Started, progressed, and completed events |
| Ratings/notes | Deferred | Private sourced observations |

WeRead data should map to canonical Iroha media works/items and append events;
it should not overwrite AniList, Bangumi, or Goodreads observations.
