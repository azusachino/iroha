# Reading and Watching History Research

## Conclusion

Iroha should model reading and watching history as a sibling module to fitness, not as a media-server clone.

The durable shape should stay the same as the running module:

```text
intake payloads, raw imports, and connector snapshots
  -> canonical media items and consumption events
  -> private dashboards and optional sanitized summaries
```

The main lesson from OSS media tools is that "the thing" and "my state on the thing" must stay separate. Jellyfin stores rich media-library items and exposes per-user data such as
played/favorite/rating state through user-data endpoints. Audiobookshelf has the clearest progress model: `duration`, `progress`, `currentTime`, `isFinished`, `hideFromContinueListening`,
`lastUpdate`, `startedAt`, and `finishedAt`. Komga exposes the same domain idea for books/comics with `read_status` filters such as unread, read, and in-progress. Seerr/Jellyseerr/Overseerr are useful
for library discovery and request workflows, but they are not the source of truth for personal reading/watching history.

For Iroha, the stable design should be ontology-first and event-backed:

- `tb_media_works`: abstract creative identity such as a franchise, story, or intellectual work.
- `tb_media_items`: concrete consumable or listable things: anime series, season, episode, OVA, movie, manga series, volume, chapter, light novel edition, book edition, article, video, audiobook,
  podcast episode.
- `tb_media_titles`: original titles, translated titles, romanizations, aliases, search aliases, and user aliases.
- `tb_media_relations`: adaptations, sequels, prequels, side stories, compilations, alternate versions, and provider-specific graph edges.
- `tb_media_external_refs`: provider IDs such as TMDb, IMDb, TVDb, Open Library, ISBN, AniList, MAL, YouTube, Jellyfin item ID, Komga book ID, or Audiobookshelf item ID.
- `tb_media_consumption_events`: append-only user history events.
- `tb_media_progress`: latest resumable state per item/part, derived from events but stored for fast "continue" views.
- `tb_media_sources` and `tb_media_import_jobs`: connector/scraper runs, following `tb_raw_files` and `tb_import_jobs`.
- `tb_media_notes`: private notes, quotes, bookmarks, review text, and spoiler flags.

Do not collapse all media into one flat item table. A single named work can have light novels, manga, anime seasons, OVAs, movies, games, specials, localized editions, alternate cuts, recap films, and
translated aliases. The model must preserve those distinctions from the start.

## What Raw Imports Mean Here

For fitness data, "raw import" usually means a file: Apple Health zip, GPX, FIT, TCX, or Strava export.

For reading and watching history, original data can come from more shapes:

- A user intent payload: "I watched Dune Part Two tonight", "started Frieren episode 9", "finished book ISBN 978...", or "read chapter 42".
- A file export: CSV/JSON from Letterboxd, Goodreads, StoryGraph, AniList/MAL export, browser history, RSS reader export, or a generic hand-written CSV.
- A personal tracker connector snapshot: AniList, Bangumi.tv, MAL, Trakt, Letterboxd, Goodreads, StoryGraph, or another service where the user already keeps list state.
- A self-hosted connector snapshot: Jellyfin, Komga, Audiobookshelf, Seerr/Jellyseerr/Overseerr, or a future local media player webhook.
- A metadata lookup payload: TMDb, Open Library, AniList, ISBN lookup, RSS/Atom feed item, or YouTube/oEmbed-style metadata.
- A manual edit payload from the web UI.

The rule is not "everything must be a user-uploaded file". The rule is "preserve the original evidence for every state change". If a Telegram bot calls an Iroha API, the original JSON request is the
raw evidence. If Iroha queries TMDb to resolve the title, the provider response is another raw payload. If the user later corrects the match, that correction is also an event, not a silent overwrite.

That suggests a more general intake model than `tb_raw_files`:

