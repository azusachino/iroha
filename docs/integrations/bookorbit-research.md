# BookOrbit and WeRead integration research

Date: 2026-10-12. Parent: [#142](https://github.com/azusachino/iroha/issues/142).

For the reopened issue's Hardcover/Kindle Unlimited handoff, read the later [usage and independent-source review](book-reading-hub-practices.md). It keeps the original evidence below intact while proposing source-independent ingestion and explicit writer ownership.

## Scope and evidence

This is source-backed preparation, not implementation or deployment. The owner scoped it as a single-session, tier-0 task: research, a staged plan, and acceptance issues, without Asobi or an independent verifier. The initial coordination setup was withdrawn. A read-only source scout had already completed; its findings were checked against source below. No upstream tests, live authenticated requests, real-device tests, or deployment were performed.

| Source | Pinned revision | Identification |
| --- | --- | --- |
| Iroha | `bb1df41b2e8a5f93c004edc80828fdc15fefe9c9` | Local `main` before this documentation branch |
| BookOrbit | `3e993af16b8ee0a3bee5a1b3a602113745de4457` | Upstream `main`, not a tested release/image |
| BookOrbit documentation | `5d0ed874b134f1dbf93e5d57398d1ca78a838830` | `bookorbit-site` |
| Tencent WeChatReading | `d98e66ee60a8c7bcd35da549d0643c4470439643` | Skill contract 1.0.5; README still says 1.0.4 |

Git checkouts were obtained over SSH and their actual full revisions recorded. BookOrbit's latest release endpoint reported `v3.3.0`, published 2026-10-07. This does not establish that the inspected `main` API matches that image. A future spike must pin the tested image digest and its corresponding source separately.

## Recommendation

Follow-up: the owner proposed a WeRead → companion → BookOrbit → Iroha topology. [The isolated companion feasibility test](weread-bookorbit-companion-feasibility.md) found that BookOrbit 3.3.0's tested ordinary write APIs cannot preserve lossless, private WeRead history: source provenance is rejected, replay changes timestamps, stale percent writes replace local progress, and shared custom metadata is not per-reader evidence. Metadata/status mirroring for already-matched local books is possible, but replacing the direct WeRead source requires an explicit reduced-goal decision or a supported upstream source-import extension.

Use a **version-pinned, authenticated application-API snapshot connector**, followed by Iroha's existing pure media parser and evidence pipeline. First import private book identity, current reading status, and reported progress. Keep the original source values and do not manufacture consumption events. Use bounded full scans until a reliable change feed exists.

Two independent decisions precede implementation:

1. Can the selected BookOrbit version and clients provide reliable daily cross-device reading? This decides whether to migrate from WeRead.
2. Can a selected provider supply useful, legally accessible evidence for Iroha? WeRead can be integrated independently of migration.

Neither decision has a runtime PASS yet. Read the [staged plan](../plans/2026-10-12-book-tracking-integration.md) for the approval and acceptance boundaries.

## BookOrbit integration surface

The global prefix is `/api/v1`. These are application routes, not a demonstrated stable third-party API. Controllers, service access checks, shared types, and response assembly were inspected. Optional Swagger is implemented at `/api/docs` and `/api/docs-json`; `SWAGGER_ENABLED` defaults to false. Generated OpenAPI uses the configured server version and marks non-public operations as Bearer-authenticated. Existence of Swagger does not promise compatibility across upgrades. [B1], [B2]

| Route | Available evidence | Scope / pagination |
| --- | --- | --- |
| `POST /api/v1/books/query` | `BooksPage`: `items`, `total`, `page`, `size`; cards include identity, files, authors, ISBN-13, rating, `readStatus`, and `readingProgress` | Read operation despite POST. Accessible libraries and current user's content filters applied. Zero-based page, size 1–200 (default 50), offset ceiling; disable series collapsing. |
| `GET /api/v1/books/:id` | Detailed metadata, ISBN-10/13, provider IDs, authors, files/hashes, status, rating | Checks book/library access; also returns private notes and filesystem paths. Do not collect those unnecessarily. |
| `GET /api/v1/books/:id/progress` | Array by file: `fileId`, percentage, CFI/page/seconds, Kobo/KOReader position fields, `updatedAt` | Checks book access and reads current user's records. No `lastReadAt` in this assembled response. |
| `GET /api/v1/books/files/:fileId/progress` | File-specific position; missing record has a zero/null fallback | A returned zero is not proof of a recorded reading action. Prefer card `readingProgress`, which distinguishes null from recorded zero. |
| `GET /api/v1/books/:bookId/sessions` | Paginated sessions plus aggregate statistics | One-based page; page size 1–100 (default 25), start-date filters, sort fields. Not a change/deletion cursor. |
| `GET /api/v1/books/:bookId/annotations` | Annotation response DTO; paginated when page supplied | User/book access must be exercised in the spike. Do not use the unpaginated branch for large libraries. |
| `GET /api/v1/annotations/export` | Existing Markdown, CSV, JSON export | Export is annotation-specific, not a combined library/status/session feed. |
| `POST /api/v1/books/metadata-export/preflight`, `/download` | Existing CSV metadata export | Requires `LibraryDownload`; not a verified replacement for status/progress ingestion. |
| `GET /api/v1/app-info` | Version and optional append-only feature names | Use for compatibility preflight; it does not report a Git commit or prove exact response compatibility. |

Sources: book controller/service/query pipe and card assembler [B3]; sessions controller/DTO [B4]; annotations controller/DTO/export service [B5]; app-info types [B6].

### Exact entities and fields

- **Book:** installation-local numeric `id`; title/subtitle, authors, language, publisher/publication dates, page count, ISBN-10/13, metadata provider IDs and optional Hardcover edition ID. Card `status` means file availability (`present`/`missing`), not reading status. [B3], [B7]
- **File:** numeric file ID, format, role, size; detail can expose `fileHash`, `contentVersion`, creation time and audio duration. Multiple files can belong to one book. Neither file ID nor hash is a universal work identifier. [B7]
- **User status:** keyed by user/book; `status`, `source` (`auto`/`manual`), nullable `startedAt`/`finishedAt`, `updatedAt`. Native states: `unread`, `want_to_read`, `reading`, `on_hold`, `rereading`, `read`, `skimmed`, `abandoned`. Preserve distinctions instead of treating every terminal state as completed. [B7], [B8]
- **Rating:** personal rating is distinct from community ratings; retain the source scale and nulls. [B7]
- **Progress:** keyed by user/file; `percentage` is 0–100. CFI, page, audio seconds, Kobo locator and KOReader XPointer are different representations. Cards expose primary-file progress, not every file's progress or an interchangeable locator. [B3], [B8]
- **Sessions:** row ID, nullable file ID, start/end, duration seconds, progress delta/end progress, source, attempt ID and session type (`read`/`tts`/`listen`). Client `sessionId` is unique per user in storage, but the session list uses row IDs. Repeated client checkpoints can extend an existing row's duration; there is no general session `updatedAt`/tombstone feed established here. [B4], [B8], [B9]
- **Annotations:** ID/book ID, text, note, color/style, origin, CFI/jump-file/page/PDF position, position status, chapter, highlighted/created/updated/starred timestamps. `highlightedAt` chooses source creation time when available. Annotations support trash/restore/purge; the presence of those operations does not establish a universal retained deletion stream. [B5]

### Time and history semantics

`reading_progress.updatedAt` is **not a universal last-read timestamp**. Its schema comment says KOReader deliberately freezes it as a last-local-write marker; `lastReadAt` is a separate ordering field. The book progress response inspected here omits `lastReadAt`. Status and metadata timestamps also do not cover every progress/rating change. Do not use any one of these as an all-entity incremental watermark or publish it as real-time reading presence. [B3], [B8]

Web session save caps client-reported active duration at the wall-clock start/end span. KOReader documentation describes sessions reconstructed from page-stat clusters, split at a 30-minute gap and recomputed when late data fills gaps. The repository also contains cumulative-counter session reconstruction. These differ from immutable, exact session events; source/type alone is insufficient to assert measured session boundaries. Sessions are a separate contract investigation, not part of the first slice. [B9], [B10], [B15]

### Authentication and permissions

Local login accepts username/password plus optional `clientKind: "native"` and device label. Native credentials include access/refresh tokens and expiries; Bearer JWT extraction is implemented. Defaults are 15-minute access and seven-day refresh lifetimes, configurable at deployment. Refresh rotates tokens and validates active account/session eligibility. Browser mode also uses HTTP-only cookies. [B11], [B16]

Prefer an ordinary account with only the necessary library access, with explicit owner-provisioned credentials. A normal account is **not a read-only API credential**: users can reset their own reading history and write progress/status. Enforce a client method/path allowlist, treating only the query POST as a permitted data-read POST. Auth login/refresh are separate session-management exceptions; prohibit reader mutations. Do not store or log auth exchanges. OIDC-only deployments may require a different machine-auth contract; unattended OIDC login was not validated. [B11], [B12]

### Incrementality, deletions and compatibility

Query pagination is offset-based; the sort builder adds book-ID ordering as a tiebreaker. It is not an atomic snapshot. Concurrent changes, permission changes and missing files can change membership. Bound scans and deduplicate by source identity; mark coverage complete only after every page and required detail read succeeds. Absence means unavailable in this scan, not permission to delete canonical history. No supported reading webhook or general monotonic change/deletion feed was found in the inspected server/docs search. This is a bounded-search result, not a promise that none exists elsewhere. [B3]

| Method | Assessment |
| --- | --- |
| Documented authenticated API | Preferred interface family, but current application API must be treated as version-pinned, not stable merely because Swagger exists. |
| Supported export | Useful annotation export and metadata CSV; not a verified complete status/progress/session snapshot. Revisit for offline evidence capture. |
| Read-only PostgreSQL | Reject as default: bypasses app scoping, couples to Drizzle schema, exposes unrelated users, and does not solve deletion/incrementality. If later necessary, require a dedicated read-only role, user/library-filtered views, explicit query contract and schema/version check; never share migrations. |
| Version-pinned HTTP adapter | Recommended first contract: validate version, allowlisted payload shapes, bounds, auth and replay semantics. Fail closed on incompatible source. |
| Upstream API/webhook contribution | Later option for scoped machine credentials, deterministic change cursors, retained tombstones and explicit measured-versus-derived session evidence. Not a prerequisite for testing snapshots. |

## Reader/deployment feasibility, not approval

Pinned README requires BookOrbit server v3.0+ and iOS 26+ for the native app. Apple's listing independently reports iOS/iPadOS 26+; app version 1.4.0 offers in-app purchases, with unlimited downloads and other features under Premium. Verify every intended device and whether offline limits are acceptable before purchasing or migrating. Android native is advertised as coming soon; KOReader is the proposed Android client, whose installation and behavior on the owner's actual devices remain untested. [B13], [A1]

BookOrbit's README explicitly disclaims writable NAS/network/nonstandard filesystems. Official installation uses Compose, persistent books/app/Postgres paths, PUID/PGID, independent secrets and a dedicated database. Database extensions include pgvector. The docs describe a pg16-to-pg18 transition: never reuse an old database volume across majors without a tested dump/restore. Do not choose an existing cluster volume/backend based on assumptions. Pin app and database images; test backup restoration, not only backup creation. No host, storage path, ingress or device inventory was chosen in this task. [B13], [D1]

## Licensing boundary

BookOrbit declares **AGPL-3.0-only**, with additional non-permissive terms for specified copyright holders. `NOTICE` and `ADDITIONAL_TERMS.md` require attribution/accessible legal notices for covered works, exact Corresponding Source when AGPL section 13 applies, modified-version/date disclosure and trademark restrictions. Preserve notices when copying covered material. This is a source summary, not a legal compatibility opinion. Prefer an independently written HTTP client without copied implementation; do not assume HTTP separation settles every legal question. Modifications/redistribution require a separate licensing review. [B14]

Tencent's skill files declare Apache-2.0. That licenses the files, **not unrestricted hosted API use or rights to publish book text**. No numeric request quota, token lifetime, scheduled-automation permission, or comprehensive API-service terms were established from the pinned skill docs or public key-setup page. Verify these before enabling unattended polling. [W1], [W2]

## Official WeRead gateway

The official Tencent repository documents `POST https://i.weread.qq.com/api/agent/gateway`, Bearer `WEREAD_API_KEY` (`wrk-…`), user identity bound to that key. Parameters are flat with `api_name` and `skill_version: "1.0.5"`; not nested in `params`. Nonzero `errcode` is an error. `upgrade_info` requires pausing and reviewing the new contract; do not auto-install arbitrary updates. `/_list` can report currently available APIs after owner-authorized credential provisioning. [W1]

| Gateway operation | Relevant documented fields / limitations |
| --- | --- |
| `/shelf/sync` | `books[]` identity/title/author/cover, `readUpdateTime`, `finishReading`, `secret`; separate `albums[]` and `mp` article-collection entry. Do not classify every shelf entry as an ebook. No changed-since guarantee documented. |
| `/book/info` | ISBN, title/author/translator, publisher/date, cover, word count and community rating. Not a personal rating source. |
| `/book/getprogress` | `book.progress` is integer 0–100: **1 means 1%, not 100%**; chapter/offset, `updateTime` documented as last reading time, cumulative seconds, start flag, optional finish time. Preserve missing fields and validate epoch units. |
| `/readdata/detail` | Weekly/monthly/annual/overall natural periods; `totalReadTime` seconds, optional day/month/year buckets. These are aggregates, not per-book exact sessions. Top-book list is truncated; do not derive complete history from it. Timezone boundary behavior still needs a live contract check. |
| `/user/notebooks` | `count` and `lastSort`, advancing from last book's `sort` until `hasMore=0`; not offset/limit pagination. |
| `/book/bookmarklist` | Personal highlights (`bookmarkId`, text, chapter/range/create time); **bookmark contents are filtered out**, though bookmark counts exist elsewhere. |
| `/review/list/mine` | Personal notes/reviews, IDs, text/optional quote/range, create time, star; `synckey`/`count` pagination. Keep separate from other users' public reviews. |

Source details: [W3], [W4], [W5], [W6]. No authenticated WeRead call was made. Retain WeRead alongside BookOrbit; never scrape protected text or transfer DRM-protected files. Source-public/private flags are evidence, not authorization to publish in Iroha.

## Reuse and actual Iroha gaps

Iroha already has works/items/titles/creators/external refs, current progress, state history and non-null-time consumption events. Reuse the fetch-to-raw-file-to-pure-parser flow (`SyncRunner`, `MediaImporter`), durable jobs, sync coverage/retries and source-owned identities. Snapshot states are not consumption events under [ADR-0005](../adr/0005-media-provider-time-semantics.md). Source/date precision and provider disagreement must remain explicit. [I1]

The inspected code does **not** make the full issue schema free:

- `observations.Media` has metadata/status/progress but no author/annotation/session payload. Creator tables exist, yet the current media import path does not populate book authors through this observation contract. First-slice author support needs a small tested extension or an explicitly documented limitation.
- Existing progress projection uses deterministic source priority (`manual/web/telegram`, then AniList, then Bangumi, then unknown sources). Decide BookOrbit/WeRead selection deliberately; do not accept lexical ordering as product policy.
- External refs are globally unique by provider/external ID. BookOrbit numeric IDs need instance/account scoping; never equate one installation's book 1 with another's book 1. The existing title/year matcher is not a safe edition/translation resolver for arbitrary books; first slice must not auto-merge those books through that fallback.
- No general reading-session duration or annotation/locator model, or per-book public sharing control, was established in the current media contract. Add only the capability required by an approved slice, with SQLx migrations where needed.
- Raw file bytes are purged after seven days by default (failed imports after 30), while rows/receipts persist. Unknown source-native fields cannot be replayed from a hash after purge. Specify a durable private evidence retention/archive policy before claiming source-independent replay; globally disabling purge requires a separate storage/operations decision.

Sources: core observations, imports and rawfiles at the pinned Iroha revision [I2].

### Public architecture correction to #142

The active public site is a static client shell consuming a **live, validated sanitized `/public/v1` projection**. The scheduled builder/export pipeline described in #142 is retired. Reuse `publicexport.BuildSnapshot`/validation and the public-only proxy boundary; never expose `/api/v1` or raw provider data. A strictly offline/static books snapshot would be a new architecture decision, not reuse of today's default. [I3]

Publishing remains a later slice: default private, explicit per-book opt-in, optional/coarsened last-read dates only when trustworthy, lawful covers, no source account IDs/locators/notes. Unpublishing requires testing snapshot invalidation and public browser/edge cache behavior; the current documented public max-age is one day, so promise no faster revocation without changing that contract.

## Spike/prototype results and remaining unknowns

Performed: pinned source inspection, route-to-service/type walkthrough, comparison with Iroha migrations/import/projection code, official App Store/key-page reads, and Iroha baseline `make check` (exit 0). This establishes candidate contracts, not deployed responses or reliable synchronization.

Not performed: executable importer prototype, BookOrbit server startup/OpenAPI capture, authenticated pagination/refresh, actual payload capture, backup restore or four-device acceptance. They require an approved isolated environment and credentials/devices. No invented fixture is presented as a real provider response. The first acceptance issue is an integration/reader spike, explicitly blocked on those owner choices.

The [plan](../plans/2026-10-12-book-tracking-integration.md) separates that runtime contract gate, a private vertical slice, public sharing and optional WeRead/history expansions. Stop after delivering preparation; do not turn these recommendations into rollout permission.

## Source references

[B1]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/AGENTS.md
[B2]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/swagger.ts
[B3]: https://github.com/bookorbit/bookorbit/tree/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/book
[B4]: https://github.com/bookorbit/bookorbit/tree/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/reading-session
[B5]: https://github.com/bookorbit/bookorbit/tree/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/annotation
[B6]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/packages/types/src/app-info.ts
[B7]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/packages/types/src/book.ts
[B8]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/db/schema/reader.ts
[B9]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/reading-session/reading-session.repository.ts
[B10]: https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/koreader.md
[B11]: https://github.com/bookorbit/bookorbit/tree/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/auth
[B12]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/reading-state/reading-state.controller.ts
[B13]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/README.md
[B14]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/ADDITIONAL_TERMS.md
[B15]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/modules/reading-session/reading-session.service.ts
[B16]: https://github.com/bookorbit/bookorbit/blob/3e993af16b8ee0a3bee5a1b3a602113745de4457/server/src/config/config.ts
[D1]: https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/installation.md
[A1]: https://apps.apple.com/us/app/bookorbit-the-official-app/id6811807346
[W1]: https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/SKILL.md
[W2]: https://weread.qq.com/r/weread-skills
[W3]: https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/shelf.md
[W4]: https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/book.md
[W5]: https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/readdata.md
[W6]: https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/notes.md
[I1]: https://github.com/azusachino/iroha/blob/bb1df41b2e8a5f93c004edc80828fdc15fefe9c9/apps/iroha-imports/sync.go
[I2]: https://github.com/azusachino/iroha/tree/bb1df41b2e8a5f93c004edc80828fdc15fefe9c9/apps
[I3]: https://github.com/azusachino/iroha/blob/bb1df41b2e8a5f93c004edc80828fdc15fefe9c9/docs/public-site-publishing.md
