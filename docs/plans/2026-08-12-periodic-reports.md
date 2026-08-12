# Iroha Periodic Reports Plan

> Status: draft for review. This plan covers weekly and monthly reports across Iroha's existing personal data domains.

## Goal

Provide one stable report response for a selected week or month, with typed sections for every personal domain Iroha currently covers:

- movement: canonical activities;
- sleep: nightly sessions and stage rollups;
- daily health: daily summaries and long-form metrics;
- media: media consumption events and current media state;
- expenses: the v0.4 expense ledger.

The report is a coordinated view, not a universal score. Distance, sleep duration, step counts, media items, and money have different units and must remain separate sections.

Operational data is not part of a personal report: raw files, import jobs, worker jobs, connector sync state, cache rows, and control-room tasks remain operational APIs.

## Current decisions

1. Weekly and monthly reports share one API shape and differ only in period resolution.
2. A period is a half-open local-date range `[from, to)`.
3. Weeks start Monday. Months start on the first calendar day.
4. The API receives an IANA timezone and returns the resolved range and timezone.
5. Each domain owns its aggregation semantics; the report service composes typed domain sections.
6. Missing data is represented as `empty` or omitted fields, never as zero measurements.
7. No cross-domain score, ranking, currency conversion, or invented correlation is produced.
8. Report sections include freshness/provenance metadata so imported, derived, and absent data are distinguishable.
9. The private web cockpit and Suzuran consume the same report API. Telegram renders a compact subset.
10. Public export remains a separate sanitized projection and does not automatically consume private reports.

## Period contract

### Request

```http
GET /api/v1/reports/periodic?period=week&anchor=2026-08-12&timezone=Asia/Tokyo
```

Supported values:

- `period=week`: Monday through the following Monday, containing `anchor`.
- `period=month`: first day through the first day of the next month, containing `anchor`.

The response always returns the resolved values, for example:

```json
{
  "period": "week",
  "anchor": "2026-08-12",
  "from": "2026-08-10",
  "to": "2026-08-17",
  "timezone": "Asia/Tokyo",
  "schema": "periodic-report.v1",
  "generated_at": "2026-08-12T10:30:00Z",
  "sections": {}
}
```

The server validates the timezone and period. Clients must not calculate week/month boundaries independently and then assume they match the server.

## Response shape

Each section has the same envelope:

```json
{
  "key": "sleep",
  "schema": "periodic-report.sleep.v1",
  "state": "available",
  "freshness": {
    "source": "canonical",
    "as_of": "2026-08-12T09:00:00Z"
  },
  "data": {}
}
```

`state` is one of `available`, `empty`, or `unavailable`. `unavailable` is reserved for a domain query failure and must include a safe error code; one failed section must not hide healthy sections.

The report top-level response remains successful when a section is empty. The API does not return a flattened map of unrelated metric names.

## Section contracts

### Movement

Source: `tb_activities` and the existing activity summary service.

```json
{
  "activity_count": 3,
  "distance_m": 28400,
  "duration_s": 9120,
  "moving_time_s": 8640,
  "by_sport": [{ "sport": "run", "activity_count": 2, "distance_m": 18000, "duration_s": 5400 }]
}
```

Use activity `started_at` converted into the requested report timezone for inclusion and period bucketing. Preserve the existing activity summary semantics and do not count route points as activities.

### Sleep

Source: `tb_sleep_sessions` and the existing sleep aggregate service.

```json
{
  "session_count": 7,
  "main_sleep_count": 7,
  "average_asleep_s": 24120,
  "average_time_in_bed_s": 25920,
  "average_efficiency": 0.93,
  "stage_seconds": {
    "core": 11200,
    "deep": 4200,
    "rem": 5600,
    "awake": 1100,
    "unspecified": 3020
  }
}
```

Use the existing `wake_date` semantics for period membership. Do not average averages from already paginated UI data; aggregate directly from canonical rows. Preserve the distinction between main
sleep and naps.

### Daily health

Source: `tb_daily_summaries` and `tb_daily_metrics`, through the daily aggregate service.

```json
{
  "observed_days": 6,
  "metric_averages": {
    "move_kcal": 612.4,
    "exercise_min": 42.0,
    "stand_hours": 10.5,
    "steps": 8420,
    "distance_km": 6.1,
    "resting_hr": 57.2,
    "hrv_sdnn": 48.1,
    "spo2_avg": 97.4,
    "respiratory_rate": 15.8,
    "vo2max": 42.3,
    "body_mass_kg": 67.8
  },
  "metric_observation_days": {
    "steps": 6,
    "vo2max": 2
  }
}
```

