# Iroha data cockpit rework plan

Status: draft for review. Based on [`2026-08-12-owid-data-cockpit-research.md`](../research/2026-08-12-owid-data-cockpit-research.md).

This plan is architecture work for the next milestone. It does not authorize a big-bang database rewrite, deployment, or expansion of the Telegram/agent workflow.

## Outcome

Make Iroha a growing personal data cockpit with:

- stable typed canonical APIs;
- a discoverable metric catalog;
- deterministic server-owned series and report aggregation;
- chart-first representations with table, record, provenance, and export views;
- genuinely distinct compositions for each registered production language, plus an extensible registry of adopted layout systems;
- scalable domain/local/search navigation that does not turn every metric into a top tab;
- monthly reports only, with no weekly report feature;
- direct expense API intake for local agents, independent of Telegram.

## Non-goals

- Copying OWID Grapher or its source code.
- Replacing Iroha's typed domain tables with EAV storage.
- Adding expense candidate/revision/mutation workflow.
- Making Telegram depend on local agents or user confirmation.
- Adding weekly reports.
- Making the frontend a second aggregation engine.
- Introducing a database-backed metric editor before runtime editing is a real requirement.
- Deploying while this architecture is under review.

## Design rules

1. Canonical records retain domain identity and constraints.
2. Metric definitions describe meaning; chart configuration describes a view; theme components describe composition.
3. Every aggregate declares period grain, timezone, unit, coverage, and method version.
4. Missing is not zero.
5. Server code owns aggregation; the browser only transforms a returned series for display.
6. API responses expose facts and semantic tokens, never theme-specific CSS colors.
7. New metrics are discoverable in the catalog and search, not added to the global header.
8. Existing `/api/v1` contracts change additively until a release/version decision is approved.
9. Each theme has one primary analytical question per page; visual metaphors never override the data relationship.

## Work breakdown

### Phase 0 — inventory and contract gate

Deliverables:

- a metric inventory covering current daily, movement, sleep, media, expense, and report values;
- a source map from each metric to canonical table/service/query;
- a classification: canonical field, derived metric, breakdown/dimension, or record count;
- unit and missingness rules;
- reducer and aggregation method names with versions;
- a domain/navigation taxonomy;
- the approved orientation matrix for registered production languages, plus the adoption contract for new shared compositions: primary question, visual grammar, time model, interaction model, and
  anti-patterns;
- fixtures containing empty, partial-coverage, multi-currency, multi-unit, and multi-dimension data.

Acceptance:

- every current chart/report number has one owner and one documented calculation;
- no metric is named only by a presentation label;
- period boundaries are explicit and consistent;
- navigation decisions identify primary domains, local views, and tools.
- each registered language can explain why its first chart and first detail surface differ from the other registered languages, and each adopted composition has a distinct layout intent.

### Phase 1 — metric catalog, code-owned first

Add a Go package, tentatively `pkg/metrics`, with typed definitions and registry lookup. The first shape should include:

```go
type Definition struct {
    ID                 string
    Domain             string
    Label              string
    Description        string
    ValueType          string
    Unit               string
    SupportedGrains    []string
    SourceKind         string
    Reducer            string
    AggregationVersion string
    CoverageKind       string
    Dimensions          []DimensionDefinition
    SemanticColorToken string
}
```

The exact Go names can follow repository style. Definitions are immutable application metadata in this phase; they are not user-entered data.

Add:

- deterministic IDs such as `health.steps`, `movement.distance_m`, `sleep.asleep_s`, `expenses.amount_minor`;
- validation for duplicate IDs, invalid units, unsupported grains, and missing method versions;
- tests that the registry is exhaustive for all exposed metric IDs;
- `GET /api/v1/metrics` and `GET /api/v1/metrics/{metric_id}`;
- OpenAPI schemas and example fixtures.

Do not add a migration in this phase.

### Phase 2 — common series contract

