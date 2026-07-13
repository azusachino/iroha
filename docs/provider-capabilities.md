# Provider capability contracts

Core contract: [Iroha core capabilities](capabilities/core.md)

Provider implementations:

- [Apple Health](capabilities/providers/apple-health.md)
- [Garmin](capabilities/providers/garmin.md)
- [AniList](capabilities/providers/anilist.md)
- [Bangumi](capabilities/providers/bangumi.md)
- [Goodreads](capabilities/providers/goodreads.md)
- [WeRead](capabilities/providers/weread.md)

This document describes how external data providers implement Iroha's core domain capabilities. It is intentionally written in trait/implementation style even though the Go code uses interfaces,
registries, and concrete adapters rather than language traits.

The purpose is to make provider expansion explicit:

```text
core capability contract
  -> provider implementation
  -> observation DTOs
  -> canonical projection
```

A provider does not need to implement every capability. Unsupported capabilities must be declared, not represented as fabricated zero values.

## Contract vocabulary

### Provider

A producer of evidence or metadata: `apple_health`, `garmin`, `gpx`, `anilist`, or `bangumi`.

### Observation

The provider's report, preserved with provider identity, source key, content hash, device/source metadata, and raw evidence reference.

### Canonical record

Iroha's domain-level record. It may have zero, one, or many observations.

### Capability

A domain contract an adapter can implement. Capabilities describe semantics, not file formats.

## Adapter shape

The conceptual adapter contract is:

```go
type ProviderAdapter interface {
    Provider() string
    Capabilities() []Capability
    Observe(input Evidence) ([]Observation, error)
}
```

The first implementation does not need to force every provider through one large Go interface. Domain-specific observation methods may remain strongly typed:

```go
type ActivityObserver interface {
    ObserveActivities(Evidence) ([]ActivityObservation, error)
}

type SleepObserver interface {
    ObserveSleep(Evidence) ([]SleepObservation, error)
}

type DailyMetricObserver interface {
    ObserveDailyMetrics(Evidence) ([]DailyMetricObservation, error)
}
```

The important rule is naming: adapter output is an observation, never a `Parsed*` canonical type.

## Core capability contracts

### Activity session

Produces a bounded activity interval with a sport/type, source identity, and optional summary metrics. Canonical targets are `tb_activities` and `tb_activity_observations`.

Required observation fields:

```text
provider, source_kind, source_key, content_hash
started_at, ended_at or duration, sport_type
```

Optional metrics retain units and provenance: distance, calories, heart rate, pace, elevation, and moving time.

### Route, sampling, and lap

These capabilities produce child measurements belonging to one activity observation:

```text
tb_activity_observation_routes
tb_activity_observation_samplings
tb_activity_observation_laps
```

Providers without a capability declare `unsupported`; they do not emit empty or fabricated rows. Missing per-lap distance or heart rate stays nullable unless the capability explicitly defines a
derivation.

### Sleep episode

Produces one provider report for a sleep interval, including stage segments and provider-specific summary metrics.

Canonical targets:

```text
tb_sleep_sessions
tb_sleep_observations
tb_sleep_observation_segments
```

Providers may disagree about boundaries or stages. The adapter preserves the report; matching and preferred-value selection happen later.

### Daily summary and metric

Daily summaries produce ring/goal facts such as move, exercise, and stand. Daily metrics produce scalars or reducer inputs such as steps, distance, resting heart rate, HRV, VO2 max, body mass, oxygen
saturation, and respiratory rate.

Canonical targets:

```text
tb_daily_summaries
tb_daily_summary_observations
tb_daily_metrics
tb_daily_metric_observations
```

Every metric declares reducer semantics such as `latest`, `average`, `minimum`, `maximum`, `interval_union`, or `source_priority`. Generic last-write-wins is not a valid default for health metrics.

### Media catalog identity

Produces provider records for a canonical media work/item, not user consumption history.

```text
tb_media_works
tb_media_items
tb_media_external_refs
```

AniList and Bangumi implement this capability independently while linking to one canonical work/item when identity matching succeeds.

### Media consumption event

Produces user events such as started, progressed, completed, abandoned, read, watched, reread, or rewatched. Provider list state may be converted into an event, but the original provider snapshot
remains evidence.

```text
tb_media_consumption_events
tb_media_progress
```

## Provider matrix

| Provider     | Activity    | Route       | Sampling    | Laps        | Sleep       | Daily summary | Daily metrics | Media identity | Consumption |
| ------------ | ----------- | ----------- | ----------- | ----------- | ----------- | ------------- | ------------- | -------------- | ----------- |
| Apple Health | implemented | implemented | implemented | implemented | implemented | implemented   | implemented   | -              | -           |
| GPX          | implemented | implemented | -           | -           | -           | -             | -             | -              | -           |
| Garmin       | planned     | planned     | planned     | planned     | planned     | planned       | planned       | -              | -           |
| AniList      | -           | -           | -           | -           | -           | -             | -             | planned        | planned     |
| Bangumi      | -           | -           | -           | -           | -           | -             | -             | planned        | planned     |
| Goodreads    | -           | -           | -           | -           | -           | -             | -             | deferred       | deferred    |
| WeRead       | -           | -           | -           | -           | -           | -             | -             | deferred       | deferred    |

“Implemented” means parser/adapter, persistence, and verification exist. A possible file format is “planned,” not implemented.

## Current Apple Health implementation

Apple Health currently implements these concrete observation paths:

```text
Apple export zip
  -> activity observations
  -> route observations
  -> sampling observations
  -> lap observations
  -> sleep observations
  -> daily summary observations
  -> daily metric observations
```

Its current code still uses `Parsed*` names and `tb_apple_source_items`; the canonical-layer refactor migrates those to the observation contracts described here. This document defines the target
contract, not a claim that the current implementation already satisfies every target table.

## Capability declaration requirements

Every new provider implementation documents:

1. supported capabilities;
2. source identity and stability guarantees;
3. content-hash inputs;
4. timezone and unit normalization;
5. snapshot versus incremental semantics;
6. unsupported fields and why they remain null;
7. reconciliation behavior for changed, missing, and duplicate observations;
8. a fixture and an end-to-end verification path.

## Avoiding provider-driven complexity

Provider support must not multiply canonical read complexity:

- adapter code owns provider parsing and normalization;
- observation tables own provenance and comparison data;
- canonical tables own selected, fast read projections;
- capability-specific services own matching and reducer rules;
- the web/API default path reads canonical projections;
- provider comparison is an explicit opt-in view.

Adding a provider should add an adapter, fixtures, capability registration, and any required observation columns. It should not add provider conditionals through every domain service or frontend
component.