```text
tb_intake_payloads
  id
  source_kind: telegram | web_manual | csv | anilist | bangumi | jellyfin | komga | audiobookshelf | tmdb | openlibrary
  source_actor: user | connector | enrichment
  content_type
  sha256
  storage_path
  received_at
  parsed_at
```

Large files can still live in `tb_raw_files`; media history can either reuse that table or add `tb_intake_payloads` as the generalized sibling once the module starts. The important product behavior is
that the user never has to re-explain their history after parser or matcher improvements.

## UX Direction

The first UX should optimize for fast capture with later cleanup.

### Telegram Capture

The Telegram bot should stay an external client, like the existing upload-client model. It should not parse, scrape, or decide canonical identity. It should forward the user's intent to
`iroha-server`.

Flow:

```text
user -> Telegram bot: "watched Dune Part Two tonight 4.5/5"
  -> bot POST /api/v1/media/intake
  -> iroha stores original request payload
  -> iroha creates an intake job
  -> iroha extracts candidate facts
  -> iroha searches local media refs
  -> iroha calls enrichment providers if needed
  -> iroha creates or links tb_media_items
  -> iroha appends tb_media_consumption_events
  -> iroha updates tb_media_progress
  -> bot replies with confirmation or disambiguation choices
```

The bot response should be concise:

```text
Added: Dune: Part Two
Event: watched on 2026-07-08
Rating: 4.5/5
```

If Iroha is not confident:

```text
Which one?
1. Dune: Part Two (2024 film)
2. Dune (2021 film)
3. Dune: Prophecy (2024 show)
```

The bot sends the selected candidate back to Iroha as another intake payload. Iroha records the match decision so future "Dune Part Two" messages do not ask again.

### Web UX

The web UI should provide three surfaces:

- Quick add: one input for "watched/read/started/progressed" text, with structured fields appearing only after Iroha parses a draft.
- Inbox: unresolved intake jobs, low-confidence matches, duplicate candidates, and provider conflicts.
- History/detail: canonical timeline, progress, notes, ratings, rereads/rewatches, and source evidence.

The user should be able to capture now and fix later. The system should be honest about confidence: "matched by TMDb ID" is strong; "matched by title/year" needs a visible source and an easy
correction path.

### Server Responsibilities

When the bot or UI adds a record, `iroha-server` should own the whole durable workflow:

- Persist the original payload before parsing.
- Normalize times, timezone, title text, progress units, rating scale, and source kind.
- Resolve media identity from existing refs first.
- Query enrichment providers only when local identity is missing or stale.
- Store enrichment responses as raw payloads.
- Create or update canonical media items and external refs.
- Append consumption events; do not overwrite history.
- Recompute current progress as a projection.
- Detect duplicates and ask for confirmation when confidence is low.
- Return a compact result that a bot can display.

This keeps external clients thin. The Telegram bot, browser extension, or share-sheet client only needs to authenticate and send user intent; Iroha remains the source of parsing, matching, dedupe, and
persistence.

### Suggested Intake API

External clients should call one small API and let the server do the rest:

```http
POST /api/v1/media/intake
Authorization: Bearer <upload-client-token>
Content-Type: application/json
```

```json
{
  "source_kind": "telegram",
  "source_event_id": "telegram:update:123456",
  "text": "watched Dune Part Two tonight 4.5/5",
  "occurred_at": "2026-07-08T21:30:00+09:00",
  "timezone": "Asia/Tokyo",
  "hints": {
    "media_type": "movie",
    "status": "completed",
    "rating": 4.5,
    "rating_scale": 5
  }
}
```

Response when accepted:

```json
{
  "intake_id": "min_<uuidv7>",
  "status": "completed",
  "media_item_id": "med_<uuidv7>",
  "event_id": "mev_<uuidv7>",
  "display": "Added Dune: Part Two as watched on 2026-07-08"
}
```

Response when ambiguous:

