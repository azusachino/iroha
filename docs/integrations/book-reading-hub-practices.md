# BookOrbit usage and independent book-source integration

Date: 2026-10-12. Parent: [#142](https://github.com/azusachino/iroha/issues/142).

The [complete Phase A package](../plans/reading-hub.md) and proposed [ADR 0011](../adr/0011-provider-neutral-reading-evidence.md) own the new issue comment's architecture, observation mapping, synchronization state machine and implementation roadmap. This note remains the usage/primary-source review, not a substitute for those six deliverables.

Status: research and proposed sequencing, not an accepted architecture contract or authorization to implement. This reviews the reopened issue's ChatGPT handoff, including Hardcover and Kindle Unlimited. No authenticated Hardcover/Amazon requests, real account access, library imports or deployment changes were made for this research.

## Evidence boundary

| Source | Pin / evidence |
| --- | --- |
| Iroha | `c6c9a8207f6e3e2650e801965aea38f8f8f0b6d2`, current `main`; media contracts and import architecture |
| BookOrbit | 3.3.0 source `e767358443419253eeed782518d6fc4f675d69fd`; Hardcover matcher, sync policy/service/client and event listener |
| BookOrbit docs | `5d0ed874b134f1dbf93e5d57398d1ca78a838830`; workflow, library structure, KOReader and Hardcover guides |
| Hardcover docs/schema | `6ee717ec9a2c810d859a2a0b09a0a95354ac2a27`; API introduction, library/progress guides, GraphQL schema |
| Amazon | Official Data Portability scopes and Kindle Unlimited terms fetched 2026-10-12; availability for the owner's region/account remains unverified |

The earlier [BookOrbit research](bookorbit-research.md) describes source inspection, not live reader acceptance. The [companion feasibility report in PR #148](https://github.com/azusachino/iroha/pull/148) separately tests synthetic ordinary BookOrbit 3.3.0 writes: they do not preserve lossless WeRead provenance/history. Live deployment work belongs to [harus-k3s #109](https://github.com/azusachino/harus-k3s/issues/109); deployment availability does not establish restore or physical-device acceptance.

## Recommended daily workflow

These are recommendations for this setup, not universal upstream requirements.

1. **Keep two reading lanes.** Use BookOrbit for lawfully obtained, importable files; use Kindle's own apps/devices for Kindle Unlimited. A KU catalogue entry can exist in Hardcover/Iroha without a local file. Keep WeRead usable while evaluating the new reader.
2. **Use BookOrbit as the server-centered reader hub.** Clients synchronize through it, not directly with each other. Start with one legal EPUB and the actual two devices; test offline reopening, deliberate rewind and conflicting updates before migrating a library. The author's [workflow](https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/my-workflow.md) is an opinionated example, not a cross-device guarantee.
3. **Choose scan mode before arranging files.** For a new manageable library, prefer Folder as Book with one folder per edition/translation and sidecars beside its files. Do not put unrelated books—or Chinese and English translations—into one folder merely because they share a work. Multiple formats can be grouped, but their reading positions are not automatically equivalent. See [library structure](https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/library-file-structure.md).
4. **Keep the reading file stable.** Avoid replacing/converting an EPUB mid-read; preserve the original file and sidecars. Use the BookOrbit [KOReader plugin](https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/koreader-plugin.md) and its user-scoped credentials where supported. Don't introduce overlapping progress-sync writers during initial testing. Native Kindle/Whispersync is not this plugin's synchronization path.
5. **Separate administration from personal reading.** An ordinary reader account is preferable for daily use; grant only needed library/plugin/Hardcover permissions. Personal status/progress is user-scoped. An unrelated service user does not automatically see the human reader's history: an Iroha reader connector must use a documented session for the correct account, not silently substitute an administrator or a second user.
6. **Opt books into Hardcover deliberately.** Begin with selected-only sync and status/rating triggers; leave noisy progress pushing off until matching is verified. Check the work **and edition/language**, select private visibility unless deliberately publishing, and review each initial result. Imported Hardcover status can fill blank BookOrbit entries, so disable that pull while testing one-way behavior. See [Hardcover integration](https://github.com/bookorbit/bookorbit-site/blob/5d0ed874b134f1dbf93e5d57398d1ca78a838830/src/content/docs/hardcover.md).
7. **Complete restore acceptance before depending on it.** Back up book files, app state and the dedicated database together; demonstrate restoration separately. Current local-path PVC/namespace deletion is destructive. This recommendation does not claim the live deployment's remaining backup/device checks have passed.

### Why edition verification is necessary

BookOrbit 3.3.0's [matcher](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/hardcover/hardcover-book-match.service.ts) prioritizes a saved Hardcover edition, ISBN match, then an ebook edition with the closest page count, or a default/other available edition. Its fallback does not filter by translation language. Therefore title/work matching alone is insufficient for bilingual reading or page-progress export. If the correct edition cannot be verified, exclude that book from automatic Hardcover sync rather than accepting a plausible default.

The [sync implementation](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/hardcover/hardcover-sync.service.ts) and policy are user/book-specific. Its native mapping also does not preserve every BookOrbit status distinction. Keep BookOrbit's native status and source evidence in Iroha rather than interpreting a Hardcover round trip as a lossless copy.

## What the new handoff gets right

The important change is to stop requiring every source to pass through BookOrbit:

- BookOrbit remains the file/reader synchronization owner.
- Iroha accepts independent source evidence, resolves identity conservatively, and controls private history/public projections.
- Hardcover can be both an external catalogue and an optional reading-list source/destination; it is not proof of device-level reading.
- Kindle Unlimited is a subscription reading lane, not an EPUB acquisition mechanism.
- WeRead may remain a source even if no companion can faithfully import its history into BookOrbit.

This supersedes the mandatory **WeRead → companion → BookOrbit → Iroha** topology, not the earlier synthetic counterexamples. A limited mirror may still be useful, but it must explicitly identify lost fields and ownership; it is not canonical evidence storage.

The handoff's “then implement” instruction should not turn this research request into implementation approval. Nor should its proposed types create a parallel book database, scheduler or generic synchronization framework before a concrete source contract needs them.

## Capability matrix

“Supported” below means documented/schema/source evidence unless explicitly identified as a synthetic test. It is not an authenticated acceptance PASS.

| Source | Useful inbound evidence for Iroha | Outbound role | Important limit |
| --- | --- | --- | --- |
| BookOrbit 3.3.0 | Local book/file identity, personal status, file progress; sessions/annotations through separate contracts | Already writes selected fields to Hardcover; reader clients write native state | Ordinary writes failed lossless WeRead-history/provenance tests; locators belong to specific files |
| WeRead | Shelf/status/progress, annotations and aggregate reading data described by the inspected skill contract | No new write capability established here | Real authorization, terms, quotas and exact response/time semantics remain unverified; aggregate time is not fabricated sessions |
| Hardcover | Catalogue book/edition/author metadata; `user_books` status/rating/privacy; `user_book_reads` attempts/dates/native progress | Schema exposes reading-history mutations; BookOrbit already uses them | API is beta; schema existence does not establish the owner's token permissions, mutation idempotency or lossless provider equivalence |
| Kindle Unlimited | Initially explicit manual observations or an owner-supplied lawful export, with ASIN/edition mapping where verified | No reader-position write path established | Subscription access expires; Amazon portability eligibility and actual reading-history suitability are unverified |

### Hardcover API: useful, but not a universal reading ledger

The official [API introduction](https://github.com/hardcoverapp/hardcover-docs/blob/6ee717ec9a2c810d859a2a0b09a0a95354ac2a27/src/content/docs/api/Getting-Started.mdx) identifies a beta GraphQL API at `https://api.hardcover.app/v1/graphql`. Use documented authorization and token expiry/available permissions, with credentials kept server-side. Fetch the user's library with bounded pagination, then related reads and editions; don't infer a universal incremental cursor from a convenient `updated_at` field.

The pinned [schema](https://github.com/hardcoverapp/hardcover-docs/blob/6ee717ec9a2c810d859a2a0b09a0a95354ac2a27/schema.graphql) distinguishes catalogue `books`/`editions`, personal `user_books`, and `user_book_reads`, including `insert_user_book_read` and `update_user_book_read`. Edition identifiers include ISBN/ASIN and language/page metadata. The [progress-update guide](https://github.com/hardcoverapp/hardcover-docs/blob/6ee717ec9a2c810d859a2a0b09a0a95354ac2a27/src/content/docs/api/guides/UpdatingABooksProgress.mdx) is still an empty draft; do not treat it as a tested write recipe.

Preserve native nullable progress, edition/read IDs and reported date precision. A page/audio-position field is **position**, not time actively spent reading/listening. A read-history row is not a measured session. Repeated reads must survive as separate attempts instead of one overwritten start/finish pair. Handle GraphQL `errors` even when HTTP returns 200; incomplete/error captures cannot clear prior evidence.

BookOrbit's [client](https://github.com/bookorbit/bookorbit/blob/e767358443419253eeed782518d6fc4f675d69fd/server/src/modules/hardcover/hardcover-client.service.ts) spaces requests by at least 1100 ms and allows three retries. Those are implementation constants, **not an authoritative Hardcover quota**. The pinned API introduction separately documents daily/burst/per-minute limits and rate headers; the [Phase A capability section](../plans/reading-hub.md#3-provider-capabilities-and-directionality) records those published values. Verify actual account/token headers and use bounded backoff/rate-limit handling in any future Iroha connector.

### Kindle: correct the “no API” blanket claim

Amazon publishes [Data Portability scopes](https://developer.amazon.com/docs/amazon-data-portability/available-scopes.html), including `portability::kindle_activity`, Kindle registrations and usage/engagement-related scopes. That makes “Amazon offers no API whatsoever” too broad.

However, the inspected Kindle activity schemas primarily describe device interaction/settings events such as hibernation, connectivity and power changes. Their event timestamps/ASINs do not establish book completion, precise reading duration or an arbitrary Whispersync read/write interface. Do not count powered-on time as reading time. The official [overview](https://developer.amazon.com/docs/amazon-data-portability/overview.html) and authorization/eligibility requirements must be checked for the actual region/account before a separate capability spike; no real account was tested here.

[KU terms](https://digprjsurvey.amazon.com/csad/help/node/201556940) say access depends on membership and selected titles are removed when membership ends. Amazon's [Send-to-Kindle/USB documentation](https://digprjsurvey.amazon.com/csad/help/node/TCUBEdEkbIhK07ysFu) supports transferring personal documents, not exporting KU loans as permanent BookOrbit EPUBs. A KU title's metadata and personal observations may outlive the loan without implying ownership of its file.

## Fit into existing Iroha boundaries

These are proposed mappings, not new schemas or approved parser contracts.

| Concept from handoff | Existing Iroha home / necessary caution |
| --- | --- |
| Work | `tb_media_works`, existing `Media.WorkExternalID` / work fields |
| Edition/translation | Separate `tb_media_items` with work relationship, language, page count and verified external references; don't merge by generic title/year fallback |
| Author/translator | Existing creator/role tables; current typed `Media` observations need the smallest demonstrated extension to carry roles |
| Source identifiers | Existing external-reference model; distinguish global Hardcover catalogue IDs from instance-local BookOrbit IDs and account-scoped personal records |
| Current reading state | Existing media progress/selected-state projection; retain competing observations and choose source precedence deliberately |
| Dated personal updates | Existing `MediaHistory`/state-update boundary where source timing is actually known; don't invent exact timestamps from date-only or current snapshots |
| Sessions/annotations/attempts | Separate demonstrated contracts; do not force unsupported evidence into consumption events or add speculative tables upfront |
| Provider capabilities/outbound state | Extend existing provider/job seams only when an approved write journey requires them; not a new service |

Raw captures remain immutable evidence; normalization is DB-free and projections are rebuildable under an approved retention policy. Existing default raw-byte purge remains a real prerequisite to resolve, not something to disable silently. Public books still use the validated live `/public/v1` projection, with explicit per-book opt-in; provider privacy and Iroha public visibility are independent controls.

### Matching and writer ownership

- One work can have a WeRead Chinese translation, an English EPUB and a Kindle ASIN edition. Link the work only with sufficient evidence; retain distinct items and positions. Title/author similarity proposes candidates, never silently confirms translation equivalence.
- Keep native source IDs, language/edition evidence, capture time, source time/precision and matching decisions. ASIN is not a work ID, ISBN alone does not authorize file/locator equivalence, and Hardcover's selected edition must not silently replace another source's edition.
- Treat Hardcover state written by BookOrbit as an echo, not independent evidence of another reading session. Never add the same time twice.
- Initial Iroha connectors stay read-only. For BookOrbit-managed books, let BookOrbit be the Hardcover writer. If later Iroha writes WeRead/manual-Kindle observations for books outside that route, assign **one writer per target user-book/field set**. Different translations can still share Hardcover's work-level personal record, so per-edition ownership alone is insufficient.
- A future outbound operation needs an explicit allowlist, verified mapping, durable pending/acknowledged/failed/uncertain result, retry/reconciliation policy and owner approval. A timeout is not proof that a mutation failed; check the target before creating another read attempt. No “largest percentage wins” or two-way writeback loop.

## Recommended next slice and remaining decisions

The smallest useful expansion is a **private, read-only Hardcover saved-snapshot contract** that can represent a Kindle/metadata-only book without an EPUB, alongside the existing BookOrbit snapshot contract. This is a sequencing proposal, not a rewrite of #144's BookOrbit acceptance criteria.

Before code: pin schema/API behavior; obtain separately approved credentials; capture sanitized complete/partial fixtures; decide source/account identity, date precision, raw retention and projection precedence. Then prove upload → pure parser → existing media library/state → replay without duplicate work/items/state, retaining the native read-attempt evidence even where normalization is initially limited. Include null progress, reread, wrong-language edition, backward progress, GraphQL partial errors and source outage. No scheduler, public UI or outbound writes are necessary for that first proof.

Still requiring explicit decisions/acceptance:

1. Which actual clients/devices pass BookOrbit's cross-device journey, and whether migration from WeRead is desirable at all.
2. Whether Hardcover is only a catalogue/list mirror or also a source of owner-entered history; verify free/account/token/rate conditions rather than promise them from API friendliness.
3. Which language/editions and fields are authoritative when the same work is present in several sources.
4. Whether to investigate eligible Amazon portability or accept manual Kindle observations for now; neither requires forcing KU content into BookOrbit.
5. Whether any outbound Iroha writes are needed after the read-only slice. BookOrbit's existing writer may already cover the useful local-file case.

Keep #142 open. Reader migration, durable evidence import, social-list mirroring and public sharing are independently accepted outcomes.
