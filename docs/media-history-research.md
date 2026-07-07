# Reading and Watching History Research

## Conclusion

Iroha should model reading and watching history as a sibling module to fitness, not as a media-server clone.

The durable shape should stay the same as the running module:

```text
raw imports and connector snapshots
  -> canonical media items and consumption events
  -> private dashboards and optional sanitized summaries
```

The main lesson from OSS media tools is that "the thing" and "my state on the thing" must stay separate. Jellyfin stores rich media-library items and exposes per-user data such as played/favorite/rating state through user-data endpoints. Audiobookshelf has the clearest progress model: `duration`, `progress`, `currentTime`, `isFinished`, `hideFromContinueListening`, `lastUpdate`, `startedAt`, and `finishedAt`. Komga exposes the same domain idea for books/comics with `read_status` filters such as unread, read, and in-progress. Seerr/Jellyseerr/Overseerr are useful for library discovery and request workflows, but they are not the source of truth for personal reading/watching history.

For Iroha, the first implementation should be event-first:

- `tb_media_items`: canonical work or episode/book/article/video identity.
- `tb_media_external_refs`: provider IDs such as TMDb, IMDb, TVDb, Open Library, ISBN, AniList, MAL, YouTube, Jellyfin item ID, Komga book ID, or Audiobookshelf item ID.
- `tb_media_consumption_events`: append-only user history events.
- `tb_media_progress`: latest resumable state per item/part, derived from events but stored for fast "continue" views.
- `tb_media_sources` and `tb_media_import_jobs`: connector/scraper runs, following `tb_raw_files` and `tb_import_jobs`.
- `tb_media_notes`: private notes, quotes, bookmarks, review text, and spoiler flags.

Do not begin with a recommendation engine, social graph, or full scraping backend. Start with importers that preserve raw evidence and normalize conservative facts.

## Metadata We Need

### Media Item

Required:

- `id`
- `media_type`: `book`, `manga`, `comic`, `article`, `movie`, `show`, `season`, `episode`, `video`, `podcast_episode`, `audiobook`
- `title`
- `sort_title`
- `original_title`
- `description`
- `release_date` or `release_year`
- `duration_seconds` for time-based media
- `page_count`, `chapter_count`, or `episode_count` where relevant
- `language`
- `country`
- `cover_image_url`
- `canonical_parent_id` for episode -> season -> show, chapter -> volume -> series, article -> feed/site

Useful:

- `creators`: author, director, cast, narrator, artist, publisher, studio, channel, site
- `genres`
- `tags`
- `content_rating`
- `series_name`
- `season_number`
- `episode_number`
- `volume_number`
- `edition`
- `isbn_10`, `isbn_13`
- `runtime_source`: local file, provider metadata, manual

### External Identity

Each item can have many external refs:

```text
provider
external_id
external_url
confidence
matched_by
raw_payload_id
created_at
updated_at
```

Use this table aggressively. Media identities are messy: a book work differs from an edition; a show differs from an episode; a YouTube upload can later be renamed; manga scan/source titles drift. Provider refs are the dedupe backbone.

### Consumption Event

Use append-only events, then derive current status.

Required:

- `id`
- `media_item_id`
- `event_type`: `started`, `progressed`, `completed`, `abandoned`, `reopened`, `rewatched`, `reread`, `rated`, `noted`, `bookmarked`, `hidden_from_continue`
- `event_at`
- `source_kind`: manual, csv, jellyfin, plex, trakt, komga, audiobookshelf, openlibrary, anilist, browser, rss
- `source_event_id`
- `raw_payload_id`

Progress fields:

- `unit`: `percent`, `seconds`, `pages`, `chapters`, `episodes`, `volume`, `position`
- `position`
- `total`
- `progress_percent`
- `duration_seconds`
- `started_at`
- `finished_at`
- `last_update_at`

Judgment fields:

- `rating`
- `rating_scale`
- `favorite`
- `review_text`
- `tags`
- `spoiler`

Context fields:

- `device`
- `app`
- `location_label`
- `visibility`
- `timezone`

### Current Progress

Store current progress separately for fast UI, but treat it as a projection:

```text
media_item_id
part_item_id
status: planned | in_progress | completed | abandoned
position
unit
progress_percent
started_at
last_update_at
finished_at
play_count/read_count
hidden_from_continue
source_kind
updated_at
```

This mirrors Audiobookshelf's `mediaProgress` and Jellyfin-style per-user item state, while keeping Iroha's audit trail intact.

## Scrapers and Connectors to Craft

Prefer "connector" over "scraper" where an API/export exists. HTML scraping should be the last resort because it is brittle and often violates site expectations.

### Phase 1: Manual and File Imports

- Manual entry and edit UI.
- CSV/JSON import for generic history.
- Letterboxd CSV import for films.
- Goodreads/StoryGraph-style CSV import for books if the user can export it.
- Browser bookmark/history import for articles only after the schema is stable.

These are the safest first sources because they produce raw files Iroha can preserve and reprocess.

### Phase 2: Self-Hosted App Connectors

- Jellyfin: import watched state, favorite/rating state, playback progress, play count, last played, and item identity. This fits Iroha well because Jellyfin already separates media items from user item data.
- Komga: import book/comic library items plus read status and progress.
- Audiobookshelf: import media progress, bookmarks, continue-listening visibility, started/finished timestamps, and podcast/audiobook item metadata.
- Seerr/Jellyseerr/Overseerr: import watchlist/blocklist/request/library-availability signals only. Do not treat it as watched history.