```json
{
  "intake_id": "min_<uuidv7>",
  "status": "needs_confirmation",
  "question": "Which one?",
  "candidates": [
    { "candidate_id": "cand_1", "title": "Dune: Part Two", "year": 2024, "media_type": "movie", "provider": "tmdb" },
    { "candidate_id": "cand_2", "title": "Dune", "year": 2021, "media_type": "movie", "provider": "tmdb" }
  ]
}
```

Confirmation should also be an API call, not bot-owned state:

```http
POST /api/v1/media/intake/{intakeId}/confirm
```

```json
{
  "candidate_id": "cand_1"
}
```

Iroha should then store the confirmation, apply the event, and return a bot-sized display string.

## Metadata We Need

### Work

Required:

- `id`
- `work_kind`: `franchise`, `story`, `series`, `single_work`, `anthology`
- `primary_title`
- `original_title`
- `original_language`
- `first_release_date`
- `description`
- `created_at`
- `updated_at`

Useful:

- `origin_country`
- `genres`
- `tags`
- `created_from_provider`

### Media Item

Required:

- `id`
- `work_id`
- `parent_item_id`
- `media_type`: `light_novel`, `book`, `manga`, `comic`, `anime`, `anime_season`, `episode`, `ova`, `ona`, `special`, `movie`, `recap_movie`, `live_action`, `game`, `article`, `video`, `podcast`,
  `podcast_episode`, `audiobook`
- `item_role`: `series`, `season`, `volume`, `chapter`, `episode`, `edition`, `cut`, `standalone`
- `title`
- `sort_title`
- `original_title`
- `language`
- `release_date` or `release_year`
- `created_at`
- `updated_at`

Useful:

- `season_number`
- `episode_number`
- `chapter_number`
- `volume_number`
- `edition_name`
- `duration_seconds`
- `page_count`
- `chapter_count`
- `episode_count`
- `publisher`
- `studio`
- `content_rating`
- `cover_image_url`
- `isbn_10`, `isbn_13`
- `runtime_source`: local file, provider metadata, manual

### Titles and Translations

Titles are first-class records, not columns only on the work/item. This is required for Japanese, Chinese, English, romanized, provider-specific, and user-local aliases.

Required:

- `id`
- `scope_type`: `work` or `item`
- `scope_id`
- `title`
- `language`: BCP-47 where possible, for example `ja`, `ja-Latn`, `zh-Hans`, `zh-Hant`, `en`
- `script`
- `region`
- `title_kind`: `original`, `localized`, `official`, `romanized`, `alias`, `search_alias`, `user_alias`, `provider`
- `provider`
- `is_primary`
- `confidence`
- `created_at`

Display rule:

```text
preferred user locale title
  -> official localized title
  -> romanized title
  -> original title
  -> provider title
```

Search must query all title rows. A query for `芙莉莲`, `Frieren`, `Sousou no Frieren`, or `葬送のフリーレン` should return the same work, then let the user choose the concrete item if needed.

### Relations

Relations connect concrete items and works without pretending they are the same thing.

Required:

- `id`
- `from_type`: `work` or `item`
- `from_id`
- `to_type`: `work` or `item`
- `to_id`
- `relation_type`
- `provider`
- `confidence`
- `created_at`

Relation types:

- `contains`
- `part_of`
- `adaptation_of`
- `adapted_as`
- `source_material_for`
- `sequel`
- `prequel`
- `side_story`
- `spin_off`
- `ova_of`
- `special_of`
- `movie_of`
- `recap_of`
- `compilation_of`
- `alternate_version_of`
- `same_continuity_as`
- `remake_of`
- `localized_edition_of`

Examples:

```text
anime season -> adaptation_of -> manga series
episode 12 -> part_of -> anime season 1
volume 3 -> part_of -> manga series
movie -> side_story -> anime season
recap movie -> recap_of -> anime season
Chinese translated edition -> localized_edition_of -> original book edition
```

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

