# Iroha data cockpit implementation plan

## Status

Draft for Gate A approval. This plan is derived from the OWID/Iroha research, the six-theme orientation decision, and the current Iroha v0.4 worktree.

Research commits already landed:

- d990209 — world-in-data cockpit research
- 510f3bb — six-theme analytical orientations

No implementation starts until Gate A is approved. This document defines exact boundaries, contracts, files, tests, gates, and stop conditions.

## 1. Outcome

Iroha becomes a personal data cockpit that can grow by adding metrics without:

- adding a new global top tab for every metric;
- changing the canonical database into a generic EAV store;
- making the browser aggregate raw domain records;
- weakening the stable expense API;
- collapsing six design languages into one neutral cockpit;
- introducing hidden Telegram/agent workflow coupling;
- inventing weekly reports.

Target data path:

```
raw evidence / direct API input
        -> provider observations or canonical domain record
        -> domain-owned deterministic adapter
        -> metric definition + series response
        -> chart / table / record / provenance representation
```

Target navigation path:

```
global shell: Today | Overview | Domains | Analyze | More | Search
        -> domain-local navigation
        -> metric shelf and chart-local views
```

## 2. Non-negotiable boundaries

### Database

Keep the existing typed canonical tables:

- activities, routes, samplings, laps;
- sleep sessions, segments, and observations;
- daily summaries, daily metrics, and observations;
- media identity, progress, and event tables;
- expenses;
- raw files, imports, jobs, and provider observations.

Do not add a generic canonical metric-values/EAV table in the first implementation. Do not add a metric-definition table yet. Definitions are application-owned metadata until runtime editing becomes a
demonstrated requirement.

If measured performance later requires a read model, it is derived data, never canonical truth, and needs its own freshness/version contract.

### API

Existing typed domain endpoints remain the source for records and details. The metric API is additive and read-only. It returns server-computed series and metadata; it does not expose SQL, table
names, provider payloads, or theme CSS.

Expense writes remain:

```
local agent or CLI -> POST /api/v1/expenses -> tb_expenses
```

Telegram is not part of this sequence. It neither owns expense storage nor becomes a required OCR/agent layer.

### Frontend

Routes load stable view models. Shared primitives handle behavior. Theme components decide composition:

```
+page.svelte controller
  -> API client / URL state / loading state
  -> page view model
  -> ThemeRouteRenderer
  -> theme-owned page composition
  -> shared charts, tables, date controls, metadata
```

No theme may change metric meaning, API semantics, privacy behavior, missingness, or error state. A shared CockpitFrame cannot remain the final renderer for expenses or reports.

### Reports

Only monthly reports are in scope. The existing /api/v1/reports/monthly contract remains compatible through the v0.4 release gate. A future extensible report envelope is additive and cannot break
current CLI/web consumers.

## 3. Current baseline

Before coding, record:

- branch: docs/iroha-v0.4-expense-ledger-plan;
- research commits: d990209 and 510f3bb;
- existing v0.4 expense, monthly-report, theme, shared-library, and visual/read-only commits;
- existing dirty files, which must not be overwritten or staged with this work:
  - apps/iroha-web/src/lib/components/BarChart.svelte
  - apps/iroha-web/src/routes/expenses/+page.svelte
  - apps/iroha-web/src/routes/reports/+page.svelte

Before implementation:

1. inspect and checkpoint the dirty changes separately;
2. create a descriptive implementation branch from the agreed baseline;
3. run make check and record the result;
4. only then begin Phase 1.

## 4. Gate A — architecture approval

Approve these decisions explicitly:

1. typed canonical database remains authoritative;
2. metric catalog is code-owned first;
3. metric series API is additive under /api/v1/metrics;
4. no database migration is needed for the vertical slice;
5. current monthly-report contract remains compatible;
6. six themes are analytical lenses;
7. global navigation becomes grouped/domain-based;
8. monthly-only reporting remains in scope;
9. expense intake remains independent from Telegram and local OCR agents;
10. deterministic fixtures, API contract checks, and browser visual checks gate each phase.

If any decision changes, update this plan before coding.

## 5. Phase 1 — metric inventory and semantic contract

### Objective

Make every displayed number traceable to a stable metric identity, canonical source, unit, missingness rule, and aggregation method.

