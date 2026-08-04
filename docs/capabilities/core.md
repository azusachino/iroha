# Iroha core capabilities

This is the provider-independent contract for the parts of Iroha that are stable enough to expose publicly. Provider documents explain how Apple Health, Garmin, AniList, and other sources implement
these capabilities.

## Evidence and intake

Iroha accepts immutable evidence from files, connector snapshots, and manual payloads. The original evidence remains reprocessable and is never replaced by a normalized result.

```text
evidence -> durable job -> provider observation -> canonical projection
```

## Canonical domain capabilities

### Activity history

Iroha can represent an activity session with:

- sport/type and title;
- start/end time and timezone;
- distance, duration, pace, calories, heart rate, and elevation where known;
- route observations;
- timestamped sampling streams;
- lap/segment observations;
- one or more provider observations for comparison.

### Sleep history

Iroha can represent a sleep episode with:

- sleep interval and wake date;
- in-bed and asleep duration;
- sleep-stage segments;
- efficiency and provider summary metrics;
- multiple device observations for one episode;
- selected and comparison projections.

### Daily health facts

Iroha can represent day-level summaries and metrics such as:

- move, exercise, and stand goals;
- steps, distance, and flights;
- resting and walking heart rate;
- HRV, VO2 max, body mass;
- oxygen saturation and respiratory rate.

Each metric has explicit reducer and source-selection semantics. Missing data is distinct from zero.

### Media identity and history

Iroha can represent:

- canonical media works and concrete items;
- titles, aliases, relations, and external provider references;
- consumption events such as started, progressed, completed, abandoned, read, watched, reread, and rewatched;
- current progress as a projection of event history;
- provider comparison and unresolved identity matches.

## Cross-cutting guarantees

- Provider observations retain source identity and raw evidence references.
- Re-importing unchanged evidence is idempotent.
- Parser or adapter upgrades can reprocess evidence without duplicate canonical records.
- Canonical list reads use selected projections and do not require provenance joins.
- Provider disagreement is preserved and visible in comparison views.
- Public views are sanitized projections, never direct raw or private reads.
- Durable jobs survive API and worker restarts.

## Public versus private surface

The core capability contract is public documentation. The default data is not public. Private canonical APIs expose the user's full records; the static public export exposes only explicitly sanitized projections.

Provider-specific quirks, source keys, parser limitations, and support status belong in the provider documents, not in this contract.
