# Iroha v0.4 cockpit gap notes

Status: pre-implementation review. No product or deployment code was changed for this note.

Date: 2026-08-13

This note records the issues reported after the latest v0.4 dev deployment, the observed evidence, the current root causes, and the acceptance bar for the next implementation pass. It is deliberately
a contract and review artifact, not a list of speculative fixes.

## First principles

These constraints are now treated as non-negotiable:

1. Iroha owns canonical records and canonical aggregation. A browser must not derive a supposedly complete lifetime result from one paginated page.
2. Every aggregate names its meaning: period grain, unit, timezone, coverage, dimensions, and reducer/method. Missing data is not zero.
3. A sleep session and a calendar night are different facts. Main sleep and nap are different facts. The UI must not label a session count as a day count.
4. A chart is a representation of a returned canonical series. The selected theme may change composition and visual grammar, but not the data semantics.
5. Shared controls own shared geometry and keyboard behavior. Themes own visual identity and composition, not arbitrary placement of the same control.
6. Timestamp display uses the shared canonical display contract: `yyyy-MM-dd HH:mm:ss`. Date-only values use `yyyy-MM-dd`.

## Evidence captured

The following was checked against the current source and the deployed v0.4 dev stack:

- The June 2026 sleep aggregate reports `session_count = 31`, `main_sleep_count = 30`, and `nap_count = 1`. This is not 31 calendar days. The live session list contains both a main sleep and a nap on
  at least one date.
- The lifetime Night view requests and initially renders 31 rows. The API is paginated (`has_more = true`); the UI exposes “Load more nights”. The reported total of 62 was not reproduced exactly, but
  incomplete lifetime presentation is confirmed and is enough to make the lifetime headline unsafe.
- Motion passes activities and a summary into the theme renderer, but no chart model. None of the six current Motion theme components renders a chart. This is a missing data/route composition path,
  not an ECharts repaint bug.
- `BarChart.svelte` enables ECharts ARIA decals globally. The diagonal decal is the source of the unrequested slash pattern inside bars.
- `NavigationMenu.svelte` uses independent native `<details>` elements. It closes the menu after its own link is clicked, but has no policy to close a previously opened sibling when another group is
  opened.
- `listAllExpenses` walks all API cursor pages, but the expense ledger renders every returned row into one unbounded list. The backend page limit is not the only scalability concern; the browser DOM
  and interaction model still need a bounded strategy for large months.
- The current metric-panel CSV uses minimum RFC-style quoting: it quotes commas, newlines, and quotes, but not every text cell. That is syntactically valid CSV while still failing the requested
  robust, human/export-friendly behavior for values such as currency plus amount text.
- Several production surfaces still print raw ISO timestamps or use localised browser formatting instead of the shared formatter, including Admin jobs, report generation, expense timestamps, and Night
  detail start/end times.
- Admin's “Canonical surface / Registered metrics” section is a metric definition catalog. It is not a browser for canonical domain records, and the current wording makes that boundary unclear.

## Issue register

### 1. Navigation groups do not behave like one tab control

**Current behavior:** opening another top-level group leaves the first group's native `<details>` open. Selecting a link only closes the details containing that link.

**Required behavior:** at most one top-level popover is open at a time. Opening another group closes the previous one; selecting a destination, pressing Escape, clicking outside, and navigating by
keyboard all leave the navigation in a closed state. The active route remains visible in the group label/style.

**Acceptance:** a browser test opens Domain, opens Analyze, and asserts Domain is closed; it repeats the sequence with More and with keyboard navigation. The test must cover desktop and the mobile
layout.

### 2. Night period semantics and lifetime completeness

**Current behavior:** the page headline says “Recorded nights” while the loaded rows and chart are sleep sessions. The first lifetime request contains only 31 rows and relies on a manual load-more
interaction. The monthly aggregate labels June's 31 sessions as if it were a number of nights.

**Required data contract:** expose and render separate values:

- `sessions`: number of sleep-session records;
- `main_sleep_nights`: number of main-sleep records, or explicitly named main sleep sessions if the domain does not guarantee one per date;
- `nap_sessions`: number of nap records;
- `observed_wake_dates`: distinct canonical wake dates where a calendar-day view is intended;
- coverage and period metadata for every aggregate/series.