Use this table aggressively. Media identities are messy: a book work differs from an edition; a show differs from an episode; a YouTube upload can later be renamed; manga scan/source titles drift.
Provider refs are the dedupe backbone.

### Consumption Event

Use append-only events, then derive current status.

Required:

- `id`
- `media_item_id`
- `event_type`: `started`, `progressed`, `completed`, `abandoned`, `reopened`, `rewatched`, `reread`, `rated`, `noted`, `bookmarked`, `hidden_from_continue`
- `event_at`
- `source_kind`: manual, telegram, web_manual, csv, anilist, bangumi, jellyfin, plex, trakt, komga, audiobookshelf, openlibrary, browser, rss
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
- AniList list import if OAuth is available, or user-provided export/backup if not.
- Bangumi.tv collection import through its API if OAuth/token access is available.
- Letterboxd CSV import for films.
- Goodreads/StoryGraph-style CSV import for books if the user can export it.
- Browser bookmark/history import for articles only after the schema is stable.

These are the safest first sources because they produce raw files Iroha can preserve and reprocess.

### Phase 2: Personal Tracker API Connectors

These are closer to Iroha's product goal than generic metadata APIs because they contain the user's own state.

- AniList: use its GraphQL API for anime/manga list state. The relevant user-list shape is media type (`ANIME`, `MANGA`), list status (`CURRENT`, `PLANNING`, `COMPLETED`, `DROPPED`, `PAUSED`,
  `REPEATING`), score, progress, progress volumes, repeat count, private flag, notes, custom lists, hidden-from-status-lists, started date, completed date, created timestamp, updated timestamp, and
  linked media metadata.
- Bangumi.tv: use its OpenAPI collection endpoints as a first-class source. User subject collections include subject ID/type, collection type, rating, comment, tags, episode progress, volume progress,
  updated timestamp, private flag, and slim subject metadata. The write API can modify collection type, rating, episode/volume progress, comment, privacy, and tags.
- Trakt: useful for movie/show watched history and watchlists if the user already uses it.
- Letterboxd: start with CSV export before considering scraping.

For AniList and Bangumi, Iroha should treat imported list entries as user-state events and current progress, not just metadata. A later connector sync should not erase locally added Telegram/web
events; conflicts go to the inbox.

### Phase 3: Self-Hosted App Connectors

- Jellyfin: import watched state, favorite/rating state, playback progress, play count, last played, and item identity. This fits Iroha well because Jellyfin already separates media items from user
  item data.
- Komga: import book/comic library items plus read status and progress.
- Audiobookshelf: import media progress, bookmarks, continue-listening visibility, started/finished timestamps, and podcast/audiobook item metadata.
- Seerr/Jellyseerr/Overseerr: import watchlist/blocklist/request/library-availability signals only. Do not treat it as watched history.

Self-hosted connectors should store connector snapshots as raw JSON, then normalize. That preserves evidence and lets parser fixes re-run.

### Phase 4: Metadata Enrichment APIs

- TMDb for movie, TV, person, image, and release metadata.
- Open Library for books and covers. Respect its API guidance: cache responses, identify the app, and do not scrape HTML.
- AniList for anime and manga metadata through GraphQL when it is not already the user-state source.
- Bangumi.tv for anime, manga, game, music, and real-subject metadata when it is not already the user-state source.
- MusicBrainz/ListenBrainz only if music listening history enters scope later.
- RSS/Atom feeds for article and podcast episode metadata.

Metadata enrichment should be idempotent and replaceable. External provider payloads should not overwrite user-entered titles, notes, ratings, or completion history unless the user explicitly accepts
the match.

### Phase 5: Activity Capture

- Browser extension or share-sheet endpoint for article saves and read-later events.
- Local webhook endpoint for media players that can emit play/pause/progress events.
- Optional Trakt-style sync import/export for watched history interoperability.

This should come after manual/file imports because live activity capture creates dedupe, privacy, and conflict-resolution work.

## Normalization Rules