Define a typed internal result and HTTP DTO. A point must carry period and value; the series must carry metric identity, unit, grain, timezone, coverage, dimensions, and method/source metadata.

Implement adapters for one metric in each of three different domains:

1. `health.steps` from `tb_daily_metrics`;
2. `movement.distance_m` from canonical activities;
3. `expenses.amount_minor` with currency as a required dimension and no cross-currency sum.

Candidate endpoint:

```text
GET /api/v1/metrics/{metric_id}/series?from=YYYY-MM-DD&to=YYYY-MM-DD&grain=day|month&timezone=...
```

Rules:

- validate the requested grain against the definition;
- use half-open `[from,to)` internally;
- sort points deterministically;
- return empty points plus coverage rather than fabricated zeros;
- reject invalid dimensions and currency mixing;
- preserve integer/minor-unit precision for money;
- never run arbitrary metric SQL from a request parameter.

Acceptance:

- API integration tests compare exact points against hand-checked fixtures;
- the same request returns byte-equivalent JSON apart from explicitly documented generated metadata;
- a frontend test proves no raw domain list endpoint is used to construct the chart.

### Phase 3 — report evolution without breaking v0.4

Keep the current monthly report endpoint and schema through the v0.4 release gate. Refactor its internal contributors to use shared metric adapters only where the semantics match; retain typed rich
sections for movement, sleep, media, and expenses.

Design the next additive report envelope:

```text
ordered sections[]
section key + schema + state + typed data
```

The new envelope must support a new metric section without adding a field to a monolithic Go struct. It must still permit domain-specific sections such as completed media items and expense currency
totals.

Do not introduce weekly reports. The only period navigation in scope is month-by-month, with direct year and month selection.

### Phase 4 — representation primitives

Build shared frontend primitives around the stable series contract:

- chart-first metric panel;
- line/bar/stacked composition where semantically valid;
- chart-local `Chart`/`Table` view state persisted in the URL;
- tooltip showing label, value, unit, period, and coverage;
- accessible ECharts ARIA configuration;
- semantic color-token resolution through the selected theme;
- empty/partial/unavailable states;
- details drawer/section with source, method, and record links;
- download/copy JSON or CSV for the displayed series;
- table parity for exact values.

Fix the existing category-color regression with a focused component test: ensure datum-level colors are not overwritten by series-level `itemStyle`. Add a color-token test for every registered
language and adopted composition that renders the chart. Do not make the API return CSS values.

### Phase 5 — expense and report page architecture

Refactor each route into:

```text
+page.svelte controller
  -> loader + URL state + stable view model
  -> ThemeRouteRenderer
  -> six page-specific theme components
  -> shared metric/chart/table/date primitives
```

For expenses:

- keep the page read-only except the already-approved delete action;
- use one month selector for list and aggregation scope;
- auto-apply filters when appropriate; no useless Apply button;
- chart first, ledger/details below;
- category/currency are dimensions with stable labels and theme colors;
- maintain direct record links and precise money formatting.

For monthly reports:

- make the month the one shared period control;
- support previous/next month with left/right arrow keys;
- support direct year selection and year+month selection;
- compare the selected month with the previous month and clearly label unavailable comparisons;
- show charts before domain detail cards;
- expose method/coverage/source details;
- keep monthly-only scope.

For each registered production language, with adopted compositions joining as their route coverage is implemented:

- add separate `expenses` and `reports` components under each theme, or an equivalent theme-owned composition registry;
- each must implement the decided lens: Atlas/territory, Grapher/comparison, Field Journal/observation, Phenology/recurrence, Sound Map/rhythm, Archive/provenance;
- each must change hierarchy, typography, surfaces, chart treatment, interaction emphasis, responsive composition, and empty/error states;
- shared data and behavior may be reused, but a shared `CockpitFrame` cannot be the final implementation.

Acceptance:

- registry tests prove distinct component identities for each registered language/page pair and addressable shared implementations for adopted compositions;
- orientation review proves the first question, chart, time framing, and detail order differ appropriately by theme;
- browser checks verify chart-first layouts and theme-specific visual markers;
- keyboard tests cover ArrowLeft, ArrowRight, Escape, and direct period selection;
- screenshot review covers all registered languages for expenses and reports, plus every adopted composition specimen.

### Phase 6 — navigation rework

Replace the hard-coded nine-item global nav with a data-driven layered model.

Proposed model:

- global: Today, Overview, stable domain destinations, Search/Command;
- local: views belonging to the selected domain;
- analysis/tools: Reports, Tasks, import/admin surfaces in a secondary menu or command palette;
- metric discovery: searchable metric catalog and overview metric shelf.

Requirements:

- no metric adds a new global top tab;
- direct URLs remain stable;
- active state works for nested routes;
- desktop has bounded primary links and an explicit overflow/menu;
- mobile has a real menu, not an indefinitely scrolling header;
- keyboard focus order and shortcuts are documented;
- navigation labels are domain journeys, not implementation tables.

The current `CommandPalette` becomes the first search surface for metrics and destinations, but it should consume the same navigation/metric registry rather than maintaining another hand-written list.

### Phase 7 — storage decision and measured optimization

Only after Phases 1–5 have real data and query measurements:

- inspect `EXPLAIN` plans and request latency for series and monthly reports;
- identify repeated joins or expensive historical scans;
- choose between indexes, a narrow materialized read model, or existing cache;
- if a read model is needed, define refresh trigger, method version, invalidation, and stale/error behavior before adding a migration;
- never silently serve a stale chart as current.

If runtime metric editing becomes necessary, add `tb_metric_definitions` as metadata only. Executable aggregation remains versioned Go code.

## Verification gates

### Gate A — architecture approval

Approve the metric definition shape, common series envelope, code-owned catalog decision, report compatibility strategy, and navigation layers.

### Gate B — vertical API slice

Pass Go unit/integration tests, OpenAPI contract checks, deterministic JSON fixtures, and a browser proof for one health, one movement, and one expense series.

### Gate C — report and cockpit migration

Pass monthly report compatibility tests, chart-first browser checks, date keyboard checks, registered-language and adopted-composition visual checks, and no raw-row frontend aggregation checks.

### Gate D — navigation and expansion

Pass nested active-state, mobile/overflow, keyboard, command-palette, metric-search, and direct-link checks. Add a new metric fixture and prove it appears without a new top tab.

### Gate E — release

Run the repository's normal `make check` and relevant web/server contract targets. Deployment status remains deferred until production release approval.

## Risks and mitigations

| Risk                                                                | Mitigation                                                                |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Generic metric API erases domain meaning                            | Keep typed domain APIs and adapter-owned semantics                        |
| Metadata becomes stale                                              | Version definitions and aggregation methods; test registry exhaustiveness |
| Charts look colorful but mislead                                    | Unit/coverage/method in response and tooltip; table parity                |
| Registered languages or adopted compositions collapse into palettes | Separate page components and visual contract tests                        |
| More metrics recreate tab overload                                  | Domain navigation plus metric search/shelf; no metric top tabs            |
| Materialized values become stale                                    | On-demand first; measured storage decision later                          |
| OWID inspiration becomes code copying                               | Use concepts only; do not copy source or licensed implementation          |
| API v0.4 destabilizes late                                          | Additive endpoints and preserve current monthly contract through release  |

## First implementation slice after approval

The smallest useful first slice is:

1. write the metric inventory and definitions for `health.steps`, `movement.distance_m`, and `expenses.amount_minor`;
2. add the code-owned catalog and tests;
3. implement one common series DTO and one endpoint;
4. render one chart-first page with table/metadata parity;
5. use that slice to settle the final API and frontend boundary before touching the database schema.
