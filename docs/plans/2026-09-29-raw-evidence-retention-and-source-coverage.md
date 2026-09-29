# Raw evidence retention and source coverage

Status: proposed; not approved implementation scope.

Prepared 2026-09-29 from the live `iroha_v05` database (1109 MB) and the admin Sources page.

## Findings

- `tb_raw_files` holds 93 files / 563 MB. 555 MB is nine `apple_health_export` zips imported once on 2026-09-28. Nothing deletes raw files today.
- The original exports live outside iroha, so a raw copy is only a short-lived replay and debugging aid, not the archive.
- Route points dominate the database: `tb_activity_observation_route_points` (461 MB) and `tb_activity_route_points` (392 MB) hold the same 640,396 rows. Samplings and sleep segments are duplicated the same way.
- Every source on the admin page shows `unknown coverage`. `connection()` starts at `unknown` and only changes on a `covered`, `covered_empty` or `partial` assertion. Only `health_auto_export` records assertions, and all 15 are `unknown`. Apple Health export, AniList, AniList activity and Bangumi record none.

## Design 1: raw file retention

Goal: keep a raw file for 7 days, then delete the bytes and keep the record.

- Add `purged_at timestamptz null` to `tb_raw_files` (new migration; `storage_path` stays as a record of where it was).
- Retention applies to every source kind (decided 2026-09-29). Add a `raw_file_purge` scheduled job on the existing `tb_jobs` interval mechanism, daily. It selects rows with `purged_at is null and created_at < now() - interval '7 days'` and whose import job is `succeeded`.
- For each row: remove the file (missing file is not an error), then set `purged_at`. Order matters: delete first, mark second, so a crash retries safely.
- Keep the `tb_raw_files` row. `sha256` uniqueness is what skips duplicate re-uploads, and receipts, import jobs and snapshots reference it by foreign key.
- Failed or queued imports are never purged, so retry-import keeps working. Retry-import on a purged file returns a clear "raw file purged" error.
- Retention is `IROHA_RAW_RETENTION_DAYS` (default 7; 0 disables). Optional per-source-kind override only if a need appears.
- The admin System card shows live bytes (`purged_at is null`) next to the total count, so the "563 MiB" figure drops after the first purge.

Trade-off: purged imports can no longer be replayed. The docs' "replay raw evidence" cut-over path then depends on the external originals. This plan accepts that because the originals exist elsewhere.

## Design 2: source coverage

Goal: each source states what it knows, or honestly stays `unknown`.

| Source | Category | Assertion |
|---|---|---|
| `apple_health_export` | `health` | `full_snapshot` / `covered` over the calendar days the export contains |
| `anilist` | `media_list` | `full_snapshot` / `covered` over the sync run once the cursor is exhausted |
| `anilist_activity` | `media_activity` | `incremental` / `covered` over the window the run fetched (the connector reports the window start) |
| `bangumi` | `media_list` | `full_snapshot` / `covered` over the sync run once the cursor is exhausted |
| any connector | as above | `partial` when a run stored pages but failed |
| `health_auto_export` | `health` | unchanged; `unknown` stays valid |

Existing imports are not backfilled. Connector sources fill in on their next daily sync; the already-imported Apple exports stay without coverage.

- Providers return `provider.CoverageAssertion` in the import batch; `persistCoverageTx` already stores it. No schema change.
- Parsers must only assert `covered` when they can prove completeness (for connectors, the cursor was exhausted). Otherwise they emit `partial` or nothing.
- The Sources page label reads "unknown coverage" for both "no assertion" and "assertion says unknown". Show "no coverage recorded" for the first case so the two are distinguishable.

## Design 3: trim the duplicated child rows

`activity_import.go` fills `tb_activity_observation_route_points` (and samplings, laps) by `insert ... select` from the canonical `tb_activity_route_points` for the same activity, so the observation copy is derived, not independent evidence. ADR 0001 keeps observations so a second provider can be compared against the first. Today an activity has one observation, so the copy adds no information.

Decision: store child rows once, on the selected observation's canonical tables, and materialize observation child rows only when an activity gains a second, non-selected observation.

- Stop the `insert ... select` copies for route points, samplings and laps (and the sleep segment equivalent) when the observation is the selected one.
- Observation read paths fall back to the canonical rows when the observation is selected and has no rows of its own.
- Backfill: `ops/sql/2026-09-29-trim-observation-copies.sql` truncates the four child tables (route points, samplings, laps, sleep segments), which reclaims the space immediately without `vacuum full`. It refuses to run if any activity or sleep session has more than one observation. Expected saving: about 460 MB of route points, about 60 MB of samplings, about 15 MB of sleep segments.
- Reprocess after a parser-version bump still rewrites the canonical rows; there is no second copy to keep in step.

Risk: when a second provider arrives for the same activity and becomes the selected one, the previously selected observation needs its rows materialized first. Do that in the selection switch, in one transaction, and test it.

## Decisions (2026-09-29)

- Purged imports are frozen at their last parse; a parser-version bump cannot reprocess them. Re-import from the external original.
- The retention clock starts at upload (`created_at`) and only covers files whose latest import job is `succeeded`.
- Failed imports are purged after 30 days. `queued` and `parsing` are never purged.
- The existing observation copies are deleted in a one-off backfill, followed by `vacuum full`, in a quiet window after a `pg_dump` of `iroha_v05`. It is not part of the automatic migration run.
- The admin label "Raw evidence" becomes "Raw files", showing live bytes against the total count. New glossary terms: Raw file, Purged raw file, Selected observation.
- Apple Health export asserts `covered` per category present in the file.
- The admin page distinguishes "no coverage recorded" from an asserted `unknown`.

- AniList and Bangumi write one assertion per completed sync (`covered` when the cursor is exhausted, `partial` otherwise), never one per page.
- AniList activity asserts the window the run actually fetched, not the whole lookback.
- The second-provider materialization is deferred. An import that would replace a selected observation from a different provider fails loudly until it is built.
- Delivery is one branch, with the work in separate commits: retention, coverage, trim.

## Verification

- Purge job test: a 8-day-old succeeded file is deleted and marked; a 6-day-old one and a failed one are untouched; a missing file still marks the row.
- Re-upload of a purged file's sha256 is skipped as a duplicate.
- Coverage: after re-importing fixtures, `/api/v1/connections` returns a non-`unknown` collection for each source with a proven window.
- Live: `select count(*) from tb_raw_files where purged_at is null` drops to the files younger than 7 days after the first run.

## Open questions

None. A grep of `apps/` (excluding tests and migrations) shows only the two import files touch the observation child tables, so nothing reads them and the copies can go. The read-path fallback in Design 3 is only needed once a second provider exists.

Run the backfill on the live database by hand after a `pg_dump` of `iroha_v05`; it is not part of the migration run.
