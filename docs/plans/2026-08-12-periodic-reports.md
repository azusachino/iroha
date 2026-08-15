# Iroha Monthly Report Plan v2

> **Media semantics amendment (2026-08-15):** report media sections now consume the provider-backed contracts in [ADR-0005](../adr/0005-media-provider-time-semantics.md). Exact sessions, dated
> provider updates, and day-level source facts are separate; the older nullable `event_at` wording below is historical and is superseded.

> Status: implementation complete for the v0.4 release candidate. This plan covers the monthly report across Iroha's existing personal data domains and its release evidence.

## Goal

Provide one stable report response for a selected calendar month, with typed sections for every personal domain Iroha currently covers:

- movement: canonical activities;
- sleep: nightly sessions and stage rollups;
- daily health: daily summaries and long-form metrics;
- media: media consumption events and current media state;
- expenses: the v0.4 expense ledger.

The report is a coordinated view, not a universal score. Distance, sleep duration, step counts, media items, and money have different units and must remain separate sections.

Operational data is not part of a personal report: raw files, import jobs, worker jobs, connector sync state, cache rows, and control-room tasks remain operational APIs.

Iroha does not push reports to Telegram or any other destination. A report is a read response generated when a client asks for it. Web and CLI are the v0.4 renderers.

## Current decisions

1. The monthly report has one stable API shape and one period resolution.
2. A month is a half-open local-date range `[from, to)`.
3. A month starts on the first calendar day and ends on the first day of the next month.
4. The API accepts an optional IANA timezone and returns the resolved range and timezone. If omitted, the server uses configured `IROHA_TIMEZONE` (default `Asia/Tokyo`). The web build uses
   `PUBLIC_IROHA_TIMEZONE` and sends it explicitly; the CLI inherits the configured value unless the operator passes an explicit override, so a report does not silently change between normal clients.
5. Each domain owns its aggregation semantics; the report service composes typed domain sections.
6. Missing data is represented as `empty` or omitted fields, never as zero measurements.
7. No cross-domain score, ranking, currency conversion, or invented correlation is produced.
8. Section state is only `available` or `empty`. Any domain query failure fails the whole request with `500`; v0.4 has no partial/unavailable section protocol.
9. The report has `generated_at` but no per-section freshness field. Ingestion freshness belongs to operational import/sync APIs.
10. The private web cockpit and CLI consume the same report API. No Telegram/Suzuran integration is part of this plan.
11. Public export remains a separate sanitized projection and does not automatically consume private reports.

## Period contract

### Request

```http
GET /api/v1/reports/monthly?month=2026-08&timezone=Asia/Tokyo
```

The `month` parameter is required and uses `YYYY-MM`. The server resolves it to the first day through the first day of the next month.

The response always returns the resolved values, for example:

```json
{
  "schema": "monthly-report.v1",
  "period": {
    "kind": "month",
    "month": "2026-08",
    "from": "2026-08-01",
    "to": "2026-09-01",
    "timezone": "Asia/Tokyo"
  },
  "generated_at": "2026-08-12T10:30:00Z",
  "sections": {}
}
```

The server validates the month and timezone. Clients must not calculate month boundaries independently and then assume they match the server.

The request is synchronous in v0.4. There is no report job, report table, report ID, or scheduled delivery owned by Iroha. A future digest scheduler may call this endpoint and deliver the result, but
delivery remains a client/application responsibility.

## Response shape

Each section has the same envelope:

```json
{
  "key": "sleep",
  "schema": "monthly-report.sleep.v1",
  "state": "available",
  "data": {}
}
```

`state` is one of `available` or `empty`. `data` is `null` when empty. Any domain query failure returns the existing top-level error envelope with `500`; the report does not return a partial success
or cache a failed section.

The report top-level response remains successful when a section is empty. The API does not return a flattened map of unrelated metric names.

### Concrete response type

The stable wire type is an object with named sections, not an array whose order clients must interpret:

```json
{
  "schema": "monthly-report.v1",
  "period": {
    "kind": "month",
    "month": "2026-08",
    "from": "2026-08-01",
    "to": "2026-09-01",
    "timezone": "Asia/Tokyo"
  },
  "generated_at": "2026-08-12T10:30:00Z",
  "sections": {
    "movement": {
      "schema": "monthly-report.movement.v1",
      "state": "available",
      "data": {
        "activity_count": 3,
        "distance_m": 28400,
        "distance_activity_count": 2,
        "duration_s": 9120,
        "by_sport": [{ "sport": "run", "activity_count": 2, "distance_m": 18000, "duration_s": 5400 }]
      }
    },
    "sleep": {
      "schema": "monthly-report.sleep.v1",
      "state": "empty",
      "data": null
    },
    "daily_health": {
      "schema": "monthly-report.daily-health.v1",
      "state": "available",
      "data": {
        "observed_days": 6,
        "metric_averages": [{ "metric": "steps", "value": 8420, "unit": "count", "observed_days": 6 }]
      }
    },
    "media": {
      "schema": "monthly-report.media.v1",
      "state": "empty",
      "data": null
    },
    "expenses": {
      "schema": "monthly-report.expenses.v1",
      "state": "available",
      "data": {
        "expense_count": 8,
        "totals_by_currency": [{ "currency": "JPY", "currency_exponent": 0, "amount_minor": 18400, "expense_count": 7 }],
        "by_category": [{ "category": "food", "currency": "JPY", "currency_exponent": 0, "amount_minor": 9200, "expense_count": 4 }]
      }
    }
  }
}
```

The corresponding Go shape should be explicit and typed rather than `map[string]any`:

```go
type MonthlyReport struct {
	Schema      string         `json:"schema"`
	Period      ReportMonth    `json:"period"`
	GeneratedAt time.Time      `json:"generated_at"`
	Sections    ReportSections `json:"sections"`
}

type ReportMonth struct {
	Kind     string `json:"kind"` // month
	Month    string `json:"month"`
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone"`
}

type ReportSections struct {
	Movement    ReportSection[MovementData]    `json:"movement"`
	Sleep       ReportSection[SleepData]       `json:"sleep"`
	DailyHealth ReportSection[DailyHealthData] `json:"daily_health"`
	Media       ReportSection[MediaData]       `json:"media"`
	Expenses    ReportSection[ExpensesData]    `json:"expenses"`
}
```

`ReportSection[T]` contains `Schema`, `State`, and `Data *T`. `Data` is `null` for `empty`; it is never a fabricated zero-valued object. The concrete section payloads are:

```go
type MovementData struct {
	ActivityCount          int                  `json:"activity_count"`
	DistanceM              float64              `json:"distance_m"`
	DistanceActivityCount  int                  `json:"distance_activity_count"`
	DurationS              int                  `json:"duration_s"`
	BySport                []MovementSportTotal `json:"by_sport"`
}

type SleepData struct {
	SessionCount      int                `json:"session_count"`
	MainSleepCount    int                `json:"main_sleep_count"`
	NapCount          int                `json:"nap_count"`
	AverageAsleepS    float64            `json:"average_asleep_s"`
	AverageTimeInBedS float64            `json:"average_time_in_bed_s"`
	AverageEfficiency float64            `json:"average_efficiency"`
	StageSeconds      SleepStageSeconds `json:"stage_seconds"`
}

type DailyHealthData struct {
	ObservedDays   int              `json:"observed_days"`
	MetricAverages []MetricAverage `json:"metric_averages"`
}

type MediaData struct {
	EventCount       int                `json:"event_count"`
	CompletedCount   int                `json:"completed_count"`
	RatedCount       int                `json:"rated_count"`
	AverageRating    *float64           `json:"average_rating,omitempty"`
	ByKind           []MediaKindTotal   `json:"by_kind"`
	CompletedItems   []MediaCompleted   `json:"completed_items"`
}