- Preserve every raw import or connector response.
- Normalize to stable items and append-only events.
- Keep current progress as a projection.
- Dedupe by provider refs first, then by conservative fallback matches.
- Distinguish work, adaptation, edition, and part:
  - franchise/story work vs concrete manga/anime/light-novel/movie items
  - source work vs adaptation
  - book work vs ISBN edition
  - translated edition vs original edition
  - show/anime series vs season vs episode vs OVA vs movie
  - manga series vs volume vs chapter
  - podcast show vs episode
- Allow multiple completions. Rewatching/rereading should create new events, not overwrite old ones.
- Do not assume every source has exact timestamps. Some imports only know date, not time.
- Keep source confidence. A TMDb match found by title/year is weaker than a direct TMDb ID.
- Separate private history from public summaries. Public summaries should aggregate counts, time, pages, tags, and favorites without exposing raw notes or exact URLs unless explicitly published.

## Stable Schema

The stable schema is intentionally broader than the first UI. It should be able to represent provider sync, manual capture, translations, editions, parts, adaptations, rereads, rewatches, and later
write-back without destructive migrations.

```sql
create table tb_media_works (
  id uuid primary key,
  work_kind text not null,
  primary_title text not null,
  original_title text not null default '',
  original_language text not null default '',
  first_release_date date,
  description text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_items (
  id uuid primary key,
  work_id uuid references tb_media_works(id),
  parent_item_id uuid references tb_media_items(id),
  media_type text not null,
  item_role text not null,
  title text not null,
  sort_title text not null default '',
  original_title text not null default '',
  description text not null default '',
  release_date date,
  season_number integer,
  episode_number integer,
  chapter_number numeric,
  volume_number numeric,
  duration_seconds integer,
  page_count integer,
  episode_count integer,
  chapter_count integer,
  language text not null default '',
  country text not null default '',
  cover_image_url text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_titles (
  id uuid primary key,
  scope_type text not null,
  scope_id uuid not null,
  title text not null,
  language text not null default '',
  script text not null default '',
  region text not null default '',
  title_kind text not null,
  provider text not null default '',
  is_primary boolean not null default false,
  confidence numeric,
  created_at timestamptz not null
);

create index idx_tb_media_titles_title on tb_media_titles using gin(to_tsvector('simple', title));
create index idx_tb_media_titles_scope on tb_media_titles(scope_type, scope_id);

create table tb_media_relations (
  id uuid primary key,
  from_type text not null,
  from_id uuid not null,
  to_type text not null,
  to_id uuid not null,
  relation_type text not null,
  provider text not null default '',
  confidence numeric,
  created_at timestamptz not null,
  unique(from_type, from_id, to_type, to_id, relation_type, provider)
);

create table tb_media_external_refs (
  id uuid primary key,
  scope_type text not null,
  scope_id uuid not null,
  provider text not null,
  external_id text not null,
  external_url text not null default '',
  confidence numeric,
  matched_by text not null default '',
  created_at timestamptz not null,
  unique(provider, external_id)
);

create table tb_media_creators (
  id uuid primary key,
  name text not null,
  sort_name text not null default '',
  original_name text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_creator_roles (
  id uuid primary key,
  creator_id uuid not null references tb_media_creators(id) on delete cascade,
  scope_type text not null,
  scope_id uuid not null,
  role text not null,
  provider text not null default '',
  created_at timestamptz not null
);

create table tb_intake_payloads (
  id uuid primary key,
  source_kind text not null,
  source_actor text not null,
  source_event_id text not null default '',
  content_type text not null,
  sha256 text not null,
  storage_path text not null default '',
  payload_json jsonb,
  received_at timestamptz not null,
  parsed_at timestamptz
);

create table tb_media_intake_jobs (
  id uuid primary key,
  intake_payload_id uuid not null references tb_intake_payloads(id),
  status text not null,
  parser_kind text not null,
  error_message text,
  created_at timestamptz not null,
  finished_at timestamptz
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
  intake_payload_id uuid,
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

create table tb_media_lists (
  id uuid primary key,
  name text not null,
  list_kind text not null,
  source_kind text not null default '',
  external_ref_id uuid references tb_media_external_refs(id),
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create table tb_media_list_items (
  id uuid primary key,
  list_id uuid not null references tb_media_lists(id) on delete cascade,
  media_item_id uuid not null references tb_media_items(id) on delete cascade,
  position numeric,
  created_at timestamptz not null,
  unique(list_id, media_item_id)
);

create table tb_media_resolution_tasks (
  id uuid primary key,
  intake_job_id uuid references tb_media_intake_jobs(id),
  task_type text not null,
  status text not null,
  candidates_json jsonb not null default '[]'::jsonb,
  resolution_json jsonb not null default '{}'::jsonb,
  created_at timestamptz not null,
  resolved_at timestamptz
);
```

