# Theme asset boundary audit

- Date: 2026-08-14
- Scope: `apps/iroha-web`, `apps/iroha-public-site`, and `packages/iroha-shared`
- Reference: `b1b6cd2` plus the current working tree

## Verdict

The theme asset boundary is currently violated. The six current production visual languages are documented in the shared package, while the newer design-workshop compositions are web-local; their
actual implementations are owned by `iroha-web`. This means the most valuable part of Iroha cannot be consumed by `iroha-public-site` or another future client without importing private-app internals
or duplicating the work.

The new hard rule is in [`AGENTS.md`](../AGENTS.md). The shared-package README was updated in the same change because it described the opposite architecture.

## Inventory

| Asset                                                                     | Current location                                                                                                                                                                                   | Finding                                                                                                                                               |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Current production theme IDs, identities, route names, and registry types | [`packages/iroha-shared/src/themes.ts`](../packages/iroha-shared/src/themes.ts)                                                                                                                    | Correctly shared. This is the contract, not the implementation; adopted future designs must extend this registry rather than create a second catalog. |
| Semantic theme tokens                                                     | [`packages/iroha-shared/src/themes.css`](../packages/iroha-shared/src/themes.css)                                                                                                                  | Correctly shared.                                                                                                                                     |
| Shared low-dependency Svelte primitives                                   | [`packages/iroha-shared/src`](../packages/iroha-shared/src)                                                                                                                                        | Correct home; the set is incomplete.                                                                                                                  |
| Theme registry and component map                                          | [`apps/iroha-web/src/lib/themes/registry.ts`](../apps/iroha-web/src/lib/themes/registry.ts)                                                                                                        | Wrong. The web app is the source of truth for the current production compositions, while newer design compositions are not in the shared registry.    |
| Theme context and route renderer                                          | [`apps/iroha-web/src/lib/themes/context.svelte.ts`](../apps/iroha-web/src/lib/themes/context.svelte.ts), [`ThemeRouteRenderer.svelte`](../apps/iroha-web/src/lib/themes/ThemeRouteRenderer.svelte) | Wrong as shared assets; they are generic theme-runtime infrastructure.                                                                                |
| Theme provider                                                            | [`apps/iroha-web/src/lib/themes/ThemeProvider.svelte`](../apps/iroha-web/src/lib/themes/ThemeProvider.svelte)                                                                                      | Mixed responsibility. Theme context belongs in the shared package; local-storage persistence is a web adapter.                                        |
| Theme frame                                                               | [`apps/iroha-web/src/lib/themes/ThemeFrame.svelte`](../apps/iroha-web/src/lib/themes/ThemeFrame.svelte)                                                                                            | Mixed responsibility. The shell host belongs with the theme package; `APP_VERSION` and private fallback footer do not.                                |
| Route compositions                                                        | [`apps/iroha-web/src/lib/themes/`](../apps/iroha-web/src/lib/themes/)                                                                                                                              | Wrong. There are 72 Svelte route components: 12 each for Atlas, Grapher, Field Journal, Phenology, Sound Map, and Archive.                            |
| Cross-theme charts and visual components                                  | [`apps/iroha-web/src/lib/components/`](../apps/iroha-web/src/lib/components/)                                                                                                                      | Mixed. Theme-aware charts, report surfaces, sleep surfaces, activity surfaces, and media surfaces are core assets but remain web-local.               |
| Public-site theme implementation                                          | [`apps/iroha-public-site/src/`](../apps/iroha-public-site/src/)                                                                                                                                    | Missing. It currently imports only the shared `StatTile` wrapper; it cannot render the registered production or newer design compositions.            |

## Detailed findings

### Critical: the current production compositions are web-owned

The route folders contain 72 Svelte files, exactly 12 per current registered production theme. The web-only registry imports every one of them through `$lib/themes/...` and constructs the only
`ThemeDefinition<Component>` instances used at runtime. That is the core asset living in the wrong application.

This is not fixed by moving only a card or a chart. The following are all theme-owned compositions and must move together behind a shared view contract:

- shell and today
- daily and dashboard
- activities and activity detail
- sleep
- media and media detail
- expenses and reports
- metrics

### Critical: the route components are coupled to private-app internals

The route files directly import `$lib/api`, `$lib/format`, `$lib/sport`, `$lib/expense-view`, `$lib/report-view`, `$lib/hero-title`, and web-local components. A blind directory move would merely
relocate broken imports. The migration needs an explicit shared view-model boundary: typed data in, callbacks/snippets out, with API fetching and route navigation kept in the application.

The most coupled areas are:

- expenses: API response types, money formatting, CSV/export policy, and the ledger currently mix with the composition;
- reports: API report types and report-view derivation are imported directly;
- activity and sleep detail: canonical API types, maps, charts, and source labels are interwoven;
- media: API row/detail types and media formatting are imported directly;
- shell: private version and theme persistence are mixed into presentation.

### High: shared visual primitives are only partially shared