The server remains the authority for these counts. Frontend row pagination must never be used as a lifetime aggregate.

**Recommended lifetime representation:** show a complete server-side month/year rollup for the lifetime chart, then show a bounded recent/session detail view with an explicit “load more sessions”
affordance. Do not render hundreds or thousands of session bars as the lifetime chart.

**Acceptance:** June visibly distinguishes `30 main sleep`, `1 nap`, and `31 sessions`; a separate calendar-day count is only shown if it is computed as distinct dates. Lifetime headline and rollup
remain stable regardless of how many detail pages the user loads.

### 3. Night detail is not consistently reachable or represented

The canonical `/night/{id}` route exists and already contains a sleep-stage timeline, but theme-rendered Night lists do not consistently provide a real link to it. The original proposal's detail
surface therefore disappears depending on the selected theme.

The detail contract must include the session kind (`main sleep` or `nap`), the canonical date/time values, duration/efficiency, source, and stage timeline. The list must link to the detail route
rather than relying on selection-only behavior.

### 4. Main sleep versus nap is visually and analytically under-specified

The row table has a type label in at least the Grapher path, and the API already returns `is_main_sleep`, but headline counts and charts mix both classes. The default Night chart should represent main
sleep as the primary series and make naps visible as a separate class (for example, a distinct color/marker or a clear toggle). No frontend inference from duration or date is acceptable.

### 5. Motion has no chart path

**Current behavior:** Motion has filters, summary cards, and activity rows, but no route-owned chart model is passed into any theme. The existing synchronized activity chart is a detail component and
is not a substitute for the overview chart.

**Required first chart set:**

- a period trend using a server-returned monthly series for lifetime/year scope;
- a daily series for a selected month, when the backend can provide it;
- a colored sport breakdown with count/distance/duration semantics stated;
- exact table parity and an empty/partial-data state.

The route should request typed canonical series rather than aggregating only the currently loaded activity rows. The theme renderer receives data plus semantic tokens; it does not invent units or
reducers.

**Acceptance:** each of the six themes presents a deliberate Motion chart in its own visual grammar, the chart is populated by the live demo data, and changing period or sport filter changes the
series without stale bars or a blank repaint.

### 6. Remove the unrequested slash texture from bars

The global ECharts ARIA decal is currently always on. Accessibility must remain, but the default chart should use readable color, labels, legend, focus state, and table parity rather than a permanent
hatch texture. If a non-color accessibility mode is needed, it must be an explicit theme/accessibility decision and tested with a legend.

### 7. Expense export must be a real, round-trippable export

There are two different export concepts and they must not be conflated:

1. a metric-panel export of the displayed aggregate series;
2. a canonical expense-ledger export of expense records.

The ledger export must have stable columns for date, currency, integer amount, display amount, category, merchant/note, items, source, and record identifiers as applicable. A parser must round-trip
spaces, plus signs, currency labels, commas, quotes, and newlines. The implementation should quote all text cells deterministically and keep numeric truth in numeric columns; a display string such as
`JPY + 800` must never be the only amount representation. UTF-8 and spreadsheet compatibility should be tested explicitly.

### 8. Expense months with more than 100 records

The API client already walks cursor pages, so 100 is not the current logical month limit. The current UI still renders the full result into an unbounded list and has no visible result count,
incremental list boundary, or large-list strategy.

Required behavior for a large month:

- charts cap visual categories using a documented top-N/Other rule where appropriate;
- the ledger uses incremental rendering or a bounded virtualized/windowed list;
- the selected record/detail area remains usable while browsing;
- loading, empty, filtered, and error states are explicit;
- CSV export still includes every canonical matching record, not only visible rows.

### 9. Shared page geometry is inconsistent

The six themes may differ in hierarchy and visual language, but the global content edge, page header, period control, domain filter, first chart, and detail region should not jump between routes.
Current routes mount the period toolbar and theme content at different levels and use independent spacing rules.

The next pass should define one route scaffold with slots for:

```text
global navigation
page identity/header
shared period control + route-specific filters
primary visualization
exact data/detail surface
```

