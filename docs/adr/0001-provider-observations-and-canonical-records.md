# ADR-0001: Provider observations and canonical domain records

Related capability contract: [Provider capability contracts](../provider-capabilities.md)

- Status: Proposed
- Date: 2026-07-13
- Scope: activities, sleep, daily health metrics, and future provider-backed
  modules such as Garmin, AniList, and Bangumi

## Context

Iroha currently has normalized parser DTOs and domain tables, but the
canonical boundary is incomplete.

The current flow is:

```text
Apple Health or GPX adapter
  -> ActivityObservation / SleepObservation / DailyMetricObservation
  -> domain table
```

The current activity-shaped DTO is reusable across Apple Health and GPX, but
Apple-specific reconciliation still lives in `tb_apple_source_items` and in
branches inside `imports.Service`. Sleep and daily records are stored directly as if the Apple
observation were already the canonical truth. Route points, samplings, and
laps attach directly to `tb_activities`, which prevents two devices from
reporting the same workout without overwriting or duplicating child data.

The roadmap requires multiple producers of the same domain facts:

- Apple Health and Garmin may both report a workout, sleep episode, heart rate,
  steps, or recovery metric.
- AniList and Bangumi may both describe one canonical anime or manga work.
- Future connectors may provide overlapping or conflicting observations.

The product must preserve both the evidence and the comparison opportunity.
It must not silently average or discard provider disagreements.

## Decision

Model provider data as observations of canonical domain records.

```text
raw evidence
  -> source record identity and content hash
  -> provider observation
  -> optional canonical record link
  -> selected/comparison projection
```

An observation is what one provider or device reported. A canonical record is
Iroha's domain-level object representing the user's event or entity. A
canonical record may have zero, one, or many observations.

### Shared source identity

Every provider observation has a provider-scoped identity:

```text
(provider, source_kind, source_key)
```

The source key is stable within that provider and source kind. It is never the
archive hash. The content hash detects a changed observation during a later
snapshot or connector sync.

The source identity layer is shared conceptually, but domain observations keep
their domain-specific columns. We do not introduce one universal key-value
`tb_facts` table.

### Activities

`tb_activities` becomes the canonical activity/session record. Provider data
moves to `tb_activity_observations`.

```text
tb_activities
  canonical session: sport, time window, chosen summary fields

tb_activity_observations
  provider/device report: source identity, source metrics, raw evidence

tb_activity_observation_routes
tb_activity_observation_samplings
tb_activity_observation_laps
  child measurements belonging to one provider observation
```

An observation may be unlinked while matching is unresolved. Two observations
with overlapping time windows are not automatically merged. Matching starts
with explicit source identity and later uses conservative time/sport/distance
heuristics with a visible match status.

Existing activity read APIs continue to return the canonical activity and its
selected observation by default. Comparison APIs can expose all observations.

### Sleep

`tb_sleep_sessions` remains a canonical sleep episode. Each provider import
creates one or more sleep observations linked to that episode.

```text
tb_sleep_sessions
  canonical episode and selected summary

tb_sleep_observations
  provider/device sleep report and metrics

tb_sleep_observation_segments
  provider-specific stage segments

tb_sleep_session_observations
  link, match status, confidence, and preferred-observation marker
```

Apple and Garmin can therefore report the same night while retaining their
different stage totals and segment boundaries.

### Daily summaries and metrics

Daily facts are observations first and projections second.

```text
tb_daily_summary_observations
tb_daily_metric_observations
  one row per provider/device/day/fact

tb_daily_summaries
tb_daily_metrics
  canonical selected daily projection
```

The canonical projection records which observation supplied the selected value
and how it was selected. When providers disagree, the comparison view exposes
both values; no implicit average is used unless the metric semantics explicitly
permit aggregation.

For cumulative metrics such as steps and distance, source selection and
interval de-duplication remain domain rules. They are not solved by a generic
last-write-wins policy.

### Provider adapters

Apple Health, Garmin, GPX, AniList, and Bangumi are provider adapters, not
canonical types and not Go traits. Each adapter emits observation contracts;
the word `parsed` is not part of the domain vocabulary. Each adapter is
responsible for:

- reading its raw evidence or connector payload;
- emitting domain observation DTOs such as `ActivityObservation`;
- producing stable provider/source keys;
- producing content hashes;
- declaring parser/adapter version.

Canonicalization, observation matching, projection selection, and conflict
status belong outside the provider adapter.

The first source-code boundary should be domain observation DTOs and an
adapter registry, not a large generic provider framework.

## Migration strategy

The migration is a clean current-schema install. Raw exports are the canonical
evidence, and the local database is intentionally rebuildable, so there is no
SQL backfill of the old Apple-derived rows.

1. Replace the historical migration chain with one current-schema migration.
2. Create provider-observation tables and nullable selected-observation links;
   leave them empty.
3. Switch source adapters and persistence to write observations plus canonical
   projections.
4. Bump the parser version and re-import the raw Apple ZIP. The import job
   recreates the current canonical data and its observations from that evidence.
5. Retire `tb_apple_source_items` only after the new import and read APIs pass
   integration and real-import smoke checks.

This keeps migration SQL focused on structure. Data transformation belongs to
the importer, where the raw ZIP, parser version, reconciliation rules, and
failure handling are visible and testable.

## Alternatives considered

### Keep one canonical row per provider

Rejected. It makes comparison possible only through ad hoc joins and treats
Apple/Garmin overlap as duplicate activity rather than multiple observations
of one event.

### Merge providers immediately using heuristics

Rejected. A false merge is harder to repair than an unresolved match. Matching
must be explicit and observable.

### Universal `tb_facts` or entity-attribute-value storage

Rejected. Stable domains need typed tables for time ranges, segments, routes,
units, and query performance. Shared source identity is sufficient reuse.

### Provider-specific parallel canonical tables

Rejected as the long-term model. It duplicates read APIs and prevents shared
cross-provider projections, although temporary compatibility tables are allowed
during migration.

## Read-model and join strategy

Observation storage must not turn ordinary cockpit reads into provenance joins.
The canonical tables are read models, not merely normalized link targets.

The default read path is deliberately denormalized:

```text
tb_activities
  -> selected_observation_id + selected summary fields

tb_sleep_sessions
  -> selected_observation_id + selected summary fields

tb_daily_metrics
  -> selected_observation_id + selected value
```

Canonical rows are refreshed in the same transaction as an observation link or
selection change. Existing list/detail APIs query these canonical rows directly
and do not join observation tables.

Observation joins are reserved for explicit surfaces:

- a comparison view (`activity/:id/observations` or `sleep/:id/compare`);
- provenance/debug views;
- reprocessing and reconciliation jobs.

For repeated comparison reads, add a domain-specific comparison projection or
materialized table rather than making every dashboard query repeat a wide join.
Keep child measurements observation-owned, but expose selected route/sampling/
lap IDs through the canonical detail service so the normal activity page still
performs one bounded lookup.

This gives us three practical rules:

1. canonical list queries stay one-table or one-purpose aggregate queries;
2. provenance is opt-in, never implicit in every DTO;
3. denormalize stable selected values at the canonical boundary and refresh
   them transactionally.

## Consequences

Positive:

- Apple data remains valid while Garmin and other providers are added.
- Device disagreement becomes visible rather than destructive.
- Raw evidence, provider identity, canonical records, and projections have
  clear ownership.
- Media providers can use the same conceptual shape without sharing health
  tables.

Costs:

- More storage and write-time projection work; ordinary reads do not pay the
  observation join cost.
- Existing route/sampling/lap APIs need a canonical-to-observation selection
  rule.
- Existing local data must be re-imported from raw evidence after the schema
  reset; this is an explicit operational step, not hidden migration work.
- Conflict and match states become product concepts that need UI later.

## Non-goals

- Automatically deciding which watch is medically or personally correct.
- Building a universal ontology for every future personal-data domain.
- Rewriting all parser code before the persistence boundary is ready.
- Adding Garmin ingestion in this ADR migration; this ADR prepares the boundary.
