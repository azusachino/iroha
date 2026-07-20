# Media Sync Connectors: AniList + Bangumi

Implementation note (updated 2026-07-20). Companion to [`media-history-research.md`](./media-history-research.md), which defines the media _ontology_ (works/items/titles/relations/external_refs/consumption_events/progress) and
the product direction. This document bridges that ontology to iroha's **actual provider abstraction** and specifies the first two API connectors: AniList (GraphQL) and Bangumi.tv (REST v0).

The first connector slice is shipped: AniList and Bangumi fetchers, durable cursor state, raw snapshot persistence, media parsing, retry-aware worker jobs, and the private trigger endpoint are implemented. The full ontology and cross-provider resolution work remain future scope.

Scope decision (2026-07-13): adopt the **full stable schema** from the research doc up front, then build the two connectors on top of it. This avoids destructive migrations once translations,
editions, parts, adaptations, and rewatches arrive.

## 1. The pipeline: file-backed import and API-pull connector

The current import pipeline is entirely **file-backed**:

```text
client uploads raw file -> tb_raw_files (sha256, storage_path)
  -> imports.Create -> tb_import_jobs (queued) -> enqueue tb_jobs
  -> iroha-job worker -> imports.Process
       -> providers.GetBySourceKind(parser_kind) -> adapter.ImportActivities/Sleep/Daily/All
       -> persist observations -> tb_import_snapshots
  -> reuse/reprocess keyed on (sha256, parser_version)
```

`provider.Source` is a file handle: `Open(ctx) (io.ReadCloser, error)` over `rawFile.StoragePath`. Apple Health and GPX adapters are **pure parsers**: bytes in, `[]observations.X` out. No adapter
reaches the network. `imports.Process` (`apps/iroha-imports/service.go`) dispatches media snapshots through `MediaImporter` and persists them under the same raw-evidence and parser-version rules as health imports.

AniList and Bangumi break the file assumption: there is no user-uploaded file. The user's list state lives behind an authenticated API and must be **fetched and paginated** by iroha itself.

### Design principle: keep adapters pure; add a connector fetch stage

Rather than let a media adapter reach the network (which would make it untestable and break the raw-evidence invariant), we keep the adapter a **pure parser** and add one new stage in front of it:

```text
                         NEW                              EXISTING (unchanged shape)
schedule/trigger -> connector.Fetch(creds, cursor)  ->  snapshot bytes
  -> store snapshot as raw evidence (tb_raw_files or tb_intake_payloads)
  -> imports.Create(parser_kind = "anilist" | "bangumi")
  -> iroha-job -> imports.Process
       -> providers.GetBySourceKind -> adapter.ImportMedia(snapshot) -> []observations.Media...
       -> persist to tb_media_* -> tb_import_snapshots
  -> reuse/reprocess keyed on (sha256(snapshot), parser_version)
```

The connector owns **auth, pagination, rate-limiting, and evidence capture**. The adapter owns **parsing one snapshot into canonical observations**. This mirrors the apple/gpx split exactly — the only
new thing is _who produces the bytes_. The whole reuse/reprocess machine (sha256 + parser_version) works unchanged: a connector snapshot is content-addressed like any raw file, so re-fetching
identical list state is a cheap skip, and bumping the media parser version reprocesses from the stored snapshot.

## 2. The connector abstraction

A connector is a scheduled, credentialed producer of snapshots. Proposed contract, living beside the provider contracts in `iroha-core` (e.g. `iroha-core/connector/v1`):

```go
type Connector interface {
    // Descriptor identifies the connector and the provider source kind its
    // snapshots parse as (so the media adapter can be found by source kind).
    Descriptor() ConnectorDescriptor // ID, SourceKind, Domains, requires-auth

    // Fetch pulls the next page/window of remote state into a snapshot. It is
    // pure w.r.t. iroha state: given credentials + a cursor it returns bytes +
    // the next cursor (nil when the sync is complete). The caller persists the
    // snapshot as raw evidence and advances the cursor durably.
    Fetch(ctx context.Context, creds Credentials, cursor *Cursor) (Snapshot, *Cursor, error)
}

type Snapshot struct {
    ContentType string // application/json
    Body        []byte // the raw provider response, stored verbatim as evidence
    SourceKind  string // "anilist" | "bangumi" — routes to the media adapter
}
```

Ownership:

- **iroha-server** exposes `POST /api/v1/media/sync/{connectorId}` for configured `anilist` and `bangumi` connectors. It queues work and returns a typed job ID; it does not accept credentials in the request body.
- **iroha-job** runs the fetch loop. Credentials are resolved from worker environment variables, keeping secrets out of `tb_jobs` and browser traffic.
- **Scheduling** reuses the existing durable `tb_jobs` + `EnqueueDueSchedules` interval mechanism (jobs.Service already supports interval schedules with `ClaimNext` + `FOR UPDATE SKIP LOCKED`). A
  media sync is a scheduled job kind (`media_sync_anilist`, `media_sync_bangumi`) that runs `Fetch` in a loop until the cursor is exhausted, creating one import job per snapshot page.
- **Cursor durability**: store per-connector sync cursor (page number / `updatedAfter` watermark) in a small `tb_media_sync_state` row so incremental syncs resume and only pull changed entries.

Connectors are registered in a connector registry sibling to the provider registry (`iroha-providers/registry`), and their snapshots' `SourceKind` must match a registered `MediaImporter` adapter's
declared source kind.

## 3. Media dispatch (shipped)

The formerly planned dispatch and persistence changes are complete:

1. **Dispatch** — `imports.Process` type-asserts `provider.MediaImporter` and calls `ImportMedia(ctx, source, options)`; media is not carried by `ImportBatch`.
2. **Persistence** — `persistMedia(...)` writes canonical media rows under the same snapshot dedupe and parser-version reprocess discipline used by health imports.

The `anilist` and `bangumi` source kinds are registered in the parser/provider registries and accepted by `imports.Create`.

## 4. Full-schema adoption

Adopt the research doc's stable schema (`tb_media_works`, `tb_media_items`, `tb_media_titles`, `tb_media_relations`, `tb_media_external_refs`, `tb_media_creators`/`_creator_roles`,
`tb_intake_payloads`, `tb_media_intake_jobs`, `tb_media_consumption_events`, `tb_media_progress`, `tb_media_lists`/`_list_items`, `tb_media_resolution_tasks`) as one migration. Notes for this
codebase:

- **Reuse `tb_raw_files` for connector snapshots** initially rather than standing up `tb_intake_payloads` as a parallel intake path. A connector snapshot _is_ a content-addressed blob with a storage
  path; it fits `tb_raw_files` and inherits the sha256 dedupe + reprocess machinery for free. Add `tb_intake_payloads` only when the Telegram/web quick-add path (natural-language intent) lands — that
  path has no file and needs the generalized intake shape. Document this as the deliberate ceiling.