The package already owns useful primitives such as `MetricPanel`, `MetricTable`, `StatTile`, period controls, source badges, and pure formatters. However, the web app still owns theme-aware or
cross-theme visual assets such as `BarChart`, `ActivityMetricChart`, `ActivityDetailChart`, `LapChart`, `SleepAggregateChart`, `SleepArchitectureChart`, `SleepTimelineChart`, `ReportComparison`,
`ReportCoverage`, `ReportEvidenceList`, `ReportFactGrid`, `ReportMetricCard`, `RingGauge`, `MediaBarChart`, and the media/report surfaces inside each theme folder.

These should be audited one by one during migration. A component is a shared asset when its visual grammar is part of Iroha's data language and it can be fed by a typed view model. A component remains
web-local when it performs API fetching, private navigation, authentication, deployment/version display, or app-only command-palette behavior.

### Critical: the design workshop is not an adoption path

[`apps/iroha-web/src/lib/design-workshop.ts`](../apps/iroha-web/src/lib/design-workshop.ts) defines seven compositions—Editorial, Command center, Chronicle, Cover page, Personal OS, Field journal, and
Quiet. Their substantial implementations live inside the web-only design route. They are presented as “implemented” specimens, but they have no shared registry identity, no shared route
implementation, and no public-site consumer. They must be promoted into the shared design registry and receive real route compositions; they are not disposable variants or concept-board leftovers.

### Medium: the registry contract is split across packages

`packages/iroha-shared/src/themes.ts` owns the current IDs, identities, route names, and registry types, while `apps/iroha-web/src/lib/themes/registry.ts` owns the actual route map. The
design-workshop IDs are separately owned by `apps/iroha-web/src/lib/design-workshop.ts`. This allows the shared contract to describe production themes while the public site has no implementation and
the web app silently owns both production and newer compositions.

The canonical registry must eventually include the implementation map in the shared theme package. The web app should provide only data and navigation adapters, not a second registry.

### Medium: public-site reuse is currently nominal

Both apps alias `@iroha/shared`, but the public site only consumes a shared `StatTile` wrapper. The current production themes, newer design compositions, their route compositions, and the theme-aware
charts are not actually reusable across the two apps today.

## Target boundary

Keep the existing `@iroha/shared` package as the source-only package for now; split a dedicated `@iroha/theme-ui` package later only if the chart/map dependency weight makes that necessary. The
important boundary is `packages/`, not the package name.

The target shape is:

```text
packages/iroha-shared/src/
  themes.ts                 # IDs, identities, route contract
  themes.css                # semantic tokens
  theme-ui/                 # registered themes and adopted compositions
    registry.ts
    context.svelte.ts
    ThemeRouteRenderer.svelte
    atlas/ ... grapher/ ... field-journal/ ...
    components/             # charts, cards, receipts, controls
  view-models/              # canonical frontend contracts

apps/iroha-web/src/
  lib/api.ts                # HTTP and response decoding
  lib/theme-adapter/        # API -> shared view model, persistence, navigation
  routes/                   # route state, loading, errors, URL/query behavior

apps/iroha-public-site/src/
  lib/fixture-adapter/      # static examples -> shared view model
  routes/                   # public shell and static page composition
```

The shared theme code may depend on Svelte and shared frontend libraries, but must not import `$lib/api`, route modules, server packages, or another app's source. It accepts typed view data, snippets,
and callbacks. The theme context must be driven by the shared registry; local storage or URL persistence is injected by the hosting app.

## Migration order

1. Define and test shared view models for the route families, starting with media, expenses, and reports where the current regressions are visible.
2. Move the theme-aware primitive layer and its theme-specific styling into `packages/iroha-shared/src/theme-ui/components/`.
3. Promote the design-workshop compositions into the shared design registry; keep Editorial, Command center, Chronicle, Cover page, Personal OS, Field Journal, and Quiet as real implementations rather
   than route-local demos.
4. Move one complete route family across all current production themes and adopted compositions, including registry entries and design-page specimens. Media is a good first slice because it exposes
   the current Grapher artwork regression and the identity differences clearly.
5. Move the remaining route families and the shared registry. Keep web route adapters thin while preserving URL, loading, and navigation behavior.
6. Add public-site fixture data and render all registered themes and adopted compositions in its design workbench. This is the cross-app proof that the boundary is real.
7. Add a CI/check target that fails if new files appear under `apps/*/src/lib/themes` or if `packages/iroha-shared` imports app aliases.

## Completion criteria

- No registered theme or adopted design composition, and no theme-aware primitive, is source-of-truth code under either app.
- `iroha-web` and `iroha-public-site` render the same shared compositions from their respective data adapters.
- Shared theme code has no `$lib` or app-source imports.
- Theme IDs, route names, identities, registry entries, and tokens have one canonical definition.
- The design page exercises the same shared components used by production routes; it is not a parallel mock.
- Web and public-site checks pass, and every theme is browser-verified with representative fixture/API data.
