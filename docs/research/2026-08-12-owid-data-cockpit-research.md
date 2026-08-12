# Iroha as a world-in-data cockpit

## Status

Research report and architecture recommendation. This is not an implementation approval and does not change the current v0.4 expense/report scope.

## Executive conclusion

Iroha should learn from Our World in Data (OWID) at the level of information architecture, metadata discipline, deterministic transformations, and view composition. It should not copy OWID's warehouse, editorial workflow, or Grapher source code.

The central recommendation is a hybrid model:

```text
immutable evidence / provider observations
        -> typed canonical domain records
        -> metric definitions + deterministic read models
        -> chart / table / record / provenance views
```

The canonical database should remain domain-shaped. Activities, routes, sleep episodes, daily health facts, media history, and expenses have different identities and semantics; converting them into one generic value table would lose those semantics. The missing layer is above those tables: a stable metric catalog and a common series/read-model contract that can describe values from any domain without making storage generic.

The frontend should stop treating every new metric as a new global destination. The current nine-link top navigation is already a flat list of routes. It should become a small set of domain destinations, with local view tabs inside a domain and searchable metric discovery for the growing metric set. OWID's useful pattern is not “more tabs”; it is a restrained global navigation plus chart-local `chart/table/map` views, topic browsing, search, metadata, and downloads.

The v0.4 expense/report work exposed a second architectural issue: the route files currently own fetching, derived aggregation, and most markup, while the theme registry maps all six languages to the same `CockpitFrame`. The next redesign should separate route data loaders from theme-specific page compositions. A common metric contract can be shared; the six visual identities must remain separate.

## Research method and evidence

This report combines:

1. OWID's official product and data-reuse documentation.
2. OWID ETL and catalog architecture documentation.
3. A pinned source snapshot of `owid/owid-grapher` at commit `3f1df6f0b442e175da85b65391b0f5f8eddec3a5` (2026-08-12), used for design and implementation study only.
4. A source audit of Iroha's current migrations, Go services, HTTP handlers, Svelte routes, theme registry, and existing v0.4 design documents.