The theme registry selects identity tokens and composition; it does not change the contract or silently relocate shared controls. Visual regression must check the scaffold across all six themes and
representative viewport widths.

### 10. Timestamp formatting is not centralized in practice

`formatDate` already expresses the intended `yyyy-MM-dd HH:mm:ss` shape, but raw values and `toLocale*` calls remain in production surfaces. Replace those display paths with shared `formatDateTime`
and `formatDateOnly` helpers, preserving the existing canonical timezone contract and keeping timezone out of the user-facing period picker.

**Acceptance:** a source timestamp displayed in Admin, Reports, Expenses, Night detail, Motion detail, and Library detail has the same exact format. Date-only domain values remain date-only and are
not silently converted to local calendar dates.

### 11. Admin's “canonical surface” needs a truthful boundary

The current section is a read-only metric catalog: metric IDs, domains, labels, units, and definitions. It is not the canonical data surface. Rename the section to “Metric catalog” or “Metric
definitions”, show whether each metric is canonical/derived, and explain the distinction in one sentence.

If Admin remains the operational page, its sections should be clearly separated:

- API/runtime health;
- metric catalog/definitions;
- import and background-job status;
- data freshness/provenance when those facts are available.

Canonical domain records remain owned and validated by the Iroha server; Admin does not become an accidental editor or a second data browser.

## Proposed implementation gates

### Gate 0 — freeze semantics and fixtures

- Add deterministic fixture cases containing two sessions on one date, a nap, an empty period, a partial period, and more than 100 expenses.
- Specify session count, main-sleep count, nap count, distinct-date count, coverage, grain, and timezone in the API examples.
- Decide whether the default Night “night” noun means main-sleep records or distinct wake dates; the UI must use the chosen noun exactly.

### Gate 1 — shared interaction and geometry

- Make top navigation a single-open control with Escape/outside-click/keyboard behavior.
- Finish the shared period/date control and exact date format helpers.
- Introduce the route scaffold boundary without changing theme identities.

### Gate 2 — canonical series and complete aggregation

- Add/adjust server-owned Night series and counts with explicit semantics.
- Add server-owned Motion overview series at month and day grains where needed.
- Ensure lifetime responses are complete aggregates, independent of detail-page pagination.

### Gate 3 — theme-specific visual representation

- Give each theme an intentional Motion chart and corrected Night chart/detail path.
- Remove default bar decals and preserve accessibility through semantic color, focus, labels, tooltips, and table parity.
- Validate no NaN/blank series and no stale chart after a period/filter change.

### Gate 4 — expense scale and export

- Add canonical ledger CSV beside metric-panel CSV.
- Test round-trip quoting and UTF-8 behavior.
- Add large-month list behavior and top-N chart rules.

### Gate 5 — browser-quality gate

- Run route × theme × viewport visual regression for Overview, Motion, Night, Expenses, Reports, Patterns, Admin, and representative detail routes.
- Add interaction assertions for navigation closing, left/right period movement, clean refresh URLs, detail links, load-more behavior, and chart repaint.
- Use live seeded data and a fixture with deliberate edge cases; a passing unit test alone is not the UX gate.

## Decisions still needing review

These are intentionally left visible instead of being silently chosen during implementation:

1. For Motion, should a selected month default to daily distance, daily duration, or a compact multi-series view? Recommendation: daily distance plus a sport breakdown, with duration in the exact
   table/tooltip.
2. For Night lifetime, should the main chart be monthly or yearly? Recommendation: year/month rollup for lifetime, session detail only for a bounded selected period.
3. Should Night headline “nights” mean main-sleep records or distinct wake dates? Recommendation: show both when they differ and never hide the session count.
4. Should all CSV text cells be quoted? Recommendation: yes, for deterministic spreadsheet/export behavior, while preserving raw numeric columns.
5. Should the metric catalog move to `/metrics` and Admin stay operational? Recommendation: yes, with a small read-only catalog summary retained in Admin.

## Not done in this note

- No source code, API contract, fixture, CSS, chart, or deployment changes were made.
- No dev resources were deleted.
- The exact user-observed lifetime value of 62 was not reproduced; the confirmed defect is that the lifetime UI is page-limited and semantically labels session data as nights.
