# Media Sync Connectors: AniList + Bangumi

> **Semantics amendment (2026-08-15):** the connector plumbing described here is retained, but the earlier list-state-to-event mapping is superseded by
> [ADR-0005](adr/0005-media-provider-time-semantics.md) and the [canonical-history redesign](plans/2026-08-15-media-provider-canonical-history.md). Current list snapshots become progress/state
> history; only exact evidence becomes a consumption event.

Implementation note (updated 2026-07-20). Companion to [`media-history-research.md`](./media-history-research.md), which defines the media _ontology_
(works/items/titles/relations/external_refs/consumption_events/progress) and the product direction. This document bridges that ontology to iroha's **actual provider abstraction** and specifies the
first two API connectors: AniList (GraphQL) and Bangumi.tv (REST v0).

The first connector slice is shipped: AniList and Bangumi fetchers, durable cursor state, raw snapshot persistence, media parsing, retry-aware worker jobs, and the private trigger endpoint are
implemented. The full ontology and cross-provider resolution work remain future scope.

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
reaches the network. `imports.Process` (`apps/iroha-imports/service.go`) dispatches media snapshots through `MediaImporter` and persists them under the same raw-evidence and parser-version rules as
health imports.

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
    SourceKind  string // "anilist" | "anilist_activity" | "bangumi" — routes to the media adapter
}
```

Ownership:

- **iroha-server** exposes `POST /api/v1/media/sync/{connectorId}` for configured `anilist` and `bangumi` connectors. An AniList run executes the current-list connector and then the bounded activity
  connector. It queues work and returns a typed job ID; it does not accept credentials in the request body.
- **iroha-job** runs the fetch loop. Credentials are resolved from worker environment variables, keeping secrets out of `tb_jobs` and browser traffic.
- **Scheduling** reuses the existing durable `tb_jobs` + `EnqueueDueSchedules` interval mechanism (jobs.Service already supports interval schedules with `ClaimNext` + `FOR UPDATE SKIP LOCKED`). A
  media sync is a scheduled job kind (`media_sync_anilist`, `media_sync_bangumi`) that runs `Fetch` in a loop until the cursor is exhausted, creating one import job per snapshot page.
- **Cursor durability**: store per-connector sync cursor (page number / `updatedAfter` watermark) in a small `tb_media_sync_state` row so incremental syncs resume and only pull changed entries.
- **AniList activity window**: the first run defaults to 365 days; set `IROHA_ANILIST_ACTIVITY_LOOKBACK_DAYS` on the worker to choose another bounded backfill window. Successful runs retain a 24-hour
  overlap cursor and deduplicate by activity ID.

Connectors are registered in a connector registry sibling to the provider registry (`iroha-providers/registry`), and their snapshots' `SourceKind` must match a registered media importer or
media-history importer adapter's declared source kind.

## 3. Media dispatch (shipped)

The formerly planned dispatch and persistence changes are complete:

1. **Dispatch** — `imports.Process` type-asserts `provider.MediaImporter` for current-list snapshots and `provider.MediaHistoryImporter` for dated provider activity; media is not carried by
   `ImportBatch`.
2. **Persistence** — `persistMedia(...)` writes canonical media rows under the same snapshot dedupe and parser-version reprocess discipline used by health imports.

The `anilist`, `anilist_activity`, and `bangumi` source kinds are registered in the parser/provider registries and accepted by `imports.Create`.

## 4. Full-schema adoption

Adopt the research doc's stable schema (`tb_media_works`, `tb_media_items`, `tb_media_titles`, `tb_media_relations`, `tb_media_external_refs`, `tb_media_creators`/`_creator_roles`,
`tb_intake_payloads`, `tb_media_intake_jobs`, `tb_media_consumption_events`, `tb_media_progress`, and `tb_media_lists`/`_list_items`) as one migration. Notes for this
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
  - **state + progress**: list status → progress projection/state observation; score/notes/dates/repeat-count remain sourced current state. They do not become consumption events without an exact event
    time. AniList `ListActivity` is a dated provider-update source with the provider's exact `createdAt`, never a consumption session.

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
  - list `status` → progress `status` (CURRENT→in_progress, COMPLETED→completed, PLANNING→planned, DROPPED→abandoned, PAUSED→in_progress+paused flag, REPEATING→in_progress).
  - `progress`/`progressVolumes` → progress position (unit episodes/chapters/volumes).
  - `score` → rating (carry AniList's scale; normalize at read time, not on ingest).
  - `startedAt`/`completedAt` → fuzzy progress dates; `updatedAt` → provider-recorded state metadata; `repeat` → current repeat count, not N historical rewatch events.

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
  - `updated_at` → retained untrusted provider metadata; it is not a canonical event time or ordering watermark.

## 7. Identity & cross-provider dedup

`tb_media_external_refs (provider, external_id) unique` is the dedupe backbone. Resolution order when an observation arrives:

1. Match on `(provider, external_id)` → existing item. Done.
2. Match via a **bridge ref** → existing item; attach the new provider ref. AniList media carries `idMal` in-band (verified). Bangumi has no in-band ref, so bridge it via a static dataset: **Bangumi
   subject id → MAL id** through [Rhilip/BangumiExtLinker](https://github.com/Rhilip/BangumiExtLinker) (Bangumi-keyed, exposes `mal_id`, CC BY 4.0), then **MAL id → AniList id** through
   [Fribb/anime-lists](https://github.com/Fribb/anime-lists) (`mal_id`↔`anilist_id`, verified). Cache both datasets locally and refresh periodically.
3. Title match across `tb_media_titles`, scoped to the same `media_type` + `item_role` (an anime season and its manga adaptation must never merge) and a release date within **±400 days** (providers
   routinely disagree on which event anchors a work's release date by several months, so an exact-year match misses real matches). The title comparison itself NFKC-folds both sides (collapsing
   fullwidth punctuation like `～`/`（）` to ASCII) and strips bracketed annotations (`(...)`, `《...》`) before comparing, since providers routinely render the same title with a trailing reading
   gloss kept on one side and dropped on the other, or the same in-title gloss in a different bracket style — a plain lowercase/whitespace-normalized string match missed these in production. Exactly
   one candidate → **auto-attach** to it (`matched_by = title_year`, confidence 0.7). Two or more candidates → keep a fresh source-owned item and preserve the ambiguity for agent-readable inspection; do not block the import or guess.
4. No match → create new work + item + titles + ref, `matched_by = provider_id`.

Cross-provider linking is what makes AniList + Bangumi complementary rather than duplicative: AniList supplies `idMal` and romaji/english titles; Bangumi supplies Chinese titles and its own subject
id. When both connectors see the same anime, the two-hop bridge (Bangumi→MAL→AniList) converges them on one `tb_media_items` row with three provider refs and four+ title rows. **The bridge is
anime-only in practice, not just "not total"**: measured directly against production data (2026-08-06, `bangumi_to_mal.json` keys vs. every Bangumi ID actually synced), **0% of Bangumi manga IDs
appear in the bridge at all**, vs. 66% of anime IDs (matching the §11 spike number) — BangumiExtLinker's dataset simply doesn't cross-reference manga/light-novel subjects. Since manga is the majority
of most Bangumi+AniList libraries (§11: 61% book), **step 3 (title/date matching) is the primary cross-provider dedup mechanism for manga, not a long-tail fallback** — its correctness matters as much
as the bridge's, not less. The spike (`iroha:media-connector-spike`) quantified the anime tail before the resolver was committed; a 2026-08 production audit found step 3 itself had a real bug (see the
CHANGELOG `[0.2.0]` entries) once alternate (non-primary) titles, cross-year release dates, and provider-specific title formatting (bracketed reading glosses, differing bracket styles, fullwidth
punctuation, inconsistent subtitle spacing) were accounted for.

## 8. Sync semantics

- **Current implementation boundary**: the AniList and Bangumi adapters checkpoint their page cursor while a sync is running, so a retry resumes from the failed page. Their current upstream requests
  do not expose a reliable changed-since filter, so a completed sync starts a new full pagination pass. Persistence is still incremental: unchanged snapshot pages and source records are content-hash
  deduplicated, while changed and new records are upserted.
- **Future network incrementality**: persist a per-connector cursor/watermark once a provider-specific changed-since API is available; Bangumi's documented `updated_at` limitation means it cannot be
  used as a trusted historical watermark without an additional provider guarantee. Full re-sync remains available on demand (or on a parser_version bump → reprocess from stored snapshots).
- **Rate limits and failures**: a connector error preserves the current cursor. HTTP 429 responses parse both delta-seconds and HTTP-date `Retry-After` values; the job queue honors that delay, with
  bounded exponential backoff as the fallback.
- **User state is authoritative current state, not a mirror**: a connector sync must **not** erase locally added Telegram/web events. Imported list state becomes current progress plus provider state
  history; conflicts (e.g. local "completed" vs remote "dropped") are retained as source observations and resolved by a deterministic domain rule, never by an inbox or arrival-order overwrite.
- **Rewatches/rereads**: AniList `repeat` and re-`completed` state are retained as counts/status/date facts. They become new events only when the source supplies an exact event or the user records
  one.
- **Reprocess**: bump the media parser_version to re-derive canonical rows from the same snapshots after a mapping fix — no re-fetch, no lost evidence, identical to apple-health reprocess-from-raw.

## 9. Auth & config

- AniList public username and Bangumi username/token are currently deployment credentials supplied to `iroha-job` through environment variables (not raw files, `tb_jobs`, or git). Per-user account
  storage and AniList OAuth remain future work. Local dev stays `LocalNoAuth`.
- Both connectors must send a descriptive `User-Agent` identifying iroha. AniList's Cloudflare **403s the default `Python-urllib`/no-UA request** (verified), so a UA is mandatory there; Bangumi reads
  work without one but send it anyway.

### Bridge cache

`make media-bridge-build` (`scripts/build_media_bridge.py`) fetches BangumiExtLinker + Fribb (§7) and upserts `tb_media_ref_bridge` (migration `00012_media_ref_bridge.sql`) — one row per
`(hop, source_id) -> target_id`, `hop` being `bangumi_to_mal` or `mal_to_anilist`, since `TwoHopMediaRefBridge.Lookup` (`apps/iroha-imports/media_resolution.go`) compares provider IDs as strings
throughout. `iroha-job` loads both hops into an in-memory map once at startup via `LoadTwoHopMediaRefBridgeFromDB`; an empty or unpopulated table degrades the same way an unset bridge always has —
that hop is simply skipped and unresolved items fall through to the non-blocking title+year matching rule (§7 step 3). This used to be two ConfigMap-mounted JSON files generated locally and committed to harus-k3s
(`IROHA_BANGUMI_BRIDGE_PATH` / `IROHA_MAL_ANILIST_BRIDGE_PATH`); moved into Postgres 2026-08-16 so a refresh is a normal DB write instead of a rebuild-and-redeploy cycle, and the table is queryable
and incrementally upsertable rather than replaced whole. The worker refreshes this crosswalk automatically once a week; `make media-bridge-build` remains the local/manual rebuild path. The anime tail in §11 is mostly recent seasonal anime
the upstream datasets haven't mapped yet, so that tail shrinks the closer to "now" the table was last refreshed — but **rebuilding will not help manga coverage**, which is 0% regardless of freshness
(§7). Don't read "bridge cache" as "the general cross-provider dedup mechanism"; for manga it isn't in the loop at all.

## 10. Delivery status and remaining work

Ordered smallest → biggest to build momentum; each ends green on `make check`.

1. **Shipped** — schema, media dispatch/persistence, connector contract, cursor state, AniList/Bangumi pagination, raw snapshot evidence, worker retry handling, private sync trigger, full ontology
   (titles, external refs, work/item linkage, events, progress projections), the bridge cache build/deploy, and cross-provider dedup auto-attach (§7).
2. **Shipped** — the worker refreshes the bridge dataset weekly while preserving the manual rebuild path; ambiguous identities stay source-owned, and an explicit agent decision can attach, keep them separate, or undo a prior attach. The decision is append-only and consumed before bridge/title matching, so replays and bridge refreshes preserve it. No routine human resolution inbox is created.
3. **Later** — connector account storage (per-user credentials instead of deployment-wide env vars) and richer agent-readable inspection/reconciliation tooling.

Deferred (explicitly out of this draft's connector scope): Telegram/web natural-language quick-add and `tb_intake_payloads`; Goodreads/WeRead/Apple Books/Kindle adapters; Letterboxd CSV; TMDb/Open
Library enrichment; self-hosted (Jellyfin/Komga/Audiobookshelf) connectors; the web media surfaces (quick-add/history).

## 11. Spike results (2026-07-13, verified on real accounts)

Measured by `scripts/anilist_explore.py`, `scripts/bangumi_explore.py`, `scripts/media_bridge_explore.py` against AniList `azusachino` (512 anime / 1062 manga) and Bangumi `mikufan2039` (425
collections). See epic `iroha:media-connector-spike`.

- **v1 must include book/manga.** Bangumi is **61% book** (261) vs 136 anime — anime-only would drop two-thirds of the collection. `media_type` maps Bangumi `subject_type` 1→manga/light_novel
  (disambiguate via `platform`), 2→anime; AniList `format` covers TV/TV_SHORT/MOVIE/OVA/ONA/SPECIAL/MANGA/ONE_SHOT/NOVEL.
- **Ratings are sparse** (AniList score 6–11%, Bangumi rate 1%) — `score`/`rating` must be nullable and never required.
- **AniList→MAL bridge is reliable**: `idMal` coverage 100% anime / 97% manga.
- **Bangumi→AniList auto-bridge = ~66%** of anime (two-hop via BangumiExtLinker + Fribb). The **~34% tail is almost entirely 2024–2026 seasonal anime** the datasets haven't mapped yet — so title+year
  matching must remain automatic and non-blocking, with ambiguous cases retained for agent-readable inspection. Cache both datasets locally and refresh periodically.
- **Bangumi→AniList auto-bridge = 0% of manga** (measured 2026-08-06 against every Bangumi manga ID in a real production sync: none present in `bangumi_to_mal.json`). BangumiExtLinker doesn't
  cross-reference manga/light-novel subjects at all, so freshness doesn't help here the way it does for anime — title+date matching is the _only_ cross-provider dedup path for manga, not a fallback.
- **Dedupe is load-bearing**: 63 of the 136 Bangumi anime are also in the AniList list (same `idMal`) — real cross-account collisions the `external_refs` ladder must merge onto one item.

## 12. Drifter reading gateway (2026-11-11 amendment)

WeRead reading state now arrives through **Drifter (`drifter-server`)**, the provider-neutral reading companion adopted in the
[ADR 0011 amendment](adr/0011-provider-neutral-reading-evidence.md). This supersedes the "WeRead adapter" line in §10's deferred list for reading state; the
[integration plan](plans/2026-11-11-drifter-reading-gateway-integration.md) stages the rest.

Shipped in this slice:

- **Connector + adapter split, unchanged.** `apps/iroha-providers/drifter/connector.go` fetches one page of `GET /api/v1/reading/snapshot?limit=N`
  (plus `&cursor=` while `has_more`) with `Authorization: Bearer <DRIFTER_API_TOKEN>` from `DRIFTER_BASE_URL`. Each page is stored verbatim as raw evidence under
  source kind `drifter_reading_snapshot`; the adapter is a pure parser over those bytes. Worker job kind: `media_sync_drifter`, connector id `drifter`.
- **The gateway names its own accounts.** No `account=` parameter is sent: the page's `source_identity`/`target_identity` fields carry that fact, so a request
  cannot silently fall back to a server default the owner never chose.
- **Replication follows a watermark, not a page cursor.** The page envelope carries `watermark` (the highest observed source timestamp) and the connector stores
  it on the page's `tb_source_receipts.ordering_basis`. `GET /api/v1/reading/changes?since=<RFC3339>&limit=100` (plus `&cursor=` while `has_more`) serves only the
  books Drifter recorded at or after that watermark. `connector.ChangeFetcher` is the protocol contract; `connector.ErrSnapshotRequired` is the signal that the
  watermark is out of horizon.
- **Follower state machine in the sync runner.** `imports.SyncRunner` resumes from the latest receipt's ordering basis. No watermark means a bootstrap full
  snapshot (`ingestion_mode=full_snapshot`). A readable watermark means a changes run (`ingestion_mode=incremental`). A 410 Gone, or a `snapshot_required` body,
  makes the same run fall back to a full snapshot instead of failing. The sync-state cursor stays reserved for an interrupted full-snapshot pagination, so the two
  resume points never get confused.
- **Identity stays with the source provider.** A Drifter book is persisted as a `weread` item (`provider`/external ref `weread`/`source_book_id`), so a future
  direct WeRead path claims the same item instead of creating a copy. `isbn` and a confirmed `matched_target.edition_id` become additional external refs
  (`isbn`, `hardcover`).
- **Creators are persisted.** Author and translator credits become `tb_media_creators` rows linked through `tb_media_creator_roles` scoped to the item. A replayed
  snapshot reuses the existing creator and role rows instead of appending duplicates.
- **Publisher is an item column.** Migration `00028_media_item_publisher` adds `tb_media_items.publisher`; it is a per-edition fact, not a work fact.
- **Honest time semantics.** WeRead's real `finished_at` becomes a `finished` consumption event with the existing deterministic source event id; completion date
  uses the import timezone for its calendar day. `progress_percent`/`last_read_at` feed `tb_media_progress`. No `started` or session event is fabricated, and
  `reading_time_seconds` is retained in the stored snapshot only — Iroha has no typed reading-time field yet.
- **Source priority.** `mediaProgressSourcePriority` places `drifter`, `weread` and `hardcover` on the AniList tier (2), above Bangumi (1); owner-authored
  sources (manual/web/telegram) still win.
- **The sync schedule is configuration-driven.** With `DRIFTER_API_TOKEN` set, `iroha-job` reconciles one enabled `media_sync_drifter` schedule at startup;
  `IROHA_DRIFTER_SYNC_INTERVAL` (Go duration such as `12h`) sets the cadence and `off` disables it. Without the token no schedule is created and an existing one
  is disabled. Pages request `drifter.DefaultLimit` (10) books with a 60-second per-page timeout: upstream WeRead paces the gateway at one request per second, so a
  50-book page times out instead of paginating.

Not yet wired: a private API trigger for `drifter` syncs, work-level linking through `matched_target.work_id`, and the cockpit surfaces from the plan's slices 3–4.