Each metric average uses only days with that metric. Missing observations are not zero. The report should expose observation counts so a sparse metric is not presented as equally complete.

### Media

Source: `tb_media_consumption_events` for period activity, with `tb_media_items`/`tb_media_progress` only for current-state fields.

```json
{
  "event_count": 12,
  "completed_count": 4,
  "rated_count": 3,
  "average_rating": 4.2,
  "by_kind": [{ "kind": "anime", "event_count": 8, "completed_count": 3 }],
  "completed_items": [{ "id": "med_01k...", "title": "...", "media_type": "anime", "occurred_at": "2026-08-11T...Z" }]
}
```

The media aggregate service currently provides all-time/current-year-oriented aggregates. Add period-aware queries for this report rather than filtering a potentially truncated media list in the
client. Define whether `completed_count` is event-based or item-state-based before implementation; the recommended v1 semantics are completion events in the selected period.

### Expenses

Source: active `tb_expenses` rows from the Expense Ledger plan.

```json
{
  "expense_count": 8,
  "totals_by_currency": [{ "currency": "JPY", "amount_minor": 18400, "expense_count": 7 }],
  "by_category": [{ "category": "food", "currency": "JPY", "amount_minor": 9200, "expense_count": 4 }]
}
```

Deleted rows are excluded. Currencies remain separate. The report never adds JPY and USD together or formats minor units without currency metadata.

## Service and API implementation

Create `apps/iroha-server/pkg/reports` with:

- period parsing and Monday/month boundary helpers;
- a typed `Request` and `Response` contract;
- one section interface or explicit orchestrator calls for movement, sleep, daily, media, and expenses;
- section-level state/error handling;
- deterministic serialization and stable schema names.

Create `apps/iroha-server/pkg/httpapi/reports.go` and register:

```text
GET /api/v1/reports/periodic
```

Update the active route inventory, OpenAPI, read-cache mapping, and cache invalidation. Add a `reports` cache namespace. Imports and expense writes invalidate reports; media sync/import reconciliation
invalidates the affected report namespace as well as its domain namespace.

The report service should issue aggregate queries directly. It must not call the HTTP layer, fetch all paginated rows, or reproduce UI calculations. A section query may return no rows and therefore
`empty`; SQL/query failures produce `unavailable` with a safe code.

## Web and Telegram representation

### Web

Add a private `/reports` route with a week/month switcher and previous/next period navigation. One API request loads the report. Render sections independently so a missing media sync does not blank a
valid sleep report.

The Overview may show a small link or selected-period summary, but it must not duplicate the full report calculations. Every curated theme receives the same typed report data and handles `available`,
`empty`, and `unavailable` states.

### Telegram

Telegram remains a thin report client:

- `/report week` requests the current weekly report.
- `/report month` requests the current monthly report.
- `/report week 2026-08-10` and `/report month 2026-08` may request an explicit period.
- The response shows period/timezone, expense totals by currency, movement totals, sleep averages, daily-health highlights, and media completions.
- A `More` button can link to the private web report; Telegram must not calculate or merge sections itself.

Keep the Telegram message compact and omit empty sections. The bot should label partial reports when a section is unavailable.

## Implementation slices

1. **Period and contract:** report package, boundary tests, response envelope, OpenAPI, route inventory.
2. **Existing domain adapters:** movement, sleep, and daily typed aggregate queries with direct SQL/service tests.
3. **Media period aggregate:** define completion/event semantics and implement period-aware media queries.
4. **Expense section:** add expense totals/category buckets and cache invalidation.
5. **API/cache:** register endpoint, section failures, cache namespace, invalidation, and integration tests.
6. **Web report:** `/reports`, week/month navigation, typed section components, all-theme wiring.
7. **Telegram report:** `/report` commands, compact renderer, partial/empty handling.
8. **Release hardening:** cross-domain fixture, freshness copy, performance checks, docs, and public-export boundary test.

## Verification matrix

- Week boundaries always resolve Monday-to-Monday in the requested timezone.
- Month boundaries handle leap years and year changes.
- The same anchor/timezone produces byte-stable section data apart from `generated_at`.
- Activity inclusion uses started-at conversion; sleep uses wake-date semantics; daily data uses date; media uses event time; expenses use occurred date.
- Missing observations are omitted/empty rather than zeroed.
- JPY and USD totals remain separate.
- Deleted expenses are excluded.
- One unavailable domain produces a partial report while healthy sections remain available.
- Media completion semantics are tested against event fixtures and current progress fixtures.
- Cache hits return the same response; import/media/expense changes invalidate report data.
- Web and Telegram consume the API response without local aggregation.
- `make check`, integration tests with representative fixtures, OpenAPI validation, and web tests pass.