### Files

Create:

- apps/iroha-server/pkg/metrics/definition.go
- apps/iroha-server/pkg/metrics/registry.go
- apps/iroha-server/pkg/metrics/registry_test.go
- apps/iroha-server/pkg/metrics/inventory_test.go
- apps/iroha-server/pkg/metricseries/service.go
- apps/iroha-server/pkg/metricseries/service_test.go
- docs/contracts/metric-catalog.md
- docs/contracts/examples/metric-catalog.json

Audit as needed:

- apps/iroha-server/pkg/daily/service.go
- apps/iroha-server/pkg/reports/*.go
- apps/iroha-runtime/models/models.go
- docs/provider-capabilities.md
- docs/contracts/api-v1-decisions.md

### Definition semantics

Use a small immutable application type:

```go
type Definition struct {
    ID                 string
    Domain             string
    Label              string
    Description        string
    Kind               string // canonical or derived
    ValueType          string // number, count, duration, money, percentage
    Unit               string
    ShortUnit          string
    SupportedGrains    []string // day, month, year
    Dimensions         []Dimension
    Reducer            string
    Rollup             string // sum, average, or count across requested periods
    AggregationVersion string
    CoverageKind       string
    SemanticColorToken string
    PreferredView      string // line, bar, stacked-bar, heatmap, table
}
```

SemanticColorToken is a stable meaning such as health, movement, sleep, media, or expense.food. It is not a CSS value; each theme maps the token to its own treatment.

The package boundary is deliberate:

- pkg/metrics contains only definitions, registry, dimensions, and series DTOs; it imports no domain service;
- pkg/metricseries owns orchestration and resolver adapters; it imports pkg/metrics plus activities, daily, sleep, media, and expenses;
- domain packages expose their existing typed period/aggregate results and never import metricseries;
- httpapi depends on metricseries through a server dependency.

This prevents a Go import cycle while keeping aggregation semantics in the domain services.

### Initial catalog

Register these first; do not expose every database scalar automatically.

| ID                      | Domain   | Unit                | Default grain | Dimensions         | Calculation                           |
| ----------------------- | -------- | ------------------- | ------------- | ------------------ | ------------------------------------- |
| health.steps            | daily    | count               | day           | none               | source-selected daily metric          |
| health.distance_km      | daily    | km                  | day           | none               | source-selected daily metric          |
| health.flights          | daily    | count               | day           | none               | source-selected daily metric          |
| health.resting_hr       | daily    | bpm                 | day           | none               | source-selected daily metric          |
| health.hrv_sdnn         | daily    | ms                  | day           | none               | source-selected daily metric          |
| health.move_kcal        | daily    | kcal                | day           | none               | canonical daily summary               |
| health.exercise_min     | daily    | min                 | day           | none               | canonical daily summary               |
| health.stand_hours      | daily    | h                   | day           | none               | canonical daily summary               |
| movement.activity_count | movement | count               | month         | sport              | canonical activity count              |
| movement.distance_m     | movement | m                   | month         | sport              | sum non-null activity distance        |
| movement.duration_s     | movement | s                   | month         | sport              | sum non-null activity duration        |
| sleep.asleep_s          | sleep    | s                   | month         | sleep_kind         | average main-sleep duration           |
| sleep.efficiency        | sleep    | %                   | month         | sleep_kind         | average main-sleep efficiency         |
| media.completed_count   | media    | count               | month         | media_kind         | completed event count                 |
| expenses.amount_minor   | expenses | minor currency unit | month         | currency, category | active-expense sum without conversion |
| expenses.count          | expenses | count               | month         | currency, category | active-expense count                  |

Unknown tb_daily_metrics.metric values remain stored and importable. They are not catalog-visible until registered and documented.

### Acceptance

- duplicate IDs fail tests;
- every definition has unit, grain, reducer, aggregation version, and coverage semantics;
- every existing report scalar is mapped or explicitly domain-only;
- money metrics require currency semantics;
- registry order is deterministic;
- no migration is introduced.

## 6. Phase 2 — period and series core

### Period contract

Reuse apps/iroha-server/pkg/reports/period.go rather than creating another timezone parser. Extract a shared pkg/periods package only if reuse is impossible, and migrate reports in the same change.

Rules:

- from inclusive, to exclusive;
- grain is day, month, or year only when permitted;
- timezone is a validated IANA location;
- date tables use calendar dates in the requested timezone;
- instant tables use timezone-aware half-open instants;
- returned periods are ascending and complete;
- absent values are null, never zero;
- wire labels are YYYY-MM-DD, YYYY-MM, or YYYY.

### Files

Create:

- apps/iroha-server/pkg/metrics/period.go
- apps/iroha-server/pkg/metrics/series.go
- apps/iroha-server/pkg/metrics/series_test.go
- apps/iroha-server/pkg/metrics/dimensions.go
- apps/iroha-server/pkg/metrics/dimensions_test.go
- apps/iroha-server/pkg/metricseries/daily.go
- apps/iroha-server/pkg/metricseries/activities.go
- apps/iroha-server/pkg/metricseries/sleep.go
- apps/iroha-server/pkg/metricseries/media.go
- apps/iroha-server/pkg/metricseries/expenses.go

Internal shape:

```go
type SeriesRequest struct {
    MetricID   string
    From       time.Time
    To         time.Time
    Grain      string
    Timezone   *time.Location
    Dimensions map[string][]string
}

type Point struct {
    Period       string
    Value        *float64
    ValueMinor   *int64
    ObservedDays int
}

type Series struct {
    MetricID   string
    Grain      string
    From       string
    To         string
    Timezone   string
    Dimensions map[string]string
    Points     []Point
    Coverage   Coverage
    Source     Source
}
```

For every point exactly one of Value or ValueMinor is populated. Value is used for ordinary numeric/count/duration/percentage metrics. ValueMinor is used for money metrics and preserves the integer
minor-unit amount. The wire schema uses value for non-money points and value_minor for money points. A point with both or neither is invalid. Expense records remain authoritative for full int64
precision and currency exponent.

Define the supporting types in pkg/metrics rather than leaving them implicit:

```go
type Dimension struct {
    ID     string
    Label  string
    Values []string
}

type Coverage struct {
    ExpectedPeriods int
    ObservedPeriods int
}

type Source struct {
    Kind        string
    Method      string
    SourceKinds []string
}
```

Resolver adapters are implemented in pkg/metricseries:

- daily resolver for daily metrics and summaries;
- activities resolver for count/distance/duration;
- sleep resolver for sessions;
- media resolver for completions;
- expenses resolver for currency/category.

Each resolver must query canonical selected records, validate dimensions, return complete buckets, return coverage, identify method/version, and have pure plus integration tests.

Money must not be converted. If JSON number precision becomes a concern, add an explicit minor-unit field before changing semantics.

### Acceptance

- exact fixtures produce deterministic JSON;
- invalid ID/grain/timezone/dimension/range returns 400/404 with common errors;
- empty ranges are valid empty series;
- missing day differs from measured zero;
- currencies never combine;
- provider payloads do not appear;
- browser does not fetch raw rows to draw the series.

## 7. Phase 3 — metric API

### Routes

Add to apps/iroha-server/pkg/httpapi/server.go:

```text
GET /api/v1/metrics
GET /api/v1/metrics/{metricId}
GET /api/v1/metrics/{metricId}/series
```

Create:

- apps/iroha-server/pkg/httpapi/metrics.go
- apps/iroha-server/pkg/httpapi/metrics_test.go
- apps/iroha-server/pkg/httpapi/metrics_integration_test.go

Add MetricSeriesService *metricseries.Service to httpapi.Dependencies and construct it in the existing server composition root from the metric registry plus the existing domain services. Handlers do
not instantiate services or databases. A missing service returns the existing service-unavailable contract, not a panic.

### Catalog response

```json
{
  "schema": "metric-catalog.v1",
  "metrics": [
    {
      "id": "health.steps",
      "domain": "daily",
      "label": "Steps",
      "description": "Source-selected daily step count.",
      "kind": "canonical",
      "value_type": "count",
      "unit": "count",
      "short_unit": "steps",
      "supported_grains": ["day", "month", "year"],
      "dimensions": [],
      "reducer": "source_priority",
      "aggregation_version": "health.steps.v1",
      "coverage_kind": "observed_days",
      "semantic_color_token": "health",
      "preferred_view": "line"
    }
  ]
}
```

### Series request

```text
GET /api/v1/metrics/expenses.amount_minor/series?
  from=2026-01-01&
  to=2027-01-01&
  grain=month&
  timezone=Asia%2FTokyo&
  dimension=currency%3AJPY&
  dimension=category%3Afood
```

dimension is repeatable as name:value. Names and values are validated against the definition. Omitting a dimension returns matching dimension series. The initial maximum is 32 combinations; exceeding
it returns 400 with too_many_series.

### Series response

```json
{
  "schema": "metric-series.v1",
  "metric_id": "expenses.amount_minor",
  "label": "Expenses",
  "unit": "minor currency unit",
  "value_type": "money",
  "period": {
    "grain": "month",
    "from": "2026-01-01",
    "to": "2027-01-01",
    "timezone": "Asia/Tokyo"
  },
  "series": [
    {
      "dimensions": { "currency": "JPY" },
      "points": [
        { "period": "2026-01", "value_minor": 12000, "observed_days": 8 },
        { "period": "2026-02", "value_minor": null, "observed_days": 0 }
      ],
      "coverage": { "expected_periods": 12, "observed_periods": 1 },
      "source": {
        "kind": "canonical",
        "method": "expenses.sum_active.v1",
        "source_kinds": ["manual", "local_agent"]
      }
    }
  ]
}
```

Do not return CSS colors, ECharts options, SQL table names, raw IDs, or provider payloads.

OpenAPI must model numeric precision explicitly: a non-money point has value (number or null) and no value_minor; a money point has value_minor (integer or null) and no value. Both forms require
period and observed_days. Do not describe both fields as optional without a mutual-exclusion rule.

### Contract files

Update:

- docs/contracts/openapi.yaml;
- apps/iroha-web/src/lib/api.ts;
- apps/iroha-web/src/lib/api.test.ts;
- scripts/iroha_cli.py after the API is stable, adding read-only metric list/series commands.

CLI JSON is lossless default. Table formatting does not reinterpret values.

### Acceptance

- make contract-check passes;
- OpenAPI examples parse;
- errors use code/message/request_id;
- unknown response fields remain ignorable;
- CLI forwards values without aggregation or reinterpretation.

## 8. Phase 4 — monthly report integration

Keep /api/v1/reports/monthly and monthly-report.v1 through the v0.4 release gate.

Refactor internals only where semantics match:

- reports/daily_health.go may use daily metric resolvers;
- reports/expenses.go may use the expense resolver;
- reports/monthly.go remains typed orchestration;
- existing report tests prove semantic compatibility.

Do not convert rich data such as completed_items or by_sport into generic metrics.

Do not implement report v2 yet. Trigger a separately versioned report envelope only when a new section would otherwise require editing fixed ReportSections, more than two independently owned metric
sections are needed, or a consumer requires catalog-driven sections.

Acceptance:

- existing monthly integration tests remain green;
- totals match direct services;
- empty sections remain empty, not zero;
- generated_at is the only documented non-deterministic field;
- CLI output is unchanged;
- no weekly endpoint, UI, CLI command, or scheduler is added.

## 9. Phase 5 — shared representation primitives

### Ownership

Use packages/iroha-shared for behavior independent of private routing:

- MonthNavigator.svelte;
- month.ts parsing/shifting/formatting;
- pure metric-series view-state helpers;
- semantic formatters.

Keep fetching, theme composition, and private route assumptions in iroha-web.

### Components

Create or extend:

- apps/iroha-web/src/lib/components/MetricChart.svelte;
- apps/iroha-web/src/lib/components/MetricTable.svelte;
- apps/iroha-web/src/lib/components/MetricMetadata.svelte;
- apps/iroha-web/src/lib/components/MetricViewTabs.svelte;
- apps/iroha-web/src/lib/components/MetricEmptyState.svelte;
- apps/iroha-web/src/lib/components/MetricDownload.svelte.

Reuse LineChart.svelte, BarChart.svelte, Heatmap.svelte, and existing formatters. Do not create a second ECharts system.

Chart rules:

- line for continuous change;
- bar for category/ranking comparison;
- stacked bar for same-unit composition;
- heatmap for recurring calendar/coverage intensity;
- maps/routes only where spatial records exist;
- units on labels and tooltips;
- missing periods remain visible as gaps/nulls;
- color is paired with labels/order/shape/dash/direct annotation;
- every chart has an accessible table or text summary;
- metadata shows unit, method, coverage, source, and record relationship.

Fix the known BarChart datum-color override before depending on it. Add tests for categorical colors, null points, and all six semantic token mappings. Enable ECharts ARIA in the shared wrapper.

### Month control

Make packages/iroha-shared/src/MonthNavigator.svelte canonical. It must support:

- previous/next buttons;
- ArrowLeft/ArrowRight while focus is in the navigator and no text input is active;
- direct year selection;
- month selection within the chosen year;
- Escape to close;
- loading/disabled state;
- URL-safe YYYY-MM;
- no weekly mode.

Expenses and reports use the same value/control contract.

Acceptance:

- chart/table points and dimensions match;
- null is not rendered as zero;
- colors are stable and accessible;
- metadata is visible;
- downloads contain displayed values plus period/metric metadata;
- keyboard, browser back, and direct year/month selection work;
- make web-check web-fmt-check web-test passes.

## 10. Phase 6 — expense cockpit

Refactor apps/iroha-web/src/routes/expenses/+page.svelte into a controller containing URL state, API calls, loading/error mapping, delete, and a stable ExpensePageModel. It must not calculate chart
totals from the paginated list.

Requests:

1. metric series for chart/aggregation;
2. expense list for exact ledger;
3. metric metadata for labels, units, and method disclosure.

Multiple currencies remain separate; no conversion.

Create distinct components:

- apps/iroha-web/src/lib/themes/atlas/Expenses.svelte
- apps/iroha-web/src/lib/themes/grapher/Expenses.svelte
- apps/iroha-web/src/lib/themes/field-journal/Expenses.svelte
- apps/iroha-web/src/lib/themes/phenology/Expenses.svelte
- apps/iroha-web/src/lib/themes/sound-map/Expenses.svelte
- apps/iroha-web/src/lib/themes/archive/Expenses.svelte

Orientation:

| Theme         | Leading representation                                                     |
| ------------- | -------------------------------------------------------------------------- |
| Atlas         | chronological spending survey; geography only when expense location exists |
| Grapher       | monthly total trend, previous-month delta, category comparison             |
| Field Journal | dated entries, notes, and continuity across the month                      |
| Phenology     | recurring category rhythm across calendar days                             |
| Sound Map     | spending intensity and bursts, without pretending it is audio              |
| Archive       | exact ledger, source identity, and immutable record detail                 |

All compositions lead with meaningful visual analysis, then exact records/details. Editing remains a non-goal; delete is the only existing mutation.

Acceptance:

- registry points to six distinct components;
- hierarchy, first visual, period framing, and detail order differ;
- filters update URL and request immediately;
- no Apply button;
- category/currency colors are semantic and theme-specific;
- delete refreshes ledger and chart;
- screenshots pass for six themes in light and dark.

## 11. Phase 7 — monthly report cockpit

Refactor apps/iroha-web/src/routes/reports/+page.svelte into a controller owning month URL state, report request, stable MonthlyReportPageModel, and section states. The client selects only a month;
the server resolves the configured personal timezone and returns it in the envelope for semantic transparency. There is no web timezone control.

Remove the current six-month eager request. Initially request selected and previous month only; load more history only when a visible chart requires it.

Page order:

1. month control and report identity;
2. headline cross-domain comparison;
3. primary charts;
4. previous-month comparison;
5. exact domain details/tables;
6. source, method, coverage, and generation metadata.

Create:

- apps/iroha-web/src/lib/themes/atlas/Reports.svelte
- apps/iroha-web/src/lib/themes/grapher/Reports.svelte
- apps/iroha-web/src/lib/themes/field-journal/Reports.svelte
- apps/iroha-web/src/lib/themes/phenology/Reports.svelte
- apps/iroha-web/src/lib/themes/sound-map/Reports.svelte
- apps/iroha-web/src/lib/themes/archive/Reports.svelte

Orientation:

| Theme         | Leading representation                                      |
| ------------- | ----------------------------------------------------------- |
| Atlas         | monthly survey across domains with drill-through to records |
| Grapher       | aligned month-over-month comparison with unit-safe series   |
| Field Journal | dated monthly record emphasizing observations and gaps      |
| Phenology     | monthly cycles and recurring phases                         |
| Sound Map     | monthly intensity/cadence bands                             |
| Archive       | preserved report folio with exact totals and provenance     |

Acceptance:

- only monthly controls exist;
- arrow keys move one month and update URL;
- direct year/month selection works;
- previous-month baseline cases are labeled honestly;
- charts precede detail tables/cards;
- six components are distinct;
- v1 section state is preserved;
- no browser errors under all themes.

## 12. Phase 8 — navigation and metric discovery

Replace the manually indexed array in apps/iroha-web/src/lib/navigation.ts with grouped typed navigation.

```ts
type NavigationItem = {
  id: string;
  label: string;
  href: string;
  kind: "primary" | "domain" | "analysis" | "tool";
  hint: string;
};

type NavigationGroup = {
  id: string;
  label: string;
  items: readonly NavigationItem[];
};
```

Initial groups:

- primary: Today /, Overview /overview;
- domains: Motion /motion, Night /night, Library /library, Expenses /expenses;
- analysis: Patterns /patterns, Reports /reports;
- tools: To-go /to-go, Admin /admin, Design /design.

Header shows primary items plus Domains, Analyze, More, and Search/Command controls. Existing direct routes do not change. Metrics never become primary header items.

Update:

- apps/iroha-web/src/lib/navigation.ts;
- apps/iroha-web/src/routes/+layout.svelte;
- apps/iroha-web/src/routes/app.css;
- apps/iroha-web/src/lib/components/CommandPalette.svelte;
- apps/iroha-web/src/routes/routes.test.ts.

Create/extend:

- apps/iroha-web/src/lib/components/NavigationMenu.svelte;
- apps/iroha-web/src/lib/components/MetricSearch.svelte;
- apps/iroha-web/src/lib/navigation.test.ts.

Command palette consumes the same grouped navigation plus metric catalog; it must not maintain a second destination list.

Accessibility:

- menu buttons expose expanded/controls state;
- Escape closes and restores focus;
- arrow keys navigate menus;
- mobile uses a real menu, not endless horizontal overflow;
- nested routes highlight parent group;
- direct links and browser navigation remain stable.

Acceptance:

- adding a metric does not change the header;
- adding a domain requires registry entries, not edits to hard-coded anchors;
- command search finds routes and metrics;
- desktop/mobile screenshots have no clipped navigation;
- keyboard labels and focus order are correct.

## 13. Phase 9 — six-theme completeness

Expense and report pages are the first mandatory six-theme slice. Then audit the existing pages against the same orientation contract.

| Page           | Atlas              | Grapher                      | Field Journal       | Phenology                | Sound Map       | Archive            |
| -------------- | ------------------ | ---------------------------- | ------------------- | ------------------------ | --------------- | ------------------ |
| Today          | territory snapshot | daily evidence               | dated entry         | current phase            | daily cadence   | current folio      |
| Overview       | domain/place map   | cross-domain comparison      | continuity overview | cycle overview           | signal board    | collection index   |
| Patterns/Daily | survey transect    | time series                  | pattern notebook    | recurrence calendar      | intensity bands | period ledger      |
| Motion         | route index/map    | activity comparison          | session log         | effort/recovery sequence | cadence trace   | activity register  |
| Night          | overnight bearing  | sleep series                 | night entry         | sleep phases             | sleep rhythm    | sleep record       |
| Library        | collection atlas   | completion/rating comparison | media notes         | media cycles             | media cadence   | catalog/provenance |
| Expenses       | spending survey    | monetary comparison          | dated entries       | recurrence               | intensity       | exact ledger       |
| Reports        | monthly survey     | aligned comparison           | monthly record      | monthly cycle            | monthly signal  | preserved folio    |

For every page/theme pair record: primary question, first visual, time framing, first interaction, detail/provenance order, empty/error treatment, responsive composition, and chart rationale.

Do not rewrite all legacy pages before the expense/report architecture proves itself. The audit can identify later work; implementation is staged by page family.

## 14. Phase 10 — measured database decision

Before adding a migration, build a repeatable fixture and capture:

- daily series for one and ten years;
- movement series with multiple sports;
- monthly report generation;
- expense series with currencies/categories;
- cache hit/miss latency;
- EXPLAIN ANALYZE BUFFERS for slow resolvers.

Development SLOs on the agreed fixture:

- normal one-year series under 250 ms p95 server time;
- monthly report under 500 ms p95 server time.

If targets pass, add no read-model migration.

If they fail, prefer:

1. canonical query/index correction;
2. cache invalidation improvement;
3. narrow derived read model.

Only option 3 may create a migration, tentatively 00008_metric_series_cache.sql. It must include metric_id, grain, period_start, dimensions JSON/hash, value, observed_days, method_version,
source_generation, computed_at, unique metric/grain/period/dimension key, invalidation, rebuild, stale behavior, rollback, and canonical-authority tests.

No chart may silently present stale values as current.

## 15. Phase 11 — tests and fixtures

### Server unit

- registry uniqueness/completeness;
- definition serialization;
- period/timezone/grain/boundaries;
- dimension parsing/canonical ordering;
- null/coverage;
- deterministic series order;
- currency isolation;
- method-version metadata.

### Server integration

Use metrics, reports, expenses, and domain period integration tests. Cover:

1. no data;
2. one point;
3. missing middle period;
4. observed zero;
5. multiple units;
6. multiple sports/kinds/categories;
7. multiple currencies;
8. deleted expenses;
9. timezone boundary;
10. repeated deterministic request;
11. invalid query/common errors;
12. unknown daily metric stored but not catalog-visible.

### Frontend

- API encoding;
- null handling;
- chart/table parity;
- semantic colors;
- month URL synchronization;
- left/right keys;
- direct year/month;
- six-theme component identity;
- orientation checklist;
- grouped navigation/palette consistency.

### Browser

Use make web-visual-check with expenses and reports for all six themes and light/dark modes. Check browser errors each run. Keep screenshots in ignored visual output.

Required after code phases:

```text
make check
```

Before release candidate:

```text
make validate
make web-visual-check THEME=atlas ROUTE=expenses,reports
```

Repeat the visual check for all themes. Do not run make db-reset casually.

## 16. Commit sequence

Use small green conventional commits:

1. docs: add iroha data cockpit implementation plan
2. feat: add metric catalog definitions
3. feat: add deterministic metric series contract
4. feat: expose metric catalog and series API
5. refactor: share metric aggregation with monthly reports
6. feat: add metric chart and metadata primitives
7. feat: rebuild expense cockpit compositions
8. feat: rebuild monthly report compositions
9. feat: group cockpit navigation and metric search
10. test: add cross-domain metric fixtures

Do not combine deployment changes, k3s version changes, or unrelated dirty frontend edits. Push and production rollout remain separate approvals.

## 17. Stop conditions

Stop and revise if:

- a resolver needs provider observations for the default canonical series;
- a metric lacks unit, reducer, coverage, or method version;
- a chart needs frontend aggregation of paginated rows;
- adding one metric requires a top-level navigation route;
- a theme differs only by CSS tokens;
- a migration stores canonical meaning only in JSON;
- a report change breaks v1 OpenAPI or CLI output;
- currency conversion appears without explicit versioned exchange-rate source;
- weekly report code appears;
- Telegram becomes a dependency of direct API/local-agent workflows.

## 18. Definition of done

Complete only when:

- catalog definitions and API examples are documented;
- initial cross-domain metrics are available through the metric API;
- monthly reports remain compatible;
- expenses/reports use server-provided chart series;
- chart/table/metadata/download views agree;
- six themes have real expense/report compositions;
- global header is grouped and not metric-shaped;
- month navigation supports direct year/month and arrow keys;
- missingness, coverage, units, dimensions, provenance, and method versions are visible;
- fixtures cover empty, partial, zero, multi-dimensional, currency, and timezone cases;
- make check and make validate pass;
- six-theme light/dark visual checks pass;
- any migration is performance-justified with invalidation/rollback proof;
- deployment remains a separately approved release step.