type ExpensesData struct {
	ExpenseCount     int                    `json:"expense_count"`
	TotalsByCurrency []ExpenseCurrencyTotal `json:"totals_by_currency"`
	ByCategory       []ExpenseCategoryTotal `json:"by_category"`
}

type MetricAverage struct {
	Metric        string  `json:"metric"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	ObservedDays  int     `json:"observed_days"`
}

type MovementSportTotal struct {
	Sport                 string  `json:"sport"`
	ActivityCount         int     `json:"activity_count"`
	DistanceM             float64 `json:"distance_m"`
	DistanceActivityCount int     `json:"distance_activity_count"`
	DurationS             int     `json:"duration_s"`
}

type SleepStageSeconds struct {
	Core        int `json:"core"`
	Deep        int `json:"deep"`
	Rem         int `json:"rem"`
	Awake       int `json:"awake"`
	Unspecified int `json:"unspecified"`
}

type MediaKindTotal struct {
	Kind          string `json:"kind"`
	EventCount    int    `json:"event_count"`
	CompletedCount int   `json:"completed_count"`
}

type MediaCompleted struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	MediaType  string    `json:"media_type"`
	CompletedAt time.Time `json:"completed_at"`
}

type ExpenseCurrencyTotal struct {
	Currency         string `json:"currency"`
	CurrencyExponent int    `json:"currency_exponent"`
	AmountMinor      int64  `json:"amount_minor"`
	ExpenseCount     int    `json:"expense_count"`
}

type ExpenseCategoryTotal struct {
	Category         string `json:"category"`
	Currency         string `json:"currency"`
	CurrencyExponent int    `json:"currency_exponent"`
	AmountMinor      int64  `json:"amount_minor"`
	ExpenseCount     int    `json:"expense_count"`
}
```

`MetricAverage` contains `metric`, `value`, `unit`, and `observed_days`; this is an array rather than a dynamic JSON map so clients can render units and sparse observations safely. The remaining small
value objects (`MovementSportTotal`, `SleepStageSeconds`, `MediaKindTotal`, `MediaCompleted`, `ExpenseCurrencyTotal`, and `ExpenseCategoryTotal`) are defined in the report package and mirrored in
OpenAPI.

All arrays use deterministic ordering: metric and sport/kind/category/currency keys sort ascending, and completed media items sort by completion time ascending then ID ascending. The expense API and
report responses include the static currency exponent so clients can render minor units without maintaining a second currency table.

## Section contracts

### Movement

Source: `tb_activities` and the existing activity summary service.

```json
{
  "activity_count": 3,
  "distance_m": 28400,
  "duration_s": 9120,
  "distance_activity_count": 2,
  "by_sport": [{ "sport": "run", "activity_count": 2, "distance_m": 18000, "distance_activity_count": 2, "duration_s": 5400 }]
}
```

Use activity `started_at` converted into the requested report timezone for inclusion and period bucketing. Do not pass the report's exclusive `to` value through existing inclusive public filters. Add
a period-aware activity adapter that reuses the existing swimming-distance correction while applying the report period explicitly; do not reuse the current session-timezone-dependent summary as-is. Do
not count route points as activities. Activities with unknown distance contribute to `activity_count` and duration but not `distance_m`; `distance_activity_count` states coverage.

### Sleep

Source: `tb_sleep_sessions` and the existing sleep aggregate service.

```json
{
  "session_count": 9,
  "main_sleep_count": 7,
  "nap_count": 2,
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

Use the existing `wake_date` semantics for period membership. Do not average averages from already paginated UI data; aggregate directly from canonical rows. `main_sleep_count`, averages, and stage
totals use only `is_main_sleep = true`; `nap_count` is reported separately. Preserve the distinction between main sleep and naps.

### Daily health

Source: `tb_daily_summaries` and `tb_daily_metrics`, through the daily aggregate service.

```json
{
  "observed_days": 6,
  "metric_averages": [
    { "metric": "steps", "value": 8420, "unit": "count", "observed_days": 6 },
    { "metric": "vo2max", "value": 42.3, "unit": "mL/kg/min", "observed_days": 2 }
  ]
}
```

Each metric average uses only days with that metric and unit. Missing observations are not zero. The adapter groups by `(metric, unit)` and never averages incompatible units under one metric name.
`observed_days` is the number of calendar days with at least one daily metric row; each entry has its own metric observation count.

### Media

Source: `tb_media_consumption_events` for period activity, with `tb_media_items`/`tb_media_progress` only for current-state fields.

```json
{
  "event_count": 12,
  "completed_count": 4,
  "rated_count": 3,
  "average_rating": 4.2,
  "by_kind": [{ "kind": "anime", "event_count": 8, "completed_count": 3 }],
  "completed_items": [{ "id": "med_01k...", "title": "...", "media_type": "anime", "completed_at": "2026-08-11T...Z" }]
}
```

The media aggregate service currently provides all-time/current-year-oriented aggregates. Add period-aware queries for this report rather than filtering a potentially truncated media list in the
client. `event_count` counts dated, non-snapshot consumption events with a non-null `event_at` in the selected month. `completed_count` counts distinct items whose resolved canonical completion date
falls in the month, resolved from `tb_media_progress.completed_on_value` only when `completed_on_precision = 'day'`, plus explicit completion events, and deduplicated by item. Rewatch events are
excluded from `completed_count` and are not a separate v0.4 bucket. Year/month fuzzy completions are excluded from monthly day-fact buckets; they are not assigned the sync timestamp.

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

## Existing feature compatibility fixes required in v0.4

The report is not allowed to paper over incompatible existing semantics. These are v0.4 fixes, not later cleanup:

- **Activity:** add a report-specific period query using the requested timezone and `[from, to)` bounds; preserve the existing swimming-distance correction; remove the misleading moving-time total and
  expose known-distance coverage.
- **Sleep:** keep existing wake-date membership, but calculate monthly averages and stage totals from main sleeps only and report naps separately.
- **Daily health:** add an exclusive-upper-bound report adapter and group averages by `(metric, unit)` so incompatible units cannot be combined.
- **Daily wire data:** stop serializing missing ring summaries as zero measurements; use nullable/nested ring data. Return `[]`, not `null`, for every repeated field.
- **Media persistence:** do not count provider list snapshots as consumption events; rebuild exact events with non-null `event_at`, persist provider state history with explicit time basis, preserve
  source day facts such as Goodreads `Date Read`, compare all semantic state fields during deduplication, and add provider-through-persistence tests. Bangumi-style status without a trusted effective
  date is excluded from date-scoped fact buckets. Give exact media events their own `medevt_` ID prefix before exposing them through the general CLI.
- **Runtime/API:** configure `IROHA_TIMEZONE` with a production timezone database, preserve the existing error envelope, register the monthly route in OpenAPI/route inventory, and allow browser
  `PUT`/`DELETE` preflights for the expense cockpit. Decode IDs separately from service errors: malformed IDs are `400`, missing rows are `404`, and database failures are `500`.
- **Shared wire contracts:** encode calendar dates as `YYYY-MM-DD` strings and aggregate periods as date/month strings; reserve RFC3339 for instants. Centralize pagination parsing so an explicitly
  invalid limit is `400` rather than silently becoming `50`.
- **OpenAPI gate:** replace generic `additionalProperties` aggregate schemas with exact schemas for expense/monthly-report and each CLI-enabled existing domain; validate OpenAPI 3.1 null unions
  instead of 3.0 `nullable`, compare the declared route/method set with Chi, and validate representative fixtures. Update or retire the stale API gap matrix in the same contract change.
- **Cache compatibility:** bump the shared cache-key contract prefix for v0.4 deployments. Expense and monthly-report routes remain uncached initially.
- **Web transport:** add typed PUT/DELETE helpers, structured error parsing, and a `204 No Content` path before implementing `/expenses`; do not use bounded Overview list sweeps for report values.
- **CLI contracts:** the general CLI initially promises only expense mutations and monthly reports. Existing-domain read wrappers follow only after their response schemas and period behavior are
  explicit.

Existing public list/aggregate filters are not silently changed by this work. The report adapters own the stricter half-open period contract and have dedicated boundary tests.

## Service and API implementation

Create `apps/iroha-server/pkg/reports` with:

- period parsing and month-boundary helpers;
- a report period containing `FromDate` (inclusive), `ToDateExclusive`, `FromInstant`, and `ToInstantExclusive`; every adapter uses the exclusive upper bound;
- a typed `Request` and `Response` contract;
- one section interface or explicit orchestrator calls for movement, sleep, daily, media, and expenses;
- deterministic serialization and stable schema names.

Create `apps/iroha-server/pkg/httpapi/reports.go` and register:

```text
GET /api/v1/reports/monthly
```

Update the active route inventory and OpenAPI. Reports are not cached in v0.4; the dataset is small and a fresh synchronous query avoids caching transient failures. Expense/media/import writes
therefore do not need report-cache invalidation.

The report service should issue aggregate queries directly. It must not call the HTTP layer, fetch all paginated rows, or reproduce UI calculations. A section query may return no rows and therefore
`empty`; SQL/query failures fail the request with the existing `{code,message,request_id}` error envelope.

HTTP behavior is explicit:

- `200 OK`: the report envelope is valid; sections are `available` or `empty`.
- `400 Bad Request`: malformed `month` or unknown IANA timezone.
- `500 Internal Server Error`: any domain query, assembly, or serialization failure. A failed section never becomes a partial `200` response.

The server loads timezone data from the image (`tzdata`) or the Go embedded timezone database and exposes `IROHA_TIMEZONE`, defaulting to `Asia/Tokyo`. An omitted `timezone` resolves to that
configured personal timezone on every period API, so the same month means the same thing on every surface. The web selects only a period in its UI but sends its configured build timezone; machine
clients may still send an explicit IANA timezone, and every response carries the resolved zone back.

The report API response is the only Iroha output. It is not posted to a Telegram chat, written to Valkey as a draft, or stored as a report artifact.

## Where the data comes from

Report generation is an orchestration read, not a second data pipeline:

```text
GET /api/v1/reports/monthly
  -> parse period and timezone
  -> query activity adapter      -> tb_activities
  -> query sleep adapter         -> tb_sleep_sessions
  -> query daily-health adapter  -> tb_daily_summaries + tb_daily_metrics
  -> query media adapter         -> tb_media_consumption_events + media state
  -> query expense adapter       -> tb_expenses
  -> assemble typed sections
  -> return JSON to the requesting client
```

The adapters use SQL/service methods against canonical tables. They do not read the web's currently loaded page, Telegram's local rows, raw import files, or job output. A domain with no rows returns
`empty`; an adapter/database error fails the report with `500`.

Report responses are not persisted. The canonical tables remain the source of truth, and the report can always be regenerated for the same period. Caching is an optimization only and must not become a
second report database.

## Client destinations and UX flows

### Web: primary full report

The private web application is the primary destination for the complete report because it can display all sections and period navigation without truncation.

```text
user opens /reports
  -> web chooses a month, not a timezone
  -> GET /api/v1/reports/monthly?month=...
  -> server resolves the configured personal timezone and echoes it in the envelope
  -> render period header and five independent section cards
  -> previous/next changes anchor and repeats the request
```

Add a private `/reports` route with a month selector and previous/next month navigation. The selected month loads the current report plus the server-owned twelve-month trend contract for comparison
charts. The series omits empty months from its plotted reports while preserving them in `empty_months`, and marks partial months explicitly. Render sections independently so an empty media section
does not hide valid sleep data. A request-level error is shown as an error state for the whole report. The route owns no aggregation logic; it renders typed section data, including loading, empty, and
error states.

The Overview may show a small link or selected-period summary, but it must not duplicate the report calculations. One `/reports` route is rendered through the six design-language shells. The report
data contract and analysis model remain shared; theme wrappers provide the visual treatment without six copies of report logic.

Future clients may call this endpoint and render the typed response, but no external client, Telegram workflow, or scheduled delivery is implemented in v0.4.

### CLI: machine-readable and operator view

Reports are one resource in the general `scripts/iroha_cli.py`; do not create a report-only CLI:

```bash
uv run python scripts/iroha_cli.py report monthly --month 2026-08
uv run python scripts/iroha_cli.py report monthly --month 2026-08 --format table
```

Default stdout is the unchanged report JSON for agent composition. Table output is a presentation convenience. The CLI does not recalculate or merge section values. Iroha's configured timezone is
authoritative; `--timezone` is an explicit override only.

Client file boundary:

- Iroha: `apps/iroha-server/pkg/reports/`, `apps/iroha-server/pkg/httpapi/reports.go`, `docs/contracts/openapi.yaml`, and `apps/iroha-server/pkg/httpapi/api_contract_test.go`.
- Web: add `getMonthlyReport()` and the report types in `apps/iroha-web/src/lib/api.ts`; add `/reports/+page.svelte` and shared section components as needed.
- CLI: add the `report monthly` resource to `scripts/iroha_cli.py` as a transport/presentation wrapper only.

## Implementation slices

1. **Existing contract repair:** exact OpenAPI schemas and validation, calendar-date encoding, error classification, pagination limits, empty-array normalization, cache-key versioning, media-event
   IDs, and provider persistence tests.
2. **Month and period contract:** report package, timezone data/configuration, half-open boundary types, response envelope, and route inventory.
3. **Existing domain adapters:** movement, sleep, and daily typed aggregate queries with direct SQL/service tests and compatibility fixtures.
4. **Media period aggregate:** dated event/completion semantics, snapshot exclusion, provider mappings, and period-aware media queries.
5. **Expense section:** add expense totals/category buckets and report integration.
6. **API:** register endpoint, top-level failure behavior, uncached execution, and integration tests.
7. **Web and CLI:** one shared `/reports` page, typed section components, expense mutation transport, `/expenses`, and only the initially supported CLI resources.
8. **Release hardening:** cross-domain fixtures, performance checks, docs, and public-export boundary tests.

## Verification matrix

- Month boundaries resolve to the first day of the requested month and the first day of the next month in the requested timezone.
- Month boundaries handle leap years and year changes.
- The same anchor/timezone produces byte-stable section data apart from `generated_at`.
- Activity inclusion uses started-at conversion; sleep uses wake-date semantics; daily data uses date; media uses event time; expenses use occurred date.
- Missing observations are omitted/empty rather than zeroed.
- Main-sleep averages exclude naps; metric-only daily days do not fabricate ring zeros; repeated fields serialize as `[]`.
- JPY and USD totals remain separate.
- Deleted expenses are excluded.
- Any domain query failure produces a top-level `500` and is not cached; an empty domain produces an `empty` section with `data: null`.
- Media provider fixtures prove snapshots are not consumption events, unknown dates are not assigned sync time, dated provider updates/source day facts are represented separately, exact events are
  deduplicated, and undated state observations are excluded from date-scoped fact buckets.
- Date-only wire fields, invalid IDs, database failures, invalid pagination limits, exact OpenAPI schemas, null unions, route/method parity, and cache-key versioning are tested.
- Fresh report requests always query canonical data; web and CLI consume the API response without local aggregation.
- `make check`, integration tests with representative fixtures, OpenAPI validation, and web tests pass.