Self-hosted connectors should store connector snapshots as raw JSON, then normalize. That preserves evidence and lets parser fixes re-run.

### Phase 3: Metadata Enrichment APIs

- TMDb for movie, TV, person, image, and release metadata.
- Open Library for books and covers. Respect its API guidance: cache responses, identify the app, and do not scrape HTML.
- AniList for anime and manga metadata through GraphQL.
- MusicBrainz/ListenBrainz only if music listening history enters scope later.
- RSS/Atom feeds for article and podcast episode metadata.

Metadata enrichment should be idempotent and replaceable. External provider payloads should not overwrite user-entered titles, notes, ratings, or completion history unless the user explicitly accepts the match.

### Phase 4: Activity Capture

- Browser extension or share-sheet endpoint for article saves and read-later events.
- Local webhook endpoint for media players that can emit play/pause/progress events.
- Optional Trakt-style sync import/export for watched history interoperability.

This should come after manual/file imports because live activity capture creates dedupe, privacy, and conflict-resolution work.

## Normalization Rules

- Preserve every raw import or connector response.
- Normalize to stable items and append-only events.
- Keep current progress as a projection.
- Dedupe by provider refs first, then by conservative fallback matches.
- Distinguish work, edition, and part:
  - book work vs ISBN edition
  - show vs season vs episode
  - manga series vs volume vs chapter
  - podcast show vs episode
- Allow multiple completions. Rewatching/rereading should create new events, not overwrite old ones.
- Do not assume every source has exact timestamps. Some imports only know date, not time.
- Keep source confidence. A TMDb match found by title/year is weaker than a direct TMDb ID.
- Separate private history from public summaries. Public summaries should aggregate counts, time, pages, tags, and favorites without exposing raw notes or exact URLs unless explicitly published.

## Minimal First Schema

```sql
create table tb_media_items (
  id uuid primary key,
  media_type text not null,
  title text not null,
  sort_title text not null default '',
  original_title text not null default '',
  description text not null default '',
  release_date date,
  duration_seconds integer,
  page_count integer,
  language text not null default '',
  parent_id uuid references tb_media_items(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_external_refs (
  id uuid primary key,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  provider text not null,
  external_id text not null,
  external_url text not null default '',
  confidence numeric,
  matched_by text not null default '',
  created_at timestamptz not null,
  unique(provider, external_id)
);

create table tb_media_consumption_events (
  id uuid primary key,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  event_type text not null,
  event_at timestamptz,
  source_kind text not null,
  source_event_id text not null default '',
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  rating numeric,
  rating_scale numeric,
  note text not null default '',
  raw_file_id uuid references tb_raw_files(id),
  created_at timestamptz not null
);

create table tb_media_progress (
  media_item_id uuid primary key references tb_media_items(id) on delete cascade,
  status text not null,
  unit text not null default '',
  position numeric,
  total numeric,
  progress_percent numeric,
  started_at timestamptz,
  last_update_at timestamptz,
  finished_at timestamptz,
  play_count integer not null default 0,
  hidden_from_continue boolean not null default false,
  source_kind text not null default '',
  updated_at timestamptz not null
);
```

This deliberately omits collections, lists, notes, creators, and images. Add them after one full import loop proves the event/progress model.

## Source Notes

- Jellyfin describes itself as a free media system for managing and streaming media, and its API exposes user-data operations such as favorite/rating updates and latest-media queries with played filters and optional user data.
- Seerr is a request and discovery manager for Jellyfin/Plex/Emby; its feature list emphasizes user import, library scans, requests, watchlisting, and blocklisting rather than personal consumption history.
- Audiobookshelf explicitly models media progress updates with duration, percent progress, current time, finished state, continue-listening visibility, started time, and finished time; it also syncs local and server progress by `lastUpdate`.
- Komga's OpenAPI exposes book listing filters for `read_status` values including unread, read, and in-progress, and supports API-key/session authentication.
- Open Library provides APIs for book discovery and lookup but explicitly asks clients to cache responses, identify themselves, avoid HTML scraping, and avoid bulk harvesting through the live API.
- TMDb exposes movie, TV, actor, and image API methods; it is the obvious enrichment source for movies/shows, not the canonical user-history store.
- AniList exposes anime and manga data through GraphQL and is appropriate for metadata enrichment or user-list import where credentials are available.

References:

- [Jellyfin repository](https://github.com/jellyfin/jellyfin)
- [Jellyfin user-data controller](https://raw.githubusercontent.com/jellyfin/jellyfin/master/Jellyfin.Api/Controllers/UserLibraryController.cs)
- [Seerr repository](https://github.com/seerr-team/seerr)
- [Audiobookshelf API: media progress](https://api.audiobookshelf.org/#createupdate-media-progress)
- [Komga OpenAPI](https://raw.githubusercontent.com/gotson/komga/master/komga/docs/openapi.json)
- [Open Library APIs](https://openlibrary.org/developers/api)
- [TMDb API getting started](https://developer.themoviedb.org/reference/intro/getting-started)
- [AniList API docs](https://anilist.gitbook.io/anilist-apiv2-docs/)