- **Polymorphic `scope_type`/`scope_id`** on titles/refs/relations/creator_roles is enforced in Go, not by FK (SQL can't FK to multiple tables). Match the existing model style.
- **UUIDv7 ids** with typed prefixes (`med_`, `mev_`, `min_`) via the existing `ids` package, consistent with activity/import ids.

### `observations.Media` must grow

Today `observations.Media` is thin (`Provider, ExternalID, MediaType, Title, Status, Progress, Score, StartedAt, CompletedAt`) — enough for a flat list, not the full ontology. To feed
works/items/titles/ external_refs/events/progress, the media observation the adapter emits needs to carry:

- **item identity**: media_type, item_role, parent linkage hints, release date/year, season/episode/volume numbers, duration/page/chapter counts.
- **titles[]**: `{title, language, script, title_kind, is_primary, provider}` — AniList and Bangumi both return native + romanized + english/localized titles; all become `tb_media_titles` rows.
- **external_refs[]**: `{provider, external_id, external_url, matched_by, confidence}` — AniList media carries `idMal`; Bangumi subjects can be cross-linked. This is the dedupe backbone.
- **work linkage**: enough to create/attach a `tb_media_works` row (series-level identity) above the item.
- **event + progress**: list status → progress projection; score/notes/dates/repeat-count → consumption events. List state is _user state_, so imported entries become events + current progress, not
  just metadata (per research doc).

Keep the emitted observation provider-neutral: both connectors map their native shape into the same `observations.Media` graph, and persistence is identical downstream.

## 5. AniList connector

- **API**: GraphQL, `https://graphql.anilist.co`. Public schema; no OAuth needed for _public_ lists (query by username). OAuth (implicit/pin) required for private entries — treat OAuth as a later
  capability; ship username-based public sync first.
- **Query**: `MediaListCollection(userName: $user, type: ANIME|MANGA)` returns lists → entries. Per entry: `status` (CURRENT/PLANNING/COMPLETED/DROPPED/PAUSED/REPEATING), `score`, `progress`,
  `progressVolumes`, `repeat`, `private`, `notes`, `customLists`, `hiddenFromStatusLists`, `startedAt`, `completedAt`, `createdAt`, `updatedAt`, and nested
  `media { id, idMal, type, format, title{romaji english native}, episodes, chapters, volumes, startDate, coverImage, ... }`.
- **Pagination**: `MediaListCollection` returns whole lists (chunked via `chunk`/`perChunk` for large libraries). Fetch ANIME and MANGA separately; one snapshot per (user, type, chunk).
- **Rate limits**: AniList is ~90 req/min (degraded budgets possible); honor `Retry-After` and back off. Map 429 to `provider.ErrorRateLimited`.
- **Mapping**:
  - `media.type` + `media.format` → `media_type` (`ANIME`+`TV`→anime_season, `MOVIE`→movie, `MANGA`→manga, `NOVEL`→light_novel, `OVA`/`ONA`/`SPECIAL`→respective).
  - `title.{romaji,english,native}` → three `tb_media_titles` rows (romanized/localized/original).
  - external refs: `{provider: anilist, external_id: media.id}` + `{provider: mal, external_id: idMal}`.
  - list `status` → progress `status` (CURRENT→in_progress, COMPLETED→completed, PLANNING→planned, DROPPED→abandoned, PAUSED→in_progress+paused flag, REPEATING→in_progress + a rewatch event).
  - `progress`/`progressVolumes` → progress position (unit episodes/chapters/volumes).
  - `score` → rating (carry AniList's scale; normalize at read time, not on ingest).
  - `startedAt`/`completedAt` → event timestamps; `repeat` → N rewatch/reread events.

## 6. Bangumi.tv connector

- **API**: REST, OpenAPI v0, `https://api.bgm.tv`. Auth via personal access token (Bearer) for private collections; public collections readable by username. Read endpoints work without a `User-Agent`
  (verified: subject reads return 200 with empty/generic/descriptive UA), but send a descriptive UA identifying iroha as good practice. The subject `infobox` carries no MAL/AniList id — cross-linking
  needs an external dataset (see §7), not an in-band ref.
- **Endpoints**: `GET /v0/users/{username}/collections` (paginated `limit`/`offset`). Per row: subject id/type, `type` (collection type: wish/collect/do/on_hold/dropped), `rate`, `comment`, `tags`,
  `ep_status`, `vol_status`, `private`, `updated_at`, and slim `subject` metadata (name, name_cn, images, eps, volumes, date). Enrich per-subject via `GET /v0/subjects/{id}` when the slim shape is
  insufficient.
- **Pagination**: `limit`/`offset`; snapshot per page. Use `updated_at` as the incremental watermark.
- **Rate limits**: be conservative; single-user sync is low volume. Map 429 → `ErrorRateLimited`.
- **Mapping**:
  - subject `type` → media_type (2=anime, 1=book/manga, 4=game, 3=music, 6=real). Ship anime + book first; game/music/real map to media_type but stay lower priority.
  - `name` (original, usually Japanese) + `name_cn` (Chinese localized) → two `tb_media_titles` rows (original + localized `zh-Hans`). This is the reason titles are first-class: Bangumi is the
    strongest Chinese-title source, AniList the strongest romaji/english source — the same work is deduped across them via external refs and surfaced under all titles.
  - external ref: `{provider: bangumi, external_id: subject.id}`.
  - collection `type` → progress status (do→in_progress, collect→completed, wish→planned, dropped→abandoned, on_hold→in_progress+paused).
  - `ep_status`/`vol_status` → progress position (unit episodes / volumes).
  - `rate` (0–10) → rating; `comment`/`tags` → event judgment fields.
  - `updated_at` → event/progress `last_update_at`.

## 7. Identity & cross-provider dedup

`tb_media_external_refs (provider, external_id) unique` is the dedupe backbone. Resolution order when an observation arrives:

1. Match on `(provider, external_id)` → existing item. Done.
2. Match via a **bridge ref** → existing item; attach the new provider ref. AniList media carries `idMal` in-band (verified). Bangumi has no in-band ref, so bridge it via a static dataset: **Bangumi
   subject id → MAL id** through [Rhilip/BangumiExtLinker](https://github.com/Rhilip/BangumiExtLinker) (Bangumi-keyed, exposes `mal_id`, CC BY 4.0), then **MAL id → AniList id** through
   [Fribb/anime-lists](https://github.com/Fribb/anime-lists) (`mal_id`↔`anilist_id`, verified). Cache both datasets locally and refresh periodically.
3. Conservative title+year match across `tb_media_titles` → **low-confidence** candidate → create a `tb_media_resolution_task` (inbox) rather than auto-merging.
4. No match → create new work + item + titles + ref, `matched_by = provider_id`.

Cross-provider linking is what makes AniList + Bangumi complementary rather than duplicative: AniList supplies `idMal` and romaji/english titles; Bangumi supplies Chinese titles and its own subject
id. When both connectors see the same anime, the two-hop bridge (Bangumi→MAL→AniList) converges them on one `tb_media_items` row with three provider refs and four+ title rows. The bridge is not total
— CN databases split some works differently than MAL/AniList, so the split-entry tail falls through to step 3 (title+year → inbox). The spike (`iroha:media-connector-spike`) quantifies that tail
against a real collection before we commit the resolver.

## 8. Sync semantics

- **Current implementation boundary**: the AniList and Bangumi adapters checkpoint their page cursor while a sync is running, so a retry resumes from the failed page. Their current upstream requests do not expose a reliable changed-since filter, so a completed sync starts a new full pagination pass. Persistence is still incremental: unchanged snapshot pages and source records are content-hash deduplicated, while changed and new records are upserted.
- **Future network incrementality**: persist a per-connector cursor/watermark once a provider-specific changed-since API is available; then fetch only entries changed since the last `updated_at`. Full re-sync remains available on demand (or on a parser_version bump → reprocess from stored snapshots).
- **Rate limits and failures**: a connector error preserves the current cursor. HTTP 429 responses parse both delta-seconds and HTTP-date `Retry-After` values; the job queue honors that delay, with bounded exponential backoff as the fallback.
- **User state is authoritative history, not a mirror**: a connector sync must **not** erase locally added Telegram/web events. Imported list state becomes events + current progress; conflicts (e.g.
  local "completed" vs remote "dropped") go to the inbox (`tb_media_resolution_tasks`), never a silent overwrite. This is the same append-only-events + projection discipline health uses.
- **Rewatches/rereads**: AniList `repeat` and re-`completed` transitions create new events, not overwrites.
- **Reprocess**: bump the media parser_version to re-derive canonical rows from the same snapshots after a mapping fix — no re-fetch, no lost evidence, identical to apple-health reprocess-from-raw.

## 9. Auth & config

- AniList public username and Bangumi username/token are currently deployment credentials supplied to `iroha-job` through environment variables (not raw files, `tb_jobs`, or git). Per-user account storage and AniList OAuth remain future work. Local dev stays `LocalNoAuth`.
- Both connectors must send a descriptive `User-Agent` identifying iroha. AniList's Cloudflare **403s the default `Python-urllib`/no-UA request** (verified), so a UA is mandatory there; Bangumi reads
  work without one but send it anyway.

## 10. Delivery status and remaining work

Ordered smallest → biggest to build momentum; each ends green on `make check`.

1. **Shipped** — schema, media dispatch/persistence, connector contract, cursor state, AniList/Bangumi pagination, raw snapshot evidence, worker retry handling, and private sync trigger.
2. **Next** — broaden observations to the full ontology (titles, external refs, work/item linkage, events, and progress projections).
3. **Later** — cross-provider dedup/inbox, bridge dataset refresh, connector account storage, and web inbox UI.

Deferred (explicitly out of this draft's scope): Telegram/web natural-language quick-add and `tb_intake_payloads`; Letterboxd/Goodreads/WeRead CSV; TMDb/Open Library enrichment; self-hosted
(Jellyfin/Komga/Audiobookshelf) connectors; the web media surfaces (quick-add/inbox/history).

## 11. Spike results (2026-07-13, verified on real accounts)

Measured by `scripts/anilist_explore.py`, `scripts/bangumi_explore.py`, `scripts/media_bridge_explore.py` against AniList `azusachino` (512 anime / 1062 manga) and Bangumi `mikufan2039` (425
collections). See epic `iroha:media-connector-spike`.

- **v1 must include book/manga.** Bangumi is **61% book** (261) vs 136 anime — anime-only would drop two-thirds of the collection. `media_type` maps Bangumi `subject_type` 1→manga/light_novel
  (disambiguate via `platform`), 2→anime; AniList `format` covers TV/TV_SHORT/MOVIE/OVA/ONA/SPECIAL/MANGA/ONE_SHOT/NOVEL.
- **Ratings are sparse** (AniList score 6–11%, Bangumi rate 1%) — `score`/`rating` must be nullable and never required.
- **AniList→MAL bridge is reliable**: `idMal` coverage 100% anime / 97% manga.
- **Bangumi→AniList auto-bridge = ~66%** of anime (two-hop via BangumiExtLinker + Fribb). The **~34% tail is almost entirely 2024–2026 seasonal anime** the datasets haven't mapped yet — so title+year
  candidate + `tb_media_resolution_tasks` inbox is **required for an active watcher's recent list**, not a nicety. Cache both datasets locally and refresh periodically.
- **Dedupe is load-bearing**: 63 of the 136 Bangumi anime are also in the AniList list (same `idMal`) — real cross-account collisions the `external_refs` ladder must merge onto one item.
