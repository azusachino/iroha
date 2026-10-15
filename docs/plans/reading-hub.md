# Reading Hub: Phase A design and implementation proposal

Date: 2026-10-12. Status: proposed; complete package for the owner's final review, not runtime acceptance or production implementation.

Origin: [#142 Universal Reading Hub comment](https://github.com/azusachino/iroha/issues/142#issuecomment-6116906753). Architecture: [ADR 0011](../adr/0011-provider-neutral-reading-evidence.md). Daily-use guidance and primary-source links: [BookOrbit/source practices](../integrations/book-reading-hub-practices.md). Existing BookOrbit acceptance: [#143](https://github.com/azusachino/iroha/issues/143).

## 1. Scope and six-deliverable index

| Requested Phase A deliverable | Authoritative location |
| --- | --- |
| Architecture decision record | Proposed ADR 0011; no implied acceptance of earlier proposed ADRs |
| Verified provider capability matrix | Section 3: distinguishes documentation, source inspection, synthetic execution and unverified live permissions |
| Minimum observation model mapped to existing tables | Sections 4–5: identity/version/time contract, actual reuse and explicit gaps |
| Synchronization state machine | Section 6: inbound capture/import, reconciliation and durable outbound transitions, retries/conflicts/echoes |
| Minimum test-first vertical slice | Section 7: two saved-snapshot sources before scheduling, then explicitly controlled outbound acceptance |
| Risks, assumptions and external prerequisites | Section 8; provider/runtime/owner gates are not silently waived |

This continues the earlier research rather than replacing it. Ownership: `iroha-core` typed contracts; `iroha-providers` provider-specific formats/fetching; `iroha-imports` identity, evidence/persistence and selection; `iroha-runtime` jobs; server/worker enqueue/execute; shared frontend assets under `packages/`. No new service, database, scheduler, renderer or runtime code in Phase A.

The owner performs one **final** review after all six deliverables and required gates are ready. No agent reviewer or WIP approval loop. Proposed signatures/schema changes below are not implemented capabilities. Real accounts, scheduled polling, outgoing writes, device migration and public publishing are separate execution boundaries.

## 2. Evidence and current implementation inventory

Baseline: Iroha `822c885f70728a6f5c2dc9d5ad632d1f2904ff65` (research branch; runtime code matches `c6c9a8207f6e3e2650e801965aea38f8f8f0b6d2`). BookOrbit 3.3.0 source `e767358443419253eeed782518d6fc4f675d69fd`; documentation `5d0ed874b134f1dbf93e5d57398d1ca78a838830`. Hardcover documentation/schema `6ee717ec9a2c810d859a2a0b09a0a95354ac2a27`. Tencent WeChatReading `d98e66ee60a8c7bcd35da549d0643c4470439643`, skill contract 1.0.5. Amazon official documentation inspected 2026-10-12; no owner-account test.

| Existing seam | Actual behavior / relevance |
| --- | --- |
| [Provider contract](../../apps/iroha-core/provider/v1/provider.go), [registry](../../apps/iroha-providers/registry/registry.go) | `Descriptor`, `MediaImporter`, optional `MediaHistoryImporter`; current media capabilities are library/progress/rating/provider activity. Registered providers are Apple Health, AniList, Bangumi and GPX—not these new reading sources. |
| [Connector contract](../../apps/iroha-core/connector/v1/connector.go), [registry](../../apps/iroha-providers/connectors/registry.go) | `Fetch` yields native snapshot bytes and a cursor; parsers normalize separately. No outbound sink interface exists here. |
| [SyncRunner](../../apps/iroha-imports/sync.go), migrations [0017](../../apps/iroha-server/db/migrations/00017_media_sync_runs.up.sql), [0016](../../apps/iroha-server/db/migrations/00016_source_coverage.up.sql) | Stores pages, queues imports and advances fetch cursors; records partial/covered acquisition. A completed fetch run is **not** proof that every child import completed or that canonical state is a single atomic full-run snapshot. |
| [Media persistence](../../apps/iroha-imports/media_import.go), [resolution](../../apps/iroha-imports/media_resolution.go) | Reuses works/items/state history/progress. New unmatched items currently create a new work; `WorkExternalID` alone does not implement shared-work edition linking. Dated `persistMediaStateUpdate` currently hardcodes AniList, so it cannot be reused unchanged for Hardcover/WeRead activity. |
| [Source receipts/instances](../../apps/iroha-server/db/migrations/00013_source_instances_receipts.up.sql), [observation ownership](../../apps/iroha-server/db/migrations/00014_observation_ownership.up.sql) | Instance-scoped identities and receipt lineage exist. Media persistence does not currently populate the generic source-observation/version links like the health-domain persistence does. |
| [Media time semantics](../adr/0005-media-provider-time-semantics.md), [background-first reconciliation](../adr/0006-background-first-reconciliation.md) | Separate current state, dated state updates and true events. Ambiguity remains inspectable/separate without a routine human inbox. Existing precedence is manual/web/telegram > AniList > Bangumi > other; it is not yet a deliberate book-field policy. |
| [Job schema](../../apps/iroha-server/db/migrations/00001_current_schema.up.sql), `apps/iroha-runtime/jobs/` | Durable queued/running/completed/failed/canceled jobs, attempts and leases already exist. They are execution infrastructure, not a remote write acknowledgement ledger. |
| [Private OpenAPI](../contracts/openapi.yaml), [publishing](../public-site-publishing.md) | Existing raw-file/import APIs, private media reads, exact-event producer and matching-decision API. Public delivery is validated `/public/v1`; the old static builder diagram in #142 is not current architecture. |

`docs/data-model.md`, `docs/import-pipeline.md` and the reading portion of `docs/roadmap.md` supply context but contain older names/future-table descriptions. Current migrations/contracts/code and accepted ADRs decide what is actually reusable; do not implement a duplicate schema from those historical examples.

## 3. Provider capabilities and directionality

Legend: **D** = documented official contract; **S** = inspected implementation/schema; **T** = prior synthetic runtime result; **U** = unverified/not established. D/S is source verification, not an authenticated account PASS. No new live requests were made. A capability is enabled in Iroha only after its adapter, permissions and conformance tests pass—not merely because this matrix contains D/S.

| Capability | BookOrbit 3.3.0 | WeRead skill 1.0.5 | Hardcover pinned beta API | Kindle/KU initial path |
| --- | --- | --- | --- | --- |
| Catalogue/books read | S/T local-file library/query/detail | D shelf + book info, ebooks/albums distinct | D/S global books/editions, personal `user_books` | Explicit manual/lawful export facts; automated account access U |
| Status read | S/T personal book status | D `finishReading`, start/progress facts; not a full equivalent enum | D/S `user_books.status_id` / status relation | Manual reported state only |
| Progress read | S/T user/file percent and native position | D `/book/getprogress`, integer 0–100 | D/S read rows: pages/seconds/native `progress`; float-unit semantics require verification | No accepted mobile Whispersync progress contract |
| Ratings read | S personal rating, native scale retained | D book `newRating` is **community** rating, not proved personal rating | D/S personal `user_books.rating` | Manual rating only |
| Sessions/time read | S native sessions, client/reconstruction caveats | D cumulative/period aggregate seconds, **not** exact session intervals | S read attempts/dates/position, **not** measured duration | U; device power/activity events are not sessions |
| Annotations read | S native user annotations/export, locator-specific | D highlights and personal thoughts; bookmarks count differs from exported content | U for a reader-highlight/locator equivalence contract | Only an explicit lawful supplied export contract, otherwise U |
| Status/progress write | T matched local-book personal writes; replay/stale-write counterexamples | U; no supported writeback established | S mutations and BookOrbit implementation; owner's scopes/idempotency U | U; no automated writeback planned |
| Ratings write | S personal write route | U | S selected BookOrbit-to-Hardcover rating writes | No automated route |
| Metadata-only creation | T ordinary tested create path returned 404; not established | Existing platform metadata can be read; no new write assumed | D/S catalogue + personal list/read records, no EPUB requirement | Iroha must accept no-file evidence; this is not KU file export |
| Stable all-entity change feed | U; bounded snapshots initially | U; endpoint-local paging is not a global change log | U; read rows have no schema `updated_at`, library timestamp alone is insufficient | U |

### Exact source contracts that shape adapters

**BookOrbit:** authenticated `/api/v1` application routes; optional Swagger is not a stability guarantee. See [the existing endpoint inventory](../integrations/bookorbit-research.md). Query pagination/card collapsing and personal-account scope must be explicit. A file-progress zero fallback is not a recorded reading action. `updatedAt` is not universal consumption time or an all-entity cursor; KOReader ordering has special semantics. Current reads preserve the server's reported state, not proof that its upstream device conflict was resolved correctly. Tested rejected provenance fields, timestamp-changing replay and stale lower-percentage overwrite remain [PR #148](https://github.com/azusachino/iroha/pull/148) counterexamples.

**WeRead:** [gateway contract](https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/SKILL.md) is `POST https://i.weread.qq.com/api/agent/gateway`, Bearer account-bound API key, top-level `api_name`, `skill_version: "1.0.5"` and flat parameters. Do not actually provision/read a key during Phase A. Nonzero `errcode` fails capture; `upgrade_info` pauses it for compatibility attention rather than continuing with an ignored upgrade.

- [Shelf](https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/shelf.md): `/shelf/sync`; `books`, `albums` and the `mp` directory entry are distinct, not interchangeable completed-book counts. Album `finishStatus` describes content publication/completion, not the user's listening completion.
- [Book/progress](https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/book.md): `/book/info` includes ISBN/publisher/translator; `/book/getprogress` reports `book.progress` 0–100 (`1` means 1%), chapter/offset, last-reading timestamp and cumulative seconds. Retain `finishTime`/start flags and contradictions; don't declare every shelf book “reading.”
- [Statistics](https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/readdata.md): `/readdata/detail` has fixed natural weekly/monthly/annual/overall scopes. `totalReadTime` is seconds, `readDays` uses a validity rule, `readLongest` is a filtered top-ten list. Do not derive a complete per-book ledger from rankings or add overlapping daily/monthly/yearly totals together.
- [Notes](https://github.com/Tencent/WeChatReading/blob/d98e66ee60a8c7bcd35da549d0643c4470439643/skills/notes.md): notebooks paginate with `lastSort`, not offset. `/book/bookmarklist` exports highlights, not bookmark content; personal reviews are distinct from public reviews. Source-native ranges do not become EPUB CFIs.

Hosted-service eligibility, terms, quota, timestamp/timezone edge cases, deletion and authenticated responses remain external prerequisites. A documented gateway is not a production connector PASS.

**Hardcover:** [getting started](https://github.com/hardcoverapp/hardcover-docs/blob/6ee717ec9a2c810d859a2a0b09a0a95354ac2a27/src/content/docs/api/Getting-Started.mdx) documents expiring scoped PATs, beta compatibility, a 30-second query timeout and server-side token handling. Pin-time Free limits: 5,000 top-level query costs/day, burst 10, 60/minute; Supporter: 50,000/day, burst 15, 60/minute. Read `RateLimit`/`RateLimit-Policy` and `Retry-After`; oversized top-level batches can be 403 `request_exceeds_capacity`, not an ordinary retryable 429. Actual token/account headers override assumptions about these published limits. BookOrbit's 1100-ms pacing is not the quota definition.

The [schema](https://github.com/hardcoverapp/hardcover-docs/blob/6ee717ec9a2c810d859a2a0b09a0a95354ac2a27/schema.graphql) has personal `user_books`, `user_book_reads`, edition ISBN/ASIN/language metadata, and `insert_user_book_read` / `update_user_book_read`. Read rows have date-only `started_at`/`finished_at`, finish precision and separate native progress fields. Preserve nulls, explicit read IDs and every attempt; don't choose the last array element without a tested ordering/current-attempt rule. Date precision codes/status enum/percentage-float interpretation need fixture/API verification. The unfinished progress-write guide is not a tested recipe. HTTP 200 with GraphQL errors is not complete capture.

**Kindle/KU:** [Amazon portability scopes](https://developer.amazon.com/docs/amazon-data-portability/available-scopes.html) exist, but eligibility and useful reading-history semantics remain U for this account. Device interaction/power records with ASINs are not completion or reading time. [KU membership terms](https://digprjsurvey.amazon.com/csad/help/node/201556940) do not grant permanent exportable EPUB ownership. Start with declared manual status/date facts or a separately approved supplied export; do not imply automatic Kindle progress integration.

### Capability registration proposal

Map initial book/status/progress/rating inputs to existing `media.library`, `media.progress`, `media.rating` descriptors and typed `MediaImporter` output. Do not clone the handoff's entire `books.*` interface family alongside the existing registry. Add an optional narrow sink contract only for the first accepted outbound Hardcover operation. Sessions/annotations require their own demonstrated typed contracts; do not declare capabilities whose methods are absent or whose permissions are unknown. Keep documentation/source support separate from configured adapter support and live readiness.

## 4. Minimum canonical observation contract

A **capture** is acquired source evidence; a **record identity** is the stable native entity; a **record version** is its facts in a capture; a **projection** is selected current state. These are different things. This proposal describes their semantics, not a new universal JSON metadata column or a mandatory standalone `ReadingObservation` table.

| Required fact | Meaning / determinism |
| --- | --- |
| Provider + source instance/account | Stable configured provider/realm/account identity, independent of hostname renames, token rotation and display names; capture/receipt owns it |
| Entity type + native ID | Distinguish global catalogue work/edition IDs from personal shelf/read/file IDs; native IDs are never assumed globally unique across BookOrbit installations/accounts |
| Work/edition/file references | Verified mapping evidence and language/edition metadata; file optional; unknown edition remains explicitly unresolved, not invented |
| Native status + normalized status | Preserve native distinctions; absence/null is not unread, abandonment is not completion, publication completion is not personal completion |
| Position + unit + optional total/percent | Null distinct from recorded zero; source-native file locator remains separate; no maximum-progress rule |
| Started/completed date + precision | Partial calendar date or exact source instant as actually reported; never fabricate midnight UTC for a date-only reading fact |
| Source revision/time + meaning | Optional native revision/update time with endpoint-specific semantics; missing revision stays missing. Server time is not automatically a reading event. |
| Capture observed time + receipt | Acquisition metadata, not parser-generated activity; stable when replaying the same capture, never regenerated by the pure parser |
| Evidence pointer + interpretation version | Native immutable capture, digest, receipt and parser version; provenance and reprocessing remain queryable |
| Quality/origin | Distinguish provider-reported snapshot, manual assertion, derived conversion, echo and measured/reconstructed interval; only proved facts get stronger labels |

Record identity: existing `(source_instance_id, source_kind, source_key)`; the source key includes entity type/native ID where needed. Global Hardcover catalogue `book/123` and `edition/123` must not collide with each other or account-local `user_book/123`. BookOrbit external refs need instance/entity-qualified IDs; retain the native ID separately in evidence. This is a source-contract choice to freeze in fixtures, not permission to rewrite old provider keys.

Version identity: receipt/import-interpretation lineage plus deterministic native-content fingerprint. The saved-snapshot envelope includes producer-assigned acquisition identity/time and source scope alongside the unchanged native response. Replaying that envelope keeps its acquisition metadata fixed; a later genuine capture with the same native value is a distinct acquisition, not an old file re-upload. This prevents raw-byte/same-version skipping from erasing A → B → A. Source record/content keys exclude acquisition metadata; record-version lineage includes it. Parser outputs exclude newly generated ingestion-time UUIDs, request IDs and “now.” Existing `tb_source_observations` is upserted latest-record state; it is **not** an immutable append-only native-version log. Keep prior receipts and native bytes/version evidence rather than overwrite the sole historical payload.

Replay rules:

- Same capture and parser version: no duplicate identities/state rows/events.
- New interpretation version: preserve other-source facts and explicit matching/selection decisions.
- New capture A → B → A: preserve the genuine return to A. A global payload hash alone cannot deduplicate every occurrence of A; capture-scoped version lineage matters.
- Known source-event ID: use provider/account/entity scope and revision semantics; edited attempts remain versions of the same attempt, not new sessions.
- Missing/partial page or omitted field: never interpret as deletion/zero/unread.

Percent conversion is allowed only with a verified unit and total for the **same edition/read/file**. Preserve original pages/seconds and conversion basis. Zero/missing total, a default work-level page count, a wrong-language edition or ambiguous raw float disables conversion. Do not clamp invalid native values into plausible progress; retain evidence and a bounded diagnostic. Progress seconds indicate an audio position, not active listening duration.

## 5. Table and contract mapping: reuse versus gaps

| Concept | Existing home | Gap / smallest proposed action |
| --- | --- | --- |
| Work / edition | `tb_media_works`, `tb_media_items` with `work_id`, language/pages; typed `Media` work fields | `ensureMediaItem` presently makes a new work per new unmatched item. Add tested work-only identity/linking for verified translations; don't attach two editions to one item to fake success. |
| File | Native BookOrbit capture + file ID/hash/format | No demonstrated canonical file/locator contract. Keep typed source references/evidence initially; an optional file table is justified only by a reader/file journey, never required for a book. |
| Authors/translators | `tb_media_creators`, `tb_media_creator_roles` | `Media` currently lacks typed creator roles. Add the narrow DTO/persistence path needed by metadata fixtures; don't stuff authors into free-form notes. |
| Global/provider identities | `tb_media_external_refs`; explicit matching decisions | Unique `(provider, external_id)` and current item resolution need scoped IDs and work-versus-item references. ISBN/title matching must not collapse translations/files. |
| Account/source lineage | `tb_source_instances`, `tb_source_receipts`, `tb_source_observations`, `tb_source_observation_receipts`, `tb_import_snapshots` | Reuse lifecycle, but wire media account-scoped observation/receipt persistence explicitly; current media path does not supply the complete proposed chain. Retained native bytes are needed for unmodeled fields. |
| Current state | `tb_media_progress`, `MediaProgress` | One row per item and `source_kind` selection are not configurable per-field/per-account authority. Implement the smallest explicit selection policy for the accepted book sources; leave richer controls for an actual need. |
| Dated changes | `tb_media_state_history`, partial-date columns, snapshot-scoped idempotency migration 00009 | Reuse source/observed-time basis. Generalize the AniList-specific dated-update path only for a proven non-AniList contract; don't invent exact updates from date-only attempts. |
| True events/sessions | `tb_media_consumption_events` / exact-event producer | No general measured/reconstructed reading-session interval contract yet. Retain native session evidence first; an accepted interval contract is required before duration analytics. |
| Read attempts / annotations | Native capture; typed extension when accepted | Single start/finish pair or repeat count cannot losslessly represent multiple edition-specific attempts. Preserve all read-row IDs/payloads; do not promise queryable full attempts/annotation UX in the first state slice. |
| Outbound intent/receipt | Existing `tb_jobs` for execution; no dedicated remote-operation ledger | Add a narrow SQLx-backed operation/receipt model only in the controlled-write slice. Queue status is not a remote receipt; inbound `tb_source_receipts` must not be repurposed as outbound acknowledgements. |
| Public visibility | Existing public projection/publishing boundary | Per-book opt-in and revocation remain separately scoped #146; source public/privacy settings never automatically opt a book in. |

Matching policy: exact explicit mapping or verified edition identifier first; reliable cross-reference next; normalized title/author/language produces candidates only. Existing general exact title/year auto-attachment must be bypassed for the new reading adapters. Keep ambiguous entries separate and inspectable, following ADR 0006, not a required human resolution inbox. A work-only match may link distinct translation items after verified evidence without merging their state.

Proposed initial selection: retain the existing explicit manual/web/telegram authority. For newly approved reading sources, define a deterministic order for compatible **same-edition** fields (candidate: BookOrbit > WeRead > Hardcover), with source-specific timestamp semantics and echo exclusions. This order is proposed, not silently inserted into the current “other” fallback. Ratings/date authority may differ; exact locators stay provider-owned and public visibility stays Iroha-owned. No cross-source percentage comparison for different translations. If a source lacks trustworthy change time, label its value as reported current state from the latest complete capture; do not call it the latest reading action or claim stale-device arbitration.

## 6. Synchronization state machines

The names below are semantic design states, **not** extra values silently added to the existing job-status enum. Inbound execution uses existing jobs/sync state; future outbound semantic state belongs to its typed durable operation record.

### 6.1 Inbound acquisition and import

| State → next | Guard/action | Failure / recovery |
| --- | --- | --- |
| Disabled → Ready | Owner approves provider/account, permissions, source/version contract and bounded acquisition scope | Otherwise no network job |
| Ready → Fetching | Claim existing job; read saved cursor; apply request/page/time/byte budgets | Missing credentials → source attention, no destructive changes |
| Fetching → Captured page | Validate response identity/version and transport/application errors; persist unchanged native evidence and receipt | 401/403 permission → pause; schema drift/WeRead upgrade → compatibility attention; malformed required data → failed capture |
| Captured page → Fetching / Fetch-complete | Persist resume position only after durable storage/child import enqueue; terminate only at proven exhaustion | 429 honors provider delay; network retry is bounded; cursor repeats/empty-with-more/budget exhaustion mark partial rather than loop or claim complete |
| Captured evidence → Importing | Separate existing import job selects versioned pure parser and validates typed facts | Same-version replay may skip; parser failure leaves capture and earlier projections intact |
| Importing → Applied | Transactionally persist identities, record versions/history and deterministic projection | Crash rolls back; lease recovery retries idempotently; no partially committed import batch |
| Fetch-complete + all required child imports Applied → Reconciled scope | Prove account/scope/version/coverage; perform only accepted non-destructive presence reconciliation | Failed/queued child import means fetch-complete only, not canonical-complete |

A page may carry positive evidence before the full fetch finishes, as existing SyncRunner allows. It cannot establish destructive absence for the whole library. First saved-snapshot slices package one bounded validated scope per input to simplify transaction/replay behavior. A scheduled connector must prove child-import completion or add the small completion barrier before any full-scope reconciliation; do not infer it from `tb_media_sync_runs.status=completed`.

Even complete offset exhaustion is not an atomic provider snapshot: concurrent remote edits may move rows between pages. Unless provider/version semantics prove a consistent scope, absence remains non-destructive. Remote removal affects that source's membership/current availability only; it never erases canonical history, another source, a match decision or public policy. Explicit tombstones are accepted only with a tested source contract.

### 6.2 Identity and selected-state reconciliation

| Input | Result |
| --- | --- |
| Explicit same-edition mapping | Attach its source fact to that item; retain every provider record/version |
| Verified same work, different edition/language | Link work; keep separate items, positions and read attempts |
| Weak/ambiguous title/author match | Separate source-owned records + inspectable candidate/conflict metadata; ordinary import continues |
| Trusted older source revision/time | Keep evidence/history; do not roll current selected state backward merely because it arrived later |
| New trustworthy rewind/reopen/reread | Apply policy to that edition/attempt; backward progress is valid, not overridden by historical maximum |
| Equal-time or unknown-time conflicting sources | Deterministic authority selects current view; preserve disagreement/quality. Don't invent a temporal winner. |
| Equivalent BookOrbit → Hardcover echo | Preserve Hardcover observation but mark/exclude it from independent activity aggregation or outbound fanout where origin is proved |
| Origin cannot be proved | Treat origin as unknown; do not invent attribution. Avoid the loop by prohibiting imported destination observations from automatically creating outbound intents. |

Parser-version replay never triggers outbound work. Reconciliation may provide machine-readable source attention, but it does not create a recurring mandatory human inbox. Identity correction and outbound conflict approval are explicit exceptional operations, not prerequisites for every read-only import.

### 6.3 Controlled outbound operation

Proposed durable fields: operation ID; target provider/account/entity/field set; source instance/observation or capture version; canonical item/edition/attempt; desired value fingerprint; local idempotency key; route/authority generation; stable remote record/read ID if known; semantic status; attempt/backoff/lease; acknowledgement/reconciliation evidence; sanitized error class. Store no token/native credential in jobs, receipts or logs.

The local uniqueness key includes the source version/attempt and route generation, not merely book/desired percent: identical progress in a later reread is not the same operation. Scope target ownership to account + target user-book + fields, since distinct translations may share that user-book record. Serialize conflicting field sets and recheck ownership/latest selected version on claim. A future accepted write slice persists intent with its triggering decision in the same transaction, then schedules execution without a lost-enqueue window.

| State → next | Required action |
| --- | --- |
| Disabled / Not eligible → no intent | Default; no configured route, unknown permissions, private annotation, missing mapping, incompatible edition or delegated BookOrbit-owned field cannot write |
| Eligible → Pending | Explicit route/field allowlist, exclusive ownership, verified mapping/unit/privacy and source-origin policy; atomically store unique intent |
| Pending → In-flight | Existing job lease claims it; revalidate route generation, authority, selected version and target ownership; serialize overlapping target fields |
| Pending → Superseded / Canceled | New selected version or disabled/reassigned route; keep audit history, do not send stale queued values |
| In-flight → Acknowledged | Remote success body and identity/value are verified; persist remote IDs/receipt; only then mark execution complete |
| In-flight → Retry-wait | Failure is proved not applied and retryable; respect provider headers, capped backoff/jitter and finite attempts |
| In-flight → Needs-auth / Failed | Auth/permission/permanent validation failure; source attention; no blind retry loop |
| In-flight → Uncertain | Timeout, lost response, crash after dispatch, or failed local acknowledgement storage; remote effect may already exist |
| Uncertain → Reconciling | Read target by stored identity/operation context, never blindly insert another attempt |
| Reconciling → Acknowledged | Intended remote effect is uniquely confirmed; persist recovered receipt |
| Reconciling → Pending | Authoritatively prove not applied, route still valid and retry safe |
| Reconciling → Blocked-conflict | Conflicting edit, ambiguous matching read row or unverifiable outcome; preserve evidence and stop automatic writes |
| Retry-wait → Pending / Failed | Due time and remaining budget / exhausted retries |

A local idempotency key is not automatically honored by Hardcover. Read-before-write cannot make a non-conditional remote mutation atomic: without verified CAS/version/idempotency guarantees, single local ownership reduces races but cannot promise exactly-once effects. Ambiguous read-attempt insertion stays blocked rather than duplicated. A status setter may be value-idempotent while its associated attempt creation is not.

Route disable/revocation prevents new claims/retries; it cannot recall an already dispatched request. Record an in-flight result honestly even after disable, and require an explicit corrective operation if needed. Never replay receipts as source-native provider events.

Loop prevention combines exclusive target ownership, explicit source-origin routes, durable source-version/desired-value receipts, and no automatic outbound generation from destination imports, echoes or parser replay. Do not use a percentage hash alone to suppress every later real reread. For BookOrbit-managed Hardcover fields, mode is delegated: Iroha creates no writer job. Status/rating/date/edition ownership must be transferred deliberately before an Iroha route may take over.

### 6.4 Required state-machine examples

| Test journey | Expected invariant |
| --- | --- |
| Same snapshot/interpretation twice; parser version bump | No duplicate identities/history/events; other-source decisions preserved; zero outgoing writes |
| Capture A → B → A; 70% then genuine newer 10% | Real reversal retained; no `MAX(progress)` behavior |
| Older 10% delivered after trusted newer 70% | Both evidence versions survive; selected state remains newer |
| Chinese WeRead + English EPUB + Kindle metadata-only edition | One verified work, distinct items/positions; no fabricated EPUB/locator or percentage equivalence |
| GraphQL errors at HTTP 200; page outage; failed child import | No complete coverage claim or destructive absence reconciliation |
| 429 / oversized Hardcover batch / expired token | Provider-aware delay / reduce invalid batch / auth attention; no indefinite retry |
| Remote write applied, response lost, worker restarted | Uncertain → reconciliation; no blind second read-attempt insertion |
| External edit while queued; route disabled in flight | Recheck/supersede stale intent; remote result still recorded, no implied undo |
| BookOrbit writes Hardcover, Iroha later imports it | No second independent session or automatic return write |
| Unknown origin, missing timestamp, repeated identical reread progress | Uncertainty labelled; route policy prevents echo fanout without suppressing valid later attempts |

These are implementation acceptance scenarios, not claims that Phase A executed them.

## 7. Minimum vertical slice and staged implementation

### B1: private two-source saved-snapshot proof

Goal: BookOrbit and Hardcover fixture-backed snapshots → existing raw-file/import boundary → deterministic media facts → private media reads → replay. Include a no-file Hardcover/Kindle title and one explicitly verified shared work with distinct translations. No network scheduler, public UI or outgoing mutation is needed.

Order small test-first patches:

1. **Freeze capture contracts and retention.** Define version/source-instance/account/scope/completeness metadata plus native records; fixtures preserve nulls, units, dates, editions and read IDs. Establish durable byte/version retention for replay without turning global purge off silently. Synthetic fixtures are labelled synthetic, not disguised live captures.
2. **Pure provider normalization.** Add narrowly registered BookOrbit and Hardcover `MediaImporter` implementations/constants. Test metadata-only book, zero versus null, native enums, backwards progress, invalid totals, date precision, wrong-language edition and partial/error inputs. Preserve unsupported attempts/annotations in retained native evidence, not false events.
3. **Identity/persistence seam.** Test account-qualified native identities, author roles, book-specific conservative matching, work-only linkage, media receipt/version lineage and deterministic reading-source selection. Add paired SQLx migrations only for a demonstrated gap; don't create parallel work/edition tables.
4. **Private upload/read/replay proof.** Existing raw-file/import APIs and media reads; same-version replay, changed-version reinterpretation and A → B → A; one provider outage/partial capture; no other-source loss. Prove distinct accounts/instances do not contaminate each other's personal source records, even if catalogue metadata links to one item.

Exit: both saved-source adapters imported; verified shared work without edition collapse; metadata-only book visible; source evidence/selection inspectable; replay is idempotent and no synthetic sessions/outgoing writes occur. Full native read-attempt/annotation querying is not falsely claimed by retaining its raw payload. Reader-device acceptance remains #143.

This expands the reading-hub roadmap; it does **not** silently amend #144's BookOrbit-specific acceptance. A new narrower implementation issue must identify the Hardcover/account/identity/retention additions and owner-approved contract before code begins.

### B2: WeRead and bounded read-only acquisition

Validate approved official gateway auth/terms/quotas and capture fixtures independently; add WeRead as a third source. Reuse `Connector.Fetch`, snapshot store, `SyncRunner` and jobs. Conformance tests cover paging, resume, cancellation/budgets, upgrade pause, application errors, rate limits, and acquisition-versus-import completion. Test aggregate scope overlap and missing time separately from book state. Do not require retiring WeRead.

### C: one controlled Hardcover outbound route

After owner-approved mappings, permissions and policy, implement a single allowlisted **status** route for non-BookOrbit-delegated WeRead/manual-source target fields. Start with status-only; page-progress/ratings/dates/read-attempt creation need their own verified mutation semantics. Add a typed sink and minimal durable intent/receipt persistence, serviced by existing jobs. Test uncertain outcomes, crash recovery, concurrent claims, external edits, echo/replay exclusion and route disable before any approved live write.

No enabled route is shipped by default. A status mutation that implicitly creates/changes read attempts must meet attempt-idempotency acceptance or remain blocked. This slice—not Phase A—earns the handoff's eventual controlled-outbound criterion.

### D and E: public publishing and extensibility

Public books/visibility/revocation use #146 and validated `/public/v1`, with private default and cache-aware unpublishing. Reading duration/calendar analytics require supported intervals or clearly labelled aggregate scopes. Shared Grapher presentation lives in `packages/`. Only after multiple concrete adapters/one proven sink justify it should a reusable provider conformance suite, richer per-field controls or package extraction be considered. No speculative framework-first phase.

Gates for implementation: focused pure/helper tests during iteration, `make check`, disposable `make test-integration` for persistence and actual private upload/read/replay acceptance, then `make validate`. Source-aware smoke tests are separately approved and credential-safe. No production `db-reset`. Phase A documentation gates are `make fmt-docs-check`, `make validate`, and diff hygiene; green formatting/builds do not establish authenticated provider behavior.

## 8. Risks, unresolved assumptions and prerequisites

| Risk / unresolved fact | Required boundary before claiming acceptance |
| --- | --- |
| Loss of unmodeled evidence after byte purge | Approve scoped durable capture/version retention and test restoration/reinterpretation after the ordinary purge window; hashes alone cannot replay |
| Existing media path lacks proposed lineage/work-link/authority semantics | Treat them as concrete B1 changes, not existing capabilities; test both provider orders, source/account collisions and parser replay |
| Beta Hardcover schema, scopes, ordering and mutation side effects | Pin API/schema; separately approved account/token; sanitized complete/partial fixtures; verify enum/precision/unit/ownership/rate headers and retry safety |
| WeRead gateway service and evolving skill contract | Actual approved auth/terms/quota/timezone/upgrade/deletion acceptance; native notes/cumulative totals never become guessed sessions |
| Amazon portability region/device/data eligibility | Separate owner-approved capability spike if desired; otherwise truthful manual/export path; no unsupported scraping or KU file export |
| BookOrbit client/device compatibility, locator sync and restore | Owning #143 / harus-k3s #109 acceptance with exact clients/files/pins and actual devices; server availability alone is insufficient |
| Wrong-language Hardcover fallback or work-level page total | Verified edition/read mapping before normalized progress export; skip ambiguous conversion, do not guess |
| Shared target user-book across several translations/writers | Exclusive account/record/field ownership and intentional transfer; target edition alone cannot prevent collision |
| Snapshot paging races / completion before child imports | No destructive absence inference; prove run/child completion semantics and scope consistency before reconciliation |
| Lost remote acknowledgement / no CAS | Durable uncertainty and read reconciliation; no exactly-once claim or blind duplicate attempt creation |
| Privacy, source terms, lawful covers/excerpts | Source-private flags do not opt into Iroha publishing; explicit per-book/annotation opt-in and lawful cover provenance; no tokens/paths/locators in public projection |

Phase A final-review decisions: accept/revise ADR 0011; choose scoped native-evidence retention; approve initial reading-field authority and book work/edition identity rules; select B1 execution scope. These are presented together for the owner's final review, not requested as WIP decisions. Pending unknowns remain explicit gates: no scope or approval is inferred from silence or a published research PR.
