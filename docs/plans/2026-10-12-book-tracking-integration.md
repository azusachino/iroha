# Book tracking integration: preparation and staged plan

Date: 2026-10-12. Parent: [#142](https://github.com/azusachino/iroha/issues/142).

Status: proposed; research and acceptance planning only. This document does not authorize implementation, deployment, purchases, account access or migration away from WeRead.

## Outcome and scope

Keep personal reading history independently of its reader, then allow deliberate public sharing. BookOrbit owns files, locators and reader synchronization; Iroha owns sourced history, canonical identities and sharing decisions. WeRead can remain an independent parallel source. Hardcover is optional and should use BookOrbit's existing write integration rather than a second Iroha writer.

The owner confirmed single-session, tier-0 preparation: research, plan and issues, with no Asobi or independent verifier. This plan and [the research](../integrations/bookorbit-research.md) are the content-owned paths. The owner additionally approved a blank-line-only formatting repair in `docs/plans/2026-10-08-cockpit-visual-overhaul.md` after the full documentation gate exposed 15 pre-existing errors. No application code or deployment is part of this task. Current evidence is source inspection, not runtime feasibility acceptance.

Follow the workstation-task playbook and Iroha's `AGENTS.md`, `CONSTRAINTS.md` and `CONTRIBUTING.md`. Future release, public contract and deployment work must be scoped under the corresponding higher tier and owning playbook before execution.

## Decisions proposed from research

1. Prefer a version-pinned authenticated HTTP snapshot over direct BookOrbit DB reads. Optional Swagger exists, but runtime contract and source/image correspondence must be captured first.
2. Preserve Iroha's fetch/evidence/pure-parser boundary, durable jobs, SQLx schema and shared frontend assets. No new persistence engine, scheduler, service or renderer.
3. Reuse media works/items/progress/state history. Never turn a list sync into consumption events; measured duration, reconstructed sessions and provider aggregate time need different evidence contracts.
4. First slice stays private. Public sharing follows the current validated live `/public/v1` projection, **not the retired static builder/export pipeline in #142**. If the owner specifically wants static-only books data, record that architectural choice before implementation.
5. Treat raw evidence retention as a real acceptance prerequisite. Default raw-byte purge prevents indefinite replay of unmodeled native fields. Do not silently disable global purge or promise replay from hashes.
6. Keep source identity and edition matching conservative. Scope BookOrbit IDs by stable configured instance/account identity; don't auto-merge book editions/translations using Iroha's generic title/year fallback. Keep differing provider states; select projection precedence deliberately.

## Dependency order and issue index

Acceptance lives in the linked issues; this file preserves sequencing and reasoning rather than another live task ledger.

| Slice | Dependency | Owning acceptance issue |
| --- | --- | --- |
| A. Isolated contract and reader feasibility spike | Owner selects environment, source/image pin and devices; provisions credentials separately | [#143](https://github.com/azusachino/iroha/issues/143) |
| B. Private saved-snapshot import | A's API contract accepted; retention and identity/projection policies approved | [#144](https://github.com/azusachino/iroha/issues/144) |
| C. Read-only scheduled snapshot connector | B, plus approved credentials and polling scope | [#145](https://github.com/azusachino/iroha/issues/145) |
| D. Explicit per-book public sharing | B and source capability accepted; public contract/privacy decision | [#146](https://github.com/azusachino/iroha/issues/146) |

A reader-sync failure blocks migration from WeRead, not necessarily importing useful evidence. A provider-contract failure blocks that provider's adapter. C and D do not depend on retiring WeRead. Optional WeRead, sessions/annotations and Hardcover expansion remain in #142 until their separate contract checks justify narrower implementation issues.

## Slice A: validate before producing a connector

**Goal:** establish reproducible API and reader evidence for a selected BookOrbit version, not bless upstream `main` from its README.

Owner choices before execution:

- Isolated host/environment, local writable storage, backup destination, approved private ingress and persistent books/app/DB volumes. No Iroha database changes.
- Tested app/database image digests, source revision, native app/plugin versions and legal DRM-free EPUB sample.
- Actual device/OS inventory, including iOS/iPadOS 26 eligibility, KOReader support on Android, and acceptable app offline/download limits.
- Ordinary BookOrbit account/library scope, credential provisioning and whether local login is available in the chosen auth configuration. No credentials in issues, files or logs.

**API acceptance:** capture version/OpenAPI when enabled; native login/refresh/revocation; ordinary-user versus inaccessible-library reads; query pagination with collapsing disabled; null/zero progress; backward progress and file replacement; status values; schema drift and outage behavior. Keep sanitized response-shape fixtures private where necessary. Freeze a versioned snapshot-envelope contract and permitted methods/routes. Determine whether metadata export can substitute for any detail reads.

**Reader acceptance:** record exact before/after locations and outcomes for iOS→Android→iOS, offline/reconnection, old-position reopen, concurrent conflicting updates, deliberate rewind, status/completion, annotations and server restart. Set acceptable position drift with the owner before claiming PASS. If the owner cannot test a device, label that journey unverified; do not replace it with web-only testing.

**Operations acceptance:** restore books/app state/DB from backup into a separate test instance. Document licensing notices and exact pins. No migration away from WeRead until the required real-device journey passes.

Likely owned deliverable: a dated Iroha spike report and redacted contract fixtures; actual deployment records belong to the selected environment's owner. Estimated scope: small research report, operational scope separately confirmed.

## Slice B: first useful private vertical slice

**Journey:** save an approved BookOrbit snapshot → upload through existing raw-file/import APIs or CLI → pure parser → canonical media library/state → existing private reads. This tests ownership/replay without introducing network scheduling or public UI.

Implementation partitions, serially verified rather than one broad rewrite:

1. Provider/parser registration and DB-free normalization fixtures: instance/account/book identity, files kept distinct, native status mapping, nullable percent, rewind/stale input, no invented events. Proposed source kind `bookorbit` needs registered constants and an envelope version, not inlined literals.
2. Import persistence seam: enforce conservative book resolution, deterministic source selection, and source date/time precision. Use existing creator tables for authors via the smallest typed observation/persistence extension. Preserve unsupported native distinctions in retained evidence and make any lossy projection visible. Exact file locator storage is deferred.
3. Private upload/read/replay journey: unchanged snapshot and parser-version replay produce no duplicate works/items/state; a real state reversal remains visible. Cover complete, failed and partial captures; absence never deletes prior evidence. Source-native payload and capture/version/hash metadata remain durably replayable under the approved retention policy.

Likely files: a new provider package and fixtures; provider/parser registry and import validation; core media observations only as required; media import/resolution/projection helpers and tests; SQLx migration only for a demonstrated gap. These are separate implementation seams, not permission for a many-module rewrite. No new book-only parallel schema.

Tests are test-first for normalization/import behavior. Use documented focused Make checks during iteration; acceptance requires `make check`, DB-backed `make test-integration` against disposable `iroha_test*`, and `make validate` before PR. Exercise actual upload/read replay, not only Go unit tests. Never use `make db-reset` on imported data.

**Checkpoint:** owner reviews the private import result and retention/identity semantics before network scheduling or public sharing begins.

## Slice C: bounded read-only synchronization

**Journey:** existing worker job fetches allowed BookOrbit reads → durable evidence → B's parser → private library updated, with retries and source attention.

Use existing connector registry, `SyncRunner`, raw snapshot store, durable jobs/schedules and coverage reporting. Separate auth-session handling from permitted data-read methods. Pagination limits, total scan/time/byte budget and polling interval are explicit approved configuration. Store neither credentials nor auth responses as evidence. Handle refresh expiry/revocation, 429/retry signals, timeouts, malformed pages and unsupported versions without clearing prior canonical state.

Use full scans initially; don't claim timestamp incrementality. Preserve page dedupe/resume and mark coverage complete only at successful exhaustion. Offset scans can race mutations: missing entries are observations of absence, not destructive reconciliation. Keep user decisions and other provider evidence through replay/source outages.

Likely paths: provider connector/tests, worker registration/configuration, server sync-trigger registration, private OpenAPI/config docs. Separate network fetch/auth and worker wiring into reviewable patches. Acceptance: fake-HTTP tests plus approved isolated live capture/retry journey, `make check`, `make test-integration` and `make validate`. No live account polling or deployment without renewed scope.

## Slice D: public currently-reading with explicit opt-in

**Journey:** private per-book sharing choice → validated sanitized public projection → shared Grapher view → public client, without exposing private API/data.

Partition into: (1) private visibility storage/control and tests; (2) public projection/schema/validation/cache invalidation; (3) shared view and public adapter. Confirm published API criteria before code. New presentation assets live in `packages/`, not web/public-app local themes.

Default private, including imports and new source matches. Publish only an allowlisted book ID/title/authors/status/optional percent and lawful cover. Omit unsupported last-read time; do not convert status to presence. Annotations require a separate opt-in slice, not this widget. Exclude provider/account IDs, filesystem paths, credentials, notes and locators from every public response.

Test unpublish across snapshot revisions and browser/edge cache semantics; today's public cache can last a day. Decide acceptable revocation latency before claiming immediate removal. Month/year/lifetime levels must work end-to-end wherever enabled.

Acceptance: privacy negative tests and API/ETag/cache tests, private/public isolation, real Playwright render and opt-in/unpublish journey on a seeded local stack, affected consumer builds, `make check`, `make test-integration` where storage changes, `make validate`, and the required fresh independent verification for the future published-contract tier. Preparation gates do not establish these behaviors.

## Deferred capabilities, without speculative implementation

- **WeRead parallel import:** first authorize a read-only API-key contract probe and hosted-service terms/quota check. Use official gateway 1.0.5, flat parameters, 0–100 percent and seconds, explicit version-upgrade attention. Saved-snapshot metadata/progress can precede unattended polling. Shelf privacy does not grant Iroha sharing permission. Edition-aware ISBN/provider matches precede any cross-source merge; ambiguous matches stay separate.
- **Time analytics:** WeRead natural-period totals are sourced aggregates, not exact sessions. BookOrbit late/session rewrites and inferred intervals require a separate duration/revision/evidence model. Never calculate measured time from percent differences or double-count overlapping provider totals.
- **Annotations:** preserve source IDs/revisions/deletes and private locator/excerpt data; determine lawful sharing independently. Existing media `note` fields are not a complete annotation schema.
- **Hardcover:** prefer BookOrbit's existing optional integration. No Iroha write client or feedback loop in the initial release.

## Preparation acceptance and stop boundary

Source revisions and citations are recorded in the research. The plan identifies a first private vertical slice, ordering, gates, actual Iroha gaps and owner-blocked runtime checks. Scoped issues are created only after local preparation gates pass. Changes stay on `docs/book-integration-research`; no unrelated PR/dependency work is touched.

Preparation verification on 2026-10-12: baseline `make check` exited 0; initial `make fmt-docs-check` failed on citation syntax and the existing plan's missing blank lines, so dependent validation did not run. After citation fixes and the owner-approved blank-line repair, full `make fmt-docs-check` and `make validate` both exited 0. The latter includes `make check` and application builds. These are preparation gates, not provider/device runtime evidence. Issue links added afterward are documentation-only changes and receive a final documentation/check pass before commit.

Run `make fmt-docs-check`, `make check` before commit, and `make validate` before a docs PR. A failing/unexecutable gate blocks commit/publication; do not lower it. Review source-backed claims directly at tier 0. Record actual gate results and owned paths in closeout. Stop for owner review after research/plan/issues; future implementation requires a new scoped task.