OWID's current repository is explicitly published for transparency and educational study, not general code reuse. Iroha must not copy its implementation; the repository license and disclaimer are especially clear that Grapher is tightly coupled to OWID's database and production environment. See the [OWID repository README](https://github.com/owid/owid-grapher) and [license](https://github.com/owid/owid-grapher/blob/master/LICENSE.md).

## What OWID actually built

### 1. A data pipeline, not a chart API glued to raw files

OWID's documented pipeline has distinct curation levels:

```text
source snapshot -> meadow cleaning -> garden curation -> grapher export
```

The [ETL workflow](https://docs.owid.io/projects/etl/architecture/workflow/) preserves source snapshots, transforms them through named steps, propagates metadata, and produces the data consumed by Grapher. The [common format](https://docs.owid.io/projects/etl/architecture/design/common-format/) pairs table data with JSON metadata. The catalog has a hierarchy of `Dataset -> Tables -> Variables`, documented in the [catalog structures](https://docs.owid.io/projects/etl/libraries/catalog/structures/).

The important property is not the names “meadow” or “garden”. It is that a displayed number has a traceable path from source evidence to a curated variable and then to a presentation. A chart is not the canonical data store.

### 2. Variables are first-class, described objects

OWID's variable metadata includes a stable identity, title, unit and short unit, description, coverage, timespan, source, origins, licenses, processing level, processing description, update cadence, dataset identity, catalog path, dimensions, and checksums. The metadata is authored alongside ETL code and propagated through the pipeline; see the [metadata architecture](https://docs.owid.io/projects/etl/architecture/metadata/) and [metadata reference](https://docs.owid.io/projects/etl/architecture/metadata/reference/).

This solves a common data-product failure: a value such as `7.2` is not meaningful without knowing whether it is hours, kilometers, a percentage, an average, a count, an observed-day average, or a derived estimate. OWID makes this context available to both the renderer and the reader.

### 3. Dataset values and chart configuration are separate

The source database documents separate concepts for datasets, variables, origins, chart configurations, chart dimensions, chart revisions, tags, and analytics. Chart configuration selects and presents variables; it is not the definition of the variable itself. OWID also stores patch/full configuration forms and revisions, which supports inheritance and repeatable updates.

Iroha should adopt the separation, not the entire admin database:

```text
metric meaning and provenance != chart choice != page composition
```

### 4. One dataset has multiple honest views

OWID's redesign explicitly treats a chart, map, and spreadsheet-like table as different views over the same dataset. The [chart redesign article](https://ourworldindata.org/redesigning-our-interactive-data-visualizations) describes controls for refining and highlighting the view, prominent sourcing, “Learn more about this data”, full-screen exploration, and downloads.

The [Chart API](https://docs.owid.io/projects/etl/api/chart-api/) exposes CSV, metadata JSON, ZIP, and README outputs. The data rows have a declared grain, normally entity plus time plus one or more variable columns. The metadata explains units, timespan, citations, and configuration. This is a strong model for Iroha's future metric pages: a chart is the first representation, but table and source/method details are always available.

### 5. Chart state is a typed transformation pipeline

In the studied source, `OwidTable` carries rows plus column definitions and metadata. `ChartState` has an input table, transformed table, optional display/selection transforms, series, color scale, supported dimensions, and sort keys. Chart types transform the same input into their own renderable form. `ChartTabs` defines a bounded set of valid view types and maps them to URL state.

This is the most transferable implementation lesson:

```text
API dataset + metadata
        -> normalized client table
        -> view-specific transform
        -> chart/table renderer
```

The renderer should not guess units, aggregate raw records, or invent labels. It should receive a well-described series and make a view of it.

### 6. Metadata and disclosure are part of the product surface

OWID's data page includes an indicator metadata box, an “about this data” section, source/processing explanations, related research, and a download section. These are not admin-only features. They answer “what is this?”, “where did it come from?”, “how was it calculated?”, and “can I export it?” at the point where a user sees the number.

For Iroha, “source” often means an Apple Health sample, an activity provider observation, a manually posted expense, or a deterministic aggregation rule—not a public citation. The same disclosure principle still applies.

### 7. OWID does not use a giant global tab bar

The current OWID site navigation has a small set of global links such as Browse by topic, Data, Latest, Resources, and About, with search and overlays. Topic browsing and search handle scale. Inside a data page, tabs are local to the chart and represent view modes such as table, map, and chart. Multi-dimensional pages use bounded dropdowns with a “Show more” affordance rather than putting every dimension in the global header.

This is the right lesson for Iroha: navigation hierarchy and metric/view selection are separate problems.

## Iroha current situation

### Canonical and evidence layers are already a strength

Iroha's current model already contains the important evidence boundary:

- `tb_raw_files`, import jobs, snapshots, and intake payloads retain evidence and reprocessing identity.
- Provider observations preserve source-specific reports.
- Canonical tables provide fast domain reads.
- Selected observations and reducer rules resolve provider disagreement.

This is documented in [Iroha core capabilities](../capabilities/core.md), [provider capabilities](../provider-capabilities.md), and [ADR 0001](../adr/0001-provider-observations-and-canonical-records.md). The existing database migration has typed tables for activities, routes, samplings, laps, sleep sessions/segments, daily summaries/metrics, and observations. This is materially closer to the right long-term model than a single EAV table.

The expense direct API is a different path and should remain so:

```text
local agent or client -> stable expense JSON -> POST /api/v1/expenses
```

An expense does not need a candidate/revision/mutation workflow. `tb_intake_payloads` remains useful for connector/file evidence, but it must not be inserted into the expense UX or required for a direct API create.

### Metrics are partly open-ended, but not yet first-class

`tb_daily_metrics` is already a long-form table with `(day, metric, value, unit)` and explicit aggregation behavior. The daily aggregate response is correspondingly open-ended: `metrics` is an array of `{metric, value, unit, observed_days}`. This is a good foundation for adding daily health measures without changing a fixed JSON object.

The rest of the API is less extensible:

- movement, sleep, media, and expenses have typed report structures;
- `MonthlyReport` has a fixed `ReportSections` object;
- the web API type mirrors that fixed shape;
- no API endpoint describes a metric's title, semantic type, reducer, coverage, source, provenance, or visual defaults;
- the frontend therefore has to know metric labels and chart choices in route code.

This is the central gap. Iroha has metric values but not a durable metric catalog.

### Database: do not replace domain tables with EAV

The current domain tables encode constraints that a generic metric table cannot express well:

- an activity has an interval, sport, route, laps, and sampling streams;
- sleep has an episode, wake date, stages, and main-sleep semantics;
- expenses have money-specific minor units, currency, category, source identity, and deletion semantics;
- media has work identity, items, events, and progress.

Replacing these with `(entity, metric, value)` would make the database look flexible while pushing identity, validation, joins, and aggregation meaning into every consumer. It would also weaken the stable API that the user wants.

The database changes that may be justified are narrower:

1. Add indexes or read models for repeated cross-domain series queries after measuring actual query plans.
2. Add durable metric-definition storage only if metric definitions must be edited at runtime by a non-developer. Until then, a versioned Go registry is simpler and deterministic.
3. Add a provenance/read-model link for derived series so a chart can identify the canonical records and aggregation rule behind it.
4. Keep materialized monthly values optional. The current single-user scale favors deterministic on-demand aggregation with cache invalidation before introducing refresh jobs and stale-result semantics.

### API: typed domain records plus a common metric envelope

The current `/api/v1` route inventory is explicit and healthy. The API already uses opaque IDs, named DTOs, pagination envelopes, common errors, and server-owned aggregation. Those decisions should stay.

The missing contract should be additive:

```json
{
  "metric_id": "health.steps",
  "label": "Steps",
  "domain": "health",
  "value_type": "number",
  "unit": "count",
  "period": { "grain": "day", "from": "2026-08-01", "to": "2026-09-01", "timezone": "Asia/Tokyo" },
  "points": [
    { "period": "2026-08-01", "value": 8120, "observed": true, "observed_days": 1 }
  ],
  "coverage": { "observed_periods": 1, "expected_periods": 1 },
  "source": { "kind": "canonical", "method": "daily_metric.average.v1" }
}
```

The exact wire shape is a plan decision, but these meanings are non-negotiable:

- `metric_id` is stable and machine-oriented; labels can change.
- `unit` is canonical and never inferred by the frontend.
- `grain`, timezone, and half-open period boundaries are explicit.
- absence is not zero; coverage is visible.
- a reducer/aggregation method has a version.
- dimensions are explicit when present (sport, currency, category, media kind).
- source/provenance is available without requiring the chart to join raw tables.

Typed endpoints remain the right choice for canonical records and rich details. The common series contract is for chartable derived values, not a license to flatten all domain objects.

### Frontend: current theme contract is good, current cockpit exception is not

The theme architecture correctly says that a theme can change shell, navigation, composition, typography, chart treatment, density, motion, and responsive behavior, while API meaning and privacy behavior remain stable. The six shell implementations are genuinely distinct.

However, the theme registry currently maps `expenses` and `reports` for all six languages to the same `CockpitFrame`. That contradicts the completeness contract and explains why the pages can feel like one neutral white/gray page with a theme wrapper. It is not solved by adding six CSS classes.

The expense and report route files also combine loading, derived values, and markup inside the route's child snippet. The next frontend boundary should be:

```text
route loader/controller
        -> stable page view model
        -> theme-specific page component
        -> shared chart/table/date/metadata primitives
```

Shared primitives may own keyboard behavior, URL state, ECharts lifecycle, accessibility, and data formatting. A theme owns the composition and visual grammar. A metric definition supplies semantic color tokens and labels; it does not supply a theme's entire appearance.

The current dirty `BarChart.svelte` change also shows why the shared chart contract needs tests: datum-level category colors are currently vulnerable to being overwritten by a series-level `itemStyle`. This is an implementation bug to fix in the later plan, not a reason to invent a new chart system.

## Decision: six themes are six analytical lenses

Research from OWID and mature design systems converges on one rule: chart selection must follow the relationship the viewer needs to understand, not the designer's preferred decoration. Official guidance groups common tasks as time series, comparison/ranking, composition, distribution, correlation, and performance; it also recommends familiar charts, clear intent, and non-color distinctions for accessibility. See the [Scottish Government chart guidance](https://designsystem.gov.scot/guidance/charts), [USWDS visualization guidance](https://designsystem.digital.gov/components/data-visualizations/), and [IBM chart guidance](https://www.ibm.com/design/language/data-visualization/charts/).

Iroha's themes should therefore be stable editorial lenses over the same metrics. They may select different composition, emphasis, vocabulary, and interaction, but they must not manufacture a relationship the data does not contain. A theme can choose a route map when a route exists; it must not turn an expense total into a fake map merely to preserve a visual metaphor.

The decided orientation matrix is:

| Theme | Primary question | Preferred visual grammar | Time model | Interaction | Must avoid |
| --- | --- | --- | --- | --- | --- |
| Atlas | Where did it happen, and what path or territory does it describe? | maps, route traces, indexed transects, spatial/ordered panels | journey and chronological survey | pan, select, drill from region/period to record | fake geography, decorative maps without spatial data |
| Grapher | What changed, and how do values compare? | line/bar/slope charts, aligned comparison, table, source panel | continuous series and period-over-period comparison | filter, highlight, change grain, inspect values | mixing unlike units, overloaded multi-series charts |
| Field Journal | What was observed, and how does it continue as a personal record? | dated entries, annotated timeline, evidence cards, small multiples, sparklines | day, sequence, and continuity | browse, expand, annotate context, open source record | scores, gamified readiness, hiding uncertainty |
| Phenology | What recurs, cycles, or changes phase? | calendar heatmaps, radial cycles, phase bands, seasonal small multiples | recurring day/week/month/season cycles | move through cycle, compare phases, inspect recurrence | implying biological causality or medical diagnosis |
| Sound Map | What is the rhythm, density, cadence, or intensity? | interval bands, waveform-like timelines, pulse bars, density heatmaps | intraday cadence and bursts over time | scrub/scan, zoom density, isolate a channel | pretending data is audio, unreadable decoration, false precision |
| Archive | What exists, when was it recorded, and where did it come from? | accession lists, ledgers, chronology, facets, provenance rails, exact tables | historical chronology and source/version order | search, filter, compare, inspect provenance | turning an exact record into an ornamental dashboard |

This matrix is a product decision, not a requirement that every page use every chart family. Each page gets one primary question and at most two supporting relationships. For example, an expense page in Grapher may lead with month-over-month comparison and category composition; the same data in Archive leads with an exact ledger and source trail. Both still show the same amount, currency, category, date, and deletion semantics.

The matrix also gives a practical completeness test. A theme implementation is real when its page changes the question, leading composition, time framing, interaction, and evidence order—not merely its colors or border radius. The theme selector can be hidden and the viewer should still recognize the lens.

## Recommended target architecture

### A. Keep canonical storage domain-shaped

Retain the current tables and observation model. Add no generic `tb_metric_values` table in the first redesign. Preserve the direct expense API and the provider evidence pipeline as separate workflows.

### B. Introduce a metric catalog

Start as a versioned Go registry, exposed through an API. A definition should include:

```text
id, domain, label, description
value type, canonical unit, display formatter
source capability, canonical field or query
reducer, aggregation method/version
coverage semantics, supported grains
dimensions and allowed values
semantic color token, good/bad direction where meaningful
```

The registry should cover existing values first: steps, distance, move, exercise, stand, resting heart rate, sleep duration/efficiency/stages, activity distance/duration, media completion, expense totals, and category totals. It should not pretend that every scalar field is automatically a useful metric.

When non-code editing becomes a real requirement, move definition metadata into `tb_metric_definitions` with a migration and keep executable aggregation implementations in Go. Do not put SQL or arbitrary code in database rows.

### C. Add a common server-side series/read-model layer

Each metric provider adapts its domain service to a common series result. The adapter owns the correct query and reducer; the common layer owns period validation, ordering, coverage, dimensions, and metadata joining. The frontend never aggregates raw activities, sleep rows, expenses, or provider observations.

Candidate read routes for the next plan:

```text
GET /api/v1/metrics
GET /api/v1/metrics/{metric_id}
GET /api/v1/metrics/{metric_id}/series?from=&to=&grain=&timezone=&dimension...
GET /api/v1/overview?from=&to=&grain=&metrics=...
```

These are design candidates, not approved paths. Existing routes remain compatible while the new contract proves itself. Reports can consume the same metric adapters but may keep richer typed domain sections for details.

### D. Evolve reports from fixed sections to ordered capabilities

The current monthly report is a valid v0.4 contract and should not be broken casually. Its next version should use an ordered section envelope, like the existing briefing pattern:

```json
{
  "period": { "kind": "month", "month": "2026-08", "timezone": "Asia/Tokyo" },
  "sections": [
    { "key": "movement", "schema": "movement.month.v1", "state": "available", "data": {} },
    { "key": "health.steps", "schema": "metric.series.v1", "state": "available", "data": {} }
  ]
}
```

This lets new metrics appear without changing one giant top-level struct. It does not mean every report section becomes an untyped map: each section still has a schema and a typed owner.

### E. Make representation a first-class product contract

For each metric or report section, the frontend should offer a deliberate sequence:

1. headline interpretation and comparison;
2. primary chart;
3. chart-local view controls (`chart`, `table`, and where meaningful, distribution/breakdown);
4. readable details and record links;
5. source, method, coverage, and export.

The chart is first because Iroha is a colorful data cockpit. The table is still required for exact inspection, not used as the primary visual. There should be no generic “apply filter” button for controls that can update immediately; filter state belongs in the URL and can be restored with browser navigation.

### F. Replace the growing global tab list with navigation layers

Metrics must not become top-level tabs. The proposed hierarchy is:

```text
global header: Today | Overview | domains | search/command
domain header: local views for the selected domain
page content: metric shelf, chart-local views, records, provenance
command/search: infrequent metrics, reports, tools, direct destinations
```

For the current routes, “Motion”, “Night”, “Library”, and “Expenses” are domains; “Patterns” and “Reports” are cross-domain analysis; “To-go” and jobs/tasks are tools. The final taxonomy needs a product decision, but the invariant is clear: global navigation names stable user journeys, not every metric or chart.

The top bar must have an overflow/mobile behavior, preserve direct links, expose active state, and support keyboard navigation. A domain page may have horizontal local tabs, but those tabs must be bounded and represent views of that domain—not a second global dump of every metric.

## Decision matrix

| Option | Storage | API | Growth | Risk | Decision |
| --- | --- | --- | --- | --- | --- |
| Copy OWID warehouse/Grapher | generic catalog/data warehouse and chart DB | broad chart API | high theoretical scale | wrong scale, high migration, licensing | Reject |
| Generic EAV metric values | one metric/value table | flexible but weakly typed | easy to add names, hard to preserve meaning | semantic loss and query complexity | Reject |
| Metadata-only registry | current domain tables | typed APIs plus metric metadata | good for definitions, limited cross-domain series | low | First foundation |
| Hybrid catalog + domain adapters + read models | current domain tables, optional later definition table | common series envelope plus typed records | grows without flattening domains | moderate and measurable | Recommended target |

## Gaps that the next plan must close

1. Define the metric identity, unit, reducer, coverage, dimensions, and method-version contract.
2. Decide whether the first catalog is code-owned or DB-backed; recommendation: code-owned first.
3. Map existing domain fields and report fields into metric definitions without changing their meaning.
4. Implement one common series adapter and prove it with health, movement, and expenses.
5. Decide whether monthly reports remain fixed in v0.4 and receive an additive extensible version afterward; recommendation: preserve v0.4, design the next report envelope additively.
6. Split expense/report route controllers from theme components.
7. Build six real expense/report compositions, not one `CockpitFrame` with six palettes.
8. Establish chart primitives with category colors, semantic tokens, ECharts accessibility, tooltip units, empty states, and table parity.
9. Add URL-persisted date/view/filter state and left/right month navigation.
10. Replace the flat top navigation with domain/local/search layers before adding more metrics.
11. Add metric metadata/source/method/export surfaces so every new metric is explainable.
12. Measure query cost before adding materialized tables or background refresh jobs.

## Proposed work sequence for the next plan

The follow-up plan should be gated, not a big-bang rewrite:

### Gate 1 — contract and inventory

Freeze the metric definition schema, enumerate current metrics and their source queries, define period/coverage semantics, and document the navigation taxonomy. No database migration yet.

### Gate 2 — one vertical slice

Implement the catalog endpoint, one series endpoint, and one chart/table/metadata view for a small cross-domain set. Verify that the response is deterministic and the frontend never aggregates raw rows.

### Gate 3 — report and overview integration

Make monthly report sections consume shared metric adapters where appropriate, while preserving rich typed domain data. Add recent-period comparisons and coverage-aware empty states.

### Gate 4 — frontend architecture

Extract route controllers, add theme-specific expense/report components, and implement the layered navigation. Keep the six design-language identities independently testable.

### Gate 5 — storage decision

Use query plans and measured latency to decide whether any metric series needs a materialized read model. If yes, add a narrowly scoped migration with refresh/invalidation semantics and tests for staleness.

### Gate 6 — expansion

Add metrics one by one through the catalog, adapter, chart/table/metadata contract, fixture data, and navigation/search index. A new metric should not require a new global tab.

## Open decisions before implementation

These are the few decisions that genuinely need approval; everything else can follow from them:

1. Is a metric definition code-owned for v0.5, or must the UI edit definitions? Recommendation: code-owned.
2. Should a metric series be on-demand initially? Recommendation: yes, with existing read caching and measurements.
3. Is a metric a canonical field, a derived series, or both? Recommendation: definitions distinguish `canonical` from `derived` and always expose the method.
4. Should the next report contract be a new endpoint/version or an additive change before v0.4 release? Recommendation: do not destabilize the current v0.4 monthly contract; introduce the extensible envelope after the release gate.
5. Which domains belong in the primary header? Recommendation: Today, Overview, Motion, Night, Health/Patterns, Library, Expenses, with Reports and Tools in cross-domain/local navigation; exact labels can be validated in the UX pass.
6. Is the six-theme requirement still “six complete compositions per supported page”? Based on the existing theme contract and previous decisions, yes.

## Source links

- [OWID mission and purpose](https://ourworldindata.org/about)
- [Choosing topics and metrics](https://ourworldindata.org/choosing-our-topics-and-metrics)
- [Redesigning OWID interactive visualizations](https://ourworldindata.org/redesigning-our-interactive-data-visualizations)
- [Making OWID data easier to reuse](https://ourworldindata.org/easier-to-reuse-our-data)
- [OWID Chart API](https://docs.owid.io/projects/etl/api/chart-api/)
- [ETL workflow](https://docs.owid.io/projects/etl/architecture/workflow/)
- [ETL metadata](https://docs.owid.io/projects/etl/architecture/metadata/)
- [ETL common format](https://docs.owid.io/projects/etl/architecture/design/common-format/)
- [ETL catalog structures](https://docs.owid.io/projects/etl/libraries/catalog/structures/)
- [OWID Grapher source repository](https://github.com/owid/owid-grapher)
- [OWID Grapher chart API specification](https://github.com/owid/owid-grapher/blob/master/docs/chart-api.openapi.yaml)
- [ECharts accessibility/ARIA guidance](https://echarts.apache.org/handbook/en/best-practices/aria/)

## Iroha files audited

- [`docs/data-model.md`](../data-model.md)
- [`docs/capabilities/core.md`](../capabilities/core.md)
- [`docs/provider-capabilities.md`](../provider-capabilities.md)
- [`docs/frontend-theme-architecture.md`](../frontend-theme-architecture.md)
- [`docs/cockpit-api-redesign.md`](../cockpit-api-redesign.md)
- [`docs/contracts/api-v1-decisions.md`](../contracts/api-v1-decisions.md)
- [`apps/iroha-server/db/migrations/00001_current_schema.sql`](../../apps/iroha-server/db/migrations/00001_current_schema.sql)
- [`apps/iroha-server/db/migrations/00007_expenses.sql`](../../apps/iroha-server/db/migrations/00007_expenses.sql)
- [`apps/iroha-server/pkg/daily/service.go`](../../apps/iroha-server/pkg/daily/service.go)
- [`apps/iroha-server/pkg/reports/types.go`](../../apps/iroha-server/pkg/reports/types.go)
- [`apps/iroha-server/pkg/httpapi/server.go`](../../apps/iroha-server/pkg/httpapi/server.go)
- [`apps/iroha-web/src/lib/navigation.ts`](../../apps/iroha-web/src/lib/navigation.ts)
- [`apps/iroha-web/src/routes/+layout.svelte`](../../apps/iroha-web/src/routes/+layout.svelte)
- [`apps/iroha-web/src/lib/themes/registry.ts`](../../apps/iroha-web/src/lib/themes/registry.ts)
- [`apps/iroha-web/src/routes/expenses/+page.svelte`](../../apps/iroha-web/src/routes/expenses/+page.svelte)
- [`apps/iroha-web/src/routes/reports/+page.svelte`](../../apps/iroha-web/src/routes/reports/+page.svelte)