Polymorphic `scope_type/scope_id` references are deliberate here. Provider refs, titles, creators, and relations can target either an abstract work or a concrete item. Application code must enforce
valid scope IDs because SQL foreign keys cannot directly target multiple tables.

## Stable Workflow

All sources use the same server-owned workflow:

```text
receive payload
  -> store tb_intake_payloads
  -> create tb_media_intake_jobs
  -> parse source-specific shape into candidate facts
  -> resolve provider refs
  -> resolve or create works/items/titles/relations
  -> append consumption events
  -> update progress projection
  -> create resolution tasks for ambiguity/conflicts
  -> return UI/bot-sized result
```

Provider-specific examples:

```text
AniList MediaList entry
  -> external ref on anime/manga item
  -> titles from media metadata
  -> list state to progress projection
  -> score/notes/private/custom lists to events/lists
```

```text
Bangumi collection
  -> external ref on subject item
  -> subject metadata to titles/item fields
  -> collection type to progress status
  -> ep_status/vol_status to progress unit and position
  -> rate/comment/tags/private to events/progress/list metadata
```

```text
Telegram quick add
  -> raw intent payload
  -> parser extracts candidate title, status, progress, rating, date
  -> title search across tb_media_titles
  -> provider search if no local match
  -> needs_confirmation if multiple works/items match
  -> append event only after identity is resolved
```

Ambiguous natural language must resolve at the correct level:

```text
"finished Frieren"
  -> ask whether anime season, manga, light novel, movie, or another item

"finished Frieren ep 12"
  -> resolve to episode under anime season if title and episode number match

"read Frieren vol 3"
  -> resolve to manga/light-novel volume, not anime
```

## Source Notes

- Jellyfin describes itself as a free media system for managing and streaming media, and its API exposes user-data operations such as favorite/rating updates and latest-media queries with played
  filters and optional user data.
- Seerr is a request and discovery manager for Jellyfin/Plex/Emby; its feature list emphasizes user import, library scans, requests, watchlisting, and blocklisting rather than personal consumption
  history.
- Audiobookshelf explicitly models media progress updates with duration, percent progress, current time, finished state, continue-listening visibility, started time, and finished time; it also syncs
  local and server progress by `lastUpdate`.
- Komga's OpenAPI exposes book listing filters for `read_status` values including unread, read, and in-progress, and supports API-key/session authentication.
- AniList's GraphQL API exposes user media-list state for anime and manga, including status, score, progress, volume progress, repeat count, privacy, notes, custom lists, started/completed dates, and
  created/updated timestamps.
- Bangumi.tv's OpenAPI exposes user subject collections and collection modification endpoints, including subject identity/type, collection type, rating, comments, tags, episode and volume progress,
  privacy, and update timestamps.
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
- [Bangumi API repository](https://github.com/bangumi/api)
- [Bangumi OpenAPI schema](https://raw.githubusercontent.com/bangumi/api/master/open-api/v0.yaml)
