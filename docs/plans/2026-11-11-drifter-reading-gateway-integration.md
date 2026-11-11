# Drifter reading gateway integration: implementation plan

Date: 2026-11-11. Related: [ADR 0011](../adr/0011-provider-neutral-reading-evidence.md), [Reading Hub Phase A](reading-hub.md), workstation plan `docs/runbooks/plans/2026-11/2026-11-11-drifter-reading-gateway-and-iroha-canonical-cockpit.md`.

Status: proposed; implementation staging.

## Outcome and scope

Ingest personal reading history from **Drifter (`drifter-server`)** into Iroha's canonical PostgreSQL media store. Drifter serves as the local provider-neutral reading intake gateway, aggregating WeRead shelf books (via official Agent Gateway 1.0.5), Douban ISBN metadata, and verified Hardcover target links.

This integration delivers 100% retention for the owner's WeRead catalog (~1,174 books, including domestic Chinese technical manuals and custom uploads) while maintaining Iroha's strict data model contracts:

1. **Connector & pure parser**: `KindMediaSyncDrifter` fetches paginated snapshots with Bearer token authentication into `tb_raw_files`. The adapter maps snapshot facts into typed `observations.Media`.
2. **Work-edition ontology**: Translations remain separate edition items (`tb_media_items`), while linking to unified conceptual works (`tb_media_works`) via Drifter's verified `matched_target.work_id`.
3. **Creator persistence**: Authors and translators are persisted into `tb_media_creators` and `tb_media_creator_roles`.
4. **Honest time semantics**: Real WeRead `finishTime` timestamps land as `finished` events in `tb_media_consumption_events`; progress updates `tb_media_progress`; no synthetic `started` or `session` events are fabricated.
5. **Decisions-first resolution**: Matching decisions $\to$ normalized ISBN refs $\to$ Drifter bridge $\to$ create item. Generic title/year auto-attach is disabled.

---

## Implementation Slices

### Slice 1: Creator observation & persistence extension

Extend Iroha's core media models to support author and translator ingestion:

1. In `apps/iroha-core/observations/observations.go`:
   - Add `Creators []MediaCreatorRole` to `Media` observation struct.
   - Define `MediaCreatorRole` with `Name`, `Role` (`"author"`, `"translator"`, `"publisher"`), and `Provider`.
2. In `apps/iroha-imports/media_import.go`:
   - Add creator persistence in `persistMediaMetadata`: insert creators into `tb_media_creators` and link roles into `tb_media_creator_roles` scoped to item ID.
3. Unit tests in `media_import_test.go` verifying that creator roles are idempotently written and deduplicated.

### Slice 2: Drifter connector and adapter

Implement the ingestion bridge for `drifter-server`:

1. In `apps/iroha-runtime/jobs/service.go`:
   - Register `KindMediaSyncDrifter = "media_sync_drifter"`.
2. In `apps/iroha-providers/drifter/`:
   - `connector.go`: implements `connector.Connector` fetching `GET /api/v1/reading/snapshot?limit=50&cursor=...` with Bearer token authentication from environment `DRIFTER_API_TOKEN`.
   - `adapter.go`: pure parser converting Drifter JSON snapshots into `[]observations.Media`.
     - Maps `status` vocabulary: `"completed"` $\to$ `"completed"`, `"in_progress"` $\to$ `"in_progress"`, `"planned"` $\to$ `"planned"`.
     - Maps authors and translators to `Creators`.
     - Emits `finished` milestone event when `finished_at` is non-zero.
     - Preserves `isbn`, `publisher`, and `matched_target` cross-references.
   - `adapter_test.go`: unit tests against synthetic Drifter snapshot fixtures.
3. In `apps/iroha-providers/connectors/registry.go`:
   - Register Drifter connector descriptor.
4. In `apps/iroha-imports/source_priority.go`:
   - Register `drifter` and `hardcover` in `mediaProgressSourceWins` priority ladder.

### Slice 3: Work-level edition linking extension

Implement cross-edition work deduplication:

1. In `apps/iroha-imports/media_import.go`:
   - When resolving an incoming book item, if `matched_target.work_id` is present, check whether a canonical `tb_media_works` row already exists for that target work ID.
   - If found, associate the new edition item with the existing work (`item.WorkID = existingWork.ID`), preserving distinct edition items while sharing the conceptual work.
2. In `apps/iroha-imports/media_resolution.go`:
   - Enforce decisions-first ladder: `Matching Decisions` $\to$ `ISBN refs` $\to$ `Drifter bridge` $\to$ `Create item`.
   - Explicitly disable generic title/year auto-attachment when `media.MediaType == "book"`.
3. Integration tests in `media_resolution_integration_test.go` verifying that Chinese WeRead edition and English Hardcover edition resolve to distinct items under the same work.

### Slice 4: Cockpit surface and report verification

1. Verify `/library?family=book` lists ingested books with accurate status and filters.
2. Verify `/library/[id]` displays authors, translators, ISBN, and read completion dates.
3. Verify monthly reports reflect finished books in media sections.

### Slice 5: Recurring schedule wiring & page limit calibration

1. In `apps/iroha-providers/drifter/connector.go`:
   - Calibrate `DefaultLimit = 10` (from 50) and `DefaultClientTimeout = 60 * time.Second` to respect 1s upstream WeRead pacing.
2. In `apps/iroha-runtime/config/config.go`:
   - Add `EnvDrifterSyncInterval = "IROHA_DRIFTER_SYNC_INTERVAL"`.
3. In `apps/iroha-job/main.go`:
   - Wire `ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncDrifter, os.Getenv(config.EnvDrifterAPIToken) != "", os.Getenv(config.EnvDrifterSyncInterval))`.
4. Update `schedule_integration_test.go` and run verification gates.

---

## Verification Gates

- Pre-commit: `make check` (quality floor + fmt + vet + lint + tests + contract checks + web checks).
- Pre-PR: `make validate` (full server, worker, and web builds).
- Integration: DB-backed `make test-integration` against disposable PostgreSQL.
